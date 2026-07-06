"use client";

import { useEffect, useMemo, useState } from "react";
import { CheckCircle2, ArrowRight, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { motion } from "framer-motion";
import { plansApi, type Plan } from "@/lib/api-client";

type Cycle = "monthly" | "yearly";

// featureForPlan turns a plan row's limits into a stable list of
// human-readable bullets for the card. Today only the site cap is a
// concrete limit; richer feature lists (AI, export, etc.) live on the
// plans row's `features` JSON blob in the backend and can be wired in
// later without UI churn — the card will grow with the data.
function featuresForPlan(plan: Plan): string[] {
  const out: string[] = [];
  if (plan.code === "free") {
    out.push(`Connect up to ${plan.max_sites} website`);
    out.push("AI-powered insights (limited)");
    out.push("Audit history (30 days)");
  } else {
    out.push(`Connect up to ${plan.max_sites} websites`);
    out.push("Full AI insights and recommendations");
    out.push("Full audit history");
    out.push("Exportable reports");
  }
  return out;
}

// priceForCycle renders a plan's price for the given billing cycle. The
// backend stores monthly_cents and yearly_cents separately; the UI
// picks one and formats it. A 0 in either field renders as "Custom".
function priceForCycle(plan: Plan, cycle: Cycle): { display: string; suffix: string } {
  if (plan.code === "free") {
    return { display: "$0", suffix: "forever" };
  }
  const cents = cycle === "monthly" ? plan.monthly_cents : plan.yearly_cents;
  if (!cents) {
    return { display: "Custom", suffix: "" };
  }
  if (cycle === "yearly") {
    // Show the per-month equivalent of the yearly price — the standard
    // "billed annually" framing so users can compare with monthly.
    const perMonth = Math.round(cents / 12 / 100);
    return { display: `$${perMonth}`, suffix: "/ mo, billed yearly" };
  }
  const dollars = Math.round(cents / 100);
  return { display: `$${dollars}`, suffix: "/ month" };
}

export default function PricingPage() {
  const [cycle, setCycle] = useState<Cycle>("monthly");
  const [plans, setPlans] = useState<Plan[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    plansApi
      .list()
      .then((res) => {
        if (cancelled) return;
        setPlans(res.data?.data ?? []);
      })
      .catch((e) => {
        if (cancelled) return;
        setError(e?.message ?? "Failed to load plans");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // The two pro plan rows are displayed as one card with a cycle
  // toggle. The free plan is its own card.
  const free = useMemo(() => plans?.find((p) => p.code === "free") ?? null, [plans]);
  const pro = useMemo(() => {
    if (!plans) return null;
    return plans.find((p) => p.code === (cycle === "monthly" ? "pro_monthly" : "pro_yearly")) ?? null;
  }, [plans, cycle]);

  return (
    <div className="py-48 px-8 bg-white min-h-screen text-[#1d1d1f] font-sans">
      <div className="max-w-[1400px] mx-auto">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-24 items-end mb-40">
          <div className="lg:col-span-7">
            <span className="text-sm font-bold text-[#0066cc] mb-6 block">Subscription</span>
            <h1 className="text-7xl sm:text-[9rem] font-bold tracking-tight leading-[0.8] text-[#1d1d1f]">
              Simple <br />
              <span className="text-gray-300">Scale.</span>
            </h1>
          </div>
          <div className="lg:col-span-4 lg:col-start-9 pb-4">
            <p className="text-xl text-gray-400 font-medium leading-relaxed">
              Transparent access to the world&apos;s most sophisticated marketing intelligence engine. No hidden tiers, just raw power.
            </p>
          </div>
        </div>

        {/* Cycle toggle — Monthly / Yearly. Hidden until plans load so
            the toggle doesn't flash "no plan" while fetching. */}
        {plans && (
          <div className="flex justify-center mb-16">
            <div className="inline-flex items-center gap-1 p-1 rounded-full border border-gray-100 bg-[#fcfcfd]">
              {(["monthly", "yearly"] as const).map((c) => (
                <button
                  key={c}
                  onClick={() => setCycle(c)}
                  className={`px-6 py-2 rounded-full text-sm font-bold transition-colors ${
                    cycle === c
                      ? "bg-[#1d1d1f] text-white"
                      : "text-gray-500 hover:text-[#1d1d1f]"
                  }`}
                >
                  {c === "monthly" ? "Monthly" : "Yearly"}
                </button>
              ))}
            </div>
          </div>
        )}

        {error && (
          <div className="mb-8 p-4 rounded-2xl bg-red-50 border border-red-100 text-red-700 text-sm font-medium max-w-2xl mx-auto">
            Couldn&apos;t load plans: {error}. <button onClick={() => location.reload()} className="underline">Retry</button>
          </div>
        )}

        {!plans && !error && (
          <div className="flex justify-center items-center py-24 text-gray-400">
            <Loader2 className="w-6 h-6 animate-spin" />
          </div>
        )}

        {plans && free && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-12">
            <PricingCard plan={free} cycle={cycle} index={0} />
            {pro && <PricingCard plan={pro} cycle={cycle} index={1} featured />}
          </div>
        )}
      </div>
    </div>
  );
}

function PricingCard({
  plan,
  cycle,
  index,
  featured = false,
}: {
  plan: Plan;
  cycle: Cycle;
  index: number;
  featured?: boolean;
}) {
  const price = priceForCycle(plan, cycle);
  const features = featuresForPlan(plan);
  const cta = plan.code === "free" ? "Start Free" : "Start Pro";

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.1, duration: 1, ease: [0.16, 1, 0.3, 1] }}
      className={`relative p-16 rounded-[4rem] flex flex-col justify-between min-h-[600px] transition-all ${
        featured
          ? "bg-[#1d1d1f] text-white shadow-3xl shadow-gray-200"
          : "bg-[#fcfcfd] border border-gray-50"
      }`}
    >
      {featured && (
        <div className="absolute top-12 right-12 text-xs font-bold bg-blue-600 px-4 py-1.5 rounded-full">
          Recommended
        </div>
      )}

      <div>
        <h2 className="text-4xl font-bold mb-6">{plan.name}</h2>
        <p
          className={`text-lg font-medium mb-12 leading-relaxed ${
            featured ? "text-gray-400" : "text-gray-500"
          }`}
        >
          {plan.description}
        </p>

        <div className="flex items-baseline gap-4 mb-16">
          <span className="text-8xl font-bold tracking-tight">{price.display}</span>
          {price.suffix && (
            <span className={`text-sm font-bold ${featured ? "text-gray-600" : "text-gray-300"}`}>
              {price.suffix}
            </span>
          )}
        </div>

        <ul className="space-y-6">
          {features.map((f) => (
            <li key={f} className="flex items-center gap-4 text-sm font-semibold">
              <CheckCircle2 className={`w-4 h-4 ${featured ? "text-blue-500" : "text-gray-300"}`} />
              {f}
            </li>
          ))}
        </ul>
      </div>

      <Link href={plan.code === "free" ? "/register" : "/register?plan=pro"} className="mt-20">
        <Button
          className={`w-full h-20 rounded-[2rem] font-bold text-xl tracking-tight transition-transform hover:scale-[0.98] ${
            featured
              ? "bg-blue-600 hover:bg-blue-500 text-white shadow-xl shadow-blue-500/20"
              : "bg-white text-[#1d1d1f] border border-gray-100 hover:bg-gray-50"
          }`}
        >
          {cta} <ArrowRight className="ml-2 w-6 h-6" />
        </Button>
      </Link>
    </motion.div>
  );
}
