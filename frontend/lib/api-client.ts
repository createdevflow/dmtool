import axios, { type AxiosError, type InternalAxiosRequestConfig } from "axios";
import {
  readCookie,
  COOKIE_TOKEN,
  COOKIE_IMPERSONATION_TOKEN,
  setToken,
  clearAuth,
  clearImpersonation,
} from "./auth-cookie";
import { accessTokenExpiresAtMs } from "./access-token";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

type RetryConfig = InternalAxiosRequestConfig & { _retry?: boolean };

const jsonHeaders = { "Content-Type": "application/json" };

const apiClient = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
  headers: jsonHeaders,
});

// Login/register/refresh/logout: cookies required, but a 401 here must
// NOT run the session interceptor (wrong password ≠ dead session).
const authClient = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
  headers: jsonHeaders,
});

let refreshInFlight: Promise<string | null> | null = null;
let keepAliveTimer: ReturnType<typeof setTimeout> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  const res = await authClient.post("/auth/refresh");
  const token = (res.data as { data?: { token?: string } })?.data?.token;
  if (typeof token !== "string" || !token) return null;
  setToken(token);
  return token;
}

function runRefresh(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = refreshAccessToken().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

export function stopSessionKeepAlive(): void {
  if (keepAliveTimer) {
    clearTimeout(keepAliveTimer);
    keepAliveTimer = null;
  }
}

export function startSessionKeepAlive(): void {
  if (typeof window === "undefined") return;
  stopSessionKeepAlive();
  const token = readCookie(COOKIE_TOKEN);
  if (!token) return;
  const exp = accessTokenExpiresAtMs(token);
  if (exp == null) return;
  const fireIn = Math.max(exp - Date.now() - 60_000, 0);
  keepAliveTimer = setTimeout(() => {
    void runRefresh()
      .then((next) => {
        if (next) startSessionKeepAlive();
      })
      .catch(() => {
        // Network blip — leave the access token; the 401 interceptor retries.
      });
  }, fireIn);
}

function forceLogout(): void {
  stopSessionKeepAlive();
  clearAuth();
  clearImpersonation();
  if (typeof window === "undefined") return;
  const path = window.location.pathname;
  if (path !== "/login" && path !== "/register") {
    window.location.href = "/login";
  }
}

// Request interceptor — attach JWT token to every request.
//
// Token priority (phase 7):
//   1. dmtool_impersonation_token cookie (admin user-support flow).
//      While present, all requests carry the impersonation token —
//      including Stop. POST /admin/users/:id/stop-impersonation is
//      registered outside RequireRole("admin") so that JWT is accepted.
//   2. dmtool_token cookie (phase 3 source of truth).
apiClient.interceptors.request.use(
  (config) => {
    if (typeof document !== "undefined") {
      if (!keepAliveTimer) startSessionKeepAlive();
      const impToken = readCookie(COOKIE_IMPERSONATION_TOKEN);
      if (impToken) {
        config.headers.Authorization = `Bearer ${impToken}`;
        return config;
      }
      const token = readCookie(COOKIE_TOKEN);
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    }
    return config;
  },
  (error) => Promise.reject(error)
);

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const original = error.config as RetryConfig | undefined;
    if (error.response?.status !== 401 || !original) {
      return Promise.reject(error);
    }

    // Impersonation JWT expired (or target disabled): drop the support
    // session and retry as the admin. Do not wipe dmtool_token.
    if (typeof document !== "undefined" && readCookie(COOKIE_IMPERSONATION_TOKEN) && !original._retry) {
      clearImpersonation();
      if (original.headers) {
        delete (original.headers as Record<string, unknown>).Authorization;
      }
      return apiClient(original);
    }

    if (original._retry) {
      forceLogout();
      return Promise.reject(error);
    }
    original._retry = true;

    try {
      const token = await runRefresh();
      if (!token) {
        forceLogout();
        return Promise.reject(error);
      }
      startSessionKeepAlive();
      original.headers = original.headers ?? {};
      original.headers.Authorization = `Bearer ${token}`;
      return apiClient(original);
    } catch (e) {
      const status = axios.isAxiosError(e) ? e.response?.status : undefined;
      if (status === 401) forceLogout();
      return Promise.reject(error);
    }
  }
);

// ── Auth API ──────────────────────────────────────────────────────────────────
export const authApi = {
  register: (data: { name: string; email: string; password: string }) =>
    authClient.post("/auth/register", data),
  login: (data: { email: string; password: string }) =>
    authClient.post("/auth/login", data),
  refresh: () => authClient.post("/auth/refresh"),
  me: () => apiClient.get("/auth/me"),
  updateMe: (data: { dashboard_mode: string }) => apiClient.patch("/auth/me", data),
  logout: () => authClient.post("/auth/logout"),
};

// ── Projects API ─────────────────────────────────────────────────────────────
export const projectsApi = {
  list: () => apiClient.get("/projects"),
  create: (data: { name: string; url: string; goal: string; ig_handle?: string; twitter_handle?: string; linkedin_handle?: string; fb_handle?: string }) =>
    apiClient.post("/projects", data),
  update: (id: number, data: Record<string, unknown>) => apiClient.patch(`/projects/${id}`, data),
  delete: (id: number) => apiClient.delete(`/projects/${id}`),
  onboard: (data: Record<string, unknown>) => apiClient.post("/onboard", data),
};

export const dashboardApi = {
  getProjects: () => apiClient.get("/projects"),
  onboard: (data: Record<string, unknown>) => apiClient.post("/onboard", data),
  deleteProject: (id: number) => apiClient.delete(`/projects/${id}`),
  updateProject: (id: number, data: Record<string, unknown>) =>
    apiClient.patch(`/projects/${id}`, data),

  getSnapshot: (projectId: number) =>
    apiClient.get(`/dashboard/snapshot?project_id=${projectId}`),
  getMetrics: (projectId: number, days = 30) =>
    apiClient.get(`/dashboard/metrics?project_id=${projectId}&days=${days}`),
  getInsights: (projectId: number) =>
    apiClient.get(`/dashboard/insights?project_id=${projectId}`),
  getTasks: (projectId: number) =>
    apiClient.get(`/dashboard/tasks?project_id=${projectId}`),
  createTask: (data: Record<string, unknown>) => apiClient.post(`/dashboard/tasks`, data),
  toggleTask: (id: number, projectId?: number) =>
    apiClient.patch(`/tasks/${id}/toggle${projectId ? `?project_id=${projectId}` : ""}`),
  getTraffic: (projectId: number, days = 30) =>
    apiClient.get(`/dashboard/traffic?project_id=${projectId}&days=${days}`),
  getAlerts: (projectId: number) => apiClient.get(`/alerts?project_id=${projectId}`),
  getCompetitors: (projectId: number) => apiClient.get(`/competitors?project_id=${projectId}`),

  runSeoAudit: (projectId: number, url?: string) =>
    apiClient.post(`/seo/audit/run`, { project_id: projectId, url: url ?? "" }),
  getSeoAudit: (projectId: number) =>
    apiClient.get(`/seo/audit?project_id=${projectId}`),
  getSeoIssues: (projectId: number, severity?: string) =>
    apiClient.get(
      `/seo/issues?project_id=${projectId}${severity ? `&severity=${severity}` : ""}`
    ),
  getSeoReport: (projectId: number) =>
    apiClient.get(`/seo/report?project_id=${projectId}`),
  getKeywords: (projectId: number, seed?: string) =>
    apiClient.get(
      `/seo/keywords?project_id=${projectId}${seed ? `&seed=${encodeURIComponent(seed)}` : ""}`
    ),
  generateKeywords: (data: { project_id: number; seed: string }) =>
    apiClient.post(`/seo/keywords`, data),
  resolveIssue: (issueId: number, projectId: number) =>
    apiClient.put(`/seo/issues/${issueId}?project_id=${projectId}`),

  getSocialInsights: (projectId: number) =>
    apiClient.get(`/social/insights?project_id=${projectId}`),
  refreshSocial: (projectId: number) =>
    apiClient.post(`/social/insights/refresh?project_id=${projectId}`),
  getSocialHistory: (projectId: number) =>
    apiClient.get(`/social/history?project_id=${projectId}`),
  getPublicProfile: (handle: string, platform = "instagram", projectId?: number) =>
    apiClient.get(
      `/social/profile?handle=${encodeURIComponent(handle)}&platform=${platform}${projectId ? `&project_id=${projectId}` : ""}`
    ),
  getRelatedProfiles: (projectId: number) =>
    apiClient.get(`/social/related?project_id=${projectId}`),

  generateContent: (data: {
    project_id: number;
    topic: string;
    platform: string;
    tone?: string;
  }) => apiClient.post(`/content/generate`, data),

  getIntegrations: () => apiClient.get("/integrations"),
  getGoogleAuthUrl: () => apiClient.get("/integrations/google/auth-url"),
  getMetaAuthUrl: () => apiClient.get("/integrations/meta/auth-url"),
  getLinkedinAuthUrl: () => apiClient.get("/integrations/linkedin/auth-url"),
  disconnectIntegration: (provider: string) =>
    apiClient.delete(`/integrations/${provider}`),

  generateKeywordsLegacy: (data: Record<string, unknown>) => apiClient.post("/seo/keywords", data),

  getProject: (id: number) => apiClient.get(`/projects/${id}`),

  getAutomations: (projectId: number) => apiClient.get(`/system/automations?project_id=${projectId}`),
  createAutomation: (data: Record<string, unknown>) => apiClient.post(`/system/automations`, data),
  toggleAutomation: (id: number) => apiClient.patch(`/system/automations/${id}/toggle`),

  getCalendar: (projectId: number) => apiClient.get(`/system/calendar?project_id=${projectId}`),
  createCalendarEvent: (data: FormData | Record<string, unknown>) => {
    const isFormData = typeof FormData !== "undefined" && data instanceof FormData;
    return apiClient.post(`/system/calendar/event`, data, isFormData ? { headers: { "Content-Type": "multipart/form-data" } } : undefined);
  },
  updateCalendarEvent: (id: number, data: FormData | Record<string, unknown>) => {
    const isFormData = typeof FormData !== "undefined" && data instanceof FormData;
    return apiClient.patch(`/system/calendar/event/${id}`, data, isFormData ? { headers: { "Content-Type": "multipart/form-data" } } : undefined);
  },
  deleteCalendarEvent: (id: number) => apiClient.delete(`/system/calendar/event/${id}`),

  syncProject: (projectId: number) => apiClient.post(`/projects/${projectId}/sync`),

  getRankTracking: (projectId: number) => apiClient.get(`/seo/rank-tracking?project_id=${projectId}`),
  getBacklinks: (projectId: number) => apiClient.get(`/seo/backlinks?project_id=${projectId}`),

  getSocialHistoryForDelta: (projectId: number, days = 7) =>
    apiClient.get(`/social/history?project_id=${projectId}&days=${days}`),
};

export const socialApi = {
  getInsights: (projectId: number) =>
    apiClient.get(`/social/insights?project_id=${projectId}`),
  refreshSocial: (projectId: number) =>
    apiClient.post(`/social/insights/refresh?project_id=${projectId}`),
  getHistory: (projectId: number) =>
    apiClient.get(`/social/history?project_id=${projectId}`),
  getPublicProfile: (handle: string, platform = "instagram") =>
    apiClient.get(
      `/social/profile?handle=${encodeURIComponent(handle)}&platform=${platform}`
    ),
};

export const seoApi = {
  runAudit: (projectId: number, url?: string) =>
    apiClient.post(`/seo/audit/run`, { project_id: projectId, url: url ?? "" }),
  getStatus: (projectId: number) =>
    apiClient.get(`/seo/audit?project_id=${projectId}`),
  getIssues: (projectId: number, severity?: string) =>
    apiClient.get(
      `/seo/issues?project_id=${projectId}${severity ? `&severity=${severity}` : ""}`
    ),
  getKeywords: (projectId: number, seed?: string) =>
    apiClient.get(
      `/seo/keywords?project_id=${projectId}${seed ? `&seed=${encodeURIComponent(seed)}` : ""}`
    ),
  generateKeywords: (projectId: number, seed: string) =>
    apiClient.post(`/seo/keywords`, { project_id: projectId, seed }),
  getReport: (projectId: number) =>
    apiClient.get(`/seo/report?project_id=${projectId}`),
  resolveIssue: (issueId: number, projectId: number) =>
    apiClient.put(`/seo/issues/${issueId}?project_id=${projectId}`),
};

export const publicApi = {
  seoAudit: (url: string) =>
    axios.get(`${API_BASE.replace("/api", "")}/api/public/seo-audit?url=${encodeURIComponent(url)}`),
};
// ── Billing API ──────────────────────────────────────────────────────────────
// Phase 6. The Subscribe and StartTrial endpoints are stub-Stripe (no
// real charge) until billing/webhook is implemented; the trial endpoint
// starts a 14-day trialing row on a pro plan.
export type Project = {
  id: number;
  created_at: string;
  updated_at: string;
  user_id: number;
  name: string;
  url: string;
  goal: string;
  status: string;
  health: string;
  health_score: number;
  ig_handle: string;
  facebook_handle: string;
  twitter_handle: string;
  linkedin_handle: string;
};

export type Plan = {
  id: number;
  code: string;
  name: string;
  description: string;
  tier_rank: number;
  monthly_cents: number;
  yearly_cents: number;
  currency: string;
  max_sites: number;
  is_active: boolean;
};

export type Subscription = {
  id: number;
  user_id: number;
  plan_code: string;
  status: "active" | "trialing" | "past_due" | "canceled";
  trial_ends_at: string | null;
  starts_at: string;
  ends_at: string | null;
  canceled_at: string | null;
  stripe_sub_id: string;
};

export type BillingState = {
  subscription: Subscription | null;
  plan: Plan;
  usage: { seo_projects: number; max_sites: number };
};

export const billingApi = {
  me: () => apiClient.get<{ data: BillingState }>("/billing/me"),
  startTrial: (planCode: "pro_monthly" | "pro_yearly") =>
    apiClient.post<{ data: Subscription }>("/billing/trial", { plan_code: planCode }),
  subscribe: (planCode: "pro_monthly" | "pro_yearly") =>
    apiClient.post<{ data: Subscription }>("/billing/subscribe", { plan_code: planCode }),
  cancel: () => apiClient.post<{ data: Subscription }>("/billing/cancel"),
};

export const plansApi = {
  list: () => axios.get<{ data: Plan[] }>(`${API_BASE}/plans`),
};

// ── Admin API ──────────────────────────────────────────────────────────────
// Phase 7. Gated by RequireRole('admin') on the backend — the routes
// return 403 to non-admins and to admins using an impersonation token.
export type AdminUserSummary = {
  id: number;
  email: string;
  name: string;
  role: string;
  plan_code: string;
  sub_status: string;
  created_at: string;
  trial_used_at?: string | null;
  last_login_at?: string | null;
  disabled_at?: string | null;
};

export type AdminPlan = {
  id: number;
  code: string;
  name: string;
  description: string;
  tier_rank: number;
  monthly_cents: number;
  yearly_cents: number;
  currency: string;
  max_sites: number;
  is_active: boolean;
};

export type AdminAuditEntry = {
  id: number;
  created_at: string;
  actor_user_id: number;
  target_user_id: number;
  action: string;
  metadata: Record<string, unknown> | null;
};

export type AdminPermission = {
  id: number;
  code: string;
  name: string;
  description: string;
  category: string;
};

export type AdminRole = {
  id: number;
  code: string;
  name: string;
  description: string;
  is_system: boolean;
  permissions?: AdminPermission[];
  user_count?: number;
};

export type AdminStats = {
  total_users: number;
  new_users_7d: number;
  new_users_30d: number;
  active_subs: number;
  trialing_subs: number;
  canceled_subs: number;
  mrr_cents: number;
  arr_proxy_cents: number;
  revenue_available?: boolean;
  plan_breakdown: Record<string, number>;
  user_growth_30d: Array<{ date: string; count: number }>;
  recent_activity: AdminAuditEntry[];
  // Phase 3: project stats + feature adoption
  total_projects: number;
  projects_new_7d: number;
  projects_new_30d: number;
  avg_health_score: number;
  health_breakdown: Record<string, number>;
  goal_breakdown: Record<string, number>;
  feature_adoption: Record<string, number>;
};

export type PlatformSEOHealth = {
  total_seo_projects: number;
  avg_health_score: number;
  total_keywords: number;
  keywords_in_top_10: number;
  total_open_issues: number;
  critical_issues: number;
  issue_breakdown: Record<string, number>;
  health_distribution: Record<string, number>;
};

export type PlatformSocialHealth = {
  total_social_projects: number;
  total_followers: number;
  avg_engagement_rate: number;
  total_reach: number;
  platform_breakdown: Record<string, number>;
  status_breakdown: Record<string, number>;
  top_projects: Array<{
    project_id: number;
    name: string;
    url: string;
    owner_email: string;
    platform: string;
    followers: number;
    engagement_rate: number;
    reach: number;
  }>;
};

export type AdminUserActivity = {
  id: number;
  created_at: string;
  user_id: number;
  action: string;
  metadata: Record<string, unknown> | null;
  ip_address: string;
  user_agent: string;
};

export type AdminProjectSummary = {
  id: number;
  name: string;
  url: string;
  user_id: number;
  owner_email: string;
  goal: string;
  status: string;
  health: string;
  health_score: number;
  created_at: string;
};

export type AdminSocialMetric = {
  id: number;
  project_id: number;
  platform: string;
  followers: number;
  following_count: number;
  posts_count: number;
  reach: number;
  engagement_rate: number;
  status: string;
  is_simulated: boolean;
  recorded_at: string;
};

export type AdminInsight = {
  id: number;
  created_at: string;
  project_id: number;
  type: string;
  title: string;
  body: string;
  priority: number;
};

export type SEOIssue = {
  id: number;
  created_at: string;
  project_id: number;
  url: string;
  severity: string;
  category: string;
  detail: string;
  resolved_at: string | null;
};

export type KeywordResult = {
  id: number;
  created_at: string;
  project_id: number;
  seed: string;
  keyword: string;
  volume: number;
  kd: number;
  position: number;
};

export type Metric = {
  id: number;
  project_id: number;
  date: string;
  clicks: number;
  impressions: number;
  reach: number;
  engagement: number;
  source: string;
};

export const adminApi = {
  me: () =>
    apiClient.get<{ data: { role: string; permissions: string[] } }>("/admin/me"),
  listRoles: () => apiClient.get<{ data: AdminRole[] }>("/admin/roles"),
  listPermissions: () => apiClient.get<{ data: AdminPermission[] }>("/admin/permissions"),
  createRole: (body: { code: string; name: string; description?: string; permissions: string[] }) =>
    apiClient.post<{ data: AdminRole }>("/admin/roles", body),
  updateRole: (id: number, body: { name: string; description?: string; permissions: string[] }) =>
    apiClient.patch<{ data: AdminRole }>(`/admin/roles/${id}`, body),
  deleteRole: (id: number) =>
    apiClient.delete<{ data: { id: number; code: string } }>(`/admin/roles/${id}`),
  listUsers: (params: { page?: number; size?: number; search?: string; plan?: string; role?: string; mode?: string; status?: string } = {}) =>
    apiClient.get<{
      data: { users: AdminUserSummary[]; total: number; page: number; size: number };
    }>("/admin/users", { params }),
  getUser: (id: number) =>
    apiClient.get<{
      data: {
        user: AdminUserSummary;
        subscription: Subscription | null;
        audit_log: AdminAuditEntry[];
      };
    }>(`/admin/users/${id}`),
  updateUser: (id: number, body: { name?: string; email?: string; role?: string }) =>
    apiClient.patch<{ data: AdminUserSummary }>(`/admin/users/${id}`, body),
  resetPassword: (id: number) =>
    apiClient.post<{
      data: { user_id: number; temporary_password: string; message: string };
    }>(`/admin/users/${id}/reset-password`),
  getUserActivity: (id: number, limit = 50) =>
    apiClient.get<{
      data: { user_id: number; activities: AdminUserActivity[] };
    }>(`/admin/users/${id}/activity`, { params: { limit } }),
  suspendUser: (id: number, reason?: string) =>
    apiClient.post<{
      data: { user_id: number; disabled_at: string; message: string };
    }>(`/admin/users/${id}/suspend`, { reason: reason || "" }),
  unsuspendUser: (id: number) =>
    apiClient.post<{
      data: { user_id: number; message: string };
    }>(`/admin/users/${id}/unsuspend`),
  exportUsers: () =>
    apiClient.get("/admin/users/export", { responseType: "blob" }),
  impersonate: (id: number) =>
    apiClient.post<{
      data: { token: string; target: { id: number; email: string; name: string }; expires_in_minutes: number };
    }>(`/admin/users/${id}/impersonate`),
  stopImpersonation: (id: number) =>
    apiClient.post<{ data: { message: string } }>(`/admin/users/${id}/stop-impersonation`, { target_id: id }),
  listPlans: () => apiClient.get<{ data: AdminPlan[] }>("/admin/plans"),
  createPlan: (body: Partial<AdminPlan>) =>
    apiClient.post<{ data: AdminPlan }>("/admin/plans", body),
  updatePlan: (id: number, body: Partial<AdminPlan>) =>
    apiClient.patch<{ data: AdminPlan }>(`/admin/plans/${id}`, body),
  listAuditLog: (params: { page?: number; size?: number; actor_id?: number; target_id?: number; action?: string } = {}) =>
    apiClient.get<{
      data: { entries: AdminAuditEntry[]; total: number; page: number; size: number };
    }>("/admin/audit-log", { params }),
  stats: () => apiClient.get<{ data: AdminStats }>("/admin/stats"),
  // ── Projects (admin) ──
  listProjects: (params: { page?: number; size?: number; search?: string; goal?: string; health?: string; owner?: string } = {}) =>
    apiClient.get<{
      data: { projects: AdminProjectSummary[]; total: number; page: number; size: number };
    }>("/admin/projects", { params }),
  getProject: (id: number) =>
    apiClient.get<{
      data: {
        project: Project;
        owner: { id: number; name: string; email: string };
        seo: { open_issues: number; critical_issues: number; keyword_count: number };
        social: AdminSocialMetric | null;
        insights: AdminInsight[];
      };
    }>(`/admin/projects/${id}`),
  getProjectMetrics: (id: number, limit = 30) =>
    apiClient.get<{
      data: { project_id: number; metrics: Metric[] };
    }>(`/admin/projects/${id}/metrics`, { params: { limit } }),
  getProjectSEO: (id: number) =>
    apiClient.get<{
      data: { project_id: number; issues: SEOIssue[]; keywords: KeywordResult[] };
    }>(`/admin/projects/${id}/seo`),
  getProjectSocial: (id: number) =>
    apiClient.get<{
      data: { project_id: number; latest: AdminSocialMetric | null; history: AdminSocialMetric[] };
    }>(`/admin/projects/${id}/social`),
  getProjectStats: () =>
    apiClient.get<{
      data: {
        total_projects: number;
        health_breakdown: Record<string, number>;
        goal_breakdown: Record<string, number>;
        avg_health_score: number;
        new_7d: number;
        new_30d: number;
      };
    }>("/admin/projects/stats"),
  // ── Platform health (Phase 3) ──
  platformSEOHealth: () =>
    apiClient.get<{ data: PlatformSEOHealth }>("/admin/platform/seo-health"),
  platformSocialHealth: () =>
    apiClient.get<{ data: PlatformSocialHealth }>("/admin/platform/social-health"),
  // ── System health (Phase 4) ──
  platformHealth: () =>
    apiClient.get<{
      data: {
        status: string;
        services: Record<string, { status: string; latency?: string; total?: number; expired?: number; sync_errors?: number; with_issues?: number }>;
      };
    }>("/admin/platform/health"),
  integrationsStatus: () =>
    apiClient.get<{
      data: {
        total_connections: number;
        active_connections: number;
        unique_users: number;
        unique_projects: number;
        providers: Array<{ provider: string; total: number; expired: number; sync_errors: number; last_synced_at: string | null }>;
      };
    }>("/admin/integrations/status"),
};

export default apiClient;
