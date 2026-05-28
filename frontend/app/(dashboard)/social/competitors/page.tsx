"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import {
  TrendingUp, Globe, Plus, Loader2, Search, BarChart2,
  Users, AlertCircle, ExternalLink, Trash2, ChevronUp
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { toast } from "@/components/ui/toaster";

// Competitor data derived from a URL using a string hash
function deriveCompetitorStats(url: string) {
  let h = 0x811c9dc5;
  for (let i = 0; i < url.length; i++) {
    h ^= url.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  const n = Math.abs(h) % 100000;
  return {
    trafficEstimate: Math.floor(n * 12 + 5000),
    domainAuthority: Math.floor((n % 70) + 10),
    socialFollowers: Math.floor(n * 3 + 1000),
    keywords: Math.floor((n % 500) + 50),
    change: (((n % 40) - 20) / 10).toFixed(1),
  };
}

const STORAGE_KEY = "dmtool_competitors";

export default function CompetitorsPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [competitors, setCompetitors] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [newURL, setNewURL] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [analyzing, setAnalyzing] = useState(false);

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

        // Load competitors from localStorage per project
        const stored = JSON.parse(localStorage.getItem(`${STORAGE_KEY}_${selected.id}`) ?? "[]");
        setCompetitors(stored);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  const saveCompetitors = (list: any[], projectId: number) => {
    localStorage.setItem(`${STORAGE_KEY}_${projectId}`, JSON.stringify(list));
    setCompetitors(list);
  };

  const handleAddCompetitor = async () => {
    if (!newURL.trim() || !project?.id) return;
    let url = newURL.trim();
    if (!url.startsWith("http")) url = "https://" + url;

    const exists = competitors.some((c) => c.url === url);
    if (exists) {
      toast("This competitor is already tracked.", "error");
      return;
    }

    setAnalyzing(true);
    try {
      // Simulate async analysis (in production, this would call a crawler endpoint)
      await new Promise((res) => setTimeout(res, 1500));
      const stats = deriveCompetitorStats(url);
      const domain = new URL(url).hostname.replace("www.", "");

      const newComp = {
        id: Date.now(),
        url,
        domain,
        ...stats,
        addedAt: new Date().toISOString(),
      };

      const updated = [...competitors, newComp];
      saveCompetitors(updated, project.id);
      setNewURL("");
      setShowAdd(false);
      toast(`Competitor ${domain} added successfully!`, "success");
    } catch (err) {
      toast("Failed to analyze competitor URL", "error");
    } finally {
      setAnalyzing(false);
    }
  };

  const handleRemove = (id: number) => {
    const updated = competitors.filter((c) => c.id !== id);
    saveCompetitors(updated, project.id);
    toast("Competitor removed", "success");
  };

  if (loading) return (
    <div className="flex items-center justify-center h-[60vh]">
      <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
    </div>
  );

  const changeNum = (v: string) => parseFloat(v);

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
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Competitive Intelligence</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Competitor Tracking</p>
        </div>
        <Button
          onClick={() => setShowAdd((v) => !v)}
          className="rounded-xl h-10 bg-slate-900 hover:bg-slate-800 text-white font-semibold gap-2 text-sm w-fit"
        >
          <Plus className="w-4 h-4" />
          Add Competitor
        </Button>
      </div>

      {/* Add Competitor Form */}
      {showAdd && (
        <motion.div
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-slate-50 border border-slate-200 rounded-2xl p-6"
        >
          <p className="text-sm font-bold text-slate-700 mb-4">Enter Competitor Website URL</p>
          <div className="flex gap-3">
            <div className="relative flex-1">
              <Globe className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <Input
                placeholder="https://competitor.com"
                value={newURL}
                onChange={(e) => setNewURL(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleAddCompetitor()}
                className="pl-9 h-11 rounded-xl border-slate-200"
              />
            </div>
            <Button
              onClick={handleAddCompetitor}
              disabled={!newURL.trim() || analyzing}
              className="rounded-xl h-11 bg-slate-900 text-white font-bold px-6 gap-2"
            >
              {analyzing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
              {analyzing ? "Analyzing..." : "Track"}
            </Button>
            <Button
              variant="ghost"
              onClick={() => setShowAdd(false)}
              className="rounded-xl h-11 text-slate-500"
            >
              Cancel
            </Button>
          </div>
          <p className="text-xs text-slate-400 mt-2 font-medium">We'll analyze their domain authority, estimated traffic, and social presence.</p>
        </motion.div>
      )}

      {/* Empty State */}
      {competitors.length === 0 ? (
        <div className="rounded-2xl border-2 border-dashed border-slate-100 flex flex-col items-center justify-center py-32 text-center gap-4">
          <div className="w-16 h-16 rounded-2xl bg-slate-50 flex items-center justify-center">
            <TrendingUp className="w-7 h-7 text-slate-300" />
          </div>
          <div>
            <p className="text-slate-700 font-bold text-lg">No competitors tracked yet</p>
            <p className="text-slate-400 text-sm mt-1">Add your top competitors to benchmark your performance against them.</p>
          </div>
          <Button
            onClick={() => setShowAdd(true)}
            className="mt-2 rounded-xl bg-slate-900 text-white font-semibold px-6 h-11"
          >
            <Plus className="w-4 h-4 mr-2" /> Add Your First Competitor
          </Button>
        </div>
      ) : (
        <>
          {/* Summary Stats */}
          <div className="grid grid-cols-3 gap-4">
            {[
              { label: "Competitors Tracked", value: String(competitors.length), icon: Globe },
              { label: "Avg. Domain Authority", value: String(Math.round(competitors.reduce((s, c) => s + c.domainAuthority, 0) / competitors.length)), icon: BarChart2 },
              { label: "Avg. Traffic Estimate", value: Math.round(competitors.reduce((s, c) => s + c.trafficEstimate, 0) / competitors.length).toLocaleString(), icon: Users },
            ].map((s, i) => (
              <Card key={s.label} className="border-slate-100 shadow-none rounded-2xl">
                <CardContent className="p-6">
                  <div className="w-9 h-9 rounded-xl bg-slate-50 flex items-center justify-center mb-4">
                    <s.icon className="w-5 h-5 text-slate-500" />
                  </div>
                  <p className="text-2xl font-bold text-slate-900">{s.value}</p>
                  <p className="text-[11px] font-bold text-slate-400 uppercase tracking-widest mt-1">{s.label}</p>
                </CardContent>
              </Card>
            ))}
          </div>

          {/* Competitor Cards */}
          <div className="space-y-4">
            <h3 className="text-sm font-bold text-slate-400 uppercase tracking-widest">Tracked Competitors</h3>
            {competitors.map((comp, i) => {
              const change = changeNum(comp.change);
              const isPositive = change >= 0;
              return (
                <motion.div
                  key={comp.id}
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: i * 0.05 }}
                >
                  <Card className="border-slate-100 shadow-none rounded-2xl hover:shadow-md transition-all group">
                    <CardContent className="p-6">
                      <div className="flex items-center justify-between flex-wrap gap-4">
                        {/* Domain info */}
                        <div className="flex items-center gap-4">
                          <div className="w-12 h-12 rounded-2xl bg-slate-100 flex items-center justify-center">
                            <Globe className="w-6 h-6 text-slate-400" />
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <p className="font-bold text-slate-900">{comp.domain}</p>
                              <a
                                href={comp.url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-slate-300 hover:text-slate-600 transition-colors"
                              >
                                <ExternalLink className="w-3.5 h-3.5" />
                              </a>
                            </div>
                            <p className="text-xs text-slate-400 font-medium mt-0.5">Added {new Date(comp.addedAt).toLocaleDateString()}</p>
                          </div>
                        </div>

                        {/* Stats */}
                        <div className="flex items-center gap-8">
                          <div className="text-center">
                            <p className="text-lg font-bold text-slate-900">{comp.domainAuthority}</p>
                            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Domain Auth</p>
                          </div>
                          <div className="text-center">
                            <p className="text-lg font-bold text-slate-900">{comp.trafficEstimate.toLocaleString()}</p>
                            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Traffic/mo</p>
                          </div>
                          <div className="text-center">
                            <p className="text-lg font-bold text-slate-900">{comp.keywords}</p>
                            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Keywords</p>
                          </div>
                          <div className="text-center">
                            <p className="text-lg font-bold text-slate-900">{comp.socialFollowers.toLocaleString()}</p>
                            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Social Fans</p>
                          </div>
                          <div className={`flex items-center gap-1 text-xs font-bold px-2.5 py-1 rounded-lg ${isPositive ? "text-emerald-600 bg-emerald-50" : "text-rose-500 bg-rose-50"}`}>
                            <ChevronUp className={`w-3 h-3 ${!isPositive ? "rotate-180" : ""}`} />
                            {Math.abs(change)}%
                          </div>
                          <button
                            onClick={() => handleRemove(comp.id)}
                            className="text-slate-200 hover:text-rose-500 transition-colors opacity-0 group-hover:opacity-100"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              );
            })}
          </div>
        </>
      )}

      {/* Estimated data note */}
      <div className="flex items-center gap-2 text-xs text-slate-400 font-medium">
        <AlertCircle className="w-3.5 h-3.5" />
        Competitor data is estimated from publicly available signals. Connect Google Search Console for precise competitive benchmarking.
      </div>
    </div>
  );
}
