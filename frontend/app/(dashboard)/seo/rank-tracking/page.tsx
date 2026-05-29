"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import {
  TrendingUp, Search, ChevronUp, ChevronDown, Loader2,
  Globe, AlertCircle, RefreshCw, Eye, BarChart2, Award
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { toast } from "@/components/ui/toaster";

function positionColor(pos: number) {
  if (pos <= 3) return "text-emerald-600 bg-emerald-50";
  if (pos <= 10) return "text-blue-600 bg-blue-50";
  if (pos <= 30) return "text-amber-600 bg-amber-50";
  return "text-slate-500 bg-slate-50";
}

function positionLabel(pos: number) {
  if (pos <= 3) return "Top 3";
  if (pos <= 10) return "Page 1";
  if (pos <= 30) return "Page 2-3";
  return "Page 4+";
}

export default function RankTrackingPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [syncing, setSyncing] = useState(false);

  const fetchData = async (targetProjectId?: number) => {
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

        const res = await dashboardApi.getRankTracking(selected.id);
        setData(res.data?.data ?? null);
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
      toast("Keyword rankings refreshed!", "success");
      fetchData(project.id);
    } catch (err: any) {
      toast("Sync failed. Please try again.", "error");
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

  const keywords = (data?.keywords ?? []).filter((k: any) =>
    search ? k.keyword?.toLowerCase().includes(search.toLowerCase()) : true
  );
  const buckets = data?.buckets ?? { top3: 0, top10: 0, top30: 0, beyond: 0 };
  const visibility = Math.round(data?.visibility ?? 0);
  const isGSC = data?.gsc_connected ?? false;

  return (
    <div className="space-y-10 max-w-7xl mx-auto pb-32 pt-4">
      <DashboardHeader
        project={project}
        projects={projects}
        onProjectChange={(p: any) => fetchData(p.id)}
        onAddSource={() => {}}
      />

      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">SEO Intelligence</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Rank Tracking</p>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            className="rounded-xl h-10 border-slate-200 font-semibold gap-2 text-sm"
            onClick={handleSync}
            disabled={syncing}
          >
            {syncing ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
            {syncing ? "Syncing..." : "Refresh Rankings"}
          </Button>
        </div>
      </div>

      {/* GSC Banner */}
      {!isGSC && (
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
              <p className="text-sm font-bold text-blue-900">Connect Google Search Console for real keyword positions</p>
              <p className="text-xs text-blue-600 font-medium">Currently showing seed keyword data — connect GSC for live positions</p>
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

      {/* Stats Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: "Visibility Score", value: `${visibility}`, icon: Eye, sub: "out of 100" },
          { label: "Top 3 Keywords", value: String(buckets.top3), icon: Award, sub: "ranking positions 1-3" },
          { label: "Page 1 Keywords", value: String(buckets.top10), icon: TrendingUp, sub: "positions 1-10" },
          { label: "Total Tracked", value: String(data?.total ?? 0), icon: Search, sub: "keywords" },
        ].map((s, i) => (
          <motion.div key={s.label} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}>
            <Card className="border-slate-100 shadow-none rounded-2xl hover:shadow-md transition-all">
              <CardContent className="p-6">
                <div className="w-9 h-9 rounded-xl bg-slate-50 flex items-center justify-center mb-4">
                  <s.icon className="w-5 h-5 text-slate-500" />
                </div>
                <p className="text-2xl font-bold text-slate-900">{s.value}</p>
                <p className="text-[11px] font-bold text-slate-400 uppercase tracking-widest mt-1">{s.label}</p>
                <p className="text-[10px] text-slate-300 mt-0.5">{s.sub}</p>
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      {/* Position Buckets Visual */}
      <Card className="border-slate-100 shadow-none rounded-2xl">
        <CardContent className="p-8">
          <h3 className="text-sm font-bold text-slate-400 uppercase tracking-widest mb-6">Position Distribution</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {[
              { label: "Positions 1-3", count: buckets.top3, color: "bg-emerald-500", total: data?.total || 1 },
              { label: "Positions 4-10", count: buckets.top10, color: "bg-blue-500", total: data?.total || 1 },
              { label: "Positions 11-30", count: buckets.top30, color: "bg-amber-400", total: data?.total || 1 },
              { label: "Position 31+", count: buckets.beyond, color: "bg-slate-200", total: data?.total || 1 },
            ].map((b) => (
              <div key={b.label} className="space-y-3">
                <div className="flex justify-between items-center">
                  <p className="text-xs font-semibold text-slate-500">{b.label}</p>
                  <p className="text-sm font-bold text-slate-900">{b.count}</p>
                </div>
                <div className="h-2 bg-slate-100 rounded-full overflow-hidden">
                  <div
                    className={`h-full ${b.color} rounded-full transition-all duration-700`}
                    style={{ width: `${Math.min(100, (b.count / (b.total || 1)) * 100)}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Keywords Table */}
      <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
        <CardHeader className="px-8 pt-8 pb-4 flex flex-row items-center justify-between gap-4">
          <div>
            <CardTitle className="text-base font-semibold">Keyword Rankings</CardTitle>
            <p className="text-xs text-slate-400 mt-1">{isGSC ? "Live positions from Google Search Console" : "Estimated positions — connect GSC for real data"}</p>
          </div>
          <div className="relative w-64">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <Input
              placeholder="Search keywords..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9 h-9 rounded-xl border-slate-200 text-sm"
            />
          </div>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          {keywords.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20">
              <AlertCircle className="w-8 h-8 text-slate-200 mb-3" />
              <p className="text-slate-500 font-semibold text-sm">No keywords found</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-t border-slate-50">
                    <th className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest px-8 py-3">#</th>
                    <th className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3">Keyword</th>
                    <th className="text-center text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3">Position</th>
                    <th className="text-center text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3">Volume</th>
                    <th className="text-center text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3">Difficulty</th>
                    <th className="text-right text-[10px] font-bold text-slate-400 uppercase tracking-widest px-8 py-3">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {keywords.map((kw: any, i: number) => (
                    <motion.tr
                      key={kw.id ?? i}
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      transition={{ delay: i * 0.02 }}
                      className="border-t border-slate-50 hover:bg-slate-50/50 transition-colors"
                    >
                      <td className="px-8 py-4 text-xs font-semibold text-slate-300">{i + 1}</td>
                      <td className="px-4 py-4">
                        <p className="font-semibold text-slate-900">{kw.keyword}</p>
                        <p className="text-xs text-slate-400 mt-0.5">Seed: {kw.seed}</p>
                      </td>
                      <td className="px-4 py-4 text-center">
                        <span className={`inline-flex items-center justify-center w-10 h-8 rounded-lg text-xs font-bold ${positionColor(kw.position)}`}>
                          #{Math.round(kw.position)}
                        </span>
                      </td>
                      <td className="px-4 py-4 text-center">
                        <span className="font-semibold text-slate-700">{kw.volume?.toLocaleString() ?? "—"}</span>
                      </td>
                      <td className="px-4 py-4 text-center">
                        <div className="flex items-center justify-center gap-2">
                          <div className="w-16 h-1.5 bg-slate-100 rounded-full overflow-hidden">
                            <div
                              className={`h-full rounded-full ${kw.kd > 60 ? "bg-rose-400" : kw.kd > 40 ? "bg-amber-400" : "bg-emerald-400"}`}
                              style={{ width: `${kw.kd}%` }}
                            />
                          </div>
                          <span className="text-xs font-semibold text-slate-500">{kw.kd}</span>
                        </div>
                      </td>
                      <td className="px-8 py-4 text-right">
                        <Badge className={`text-[10px] font-bold border-0 ${positionColor(kw.position)}`}>
                          {positionLabel(kw.position)}
                        </Badge>
                      </td>
                    </motion.tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
