"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Search, ArrowLeft, Globe, AlertTriangle, TrendingUp, BarChart3
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { adminApi, type PlatformSEOHealth } from "@/lib/api-client";

export default function AdminSEOHealthPage() {
  const [data, setData] = useState<PlatformSEOHealth | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    adminApi.platformSEOHealth()
      .then((r) => { if (!cancelled) setData(r.data.data); })
      .catch(() => {})
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  if (loading) {
    return <div className="text-slate-400 py-12 text-center">Loading SEO health data…</div>;
  }
  if (!data) {
    return <div className="text-slate-500 py-12 text-center">SEO health data unavailable.</div>;
  }

  const healthColor = (bucket: string) => {
    if (bucket.startsWith("90")) return "bg-emerald-500";
    if (bucket.startsWith("70")) return "bg-blue-500";
    if (bucket.startsWith("50")) return "bg-amber-500";
    if (bucket.startsWith("0")) return "bg-red-500";
    return "bg-slate-300";
  };

  const totalProjects = Object.values(data.health_distribution).reduce((a, b) => a + b, 0);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Link
            href="/admin"
            className="text-xs text-slate-500 hover:text-slate-900 inline-flex items-center gap-1 mb-2"
          >
            <ArrowLeft className="w-3 h-3" /> Back to overview
          </Link>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Search className="w-6 h-6 text-slate-700" />
            Platform SEO Health
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            SEO metrics across all projects.
          </p>
        </div>
        <Link href="/admin/projects?goal=seo">
          <Button variant="outline" size="sm">
            <Globe className="w-4 h-4 mr-2" />
            View SEO Projects
          </Button>
        </Link>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="SEO Projects" value={data.total_seo_projects} />
        <StatCard label="Avg Health" value={data.avg_health_score > 0 ? `${Math.round(data.avg_health_score)}/100` : "—"} />
        <StatCard label="Keywords" value={data.total_keywords.toLocaleString()} />
        <StatCard label="Keywords in Top 10" value={data.keywords_in_top_10.toLocaleString()} className="text-emerald-600" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Issues Overview */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-amber-500" />
              Open Issues
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-4 mb-4">
              <div>
                <div className="text-xs text-slate-500">Total Open</div>
                <div className="text-2xl font-bold tabular-nums">{data.total_open_issues}</div>
              </div>
              <div>
                <div className="text-xs text-slate-500">Critical</div>
                <div className="text-2xl font-bold tabular-nums text-red-600">{data.critical_issues}</div>
              </div>
            </div>
            {Object.keys(data.issue_breakdown).length > 0 && (
              <div className="space-y-2">
                <div className="text-xs font-medium text-slate-500 uppercase tracking-wide">By Category</div>
                {Object.entries(data.issue_breakdown)
                  .sort(([, a], [, b]) => b - a)
                  .map(([cat, count]) => {
                    const pct = data.total_open_issues === 0 ? 0 : (count / data.total_open_issues) * 100;
                    return (
                      <div key={cat} className="flex items-center gap-3">
                        <div className="w-28 text-xs font-medium text-slate-700 truncate">{cat}</div>
                        <div className="flex-1 h-1.5 rounded-full bg-slate-100 overflow-hidden">
                          <div className="h-full bg-amber-500 rounded-full" style={{ width: `${Math.max(2, pct)}%` }} />
                        </div>
                        <div className="w-12 text-right text-xs tabular-nums">{count}</div>
                      </div>
                    );
                  })}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Health Distribution */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <BarChart3 className="w-4 h-4 text-slate-500" />
              Health Distribution
            </CardTitle>
            <CardDescription>Projects by health score range.</CardDescription>
          </CardHeader>
          <CardContent>
            {Object.keys(data.health_distribution).length === 0 ? (
              <div className="text-sm text-slate-500">No data yet.</div>
            ) : (
              <div className="space-y-3">
                {["90-100", "70-89", "50-69", "0-49", "unscored"].map((bucket) => {
                  const count = data.health_distribution[bucket] || 0;
                  if (count === 0) return null;
                  const pct = totalProjects === 0 ? 0 : (count / totalProjects) * 100;
                  return (
                    <div key={bucket} className="flex items-center gap-3">
                      <div className={`w-3 h-3 rounded-full ${healthColor(bucket)} shrink-0`} />
                      <div className="w-16 text-sm font-medium text-slate-700">{bucket}</div>
                      <div className="flex-1 h-2 rounded-full bg-slate-100 overflow-hidden">
                        <div className="h-full bg-slate-900 rounded-full" style={{ width: `${Math.max(2, pct)}%` }} />
                      </div>
                      <div className="w-12 text-right text-sm tabular-nums">{count}</div>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
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
