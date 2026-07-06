import axios from "axios";
import { readCookie, COOKIE_TOKEN, COOKIE_IMPERSONATION_TOKEN, clearAuthCookie } from "./auth-cookie";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

const apiClient = axios.create({
  baseURL: API_BASE,
  headers: {
    "Content-Type": "application/json",
  },
});

// Request interceptor — attach JWT token to every request.
//
// Token priority (phase 7):
//   1. dmtool_impersonation_token cookie (admin user-support flow).
//      The admin sets this cookie via setImpersonation() when they
//      hit "Impersonate" on /admin/users/:id. While present, all
//      requests carry the impersonation token. The banner's "Stop
//      impersonation" clears the cookie and the regular dmtool_token
//      cookie resumes carrying the admin's real token.
//   2. dmtool_token cookie (phase 3 source of truth).
//
// The proxy at frontend/proxy.ts is configured to accept either
// cookie as a valid auth shape for gated routes — see that file's
// auth gate for the matching logic.
apiClient.interceptors.request.use(
  (config) => {
    if (typeof document !== "undefined") {
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
// Response interceptor — handle 401 (token expired) globally.
// Clears the auth cookie instead of poking localStorage.
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      if (typeof window !== "undefined") {
        clearAuthCookie(COOKIE_TOKEN);
        const path = window.location.pathname;
        if (path !== "/login" && path !== "/register") {
          window.location.href = "/login";
        }
      }
    }
    return Promise.reject(error);
  }
);

// ── Auth API ──────────────────────────────────────────────────────────────────
export const authApi = {
  register: (data: { name: string; email: string; password: string }) =>
    axios.post(`${API_BASE}/auth/register`, data),
  login: (data: { email: string; password: string }) =>
    axios.post(`${API_BASE}/auth/login`, data),
  me: () => apiClient.get("/auth/me"),
  logout: () => apiClient.post("/auth/logout"),
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

export type AdminStats = {
  total_users: number;
  new_users_7d: number;
  new_users_30d: number;
  active_subs: number;
  trialing_subs: number;
  canceled_subs: number;
  mrr_cents: number;
  arr_proxy_cents: number;
  plan_breakdown: Record<string, number>;
  user_growth_30d: Array<{ date: string; count: number }>;
  recent_activity: AdminAuditEntry[];
};

export const adminApi = {
  listUsers: (params: { page?: number; size?: number; search?: string; plan?: string } = {}) =>
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
};

export default apiClient;
