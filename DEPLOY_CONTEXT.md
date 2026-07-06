# DEPLOY_CONTEXT.md

Deployment facts the repo doesn't tell you. Read this first in a cold-start
session. **Do not** touch any of this from a code change — env vars,
hosting, and the CORS trust list are owned by humans via dashboards.

> Snapshot: this describes the production topology as of 2026-07-06.
> If a URL or service name changes, update this file in the same PR
> that changes the actual deployment.

---

## 1. Live URLs

| Surface | URL | Host |
|---|---|---|
| Frontend (user-facing) | `https://dmtool-eight.vercel.app` | Vercel |
| Backend API (server-to-server + `/api/*` from frontend) | `https://dmtool-backend.onrender.com` | Render |
| API base path | `/api` | (path prefix, not a host) |

The frontend and backend live on **different hosts**. The frontend talks
to the backend by absolute URL — the browser issues a CORS request —
which is why `ALLOWED_ORIGINS` on the backend must list the frontend
host (see § 5).

Local dev (per `docker-compose.yml` + `backend/.env.example`):

- Backend: `http://localhost:8080`
- Frontend: `http://localhost:3000`
- Frontend's `NEXT_PUBLIC_API_URL`: `http://localhost:8080/api`

---

## 2. Repos & branches

- **GitHub**: `createdevflow/dmtool` (private).
- **Local working branch** (this session's): `phase-1/migrations`.
- **Base branch** for merges: `main`.
- Other local-only branches present: `feature/phase-1-2-implementation`,
  `phase-3/perf-cleanup-and-ssr-fixes`, `phase-3/real-analytics-data`.

**Push rule**: local commits only. The user explicitly says
"push it" when all phases are done and reviewed. Do not `git push
origin main` from a phase commit — it hasn't been signed off.

---

## 3. Hosting topology

### Vercel (frontend)

- Vercel project name: `dmtool` (or whatever maps to the
  `dmtool-eight.vercel.app` URL).
- Build command: `npx next build` (Next.js 16 App Router).
- Output: `.next/`.
- Framework preset: Next.js.
- **Environment variables** (set in Vercel dashboard → Settings →
  Environment Variables):

  | Var | Value |
  |---|---|
  | `NEXT_PUBLIC_API_URL` | `https://dmtool-backend.onrender.com/api` |

  `NEXT_PUBLIC_*` vars are inlined at build time — re-deploy after
  changing them, the running build will not pick up the new value.

### Render (backend)

- Service: `dmtool-backend` (Go binary built from `backend/cmd/api`).
- Build command: `go build -o api ./cmd/api` (or whatever the
  Render `render.yaml` / dashboard says — verify in the dashboard).
- Start command: `./api`.
- Health check: `GET /health` (returns 200 + `{status: "ok", db: "ok", ...}`).
- **Environment variables** (set in Render dashboard → Environment):

  | Var | Required | Source / how to generate |
  |---|---|---|
  | `APP_ENV` | yes | `production` (never change locally; the dev mode enables GORM AutoMigrate and a fixed dev JWT key) |
  | `PORT` | yes | `8080` (matches the docker-compose mapping) |
  | `VERSION` | no | `2.0.0` (informational; surfaced in `/health` and `Server` header) |
  | `DATABASE_URL` | yes | Render's internal Postgres connection string. The local dev path is `file:dmtool.db` (SQLite). The GORM driver auto-detects from the URL prefix (`postgres://` vs `file:`). |
  | `JWT_PRIVATE_KEY` | yes (prod) | PEM-encoded RS256 private key. Generate per `SECURITY.md` § 3. |
  | `JWT_PUBLIC_KEY` | yes (prod) | PEM-encoded RS256 public key, paired with the private key. |
  | `ENCRYPTION_KEY` | yes | 64 hex chars (32 bytes). Encrypts stored OAuth tokens at rest. See `SECURITY.md` § 7. |
  | `ALLOWED_ORIGINS` | yes | **Must match the frontend's URL exactly.** See § 5. |
  | `FRONTEND_URL` | yes | Same value as `ALLOWED_ORIGINS` (the OAuth callback redirect uses this). |
  | `GOOGLE_CLIENT_ID` | yes for GSC integration | Google Cloud OAuth client. |
  | `GOOGLE_CLIENT_SECRET` | yes for GSC integration | Rotate per `SECURITY.md` § 4. |
  | `META_APP_ID` | yes for IG/FB | Meta app ID. |
  | `META_APP_SECRET` | yes for IG/FB | Rotate per `SECURITY.md` § 5. |
  | `LINKEDIN_CLIENT_ID` | yes for LinkedIn | LinkedIn app. |
  | `LINKEDIN_CLIENT_SECRET` | yes for LinkedIn | Rotate per `SECURITY.md` § 6. |
  | `LINKEDIN_SCOPES` | no | Comma-separated scope list (defaults via `config.go`). |
  | `OPENAI_API_KEY` | no | If empty, AI features use the rule-based fallback (`[main] No OPENAI_API_KEY — AI insights will use rule-based engine`). |
  | `DATAFORSEO_LOGIN` / `DATAFORSEO_PASSWORD` | no | DataForSEO creds for keyword research; unused in dev. |
  | `RAPIDAPI_KEY` | no | RapidAPI key for social scraping; unused in dev. |

  All `*_SECRET` / `*_KEY` values **must** be rotated if they ever
  appear in git history. The current `.env.example` files contain
  values that were at one point live — treat them as public and
  follow the rotation procedures in `SECURITY.md` before the next
  production deploy if you haven't already.

### Local stack (`docker-compose.yml` + `nginx.conf`)

For a full local-prod-mirror setup:

```bash
docker-compose up --build
```

This brings up:

- `dmtool-backend` (Go API on `:8080` inside the container)
- `dmtool-frontend` (Next.js on `:3000` inside the container)
- `dmtool-nginx` (reverse proxy on `:80`/`:443`, config in
  `nginx.conf`)

The frontend talks to the backend via `http://backend:8080` (the
Docker Compose service name), not `localhost` — that is what the
`NEXT_PUBLIC_API_URL` override in `docker-compose.yml` is for.

`nginx.conf` enforces:

- HTTP → HTTPS redirect.
- Stricter rate limit on `/api/auth/*` (5 req/min) vs the rest of
  `/api/*` (200 req/min).
- Security headers (`HSTS`, `X-Frame-Options: DENY`, etc.) — the
  backend's `SecurityHeaders()` middleware sets the same set
  belt-and-suspenders.
- 10 MB body cap.

---

## 4. Database

- **Production**: Render-managed PostgreSQL. The connection string
  goes in `DATABASE_URL`. GORM's `db.Init` auto-selects the
  `postgres` driver when the URL starts with `postgres://` or
  `postgresql://`.
- **Development**: SQLite file (`dmtool.db` in the working dir, or
  `file:<path>` in `DATABASE_URL`). Uses the pure-Go `glebarez/sqlite`
  driver so no CGO toolchain is required on Windows.

### Schema migrations

Two paths, both idempotent:

1. **Dev** (`APP_ENV=development`): GORM `AutoMigrate` runs at boot
   from `internal/db/db.go`. Adds new columns and tables forward-only.
2. **Prod** (any `APP_ENV`): `RunMigrations` from `internal/db/migrate.go`
   runs at boot. Today it does two things:
   - Seeds the 3 plan rows (`free`, `pro_monthly`, `pro_yearly`) on a
     fresh database using `ON CONFLICT DO NOTHING`.
   - Backfills the `users.dashboard_mode` column on any pre-phase-4
     user that lacks it (forward-only ALTER for the column phase 4
     added).

   Anything you add to a model struct that you want prod to pick up
   must be either (a) handled by `RunMigrations` for non-AutoMigrate
   reasons (data backfill, constraint changes), or (b) covered by
   GORM AutoMigrate being run in dev — for production, ship a
   proper SQL migration file alongside the model change.

   **Current state**: AutoMigrate is dev-only. Production has no
   forward migration for any GORM struct change beyond
   `RunMigrations`'s plan-seed and `dashboard_mode` backfill. If
   you add a column that needs to exist in prod, either (a)
   extend `RunMigrations` with a `CREATE INDEX IF NOT EXISTS` /
   `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, or (b) document
   the manual `psql` step in this file.

---

## 5. CORS / `ALLOWED_ORIGINS` — the most common deploy footgun

`ALLOWED_ORIGINS` is the trust list for both the **backend's** CORS
middleware (`internal/middleware/cors.go`) and the **frontend's**
proxy trust list (`frontend/proxy.ts` via
`process.env.ALLOWED_ORIGINS`). Both sides must agree, or:

- If the backend is stricter than the frontend: browsers see CORS
  failures on every state-changing request.
- If the frontend is stricter than the backend: the proxy blocks
  navigation paths that the API would have allowed.

**Production values today** (and what they must remain in sync with):

```
Render (dmtool-backend service → Environment):
  ALLOWED_ORIGINS=https://dmtool-eight.vercel.app
  FRONTEND_URL=https://dmtool-eight.vercel.app

Vercel (dmtool project → Settings → Environment Variables):
  ALLOWED_ORIGINS=https://dmtool-eight.vercel.app
```

Comma-separate multiple origins if you ever add a staging URL, e.g.
`ALLOWED_ORIGINS=https://dmtool-eight.vercel.app,https://staging-dmtool.vercel.app`.
The backend splits on `,` in `config.go`; the frontend proxy does
the same.

This is also logged in the defer log as **DEFER-002**: "Production
environment variables" (`IMPLEMENTATION_PLAN.md` §10). It is not
deferred in the technical-debt sense — the values exist — it is
flagged because missing it is the most common reason a deploy fails
silently with CORS errors.

---

## 6. Auth flow at a glance

- JWT access token: **15 min TTL**, RS256 signed, sent via
  `Authorization: Bearer <token>`.
- Refresh token: **7 day TTL**, opaque random string, stored in
  the `refresh_tokens` table SHA-256-hashed, set as an `HttpOnly;
  SameSite=Lax; Secure` cookie named `refresh_token`.
- Frontend's single source of truth for the access token is the
  cookie path (phase 3) — `localStorage` is not used for auth.
- The frontend's `apiClient` reads the token from the cookie via
  `lib/auth-cookie.ts` and attaches it as `Authorization: Bearer`.
  On 401, the cookie is cleared and the user is bounced to
  `/login` (unless they're already on `/login` or `/register`).
- Logout: `POST /api/auth/logout` revokes the refresh-token row
  and clears the cookie.

See `SECURITY.md` for credential rotation procedures, the JWT-key
leak history, and the CSRF defense in depth.

---

## 7. Pre-deploy / pre-push checklist

Run before every merge to `main`:

1. **Backend**:
   - `cd backend && go build ./...` (clean)
   - `cd backend && go vet ./...` (clean)
   - `cd backend && go test ./...` (no failures)
   - `cd backend && go run ./_tools/phase4_smoke/` (7/7 OK) — the
     only committed end-to-end smoke; covers the /me contract.
2. **Frontend**:
   - `cd frontend && npx next build` (Compiled successfully; all
     routes still present; `ƒ Proxy (Middleware)` still wired)
3. **Schema**: any model change since last deploy? If so,
   `RunMigrations` either covers it or you ship a SQL file.
4. **Env vars**: did you add a new env var? Is it set on both
   Render and Vercel where applicable? Are secrets rotated per
   `SECURITY.md` if they were ever in git?
5. **`ALLOWED_ORIGINS`** on Render and Vercel still match.

---

## 8. Things that look like deploy but aren't

- `DEPLOY_CONTEXT.md` is this file. There is no other deployment
  doc in the repo. Don't search for one — there isn't one.
- `SYSTEM_ARCHITECTURE.md` has a "Deployment Strategy" section
  (lines 338-353) but it's aspirational, not the live topology.
  It describes a phased roadmap (compose → microservices →
  Kubernetes), not what we're running today.
- `nestjs-backend/`, `crawler/`, `ai-engine/` are legacy dirs from
  the initial commit. They are not deployed.

---

## 9. Open deploy items

- **DEFER-002** (`IMPLEMENTATION_PLAN.md` §10): the `ALLOWED_ORIGINS`
  values must be set on both Render and Vercel to the same production
  URL. If you have not verified this on the live services, do so
  before the first push to `main`.
- **SubscriptionEvent history table** (mentioned in `billing_handlers.go`):
  no production-side migration needed (the model hasn't been added
  yet) but when it is added, ship a migration alongside the model.
- **Real Stripe**: the webhook returns 501 today. When the real
  webhook lands, it will need `STRIPE_WEBHOOK_SECRET` (or similar)
  in the Render env, plus the `whsec_` URL registration in the
  Stripe dashboard.

---

End of file. Update when a URL, env var, or service topology changes.
