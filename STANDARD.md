# DMTool — Project Standard File
> **READ THIS BEFORE TOUCHING ANY CODE.**
> Last Updated: June 2026 | Status: Active Development

---

## 1. What Is This Product

DMTool is a SaaS dashboard for digital marketers and small business owners.
It centralizes SEO auditing, social media analytics, and AI content generation in one place.

**Target User:** Freelance marketers, agencies, small business owners managing their own marketing.

**Core Value:** One login → see your Google Search Console traffic + Instagram/Facebook analytics + run SEO audits + schedule content — no jumping between tools.

---

## 2. Tech Stack (Exact Versions)

### Backend
| Tech | Version | Purpose |
|---|---|---|
| Go | 1.26 | Backend language |
| Gin | v1.12 | HTTP framework |
| GORM | v1.31 | ORM |
| SQLite | via glebarez/sqlite | Local dev database |
| PostgreSQL | via gorm postgres driver | Production database |
| JWT | golang-jwt/jwt v5, RS256 | Authentication |
| OpenAI | sashabaranov/go-openai | AI content + insights |
| Zerolog | rs/zerolog | Structured logging |
| Viper | spf13/viper | Config/env loading |
| OAuth2 | golang.org/x/oauth2 | Google + Meta OAuth |

### Frontend
| Tech | Version | Purpose |
|---|---|---|
| Next.js | 16.2.2 | React framework (App Router) |
| React | 19.2.4 | UI library |
| TypeScript | ^5 | Type safety |
| TailwindCSS | ^4 | Styling |
| Framer Motion | ^12.38 | Animations |
| Recharts | ^3.8 | Charts |
| Axios | ^1.15 | HTTP client |
| Lucide React | ^0.454 | Icons |
| Radix UI | ^2.1 | Dropdown primitives |

### Infrastructure
| Tech | Purpose |
|---|---|
| Docker + Docker Compose | Containerization |
| Nginx | Reverse proxy |
| SQLite (dev) / PostgreSQL (prod) | Database |


---

## 3. Project Structure

```
dmtool/
├── backend/                    ← Go API (main backend)
│   ├── cmd/api/main.go         ← Entry point, router setup, worker startup
│   └── internal/
│       ├── config/config.go    ← All env vars loaded via Viper
│       ├── db/database.go      ← DB init, AutoMigrate
│       ├── handlers/           ← HTTP handlers (one file per domain)
│       ├── middleware/         ← auth, cors, ratelimit, security, logger, recover
│       ├── models/             ← GORM models (one file per model)
│       ├── repository/         ← DB query layer (one file per model)
│       ├── services/           ← Business logic + third-party API calls
│       ├── utils/              ← jwt, crypto, response, pagination, seeder
│       └── workers/            ← Background goroutines
│
├── frontend/                   ← Next.js app
│   ├── app/
│   │   ├── (auth)/             ← login, register, onboarding
│   │   ├── (dashboard)/        ← all protected dashboard pages
│   │   └── (public)/           ← landing, pricing, docs, privacy
│   ├── components/
│   │   ├── ui/                 ← Design system (Button, Card, Badge, Input, etc.)
│   │   ├── dashboard/          ← Shared dashboard components
│   │   └── layout/             ← Public nav/footer
│   └── lib/
│       ├── api-client.ts       ← Axios instance + all API call functions
│       └── utils.ts            ← Shared utilities
│
├── crawler/                    ← Standalone Go crawler (NOT connected to backend yet)
├── ai-engine/                  ← Python FastAPI stub (NOT used, backend uses OpenAI directly)
├── nestjs-backend/             ← NestJS boilerplate (UNUSED, ignore completely)
├── docker-compose.yml          ← Runs backend + frontend + nginx
├── nginx.conf                  ← Reverse proxy config
├── STANDARD.md                 ← THIS FILE
├── CHAT_CONTEXT.md             ← Calendar/Instagram publishing dev notes
└── .env.example                ← Root env template for Docker
```

### Deprecated / Empty Files (DO NOT EDIT)
| File | Status |
|---|---|
| `backend/internal/services/analyzer.go` | Empty — 2 lines |
| `backend/internal/services/thirdparty.go` | Empty — 2 lines |
| `backend/internal/db/seeder.go` | Empty — deprecated |
| `backend/internal/models/models.go` | Empty — models split into individual files |
| `nestjs-backend/` | Entire folder unused |
| `ai-engine/` | Just a requirements.txt, not integrated |
| `crawler/` | Exists but not wired to backend |


---

## 4. Environment Variables

### Backend `.env` (at `backend/.env`)
| Variable | Required | Default | Purpose |
|---|---|---|---|
| `PORT` | No | 8080 | API server port |
| `APP_ENV` | No | development | `development` or `production` |
| `DATABASE_URL` | No | file:dmtool.db | SQLite path or Postgres URL |
| `JWT_PRIVATE_KEY` | Prod only | auto (dev key) | RS256 private key PEM |
| `JWT_PUBLIC_KEY` | Prod only | auto (dev key) | RS256 public key PEM |
| `ENCRYPTION_KEY` | Yes | — | 32-char key for OAuth token encryption |
| `ALLOWED_ORIGINS` | No | http://localhost:3000 | CORS allowed origins |
| `FRONTEND_URL` | No | http://localhost:3000 | Used in OAuth redirect URLs |
| `PUBLIC_BASE_URL` | No | http://localhost:8080 | Used for asset URLs |
| `UPLOAD_DIR` | No | ./uploads | File upload storage path |
| `OPENAI_API_KEY` | No | — | Without this, AI uses rule-based fallback |
| `META_APP_ID` | No | — | Facebook App ID for OAuth |
| `META_APP_SECRET` | No | — | Facebook App Secret |
| `META_PAGE_ACCESS_TOKEN` | No | — | Direct page token (bypass OAuth) |
| `GOOGLE_CLIENT_ID` | No | — | Google OAuth Client ID |
| `GOOGLE_CLIENT_SECRET` | No | — | Google OAuth Client Secret |
| `LINKEDIN_CLIENT_ID` | No | — | LinkedIn OAuth Client ID |
| `LINKEDIN_CLIENT_SECRET` | No | — | LinkedIn OAuth Client Secret |
| `DATAFORSEO_LOGIN` | No | — | DataForSEO API login (currently unused in code) |
| `DATAFORSEO_PASSWORD` | No | — | DataForSEO API password (currently unused in code) |
| `RAPIDAPI_KEY` | No | — | RapidAPI key (currently unused in code) |

### Frontend `.env.local` (at `frontend/.env.local`)
| Variable | Required | Default | Purpose |
|---|---|---|---|
| `NEXT_PUBLIC_API_URL` | No | http://localhost:8080/api | Backend API base URL |

---

## 5. How to Run Locally

### Prerequisites
- Go 1.21+ installed and in PATH
- Node.js 18+ and npm installed
- Open two PowerShell windows

### Terminal 1 — Backend
```powershell
cd "d:\Work_Projects\DM Tool\dmtool\backend"
copy .env.example .env     # first time only
go run cmd/api/main.go
```
Backend runs on `http://localhost:8080`
Verify: open `http://localhost:8080/health` → should return `{"status":"ok"}`

### Terminal 2 — Frontend
```powershell
cd "d:\Work_Projects\DM Tool\dmtool\frontend"
npm install                 # first time only
npm run dev
```
Frontend runs on `http://localhost:3000`

### First Use
1. Go to `http://localhost:3000/register` — create an account
2. Complete onboarding — enter project name and URL
3. Backend auto-seeds 30 days of fake data so dashboard is not empty
4. SEO Audit works immediately on any real URL
5. AI Content requires `OPENAI_API_KEY` in `.env`


---

## 6. Coding Standards

### Backend (Go)

**Pattern: Handler → Service → Repository**
- Handlers only handle HTTP (parse request, call service, return response)
- Services contain all business logic
- Repositories contain all DB queries — no raw SQL anywhere, GORM only

**Response Envelope — always use utils/response.go**
```go
// Success
utils.Success(c, data, meta)

// Error
utils.BadRequest(c, "message", "ERROR_CODE")
utils.NotFound(c, "message")
utils.InternalError(c, "message")
utils.ValidationError(c, err)
```

**Error codes used:** `VALIDATION_ERROR`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `RATE_LIMITED`, `INTEGRATION_ERROR`, `INTERNAL_ERROR`

**Every DB query must be scoped by UserID + ProjectID — no exceptions.**

**No new Go packages without discussion.**

### Frontend (TypeScript/Next.js)

**All API calls go through `lib/api-client.ts`** — never use raw fetch or create new axios instances.

**Component file naming:** PascalCase for components, kebab-case for pages (Next.js convention).

**No `any` types where avoidable** — define interfaces for API responses.

**Tailwind only** — no inline styles, no CSS modules, no styled-components.

**Toast notifications** — always use `toast()` from `@/components/ui/toaster` for user feedback.

**Loading states** — every async operation must show a `Loader2` spinner while pending.

**No new npm packages without discussion.**

---

## 7. What Is DONE (Verified Working)

### Backend
| Feature | File | Notes |
|---|---|---|
| Register / Login / Logout | `handlers/auth_handlers.go` | JWT RS256, refresh tokens, bcrypt |
| JWT Auth middleware | `middleware/auth.go` | RS256 validation, claims extraction |
| CORS, Rate Limiting, Security Headers | `middleware/` | All active |
| Project CRUD | `handlers/project_handlers.go` | Create, list, update, delete |
| Onboarding + Auto-seeder | `handlers/project_handlers.go` + `utils/seeder.go` | Seeds 30 days fake data |
| Dashboard snapshot | `handlers/dashboard_handlers.go` | Real math from DB data |
| SEO Crawler (13 checks) | `services/seo_crawler.go` | Crawls real URLs, scores 0-100 |
| SEO Handlers | `handlers/seo_handlers.go` | Audit, issues, keywords, report |
| Keyword Service | `services/keyword_service.go` | Google Autocomplete + GSC |
| OpenAI Content Generation | `services/openai.go` | GPT-4o-mini, 3 variants |
| OpenAI Insight Generation | `services/openai.go` | GPT-4o-mini, structured insights |
| Rule-based Insight Generator | `workers/insight_generator.go` | Works without OpenAI key |
| MetricsSyncer worker | `workers/metrics_syncer.go` | Syncs GSC + Meta every 6h |
| InsightGenerator worker | `workers/insight_generator.go` | Runs every 12h |
| CalendarPublisher worker | `workers/calendar_publisher.go` | Instagram + Facebook + LinkedIn |
| Meta Instagram publish | `services/meta.go` | Real Graph API, post/story/reel |
| Meta Facebook publish | `services/meta.go` | Real Graph API, photo/video |
| Meta Instagram metrics | `services/meta.go` | Followers, reach, engagement, audience |
| Meta Facebook metrics | `services/meta.go` | Fan count, reach, engagement |
| LinkedIn publish | `services/linkedin.go` | Real LinkedIn API, text + link posts |
| Google Search Console | `services/google.go` | Real API, clicks/impressions sync |
| OAuth Integration handlers | `handlers/integration_handlers.go` | Google + Meta + LinkedIn OAuth |
| Social handlers | `handlers/social_handlers.go` | Insights, history, profile, related |
| Content handlers | `handlers/content_handlers.go` | AI generation endpoint |
| Task CRUD | `handlers/task_handlers.go` + `handlers/dashboard_handlers.go` | Toggle, create, list |
| System handlers | `handlers/system_handlers.go` | Automations, calendar CRUD |
| All Repository layer | `repository/` | Typed GORM queries for all models |
| All Models | `models/` | User, Project, Metric, SocialMetric, Insight, Task, SEOIssue, KeywordResult, OAuthCredential, RefreshToken |

### Frontend
| Page/Feature | File | Notes |
|---|---|---|
| Login | `app/(auth)/login/page.tsx` | Working |
| Register | `app/(auth)/register/page.tsx` | Working |
| Onboarding | `app/(auth)/onboarding/page.tsx` | 4-step flow, OAuth popup, working |
| Main Dashboard | `app/(dashboard)/dashboard/page.tsx` | Charts, tasks, insights, snapshot |
| Dashboard layout + sidebar | `components/dashboard/` | Full navigation |
| Projects page | `app/(dashboard)/projects/page.tsx` | List + delete |
| Integrations page | `app/(dashboard)/integrations/page.tsx` | Connect/disconnect Google, Meta, LinkedIn |
| SEO Site Explorer | `app/(dashboard)/seo/site-explorer/` | Real audit runner |
| SEO Keywords | `app/(dashboard)/seo/keywords/` | Keyword research table |
| SEO Backlinks | `app/(dashboard)/seo/backlinks/` | UI built, API connected |
| Social Insights | `app/(dashboard)/social/insights/` | Real API calls |
| Social Growth | `app/(dashboard)/social/growth/` | Charts, real API calls |
| Social Profile Analyzer | `app/(dashboard)/social/profile-analyzer/` | Most complete (787 lines) |
| AI Insights | `app/(dashboard)/ai-insights/page.tsx` | Real API calls |
| Analytics Traffic | `app/(dashboard)/analytics/traffic/` | Real API calls |
| Content Calendar | `app/(dashboard)/system/calendar/page.tsx` | Schedule + publish posts |
| API Client | `lib/api-client.ts` | All endpoints typed and mapped |


---

## 8. What Is REMAINING — By Phase

---

### PHASE 1 — Make the Core Product Actually Work
> Goal: A user can register, connect accounts, and see REAL data. No fake numbers showing as if they are real.

| # | Task | File(s) | Priority |
|---|---|---|---|
| 1.1 | ✅ Fix `HealthScorer` worker — wired SEO crawler into 24h loop, updates `project.HealthScore` and persists issues | `workers/health_scorer.go` | DONE |
| 1.2 | ✅ Added `SimulatedBanner` component — shown on social insights page when `is_simulated=true` | `components/dashboard/simulated-banner.tsx` | DONE |
| 1.3 | Meta OAuth full flow — code exists, needs live test with real Meta App credentials | `handlers/integration_handlers.go` + `workers/metrics_syncer.go` | NEEDS LIVE TEST |
| 1.4 | ✅ `SimulatedBanner` wired into social insights page — extends to other pages as needed | Social pages | DONE |
| 1.5 | ✅ Already implemented — `saveUploadedAsset()` in system handler, `r.Static("/uploads")` in main.go | `handlers/system_handlers.go` | DONE |
| 1.6 | ✅ Already implemented — LinkedIn routes registered in `main.go`, handler in `integration_handlers.go` | `main.go` | DONE |

---

### PHASE 2 — Calendar Publishing End to End
> Goal: User can schedule a post, upload an image, and it publishes to Instagram/Facebook/LinkedIn at the right time.

| # | Task | File(s) | Priority |
|---|---|---|---|
| 2.1 | ✅ Already done in Phase 1 — `saveUploadedAsset()` in system handler | `handlers/system_handlers.go` | DONE |
| 2.2 | ✅ Already done in Phase 1 — `r.Static("/uploads")` in main.go | `main.go` | DONE |
| 2.3 | Test Instagram publish end to end with a real connected account | `workers/calendar_publisher.go` + `services/meta.go` | NEEDS LIVE TEST |
| 2.4 | Test Facebook publish end to end | `services/meta.go` | NEEDS LIVE TEST |
| 2.5 | Test LinkedIn publish end to end | `services/linkedin.go` | NEEDS LIVE TEST |
| 2.6 | ✅ Publish status display — shows scheduled/published/failed badges + error message on failed posts | `frontend/app/(dashboard)/system/calendar/page.tsx` | DONE |
| 2.7 | ✅ Retry logic — Retry button on failed posts reschedules 2 min ahead, backend resets publish_status to scheduled on due_date update | `calendar/page.tsx` + `system_handlers.go` | DONE |

---

### PHASE 3 — Real Analytics Data
> Goal: Dashboard shows real GSC + Meta data, not seeded numbers.

| # | Task | File(s) | Priority |
|---|---|---|---|
| 3.1 | ✅ GSC OAuth flow verified correct — `FetchMetrics` calls real Search Console API with user token; `MetricsSyncer` worker runs every 6h and upserts per-day rows | `services/google.go` + `workers/metrics_syncer.go` | DONE |
| 3.2 | ✅ `SyncProject` now prefers live GSC data over DataForSEO when Google is connected. Falls back to DataForSEO when not. Traffic chart reads from DB so it gets real rows after first sync. | `handlers/sync_handlers.go` | DONE |
| 3.3 | ✅ `SocialInsights` handler already tries real Meta API first, falls back to scraper/simulation. `SyncProject` now also prefers Meta API for social. `is_simulated` flag surfaced in response and shown as banner on dashboard. | `handlers/social_handlers.go` + `handlers/sync_handlers.go` | DONE |
| 3.4 | ✅ Implemented real DataForSEO v3 HTTP call (`task_post` + `task_get` poll). Falls back to deterministic simulation when credentials absent. | `services/dataforseo.go` | DONE |
| 3.5 | ✅ Implemented real RapidAPI call (`instagram-data1.p.rapidapi.com/user/info`). Falls back to simulation when API key absent. | `services/rapidapi.go` | DONE |

**Also fixed (Phase 3 build cleanup):**
- Deleted stray `package main` debug files at `backend/` root: `test_fb_debug.go`, `test_linkedin_api.go`, `test_linkedin_url.go`, `test_worker.go`, `scratch.go`, `scratch_list.go` — these blocked `go build ./...`
- Fixed `cmd/test2/main.go` — outdated `StartCalendarPublisher` call missing `linkedinService` arg

---

### PHASE 4 — Cleanup and Decisions
> Goal: Remove dead code, decide what to cut.

| # | Task | Notes |
|---|---|---|
| 4.1 | Delete or implement `services/analyzer.go` and `services/thirdparty.go` | Currently empty, misleading |
| 4.2 | AI Chat (`app/(dashboard)/ai/chat/`) — decide: build backend or remove the page | No backend exists |
| 4.3 | AI Visual Generator (`app/(dashboard)/ai/visual/`) — decide: integrate DALL-E or remove | No backend exists |
| 4.4 | Competitor tracking — currently returns empty array always | `handlers/dashboard_handlers.go` Competitors() |
| 4.5 | Alerts system — currently returns empty array always | `handlers/dashboard_handlers.go` Alerts() |
| 4.6 | Twitter publishing — Twitter API costs $100/month minimum — decide if worth it | N/A |
| 4.7 | Remove or explain `nestjs-backend/`, `ai-engine/`, `crawler/` folders | All unused |


---

## 9. Known Broken / Stub Files

| File | Problem | What It Should Do |
|---|---|---|
| `backend/internal/services/dataforseo.go` | ✅ Fixed — now calls real DataForSEO v3 API; falls back to simulation when no credentials | Call DataForSEO API for traffic estimation |
| `backend/internal/services/rapidapi.go` | ✅ Fixed — now calls real RapidAPI endpoint; falls back to simulation when no key | Call RapidAPI for Instagram profile data |
| `backend/internal/services/analyzer.go` | Completely empty (2 lines) | Was meant to be a site analyzer — functionality exists in `seo_crawler.go` instead |
| `backend/internal/services/thirdparty.go` | Completely empty (2 lines) | Was meant to aggregate third-party calls — now split into individual service files |
| `frontend/app/(dashboard)/ai/chat/page.tsx` | UI shell, no backend | AI chat interface |
| `frontend/app/(dashboard)/ai/visual/page.tsx` | UI only, no API calls | AI image/visual generation |

---

## 10. Testing Standards

### Before Starting Any Session
```powershell
# In project root
git fetch origin
git log HEAD..origin/main --oneline   # check for new commits
git status                             # check local changes
```

### Verify Backend is Running
```
GET http://localhost:8080/health
Expected: { "status": "ok", "db": "ok" }
```

### Key Endpoints to Test Manually

| What | Method | URL |
|---|---|---|
| Register | POST | `http://localhost:8080/api/auth/register` |
| Login | POST | `http://localhost:8080/api/auth/login` |
| Dashboard snapshot | GET | `http://localhost:8080/api/dashboard/snapshot?project_id=1` |
| SEO Audit | POST | `http://localhost:8080/api/seo/audit/run` body: `{"project_id":1,"url":"https://example.com"}` |
| Public SEO audit (no auth) | GET | `http://localhost:8080/api/public/seo-audit?url=https://example.com` |
| Social insights | GET | `http://localhost:8080/api/social/insights?project_id=1` |
| Generate content | POST | `http://localhost:8080/api/content/generate` |
| List integrations | GET | `http://localhost:8080/api/integrations` |

### Headers Required for Protected Routes
```
Authorization: Bearer <token_from_login_response>
Content-Type: application/json
```

---

## 11. Git Workflow

- **Always check before starting:** `git fetch origin` then `git log HEAD..origin/main --oneline`
- **Never push directly to main**
- **Branch naming:** `feature/phase-1-health-scorer`, `fix/meta-oauth-callback`, `phase-2/calendar-upload`
- **Commit messages:** `feat:`, `fix:`, `refactor:`, `chore:` prefixes
- **Before every PR:** run `npm run build` in frontend to confirm no TypeScript errors
- **Before every PR:** run `go build ./...` in backend to confirm no compile errors

---

## 12. Rules — Before Starting Any Phase

**Read this checklist before writing a single line of code:**

- [ ] Read this `STANDARD.md` file fully
- [ ] Run `git fetch origin` + `git log HEAD..origin/main --oneline` — check for new commits
- [ ] Run `git status` — confirm clean working tree
- [ ] Start backend: `go run cmd/api/main.go` — confirm health endpoint returns ok
- [ ] Start frontend: `npm run dev` — confirm `localhost:3000` loads
- [ ] Read the specific Phase section in this file for the work you are about to do
- [ ] Read the actual files you will be modifying before changing anything
- [ ] Do not install new packages without justification
- [ ] Do not create new files if existing files can be extended
- [ ] After changes: run `go build ./...` (backend) and `npm run build` (frontend)
- [ ] Test the specific feature manually using the endpoints in Section 10
- [ ] Update this `STANDARD.md` file to mark completed tasks as done

---

*This file is the single source of truth for the DMTool project.*
*When in doubt — read this file first.*
