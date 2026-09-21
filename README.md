# DMTool

Semrush-style marketing platform — SEO + social + AI in one dashboard.

This repo contains the full stack:

| Path | What it is |
|------|------------|
| `backend/` | Go (Gin) API server. SQLite for dev, Postgres in prod via the same `database/sql`-managed GORM layer. RS256 JWT auth, billing, admin panel, audit log, entitlements service. |
| `frontend/` | Next.js 16 (App Router) dashboard. Radix UI primitives + Tailwind v4. Auth via cookies gated by `proxy.ts`. |
| `docker-compose.yml` | Postgres + adminer for local prod-like runs. |
| `nginx.conf` | Production reverse-proxy sample (gzip, rate limits, WebSocket upgrade). |
| `DEPLOY_CONTEXT.md`, `IMPLEMENTATION_PLAN.md`, `CHANGELOG.md` | Plan / history / charter. |

Older code under `nestjs-backend/`, `crawler/`, `ai-engine/` is
**out of scope** for the phase-7/8/9 work — see
`IMPLEMENTATION_PLAN.md` for the rewrite phasing.

---

## Quickstart (local dev)

Required: Go 1.24+, Node 20+, npm.

```bash
# 1. Backend on port 8097 (port-isolated from prod-port 8080)
cd backend
go build -o ./tmp_api.exe ./cmd/api/
DATABASE_URL=file:dev.db APP_ENV=development PORT=8097 ./tmp_api.exe &

# 2. Bootstrap an admin account (one-time)
ADMIN_EMAIL=admin@dmtool.local ADMIN_PASSWORD_HASH='$2a$12$...' \
  ADMIN_NAME='Local Admin' APP_ENV=development DATABASE_URL=file:dev.db \
  go run ./_tools/bootstrap_admin/

# 3. Frontend on port 3000
cd frontend
npm install
NEXT_PUBLIC_API_URL=http://localhost:8097/api npm run dev
```

Open <http://localhost:3000>. Log in as the bootstrapped admin, or
register a new owner-tier account.

The backend's `internal/db.Init(...)` runs migrations + seeds the
three plans (`free`, `pro_monthly`, `pro_yearly`) on first boot —
no separate `migrate` step needed.

---

## Environment variables

| Var | Required | Default | Used by |
|-----|----------|---------|---------|
| `PORT` | no | `8080` | backend `cmd/api` |
| `DATABASE_URL` | no | `file:dmtool.db` | backend |
| `APP_ENV` | no | `development` | backend (informational; controls logging verbosity) |
| `ALLOWED_ORIGINS` | yes (prod) | `http://localhost:3000` | both backend CORS **and** the Next proxy's CSRF trust list — keep them in sync (see DEPLOY-001 in `IMPLEMENTATION_PLAN.md`) |
| `FRONTEND_URL` | no | `http://localhost:3000` | backend (for redirect URLs) |
| `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY` | yes (prod) | dev-key (insecure) | backend RS256 mint + verify |
| `ENCRYPTION_KEY` | yes (prod) | dev-key | backend OAuth token encryption |
| `NEXT_PUBLIC_API_URL` | no | `http://localhost:8080/api` | Next proxy |

Production must set real `ALLOWED_ORIGINS` on **both** the backend
(or Render) and the frontend (Vercel). They MUST match.

---

## Architecture

```
┌───────────────────────────────────────────────────────────────────────┐
│ Browser                                                                │
│   ├─ /login, /register, /onboarding, /dashboard/*, /admin/*           │
│   └─ Cookie: dmtool_token / dmtool_user / dmtool_mode /                 │
│            dmtool_impersonation_{token,target}                        │
└───────────────────────────────────────────────────────────────────────┘
              │                                               ▲
   proxy.ts  ▼  (Next 16 middleware)             (axios interceptor mirrors
              ├─ CSRF origin check                          cookie into Auth
              ├─ JWT-shape gate on /dashboard, /admin        header)
              └─ Mode rewrite /dashboard → /dashboard/<mode>
              │
              ▼
┌───────────────────────────────────────────────────────────────────────┐
│ Go (Gin) — backend/cmd/api                                              │
│                                                                          │
│  /auth (register / login / refresh / logout / me)                       │
│  /dashboard/* (data)              ← JWT-auth gated                     │
│  /billing/{trial, subscribe, me, cancel, webhook}                       │
│  /admin/*                         ← JWT-auth + RequireRole(admin)       │
│                                                                          │
│  ent (entitlements)                  billingHand                         adminHand                │
│  ├ CanCreateSEOProject             ├ StartTrial (one-shot via           ├ ListUsers/GetUser/...  │
│  ├ ResolveSubscription             │   user.trial_used_at)              ├ Impersonate             │
│  └ QuotaExceededError              └ Subscribe (active|trialing;       └ StopImpersonation      │
│                                       active=trial replaced)                                       │
└───────────────────────────────────────────────────────────────────────┘
                │                                  ▲
                ▼                                  │
    ┌─────────────────────────┐        ┌─────────────────────────────┐
    │ SQLite (dev) / Postgres │        │ External OAuth + APIs        │
    │ (prod) via GORM         │        │ Google Search Console,       │
    │                         │        │ Meta Graph, etc. (out of      │
    │ Tables: users, projects,│        │ phase-9 scope)               │
    │ plans, subscriptions,   │        └─────────────────────────────┘
    │ user_preferences,        │
    │ admin_audit_logs, ...    │
    └─────────────────────────┘
```

The auth token chain:
1. User logs in → backend returns a 15-min RS256 access token +
   7-day opaque refresh token (stored hashed in
   `refresh_tokens`).
2. `frontend/lib/auth-cookie.ts.setToken` writes the access JWT
   to the `dmtool_token` cookie (non-HttpOnly, SameSite=Lax, so
   axios can read it via `document.cookie`).
3. `frontend/lib/api-client.ts` request interceptor prefers
   `dmtool_impersonation_token` when present, falling back to
   `dmtool_token` — that's why the admin's full-page navigations
   during a support session stay authenticated.

Phase 7 added an audit log of every admin mutation. Phase 8
applied real-world fixes (impersonation on cookies, plan create/
edit UI, shared Dialog/Sheet/Tooltip primitives, mobile sidebar,
real plan data in the topbar).

---

## Verify commands (post-change sanity)

| Command | What it proves |
|---------|----------------|
| `cd backend && go build ./...` | Backend compiles. |
| `cd backend && go vet ./...` | Lints. |
| `cd backend && go test ./...` | Runs `entitlements`, `handlers`, `db`, `services/entitlements` packages (16 phase-9 tests). |
| `cd backend && go run ./_tools/phase4_smoke/` | 7/7 dashboard mode scenarios against an in-memory DB + httptest server. |
| `cd backend && go run ./_tools/phase7_verify/` | **Backend on `:8097` required.** 25 cookie-impersonation + plan-CRUD scenarios. |
| `cd backend && go run ./_tools/phase8_verify/` | **Backend on `:8097` required.** `/api/billing/me` + nav completeness audit + regression gates. |
| `cd frontend && npx vitest run` | 18 unit/component tests (phase 9). |
| `cd frontend && NEXT_PUBLIC_API_URL=http://localhost:8080/api npx next build` | Compile-check, full route table. |

The `phase7_verify` and `phase8_verify` tools need a live backend
on port 8097 (start one with `PORT=8097 ... ./tmp_api.exe`
before invoking). Both fresh-DB: they assume no rows present on
boot — running them twice in a row against the same DB will land
on the idempotent catch in plan-create (PLAN_CODE_TAKEN), which is
fine for re-run smoke.

---

## Project conventions

* **No JS/TS `any`.** Backend: Go has no `any` in the module surface;
  if a type escape hatch is unavoidable, it's a 1-line comment with a
  reason. Frontend: prefer `unknown` + Zod-style guards over
  `as any`. Recorded in `ts-no-any.md` style project rule — the
  Vitest config and test files use explicit typed helpers.
* **Commits land locally first.** No push until the user reviews.
  `git log --oneline` carries the whole story.
* **DEFER-001..006** in `IMPLEMENTATION_PLAN.md §10` track known
  tech debt for future passes. Each entry names the file(s), the
  acceptance criteria, and the "logged" date.

---

## File map (high-level)

```
.
├── backend/
│   ├── cmd/api/main.go            ← gin routes wiring (admin, billing, etc.)
│   ├── internal/
│   │   ├── handlers/              ← AuthHandler, BillingHandler, AdminHandler…
│   │   ├── middleware/             ← JWTAuth, RequireRole (admin-only), CORS
│   │   ├── models/                 ← Plan/Subscription/User/AuditLog
│   │   ├── repositories/           ← SQL-by-key wrappers
│   │   ├── services/entitlements/  ← CanCreateSEOProject, ResolveSubscription
│   │   └── db/                     ← AutoMigrate + idempotent seed
│   └── _tools/
│       ├── bootstrap_admin/        ← one-shot ADMIN_EMAIL seeding
│       ├── genhash/                ← bcrypt CLI
│       ├── phase4_smoke/           ← smoke (in-memory)
│       ├── phase7_verify/          ← live-server E2E
│       └── phase8_verify/          ← live + nav_audit + regression
├── frontend/
│   ├── app/(dashboard)/            ← protected pages
│   ├── app/(auth)/                 ← login/register/onboarding
│   ├── components/
│   │   ├── ui/                     ← Button, Card, Dialog, Sheet, Tooltip, …
│   │   └── dashboard/              ← Sidebar, Topbar, PlanBadge, CommandMenu, …
│   ├── lib/
│   │   ├── api-client.ts           ← axios singleton + interceptor
│   │   └── auth-cookie.ts          ← cookie read/write/clear
│   ├── proxy.ts                    ← Next 16 middleware (CSRF + cookie gate)
│   └── vitest.config.ts            ← jsdom + RTL
├── docker-compose.yml
├── nginx.conf
├── IMPLEMENTATION_PLAN.md          ← live plan + DEFER-001..006 log
├── SECURITY.md
├── CHANGELOG.md                    ← current file
└── README.md                       ← current file
```

---

## License

Internal — no external license granted.
