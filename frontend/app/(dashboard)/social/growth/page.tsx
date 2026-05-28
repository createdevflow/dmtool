"use client";

import { motion } from "framer-motion";
import { TrendingUp, Users, UserPlus, Loader2, RefreshCw, ArrowUpRight, ArrowDownRight } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid } from "recharts";
import { useEffect, useState } from "react";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { toast } from "@/components/ui/toaster";

function computeDelta(history: any[], platform: string, field: "followers" | "reach" | "engagement_count") {
  const sorted = history
    .filter((h: any) => h.platform?.toLowerCase() === platform.toLowerCase())
    .sort((a: any, b: any) => new Date(a.recorded_at).getTime() - new Date(b.recorded_at).getTime());

  if (sorted.length < 2) return null;
  const prev = sorted[Math.max(0, sorted.length - 8)][field] ?? 0;
  const curr = sorted[sorted.length - 1][field] ?? 0;
  if (prev === 0) return null;
  const pct = ((curr - prev) / prev) * 100;
  return { pct: pct.toFixed(1), up: pct >= 0 };
}

export default function GrowthTrackingPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [metrics, setMetrics] = useState<any[]>([]);
  const [socialMetrics, setSocialMetrics] = useState<any[]>([]);
  const [socialHistory, setSocialHistory] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);

  const fetchData = async (targetProjectId?: number) => {
    setLoading(true);
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data.data || [];
      setProjects(allProjects);

      if (allProjects.length > 0) {
        const selected = targetProjectId
          ? allProjects.find((p: any) => p.id === targetProjectId) || allProjects[allProjects.length - 1]
          : allProjects[allProjects.length - 1];
        setProject(selected);

        const [mRes, sRes, hRes] = await Promise.all([
          dashboardApi.getMetrics(selected.id),
          dashboardApi.getSocialInsights(selected.id),
          dashboardApi.getSocialHistoryForDelta(selected.id, 30),
        ]);

        const mData = mRes.data?.data;
        const sData = sRes.data?.data;
        const hData = hRes.data?.data;
        setMetrics(Array.isArray(mData) ? mData : []);
        setSocialMetrics(Array.isArray(sData) ? sData : []);
        setSocialHistory(Array.isArray(hData) ? hData : []);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSync = async () => {
    if (!project?.id) return;
    setSyncing(true);
    try {
      await dashboardApi.syncProject(project.id);
      toast("Growth data refreshed!", "success");
      fetchData(project.id);
    } catch {
      toast("Sync failed", "error");
    } finally {
      setSyncing(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  if (loading) return (
    <div className="flex items-center justify-center h-[60vh]">
      <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
    </div>
  );

  // Compute real deltas from history
  const totalFollowers = socialMetrics.reduce((s: number, m: any) => s + (m.followers ?? 0), 0);
  const totalEngagement = socialMetrics.reduce((s: number, m: any) => s + (m.engagement_count ?? 0), 0);
  const totalReach = socialMetrics.reduce((s: number, m: any) => s + (m.reach ?? 0), 0);
  const growthRate = totalReach > 0 ? ((totalEngagement / totalReach) * 100).toFixed(1) : "0.0";

  // Get per-platform deltas
  const platforms = [...new Set(socialHistory.map((h: any) => h.platform).filter(Boolean))];
  const platformDeltas: Record<string, any> = {};
  for (const p of platforms) {
    platformDeltas[p] = {
      followers: computeDelta(socialHistory, p, "followers"),
      reach: computeDelta(socialHistory, p, "reach"),
    };
  }

  // Follower growth chart data from history
  const followerChartData = socialHistory
    .filter((h: any) => h.platform?.toLowerCase() === (platforms[0] ?? "").toLowerCase())
    .sort((a: any, b: any) => new Date(a.recorded_at).getTime() - new Date(b.recorded_at).getTime())
    .slice(-14)
    .map((h: any) => ({
      date: new Date(h.recorded_at).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
      followers: h.followers ?? 0,
      reach: h.reach ?? 0,
    }));

  // Traffic chart from metrics
  const trafficChartData = metrics.map((m: any) => ({
    date: m.date ? new Date(m.date).toLocaleDateString("en-US", { month: "short", day: "numeric" }) : m.name,
    traffic: m.clicks || m.traffic || 0,
    engagement: m.engagement || 0,
  }));

  return (
    <div className="space-y-10 max-w-7xl mx-auto pb-32 pt-4">
      <DashboardHeader
        project={project}
        projects={projects}
        onProjectChange={(p: any) => fetchData(p.id)}
        onAddSource={() => {}}
      />

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Social Analytics</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Growth Tracking</p>
        </div>
        <Button
          variant="outline"
          className="rounded-xl h-10 border-slate-200 font-semibold gap-2 text-sm w-fit"
          onClick={handleSync}
          disabled={syncing}
        >
          {syncing ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
          {syncing ? "Syncing..." : "Refresh Data"}
        </Button>
      </div>

      {/* Summary Stats with REAL deltas */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {[
          {
            label: "Total Followers",
            value: totalFollowers > 0 ? `+${totalFollowers.toLocaleString()}` : "—",
            icon: UserPlus,
            color: "bg-slate-900 text-white",
            delta: platformDeltas[platforms[0]]?.followers,
          },
          {
            label: "Total Engagement",
            value: totalEngagement > 0 ? totalEngagement.toLocaleString() : "—",
            icon: TrendingUp,
            color: "bg-emerald-50 text-emerald-600",
            delta: null,
          },
          {
            label: "Engagement Rate",
            value: `${growthRate}%`,
            icon: Users,
            color: "bg-blue-50 text-blue-600",
            delta: null,
          },
        ].map((s, i) => (
          <motion.div key={s.label} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}>
            <Card className="border-slate-100 shadow-none rounded-2xl hover:shadow-md transition-all">
              <CardContent className="p-6 flex items-center gap-4">
                <div className={`p-3 rounded-2xl ${s.color}`}>
                  <s.icon className="w-6 h-6" />
                </div>
                <div className="flex-1">
                  <p className="text-2xl font-black text-slate-900">{s.value}</p>
                  <p className="text-xs font-bold text-slate-400 uppercase tracking-widest mt-0.5">{s.label}</p>
                </div>
                {s.delta && (
                  <div className={`flex items-center gap-1 text-xs font-bold px-2 py-1 rounded-lg ${s.delta.up ? "text-emerald-600 bg-emerald-50" : "text-rose-500 bg-rose-50"}`}>
                    {s.delta.up ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                    {s.delta.up ? "+" : ""}{s.delta.pct}%
                  </div>
                )}
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      {/* Follower Growth Chart */}
      <Card className="border-slate-100 shadow-none rounded-3xl overflow-hidden">
        <CardHeader className="p-8 pb-0">
          <CardTitle className="text-base font-semibold text-slate-900">Follower Growth Trend</CardTitle>
          <CardDescription className="text-slate-400">Historical follower count from your connected profiles</CardDescription>
        </CardHeader>
        <CardContent className="p-8">
          {followerChartData.length === 0 ? (
            <div className="flex items-center justify-center h-60 text-slate-300 text-sm flex-col gap-3">
              <Users className="w-8 h-8" />
              <span>Sync profiles to see follower growth over time.</span>
              <Button size="sm" className="rounded-xl" onClick={handleSync} disabled={syncing}>
                {syncing ? "Syncing..." : "Sync Now"}
              </Button>
            </div>
          ) : (
            <div className="h-[320px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={followerChartData}>
                  <defs>
                    <linearGradient id="followersGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#0f172a" stopOpacity={0.08} />
                      <stop offset="95%" stopColor="#0f172a" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                  <XAxis dataKey="date" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={10} />
                  <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} />
                  <Tooltip
                    contentStyle={{ borderRadius: "12px", border: "1px solid #f1f5f9", backgroundColor: "#fff", boxShadow: "0 10px 15px rgba(0,0,0,0.06)" }}
                    itemStyle={{ fontSize: "12px", fontWeight: 600 }}
                  />
                  <Area type="monotone" dataKey="followers" stroke="#0f172a" strokeWidth={2.5} fillOpacity={1} fill="url(#followersGrad)" name="Followers" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Traffic & Engagement Chart */}
      <Card className="border-slate-100 shadow-none rounded-3xl overflow-hidden">
        <CardHeader className="p-8 pb-0">
          <CardTitle className="text-base font-semibold text-slate-900">Traffic & Engagement Trend</CardTitle>
          <CardDescription className="text-slate-400">Real data from your connected project</CardDescription>
        </CardHeader>
        <CardContent className="p-8">
          {trafficChartData.length === 0 ? (
            <div className="flex items-center justify-center h-60 text-slate-300 text-sm">
              No metrics data yet. Complete onboarding to populate charts.
            </div>
          ) : (
            <div className="h-[360px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={trafficChartData}>
                  <defs>
                    <linearGradient id="trafficGrad2" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#0f172a" stopOpacity={0.08} />
                      <stop offset="95%" stopColor="#0f172a" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="engagementGrad2" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#10b981" stopOpacity={0.08} />
                      <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                  <XAxis dataKey="date" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8", fontWeight: 500 }} dy={10} />
                  <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8", fontWeight: 500 }} />
                  <Tooltip
                    contentStyle={{ borderRadius: "12px", border: "1px solid #f1f5f9", backgroundColor: "#fff", boxShadow: "0 10px 15px rgba(0,0,0,0.08)" }}
                    itemStyle={{ fontSize: "12px", fontWeight: 600 }}
                  />
                  <Area type="monotone" dataKey="traffic" stroke="#0f172a" strokeWidth={2.5} fillOpacity={1} fill="url(#trafficGrad2)" name="Traffic" />
                  <Area type="monotone" dataKey="engagement" stroke="#10b981" strokeWidth={2} fillOpacity={1} fill="url(#engagementGrad2)" name="Engagement" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Per-platform breakdown with REAL deltas */}
      {socialMetrics.length > 0 && (
        <div className="space-y-4">
          <div className="space-y-1">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Platform Breakdown</h2>
            <p className="text-xl font-semibold text-slate-900 tracking-tight">Channel Performance</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {socialMetrics.map((s: any, i: number) => {
              const platDelta = platformDeltas[s.platform];
              const followerDelta = platDelta?.followers;
              const reachDelta = platDelta?.reach;
              return (
                <motion.div
                  key={s.id ?? i}
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.06 }}
                >
                  <Card className="border-slate-100 shadow-none hover:shadow-lg hover:shadow-slate-200/30 transition-all rounded-2xl">
                    <CardContent className="p-6">
                      <div className="flex items-center justify-between mb-4">
                        <p className="font-bold text-slate-900">{s.platform}</p>
                        {followerDelta && (
                          <div className={`flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded-lg ${followerDelta.up ? "text-emerald-600 bg-emerald-50" : "text-rose-500 bg-rose-50"}`}>
                            {followerDelta.up ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                            {followerDelta.up ? "+" : ""}{followerDelta.pct}% followers (7d)
                          </div>
                        )}
                      </div>
                      <div className="flex gap-8">
                        <div>
                          <p className="text-xl font-bold text-slate-900">{s.reach?.toLocaleString()}</p>
                          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-0.5">Reach</p>
                        </div>
                        <div>
                          <p className="text-xl font-bold text-emerald-600">{s.engagement_count?.toLocaleString() ?? s.engagement?.toLocaleString()}</p>
                          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-0.5">Engagement</p>
                        </div>
                        <div>
                          <p className="text-xl font-bold text-slate-900">{(s.engagement ?? 0).toFixed(1)}%</p>
                          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-0.5">Eng. Rate</p>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
