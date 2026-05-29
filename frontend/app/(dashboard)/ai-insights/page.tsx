"use client";

import { motion } from "framer-motion";
import { 
  BrainCircuit, Sparkles, Target, Zap, 
  Loader2, ChevronRight, Lightbulb, RefreshCw
} from "lucide-react";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { AIInsightCard } from "@/components/dashboard/ai-insight-card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/components/ui/toaster";

export default function AIInsightsPage() {
  const [insights, setInsights] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [project, setProject] = useState<any>(null);
  const [projects, setProjects] = useState<any[]>([]);
  const [filter, setFilter] = useState<string>("all");

  const fetchData = async (targetProjectId?: number) => {
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data?.data || [];
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
        const iRes = await dashboardApi.getInsights(selected.id);
        const raw = iRes.data?.data || [];
        setInsights(Array.isArray(raw) ? raw.map((i: any) => ({
          ...i,
          description: i.body || i.description,
          color: i.type === 'critical' ? 'rose' : i.type === 'opportunity' ? 'emerald' : i.type === 'warning' ? 'amber' : 'brand',
          actionText: i.type === 'critical' ? 'Fix Issue' : i.type === 'opportunity' ? 'Capitalize' : 'Learn More',
        })) : []);
      }
    } catch (err) {
      console.error("Failed to fetch insights", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleProjectChange = (newProj: any) => {
    setLoading(true);
    fetchData(newProj.id);
  };

  const handleRefresh = async () => {
    if (!project?.id) return;
    setRefreshing(true);
    try {
      const res = await dashboardApi.getInsights(project.id);
      const raw = res.data?.data || [];
      setInsights(Array.isArray(raw) ? raw.map((i: any) => ({
        ...i,
        description: i.body || i.description,
        color: i.type === 'critical' ? 'rose' : i.type === 'opportunity' ? 'emerald' : i.type === 'warning' ? 'amber' : 'brand',
        actionText: i.type === 'critical' ? 'Fix Issue' : i.type === 'opportunity' ? 'Capitalize' : 'Learn More',
      })) : []);
      toast("Insights refreshed successfully.", "success");
    } catch (err) {
      toast("Failed to refresh insights.", "error");
    } finally {
      setRefreshing(false);
    }
  };

  const filteredInsights = insights.filter(i => {
    if (filter === "all") return true;
    return i.type?.toLowerCase() === filter.toLowerCase();
  });

  // Derive predictive opportunities from real insights
  const opportunityInsights = insights.filter(i => i.type === "opportunity").slice(0, 3);
  const warningInsights = insights.filter(i => i.type === "warning").slice(0, 1);
  const criticalInsights = insights.filter(i => i.type === "critical").slice(0, 1);

  // Build predictive cards from real data — fall back only if truly no insights
  const predictiveCards = [
    ...opportunityInsights.map(i => ({
      title: i.title,
      desc: i.body || i.description || "Capitalize on this opportunity to boost growth.",
      icon: Zap,
      color: "emerald",
      type: "opportunity",
    })),
    ...warningInsights.map(i => ({
      title: i.title,
      desc: i.body || i.description || "Monitor this signal closely.",
      icon: Target,
      color: "amber",
      type: "warning",
    })),
    ...criticalInsights.map(i => ({
      title: i.title,
      desc: i.body || i.description || "Address this critical issue.",
      icon: Lightbulb,
      color: "rose",
      type: "critical",
    })),
  ].slice(0, 3);

  // Count by type
  const countByType = {
    all: insights.length,
    critical: insights.filter(i => i.type === "critical").length,
    opportunity: insights.filter(i => i.type === "opportunity").length,
    warning: insights.filter(i => i.type === "warning").length,
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  const colorClasses: Record<string, string> = {
    emerald: "bg-emerald-500/10 text-emerald-600",
    amber: "bg-amber-500/10 text-amber-600",
    rose: "bg-rose-500/10 text-rose-600",
  };

  return (
    <div className="space-y-12 max-w-7xl mx-auto pb-40 pt-4">
      
      <DashboardHeader 
        project={project}
        projects={projects}
        onProjectChange={handleProjectChange}
        onAddSource={() => {}} 
      />

      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Growth Intelligence</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">AI Insights</p>
        </div>
        
        <div className="flex items-center gap-2 flex-wrap">
          {(["all", "critical", "opportunity", "warning"] as const).map((f) => (
            <Button
              key={f}
              variant={filter === f ? "default" : "ghost"}
              onClick={() => setFilter(f)}
              className={`rounded-xl h-10 px-4 text-xs font-semibold capitalize transition-all ${
                filter === f 
                  ? "bg-slate-900 text-white shadow-lg shadow-slate-900/10" 
                  : "text-slate-500 hover:bg-slate-50"
              }`}
            >
              {f}
              {countByType[f] > 0 && (
                <span className={`ml-1.5 px-1.5 py-0.5 rounded text-[9px] font-bold ${
                  filter === f ? 'bg-white/20 text-white' : 'bg-slate-100 text-slate-600'
                }`}>{countByType[f]}</span>
              )}
            </Button>
          ))}

          <Button
            variant="outline"
            onClick={handleRefresh}
            disabled={refreshing}
            className="rounded-xl h-10 px-4 text-xs font-semibold border-slate-200 gap-2 ml-2"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-8">
        {filteredInsights.length > 0 ? (
          filteredInsights.map((insight, i) => (
            <motion.div
              key={insight.id}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.07 }}
            >
              <AIInsightCard insight={insight} />
            </motion.div>
          ))
        ) : (
          <div className="rounded-3xl border-2 border-dashed border-slate-100 flex flex-col items-center justify-center p-24 text-center">
            <div className="w-16 h-16 rounded-2xl bg-slate-50 flex items-center justify-center mb-6">
              <BrainCircuit className="w-8 h-8 text-slate-200" />
            </div>
            <h3 className="text-lg font-semibold text-slate-900 mb-2">No insights discovered yet</h3>
            <p className="text-slate-500 max-w-sm mx-auto text-sm leading-relaxed">
              {filter !== "all"
                ? `No ${filter} insights found. Try switching to "all" or run an SEO/social refresh.`
                : "Our AI engine is analyzing your connected properties. Check back in a few minutes."}
            </p>
            <Button 
              className="mt-8 rounded-xl bg-slate-900 text-white px-8 h-12 font-semibold gap-2"
              onClick={handleRefresh}
              disabled={refreshing}
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
              {refreshing ? "Refreshing..." : "Force Refresh Analysis"}
            </Button>
          </div>
        )}
      </div>

      {/* Predictive Insights — Real data from insights, not dummy */}
      {insights.length > 0 && (
        <div className="pt-8 space-y-8">
          <div className="space-y-1">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Based on Your Data</h2>
            <p className="text-2xl font-semibold text-slate-900 tracking-tight">
              {predictiveCards.length > 0 ? "Key Opportunities & Risks" : "Growth Recommendations"}
            </p>
          </div>
          
          {predictiveCards.length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {predictiveCards.map((card, i) => (
                <div key={i} className="bg-white border border-slate-100 rounded-2xl p-6 hover:bg-white hover:border-slate-200 transition-all group cursor-pointer">
                  <div className={`w-10 h-10 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform ${colorClasses[card.color] || 'bg-slate-100 text-slate-600'}`}>
                    <card.icon className="w-5 h-5" />
                  </div>
                  <Badge className={`mb-3 text-[9px] font-bold uppercase tracking-widest border-0 ${
                    card.type === 'opportunity' ? 'bg-emerald-50 text-emerald-700' :
                    card.type === 'warning' ? 'bg-amber-50 text-amber-700' :
                    'bg-rose-50 text-rose-700'
                  }`}>
                    {card.type}
                  </Badge>
                  <h4 className="font-semibold text-slate-900 mb-2">{card.title}</h4>
                  <p className="text-xs text-slate-500 leading-relaxed font-medium line-clamp-3">{card.desc}</p>
                  <div className="mt-6 flex items-center text-[10px] font-bold uppercase tracking-widest text-brand-600 gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    View Full Insight <ChevronRight className="w-3 h-3" />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="rounded-2xl bg-slate-50 border border-slate-100 p-8 text-center">
              <Sparkles className="w-8 h-8 text-slate-300 mx-auto mb-3" />
              <p className="text-sm text-slate-500 font-medium">More specific predictions will appear after your first full audit cycle.</p>
            </div>
          )}
        </div>
      )}

    </div>
  );
}
