"use client";

import dynamic from "next/dynamic";

const MotionDiv = dynamic(
  () => import("framer-motion").then((m) => m.motion.div),
  { ssr: false }
);
import { 
  ArrowUpRight, ArrowDownRight, Activity, Globe, 
  Users, Target, Zap, MousePointer2, TrendingUp,
  BarChart3, Sparkles
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import type { Mode } from "./dashboard-mode-context";

interface GrowthSnapshotProps {
  mode: Mode;
  project: any;
  snapshotData?: {
    websiteStats?: any[];
    socialStats?: any[];
    combinedStats?: any[];
  } | null;
}

// Map icon name strings from API to actual icon components
const iconMap: Record<string, any> = {
  Target, Globe, TrendingUp, MousePointer2,
  Activity, Users, Zap, BarChart3, Sparkles,
};

// Fallback stats shown while API data loads. trend is "flat" (never
// "up") so the change badge renders neutral until real data arrives —
// a placeholder must never wear positive/green trend styling.
const fallbackWebsite = [
  { label: "SEO Health", value: "—", change: "—", trend: "flat", icon: Target },
  { label: "Organic Traffic", value: "—", change: "—", trend: "flat", icon: Globe },
  { label: "Search Impressions", value: "—", change: "—", trend: "flat", icon: TrendingUp },
  { label: "CTR", value: "—", change: "—", trend: "flat", icon: MousePointer2 },
];
const fallbackSocial = [
  { label: "Engagement Rate", value: "—", change: "—", trend: "flat", icon: Activity },
  { label: "Total Followers", value: "—", change: "—", trend: "flat", icon: Users },
  { label: "Audience Reach", value: "—", change: "—", trend: "flat", icon: Zap },
  { label: "Content Score", value: "—", change: "—", trend: "flat", icon: BarChart3 },
];
const fallbackCombined = [
  { label: "Growth Index", value: "—", change: "—", trend: "flat", icon: TrendingUp },
  { label: "Aggregate Reach", value: "—", change: "—", trend: "flat", icon: Zap },
  { label: "Open SEO Issues", value: "—", change: "—", trend: "flat", icon: Target },
  { label: "Ranked Keywords", value: "—", change: "—", trend: "flat", icon: Activity },
];

function mapStats(raw?: any[]) {
  if (!raw || raw.length === 0) return null;
  return raw.map((s: any) => ({
    ...s,
    icon: iconMap[s.icon] ?? Activity,
  }));
}

// parseChangeValue turns "+1.2%", "0.0%", "-3.4%" into a number.
// Returns null for non-numeric labels ("—", "Live", "Healthy", …).
function parseChangeValue(change: unknown): number | null {
  if (typeof change !== "string") return null;
  const cleaned = change.replace(/[+%\s,]/g, "");
  if (cleaned === "" || !/^-?\d*\.?\d+$/.test(cleaned)) return null;
  return parseFloat(cleaned);
}

// parseDisplayNumber reads a tile value ("0", "0.0%", "1.2k") into a
// number. Non-numeric placeholders ("—") return null.
function parseDisplayNumber(value: unknown): number | null {
  if (typeof value !== "string") return null;
  const trimmed = value.trim();
  if (trimmed === "" || trimmed === "—") return null;
  const cleaned = trimmed.replace(/[%,\s]/g, "");
  const m = cleaned.match(/^(-?\d*\.?\d+)([km])?$/i);
  if (!m) return null;
  let n = parseFloat(m[1]);
  if (!Number.isFinite(n)) return null;
  const suffix = (m[2] || "").toLowerCase();
  if (suffix === "k") n *= 1000;
  if (suffix === "m") n *= 1_000_000;
  return n;
}

const TRAFFIC_SIGNAL_LABELS = new Set(["Organic Traffic", "Search Impressions"]);
const SOCIAL_SIGNAL_LABELS = new Set([
  "Engagement Rate",
  "Total Followers",
  "Audience Reach",
]);

function tileHasSignal(stat: any): boolean {
  const n = parseDisplayNumber(stat?.value);
  if (n !== null && n !== 0) return true;
  const pct = parseChangeValue(stat?.change);
  return pct !== null && pct !== 0;
}

// Growth Index is a backend composite. With no traffic, social, or
// health signal the formula still emits a number (~20). Treat that as
// empty and show "—" instead of a fake score.
function hasUnderlyingGrowthSignal(
  project: any,
  snapshotData?: GrowthSnapshotProps["snapshotData"] | null
): boolean {
  if (Number(project?.health_score) > 0) return true;
  const website = snapshotData?.websiteStats ?? [];
  const social = snapshotData?.socialStats ?? [];
  for (const s of website) {
    if (TRAFFIC_SIGNAL_LABELS.has(s.label) && tileHasSignal(s)) return true;
  }
  for (const s of social) {
    if (SOCIAL_SIGNAL_LABELS.has(s.label) && tileHasSignal(s)) return true;
  }
  return false;
}

function honestCombinedStats(
  stats: any[],
  project: any,
  snapshotData?: GrowthSnapshotProps["snapshotData"] | null
): any[] {
  if (hasUnderlyingGrowthSignal(project, snapshotData)) return stats;
  return stats.map((s) =>
    s.label === "Growth Index"
      ? { ...s, value: "—", change: "—", trend: "flat" }
      : s
  );
}

export function GrowthSnapshot({ mode, project, snapshotData }: GrowthSnapshotProps) {
  const websiteStats  = mapStats(snapshotData?.websiteStats)  ?? fallbackWebsite;
  const socialStats   = mapStats(snapshotData?.socialStats)   ?? fallbackSocial;
  const combinedStats = honestCombinedStats(
    mapStats(snapshotData?.combinedStats) ?? fallbackCombined,
    project,
    snapshotData
  );

  // App-wide modes: search → website/SEO KPI set, social → social KPI
  // set, combined → combined KPI set. "website" is only an internal
  // KPI-set name; the real mode system (useDashboardMode) uses
  // "search". No separate UI switcher here — the sidebar ModeSwitcher
  // owns mode changes.
  const stats =
    mode === "search" ? websiteStats :
    mode === "social" ? socialStats :
    combinedStats;

  return (
    <div className="space-y-8">
      <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Performance Metrics</h2>
          <p className="text-2xl font-semibold text-slate-900 tracking-tight">Growth Snapshot</p>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {stats.map((stat: any, i: number) => {
          const pct = parseChangeValue(stat.change);
          // Color and arrows only for a real numeric non-zero delta.
          // Non-numeric labels ("Live", "30d total", "Run audit", …)
          // stay slate regardless of the API trend field.
          const isUp = pct !== null && pct > 0;
          const isDown = pct !== null && pct < 0;
          const changeText =
            pct === 0 && typeof stat.change === "string"
              ? stat.change.replace(/^\+/, "")
              : stat.change;
          return (
            <MotionDiv 
              key={`${mode}-${stat.label}`} 
              initial={{ opacity: 0, y: 8 }} 
              animate={{ opacity: 1, y: 0 }} 
              transition={{ duration: 0.3, delay: i * 0.05 }}
            >
              <Card className="border-slate-200/60 shadow-none hover:shadow-lg hover:shadow-slate-200/30 transition-all group bg-white">
                 <CardContent className="p-6">
                    <div className="flex justify-between items-start mb-6">
                       <div className="p-2.5 rounded-lg bg-slate-50 border border-slate-200/60 text-slate-400 group-hover:text-slate-900 transition-colors">
                          <stat.icon className="w-4 h-4" />
                       </div>
                       <div className={`flex items-center gap-1 text-[11px] font-semibold px-2 py-0.5 rounded-md ${
                         isUp
                           ? "text-emerald-600 bg-emerald-50"
                           : isDown
                           ? "text-rose-600 bg-rose-50"
                           : "text-slate-500 bg-slate-100"
                       }`}>
                          {isUp && <ArrowUpRight className="w-3 h-3" />}
                          {isDown && <ArrowDownRight className="w-3 h-3" />}
                          {changeText}
                       </div>
                    </div>
                    <div className="space-y-1">
                      <p className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">
                        {stat.label}
                        {stat.label === "Growth Index" && (
                          <span className="normal-case tracking-normal font-normal"> (computed)</span>
                        )}
                      </p>
                      <p className="text-3xl font-semibold text-slate-900 tracking-tight tabular-nums">{stat.value}</p>
                    </div>
                 </CardContent>
              </Card>
            </MotionDiv>
          );
        })}
      </div>
    </div>
  );
}
