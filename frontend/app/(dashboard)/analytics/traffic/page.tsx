"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import {
  TrendingUp, MousePointer2, Eye, Activity, Loader2,
  Link2, RefreshCw, AlertCircle, ChevronUp, ChevronDown,
  Globe, BarChart2, Calendar
} from "lucide-react";
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, BarChart, Bar
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";

function Stat({ label, value, change, trend, icon: Icon, loading }: any) {
  const isUp = trend === "up";
  return (
    <Card className="border-slate-100 shadow-none bg-white rounded-2xl hover:shadow-md transition-all">
      <CardContent className="p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="w-10 h-10 rounded-xl bg-slate-50 flex items-center justify-center">
            <Icon className="w-5 h-5 text-slate-500" />
          </div>
          {change !== "—" && (
            <div className={`flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded-md ${isUp ? "text-emerald-600 bg-emerald-50" : "text-rose-500 bg-rose-50"}`}>
              {isUp ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
              {change}
            </div>
          )}
        </div>
        <p className="text-2xl font-bold text-slate-900 tabular-nums">{loading ? "—" : value}</p>
        <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-widest mt-1">{label}</p>
      </CardContent>
    </Card>
  );
}

export default function TrafficPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [days, setDays] = useState(30);
  const [traffic, setTraffic] = useState<any>(null);
  const [metrics, setMetrics] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [isGSCConnected, setIsGSCConnected] = useState(false);

  const fetchData = async (targetProjectId?: number, d = days) => {
    setLoading(true);
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data.data ?? [];
      setProjects(allProjects);

      if (allProjects.length > 0) {
        let savedProjectId = null;
        try { savedProjectId = parseInt(localStorage.getItem("dmtool_active_project_id") || "0"); } catch (e) {}
        const defaultProject = allProjects.find((p: any) => p.id === savedProjectId) || allProjects[allProjects.length - 1];
        const selected = targetProjectId
          ? allProjects.find((p: any) => p.id === targetProjectId) || defaultProject
          : defaultProject;
        if (selected) localStorage.setItem("dmtool_active_project_id", selected.id.toString());
        setProject(selected);

        const [tRes, mRes, intRes] = await Promise.all([
          dashboardApi.getTraffic(selected.id, d),
          dashboardApi.getMetrics(selected.id, d),
          dashboardApi.getIntegrations(),
        ]);

        setTraffic(tRes.data?.data ?? null);
        const mData = mRes.data?.data;
        setMetrics(Array.isArray(mData) ? mData : []);

        // Check if GSC is connected
        const creds = intRes.data?.data ?? [];
        setIsGSCConnected(creds.some((c: any) => c.provider === "google"));
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  const handleSync = async () => {
    if (!project?.id) return;
    setSyncing(true);
    try {
      const res = await dashboardApi.syncProject(project.id);
      toast("Traffic data refreshed successfully!", "success");
      fetchData(project.id);
    } catch (err: any) {
      toast(err.response?.data?.error?.message || "Sync failed", "error");
    } finally {
      setSyncing(false);
    }
  };

  const handleDaysChange = (val: string) => {
    const d = parseInt(val);
    setDays(d);
    fetchData(project?.id, d);
  };

  // Format chart data
  const chartData = metrics.map((m: any) => ({
    date: new Date(m.date || m.Date).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
    clicks: m.clicks || m.Clicks || 0,
    impressions: m.impressions || m.Impressions || 0,
    ctr: m.impressions > 0 ? ((m.clicks / m.impressions) * 100).toFixed(2) : "0",
  }));

  const summary = traffic?.summary ?? {};
  const hasData = metrics.length > 0;

  const fmtChange = (v: number) => {
    if (!v && v !== 0) return "—";
    return v >= 0 ? `+${v.toFixed(1)}%` : `${v.toFixed(1)}%`;
  };
  const trendDir = (v: number) => (v >= 0 ? "up" : "down");

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
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Web Analytics</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Traffic Dashboard</p>
        </div>
        <div className="flex items-center gap-3">
          <Select value={String(days)} onValueChange={handleDaysChange}>
            <SelectTrigger className="w-36 rounded-xl h-10 border-slate-200 text-sm font-semibold">
              <Calendar className="w-4 h-4 mr-2 text-slate-400" />
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="7">Last 7 days</SelectItem>
              <SelectItem value="30">Last 30 days</SelectItem>
              <SelectItem value="90">Last 90 days</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            className="rounded-xl h-10 border-slate-200 font-semibold gap-2 text-sm"
            onClick={handleSync}
            disabled={syncing}
          >
            {syncing ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
            {syncing ? "Syncing..." : "Sync Now"}
          </Button>
        </div>
      </div>

      {/* GSC Connection CTA */}
      {!isGSCConnected && (
        <motion.div
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-blue-50 border border-blue-100 rounded-2xl p-5 flex items-center justify-between gap-4"
        >
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-blue-100 flex items-center justify-center">
              <Globe className="w-4 h-4 text-blue-600" />
            </div>
            <div>
              <p className="text-sm font-bold text-blue-900">Connect Google Search Console for real traffic data</p>
              <p className="text-xs text-blue-600 font-medium">Currently showing estimated data based on your audit</p>
            </div>
          </div>
          <Button
            size="sm"
            className="rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-bold text-xs uppercase tracking-wider"
            onClick={() => window.location.href = "/integrations"}
          >
            Connect GSC
          </Button>
        </motion.div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Stat
          label="Organic Clicks"
          value={summary.clicks?.toLocaleString() ?? (hasData ? metrics.reduce((s: number, m: any) => s + (m.clicks || 0), 0).toLocaleString() : "0")}
          change={fmtChange(summary.clicks_change)}
          trend={trendDir(summary.clicks_change)}
          icon={MousePointer2}
          loading={loading}
        />
        <Stat
          label="Impressions"
          value={summary.impressions?.toLocaleString() ?? (hasData ? metrics.reduce((s: number, m: any) => s + (m.impressions || 0), 0).toLocaleString() : "0")}
          change={fmtChange(summary.impressions_change)}
          trend={trendDir(summary.impressions_change)}
          icon={Eye}
          loading={loading}
        />
        <Stat
          label="Click-Through Rate"
          value={`${(summary.ctr ?? 0).toFixed(1)}%`}
          change={summary.ctr_change !== undefined ? fmtChange(summary.ctr_change) : "—"}
          trend={trendDir(summary.ctr_change)}
          icon={Activity}
          loading={loading}
        />
        <Stat
          label="SEO Health Score"
          value={`${project?.health_score ?? 0}/100`}
          change={project?.health ? project.health.charAt(0).toUpperCase() + project.health.slice(1) : "—"}
          trend={project?.health_score >= 75 ? "up" : "down"}
          icon={TrendingUp}
          loading={loading}
        />
      </div>

      {/* Traffic Chart */}
      <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
        <CardHeader className="px-8 pt-8 pb-0 flex flex-row items-center justify-between">
          <div>
            <CardTitle className="text-base font-semibold text-slate-900">Clicks & Impressions Over Time</CardTitle>
            <p className="text-xs text-slate-400 font-medium mt-1">
              {isGSCConnected ? "Live data from Google Search Console" : "Estimated based on available data"}
            </p>
          </div>
          {isGSCConnected && (
            <Badge className="bg-emerald-50 text-emerald-700 border-0 text-[10px] font-bold uppercase tracking-wider">
              Live GSC Data
            </Badge>
          )}
        </CardHeader>
        <CardContent className="p-8">
          {!hasData ? (
            <div className="h-72 flex flex-col items-center justify-center text-center gap-4">
              <AlertCircle className="w-10 h-10 text-slate-200" />
              <div>
                <p className="text-slate-500 font-semibold">No traffic data yet</p>
                <p className="text-sm text-slate-400 mt-1">
                  {isGSCConnected
                    ? "GSC data takes 24-48 hours to appear. Try syncing."
                    : "Connect Google Search Console to see real traffic data."}
                </p>
              </div>
              <Button
                size="sm"
                className="rounded-xl bg-slate-900 text-white font-semibold"
                onClick={isGSCConnected ? handleSync : () => window.location.href = "/integrations"}
              >
                {isGSCConnected ? "Sync Data" : "Connect GSC"}
              </Button>
            </div>
          ) : (
            <div className="h-[360px]">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={chartData}>
                  <defs>
                    <linearGradient id="clicksGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#0f172a" stopOpacity={0.08} />
                      <stop offset="95%" stopColor="#0f172a" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="impressionsGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.08} />
                      <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                  <XAxis dataKey="date" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={12} />
                  <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} />
                  <Tooltip
                    contentStyle={{ borderRadius: "12px", border: "1px solid #f1f5f9", backgroundColor: "#fff", boxShadow: "0 10px 25px rgba(0,0,0,0.08)" }}
                    itemStyle={{ fontSize: "12px", fontWeight: 600 }}
                  />
                  <Area type="monotone" dataKey="clicks" name="Clicks" stroke="#0f172a" strokeWidth={2.5} fillOpacity={1} fill="url(#clicksGrad)" />
                  <Area type="monotone" dataKey="impressions" name="Impressions" stroke="#3b82f6" strokeWidth={1.5} strokeDasharray="4 4" fill="url(#impressionsGrad)" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}
        </CardContent>
      </Card>

      {/* CTR Bar Chart */}
      {hasData && (
        <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
          <CardHeader className="px-8 pt-8 pb-0">
            <CardTitle className="text-base font-semibold text-slate-900">Click-Through Rate Trend</CardTitle>
            <p className="text-xs text-slate-400 font-medium mt-1">CTR = Clicks ÷ Impressions × 100</p>
          </CardHeader>
          <CardContent className="p-8">
            <div className="h-[220px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} barSize={12} barGap={2}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                  <XAxis dataKey="date" axisLine={false} tickLine={false} tick={{ fontSize: 10, fill: "#94a3b8" }} dy={10} />
                  <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 10, fill: "#94a3b8" }} unit="%" />
                  <Tooltip
                    contentStyle={{ borderRadius: "12px", border: "1px solid #f1f5f9", backgroundColor: "#fff" }}
                    formatter={(val: any) => [`${val}%`, "CTR"]}
                  />
                  <Bar dataKey="ctr" name="CTR %" fill="#0f172a" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Traffic Breakdown */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {[
          { label: "Organic Search", share: "68%", icon: Globe, color: "bg-emerald-50 text-emerald-600" },
          { label: "Direct", share: "18%", icon: Link2, color: "bg-blue-50 text-blue-600" },
          { label: "Social / Referral", share: "14%", icon: BarChart2, color: "bg-violet-50 text-violet-600" },
        ].map((src) => (
          <Card key={src.label} className="border-slate-100 shadow-none rounded-2xl hover:shadow-md transition-all">
            <CardContent className="p-6 flex items-center gap-4">
              <div className={`w-12 h-12 rounded-xl ${src.color} flex items-center justify-center`}>
                <src.icon className="w-6 h-6" />
              </div>
              <div>
                <p className="text-2xl font-bold text-slate-900">{src.share}</p>
                <p className="text-xs font-semibold text-slate-400 uppercase tracking-widest">{src.label}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
