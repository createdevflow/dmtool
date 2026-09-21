"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  ArrowLeft, Globe, Activity, Search, Share2, AlertTriangle,
  TrendingUp, Users, BarChart3, Target, ExternalLink
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  adminApi, type AdminProjectSummary, type AdminSocialMetric,
  type AdminInsight, type SEOIssue, type KeywordResult, type Metric
} from "@/lib/api-client";
import { formatDate, formatDateTime } from "@/lib/utils";
import { cn } from "@/lib/utils";

type Tab = "overview" | "seo" | "social";

const TABS: { key: Tab; label: string; icon: typeof Globe }[] = [
  { key: "overview", label: "Overview", icon: Globe },
  { key: "seo", label: "SEO", icon: Search },
  { key: "social", label: "Social", icon: Share2 },
];

export default function AdminProjectDetailPage() {
  const params = useParams<{ id: string }>();
  const id = Number(params.id);
  const router = useRouter();

  const [project, setProject] = useState<any>(null);
  const [owner, setOwner] = useState<{ id: number; name: string; email: string } | null>(null);
  const [seoData, setSeoData] = useState<{ open_issues: number; critical_issues: number; keyword_count: number } | null>(null);
  const [socialData, setSocialData] = useState<AdminSocialMetric | null>(null);
  const [insights, setInsights] = useState<AdminInsight[]>([]);
  const [seoIssues, setSeoIssues] = useState<SEOIssue[]>([]);
  const [keywords, setKeywords] = useState<KeywordResult[]>([]);
  const [socialHistory, setSocialHistory] = useState<AdminSocialMetric[]>([]);
  const [metrics, setMetrics] = useState<Metric[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<Tab>("overview");

  const fetchOverview = useCallback(async () => {
    try {
      const r = await adminApi.getProject(id);
      setProject(r.data.data.project);
      setOwner(r.data.data.owner);
      setSeoData(r.data.data.seo);
      setSocialData(r.data.data.social);
      setInsights(r.data.data.insights);
    } catch {
      // Guard handles 403.
    }
  }, [id]);

  const fetchSEO = useCallback(async () => {
    try {
      const r = await adminApi.getProjectSEO(id);
      setSeoIssues(r.data.data.issues);
      setKeywords(r.data.data.keywords);
    } catch {
      // Non-critical.
    }
  }, [id]);

  const fetchSocial = useCallback(async () => {
    try {
      const r = await adminApi.getProjectSocial(id);
      setSocialData(r.data.data.latest);
      setSocialHistory(r.data.data.history);
    } catch {
      // Non-critical.
    }
  }, [id]);

  const fetchMetrics = useCallback(async () => {
    try {
      const r = await adminApi.getProjectMetrics(id);
      setMetrics(r.data.data.metrics);
    } catch {
      // Non-critical.
    }
  }, [id]);

  useEffect(() => {
    if (isNaN(id)) return;
    setLoading(true);
    Promise.all([fetchOverview(), fetchMetrics()]).finally(() => setLoading(false));
  }, [id, fetchOverview, fetchMetrics]);

  useEffect(() => {
    if (activeTab === "seo") fetchSEO();
    if (activeTab === "social") fetchSocial();
  }, [activeTab, fetchSEO, fetchSocial]);

  if (loading || !project) {
    return <div className="text-slate-400 py-12 text-center">Loading project…</div>;
  }

  const goalLabel = (g: string) => {
    switch (g) {
      case "seo": return "SEO";
      case "social": return "Social";
      case "both": return "SEO + Social";
      default: return g;
    }
  };

  const healthColor = (h: string) => {
    switch (h) {
      case "healthy": return "bg-emerald-100 text-emerald-800";
      case "issues": return "bg-amber-100 text-amber-800";
      case "scanning": return "bg-blue-100 text-blue-800";
      default: return "bg-slate-100 text-slate-700";
    }
  };

  const severityColor = (s: string) => {
    switch (s) {
      case "high": return "bg-red-100 text-red-800";
      case "medium": return "bg-amber-100 text-amber-800";
      case "low": return "bg-slate-100 text-slate-700";
      default: return "bg-slate-100 text-slate-700";
    }
  };

  return (
    <div className="space-y-6">
      <button
        onClick={() => router.push("/admin/projects")}
        className="text-xs text-slate-500 hover:text-slate-900 inline-flex items-center gap-1"
      >
        <ArrowLeft className="w-3 h-3" /> Back to projects
      </button>

      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Globe className="w-6 h-6 text-slate-700" />
            {project.name}
          </h1>
          <p className="text-sm text-slate-500 mt-1 flex items-center gap-2">
            <span>{project.url}</span>
            <a href={project.url} target="_blank" rel="noopener noreferrer" className="text-slate-400 hover:text-slate-600">
              <ExternalLink className="w-3 h-3" />
            </a>
          </p>
          {owner && (
            <p className="text-xs text-slate-400 mt-1">
              Owner: {owner.name} ({owner.email})
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Badge className={healthColor(project.health)}>
            {project.health} {project.health_score > 0 && `(${project.health_score}/100)`}
          </Badge>
          <Badge variant="outline">{goalLabel(project.goal)}</Badge>
        </div>
      </div>

      {/* Tab navigation */}
      <div className="flex items-center gap-1 border-b border-slate-200">
        {TABS.map((t) => {
          const Icon = t.icon;
          return (
            <button
              key={t.key}
              onClick={() => setActiveTab(t.key)}
              className={cn(
                "px-4 py-2.5 text-sm font-medium inline-flex items-center gap-2 transition-colors border-b-2 -mb-px",
                activeTab === t.key
                  ? "border-slate-900 text-slate-900"
                  : "border-transparent text-slate-500 hover:text-slate-700"
              )}
            >
              <Icon className="w-4 h-4" />
              {t.label}
            </button>
          );
        })}
      </div>

      {/* Tab content */}
      {activeTab === "overview" && (
        <div className="space-y-6">
          {/* Stats grid */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard label="Health Score" value={`${project.health_score}/100`} />
            <StatCard label="Open Issues" value={seoData?.open_issues || 0} className="text-amber-600" />
            <StatCard label="Critical Issues" value={seoData?.critical_issues || 0} className="text-red-600" />
            <StatCard label="Keywords" value={seoData?.keyword_count || 0} />
          </div>

          {/* Social summary */}
          {socialData && socialData.followers > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Social Summary</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <div className="text-xs text-slate-500">Platform</div>
                    <div className="font-medium">{socialData.platform}</div>
                  </div>
                  <div>
                    <div className="text-xs text-slate-500">Followers</div>
                    <div className="font-medium">{socialData.followers.toLocaleString()}</div>
                  </div>
                  <div>
                    <div className="text-xs text-slate-500">Posts</div>
                    <div className="font-medium">{socialData.posts_count}</div>
                  </div>
                  <div>
                    <div className="text-xs text-slate-500">Engagement</div>
                    <div className="font-medium">{socialData.engagement_rate?.toFixed(1)}%</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Recent insights */}
          {insights.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Recent Insights</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {insights.slice(0, 5).map((insight) => (
                    <div key={insight.id} className="flex items-start gap-3">
                      <Badge
                        className={cn(
                          "text-[10px] mt-0.5",
                          insight.type === "CRITICAL" ? "bg-red-100 text-red-800" :
                          insight.type === "OPPORTUNITY" ? "bg-emerald-100 text-emerald-800" :
                          "bg-slate-100 text-slate-700"
                        )}
                      >
                        {insight.type}
                      </Badge>
                      <div>
                        <div className="text-sm font-medium text-slate-900">{insight.title}</div>
                        <div className="text-xs text-slate-500 line-clamp-2">{insight.body}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Metrics chart (simple text representation) */}
          {metrics.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Traffic Trends</CardTitle>
                <CardDescription>Last {metrics.length} data points</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {metrics.slice(0, 10).map((m) => (
                    <div key={m.id} className="flex items-center gap-4 text-sm">
                      <span className="text-slate-500 tabular-nums w-24">{m.date}</span>
                      <span className="font-medium tabular-nums">{m.clicks} clicks</span>
                      <span className="text-slate-400 tabular-nums">{m.impressions} imp</span>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {activeTab === "seo" && (
        <div className="space-y-6">
          {/* SEO Issues */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Open SEO Issues ({seoIssues.length})</CardTitle>
            </CardHeader>
            <CardContent>
              {seoIssues.length === 0 ? (
                <div className="text-sm text-slate-500 py-4 text-center">No open SEO issues.</div>
              ) : (
                <div className="space-y-3">
                  {seoIssues.map((issue) => (
                    <div key={issue.id} className="flex items-start gap-3 p-3 rounded-lg bg-slate-50">
                      <Badge className={cn("text-[10px] mt-0.5", severityColor(issue.severity))}>
                        {issue.severity}
                      </Badge>
                      <div className="flex-1">
                        <div className="text-sm font-medium text-slate-900">{issue.category}</div>
                        <div className="text-xs text-slate-500">{issue.detail}</div>
                        <div className="text-xs text-slate-400 mt-1">{issue.url}</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Keywords */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Keywords ({keywords.length})</CardTitle>
            </CardHeader>
            <CardContent>
              {keywords.length === 0 ? (
                <div className="text-sm text-slate-500 py-4 text-center">No keywords tracked.</div>
              ) : (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-left text-xs text-slate-500 border-b border-slate-200">
                      <th className="pb-2 font-medium">Keyword</th>
                      <th className="pb-2 font-medium">Volume</th>
                      <th className="pb-2 font-medium">KD</th>
                      <th className="pb-2 font-medium">Position</th>
                    </tr>
                  </thead>
                  <tbody>
                    {keywords.slice(0, 20).map((kw) => (
                      <tr key={kw.id} className="border-b border-slate-100">
                        <td className="py-2 font-medium">{kw.keyword}</td>
                        <td className="py-2 tabular-nums">{kw.volume.toLocaleString()}</td>
                        <td className="py-2 tabular-nums">{kw.kd}</td>
                        <td className="py-2 tabular-nums">{kw.position > 0 ? `#${kw.position}` : "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {activeTab === "social" && (
        <div className="space-y-6">
          {/* Current social stats */}
          {socialData && socialData.followers > 0 ? (
            <>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <StatCard label="Followers" value={socialData.followers.toLocaleString()} />
                <StatCard label="Posts" value={socialData.posts_count} />
                <StatCard label="Reach" value={socialData.reach.toLocaleString()} />
                <StatCard label="Engagement" value={`${socialData.engagement_rate?.toFixed(1)}%`} />
              </div>

              {/* Social history */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Social History</CardTitle>
                  <CardDescription>Last {socialHistory.length} snapshots</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {socialHistory.slice(0, 15).map((s) => (
                      <div key={s.id} className="flex items-center gap-4 text-sm">
                        <span className="text-slate-500 tabular-nums w-32">{formatDate(s.recorded_at)}</span>
                        <span className="font-medium tabular-nums">{s.followers.toLocaleString()} followers</span>
                        <Badge className={cn("text-[10px]",
                          s.status === "growing" ? "bg-emerald-100 text-emerald-800" :
                          s.status === "dropping" ? "bg-red-100 text-red-800" :
                          "bg-slate-100 text-slate-700"
                        )}>
                          {s.status}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </>
          ) : (
            <Card>
              <CardContent className="py-12 text-center text-slate-500">
                No social data connected for this project.
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </div>
  );
}

function StatCard({ label, value, className }: { label: string; value: string | number; className?: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="text-xs font-medium text-slate-500 uppercase tracking-wide">{label}</div>
        <div className={`text-2xl font-bold tabular-nums ${className || "text-slate-900"}`}>{value}</div>
      </CardContent>
    </Card>
  );
}
