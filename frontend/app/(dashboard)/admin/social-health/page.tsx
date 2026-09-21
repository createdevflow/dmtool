"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  Share2, ArrowLeft, Globe, TrendingUp, Users, BarChart3
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { adminApi, type PlatformSocialHealth } from "@/lib/api-client";
import { cn } from "@/lib/utils";

export default function AdminSocialHealthPage() {
  const [data, setData] = useState<PlatformSocialHealth | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    adminApi.platformSocialHealth()
      .then((r) => { if (!cancelled) setData(r.data.data); })
      .catch(() => {})
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  if (loading) {
    return <div className="text-slate-400 py-12 text-center">Loading social health data…</div>;
  }
  if (!data) {
    return <div className="text-slate-500 py-12 text-center">Social health data unavailable.</div>;
  }

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
            <Share2 className="w-6 h-6 text-slate-700" />
            Platform Social Health
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            Social media metrics across all projects.
          </p>
        </div>
        <Link href="/admin/projects?goal=social">
          <Button variant="outline" size="sm">
            <Globe className="w-4 h-4 mr-2" />
            View Social Projects
          </Button>
        </Link>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Social Projects" value={data.total_social_projects} />
        <StatCard label="Total Followers" value={data.total_followers.toLocaleString()} />
        <StatCard label="Avg Engagement" value={`${data.avg_engagement_rate.toFixed(1)}%`} />
        <StatCard label="Total Reach" value={data.total_reach.toLocaleString()} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Platform Breakdown */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Platform Breakdown</CardTitle>
            <CardDescription>Projects by social platform.</CardDescription>
          </CardHeader>
          <CardContent>
            {Object.keys(data.platform_breakdown).length === 0 ? (
              <div className="text-sm text-slate-500">No data yet.</div>
            ) : (
              <div className="space-y-3">
                {Object.entries(data.platform_breakdown)
                  .sort(([, a], [, b]) => b - a)
                  .map(([platform, count]) => {
                    const pct = data.total_social_projects === 0 ? 0 : (count / data.total_social_projects) * 100;
                    return (
                      <div key={platform} className="flex items-center gap-3">
                        <div className="w-24 text-sm font-medium text-slate-700 capitalize">{platform}</div>
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

        {/* Status Breakdown */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Growth Status</CardTitle>
            <CardDescription>Projects by growth trend.</CardDescription>
          </CardHeader>
          <CardContent>
            {Object.keys(data.status_breakdown).length === 0 ? (
              <div className="text-sm text-slate-500">No data yet.</div>
            ) : (
              <div className="space-y-3">
                {Object.entries(data.status_breakdown)
                  .sort(([, a], [, b]) => b - a)
                  .map(([status, count]) => {
                    const total = Object.values(data.status_breakdown).reduce((a, b) => a + b, 0);
                    const pct = total === 0 ? 0 : (count / total) * 100;
                    return (
                      <div key={status} className="flex items-center gap-3">
                        <Badge className={cn(
                          "text-[10px]",
                          status === "growing" ? "bg-emerald-100 text-emerald-800" :
                          status === "dropping" ? "bg-red-100 text-red-800" :
                          "bg-slate-100 text-slate-700"
                        )}>
                          {status}
                        </Badge>
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

        {/* Top Projects */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">Top Performing Projects</CardTitle>
            <CardDescription>Projects ranked by follower count.</CardDescription>
          </CardHeader>
          <CardContent>
            {data.top_projects.length === 0 ? (
              <div className="text-sm text-slate-500">No social projects with data yet.</div>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-xs text-slate-500 border-b border-slate-200">
                    <th className="pb-2 font-medium">Project</th>
                    <th className="pb-2 font-medium">Platform</th>
                    <th className="pb-2 font-medium text-right">Followers</th>
                    <th className="pb-2 font-medium text-right">Engagement</th>
                    <th className="pb-2 font-medium text-right">Reach</th>
                  </tr>
                </thead>
                <tbody>
                  {data.top_projects.map((p) => (
                    <tr key={p.project_id} className="border-b border-slate-100">
                      <td className="py-3">
                        <div className="font-medium">{p.name}</div>
                        <div className="text-xs text-slate-400">{p.owner_email}</div>
                      </td>
                      <td className="py-3 capitalize">{p.platform}</td>
                      <td className="py-3 text-right tabular-nums">{p.followers.toLocaleString()}</td>
                      <td className="py-3 text-right tabular-nums">{p.engagement_rate.toFixed(1)}%</td>
                      <td className="py-3 text-right tabular-nums">{p.reach.toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
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
