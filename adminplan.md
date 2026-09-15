# DMTool Admin Panel — Expansion Plan

> **Status**: Planning  
> **Last updated**: 2026-07-13

---

## Table of Contents

1. [Current State](#1-current-state)
2. [Vision & Goals](#2-vision--goals)
3. [Admin Sidebar Redesign](#3-admin-sidebar-redesign)
4. [Feature Roadmap](#4-feature-roadmap)
5. [New Backend Endpoints](#5-new-backend-endpoints)
6. [New Frontend Pages](#6-new-frontend-pages)
7. [Data Model Additions](#7-data-model-additions)
8. [User Detail Deep Dive](#8-user-detail-deep-dive)
9. [Project Monitoring](#9-project-monitoring)
10. [SEO Admin View](#10-seo-admin-view)
11. [Social Admin View](#11-social-admin-view)
12. [Combined / Platform View](#12-combined--platform-view)
13. [System Health & Observability](#13-system-health--observability)
14. [Implementation Phases](#14-implementation-phases)
15. [Deferred Items](#15-deferred-items)

---

## 1. Current State

### What exists today

| Layer | Status | Details |
|-------|--------|---------|
| **Backend API** | ✅ Complete | 11 admin endpoints under `/api/admin/*` with `RequireRole("admin")` |
| **Frontend Pages** | ✅ Basic | 5 pages: Overview, Users list, User detail, Plans, Audit Log |
| **Admin Guard** | ✅ Complete | Client-side guard + server-side middleware |
| **Sidebar Link** | ✅ Complete | "Admin Panel" link visible to admin users only |
| **Admin Sub-Nav** | ✅ Complete | Horizontal tab strip: Overview, Users, Plans, Audit Log |
| **Impersonation** | ✅ Complete | 30-min JWT swap with banner + stop button |
| **Audit Logging** | ✅ Complete | 7 action types logged |

### What's missing (gaps)

- No admin-specific sidebar (uses same sidebar as regular users)
- No user website/project details view
- No per-project SEO/Social data inspection
- No platform-wide analytics (API usage, feature adoption)
- No system health monitoring
- No user activity timeline
- No bulk actions (mass email, bulk role change)
- No user suspension/deletion (DEFER-004)
- No subscription history (DEFER-005)
- No automated admin bootstrap (DEFER-006)

---

## 2. Vision & Goals

### Design Principles

1. **Admin sidebar replaces the user sidebar** — when on `/admin/*`, the left sidebar shows admin-only navigation, not the user's dashboard nav
2. **Mode-aware admin views** — admin can inspect data per mode (SEO, Social, Combined) for any user
3. **Drill-down everywhere** — every list → detail → action. No dead-end pages
4. **Platform health at a glance** — the admin overview should answer "is the platform healthy?" in 3 seconds
5. **Audit everything** — every admin action is logged and traceable

### User Stories

| # | As an admin, I want to... | So that... |
|---|---|---|
| US-1 | See a dedicated admin sidebar | I don't confuse admin nav with user nav |
| US-2 | View any user's website/project details | I can debug issues without impersonating |
| US-3 | See SEO metrics for any user's projects | I can assess platform SEO health |
| US-4 | See social metrics for any user's projects | I can assess platform social health |
| US-5 | View system health (API latency, error rates) | I can proactively detect issues |
| US-6 | Filter users by plan, role, activity, mode | I can segment users for targeted outreach |
| US-7 | See a user's full activity timeline | I can understand their usage patterns |
| US-8 | Bulk-manage users (export, role changes) | I can handle scale efficiently |
| US-9 | View platform-wide analytics | I can make data-driven product decisions |

---

## 3. Admin Sidebar Redesign

### Current: User sidebar + Admin link

```
┌──────────────┐
│ DMTool       │
│ [mode switch]│
│──────────────│
│ Dashboard    │
│ SEO Intel    │
│ Social Media │
│ AI Engine    │
│ Account      │
│──────────────│
│ 🛡 Admin     │  ← only for admin users
└──────────────┘
```

### New: Dedicated admin sidebar on /admin/* routes

```
┌──────────────┐
│ DMTool Admin │  ← different branding
│──────────────│
│ Overview     │  ← platform stats
│──────────────│
│ USERS        │
│  All Users   │
│  Roles       │
│  Subscriptions│
│──────────────│
│ PROJECTS     │
│  All Projects│
│  SEO Health  │
│  Social Health│
│──────────────│
│ FINANCE      │
│  Plans       │
│  Revenue     │
│  Trials      │
│──────────────│
│ SYSTEM       │
│  Health      │
│  Audit Log   │
│  Settings    │
│──────────────│
│ ← Back to App│
└──────────────┘
```

### Implementation

**File**: `frontend/components/admin/admin-sidebar.tsx` (new)

- Replaces the user sidebar when `pathname.startsWith("/admin")`
- Rendered in `app/(dashboard)/layout.tsx` with a conditional:
  ```tsx
  {pathname.startsWith("/admin") ? <AdminSidebar /> : <Sidebar />}
  ```
- Groups organized by domain (Users, Projects, Finance, System)
- Active state uses `usePathname()` matching
- "Back to App" link returns to `/dashboard/combined`

**File**: `frontend/app/(dashboard)/layout.tsx` (modify)

- Add pathname-aware sidebar switching
- Keep the shared topbar (or create a simplified admin topbar)

---

## 4. Feature Roadmap

### Phase 1: Admin Sidebar + User Management Enhancement

| Feature | Priority | Backend | Frontend |
|---------|----------|---------|----------|
| Dedicated admin sidebar | P0 | — | New component |
| User list with advanced filters | P0 | Enhance `ListUsers` | New filters UI |
| User activity timeline | P1 | New endpoint | New section |
| User suspension/deactivation | P1 | New column + endpoint | Toggle UI |
| Bulk user export (CSV) | P2 | New endpoint | Export button |

### Phase 2: Project Monitoring

| Feature | Priority | Backend | Frontend |
|---------|----------|---------|----------|
| All projects list (admin view) | P0 | New endpoint | New page |
| Per-project detail view | P0 | New endpoint | New page |
| SEO metrics per project | P1 | Enhance existing | New section |
| Social metrics per project | P1 | Enhance existing | New section |
| Project health dashboard | P1 | Aggregation endpoint | New page |

### Phase 3: Platform Analytics

| Feature | Priority | Backend | Frontend |
|---------|----------|---------|----------|
| Platform-wide metrics | P0 | New endpoint | Overview enhancement |
| Feature adoption rates | P1 | New endpoint | New chart |
| API usage analytics | P1 | New middleware | New page |
| User engagement scores | P2 | New calculation | New metric |

### Phase 4: System Health

| Feature | Priority | Backend | Frontend |
|---------|----------|---------|----------|
| System health endpoint | P0 | New endpoint | New page |
| Error rate monitoring | P1 | Log aggregation | New chart |
| Integration status | P1 | New endpoint | Status page |
| Scheduled jobs status | P2 | New endpoint | New page |

---

## 5. New Backend Endpoints

### Admin Projects

```
GET    /api/admin/projects                  — List all projects (paginated, filterable)
GET    /api/admin/projects/:id              — Project detail + metrics summary
GET    /api/admin/projects/:id/metrics      — Project metrics time-series
GET    /api/admin/projects/:id/seo          — Project SEO data (issues, keywords, backlinks)
GET    /api/admin/projects/:id/social       — Project social data (insights, history)
GET    /api/admin/projects/stats            — Platform-wide project stats
```

### Admin Users (Enhanced)

```
GET    /api/admin/users/:id/activity        — User activity timeline
POST   /api/admin/users/:id/suspend         — Suspend user account
POST   /api/admin/users/:id/unsuspend       — Reactivate suspended account
GET    /api/admin/users/export              — Export users as CSV
POST   /api/admin/users/bulk-role           — Bulk role change
```

### Admin Platform

```
GET    /api/admin/platform/stats            — Platform-wide analytics
GET    /api/admin/platform/health           — System health check
GET    /api/admin/platform/engagement       — User engagement metrics
GET    /api/admin/platform/feature-usage    — Feature adoption rates
```

### Admin Integrations

```
GET    /api/admin/integrations/status       — All users' integration health
```

---

## 6. New Frontend Pages

### Admin Sidebar Routes

```
/admin                              — Platform Overview (existing, enhanced)
/admin/users                        — User List (existing, enhanced filters)
/admin/users/[id]                   — User Detail (existing, enhanced)
/admin/users/[id]/projects          — User's Projects (NEW)
/admin/users/[id]/activity          — User Activity Timeline (NEW)
/admin/projects                     — All Projects (NEW)
/admin/projects/[id]                — Project Detail (NEW)
/admin/projects/[id]/seo            — Project SEO Data (NEW)
/admin/projects/[id]/social         — Project Social Data (NEW)
/admin/plans                        — Plan Management (existing)
/admin/revenue                      — Revenue Dashboard (NEW)
/admin/audit-log                    — Audit Log (existing)
/admin/system                       — System Health (NEW)
/admin/integrations                 — Integration Status (NEW)
```

---

## 7. Data Model Additions

### User Model Changes

```go
// Add to User struct
DisabledAt     *time.Time `json:"disabled_at,omitempty"`  // Soft suspension
DisabledReason string     `json:"disabled_reason,omitempty"`
LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
LoginCount     int        `json:"login_count" gorm:"default:0"`
```

### New: UserActivity Model

```go
type UserActivity struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UserID    uint      `gorm:"index" json:"user_id"`
    Action    string    `json:"action"`     // login, project_created, audit_run, etc.
    Metadata  JSON      `json:"metadata"`   // context-specific data
    IPAddress string    `json:"ip_address"`
    UserAgent string    `json:"user_agent"`
}
```

### New: PlatformMetric Model

```go
type PlatformMetric struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Date      time.Time `gorm:"uniqueIndex" json:"date"`
    TotalAPICalls      int `json:"total_api_calls"`
    TotalPageViews     int `json:"total_page_views"`
    ActiveUsers        int `json:"active_users"`
    NewSignups         int `json:"new_signups"`
    ProjectsCreated    int `json:"projects_created"`
    AuditsRun          int `json:"audits_run"`
    ContentGenerated   int `json:"content_generated"`
}
```

### New: SystemHealth Model

```go
type SystemHealth struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CheckedAt time.Time `json:"checked_at"`
    Service   string    `json:"service"`    // "api", "ai_engine", "database"
    Status    string    `json:"status"`     // "healthy", "degraded", "down"
    LatencyMs int       `json:"latency_ms"`
    Metadata  JSON      `json:"metadata"`
}
```

---

## 8. User Detail Deep Dive

### Current User Detail Page

```
┌─────────────────────────────────────────────┐
│ User Profile          [Edit] [Reset PW]     │
│ Name: John Doe         Role: owner ▼        │
│ Email: john@example.com                     │
│ Joined: Jan 15, 2026                        │
├─────────────────────────────────────────────┤
│ Subscription                               │
│ Plan: pro_monthly   Status: active          │
│ Started: Feb 1, 2026                        │
├─────────────────────────────────────────────┤
│ Audit Log (last 20)                         │
│ ...                                         │
└─────────────────────────────────────────────┘
```

### Enhanced User Detail Page

```
┌─────────────────────────────────────────────┐
│ ← Back to Users                             │
├─────────────────────────────────────────────┤
│ [Profile] [Projects] [Activity] [Subscription] │  ← tabbed
├─────────────────────────────────────────────┤
│ Profile Tab                                 │
│ ┌──────────────┐ ┌────────────────────────┐ │
│ │ Avatar       │ │ Name: John Doe    [Edit]│ │
│ │ JD           │ │ Email: john@...  [Edit] │ │
│ │              │ │ Role: owner ▼           │ │
│ │              │ │ Mode: combined          │ │
│ │              │ │ Status: Active ✅       │ │
│ │              │ │ Last Login: 2h ago      │ │
│ │              │ │ Login Count: 47         │ │
│ └──────────────┘ └────────────────────────┘ │
├─────────────────────────────────────────────┤
│ Quick Actions                               │
│ [Reset Password] [Impersonate] [Suspend]    │
│ [Change Plan]    [Change Role]              │
├─────────────────────────────────────────────┤
│ Subscription                                │
│ Plan: Pro Monthly ($29/mo)                  │
│ Status: Active                              │
│ Started: Feb 1, 2026                        │
│ [Change Plan ▼] [Cancel Sub]               │
├─────────────────────────────────────────────┤
│ Audit Log (this user)                       │
│ Action        Actor    When                 │
│ impersonate   admin    2 hours ago          │
│ plan.change   admin    3 days ago           │
│ login         system   3 days ago           │
│ ...                                         │
└─────────────────────────────────────────────┘
```

### Projects Tab (per user)

```
┌─────────────────────────────────────────────┐
│ John Doe's Projects (3)                     │
├─────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────┐ │
│ │ 🌐 example.com                         │ │
│ │ Goal: SEO + Social  Health: 87/100 ✅   │ │
│ │ Keywords: 45  Backlinks: 120            │ │
│ │ Posts: 23     Followers: 1,200          │ │
│ │ [View Details] [Impersonate → Project]  │ │
│ └─────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────┐ │
│ │ 📱 @johndoe (Instagram)                │ │
│ │ Goal: Social     Health: 72/100 ⚠️      │ │
│ │ Followers: 5,400  Posts: 89             │ │
│ │ Engagement: 3.2%  Reach: 12K           │ │
│ │ [View Details] [Impersonate → Project]  │ │
│ └─────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

### Activity Tab (per user)

```
┌─────────────────────────────────────────────┐
│ Activity Timeline — Last 30 Days            │
├─────────────────────────────────────────────┤
│ ● Logged in                  2 hours ago    │
│ ● Ran SEO audit on example.com  3h ago     │
│   → Found 12 issues, resolved 8            │
│ ● Generated content (3 posts)  5h ago       │
│ ● Updated project settings    1 day ago     │
│ ● Logged in                  1 day ago      │
│ ● Created project mysite.com  3 days ago    │
│ ● Started free trial          5 days ago     │
│ ● Registered account          5 days ago     │
└─────────────────────────────────────────────┘
```

---

## 9. Project Monitoring

### Admin Projects List Page

```
┌─────────────────────────────────────────────┐
│ All Projects (127)                          │
├─────────────────────────────────────────────┤
│ Filters: [Owner ▼] [Goal ▼] [Health ▼]     │
│ Search: [_______________]                   │
├─────────────────────────────────────────────┤
│ Name        Owner     Goal   Health  Status │
│ example.com john@..   SEO    87/100  active │
│ mysite.com  jane@..   Both   45/100  issues │
│ @johndoe    john@..   Social 72/100  active │
│ ...                                         │
├─────────────────────────────────────────────┤
│ Stats Bar                                   │
│ Total: 127 | Healthy: 89 | Issues: 31 |     │
│ Scanning: 7                                 │
└─────────────────────────────────────────────┘
```

### Admin Project Detail Page

```
┌─────────────────────────────────────────────┐
│ ← Back to Projects                          │
│ example.com — Owner: John Doe               │
├─────────────────────────────────────────────┤
│ [Overview] [SEO] [Social] [Settings]        │
├─────────────────────────────────────────────┤
│ Health Score: 87/100  Status: Active         │
│ Goal: SEO + Social                          │
│ Created: Jan 15, 2026                       │
├─────────────────────────────────────────────┤
│ SEO Summary                                 │
│ Keywords tracked: 45 (12 in top 10)         │
│ Backlinks: 120 (DA avg: 34)                 │
│ Issues: 3 critical, 5 warnings              │
│ Last audit: 2 hours ago                     │
├─────────────────────────────────────────────┤
│ Social Summary                              │
│ Platform: Instagram (@example)              │
│ Followers: 5,400  Posts: 89                 │
│ Avg engagement: 3.2%                        │
│ Last refresh: 1 hour ago                    │
├─────────────────────────────────────────────┤
│ Recent Insights                             │
│ 🔴 Critical: Missing meta description       │
│ 🟡 Warning: Slow page load (3.2s)           │
│ 🟢 Opportunity: Add structured data         │
└─────────────────────────────────────────────┘
```

---

## 10. SEO Admin View

### Platform SEO Health (from Projects list filtered by SEO goal)

```
┌─────────────────────────────────────────────┐
│ Platform SEO Health                         │
├─────────────────────────────────────────────┤
│ Total SEO projects: 89                      │
│ Avg Health Score: 72/100                    │
│ Total Keywords tracked: 2,340               │
│ Keywords in Top 10: 456 (19.5%)            │
│ Total Backlinks: 12,400                     │
│ Critical Issues (platform-wide): 45         │
├─────────────────────────────────────────────┤
│ Top Issues Across All Projects              │
│ 1. Missing meta descriptions (23 projects)  │
│ 2. Slow page load (15 projects)             │
│ 3. Missing alt text (12 projects)           │
├─────────────────────────────────────────────┤
│ Projects by Health Score                    │
│ ████████████████░░░░ 90-100: 12             │
│ ██████████████░░░░░░ 70-89:  34             │
│ ████████░░░░░░░░░░░░ 50-69:  28             │
│ ████░░░░░░░░░░░░░░░░ 0-49:   15             │
└─────────────────────────────────────────────┘
```

---

## 11. Social Admin View

### Platform Social Health (from Projects list filtered by Social goal)

```
┌─────────────────────────────────────────────┐
│ Platform Social Health                      │
├─────────────────────────────────────────────┤
│ Total Social projects: 67                   │
│ Platforms connected:                        │
│   Instagram: 45  Facebook: 32  Twitter: 28  │
│   LinkedIn: 15                             │
│ Total followers (all users): 156,000        │
│ Avg engagement rate: 2.8%                   │
│ Content generated this month: 234 posts     │
├─────────────────────────────────────────────┤
│ Top Performing Projects                     │
│ 1. @fashionbrand — 45K followers, 4.2% eng │
│ 2. @techstartup — 23K followers, 3.8% eng  │
│ 3. @localbiz — 12K followers, 5.1% eng     │
├─────────────────────────────────────────────┤
│ Engagement Trend (30d)                      │
│ [chart showing platform-wide engagement]    │
└─────────────────────────────────────────────┘
```

---

## 12. Combined / Platform View

### Platform Overview (Enhanced Admin Home)

```
┌─────────────────────────────────────────────┐
│ 🛡 Platform Overview                        │
├─────────────────────────────────────────────┤
│ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────┐│
│ │ Users   │ │Projects │ │  MRR    │ │API ││
│ │  127    │ │   89    │ │ $2,340  │ │12K ││
│ │ +12/wk  │ │ +8/wk   │ │ +$200   │ │/day││
│ └─────────┘ └─────────┘ └─────────┘ └────┘│
├─────────────────────────────────────────────┤
│ Health Score Distribution (all projects)    │
│ [horizontal bar chart]                      │
├─────────────────────────────────────────────┤
│ Revenue (MRR/ARR)                           │
│ [line chart, last 12 months]                │
├─────────────────────────────────────────────┤
│ User Growth                                 │
│ [bar chart, last 30 days]                   │
├─────────────────────────────────────────────┤
│ Feature Adoption                            │
│ SEO tools: 67% of projects                  │
│ Social tools: 52% of projects               │
│ AI content: 34% of users                    │
│ Integrations: 28% of users                  │
├─────────────────────────────────────────────┤
│ Recent Admin Actions                        │
│ [last 20 audit entries]                     │
└─────────────────────────────────────────────┘
```

---

## 13. System Health & Observability

### System Health Page

```
┌─────────────────────────────────────────────┐
│ System Health                               │
├─────────────────────────────────────────────┤
│ Service          Status    Latency  Uptime  │
│ API Server       ✅ Healthy  45ms   99.9%   │
│ AI Engine        ✅ Healthy  230ms  99.8%   │
│ Database         ✅ Healthy  12ms   100%    │
│ SEO Crawler      ⚠️ Degraded  —     98.5%   │
│ Social API       ✅ Healthy  180ms  99.7%   │
├─────────────────────────────────────────────┤
│ Error Rate (24h)                            │
│ 4xx: 23 (0.3%)  5xx: 2 (0.03%)             │
│ [chart showing error rate over time]        │
├─────────────────────────────────────────────┤
│ Active Sessions: 34                         │
│ Concurrent API calls: 12                    │
│ Queue depth: 0                              │
├─────────────────────────────────────────────┤
│ Scheduled Jobs                              │
│ SEO rank tracking    ✅ Last run: 2h ago    │
│ Social data refresh  ✅ Last run: 1h ago    │
│ Insight generation   ✅ Last run: 30m ago   │
│ Metric aggregation   ✅ Last run: 15m ago   │
└─────────────────────────────────────────────┘
```

---

## 14. Implementation Phases

### Phase 1: Admin Sidebar + Enhanced User Management (Week 1-2)

**Backend:**
- [ ] Add `disabled_at`, `disabled_reason`, `last_login_at`, `login_count` to User model
- [ ] Create migration for new columns
- [ ] Add `GET /api/admin/users/:id/activity` endpoint
- [ ] Add `POST /api/admin/users/:id/suspend` endpoint
- [ ] Add `POST /api/admin/users/:id/unsuspend` endpoint
- [ ] Add `GET /api/admin/users/export` endpoint (CSV)
- [ ] Enhance `ListUsers` with additional filters (role, mode, status, date range)
- [ ] Log login events to UserActivity table

**Frontend:**
- [ ] Create `AdminSidebar` component with grouped navigation
- [ ] Modify `app/(dashboard)/layout.tsx` for sidebar switching
- [ ] Enhance user list page with advanced filter UI
- [ ] Add tabbed interface to user detail page (Profile, Projects, Activity, Subscription)
- [ ] Add activity timeline component
- [ ] Add suspend/unsuspend toggle
- [ ] Add user export button

### Phase 2: Project Monitoring (Week 3-4)

**Backend:**
- [ ] Create `ProjectMetric` summary endpoint for admin
- [ ] Add `GET /api/admin/projects` endpoint (paginated, filterable by owner, goal, health)
- [ ] Add `GET /api/admin/projects/:id` endpoint (full detail)
- [ ] Add `GET /api/admin/projects/:id/metrics` endpoint (time-series)
- [ ] Add `GET /api/admin/projects/:id/seo` endpoint (aggregated SEO data)
- [ ] Add `GET /api/admin/projects/:id/social` endpoint (aggregated social data)
- [ ] Add `GET /api/admin/projects/stats` endpoint (platform-wide aggregation)

**Frontend:**
- [ ] Create admin projects list page with filters
- [ ] Create admin project detail page with tabs (Overview, SEO, Social, Settings)
- [ ] Create SEO summary component for project detail
- [ ] Create social summary component for project detail
- [ ] Add "View Projects" link to user detail page
- [ ] Add platform SEO health dashboard
- [ ] Add platform social health dashboard

### Phase 3: Platform Analytics (Week 5-6)

**Backend:**
- [ ] Create `PlatformMetric` model + migration
- [ ] Create daily aggregation job (runs at midnight)
- [ ] Add `GET /api/admin/platform/stats` endpoint
- [ ] Add `GET /api/admin/platform/engagement` endpoint
- [ ] Add `GET /api/admin/platform/feature-usage` endpoint
- [ ] Add API usage tracking middleware

**Frontend:**
- [ ] Enhance admin overview with platform-wide metrics
- [ ] Create feature adoption chart
- [ ] Create revenue dashboard page
- [ ] Add engagement trends visualization

### Phase 4: System Health (Week 7-8)

**Backend:**
- [ ] Create `SystemHealth` model + migration
- [ ] Add health check endpoints for each service
- [ ] Add `GET /api/admin/platform/health` endpoint
- [ ] Add `GET /api/admin/integrations/status` endpoint
- [ ] Add scheduled job status endpoint
- [ ] Create periodic health check job (every 5 minutes)

**Frontend:**
- [ ] Create system health page
- [ ] Create integration status page
- [ ] Add real-time status indicators
- [ ] Add error rate charts

---

## 15. Deferred Items

These items from the original IMPLEMENTATION_PLAN.md remain deferred:

| ID | Description | Reason | Target Phase |
|----|-------------|--------|--------------|
| DEFER-004 | User suspension/deletion | Needs `disabled_at` column | Phase 1 |
| DEFER-005 | SubscriptionEvent history | Needs new table | Phase 3 |
| DEFER-006 | Auto-bootstrap admin on first boot | Needs startup hook | Phase 4 |
| DEFER-007 | Subscription row accumulation | Needs cleanup job | Phase 3 |

---

## Appendix A: Admin Sidebar Component Spec

```typescript
// components/admin/admin-sidebar.tsx

interface AdminNavItem {
  name: string;
  href: string;
  icon: LucideIcon;
}

interface AdminNavGroup {
  title: string;
  items: AdminNavItem[];
}

const ADMIN_NAV_GROUPS: AdminNavGroup[] = [
  {
    title: "Overview",
    items: [
      { name: "Platform Overview", href: "/admin", icon: LayoutDashboard },
    ],
  },
  {
    title: "Users",
    items: [
      { name: "All Users", href: "/admin/users", icon: Users },
      { name: "Subscriptions", href: "/admin/subscriptions", icon: CreditCard },
    ],
  },
  {
    title: "Projects",
    items: [
      { name: "All Projects", href: "/admin/projects", icon: Globe },
      { name: "SEO Health", href: "/admin/seo-health", icon: Search },
      { name: "Social Health", href: "/admin/social-health", icon: Share2 },
    ],
  },
  {
    title: "Finance",
    items: [
      { name: "Plans", href: "/admin/plans", icon: Package },
      { name: "Revenue", href: "/admin/revenue", icon: TrendingUp },
    ],
  },
  {
    title: "System",
    items: [
      { name: "Health", href: "/admin/system", icon: Activity },
      { name: "Audit Log", href: "/admin/audit-log", icon: ScrollText },
    ],
  },
];
```

## Appendix B: Route Mapping

| Admin Sidebar Link | Route | Backend Endpoint | Description |
|--------------------|-------|------------------|-------------|
| Platform Overview | `/admin` | `GET /api/admin/stats` | Dashboard with key metrics |
| All Users | `/admin/users` | `GET /api/admin/users` | Paginated user list |
| User Detail | `/admin/users/[id]` | `GET /api/admin/users/:id` | User profile + actions |
| User Projects | `/admin/users/[id]/projects` | `GET /api/admin/projects?user_id=X` | User's projects |
| User Activity | `/admin/users/[id]/activity` | `GET /api/admin/users/:id/activity` | Activity timeline |
| All Projects | `/admin/projects` | `GET /api/admin/projects` | All projects list |
| Project Detail | `/admin/projects/[id]` | `GET /api/admin/projects/:id` | Project deep dive |
| SEO Health | `/admin/seo-health` | `GET /api/admin/projects/stats?goal=seo` | Platform SEO overview |
| Social Health | `/admin/social-health` | `GET /api/admin/projects/stats?goal=social` | Platform social overview |
| Plans | `/admin/plans` | `GET /api/admin/plans` | Plan management |
| Revenue | `/admin/revenue` | `GET /api/admin/platform/stats` | Revenue analytics |
| Health | `/admin/system` | `GET /api/admin/platform/health` | System status |
| Audit Log | `/admin/audit-log` | `GET /api/admin/audit-log` | Action history |
