# DMTool — Unified Digital Marketing Intelligence Portal
## Full-Stack Architecture & Engineering Blueprint
**Version 2.0 — Production-Ready Specification**

| Property | Value |
|---|---|
| Stack | Next.js 14 · Go (Gin) · SQLite / PostgreSQL |
| Auth | JWT RS256 + Refresh Tokens + CSRF |
| Security | OWASP Top-10 compliant, rate-limited, helmet headers |
| APIs | Google Search Console · Meta Graph API · OpenAI |
| Deployment | Docker + Nginx · GitHub Actions CI/CD |

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [System Architecture & Design Principles](#2-system-architecture--design-principles)
3. [Security Architecture](#3-security-architecture)
4. [Complete User & Application Flow](#4-complete-user--application-flow)
5. [Backend — Go (Gin) Specification](#5-backend--go-gin-specification)
6. [Frontend — Next.js 14 Specification](#6-frontend--nextjs-14-specification)
7. [Database Schema & Migrations](#7-database-schema--migrations)
8. [API Contracts (All Endpoints)](#8-api-contracts-all-endpoints)
9. [Third-Party Integration Blueprints](#9-third-party-integration-blueprints)
10. [Dashboard Pages — Feature Specifications](#10-dashboard-pages--feature-specifications)
11. [AI Content Engine](#11-ai-content-engine)
12. [Error Handling & Observability](#12-error-handling--observability)
13. [Testing Strategy](#13-testing-strategy)
14. [DevOps & Deployment](#14-devops--deployment)
15. [Setup Guide (Developer Quickstart)](#15-setup-guide-developer-quickstart)

---

## 1. Executive Summary

DMTool is an enterprise-grade, multi-tenant SaaS platform that centralises digital marketing intelligence across Google Search Console and Meta (Facebook / Instagram). This document is the single source of truth for all engineering decisions: architecture, security, flows, API contracts, UI specifications, and DevOps pipelines.

> **v2.0 Note:** This blueprint corrects all known architectural gaps in the original design. Every page is functional, every API endpoint is secured, every third-party call is gracefully handled, and every user interaction produces accurate, real-time data.

| Goal | Implementation Approach |
|---|---|
| Functional Dashboard | Real data from GSC + Meta with seeded fallback; full CRUD on all entities |
| Zero Broken Pages | Strict error boundaries, skeleton loaders, and 404/500 fallback routes in Next.js |
| Security Hardened | RS256 JWT, refresh tokens, CSRF, rate limiting, RBAC, input sanitisation, CSP headers |
| Accurate Data | Live API polling with SWR, delta-comparison snapshots, timezone-aware metrics |
| Developer DX | One-command setup, typed API client, Zod validators, automatic DB migrations |

---

## 2. System Architecture & Design Principles

### 2.1 High-Level Architecture

DMTool follows a clean, layered architecture with strict separation of concerns between the presentation layer (Next.js), the application layer (Go/Gin), and the data layer (SQLite in dev / PostgreSQL in production).

> **Architectural Rule:** Every layer communicates only with its adjacent layer. The frontend never touches the database. The backend never renders HTML. External APIs are always wrapped in a dedicated service layer with retry logic, timeouts, and structured error mapping.

| Layer | Technology | Responsibility |
|---|---|---|
| Presentation | Next.js 14 App Router | Server Components + Client Islands; route protection; optimistic UI |
| API Gateway | Go / Gin | JWT validation, rate limiting, request routing, response normalisation |
| Business Logic | Go Services | Orchestration of DB + third-party APIs; background workers |
| Data Access | GORM + Repository Pattern | Typed queries, migrations, multi-tenant scoping by UserID+ProjectID |
| External APIs | GSC, Meta, OpenAI | Wrapped in retry-capable HTTP clients with exponential back-off |
| Cache | In-process + Redis (prod) | Token cache, API response cache with TTLs per endpoint |

### 2.2 Multi-Tenancy & Data Isolation

Every database query is scoped by both `UserID` (from JWT claims) and `ProjectID` (from validated query parameters). There is no global query that returns cross-user data under any code path.

- `UserID` is extracted exclusively from the validated JWT — never from request body or query params
- `ProjectID` is validated against the authenticated user's project list before any DB operation
- GORM scopes enforce automatic `WHERE user_id = ? AND project_id = ?` on all relevant models
- SQL injection is prevented by GORM's parameterised queries — raw SQL is forbidden in application code

### 2.3 Background Workers

Three background goroutines run on a configurable schedule to keep data fresh without blocking HTTP requests.

| Worker | Interval | Job |
|---|---|---|
| MetricsSyncer | Every 6 hours | Pulls GSC search analytics and Meta insights; writes to Metrics table |
| HealthScorer | Every 24 hours | Re-runs SEO audit on each project URL; updates `Project.HealthScore` |
| InsightGenerator | Every 12 hours | Calls OpenAI with latest metrics to produce fresh AI insights |

---

## 3. Security Architecture

Security is not a feature — it is a foundational constraint applied at every layer. The following specification is OWASP Top-10 compliant.

### 3.1 Authentication & Token Strategy

| Component | Specification |
|---|---|
| Access Token | RS256 JWT, 15-minute TTL, signed with private key stored in env; contains `UserID`, `Email`, `Role` claims |
| Refresh Token | Opaque 256-bit random token, 30-day TTL, stored in DB with device fingerprint |
| Storage (FE) | Access token in memory (React context) only. Refresh token in `HttpOnly`, `Secure`, `SameSite=Strict` cookie |
| Rotation | Refresh token is rotated on every use (single-use); old token is immediately invalidated |
| CSRF | Double-submit cookie pattern for all state-changing requests from the browser |
| Logout | Clears memory token + calls `/auth/logout` to server-side invalidate refresh token in DB |

### 3.2 HTTP Security Headers (via Gin Middleware)

```
Content-Security-Policy: default-src 'self'; restrict inline scripts and external origins
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer-when-downgrade
Permissions-Policy: camera=(), microphone=(), geolocation=()
```

### 3.3 Rate Limiting

| Endpoint Group | Limit | Window |
|---|---|---|
| `/api/auth/login`, `/api/auth/register` | 5 requests | Per IP per 15 minutes (strict) |
| `/api/content/generate` (AI) | 20 requests | Per user per hour |
| `/api/seo/audit/run` | 10 requests | Per user per hour |
| All other `/api/*` routes | 200 requests | Per user per minute |

### 3.4 Input Validation & Sanitisation

- All request bodies are validated using struct tags with `go-playground/validator` before handler logic runs
- All text inputs that may be rendered in the frontend are sanitised with `bluemonday` HTML sanitiser to prevent XSS
- File uploads (if any) are validated for MIME type, file size (max 5 MB), and stored outside the web root
- OAuth tokens from third parties are encrypted at rest using AES-256-GCM before storage in `OAuthCredential` table

### 3.5 Frontend Route Protection

Next.js middleware runs before every request. It validates the presence and structure of the in-memory access token and performs a lightweight JWT signature check. On failure it redirects to `/login` with a `returnUrl` parameter.

- **Protected routes:** `/dashboard/**`, `/onboarding/**`
- **Public routes:** `/`, `/login`, `/register`, `/api/auth/**`
- API calls from Server Components use server-side session validation — no token exposed to browser on SSR pages

---

## 4. Complete User & Application Flow

### 4.1 Registration & First-Time Onboarding

1. User lands on `/register`. Form: Full Name, Email, Password (min 8 chars, 1 uppercase, 1 special char). Client-side Zod validation fires before submit.
2. `POST /api/auth/register` — backend hashes password with bcrypt (cost 12), creates User record, returns access token + sets refresh cookie.
3. Frontend stores access token in `AuthContext`. User is redirected to `/onboarding/step-1`.
4. **Onboarding Step 1:** Project Name, Website URL (validated as real URL format), Primary Goal (SEO / Social / Both). `PATCH /api/projects` creates the project record.
5. **Onboarding Step 2:** Social Handles — Instagram, Facebook Page ID, Twitter, LinkedIn. Optional. Stored on Project model.
6. **Onboarding Step 3:** Connect Integrations — OAuth buttons for Google Search Console and Meta. Users can skip and connect later from the Integrations page.
7. On completion, backend auto-seeds 30 days of realistic metric data if no real API is connected. User is redirected to `/dashboard`.

### 4.2 Returning User Login

1. `POST /api/auth/login` — validates credentials, returns new access token in response body + rotates refresh cookie.
2. Frontend stores access token in memory. `AuthContext` triggers re-render of all protected route guards.
3. **Silent refresh:** an Axios interceptor catches 401 responses, calls `POST /api/auth/refresh`, and retries the original request transparently.

### 4.3 Dashboard Data Loading Flow

1. User arrives at `/dashboard`. Next.js Server Component fetches `GET /api/dashboard/snapshot` (server-side, token from secure cookie).
2. Snapshot returns: clicks, impressions, reach, engagement, followers, health score, today's AI task, and delta vs. previous period.
3. Client Components hydrate with SWR for live polling: snapshot refreshes every 5 minutes; charts refresh every 15 minutes.
4. Project switcher in Topbar dispatches a global event; all SWR hooks re-fetch with the new `project_id`.
5. If a third-party API call fails during snapshot fetch, the backend returns cached data from the last successful sync with a `degraded: true` flag. Frontend renders a subtle banner informing the user.

### 4.4 OAuth Integration Flow (Google / Meta)

1. User clicks "Connect Google Search Console" on Integrations page.
2. Frontend calls `GET /api/integrations/google/auth-url`. Backend generates OAuth URL with `state` parameter (CSRF token) and returns it.
3. User is redirected to Google consent screen, then back to `/integrations/callback?code=...&state=...`
4. Frontend sends the code and state to `POST /api/integrations/google/callback`. Backend validates state, exchanges code for tokens, encrypts and stores in `OAuthCredential`, immediately triggers first data sync.
5. On success: Toaster shows "Google Search Console connected!" and the integration card updates to "Connected" status.

---

## 5. Backend — Go (Gin) Specification

### 5.1 Directory Structure

```
cmd/api/main.go              — Entry point: initialises DB, starts workers, starts Gin server
internal/
  config/config.go           — Env-based config loading (Viper)
  middleware/                — auth.go, ratelimit.go, security.go, cors.go, logger.go
  handlers/                  — auth.go, onboarding.go, dashboard.go, seo.go, social.go,
                               content.go, integrations.go, tasks.go, projects.go
  services/                  — thirdparty/ (gsc.go, meta.go, openai.go), seo/analyzer.go,
                               metrics/syncer.go, insights/generator.go
  models/                    — All GORM models (user.go, project.go, metric.go, etc.)
  repository/                — Repository pattern: user_repo.go, project_repo.go, etc.
  utils/                     — jwt.go, crypto.go, response.go, pagination.go, seeder.go
  workers/                   — metrics_syncer.go, health_scorer.go, insight_generator.go
```

### 5.2 Middleware Stack (Applied in Order)

| Middleware | What It Does |
|---|---|
| SecurityHeaders | Injects all OWASP-recommended HTTP response headers on every response |
| CORS | Allows only the configured frontend origin; rejects others with 403 |
| Logger | Structured JSON logging (zerolog) with request ID, latency, status code |
| RateLimiter | Per-IP sliding window using `golang.org/x/time/rate`; 429 on breach |
| JWTAuth | Validates RS256 signature, checks expiry, extracts claims into Gin context |
| ProjectGuard | On project-scoped routes: verifies user owns the requested `project_id` |
| Recover | Catches panics; returns 500 JSON; logs stack trace |

### 5.3 Standardised Response Envelope

All API responses use a consistent JSON envelope to make frontend error handling deterministic.

```json
// Success
{
  "success": true,
  "data": { ... },
  "meta": { "page": 1, "total": 42 }
}

// Error
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Email is required.",
    "field": "email"
  }
}
```

---

## 6. Frontend — Next.js 14 Specification

### 6.1 Directory Structure

```
app/
  (auth)/
    login/page.tsx           — Login form with Zod validation + silent refresh init
    register/page.tsx        — Register form + redirects to onboarding
  (dashboard)/
    layout.tsx               — Protected layout: checks AuthContext; renders Sidebar + Topbar
    page.tsx                 — Dashboard home: Growth Snapshot, AI Focus, Trend Charts
    seo/
      explorer/page.tsx      — Site Explorer: real-time SEO audit runner
      keywords/page.tsx      — Keyword Research with table + KD badges
      audit/page.tsx         — Audit Report: issues list with severity filter
    social/
      page.tsx               — Social Media Analyzer: channel overview + profile cards
    content/page.tsx         — Content AI: prompt builder + generated results
    integrations/page.tsx    — OAuth connect cards + status indicators
    settings/page.tsx        — Profile, password change, project management
  onboarding/
    step-[1-3]/page.tsx      — Multi-step onboarding wizard
  (public)/
    page.tsx                 — Landing page
components/
  ui/                        — Design system: Button, Input, Badge, Card, Toaster, Skeleton
  dashboard/                 — Snapshot, TrendChart, InsightCard, TaskList, ProjectSwitcher
  seo/                       — AuditTable, KeywordTable, SiteExplorerForm
  social/                    — ChannelCard, MetricBadge, ProfileStats
  content/                   — PromptBuilder, GeneratedOutput, PlatformPicker
lib/
  api.ts                     — Typed Axios client with interceptors
  auth-context.tsx           — AuthContext provider with token store + refresh logic
  swr-keys.ts                — Centralised SWR cache key factory
  validators.ts              — Shared Zod schemas matching backend validation
```

### 6.2 Data Fetching Strategy

| Scenario | Approach |
|---|---|
| Initial page load (SEO-sensitive) | Next.js Server Component fetches data server-side; HTML shipped pre-rendered |
| Live dashboard data | SWR with 5-min revalidation; skeleton shown on first load |
| User-triggered actions | `SWR mutate()` for optimistic updates; revert on error with toast |
| Slow/expensive calls (SEO audit) | Streaming response via `ReadableStream`; progress bar shown in UI |
| Form submissions | `react-hook-form` + Zod resolver; errors mapped field-by-field |

### 6.3 Error Handling & UX Guardrails

- Every async operation is wrapped in a try/catch that maps API error codes to user-friendly toast messages
- Every page has an `error.tsx` boundary (Next.js App Router) that catches render errors and shows a styled error card with a "Retry" button
- Loading states use consistent `Skeleton` components — no flickering blank areas
- Network offline detection: a banner appears when `navigator.onLine` is false; SWR retries automatically when connectivity resumes
- All external links use `rel="noopener noreferrer"` to prevent tab-napping

---

## 7. Database Schema & Migrations

### 7.1 Core Models

| Model | Key Fields | Notes |
|---|---|---|
| User | ID, Name, Email (unique), PasswordHash, Role, CreatedAt | Role: `owner \| admin \| viewer` (RBAC ready) |
| RefreshToken | ID, UserID, Token (hashed), ExpiresAt, Revoked, DeviceInfo | Revoked on every rotation; old tokens purged by cron |
| Project | ID, UserID, Name, URL, Goal, HealthScore, IGHandle, FBPageID, etc. | Cascade delete removes all child records |
| OAuthCredential | ID, UserID, Provider, AccessToken (enc), RefreshToken (enc), ExpiresAt | AES-256-GCM encrypted before storage |
| Metric | ID, ProjectID, Date, Clicks, Impressions, Reach, Engagement, Source | Source: `gsc \| meta \| seed`; Date index for range queries |
| SocialMetric | ID, ProjectID, Platform, Followers, Reach, Engagement, RecordedAt | Point-in-time snapshots for growth tracking |
| Insight | ID, ProjectID, Type (CRITICAL/OPPORTUNITY), Title, Body, Priority, CreatedAt | Type maps to badge colour in UI |
| Task | ID, ProjectID, Title, Completed, DueDate, Source (ai/manual) | Source tracks whether AI or user created the task |
| SEOIssue | ID, ProjectID, URL, Severity (high/med/low), Category, Detail, ResolvedAt | `ResolvedAt` null = open issue |
| KeywordResult | ID, ProjectID, Seed, Keyword, Volume, KD, Position, UpdatedAt | Cached results with 24-hour TTL |

### 7.2 Migration Strategy

- GORM `AutoMigrate` runs on startup in development only. Production uses numbered SQL migration files in `/migrations`
- Migration naming: `001_create_users.sql`, `002_create_projects.sql`, etc. — run in order by the `migrate` CLI
- Every migration is idempotent (`CREATE TABLE IF NOT EXISTS`, `ADD COLUMN IF NOT EXISTS`)
- Rollback scripts (down migrations) are included for every migration file

---

## 8. API Contracts (All Endpoints)

### 8.1 Authentication

| Method | Path | Request Body | Response |
|---|---|---|---|
| POST | `/api/auth/register` | `{ name, email, password }` | `{ user, accessToken }` + `Set-Cookie: refresh` |
| POST | `/api/auth/login` | `{ email, password }` | `{ user, accessToken }` + `Set-Cookie: refresh` |
| POST | `/api/auth/refresh` | `Cookie: refresh_token` | `{ accessToken }` + rotated cookie |
| POST | `/api/auth/logout` | `Cookie: refresh_token` | `204 No Content`; cookie cleared |
| GET | `/api/auth/me` | Bearer token | `{ user }` |

### 8.2 Projects

| Method | Path | Request Body | Response |
|---|---|---|---|
| GET | `/api/projects` | — | `[ { id, name, url, goal, healthScore } ]` |
| POST | `/api/projects` | `{ name, url, goal, handles }` | `{ project }` + triggers auto-seed |
| PATCH | `/api/projects/:id` | Partial project fields | `{ project }` |
| DELETE | `/api/projects/:id` | — | `204`; cascades all child data |

### 8.3 Dashboard & Metrics

| Method | Path | Response |
|---|---|---|
| GET | `/api/dashboard/snapshot?project_id` | `{ clicks, impressions, reach, engagement, followers, healthScore, todayTask, deltas, degraded }` |
| GET | `/api/dashboard/metrics?project_id&from&to` | `{ dates[], clicks[], impressions[], engagement[] }` |
| GET | `/api/dashboard/insights?project_id` | `[ { id, type, title, body, priority } ]` |

### 8.4 SEO Tools

| Method | Path | Notes |
|---|---|---|
| POST | `/api/seo/audit/run` | Body: `{ project_id, url }`. Streams results via SSE. Checks H1, meta desc, OG tags, response time, robots.txt, sitemap, HTTPS, Core Web Vitals. |
| GET | `/api/seo/issues?project_id` | Returns all open `SEOIssue` records, sorted by severity. Supports `?severity=high` filter. |
| POST | `/api/seo/keywords` | Body: `{ project_id, seed }`. Returns `[ { keyword, volume, kd, position } ]`. Results cached 24 hours. |
| GET | `/api/seo/report?project_id` | Returns a structured audit summary: score, issue counts by severity, top issues. |

### 8.5 Social & Content

| Method | Path | Notes |
|---|---|---|
| GET | `/api/social/insights?project_id` | Live Meta API pull: follower count, reach, engagement rate for each connected platform. |
| GET | `/api/social/history?project_id&days=30` | Returns `SocialMetric` snapshots for time-series chart. |
| POST | `/api/content/generate` | Body: `{ project_id, platform, tone, topic }`. Returns generated content variants. Max 20/hr. |

### 8.6 Integrations & System

| Method | Path | Notes |
|---|---|---|
| GET | `/api/integrations` | Returns list of all integrations with connected status, last synced time, and error state. |
| GET | `/api/integrations/google/auth-url` | Returns Google OAuth URL with CSRF state token. State stored in DB for validation. |
| POST | `/api/integrations/google/callback` | Body: `{ code, state }`. Validates, exchanges, stores encrypted tokens. Triggers first sync. |
| GET | `/api/integrations/meta/auth-url` | Same as Google but for Facebook OAuth 2.0 flow. |
| POST | `/api/integrations/meta/callback` | Same as Google callback but for Meta tokens. |
| DELETE | `/api/integrations/:provider` | Revokes and deletes the `OAuthCredential`. Switches metrics source to seeded data. |
| PATCH | `/api/tasks/:id/toggle` | Toggles `task.Completed`. Returns updated task. |

---

## 9. Third-Party Integration Blueprints

### 9.1 Google Search Console

- **OAuth Scopes Required:** `https://www.googleapis.com/auth/webmasters.readonly`
- **Data Pulled:** Search analytics (clicks, impressions, CTR, position) by date, page, and query
- **Polling:** Every 6 hours by the `MetricsSyncer` worker. GSC data lags 2–3 days; the syncer fetches `data[today - 4 days]` to avoid empty responses
- **Error Handling:** On 401, attempt token refresh using stored refresh token. On failure, mark credential as expired and notify user via in-app alert
- **Site Verification:** Before fetching data, verify that the project URL is present in the user's verified GSC properties. If not, return a clear error with a link to add it

### 9.2 Meta (Facebook / Instagram) Graph API

- **OAuth Scopes Required:** `pages_read_engagement`, `instagram_basic`, `instagram_manage_insights`, `read_insights`, `pages_show_list`
- **Data Pulled:** Page fan count, post reach, post impressions, engagement rate, follower delta
- **Instagram Note:** Instagram insights require the account to be a Business or Creator account linked to a Facebook Page
- **Pagination:** All Graph API list responses are paginated using cursor-based pagination. The service automatically follows `next` cursors until all data is fetched
- **Token Long-Lived Exchange:** Short-lived user tokens (60 min) are immediately exchanged for long-lived tokens (60 days) and stored. A worker refreshes them before expiry

### 9.3 OpenAI (Content AI & Insights)

- **Model:** `gpt-4o-mini` for content generation (cost-efficient); `gpt-4o` for insight generation (quality-critical)
- **System Prompt:** Injected with project context (URL, goal, niche, top keywords) to ensure domain-specific output
- **Content Generation:** Returns 3 variants per request. Each variant is scored for clarity, CTA strength, and platform fit by a second LLM pass
- **Insight Generation:** Takes last 30 days of metrics as structured input; outputs prioritised recommendations with specific actions
- **Cost Control:** Token usage is logged per user. A soft cap of $5/month per user is enforced; users get a warning at 80% usage

---

## 10. Dashboard Pages — Feature Specifications

### 10.1 Main Dashboard (`/dashboard`)

- **Growth Snapshot:** 6 KPI cards — Clicks, Impressions (GSC), Reach, Engagement, Followers, Health Score. Each shows current value, delta vs. previous period, and a sparkline
- **Today's Focus:** AI-generated priority task displayed in a prominent card with one-click completion. Completion calls `PATCH /api/tasks/:id/toggle` and animates the card out
- **Trend Chart:** Area chart (Recharts) showing Clicks + Impressions over 30 days. Toggle between GSC and Meta data. Date range picker (7 / 30 / 90 days)
- **Recent Insights:** 3 latest AI insight cards with type badges (`CRITICAL` in red, `OPPORTUNITY` in blue). "View All" links to `/dashboard/insights`
- **Active Tasks:** Checklist of the 5 most recent open tasks. Inline toggle completion

### 10.2 SEO Intelligence (`/dashboard/seo`)

#### Site Explorer
- **Input:** URL (defaults to project URL, editable). "Run Audit" button
- **Results** stream in via SSE: each check appears as it completes (H1, meta desc, OG, response time, HTTPS, sitemap, robots.txt, canonical, structured data)
- **Each result:** icon (pass ✓ / warning ⚠ / fail ✗), label, found value, and recommendation
- **Final score badge:** 0–100. Persisted to `Project.HealthScore` on completion

#### Keyword Research
- **Input:** Seed keyword. Returns table: Keyword, Monthly Volume, KD (0–100 with colour bar), Current Position, Trend
- **KD Colour:** 0–30 green (easy), 31–60 yellow (medium), 61–100 red (hard)
- **Export:** CSV download of the full keyword table

#### Audit Report
- Table of all open `SEOIssue`s for the project. Columns: URL, Severity (badge), Category, Issue Detail, Date Found, Status (open/resolved)
- Filter by severity. Mark as resolved inline (calls `PUT /api/seo/issues/:id`)

### 10.3 Social Media Analyzer (`/dashboard/social`)

- **Channel Overview:** Side-by-side cards for each connected platform (Facebook, Instagram). Shows platform icon, follower count, 30-day reach, engagement rate, and a mini sparkline
- **Profile Analysis:** Clicking a platform card expands to show demographic breakdown, top posts by engagement, and posting frequency analysis
- **Comparison Chart:** A grouped bar chart comparing reach and engagement across all connected platforms for the last 30 days
- **Connect Prompt:** If a platform is not connected, the card shows a "Connect" CTA that opens the integrations OAuth flow

### 10.4 Content AI (`/dashboard/content`)

- **Platform Picker:** LinkedIn, Twitter/X, Instagram Caption, Blog Title, Email Subject Line
- **Tone Selector:** Professional, Casual, Persuasive, Informative, Witty
- **Topic Input:** Free-text topic (e.g., "our new product launch", "SEO tips for 2025")
- **Output:** 3 generated variants displayed in styled cards. Each card has a "Copy" button, "Regenerate this" button, and character count badge
- **History:** Last 20 generated pieces are saved to local storage and shown in a collapsible history panel

### 10.5 Integrations (`/dashboard/integrations`)

- Two integration cards: Google Search Console and Meta. Each shows: logo, connection status (badge), last synced time, sync status (success / error / pending)
- **Connected state:** "Disconnect" button (with confirmation modal). "Force Sync" button triggers immediate background sync
- **Disconnected state:** "Connect" button initiates OAuth flow
- **Error state:** Red error message with the specific reason (e.g., "Token expired — please reconnect") and a "Reconnect" button

### 10.6 Settings (`/dashboard/settings`)

- **Profile Tab:** Edit Name, Email. Save triggers `PATCH /api/auth/me`
- **Password Tab:** Change password form with current password confirmation. bcrypt re-hashed on backend
- **Projects Tab:** List all projects with edit (name, URL, goal, handles) and delete (with confirmation)
- **Danger Zone:** "Delete Account" — requires typing `CONFIRM`, sends `DELETE /api/auth/account`, clears all data, and redirects to landing page

---

## 11. AI Content Engine

### 11.1 Content Generation Pipeline

1. Frontend sends `POST /api/content/generate` with `{ project_id, platform, tone, topic }`
2. Handler validates input, checks rate limit, fetches project context (URL, goal, top 5 keywords from last audit)
3. Constructs system prompt: *"You are a marketing expert for [project URL], a [goal] focused project in the [niche] space. Your top-performing keywords are: [keywords]."*
4. Calls OpenAI API with structured output request; parses 3 variants from response
5. Each variant is scored (0–10) for: platform fit, clarity, CTA strength. Score is returned with the content
6. Response is streamed back to frontend for low perceived latency

### 11.2 Insight Generation Pipeline

1. `InsightGenerator` worker runs every 12 hours for all active projects
2. Fetches last 30 days of `Metrics` + `SocialMetrics` for the project
3. Computes key signals: traffic trend direction, engagement rate delta, keyword position changes, health score trend
4. Sends structured signals JSON to OpenAI with instruction to return 3–5 prioritised insights as a JSON array
5. Parsed insights are written to the `Insight` table, replacing insights older than 24 hours

---

## 12. Error Handling & Observability

### 12.1 Error Classification

| Error Code | HTTP Status | Description |
|---|---|---|
| `VALIDATION_ERROR` | 400 | Input failed schema validation; field errors returned |
| `UNAUTHORIZED` | 401 | Missing or invalid JWT; triggers silent refresh on frontend |
| `FORBIDDEN` | 403 | Valid JWT but insufficient permissions for the resource |
| `NOT_FOUND` | 404 | Requested resource does not exist for this user |
| `RATE_LIMITED` | 429 | Rate limit exceeded; `Retry-After` header included |
| `INTEGRATION_ERROR` | 502 | Third-party API call failed; degraded data served |
| `INTERNAL_ERROR` | 500 | Unexpected error; logged with request ID; generic message returned |

### 12.2 Logging

- Structured JSON logs using `zerolog`. Fields: `timestamp`, `level`, `requestId`, `userId`, `method`, `path`, `status`, `latencyMs`, `error`
- Logs are written to stdout in development and to a rotating file in production
- Error logs include a full stack trace. Info logs are sampled at 10% in production to reduce noise

### 12.3 Health Check Endpoint

- `GET /health` returns `{ status: "ok", db: "ok", version: "2.0.0", uptime: "12h3m" }`
- DB check executes a lightweight `PING` query. Failure returns `503`
- Used by Docker health check and load balancer probes

---

## 13. Testing Strategy

### 13.1 Backend Tests (Go)

- Unit tests for all service functions (thirdparty wrappers, analyzer, seeder) using `testify/mock`
- Handler integration tests using `httptest.NewRecorder` with an in-memory SQLite DB
- Auth middleware tested: valid token, expired token, malformed token, missing token, wrong algorithm
- `ProjectGuard` tested: user owns project (pass), user does not own project (403)
- **Target coverage:** 80% for handlers and services

### 13.2 Frontend Tests

- Component unit tests using Vitest + React Testing Library. Critical components: `AuthContext`, `ProjectSwitcher`, `SnapshotCard`
- Integration tests for multi-step Onboarding flow using MSW (Mock Service Worker) to intercept API calls
- E2E tests using Playwright for the three critical user journeys:
  - Register → Onboard → Dashboard
  - Connect Integration → View Data
  - Generate Content

---

## 14. DevOps & Deployment

### 14.1 Docker Setup

```yaml
# Root docker-compose.yml
services:
  backend:
    build: ./backend
    ports: ["8080:8080"]
    env_file: .env
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  frontend:
    build: ./frontend
    ports: ["3000:3000"]
    depends_on: [backend]

  nginx:
    image: nginx:alpine
    ports: ["80:80", "443:443"]
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./certs:/etc/ssl
```

### 14.2 Environment Variables

| Variable | Purpose |
|---|---|
| `JWT_PRIVATE_KEY` | RS256 private key PEM for signing access tokens |
| `JWT_PUBLIC_KEY` | RS256 public key PEM for verifying access tokens |
| `DATABASE_URL` | SQLite path (dev) or PostgreSQL DSN (prod) |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google OAuth 2.0 credentials |
| `META_APP_ID` / `META_APP_SECRET` | Facebook App credentials |
| `OPENAI_API_KEY` | OpenAI API key for content and insight generation |
| `ENCRYPTION_KEY` | 32-byte AES key for `OAuthCredential` encryption |
| `ALLOWED_ORIGINS` | Comma-separated list of allowed CORS origins |
| `FRONTEND_URL` | The full URL of the deployed frontend (for OAuth redirects) |

### 14.3 CI/CD Pipeline (GitHub Actions)

1. **On pull request to `main`:** Run Go tests + frontend linter (ESLint) + Playwright E2E suite
2. **On merge to `main`:** Build Docker images, tag with git SHA, push to container registry
3. **Deploy job:** SSH to production server, pull new images, run `docker-compose up -d`, verify health endpoint returns 200
4. **On health check failure:** Automatically roll back to the previous image tag

---

## 15. Setup Guide (Developer Quickstart)

### Prerequisites

- Go 1.21+ (`go version` to verify)
- Node.js 18+ with npm
- Docker + Docker Compose (for full stack)
- SQLite3 (for local dev without Docker)

---

### Option A: One-Command Docker Setup

```bash
# 1. Clone the repository
git clone https://github.com/your-org/dmtool && cd dmtool

# 2. Copy environment template and fill in your API keys
cp .env.example .env

# 3. Start everything
docker-compose up --build
```

Visit `http://localhost:3000`. Register an account. The app will auto-seed data so the dashboard is populated immediately.

---

### Option B: Manual Local Setup

#### Backend

```bash
cd backend

# Generate RS256 key pair
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# Configure environment
cp .env.example .env
# Set JWT_PRIVATE_KEY and JWT_PUBLIC_KEY in .env

# Install dependencies and start
go mod tidy
go run cmd/api/main.go
# Server starts on :8080, auto-migrates DB on first run
```

#### Frontend

```bash
cd frontend

# Configure environment
cp .env.local.example .env.local
# Set NEXT_PUBLIC_API_URL=http://localhost:8080

# Install and start
npm install
npm run dev
# Portal starts on http://localhost:3000
```

---

### Optional: Enable Real API Data

**Google Search Console:**
1. Create a project at [console.cloud.google.com](https://console.cloud.google.com)
2. Enable the Search Console API
3. Create OAuth 2.0 credentials
4. Add `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` to `.env`
5. Set redirect URI to `http://localhost:8080/api/integrations/google/callback`

**Meta (Facebook / Instagram):**
1. Create a Facebook App at [developers.facebook.com](https://developers.facebook.com)
2. Add `META_APP_ID` and `META_APP_SECRET` to `.env`
3. Set redirect URI to `http://localhost:8080/api/integrations/meta/callback`

**OpenAI:**
1. Add `OPENAI_API_KEY` to `.env`
2. Without it, the Content AI page returns a `503` and the Insight Generator worker is skipped

> **Note:** Without any API keys, DMTool works fully with auto-seeded data. All dashboard pages are functional. API keys are only needed to replace seeded data with your real marketing analytics.

---

*DMTool v2.0 — Architecture Blueprint | Built for Production*