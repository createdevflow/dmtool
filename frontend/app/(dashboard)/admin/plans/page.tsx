"use client";

import { useEffect, useState } from "react";
import { CreditCard } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { adminApi, type AdminPlan } from "@/lib/api-client";

export default function AdminPlansPage() {
  const [plans, setPlans] = useState<AdminPlan[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    adminApi
      .listPlans()
      .then((r) => {
        if (cancelled) return;
        setPlans(r.data.data);
      })
      .catch(() => {
        // Guard handles 403/401.
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const fmtCents = (cents: number) =>
    cents === 0 ? "—" : `$${(cents / 100).toFixed(0)}`;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <CreditCard className="w-6 h-6 text-slate-700" />
          Plans
        </h1>
        <p className="text-sm text-slate-500 mt-1">
          {plans.length} plan{plans.length === 1 ? "" : "s"} configured.
          Edit prices and limits; new tiers ship without code changes.
        </p>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-slate-400">Loading…</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-slate-500 border-b border-slate-200">
                  <th className="px-4 py-3 font-medium">Code</th>
                  <th className="px-4 py-3 font-medium">Name</th>
                  <th className="px-4 py-3 font-medium">Tier</th>
                  <th className="px-4 py-3 font-medium">Monthly</th>
                  <th className="px-4 py-3 font-medium">Yearly</th>
                  <th className="px-4 py-3 font-medium">Max sites</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {plans.map((p) => (
                  <tr key={p.id} className="border-b border-slate-100">
                    <td className="px-4 py-3 font-mono text-xs text-slate-900">{p.code}</td>
                    <td className="px-4 py-3 text-slate-700">
                      <div className="font-medium text-slate-900">{p.name}</div>
                      <div className="text-xs text-slate-500">{p.description || "—"}</div>
                    </td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">{p.tier_rank}</td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">{fmtCents(p.monthly_cents)}</td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">{fmtCents(p.yearly_cents)}</td>
                    <td className="px-4 py-3 text-slate-700 tabular-nums">{p.max_sites}</td>
                    <td className="px-4 py-3">
                      <Badge
                        className={
                          p.is_active
                            ? "bg-emerald-100 text-emerald-800 hover:bg-emerald-100"
                            : "bg-slate-200 text-slate-700 hover:bg-slate-200"
                        }
                      >
                        {p.is_active ? "active" : "inactive"}
                      </Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
