"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { Search, Globe, ChevronRight, Filter, X, Activity } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { adminApi, type AdminProjectSummary } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { useSearchParams } from "next/navigation";

export default function AdminProjectsPage() {
  const searchParams = useSearchParams();
  const initialGoal = searchParams.get("goal") || "";

  const [projects, setProjects] = useState<AdminProjectSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [goal, setGoal] = useState(initialGoal);
  const [health, setHealth] = useState("");
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [showFilters, setShowFilters] = useState(false);
  const [stats, setStats] = useState<{
    total_projects: number;
    health_breakdown: Record<string, number>;
    avg_health_score: number;
    new_7d: number;
  } | null>(null);
  const size = 25;

  const fetchProjects = useCallback(async () => {
    setLoading(true);
    try {
      const r = await adminApi.listProjects({
        page,
        size,
        search: search || undefined,
        goal: goal || undefined,
        health: health || undefined,
      });
      setProjects(r.data.data.projects);
      setTotal(r.data.data.total);
    } catch {
      // Guard handles 403.
    } finally {
      setLoading(false);
    }
  }, [page, search, goal, health]);

  const fetchStats = useCallback(async () => {
    try {
      const r = await adminApi.getProjectStats();
      setStats(r.data.data);
    } catch {
      // Non-critical.
    }
  }, []);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const hasActiveFilters = goal || health;
  const clearFilters = () => {
    setGoal("");
    setHealth("");
    setPage(1);
  };

  const healthColor = (h: string) => {
    switch (h) {
      case "healthy": return "bg-emerald-100 text-emerald-800";
      case "issues": return "bg-amber-100 text-amber-800";
      case "scanning": return "bg-blue-100 text-blue-800";
      default: return "bg-slate-100 text-slate-700";
    }
  };

  const goalLabel = (g: string) => {
    switch (g) {
      case "seo": return "SEO";
      case "social": return "Social";
      case "both": return "SEO + Social";
      default: return g;
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <Globe className="w-6 h-6 text-slate-700" />
          All Projects
        </h1>
        <p className="text-sm text-slate-500 mt-1">
          {total.toLocaleString()} total project{total === 1 ? "" : "s"}
        </p>
      </div>

      {/* Stats bar */}
      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatCard label="Total" value={stats.total_projects} />
          <StatCard label="Healthy" value={stats.health_breakdown["healthy"] || 0} className="text-emerald-600" />
          <StatCard label="Issues" value={stats.health_breakdown["issues"] || 0} className="text-amber-600" />
          <StatCard label="Scanning" value={stats.health_breakdown["scanning"] || 0} className="text-blue-600" />
        </div>
      )}

      {/* Search + Filter toggle */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <Input
            placeholder="Search by name or URL"
            value={search}
            onChange={(e) => {
              setPage(1);
              setSearch(e.target.value);
            }}
            className="pl-9"
          />
        </div>
        <Button
          variant={showFilters ? "default" : "outline"}
          size="sm"
          onClick={() => setShowFilters(!showFilters)}
        >
          <Filter className="w-4 h-4 mr-2" />
          Filters
        </Button>
        {hasActiveFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters}>
            <X className="w-4 h-4 mr-1" />
            Clear
          </Button>
        )}
      </div>

      {/* Filter dropdowns */}
      {showFilters && (
        <Card>
          <CardContent className="p-4">
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              <div>
                <label className="text-xs font-medium text-slate-500 uppercase tracking-wide">Goal</label>
                <select
                  value={goal}
                  onChange={(e) => { setGoal(e.target.value); setPage(1); }}
                  className="mt-1 block w-full rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm"
                >
                  <option value="">All goals</option>
                  <option value="seo">SEO</option>
                  <option value="social">Social</option>
                  <option value="both">SEO + Social</option>
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-slate-500 uppercase tracking-wide">Health</label>
                <select
                  value={health}
                  onChange={(e) => { setHealth(e.target.value); setPage(1); }}
                  className="mt-1 block w-full rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm"
                >
                  <option value="">All health</option>
                  <option value="healthy">Healthy</option>
                  <option value="issues">Issues</option>
                  <option value="scanning">Scanning</option>
                </select>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Projects table */}
      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-slate-400">Loading…</div>
          ) : projects.length === 0 ? (
            <div className="p-8 text-center text-slate-500">No projects match your filters.</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-slate-500 border-b border-slate-200">
                  <th className="px-4 py-3 font-medium">Name</th>
                  <th className="px-4 py-3 font-medium">Owner</th>
                  <th className="px-4 py-3 font-medium">Goal</th>
                  <th className="px-4 py-3 font-medium">Health</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3 font-medium">Created</th>
                  <th className="px-2 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {projects.map((p) => (
                  <tr key={p.id} className="border-b border-slate-100 hover:bg-slate-50">
                    <td className="px-4 py-3">
                      <div className="font-medium text-slate-900">{p.name}</div>
                      <div className="text-xs text-slate-400 truncate max-w-[200px]">{p.url}</div>
                    </td>
                    <td className="px-4 py-3 text-slate-700">{p.owner_email || "—"}</td>
                    <td className="px-4 py-3">
                      <Badge variant="outline">{goalLabel(p.goal)}</Badge>
                    </td>
                    <td className="px-4 py-3">
                      <Badge className={healthColor(p.health)}>
                        {p.health} {p.health_score > 0 && `(${p.health_score})`}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-slate-700">{p.status}</td>
                    <td className="px-4 py-3 text-slate-500 tabular-nums">{formatDate(p.created_at)}</td>
                    <td className="px-2 py-3">
                      <Link
                        href={`/admin/projects/${p.id}`}
                        className="text-slate-400 hover:text-slate-700 inline-flex"
                      >
                        <ChevronRight className="w-4 h-4" />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {/* Pagination */}
      {total > size && (
        <div className="flex items-center justify-between text-sm text-slate-500">
          <span>
            Showing {(page - 1) * size + 1}–{Math.min(page * size, total)} of {total}
          </span>
          <div className="flex gap-2">
            <button
              disabled={page === 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              className="px-3 py-1 rounded-md border border-slate-200 disabled:opacity-50"
            >
              Prev
            </button>
            <button
              disabled={page * size >= total}
              onClick={() => setPage((p) => p + 1)}
              className="px-3 py-1 rounded-md border border-slate-200 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({ label, value, className }: { label: string; value: number; className?: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="text-xs font-medium text-slate-500 uppercase tracking-wide">{label}</div>
        <div className={`text-2xl font-bold tabular-nums ${className || "text-slate-900"}`}>{value}</div>
      </CardContent>
    </Card>
  );
}
