"use client";

import { useEffect, useState } from "react";
import {
  DollarSign, TrendingUp, CreditCard, Users, ArrowLeft
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { adminApi, type AdminStats } from "@/lib/api-client";
import { useRouter } from "next/navigation";

export default function AdminRevenuePage() {
  const router = useRouter();
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    adminApi.stats()
      .then((r) => { if (!cancelled) setStats(r.data.data); })
      .catch(() => {})
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  if (loading) {
    return <div className="text-slate-400 py-12 text-center">Loading revenue data…</div>;
  }
  if (!stats) {
    return <div className="text-slate-500 py-12 text-center">Revenue data unavailable.</div>;
  }

  const fmtDollars = (cents: number) =>
    cents === 0 ? "$0" : `$${(cents / 100).toLocaleString(undefined, { minimumFractionDigits: 0 })}`;

  // Compute monthly vs yearly revenue breakdown
  const proMonthlyCount = stats.plan_breakdown["pro_monthly"] || 0;
  const proYearlyCount = stats.plan_breakdown["pro_yearly"] || 0;
  const monthlyRevenueCents = proMonthlyCount * 2900;
  const yearlyRevenueCents = proYearlyCount * 29000;
  const monthlyContribution = monthlyRevenueCents;
  const yearlyContribution = Math.round(yearlyRevenueCents / 12);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <button
            onClick={() => router.push("/admin")}
            className="text-xs text-slate-500 hover:text-slate-900 inline-flex items-center gap-1 mb-2"
          >
            <ArrowLeft className="w-3 h-3" /> Back to overview
          </button>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <DollarSign className="w-6 h-6 text-slate-700" />
            Revenue Dashboard
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            MRR, ARR, and subscription metrics.
          </p>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-2 text-slate-500 mb-1">
              <TrendingUp className="w-4 h-4" />
              <span className="text-xs font-medium uppercase tracking-wide">MRR</span>
            </div>
            <div className="text-2xl font-bold text-slate-900 tabular-nums">{fmtDollars(stats.mrr_cents)}</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-2 text-slate-500 mb-1">
              <TrendingUp className="w-4 h-4" />
              <span className="text-xs font-medium uppercase tracking-wide">ARR</span>
            </div>
            <div className="text-2xl font-bold text-slate-900 tabular-nums">{fmtDollars(stats.arr_proxy_cents)}</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-2 text-slate-500 mb-1">
              <CreditCard className="w-4 h-4" />
              <span className="text-xs font-medium uppercase tracking-wide">Active Subs</span>
            </div>
            <div className="text-2xl font-bold text-slate-900 tabular-nums">{stats.active_subs}</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-2 text-slate-500 mb-1">
              <Users className="w-4 h-4" />
              <span className="text-xs font-medium uppercase tracking-wide">Trials</span>
            </div>
            <div className="text-2xl font-bold text-slate-900 tabular-nums">{stats.trialing_subs}</div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* MRR Breakdown */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">MRR Breakdown</CardTitle>
            <CardDescription>Revenue contribution by plan type.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm font-medium text-slate-900">Pro Monthly</div>
                  <div className="text-xs text-slate-500">{proMonthlyCount} subscribers × $29/mo</div>
                </div>
                <div className="text-lg font-bold tabular-nums">{fmtDollars(monthlyContribution)}</div>
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm font-medium text-slate-900">Pro Yearly</div>
                  <div className="text-xs text-slate-500">{proYearlyCount} subscribers × $290/yr (÷12)</div>
                </div>
                <div className="text-lg font-bold tabular-nums">{fmtDollars(yearlyContribution)}</div>
              </div>
              <div className="border-t border-slate-200 pt-3 flex items-center justify-between">
                <div className="text-sm font-semibold text-slate-900">Total MRR</div>
                <div className="text-xl font-bold tabular-nums">{fmtDollars(stats.mrr_cents)}</div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Subscription Status */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Subscription Status</CardTitle>
            <CardDescription>Breakdown of all subscription states.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {[
                { label: "Active", count: stats.active_subs, color: "bg-emerald-500" },
                { label: "Trialing", count: stats.trialing_subs, color: "bg-blue-500" },
                { label: "Canceled", count: stats.canceled_subs, color: "bg-slate-400" },
              ].map((s) => {
                const total = stats.active_subs + stats.trialing_subs + stats.canceled_subs;
                const pct = total === 0 ? 0 : (s.count / total) * 100;
                return (
                  <div key={s.label} className="flex items-center gap-3">
                    <div className={`w-3 h-3 rounded-full ${s.color} shrink-0`} />
                    <div className="w-20 text-sm font-medium text-slate-700">{s.label}</div>
                    <div className="flex-1 h-2 rounded-full bg-slate-100 overflow-hidden">
                      <div
                        className="h-full bg-slate-900 rounded-full"
                        style={{ width: `${Math.max(2, pct)}%` }}
                      />
                    </div>
                    <div className="w-16 text-right text-sm text-slate-700 tabular-nums">{s.count}</div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>

        {/* Plan Distribution */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Plan Distribution</CardTitle>
            <CardDescription>Users per plan code.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
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
                      <div className="w-16 text-right text-sm text-slate-700 tabular-nums">{count}</div>
                    </div>
                  );
                })}
            </div>
          </CardContent>
        </Card>

        {/* Notes */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Notes</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm text-slate-500">
              <p>Revenue figures are computed from the plan table and active subscriptions.</p>
              <p>MRR = (pro_monthly × $29) + (pro_yearly × $290 ÷ 12)</p>
              <p>ARR = MRR × 12</p>
              <p className="text-xs text-slate-400 pt-2">
                These will switch to Stripe-sourced numbers once the real billing integration lands.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
