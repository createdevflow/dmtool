# Honesty sprint — daily tracker

**Repo:** `dmtool`  
**File you edit daily:** `HONESTY_SPRINT.md` (this file, repo root)  
**Scope:** make existing tools honest. No CRM, campaigns, automations engine, DataForSEO, Stripe, or admin expansion.

**Locked decisions**
- New projects stay empty until GSC/Meta is connected. No seed traffic.
- Rank Tracking stays in the sidebar, labeled GSC-only.

How to use: after each day, tick `[x]`, set Status, move leftovers into **Parking**, write Notes. This is the source of truth for what is still pending.

Status values: `pending` | `in progress` | `done` | `skipped`

---

## Honesty rules (do not break)

1. Do not write simulated marketing metrics into the DB.
2. Do not invent chart series in the frontend.
3. Pages that cannot tell the truth leave nav / Cmd+K / SectionTabs. Bookmarked URLs get an honest empty, not fake tables.
4. Autocomplete keyword *strings* stay. Invented volume/KD show as **—**.
5. Empty dashboard after signup is correct.

---

## Day 1–2 — Hide fake surfaces

Status: done

- [x] Remove from `frontend/components/dashboard/sidebar.tsx` `navGroups`:
  - [x] Backlink Analysis `/seo/backlinks`
  - [x] Visual AI `/ai/visual`
  - [x] Profile Discovery `/social/competitors`
- [x] Remove from `frontend/app/(dashboard)/dashboard/[mode]/overview-client.tsx` SectionTabs: Backlinks, Alerts, Profile Discovery
- [x] Replace page bodies with a one-line honest empty (keep the route, no fake tables):
  - [x] `frontend/app/(dashboard)/seo/backlinks/page.tsx`
  - [x] `frontend/app/(dashboard)/ai/visual/page.tsx`
  - [x] `frontend/app/(dashboard)/social/competitors/page.tsx`
  - [x] `frontend/app/(dashboard)/alerts/page.tsx` — “No alert engine is wired.”
- [x] Confirm Cmd+K no longer lists those tools (command menu reads `navGroups`)

**Keep in nav:** Rank Tracking, AI Chat, Site Explorer, Keywords, Profile Analyzer, Growth, Content Generator.

Notes:
- 2026-09-11: Cmd+K has no separate item list — `command-menu.tsx` maps `navGroups` from sidebar.tsx, so dropping those three items is enough for palette + desktop + mobile sidebar.
- Left the topbar bell (`frontend/components/dashboard/topbar.tsx` ~line 147) pointing at `/alerts`. Day 1–2 only asked to drop Alerts from SectionTabs, not the bell. After login, that URL is the honest one-liner, not a fake list.
- Did not touch `api-client.ts` helpers or backend `/seo/backlinks` / `/alerts` routes (Day 3).
- Did not push.

---

## Day 3 — Stop writing fake data (backend)

Status: done

- [x] Onboard: drop `utils.SeedProject` in `backend/internal/handlers/project_handlers.go`
- [x] Keep `autoSyncNewProject` but only the real URL crawl
- [x] Do not call DataForSEO `FetchEstimatedTraffic` (always simulates)
- [x] Do not call RapidAPI `FetchInstagramProfile` (always simulates)
- [x] `dataforseo.go` / `rapidapi.go`: return error instead of `simulate*`
- [x] `SEOHandler.Backlinks`: zeros + `available: false` (no hash math, do not key off Google cred)
- [x] `RelatedProfiles`: delete hardcoded `capturedbyaaryan` / `aryanvishwakarma_01` lists; return `[]`
- [x] Social insights/refresh: after Meta / LinkedIn / public IG fail, return empty — do not persist hash estimates
- [x] Autocomplete keywords: `Volume=0`, `KD=0`; GSC path may keep real impressions/position

**Check:** new project has no `metrics` with `source=seed` or `source=dataforseo`, no hashed social rows.

Notes:
- 2026-09-14: Also dropped `SeedProject` from `Create` (not only Onboard) and run the real crawl there too. Onboard no longer pre-fills `HealthScore: 75`.
- Public IG scrape still allowed only when the scrape is real (`IsSimulated == false`). Invented reach/engagement percentages are not stored.
- Social insights/history ignore existing `is_simulated` rows so leftover seed does not show as live.
- SQLite `backend/dev.db` may still contain old seed metrics for projects created before today. **Create a new project** (or wipe that DB) to confirm empty traffic. Did not delete existing rows.
- `utils/seeder.go` left in the tree, unused. Sync handler still exists but is not mounted (Day 6); DataForSEO/RapidAPI now error so it cannot write fakes if called.
- Did not push. Backend rebuilt and restarted on :8097.

---

## Day 4 — Charts tell the truth

Status: done

- [x] Snapshot traffic totals: GSC only (or empty)
- [x] `social/growth/page.tsx`: delete the 30-day invented curve when history length ≤ 1
- [x] `social/profile-analyzer/page.tsx`: remove `mockAudience` / content-split / posting-times mocks; hide cards with no API fields
- [x] Keywords UI: volume/KD as **—** when 0
- [x] Analytics traffic/custom: do not chart `source=seed` as traffic

**Check:** growth page with one snapshot does not draw a 30-day curve.

Notes:
- 2026-09-14: Snapshot, Metrics, and Traffic APIs now drop non-`gsc` rows. Simulated social rows are ignored in snapshot totals. `traffic_source` is `"gsc"` or empty.
- Growth page no longer invents 30 days of history. 0–1 snapshots → empty chart (“Not enough snapshots yet.”).
- Keyword easy/medium/hard filters ignore KD 0 so autocomplete rows are not labeled Easy.
- Frontend traffic/custom also filter `source === "gsc"` as a second guard.
- Did not push. Backend rebuilt for snapshot change.

---

## Day 5 — Fix AI Chat

Status: done

Today it sends `project_id: 0` → 404.

- [x] Load projects; use `dmtool_active_project_id`
- [x] No project → disable send
- [x] Label: “Drafts a caption. Not a strategy assistant.”
- [x] Show API `source` (`ai` vs `template`)
- [x] Visual AI stays hidden + honest empty (no Unsplash button)

**Check:** network tab never shows `project_id: 0`. With a project, chat returns a variant.

Notes:
- 2026-09-14: `handleSend` returns before any request if `!project?.id`. Send is disabled with no project. Fake New Chat / History buttons removed (they did nothing).
- Visual AI was already the Day 1–2 one-liner; left unchanged.
- Did not push. Frontend-only; no backend rebuild.

---

## Day 6 — Mount Sync; honest Rank Tracking

Status: done

- [x] Call `registerSyncRoutes` from `backend/cmd/api/main.go` (defined, never invoked)
- [x] Sync only: GSC if connected, Meta/LinkedIn if connected, SEO crawl
- [x] Nothing connected → 200 with `skipped`, not fake rows
- [x] Rank Tracking copy: “Positions from Google Search Console, not a daily rank tracker.”
- [x] Empty if no `position > 0`
- [x] Do not invent visibility/top-3 from position-0 autocomplete rows
- [x] Sync button on rank tracking no longer 404s

**Check:** `POST /api/projects/:id/sync` with JWT is not 404.

Notes:
- 2026-09-14: `registerSyncRoutes` is now invoked. SyncHandler no longer calls DataForSEO/RapidAPI. It pulls GSC traffic + GSC query positions when Google OAuth is connected, Meta/LinkedIn when those tokens exist, and the real SEO crawl when the project has a URL. Simulated social rows are not persisted. All-skipped still returns 200.
- Rank Tracking API is empty unless Google OAuth is connected, then only `position > 0`. Leftover seed/autocomplete rows in `dev.db` no longer fill visibility/top-3. Empty seed on `FindKeywords` means every keyword for the project so GSC rows are visible after sync.
- Rank Tracking page copy is GSC-only; volume 0 shows —. Sync toast reports skipped vs success. Did not push.

---

## Day 7 — Copy leftovers

Status: done

- [x] Landing page: remove fake `99.9% / <100ms / 98.4%` (`frontend/app/(public)/page.tsx`)
- [x] `/system/integrations` redirect to `/integrations`
- [x] HealthScorer: actually crawl project URLs, or stop logging “scoring complete” while doing nothing
- [x] Keyword UI: no KD color badges when KD is 0

Notes:
- 2026-09-14: Removed the invented landing stats strip (including 24/7). `/system/integrations` now redirects to `/integrations` and OAuth callback routes follow. HealthScorer crawls each project URL and writes the real score; no “scoring complete” no-op. Keywords and Rank Tracking show — instead of Easy/green when KD is 0. Google OAuth env vars loaded from the committed `.env.example` values so Connect Now can attempt Google; redirect URI still needs Google Cloud whitelist (evening). Did not push.

---

## Day 8 — Tests

Status: done

- [x] Backlinks returns `available: false` and zeros
- [x] RelatedProfiles empty for a normal project
- [x] Onboard inserts no `source=seed` metrics
- [x] RankTracking does not invent top-3 from position 0
- [x] Sync route registered (httptest POST ≠ 404)
- [x] Content generate `project_id=0` still 404; real project 200
- [x] `navGroups` does not contain `/seo/backlinks`, `/ai/visual`, `/social/competitors`
- [x] Chat helper never sends `project_id: 0`
- [x] `cd backend && go test ./...`
- [x] `cd frontend && npx vitest run`

Notes:
- 2026-09-15: Handler tests in `backend/internal/handlers/honesty_sprint_test.go`; sync route in `backend/cmd/api/sync_route_test.go`. Frontend: `sidebar.honesty.test.ts` and `chat/page.honesty.test.tsx`. Content generate treats `project_id=0` as 404 (was validation 400). Fixed stale NewAuthHandler / NewAdminHandler signatures so `go test ./...` compiles. `go test ./...` ok; `npx vitest run` 21 passed / 2 todo. Did not push.

---

## Day 9–10 — Verify like a user

Status: done

- [x] Register → empty dashboard / Connect CTAs, not a 30-day hockey stick
- [x] Site Explorer audit: real checks for the URL
- [x] Keywords: suggestion strings, volume **—**
- [x] Typed URL Backlinks / Visual AI / Alerts: honest empty
- [x] Those tools absent from sidebar + Cmd+K
- [x] Chat with a project works; no project blocked
- [x] Sync on rank tracking: 200 skipped, no 404
- [x] Growth: no fake 30-day curve
- [x] Desktop + mobile, modes Search / Social / Combined

Notes:
- 2026-09-15: Click-through in Chromium (Playwright) as a new user `day910-…@example.com`, project `Example` / `https://example.com`. Screenshots in `.mimocode/honesty-d910/`.
- Dashboard after register is a tool directory, not a seeded traffic chart. Rank Tracking toast “Sync complete.” Network: 200, traffic/rankings/social skipped, SEO crawl success (real URL).
- Site Explorer on example.com: 13 live checks (title “Example Domain”, no OG tags, robots.txt 404). Could not add a second SEO project on the free plan (quota), so a second-URL comparison was not completed in the picker.
- Keywords: autocomplete strings for “digital marketing”; volume/KD/position all **—**. Cmd+K has no Backlinks / Visual AI / Profile Discovery. Hidden routes show one-liners.
- Chat with project returned a labeled TEMPLATE caption. Second account with no project: “Create a project first”, Send disabled.
- Growth in Social mode: “Not enough snapshots yet.” / “No metrics data yet.” No 30-day invented series. Mobile 390px: dashboard, keywords, backlinks still honest.
- Combined is the default after Search-goal onboard. Mode switcher on Growth hides SEO Intelligence. Automated check on `/dashboard` still saw both families because Combined overview cards list them.
- Did not push. New issues listed in the Day 9–10 report (not fixed).

### Post-verification fixes (2026-09-15)

Closed the 3 honesty-rule breakers from Day 9–10. Parked: onboarding copy tone, overview card wording, keyword mix noise, free-plan SEO quota.

- Insight generator: only `source=gsc` metrics and non-simulated social. Citc no longer gets “Organic Traffic Up 1314%”; it now stores “Connect Google Search Console…” plus the real crawl health score.
- Site Explorer: `ReplaceOpenIssues` on each audit (also sync/auto-crawl). Two audits on example.com: 24 leftover duplicates → 6, then stayed 6.
- Onboarding goal now writes `users.dashboard_mode` and `dmtool_mode`. Search onboard → Search overview + SEO selected; Social onboard → Social overview + Social selected.

Did not push.

---

## Explicitly not doing (leave parked)

- [ ] DataForSEO / RapidAPI real HTTP
- [ ] Stripe
- [ ] Admin suspend-blocks-login / activity logging / commit admin UI
- [ ] Automations engine
- [ ] CRM / campaigns
- [ ] Full-site crawler, SERP tracker, backlink index
- [ ] Deleting leftover `nestjs-backend/`, `crawler/`, `ai-engine/`

---

## Parking / slipped from a day

_Move unfinished checkboxes here with the date they slipped._

---

## Done looks like

A new user sees an empty-but-true dashboard, can audit their URL, get keyword ideas without fake volume, draft captions, and sync only real sources. They cannot stumble into a page that looks like Semrush backlinks or an image generator.
