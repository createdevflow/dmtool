"use client";

import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  BrainCircuit, Zap, AlertTriangle, Info, RefreshCw,
  Loader2, ArrowRight, CheckCircle2, TrendingUp, Sparkles
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { toast } from "@/components/ui/toaster";

const insightConfig: Record<string, { icon: any; color: string; bg: string; border: string; label: string }> = {
  CRITICAL: { icon: AlertTriangle, color: "text-rose-600", bg: "bg-rose-50", border: "border-rose-100", label: "Critical Issue" },
  OPPORTUNITY: { icon: Zap, color: "text-amber-600", bg: "bg-amber-50", border: "border-amber-100", label: "Opportunity" },
  WARNING: { icon: AlertTriangle, color: "text-orange-500", bg: "bg-orange-50", border: "border-orange-100", label: "Warning" },
  INFO: { icon: Info, color: "text-blue-600", bg: "bg-blue-50", border: "border-blue-100", label: "Insight" },
};

const actionUrls: Record<string, string> = {
  CRITICAL: "/seo/site-explorer",
  OPPORTUNITY: "/seo/keywords",
  INFO: "/integrations",
  WARNING: "/seo/site-explorer",
};

export default function SocialInsightsPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [insights, setInsights] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [filter, setFilter] = useState("all");
  const [dismissed, setDismissed] = useState<Set<number>>(new Set());

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

        const res = await dashboardApi.getInsights(selected.id);
        const raw = res.data?.data ?? [];
        // Normalize type fields to uppercase
        const normalized = raw.map((i: any) => ({
          ...i,
          type: (i.type || "INFO").toUpperCase(),
          description: i.description || i.body || i.Body || "",
        }));
        setInsights(normalized);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateNew = async () => {
    if (!project?.id) return;
    setGenerating(true);
    try {
      // Sync first to get fresh data, then re-fetch insights
      await dashboardApi.syncProject(project.id);
      await fetchData(project.id);
      toast("AI insights refreshed with latest data!", "success");
    } catch (err: any) {
      toast("Could not generate insights. Please try again.", "error");
    } finally {
      setGenerating(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  const filteredInsights = insights
    .filter((i) => !dismissed.has(i.id))
    .filter((i) => filter === "all" || i.type === filter);

  const criticalCount = insights.filter((i) => i.type === "CRITICAL" && !dismissed.has(i.id)).length;
  const oppCount = insights.filter((i) => i.type === "OPPORTUNITY" && !dismissed.has(i.id)).length;

  if (loading) return (
    <div className="flex items-center justify-center h-[60vh]">
      <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
    </div>
  );

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
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">AI Intelligence</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Actionable Insights</p>
        </div>
        <Button
          className="rounded-xl h-10 bg-slate-900 hover:bg-slate-800 text-white font-semibold gap-2 text-sm w-fit"
          onClick={handleGenerateNew}
          disabled={generating}
        >
          {generating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
          {generating ? "Generating..." : "Regenerate Insights"}
        </Button>
      </div>

      {/* Hero Banner */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="rounded-2xl bg-gradient-to-r from-brand-600 to-indigo-600 p-8 text-white relative overflow-hidden shadow-xl shadow-brand-500/20"
      >
        <div className="absolute top-0 right-0 w-72 h-72 bg-white/10 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2" />
        <div className="relative z-10 flex flex-col md:flex-row md:items-center gap-8">
          <div className="w-14 h-14 rounded-2xl bg-white/20 backdrop-blur-xl flex items-center justify-center border border-white/20">
            <BrainCircuit className="w-7 h-7 text-white" />
          </div>
          <div className="flex-1">
            <h2 className="text-xl font-bold mb-1">AI Strategy Engine</h2>
            <p className="text-brand-100 font-medium">
              {criticalCount > 0
                ? `${criticalCount} critical issue${criticalCount > 1 ? "s" : ""} need immediate attention. ${oppCount} growth opportunities identified.`
                : `${insights.length} insights generated from your real performance data. ${oppCount} growth opportunities ready to action.`}
            </p>
          </div>
          <div className="flex gap-3 shrink-0">
            <div className="text-center px-5 py-3 bg-white/20 rounded-xl">
              <p className="text-xl font-bold">{criticalCount}</p>
              <p className="text-xs text-white/70 font-semibold mt-0.5">Critical</p>
            </div>
            <div className="text-center px-5 py-3 bg-white/20 rounded-xl">
              <p className="text-xl font-bold">{oppCount}</p>
              <p className="text-xs text-white/70 font-semibold mt-0.5">Opportunities</p>
            </div>
          </div>
        </div>
      </motion.div>

      {/* Filter Tabs */}
      <div className="flex items-center gap-2">
        {["all", "CRITICAL", "OPPORTUNITY", "INFO"].map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-4 py-2 rounded-xl text-xs font-bold uppercase tracking-wider transition-all border ${
              filter === f
                ? "bg-slate-900 text-white border-slate-900"
                : "bg-white text-slate-500 border-slate-200 hover:border-slate-300"
            }`}
          >
            {f === "all" ? `All (${insights.length})` : f}
          </button>
        ))}
      </div>

      {/* Insights Grid */}
      <AnimatePresence mode="popLayout">
        {filteredInsights.length === 0 ? (
          <motion.div
            key="empty"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="rounded-2xl border-2 border-dashed border-slate-100 flex flex-col items-center justify-center py-24 text-center"
          >
            <CheckCircle2 className="w-10 h-10 text-slate-200 mb-3" />
            <p className="text-slate-500 font-semibold">
              {filter === "all" ? "No insights yet — click Regenerate to generate fresh insights." : `No ${filter.toLowerCase()} insights found.`}
            </p>
          </motion.div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
            {filteredInsights.map((insight, i) => {
              const cfg = insightConfig[insight.type] ?? insightConfig.INFO;
              const Icon = cfg.icon;
              return (
                <motion.div
                  key={insight.id ?? i}
                  layout
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.95 }}
                  transition={{ delay: i * 0.04 }}
                >
                  <Card className={`h-full border ${cfg.border} shadow-none rounded-2xl overflow-hidden hover:shadow-lg hover:shadow-slate-200/40 transition-all group`}>
                    <div className={`h-1 w-full ${cfg.bg} border-b ${cfg.border}`} />
                    <CardContent className="p-6 flex flex-col h-full">
                      <div className="flex items-start justify-between mb-5">
                        <div className={`w-9 h-9 rounded-xl ${cfg.bg} flex items-center justify-center`}>
                          <Icon className={`w-4 h-4 ${cfg.color}`} />
                        </div>
                        <span className={`text-[10px] font-black uppercase tracking-widest px-2 py-1 rounded-lg ${cfg.bg} ${cfg.color}`}>
                          {cfg.label}
                        </span>
                      </div>

                      <h3 className={`text-base font-bold text-slate-900 mb-3 group-hover:${cfg.color} transition-colors`}>
                        {insight.title}
                      </h3>
                      <p className="text-sm text-slate-500 font-medium leading-relaxed flex-1">
                        {insight.description}
                      </p>

                      <div className="flex items-center justify-between mt-6 pt-4 border-t border-slate-50">
                        <span className="text-[10px] font-bold text-slate-300 uppercase tracking-widest">
                          Priority #{insight.priority ?? i + 1}
                        </span>
                        <div className="flex items-center gap-3">
                          <button
                            onClick={() => setDismissed(prev => new Set([...prev, insight.id]))}
                            className="text-xs text-slate-300 hover:text-slate-500 font-semibold transition-colors"
                          >
                            Dismiss
                          </button>
                          <a
                            href={actionUrls[insight.type] ?? "#"}
                            className={`flex items-center gap-1.5 text-xs font-bold ${cfg.color} hover:gap-2.5 transition-all`}
                          >
                            Take Action <ArrowRight className="w-3 h-3" />
                          </a>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              );
            })}
          </div>
        )}
      </AnimatePresence>
    </div>
  );
}