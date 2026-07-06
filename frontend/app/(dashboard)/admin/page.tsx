"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Users, CreditCard, TrendingUp, UserPlus, Activity, Crown, ShieldAlert } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { adminApi, type AdminStats } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";

export default function AdminOverviewPage() {
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    adminApi
      .stats()
      .then((r) => {
        if (!cancelled) setStats(r.data.data);
      })
      .catch(() => {
        // Guard already handled 403/401; just show what we can.
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (loading) {
    return <div className="text-slate-400 py-12 text-center">Loading stats…</div>;
  }
  if (!stats) {
    return <div className="text-slate-500 py-12 text-center">Stats unavailable.</div>;
  }

  const fmtDollars = (cents: number) =>
    cents === 0 ? "$0" : `$${(cents / 100).toLocaleString(undefined, { minimumFractionDigits: 0 })}`;

  // Build the "MRR is real" warning when we have no pro subscribers.
  // Today MRR is computed from active subs; when Stripe is wired, this
  // section will show the real revenue numbers instead.
  const mrrIsStub = stats.mrr_cents === 0 && stats.active_subs > 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <ShieldAlert className="w-6 h-6 text-slate-700" />
          Admin overview
        </h1>
        <p className="text-sm text-slate-500 mt-1">
          Platform health, plan distribution, and recent admin activity.
        </p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Total users" value={stats.total_users.toLocaleString()} icon={Users} />
        <StatCard
          label="New (7d)"
          value={stats.new_users_7d.toLocaleString()}
          icon={UserPlus}
          sub={stats.new_users_30d.toLocaleString() + " in last 30d"}
        />
        <StatCard
          label="Active subs"
          value={stats.active_subs.toLocaleString()}
          icon={CreditCard}
          sub={`${stats.trialing_subs} trialing · ${stats.canceled_subs} canceled`}
        />
        <StatCard
          label="MRR"
          value={fmtDollars(stats.mrr_cents)}
          icon={TrendingUp}
          sub={`ARR proxy: ${fmtDollars(stats.arr_proxy_cents)}`}
          warn={mrrIsStub}
        />
      </div>

      {/* Plan breakdown */}
      <Card>
        <CardHeader>
          <CardTitle>Plan distribution</CardTitle>
          <CardDescription>How many users are on each plan.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {Object.keys(stats.plan_breakdown).length === 0 && (
              <div className="text-sm text-slate-500">No data yet.</div>
            )}
            {Object.entries(stats.plan_breakdown)
              .sort(([, a], [, b]) => b - a)
              .map(([code, count]) => {
                const pct = stats.total_users === 0 ? 0 : (count / stats.total_users) * 100;
                return (
                  <div key={code} className="flex items-center gap-3">
                    <div className="w-24 text-sm font-medium text-slate-700">{code}</div>
                    <div className="flex-1 h-2 rounded-full bg-slate-100 overflow-hidden">
                      <div
                        className="h-full bg-slate-900"
                        style={{ width: `${Math.max(2, pct)}%` }}
                      />
                    </div>
                    <div className="w-20 text-right text-sm text-slate-700 tabular-nums">{count}</div>
                  </div>
                );
              })}
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* User growth */}
        <Card>
          <CardHeader>
            <CardTitle>User growth (30d)</CardTitle>
            <CardDescription>New signups per day, last 30 days.</CardDescription>
          </CardHeader>
          <CardContent>
            {stats.user_growth_30d.length === 0 ? (
              <div className="text-sm text-slate-500">No signups in the last 30 days.</div>
            ) : (
              <div className="flex items-end gap-1 h-32">
                {stats.user_growth_30d.map((g) => {
                  const max = Math.max(...stats.user_growth_30d.map((x) => x.count), 1);
                  return (
                    <div
                      key={g.date}
                      title={`${g.date}: ${g.count}`}
                      className="flex-1 bg-slate-700 rounded-t"
                      style={{ height: `${(g.count / max) * 100}%`, minHeight: "2px" }}
                    />
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent activity */}
        <Card>
          <CardHeader>
            <CardTitle>Recent activity</CardTitle>
            <CardDescription>Last 20 admin actions.</CardDescription>
          </CardHeader>
          <CardContent>
            {stats.recent_activity.length === 0 ? (
              <div className="text-sm text-slate-500">No admin actions yet.</div>
            ) : (
              <ul className="space-y-2 max-h-80 overflow-auto">
                {stats.recent_activity.map((a) => (
                  <li key={a.id} className="text-xs flex items-start gap-2">
                    <Badge variant="outline" className="text-[10px]">{a.action}</Badge>
                    <span className="text-slate-500 flex-1">
                      actor {a.actor_user_id} → target {a.target_user_id}
                    </span>
                    <span className="text-slate-400 tabular-nums">
                      {formatDate(a.created_at)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="text-xs text-slate-400">
        Revenue figures are computed from the plan table; they will
        switch to Stripe-sourced numbers once the real billing
        integration lands.
      </div>
    </div>
  );
}

function StatCard({
  label,
  value,
  icon: Icon,
  sub,
  warn,
}: {
  label: string;
  value: string;
  icon: React.ComponentType<{ className?: string }>;
  sub?: string;
  warn?: boolean;
}) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center gap-2 text-slate-500 mb-1">
          <Icon className="w-4 h-4" />
          <span className="text-xs font-medium uppercase tracking-wide">{label}</span>
          {warn && <Badge className="bg-amber-100 text-amber-800 hover:bg-amber-100 text-[10px]">stub</Badge>}
        </div>
        <div className="text-2xl font-bold text-slate-900 tabular-nums">{value}</div>
        {sub && <div className="text-xs text-slate-500 mt-1">{sub}</div>}
      </CardContent>
    </Card>
  );
}
