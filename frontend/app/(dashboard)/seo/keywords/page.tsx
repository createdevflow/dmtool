"use client";

import { motion } from "framer-motion";
import {
  Search, Zap, Loader2, RefreshCw, Download,
  TrendingUp, TrendingDown, Minus, Database, Globe
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";

const difficultyBadge = (kd: number) => {
  if (kd >= 60) return "border-rose-200 text-rose-600 bg-rose-50";
  if (kd >= 35) return "border-amber-200 text-amber-600 bg-amber-50";
  return "border-emerald-200 text-emerald-600 bg-emerald-50";
};
const difficultyLabel = (kd: number) => kd >= 60 ? "Hard" : kd >= 35 ? "Medium" : "Easy";

const formatVolume = (vol: number) => {
  if (vol >= 1_000_000) return `${(vol / 1_000_000).toFixed(1)}M`;
  if (vol >= 1_000) return `${(vol / 1_000).toFixed(1)}k`;
  return String(vol);
};

export default function KeywordsPage() {
  const [project, setProject] = useState<any>(null);
  const [keywords, setKeywords] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [seed, setSeed] = useState("");
  const [source, setSource] = useState("");
  const [filter, setFilter] = useState<"all" | "easy" | "medium" | "hard">("all");

  useEffect(() => {
    (async () => {
      try {
        const pRes = await dashboardApi.getProjects();
        const projects = pRes.data?.data ?? pRes.data ?? [];
        if (projects.length > 0) {
          const p = projects[projects.length - 1];
          setProject(p);
          const res = await dashboardApi.getKeywords(p.id);
          const data = res.data?.data ?? res.data;
          setKeywords(data?.keywords ?? (Array.isArray(data) ? data : []));
          setSource(data?.source ?? "");
        }
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const handleGenerate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!seed.trim() || !project) return;
    setGenerating(true);
    try {
      const res = await dashboardApi.generateKeywords({
        project_id: project.id,
        seed: seed.trim(),
      });
      const data = res.data?.data ?? res.data;
      const newKeywords = data?.keywords ?? (Array.isArray(data) ? data : []);
      setKeywords((prev) => {
        // Merge, dedup by keyword string
        const existingKeys = new Set(prev.map((k: any) => k.keyword));
        const fresh = newKeywords.filter((k: any) => !existingKeys.has(k.keyword));
        return [...fresh, ...prev];
      });
      setSource(data?.source ?? "autocomplete");
      setSeed("");
    } catch (err) {
      console.error(err);
    } finally {
      setGenerating(false);
    }
  };

  const handleExportCSV = () => {
    if (!keywords.length) return;
    const header = "Keyword,Monthly Volume,Difficulty,Position,Source\n";
    const rows = keywords.map((k: any) =>
      `"${k.keyword}",${k.volume},${k.kd},${k.position > 0 ? k.position.toFixed(1) : "—"},${k.seed ?? ""}`
    ).join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `keywords-${project?.name ?? "export"}.csv`;
    a.click();
  };

  const filtered = keywords.filter((k: any) => {
    if (filter === "easy")   return k.kd < 35;
    if (filter === "medium") return k.kd >= 35 && k.kd < 60;
    if (filter === "hard")   return k.kd >= 60;
    return true;
  });

  return (
    <div className="space-y-8 max-w-6xl mx-auto pb-20">
      <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }}>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-slate-900">Keyword Research</h1>
            <p className="text-slate-500 mt-1">
              Discover search terms that drive organic traffic.
              {source && (
                <span className="ml-2 inline-flex items-center gap-1 text-xs text-slate-400">
                  {source === "gsc" ? (
                    <><Globe className="w-3.5 h-3.5" /> Data from Google Search Console</>
                  ) : source === "cache" ? (
                    <><Database className="w-3.5 h-3.5" /> Loaded from cache</>
                  ) : (
                    <><Zap className="w-3.5 h-3.5" /> From Google autocomplete</>
                  )}
                </span>
              )}
            </p>
          </div>
        </div>
      </motion.div>

      <Card className="border-slate-100 shadow-sm overflow-hidden rounded-3xl">
        <CardContent className="p-0">
          {/* Search bar */}
          <form onSubmit={handleGenerate}
            className="bg-slate-50/50 p-6 flex flex-col md:flex-row gap-4 border-b border-slate-100">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-3 w-5 h-5 text-slate-400" />
              <Input
                placeholder="Enter a seed keyword or topic..."
                className="pl-10 h-11 rounded-xl bg-white border-slate-200 focus:ring-slate-900/5 text-sm"
                value={seed}
                onChange={(e) => setSeed(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={generating || !seed.trim() || !project}
              className="bg-slate-900 hover:bg-slate-800 text-white rounded-xl h-11 px-6 gap-2 disabled:opacity-60 shrink-0">
              {generating
                ? <><RefreshCw className="w-4 h-4 animate-spin" /> Generating...</>
                : <><Zap className="w-4 h-4" /> Generate Keywords</>
              }
            </Button>
          </form>

          {/* Controls row */}
          <div className="p-5 flex flex-wrap items-center justify-between gap-4 border-b border-slate-50">
            <div className="flex gap-2">
              {(["all", "easy", "medium", "hard"] as const).map((f) => (
                <button key={f} onClick={() => setFilter(f)}
                  className={`px-3 py-1.5 rounded-full text-xs font-semibold capitalize transition-all ${
                    filter === f
                      ? "bg-slate-900 text-white"
                      : "bg-slate-100 text-slate-500 hover:bg-slate-200"
                  }`}>
                  {f === "all" ? `All (${keywords.length})` : f}
                </button>
              ))}
            </div>
            <Button variant="outline" size="sm" onClick={handleExportCSV}
              disabled={!keywords.length}
              className="rounded-xl h-9 gap-2 text-slate-600 border-slate-200 text-xs font-semibold">
              <Download className="w-3.5 h-3.5" /> Export CSV
            </Button>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            {loading ? (
              <div className="flex justify-center py-20">
                <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
              </div>
            ) : filtered.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-20 text-center">
                <Search className="w-10 h-10 text-slate-200 mb-4" />
                <p className="text-slate-600 font-semibold">No keywords found</p>
                <p className="text-slate-400 text-sm mt-1 max-w-xs">
                  {source === "gsc"
                    ? "Connect Google Search Console to see your real organic keywords."
                    : "Enter a seed keyword above to generate suggestions from Google."}
                </p>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-slate-400 border-b border-slate-50 bg-slate-50/30">
                    <th className="text-left font-semibold py-4 pl-6 text-[10px] uppercase tracking-widest">Keyword</th>
                    <th className="text-left font-semibold py-4 text-[10px] uppercase tracking-widest">Monthly Volume</th>
                    <th className="text-left font-semibold py-4 text-[10px] uppercase tracking-widest">Difficulty</th>
                    <th className="text-left font-semibold py-4 text-[10px] uppercase tracking-widest">Position</th>
                    <th className="text-left font-semibold py-4 text-[10px] uppercase tracking-widest pr-6">Trend</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-50">
                  {filtered.map((kw: any, i: number) => {
                    const hasPosition = kw.position > 0;
                    const positionColor = kw.position <= 3 ? "text-emerald-600"
                      : kw.position <= 10 ? "text-amber-600" : "text-slate-400";

                    return (
                      <motion.tr key={kw.id ?? i}
                        initial={{ opacity: 0, y: 4 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: Math.min(i * 0.02, 0.6) }}
                        className="group hover:bg-slate-50/60 transition-colors">
                        <td className="py-4 pl-6">
                          <div className="font-semibold text-slate-900">{kw.keyword}</div>
                          {kw.seed && kw.seed !== kw.keyword && (
                            <div className="text-[10px] text-slate-400 mt-0.5">seed: {kw.seed}</div>
                          )}
                        </td>
                        <td className="py-4 font-mono text-slate-700 font-semibold">
                          {kw.volume ? formatVolume(kw.volume) : "—"}
                          {kw.volume > 10000 && (
                            <span className="ml-1.5 text-[10px] text-emerald-500 font-bold">HIGH</span>
                          )}
                        </td>
                        <td className="py-4">
                          <Badge variant="outline" className={`${difficultyBadge(kw.kd)} text-[10px] font-bold`}>
                            {difficultyLabel(kw.kd)} ({kw.kd})
                          </Badge>
                        </td>
                        <td className={`py-4 font-bold ${hasPosition ? positionColor : "text-slate-300"}`}>
                          {hasPosition ? `#${Math.round(kw.position)}` : "—"}
                        </td>
                        <td className="py-4 pr-6">
                          {kw.volume > 10000 ? (
                            <TrendingUp className="w-4 h-4 text-emerald-500" />
                          ) : kw.volume < 500 ? (
                            <TrendingDown className="w-4 h-4 text-slate-300" />
                          ) : (
                            <Minus className="w-4 h-4 text-amber-400" />
                          )}
                        </td>
                      </motion.tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
