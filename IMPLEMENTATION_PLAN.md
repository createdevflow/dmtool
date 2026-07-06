# Implementation Plan: Onboarding, Semrush-style Dashboard, Plan Gating, Admin Panel

---

## 1. Current Codebase State

### Onboarding (`/onboarding`)
- 4-step wizard under the `(auth)` route group (no sidebar/topbar wrapper)
- Step 1: goal selection — `seo`, `social`, `both` — rendered as 3 cards
- Step 2: asset connection — URL field for SEO, platform toggles for social, both for combined
- Steps 3-4: animated loader, then success screen linking to `/dashboard`
- Backend: `POST /api/onboard` creates a `Project` record (with `Goal` field) and seeds demo data in the background
- **Key gap**: the goal lives on the Project, not the User. The sidebar reads all projects to decide which nav groups to show. There is no user-level "mode" concept.

### Dashboard Shell
- `(dashboard)/layout.tsx` renders `<Sidebar>` + `<Topbar>` + `<CommandMenu>` around children
- Sidebar: 7 nav groups, 22 items, fixed 264px left. SEO Intelligence and Social Media groups hide when no matching projects exist
- Topbar: broken hamburger (no click handler), global search, decorative plan badge (always "Pro"), theme toggle, notification bell, avatar dropdown
- DashboardHeader: project switcher, connection status pills, date range selector
- **No mobile sidebar** — the `lg:hidden` hamburger is dead code

### Auth Architecture
- User model: `ID, Name, Email, Phone, PasswordHash, Role (default:"owner")`
- Role constants defined (`owner`, `admin`, `viewer`) but **zero enforcement** — every registrant is `owner`, no middleware checks roles
- JWT RS256 (15min) + opaque refresh tokens (7-day, HttpOnly cookie, SHA-256 hashed)
- No Next.js middleware, no auth context provider. Dashboard pages load for unauthenticated visitors (API calls fail with 401)
- Logout only clears localStorage — refresh token not revoked server-side

### Billing/Plans — Nearly Nonexistent
- Static pricing page: Standard ($0), Professional ($29) — both link to `/register`, no checkout flow
- Topbar badge reads `user?.plan || "Pro"` — plan field doesn't exist on User model, so always shows "Pro"
- Sidebar has a "Billing" link pointing to a nonexistent page (404)
- "Export CSV — Pro+" disabled button in project settings (no backend support)
- No plan model, no feature gating, no Stripe, no subscription management

### Component Library
- **Have**: Card, Button (6 variants), Badge, Avatar, Input, Label, Select, Tabs, Switch, Slider, Progress, Toaster, DropdownMenu
- **Need to build**: Dialog/Modal, Sheet (drawer), AlertDialog, Skeleton, DataTable, Breadcrumb, Tooltip, Popover
- **Charts**: Recharts (Area, Bar, Line, Pie) — already used with consistent styling

---

## 2. Sidebar & Routing: Mode-based Dashboard

### Mode Persistence
Add a `DashboardMode` field to the User model (values: `search | social | combined`, default `combined`). This gets persisted via `PATCH /api/auth/me` and returned with the user profile. The frontend mirrors it in localStorage for instant reads.

### Mode Switcher Component
Place a segmented 3-button control in the sidebar header (below the DMTool logo). Each button labels one mode. Clicking updates both server-side (API call) and client-side (localStorage + route navigation).

### Bookmarkable Routes
Each mode gets its own URL path — `/dashboard/search`, `/dashboard/social`, `/dashboard/combined`. The bare `/dashboard` path redirects to the user's saved mode. This preserves deep-linking and browser history per mode.

### Nav Group Visibility by Mode

| Group | Search | Social | Combined |
|-------|--------|--------|----------|
| Overview (Dashboard, AI Insights, Action Center, Alerts) | Show | Show | Show |
| Project Management | Show | Show | Show |
| SEO Intelligence | Show | Hide | Show |
| Social Media | Hide | Show | Show |
| AI Engine | Show | Show | Show |
| Analytics & Reports | Show | Show | Show |
| System | Show | Show | Show |

### Combined Mode Recommendation
**Merge both full sidebars.** Here's the tradeoff analysis:

- **Curated subset approach**: fewer items, less overwhelm. But users who picked Combined explicitly want both capabilities. Hiding items they can access creates confusion — they'd need to switch modes to find a tool they know exists.
- **Full merge approach**: all SEO + all Social items visible. The nav grows from 22 to ~26 items. Semrush itself shows 30+ sidebar items. With collapsible groups, the visual load stays manageable.
- **Recommendation**: Full merge. The collapsible groups already handle density. Combined users expect complete access. Artificially hiding items undermines the "unified" promise.

### Implementation Approach
Rather than duplicating the sidebar for each mode, make the existing `Sidebar` component accept the current mode as a prop (or read it from context). Filter the `navGroups` array before rendering. The nav group definitions get a `modes` property indicating which modes include them:

```typescript
{ title: "SEO Intelligence", modes: ["search", "combined"], ... }
{ title: "Social Media", modes: ["social", "combined"], ... }
```

Individual items that genuinely serve both workflows (Dashboard, AI, Analytics, System) have `modes: ["search", "social", "combined"]`.

### Route Structure
```
/dashboard                  → redirect to /dashboard/{user's mode}
/dashboard/search           → Search mode overview
/dashboard/social           → Social mode overview
/dashboard/combined         → Combined mode overview
/seo/*                      → unchanged (visible in Search + Combined)
/social/*                   → unchanged (visible in Social + Combined)
/ai/*                       → unchanged (visible in all modes)
/analytics/*                → unchanged (visible in all modes)
/system/*                   → unchanged (visible in all modes)
/admin/*                    → new (admin-only, see Section 6)
/billing                    → new (plan management page)
```

---

## 3. Data Model Changes

### New Models

#### Plan
```go
type Plan struct {
    ID              uint           `gorm:"primaryKey" json:"id"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
    Name            string         `gorm:"uniqueIndex;not null" json:"name"`          // "free", "pro"
    DisplayName     string         `gorm:"not null" json:"display_name"`              // "Free", "Pro"
    Description     string         `json:"description"`
    PriceMonthly    int            `json:"price_monthly"`                             // cents, 0 for free
    PriceYearly     int            `json:"price_yearly"`                              // cents, 0 for free
    MaxProjects     int            `gorm:"not null" json:"max_projects"`              // 1 for free, 3 for pro
    Features        datatypes.JSON `json:"features"`                                  // JSON blob of feature flags
    IsActive        bool           `gorm:"default:true" json:"is_active"`
    StripeMonthlyID string         `json:"stripe_monthly_id"`                         // placeholder for future Stripe
    StripeYearlyID  string         `json:"stripe_yearly_id"`                          // placeholder for future Stripe
}
```

#### Subscription
```go
type Subscription struct {
    ID               uint       `gorm:"primaryKey" json:"id"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
    UserID           uint       `gorm:"uniqueIndex;not null" json:"user_id"`     // one active sub per user
    PlanID           uint       `gorm:"not null" json:"plan_id"`
    BillingCycle     string     `gorm:"default:monthly" json:"billing_cycle"`    // monthly | yearly
    Status           string     `gorm:"default:active" json:"status"`            // active | trialing | canceled | past_due
    TrialEndsAt      *time.Time `json:"trial_ends_at"`
    CurrentPeriodEnd *time.Time `json:"current_period_end"`
    CanceledAt       *time.Time `json:"canceled_at"`
    StripeSubID      string     `json:"stripe_sub_id"`                           // placeholder
    Plan             Plan       `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}
```

#### AuditLog (for impersonation tracking)
```go
type AuditLog struct {
    ID         uint      `gorm:"primaryKey" json:"id"`
    CreatedAt  time.Time `json:"created_at"`
    ActorID    uint      `gorm:"not null;index" json:"actor_id"`     // admin who performed action
    ActorEmail string    `gorm:"not null" json:"actor_email"`
    TargetID   uint      `gorm:"index" json:"target_id"`             // user affected (0 for non-targeted actions)
    Action     string    `gorm:"not null;index" json:"action"`       // "impersonate", "change_plan", "reset_password"
    Details    string    `json:"details"`                             // JSON or human-readable detail
    IPAddress  string    `json:"ip_address"`
}
```

### Modified Models

#### User — add fields
```go
type User struct {
    // ... existing fields ...
    DashboardMode string `gorm:"default:combined" json:"dashboard_mode"` // search | social | combined
    PlanID        *uint  `json:"plan_id"`                                // nullable, defaults to free plan
    TrialEndsAt   *time.Time `json:"trial_ends_at"`
}
```

### Relationships
- `User` belongs to `Plan` (via `PlanID`)
- `User` has one `Subscription`
- `User` has many `AuditLog` entries (as actor)
- `Plan` has many `Subscription` records

### Migration Strategy
Use GORM AutoMigrate in development (existing pattern). For production, generate SQL migration files from the model diffs. The existing `db.go` already handles auto-migration in dev mode.

---

## 4. Server-side vs UI-only Enforcement

### Server-side (non-negotiable)

| Enforcement Point | Location | Logic |
|---|---|---|
| Website connection limit | `ProjectHandler.Create` and `ProjectHandler.Onboard` | Count user's active projects. If `>= plan.MaxProjects`, return `403 PLAN_LIMIT_EXCEEDED` |
| Plan check middleware | New `middleware.RequirePlan(minPlan)` | Compares user's plan tier against required minimum. Applied to route groups |
| Admin route protection | New `middleware.RequireRole("admin")` | Checks `user_role` from JWT context. Applied to all `/api/admin/*` routes |
| Impersonation logging | Admin handler | Writes AuditLog entry before creating impersonation session |
| Password reset (admin) | Admin handler | Validates admin role, hashes new password, logs action |

### UI-only (enhancement, not security)

| What | Where |
|---|---|
| Upgrade prompt modal | Shown when free user hits project limit in the UI — offers plan comparison |
| Disabled "locked" features | Visual indicators on features unavailable at current plan tier |
| Plan badge in topbar | Reflects user's current plan name |
| Billing page | Shows current plan, usage, upgrade/downgrade options |

**The rule**: the API layer is the source of truth. The UI can suggest, guide, and warn, but a determined user bypassing the UI must still be blocked by the server.

---

## 5. Existing Components to Reuse vs Net-New

### Reuse As-Is
| Component | Used For |
|---|---|
| `Card`, `CardHeader`, `CardContent`, `CardTitle`, `CardDescription` | All dashboard widgets, admin panels, plan cards |
| `Button` (all variants) | Actions, mode switcher, form submissions |
| `Badge` | Status indicators, plan labels, alert counts |
| `Avatar`, `AvatarFallback` | User avatars in admin user list, topbar |
| `Input`, `Label` | Forms (admin user search, plan editing, billing) |
| `Select` | Dropdowns (date range, plan selection, filters) |
| `Tabs` | Sub-views within sections (admin user details, billing history) |
| `Switch` | Feature toggles in plan editing |
| `Progress` | Loading states, completion indicators |
| `DropdownMenu` | Context menus, user menu, action menus |
| Recharts (Area, Bar, Line) | Revenue dashboard charts, trend sparklines |

### Modify/Extend
| Component | Changes Needed |
|---|---|
| `Sidebar` | Accept mode prop, filter nav groups, add mode switcher in header |
| `Topbar` | Wire plan badge to real plan data, fix hamburger for mobile |
| `DashboardHeader` | Mode-aware (hide connection status pills irrelevant to current mode) |
| `GrowthSnapshot` | Already mode-aware (has website/social/combined tabs) — no changes |
| CommandMenu | Fix stale routes, add mode-aware command filtering |

### Net-New Components
| Component | Purpose | Priority |
|---|---|---|
| `Sheet` (side drawer) | Mobile sidebar, admin detail panels, upgrade prompts | High |
| `Dialog` / `AlertDialog` | Confirmation dialogs (delete, impersonate, plan change) | High |
| `Skeleton` | Loading states for all data-heavy pages | Medium |
| `DataTable` | Sortable/filterable tables for admin user list, audit log, plan list | High |
| `ModeSwitcher` | Segmented control in sidebar header | High |
| `PlanBadge` | Dynamic badge reflecting real plan tier | Medium |
| `UpgradePrompt` | Modal shown when hitting plan limits | High |
| `MetricCard` | Reusable stat card with number, trend arrow, optional sparkline | Medium |
| `Breadcrumb` | Navigation context in nested admin pages | Low |
| `Tooltip` | Hover info on locked features, admin actions | Low |

---

## 6. Billing Recommendation: Stub Now, Stripe Later

### What to Build Now

1. **Plan data model** (Plan + Subscription tables) — fully wired, seeded with Free and Pro tiers
2. **Server-side gating** — project limit enforcement, plan-check middleware
3. **Billing UI** — `/billing` page showing current plan, usage (projects used / max), monthly/yearly toggle, upgrade/downgrade buttons
4. **Pricing page redesign** — rewrite the existing `/pricing` page to match Semrush-style with monthly/yearly toggle and real feature comparison
5. **Trial system** — every plan upgrade starts a 14-day trial (`TrialEndsAt` on Subscription). Trial can be canceled without charge.
6. **Admin plan management** — admins can manually assign/change plans for users (useful for support and before Stripe is wired)

### What to Stub (placeholder, no real payment)

- `StripeMonthlyID` and `StripeYearlyID` on Plan — stored as empty strings
- `StripeSubID` on Subscription — stored as empty string
- "Subscribe" button on billing page — for now, immediately activates the plan (simulates successful payment). Add a visible banner: "Billing integration pending — click activates plan at no charge"
- Webhook endpoint — stubbed at `/api/billing/webhook` but returns 501

### Why This Approach

Building the data model and gating logic now means Stripe integration later is purely an additive change: wire the "Subscribe" button to create a Stripe Checkout session, add a webhook handler to update Subscription status, and replace the stubbed activation. The Plan/Subscription models, admin tools, and UI remain unchanged.

### Pricing Tiers

| | Free | Pro |
|---|---|---|
| Connected websites | 1 | 3 |
| Monthly price | $0 | $29/mo placeholder |
| Yearly price | $0 | $249/yr placeholder |
| SEO tools | Basic | Full |
| Social tools | Basic | Full |
| AI features | Limited | Full |
| Export reports | No | Yes |
| Competitor tracking | No | Yes |
| Priority support | No | Yes |

**Note on Pro features**: competitor tracking, keyword position history, deeper crawl frequency, and exportable reports all require backend work that may not exist yet. I recommend marking them as "included in Pro" in the plan model but building the actual feature differentiation in follow-up sprints. The gating infrastructure should be in place; the features behind it can ship incrementally.

---

## 7. Admin Panel

### Route Structure
```
/admin                          → Admin dashboard (user stats, MRR, recent activity)
/admin/users                    → User list with search/filter
/admin/users/:id                → User detail view (profile, subscription, activity)
/admin/plans                    → Plan list, create/edit plans
/admin/plans/:id                → Plan detail/edit form
/admin/audit-log                → Audit log table (impersonations, plan changes, password resets)
```

### Backend API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/admin/users` | List users with pagination, search, plan filter |
| GET | `/api/admin/users/:id` | Get user details with subscription |
| PATCH | `/api/admin/users/:id` | Update user (name, email, role, plan) |
| POST | `/api/admin/users/:id/reset-password` | Generate temporary password, log action |
| POST | `/api/admin/users/:id/impersonate` | Create impersonation JWT, log action |
| POST | `/api/admin/users/:id/stop-impersonation` | End impersonation session |
| GET | `/api/admin/plans` | List all plans |
| POST | `/api/admin/plans` | Create plan |
| PATCH | `/api/admin/plans/:id` | Update plan |
| GET | `/api/admin/audit-log` | List audit log entries with filters |
| GET | `/api/admin/stats` | Revenue dashboard data (MRR, plan breakdown, user growth) |

### Admin Role Enforcement

1. Add `RequireRole("admin")` middleware in `backend/internal/middleware/`
2. Apply it to all `/api/admin/*` routes
3. Check `c.MustGet("user_role")` against `models.RoleAdmin`
4. Return 403 `FORBIDDEN` if role doesn't match

### Admin Account Setup (No Hardcoded Credentials)

Add to `.env.example`:
```
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD_HASH=$2a$12$...   # bcrypt hash, generated offline
```

A one-time setup script (`cmd/setup-admin/main.go`) reads these env vars, checks if a user with that email exists, and if not, creates the user with the `admin` role. This runs once during deployment. The actual credentials never enter the codebase.

### Impersonation Flow
1. Admin clicks "Impersonate" on a user in the admin panel
2. Frontend calls `POST /api/admin/users/:id/impersonate`
3. Backend creates a special short-lived JWT with a claim `impersonator_id` set to the admin's user ID
4. Backend writes an AuditLog entry: `{actor_id, target_id, action: "impersonate", ip_address}`
5. Frontend replaces localStorage token with the impersonation token
6. All subsequent requests run as the target user, but the backend knows the real actor
7. A banner appears at the top: "Impersonating {user name}. Click to stop."
8. "Stop Impersonation" reverts to the admin's real token

### Revenue Dashboard
- MRR: sum of all active Pro subscriptions (monthly price)
- Monthly revenue: same calculation, broken down by billing cycle
- Plan breakdown: count of users per plan
- User growth: new signups per month
- **Initially backed by mock/placeholder data** until Stripe is wired. The admin panel renders charts with the plan model data we have (user counts, plan assignments). When Stripe arrives, real revenue figures replace the calculations.

---

## 8. Risks, Unknowns, and Questions

### Risks
1. **GORM AutoMigrate in production** — the current codebase auto-migrates in dev but has no production migration files. Adding Plan/Subscription/AuditLog tables needs proper migration SQL for production. I'll generate these from the model definitions.
2. **Impersonation security** — the impersonation JWT must be clearly different from a normal JWT (different issuer or a flag claim) to prevent abuse. The middleware should reject impersonation tokens on admin routes to prevent privilege escalation.
3. **Mode switching race conditions** — if the user clicks a mode switcher rapidly, the API call and navigation could interleave. The implementation should debounce or disable the switcher during the transition.
4. **Mobile sidebar** — currently non-functional. Building it as part of this work is necessary since the mode switcher lives in the sidebar. I'll use a Sheet component (Radix Dialog-based drawer) triggered by the hamburger button.

### Unknowns
- **Stripe pricing**: exact dollar amounts for Pro tier (I'm using $29/mo, $249/yr as placeholders)
- **Trial length**: I've proposed 14 days — let me know if you prefer a different duration
- **Admin panel scope**: should it include any billing-specific views (e.g., viewing a user's payment history), or is the plan management + audit log sufficient for now?
- **Pro feature specifics**: the feature list in Section 6 includes items I noted as "may not exist yet." Which Pro features are already built vs need scaffolding?

### What I Need From You Before Implementation
1. Confirm the Combined mode recommendation (full merge vs curated subset)
2. Confirm billing approach (stub now, Stripe later)
3. Confirm trial length preference
4. Confirm pricing placeholders or provide real numbers
5. Any additional admin panel features beyond what's listed
6. Priority ordering — which of the 4 major areas (onboarding, dashboard layout, plan gating, admin) should I tackle first?

---

## 9. Suggested Implementation Order

1. **Data models + migrations** — Plan, Subscription, AuditLog, User model updates. Foundation everything else depends on.
2. **Mode switcher + sidebar filtering** — User model update, mode API endpoint, sidebar refactor, route restructuring to `/dashboard/{mode}`.
3. **Plan gating (server-side)** — `RequirePlan` middleware, project limit enforcement in Create/Onboard handlers, plan seeding.
4. **Billing UI** — `/billing` page, plan assignment, upgrade flow (stubbed), pricing page redesign.
5. **Admin panel** — Admin routes, RequireRole middleware, user management, plan management, impersonation, audit log, revenue dashboard.
6. **Polish** — Mobile sidebar (Sheet), Dialog/AlertDialog components, Skeleton loading states, CommandMenu fixes, topbar wiring to real plan data.

Each step produces a working increment. Steps 1-3 are backend-heavy. Steps 4-6 are frontend-heavy. Steps 1-2 can partially overlap.

## 10. Deferred Items (defer log — not in any current phase)

These are intentional technical debts that surfaced during phase 1–3.
They live here so future passes can find and address them.

### DEFER-001 — Frontend UI-state localStorage → cookie or context

**What**: 14 dashboard pages read `dmtool_active_project_id`,
`dmtool_active_platform`, and `dmtool_selected_project` from
`localStorage` to remember the last-viewed project / platform across
navigations. The auth-token localStorage path was migrated to cookies
in phase 3. **This UI-state path was deliberately left as-is** — it
carries no security weight (XSS-side deanonymisation only) and
migrating it requires either (a) a React Context provider in
`(dashboard)/layout.tsx`, or (b) a query-string URL contract on every
route. Both are larger refactors and a deliberate phase of their own.

**Files (14)**:
- `frontend/app/(dashboard)/ai-insights/page.tsx`
- `frontend/app/(dashboard)/alerts/page.tsx`
- `frontend/app/(dashboard)/analytics/custom/page.tsx`
- `frontend/app/(dashboard)/analytics/traffic/page.tsx`
- `frontend/app/(dashboard)/dashboard/page.tsx`
- `frontend/app/(dashboard)/projects/page.tsx` (writes `dmtool_selected_project`)
- `frontend/app/(dashboard)/seo/backlinks/page.tsx`
- `frontend/app/(dashboard)/seo/rank-tracking/page.tsx`
- `frontend/app/(dashboard)/social/competitors/page.tsx`
- `frontend/app/(dashboard)/social/growth/page.tsx` (also `dmtool_active_platform`)
- `frontend/app/(dashboard)/social/insights/page.tsx`
- `frontend/app/(dashboard)/social/profile-analyzer/page.tsx` (also `dmtool_active_platform`)
- `frontend/app/(dashboard)/system/calendar/page.tsx`
- `frontend/app/(dashboard)/automations/page.tsx` (likely; verify)

**Why deferred**: scope. Phase 3's focus was the auth-token
single-source-of-truth. UI state does not reduce security; the
auth-cookie transition already removed the larger attack surface.

**Acceptance criteria for the future pass**:
1. UI state no longer reads/writes `localStorage`.
2. Single source of truth: either a Context provider OR URL
   query-strings (project id on every dashboard route).
3. All 14 files migrated; one of them in the diff is `system/calendar`
   which already has a slightly different write style (`window.localStorage`
   with explicit `typeof` guard) — handle that case too.

**Logged**: 2026-07-03, end of phase 3.

### DEFER-002 — Production environment variables

**What**: Before the final `git push origin main` at the end of all
phases, set `ALLOWED_ORIGINS` on Render **and** Vercel to the same
production URL(s). Both sides must agree, or browser CORS will fail.

**Where**:
- Render dashboard → dmtool-backend service → Environment →
  add `ALLOWED_ORIGINS=https://dmtool-eight.vercel.app`
- Vercel dashboard → dmtool project → Settings → Environment
  Variables → `ALLOWED_ORIGINS=https://dmtool-eight.vercel.app`

**Frontend picks it up via**: `process.env.ALLOWED_ORIGINS` in
`proxy.ts` (proxy runs on the edge).
**Backend picks it up via**: `cfg.AllowedOrigins` in
`internal/config/config.go` (used by `middleware.CORS`).

**Logged**: 2026-07-03, end of phase 3.

### DEFER-003 — Overview tab in per-mode overview is mildly redundant

**What**: The per-mode overview pages
(`app/(dashboard)/dashboard/[mode]/page.tsx`) have an "Overview"
tab that links to `/dashboard`. The proxy at `frontend/proxy.ts`
rewrites `/dashboard` → `/dashboard/<mode>`, so the user lands on
the same per-mode overview they were just on. Visually they
navigate (URL changes, tab bar updates), but the content is
identical.

**Why deferred**: the real fix — duplicating the existing
`app/(dashboard)/dashboard/page.tsx` (~389 lines of project-aware
dashboard content) into the per-mode overview so that the
"Overview" tab actually surfaces mode-specific summary — is a real
refactor. The 389 lines of the existing dashboard pull metrics,
insights, tasks, snapshot, charts, action center items, and AI
insight cards. None of that is mode-aware today. Adding the mode
filter at the data layer (separate metrics for SEO vs Social) is
its own design decision that should be a separate plan item.

**Current behavior confirmed**: click → 200 OK → page renders
identical content. Not a dead link, not an error state. Just
redundant navigation the proxy introduces as a side effect of the
mode-rewrite rule.

**Acceptance criteria for the future pass**:
1. The "Overview" tab on per-mode overviews renders content that
   actually differs by mode (e.g. SEO summary for search, Social
   summary for social, both for combined).
2. The duplication of dashboard data into mode-specific pages is
   done via a shared component, not copy-paste.
3. The metrics endpoint (or a new mode-aware endpoint) returns the
   right data for the active mode.

**Logged**: 2026-07-03, end of phase 5.

### DEFER-004 — Admin user suspension / deletion

**What**: the admin panel can edit a user's profile (name/email/role),
reset their password, and impersonate them, but it cannot yet
**suspend** (set a `disabled_at` column that the JWT middleware
honors) or **delete** the user. Deletion is non-trivial because
of the cascade to `projects`, `subscriptions`, `oauth_credentials`,
and `admin_audit_logs` — soft-delete is the safer default.

**Why deferred**: explicit scope cap. The current brief asked for
user management surfaces, not moderation. Suspension is a
useful next step but requires (a) a `disabled_at` column on
`User` (forward migration), (b) a `RequireUserEnabled` middleware
applied after `JWTAuth`, and (c) the admin-side UI.

**Acceptance criteria for the future pass**:
1. `users.disabled_at` column added; AutoMigrate handles dev.
2. A `RequireUserEnabled` middleware on the protected `/api` group
   that rejects disabled users with 403 `ACCOUNT_DISABLED`.
3. PATCH `/api/admin/users/:id {disabled: true}` and the
   corresponding list-column display.
4. Soft-delete endpoint that stamps `deleted_at` on the user and
   revokes all refresh tokens.

**Logged**: 2026-07-06, end of phase 7.

### DEFER-005 — SubscriptionEvent history table

**What**: today the `subscriptions` table is the single source of
truth for a user's current plan; `phase 1` ships the
`admin_audit_logs` table for admin actions but there is no
**event** table that records "user upgraded from pro_monthly to
pro_yearly on 2026-07-01" or "trial converted to paid on ...". The
admin panel can show recent admin actions but cannot show a user's
subscription timeline over time.

**Why deferred**: not in the explicit scope of the admin panel.
Today the row mutation happens via `subscriptionRepo.UpdatePlanAndStatus`
which updates the existing row in place; an `event` table would
require either (a) an `append_event` on every mutation (adds a
write per change), or (b) a trigger. Either is a small amount of
code but needs design decisions about retention and access.

**Acceptance criteria for the future pass**:
1. New `subscription_events` table (`id, subscription_id, from_plan,
   to_plan, from_status, to_status, occurred_at, actor_user_id`).
2. `BillingHandler.Subscribe` and `StartTrial` write a row on every
   successful state change.
3. New admin endpoint `GET /api/admin/users/:id/subscription-history`
   returns the events.
4. UI on `/admin/users/:id` renders the timeline.

**Logged**: 2026-07-06, end of phase 7.

### DEFER-006 — Auto-bootstrap admin on first boot

**What**: `backend/_tools/bootstrap_admin/main.go` exists as a
standalone seeder (idempotent: re-runs are no-ops). It is not
called automatically anywhere. Some deployment flows want the
admin user to be created on first server boot when `ADMIN_EMAIL`
is set, so a fresh Render deploy with the env var configured
doesn't require a separate `go run ./_tools/bootstrap_admin/` step.

**Why deferred**: idempotency makes it safe to call, but a render
hook adds a side effect to every boot. Running it in `main.go`
before `r.Run()` would mean the server blocks boot for ~100ms
on the bcrypt lookup. Some teams prefer this; some prefer an
explicit deploy step. Out of scope for phase 7's admin UI.

**Acceptance criteria for the future pass**:
1. In `cmd/api/main.go`, after `db.Init`, check `os.Getenv("ADMIN_EMAIL")`.
   If set and the user doesn't exist, call the bootstrap logic and
   log the result.
2. Keep the standalone `_tools/bootstrap_admin/` working for
   manual recovery.
3. Document the auto-bootstrap in `DEPLOY_CONTEXT.md`.

**Logged**: 2026-07-06, end of phase 7.

