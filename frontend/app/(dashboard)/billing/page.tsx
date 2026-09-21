"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import {
  CreditCard,
  CheckCircle2,
  Loader2,
  XCircle,
  Sparkles,
  Calendar,
  Activity,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/components/ui/toaster";
import {
  billingApi,
  plansApi,
  type BillingState,
  type Plan,
  type Subscription,
} from "@/lib/api-client";

// Pro plan codes we accept on the write endpoints. Free is implicit.
type ProPlan = "pro_monthly" | "pro_yearly";

const PRO_FEATURES = [
  "Up to 3 connected websites",
  "Full AI insights and recommendations",
  "Unlimited audit history",
  "Exportable CSV reports",
  "Priority support",
];

function formatDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function formatPrice(cents: number, cycle: "monthly" | "yearly"): string {
  if (!cents) return "Custom";
  if (cycle === "yearly") return `$${Math.round(cents / 12 / 100)}/mo`;
  return `$${Math.round(cents / 100)}/mo`;
}

export default function BillingPage() {
  const [state, setState] = useState<BillingState | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [cycle, setCycle] = useState<"monthly" | "yearly">("monthly");

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const [me, planList] = await Promise.all([
        billingApi.me(),
        plansApi.list(),
      ]);
      setState(me.data.data);
      setPlans(planList.data.data ?? []);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Failed to load billing state";
      toast(msg, "error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  // The user's current plan + status drive most of the page.
  const currentPlan = state?.plan;
  const currentSub = state?.subscription;
  const isTrialing = currentSub?.status === "trialing";
  const isActive = currentSub?.status === "active" && currentPlan?.code !== "free";
  const isCanceled = currentSub?.status === "canceled";
  const onFree = !currentPlan || currentPlan.code === "free";

  const pro = useMemo(() => {
    if (!plans) return null;
    return plans.find((p) => p.code === (cycle === "monthly" ? "pro_monthly" : "pro_yearly")) ?? null;
  }, [plans, cycle]);

  // ── Action handlers ────────────────────────────────────────────────
  const handleStartTrial = async () => {
    setBusy(true);
    try {
      await billingApi.startTrial(cycle === "monthly" ? "pro_monthly" : "pro_yearly");
      toast("Trial started — 14 days, no charge.", "success");
      await refresh();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Could not start trial";
      toast(msg, "error");
    } finally {
      setBusy(false);
    }
  };

  const handleSubscribe = async () => {
    setBusy(true);
    try {
      await billingApi.subscribe(cycle === "monthly" ? "pro_monthly" : "pro_yearly");
      toast("Plan activated. (Stripe is stubbed — no charge applied.)", "success");
      await refresh();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Could not activate plan";
      toast(msg, "error");
    } finally {
      setBusy(false);
    }
  };

  const handleCancel = async () => {
    if (!confirm("Cancel your subscription? You'll keep access until the end of the current period.")) {
      return;
    }
    setBusy(true);
    try {
      await billingApi.cancel();
      toast("Subscription canceled.", "info");
      await refresh();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Could not cancel";
      toast(msg, "error");
    } finally {
      setBusy(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px] text-slate-400">
        <Loader2 className="w-6 h-6 animate-spin" />
      </div>
    );
  }

  if (!state) {
    return (
      <div className="p-8 text-slate-500">Billing state unavailable. Please refresh.</div>
    );
  }

  return (
    <div className="space-y-8 max-w-6xl">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-slate-900 flex items-center gap-3">
            <CreditCard className="w-7 h-7 text-slate-700" />
            Billing
          </h1>
          <p className="text-slate-500 mt-1">
            Manage your plan, view usage, and review billing history.
          </p>
        </div>
        <Badge variant="outline" className="text-xs">
          Stripe is stubbed · no real charges
        </Badge>
      </div>

      {/* Stub banner — clearly visible while the stub is in place */}
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
        <strong className="font-semibold">Heads up:</strong> Stripe isn&apos;t wired yet.
        The Subscribe button activates the plan at no charge today. When billing is
        live, this same button opens a Stripe Checkout session and the real
        activation moves into the webhook.
      </div>

      {/* Current plan + usage row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              Current plan
              {isTrialing && (
                <Badge className="bg-blue-100 text-blue-800 hover:bg-blue-100">Trialing</Badge>
              )}
              {isActive && (
                <Badge className="bg-emerald-100 text-emerald-800 hover:bg-emerald-100">Active</Badge>
              )}
              {isCanceled && (
                <Badge className="bg-slate-200 text-slate-700 hover:bg-slate-200">Canceled</Badge>
              )}
              {onFree && !isCanceled && (
                <Badge className="bg-slate-100 text-slate-700 hover:bg-slate-100">Free</Badge>
              )}
            </CardTitle>
            <CardDescription>
              {currentPlan?.description ?? "Free plan — get started with one connected website."}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-baseline gap-3">
              <span className="text-5xl font-bold text-slate-900">
                {currentPlan?.code === "free" ? "$0" : `$${Math.round((currentPlan?.monthly_cents ?? 0) / 100)}`}
              </span>
              {currentPlan && currentPlan.code !== "free" && (
                <span className="text-slate-400 font-medium">/ month</span>
              )}
            </div>

            {isTrialing && currentSub?.trial_ends_at && (
              <div className="flex items-center gap-2 text-sm text-slate-600">
                <Calendar className="w-4 h-4 text-slate-400" />
                Trial ends <strong>{formatDate(currentSub.trial_ends_at)}</strong>
              </div>
            )}

            {isCanceled && currentSub?.canceled_at && (
              <div className="flex items-center gap-2 text-sm text-slate-600">
                <XCircle className="w-4 h-4 text-slate-400" />
                Canceled on {formatDate(currentSub.canceled_at)}. You&apos;re back on the Free plan.
              </div>
            )}

            {isActive && (
              <div className="flex items-center gap-2 text-sm text-slate-600">
                <Activity className="w-4 h-4 text-slate-400" />
                Active since {formatDate(currentSub?.starts_at)}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Usage</CardTitle>
            <CardDescription>Connected SEO projects</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-baseline gap-2">
              <span className="text-4xl font-bold text-slate-900">
                {state.usage.seo_projects}
              </span>
              <span className="text-slate-400 font-medium">
                / {state.usage.max_sites}
              </span>
            </div>
            <div className="mt-4 h-2 w-full rounded-full bg-slate-100 overflow-hidden">
              <motion.div
                initial={{ width: 0 }}
                animate={{
                  width: `${Math.min(100, (state.usage.seo_projects / Math.max(1, state.usage.max_sites)) * 100)}%`,
                }}
                transition={{ duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
                className="h-full bg-slate-900"
              />
            </div>
            <p className="mt-3 text-xs text-slate-500">
              Upgrade for more websites, AI insights, and full audit history.
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Upgrade / trial CTA */}
      {(onFree || isCanceled) && pro && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Sparkles className="w-5 h-5 text-amber-500" />
              Try Pro free for 14 days
            </CardTitle>
            <CardDescription>
              Get full access to AI insights, more websites, and exportable reports. Cancel anytime.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <CycleToggle cycle={cycle} onChange={setCycle} disabled={busy} />
            <div className="flex flex-wrap items-center gap-4">
              <div>
                <div className="text-3xl font-bold text-slate-900">
                  {formatPrice(cycle === "monthly" ? pro.monthly_cents : pro.yearly_cents, cycle)}
                </div>
                <div className="text-xs text-slate-500">
                  after trial · cancel anytime
                </div>
              </div>
              <div className="flex gap-3 ml-auto">
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={handleStartTrial}
                  className="rounded-xl"
                >
                  {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
                  Start 14-day trial
                </Button>
                <Button
                  disabled={busy}
                  onClick={handleSubscribe}
                  className="rounded-xl bg-slate-900 hover:bg-slate-800"
                >
                  {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
                  Subscribe now
                </Button>
              </div>
            </div>

            <ul className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-4 border-t border-slate-100">
              {PRO_FEATURES.map((f) => (
                <li key={f} className="flex items-center gap-2 text-sm text-slate-700">
                  <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                  {f}
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {/* Active plan actions */}
      {(isActive || isTrialing) && pro && (
        <Card>
          <CardHeader>
            <CardTitle>Manage subscription</CardTitle>
            <CardDescription>
              {isTrialing
                ? "Your trial is active. Subscribe now to keep access when it ends."
                : `You're on the ${currentPlan?.name ?? "Pro"} plan.`}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <CycleToggle cycle={cycle} onChange={setCycle} disabled={busy} />
            <div className="flex flex-wrap items-center gap-3">
              <div className="text-sm text-slate-600">
                {isTrialing ? "After trial: " : "Switch to: "}
                <strong className="text-slate-900">
                  {formatPrice(cycle === "monthly" ? pro.monthly_cents : pro.yearly_cents, cycle)}
                </strong>
              </div>
              <div className="flex gap-3 ml-auto">
                <Button
                  variant="outline"
                  disabled={busy || isActive}
                  onClick={handleSubscribe}
                  className="rounded-xl"
                >
                  {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
                  {isActive ? "Already active" : "Switch plan"}
                </Button>
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={handleCancel}
                  className="rounded-xl text-rose-600 border-rose-200 hover:bg-rose-50"
                >
                  {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <XCircle className="w-4 h-4" />}
                  Cancel subscription
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="text-center pt-8 text-xs text-slate-400">
        Need help? <Link href="/docs" className="underline">Read the docs</Link> or contact support.
      </div>
    </div>
  );
}

function CycleToggle({
  cycle,
  onChange,
  disabled,
}: {
  cycle: "monthly" | "yearly";
  onChange: (c: "monthly" | "yearly") => void;
  disabled?: boolean;
}) {
  return (
    <div className="inline-flex items-center gap-1 p-1 rounded-full border border-slate-200 bg-white">
      {(["monthly", "yearly"] as const).map((c) => (
        <button
          key={c}
          onClick={() => onChange(c)}
          disabled={disabled}
          className={`px-4 py-1.5 rounded-full text-sm font-bold transition-colors disabled:opacity-50 ${
            cycle === c
              ? "bg-slate-900 text-white"
              : "text-slate-500 hover:text-slate-900"
          }`}
        >
          {c === "monthly" ? "Monthly" : "Yearly"}
        </button>
      ))}
    </div>
  );
}
