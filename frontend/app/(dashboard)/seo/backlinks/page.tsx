"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import {
  Link2, ExternalLink, Globe, AlertCircle, TrendingUp,
  Loader2, RefreshCw, Shield, CheckCircle2, XCircle, Star
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";

export default function BacklinksPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);

  const fetchData = async (targetProjectId?: number) => {
    setLoading(true);
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data.data ?? [];
      setProjects(allProjects);

      if (allProjects.length > 0) {
        const selected = targetProjectId
          ? allProjects.find((p: any) => p.id === targetProjectId) || allProjects[allProjects.length - 1]
          : allProjects[allProjects.length - 1];
        setProject(selected);

        const res = await dashboardApi.getBacklinks(selected.id);
        setData(res.data?.data ?? null);
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
      await dashboardApi.syncProject(project.id);
      fetchData(project.id);
    } finally {
      setSyncing(false);
    }
  };

  if (loading) return (
    <div className="flex items-center justify-center h-[60vh]">
      <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
    </div>
  );

  const isEstimated = data?.is_estimated ?? true;
  const isGSC = data?.gsc_connected ?? false;
  const topDomains = data?.top_domains ?? [];

  const daColor = (da: number) => {
    if (da >= 60) return "text-emerald-600 bg-emerald-50";
    if (da >= 30) return "text-blue-600 bg-blue-50";
    return "text-slate-500 bg-slate-50";
  };

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
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Backlink Analysis</p>
        </div>
        <Button
          variant="outline"
          className="rounded-xl h-10 border-slate-200 font-semibold gap-2 text-sm w-fit"
          onClick={handleSync}
          disabled={syncing}
        >
          {syncing ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
          {syncing ? "Refreshing..." : "Refresh Data"}
        </Button>
      </div>

      {/* Estimated Data Banner */}
      {isEstimated && (
        <motion.div
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-amber-50 border border-amber-100 rounded-2xl p-5 flex items-center justify-between gap-4"
        >
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-amber-100 flex items-center justify-center">
              <AlertCircle className="w-4 h-4 text-amber-600" />
            </div>
            <div>
              <p className="text-sm font-bold text-amber-900">Backlink data is estimated</p>
              <p className="text-xs text-amber-700/80 font-medium">{data?.upgrade_message}</p>
            </div>
          </div>
          <Button
            size="sm"
            className="rounded-xl bg-amber-600 hover:bg-amber-700 text-white font-bold text-xs uppercase tracking-wider shrink-0"
            onClick={() => window.location.href = "/integrations"}
          >
            Connect GSC
          </Button>
        </motion.div>
      )}

      {/* Summary Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: "Total Backlinks", value: (data?.total_backlinks ?? 0).toLocaleString(), icon: Link2 },
          { label: "Referring Domains", value: (data?.referring_domains ?? 0).toLocaleString(), icon: Globe },
          { label: "Domain Authority", value: String(data?.domain_authority ?? 0), icon: Star },
          { label: "Do-Follow Links", value: (data?.do_follow ?? 0).toLocaleString(), icon: Shield },
        ].map((s, i) => (
          <motion.div key={s.label} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.06 }}>
            <Card className="border-slate-100 shadow-none rounded-2xl hover:shadow-md transition-all">
              <CardContent className="p-6">
                <div className="w-9 h-9 rounded-xl bg-slate-50 flex items-center justify-center mb-4">
                  <s.icon className="w-5 h-5 text-slate-500" />
                </div>
                <p className="text-2xl font-bold text-slate-900">{s.value}</p>
                <p className="text-[11px] font-bold text-slate-400 uppercase tracking-widest mt-1">{s.label}</p>
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      {/* Do-Follow vs No-Follow breakdown */}
      <Card className="border-slate-100 shadow-none rounded-2xl">
        <CardContent className="p-8">
          <h3 className="text-sm font-bold text-slate-400 uppercase tracking-widest mb-6">Link Type Distribution</h3>
          <div className="space-y-4">
            {[
              { label: "Do-Follow", value: data?.do_follow ?? 0, total: data?.total_backlinks ?? 1, color: "bg-emerald-500" },
              { label: "No-Follow", value: data?.no_follow ?? 0, total: data?.total_backlinks ?? 1, color: "bg-slate-300" },
            ].map((item) => {
              const pct = item.total > 0 ? Math.round((item.value / item.total) * 100) : 0;
              return (
                <div key={item.label} className="space-y-2">
                  <div className="flex justify-between items-center">
                    <p className="text-sm font-semibold text-slate-700">{item.label}</p>
                    <p className="text-sm font-bold text-slate-900">{item.value.toLocaleString()} <span className="text-slate-400 font-medium">({pct}%)</span></p>
                  </div>
                  <div className="h-2.5 bg-slate-100 rounded-full overflow-hidden">
                    <motion.div
                      className={`h-full ${item.color} rounded-full`}
                      initial={{ width: 0 }}
                      animate={{ width: `${pct}%` }}
                      transition={{ duration: 0.8, ease: "easeOut" }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>

      {/* Top Referring Domains */}
      <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
        <CardHeader className="px-8 pt-8 pb-4">
          <CardTitle className="text-base font-semibold">Top Referring Domains</CardTitle>
          <p className="text-xs text-slate-400 mt-1">
            {isGSC ? "Real backlink data from Google Search Console" : "Estimated domain signals — connect GSC for full analysis"}
          </p>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-t border-slate-50">
                  <th className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest px-8 py-3">Domain</th>
                  <th className="text-center text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3">Authority</th>
                  <th className="text-center text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3">Links</th>
                  <th className="text-center text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3">Type</th>
                  <th className="text-right text-[10px] font-bold text-slate-400 uppercase tracking-widest px-8 py-3">Visit</th>
                </tr>
              </thead>
              <tbody>
                {topDomains.map((d: any, i: number) => (
                  <motion.tr
                    key={i}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: i * 0.04 }}
                    className="border-t border-slate-50 hover:bg-slate-50/50 transition-colors"
                  >
                    <td className="px-8 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-lg bg-slate-100 flex items-center justify-center">
                          <Globe className="w-4 h-4 text-slate-400" />
                        </div>
                        <p className="font-semibold text-slate-900">{d.domain}</p>
                      </div>
                    </td>
                    <td className="px-4 py-4 text-center">
                      <span className={`inline-flex items-center justify-center w-10 h-8 rounded-lg text-xs font-bold ${daColor(d.authority)}`}>
                        {d.authority}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-center">
                      <span className="font-semibold text-slate-700">{d.links}</span>
                    </td>
                    <td className="px-4 py-4 text-center">
                      <div className="flex items-center justify-center gap-1">
                        {d.do_follow
                          ? <><CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" /><span className="text-xs font-semibold text-emerald-600">DoFollow</span></>
                          : <><XCircle className="w-3.5 h-3.5 text-slate-400" /><span className="text-xs font-semibold text-slate-400">NoFollow</span></>
                        }
                      </div>
                    </td>
                    <td className="px-8 py-4 text-right">
                      <a
                        href={`https://${d.domain}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs font-bold text-slate-400 hover:text-slate-700 transition-colors"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                      </a>
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Premium Upgrade CTA */}
      <Card className="border-slate-100 shadow-none rounded-2xl bg-gradient-to-br from-slate-900 to-slate-800 text-white overflow-hidden">
        <CardContent className="p-8 relative">
          <div className="absolute top-0 right-0 w-48 h-48 bg-white/5 rounded-full -translate-y-1/2 translate-x-1/2" />
          <div className="relative z-10 flex flex-col md:flex-row md:items-center gap-6">
            <div className="w-14 h-14 rounded-2xl bg-white/10 flex items-center justify-center shrink-0">
              <TrendingUp className="w-7 h-7 text-white" />
            </div>
            <div className="flex-1">
              <p className="font-bold text-lg mb-1">Unlock Full Backlink Database</p>
              <p className="text-slate-300 text-sm">Get access to 40+ trillion backlinks, competitor analysis, link intersect, and toxic link detection.</p>
            </div>
            <Button className="rounded-xl bg-white text-slate-900 hover:bg-slate-100 font-bold px-6 shrink-0">
              Upgrade Plan
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
