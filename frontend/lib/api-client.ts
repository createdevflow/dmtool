import axios from "axios";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

const apiClient = axios.create({
  baseURL: API_BASE,
  headers: {
    "Content-Type": "application/json",
  },
});

// Request interceptor — attach JWT token to every request
apiClient.interceptors.request.use(
  (config) => {
    if (typeof window !== "undefined") {
      const token = localStorage.getItem("dmtool_token");
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor — handle 401 (token expired) globally
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      if (typeof window !== "undefined") {
        localStorage.removeItem("dmtool_token");
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

// ── Projects API ──────────────────────────────────────────────────────────────
export const projectsApi = {
  list: () => apiClient.get("/projects"),
  create: (data: any) => apiClient.post("/projects", data),
  update: (id: number, data: any) => apiClient.patch(`/projects/${id}`, data),
  delete: (id: number) => apiClient.delete(`/projects/${id}`),
  onboard: (data: any) => apiClient.post("/onboard", data),
};

// ── Dashboard API ─────────────────────────────────────────────────────────────
export const dashboardApi = {
  // Projects (used in dashboard)
  getProjects: () => apiClient.get("/projects"),
  onboard: (data: any) => apiClient.post("/onboard", data),
  deleteProject: (id: number) => apiClient.delete(`/projects/${id}`),
  updateProject: (id: number, data: any) => apiClient.patch(`/projects/${id}`, data),

  // Dashboard data
  getSnapshot: (projectId: number) =>
    apiClient.get(`/dashboard/snapshot?project_id=${projectId}`),
  getMetrics: (projectId: number, days = 30) =>
    apiClient.get(`/dashboard/metrics?project_id=${projectId}&days=${days}`),
  getInsights: (projectId: number) =>
    apiClient.get(`/dashboard/insights?project_id=${projectId}`),
  getTasks: (projectId: number) =>
    apiClient.get(`/dashboard/tasks?project_id=${projectId}`),
  createTask: (data: any) => apiClient.post(`/dashboard/tasks`, data),
  toggleTask: (id: number, projectId?: number) => apiClient.patch(`/tasks/${id}/toggle${projectId ? `?project_id=${projectId}` : ''}`),
  getTraffic: (projectId: number, days = 30) =>
    apiClient.get(`/dashboard/traffic?project_id=${projectId}&days=${days}`),
  getAlerts: (projectId: number) =>
    apiClient.get(`/alerts?project_id=${projectId}`),
  getCompetitors: (projectId: number) =>
    apiClient.get(`/competitors?project_id=${projectId}`),

  // SEO
  runSeoAudit: (projectId: number, url?: string) =>
    apiClient.post(`/seo/audit/run`, { project_id: projectId, url: url || "" }),
  getSeoAudit: (projectId: number) =>
    apiClient.get(`/seo/audit?project_id=${projectId}`),
  getSeoAuditById: (projectId: number) =>
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

  // Social
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

  // Content generation
  generateContent: (data: {
    project_id: number;
    topic: string;
    platform: string;
    tone?: string;
  }) => apiClient.post(`/content/generate`, data),

  // Integrations
  getIntegrations: () => apiClient.get("/integrations"),
  getGoogleAuthUrl: () => apiClient.get("/integrations/google/auth-url"),
  getMetaAuthUrl: () => apiClient.get("/integrations/meta/auth-url"),
  disconnectIntegration: (provider: string) =>
    apiClient.delete(`/integrations/${provider}`),

  // Legacy aliases (keep for compatibility with existing pages)
  generateKeywordsLegacy: (data: any) => apiClient.post("/seo/keywords", data),

  getProject: (id: number) => apiClient.get(`/projects/${id}`),

  getAutomations: (projectId: number) => apiClient.get(`/system/automations?project_id=${projectId}`),
  createAutomation: (data: any) => apiClient.post(`/system/automations`, data),
  toggleAutomation: (id: number) => apiClient.patch(`/system/automations/${id}/toggle`),
  
  getCalendar: (projectId: number) => apiClient.get(`/system/calendar?project_id=${projectId}`),
  createCalendarEvent: (data: any) => apiClient.post(`/system/calendar/event`, data),

  // Sync
  syncProject: (projectId: number) => apiClient.post(`/projects/${projectId}/sync`),

  // SEO Extended
  getRankTracking: (projectId: number) => apiClient.get(`/seo/rank-tracking?project_id=${projectId}`),
  getBacklinks: (projectId: number) => apiClient.get(`/seo/backlinks?project_id=${projectId}`),

  // Social History (for delta calculations)
  getSocialHistoryForDelta: (projectId: number, days = 7) =>
    apiClient.get(`/social/history?project_id=${projectId}&days=${days}`),
};

// ── Social API (separate namespace for clarity) ───────────────────────────────
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

// ── SEO API (separate namespace) ──────────────────────────────────────────────
export const seoApi = {
  runAudit: (projectId: number, url?: string) =>
    apiClient.post(`/seo/audit/run`, { project_id: projectId, url: url || "" }),
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

// ── Public API (no auth required) ────────────────────────────────────────────
export const publicApi = {
  seoAudit: (url: string) =>
    axios.get(`${API_BASE.replace("/api", "")}/api/public/seo-audit?url=${encodeURIComponent(url)}`),
};

export default apiClient;
