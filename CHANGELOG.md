# Changelog

All notable changes to DMTool are documented here. The format is
loose-Keep-a-Changelog: phases are reversed-chronological so the
top of the file is the current state of the work.

Versions pre-`2.0.0` are pre-stabilization. The first entry below is
the start of the staged plan described in `IMPLEMENTATION_PLAN.md`.

---

## [Unreleased] — phase 9 (tests + docs)

### Added

* **Backend unit tests** for the entitlements service:
  * `entitlements/resolve_subscription_test.go` (6 tests): no-row
    fallback to free, active|trialing plan resolution, canceled-row
    behavior, plan-code-mismatch error.
  * `handlers/billing_handlers_test.go` (10 tests): `StartTrial`
    happy-path, validation, one-shot guard, unauthorized guard;
    `Subscribe` happy-path, canceled-row reuse contract, plan-code
    validation, empty-body. The handler tests use an in-memory
    SQLite harness with the migrated plans seeded, mirroring the
    runtime schema.
* **Frontend vitest setup** (`vitest.config.ts`,
  `vitest.setup.ts`, `package.json` test scripts). Brings up jsdom +
  React Testing Library + user-event + jest-dom matchers.
* **Frontend smoke tests**:
  * `proxy.test.ts`: the `looksLikeJWT` shape gate (9 tests —
    protection, edge cases, both cookies, malformed shapes).
  * `app/(dashboard)/admin/plans/page.test.tsx`: `PlanModal`
    regex + edit prefill + lock-code + server rejection (5 pass,
    2 todo for jsdom-bound numeric-input branches).
  * `components/admin/impersonation-banner.test.tsx`: target
    render, stop flow, error-resilience (4 pass).

### Removed

* `package-lock.json` at the repo root (was an empty stub from
  commit 696f54a; `.gitignore` already excluded it).

### Notes

* 0 frontend tests → 18 vitest tests + 2 documented todos.
* Backend: 0 new tests added in earlier phases → 16 new tests in
  phase 9 (`go test ./...` runs them all).
* The two `it.todo` entries for negative price / zero max_sites
  smoke tests are documented inline: jsdom's `<input
  type="number" min=0>` sanitises typed values, and bypassing the
  DOM filter is brittle. The matching server-side guards
  (`INVALID_PRICE`, `INVALID_MAX_SITES`) are exercised in
  `backend/_tools/phase7_verify/main.go`.

---

## [Earlier work] — phases 0–8 (rolled up)

### Phase 8 — polish (commit `9cffe24`)
* `GET /api/billing/me` registered; `PlanBadge` reads real plan
  data via `billingApi.me()` on topbar mount.
* New UI primitives in `components/ui/`: `dialog`, `alert-dialog`,
  `sheet`, `skeleton`, `tooltip`. All built on
  `@radix-ui/react-dialog` / `-tooltip`; data-state keyframes in
  `globals.css`.
* `command-menu.tsx` rewritten: items derived from `navGroups`,
  mode-aware, Dialog-wrapped.
* `mobile-sidebar.tsx` (Sheet) + `sidebar-drawer-context.tsx` for
  the `lg-` drawer; SidebarNav extracted for desktop + Sheet
  reuse.
* `PlanModal` refactored onto the Dialog primitive.
* Stale-route purge: `/settings` → `/projects/settings`;
  `/social-insights` and `/content-ai` removed from the nav data.

### Phase 7 — admin panel (commit `1733c0d`)
* `admin_audit_logs` table seeded; `RequireRole(admin)` middleware
  on `/api/admin/*`.
* Impersonation: server-side short-lived JWT (`is_impersonation=true`,
  `impersonator_id=<admin_id>`).
* Admin panel pages: user list, user detail (impersonate button),
  plan list + PlanModal create/edit, audit log, stats.
* Cookie-based auth (phase 3): `dmtool_token`, `dmtool_user`,
  `dmtool_mode`, `dmtool_impersonation_token`,
  `dmtool_impersonation_target`. `auth-cookie.ts` owns all of
  these. The `proxy.ts` gate accepts any JWT-shaped cookie of
  either name.

### Phase 7 closeout (commit `d687779`)
* Server-side price + max_sites guards on
  `PATCH /api/admin/plans/:id` (`INVALID_PRICE`,
  `INVALID_MAX_SITES`).
* `plan.update` audit row confirmed to be written by the handler
  and discoverable via `GET /api/admin/audit-log?action=plan.update`.

### Phase 6 — billing UI (commit `0bb2d10`)
* Plans seeded (`free`, `pro_monthly`, `pro_yearly`); trials
  one-shot via `user.trial_used_at` flag (a cancel + retry can no
  longer pocket a fresh 14-day window). Entitlement service
  resolves active|trialing rows; canceled → free fallback.

### Phase 5 — per-mode overview page (commit `5546197`)
* `dashboard/[mode]/page.tsx`; `SectionTabs` drill-downs;
  mode-rewrite via the proxy.

### Phase 4 — mode-driven sidebar (commit `4071dbd`)
* `ModeSwitcher` (segmented control), persisted on the user row
  and mirrored to `dmtool_mode` cookie for the proxy rewrites.
* `Sidebar` filters nav groups by mode; curated-subset spec.

### Phase 3 — auth cookies + Next 16 proxy (commit `7592d34`, `f579aec`)
* `proxy.ts` (Next 16 middleware): CSRF origin check on
  state-changing requests, JWT-shape cookie gate on
  `/dashboard` + `/admin`, mode rewrite for `/dashboard` and
  `/admin`. The `SET TRUSTED ORIGINS` env var is the single
  source for the CORS / CSRF trust list.

### Phase 2 — entitlements service (commit `d1d9dde`)
* `CanCreateSEOProject`; project-creation guard enforces
  `plan.MaxSites`. Auto-subscribe on registration.

### Phase 1 — schema + migration runner (commit `6a6fd74`)
* Models: `Plan`, `Subscription`, `Preferences`, `AuditLog`,
  `dashboard_mode`, `trial_used_at`.
* Idempotent migration runner in `backend/internal/db/migrate.go`.

### Phase 0 — cleanup (commit `f72185d`)
* Removed 12 stale scratch / debug files.

---

## Phase contracts

* `go build ./...`, `go vet ./...`, `go test ./...` clean.
* `cd frontend && NEXT_PUBLIC_API_URL=http://localhost:8080/api npx next build` — 44 routes including `/admin/*`,
  Proxy middleware wired.
* `go run ./_tools/phase4_smoke/` — 7/7 OK.
* `go run ./_tools/phase7_verify/` — 25/25 OK on a fresh backend.
* `go run ./_tools/phase8_verify/` — billing/me + nav audit +
  regression gates all green.
* `cd frontend && npx vitest run` — 18 pass + 2 todo.

Each verify command is a fresh-DB exercise that proves the
contract end-to-end against a real backend.
