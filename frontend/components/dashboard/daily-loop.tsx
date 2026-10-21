"use client";

import Link from "next/link";
import {
  CheckCircle2, Circle, ChevronRight, Globe, Sparkles, Link as LinkIcon,
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

function fmtNum(n: unknown): string {
  if (n === null || n === undefined || n === "") return "—";
  const v = Number(n);
  if (!Number.isFinite(v)) return "—";
  if (Math.abs(v) >= 1_000_000) return `${(v / 1_000_000).toFixed(1)}M`;
  if (Math.abs(v) >= 1_000) return `${(v / 1_000).toFixed(1)}k`;
  return Number.isInteger(v) ? v.toLocaleString() : v.toFixed(1);
}

function lookupValue(lookup: any, key: string): string {
  if (!lookup?.available) return "—";
  return fmtNum(lookup[key]);
}

type LoopStep = {
  id: string;
  title: string;
  href: string;
  done: boolean;
  detail?: string;
};

export function DailyLoop({ snapshot }: { snapshot?: any }) {
  const loop: LoopStep[] = Array.isArray(snapshot?.today_loop) ? snapshot.today_loop : [];
  const brief = snapshot?.vendor_brief ?? null;
  const domain = brief?.domain_explorer ?? {};
  const ai = brief?.ai_visibility ?? {};
  const backlinks = brief?.backlink_explorer ?? {};

  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
      <div className="lg:col-span-7 space-y-6">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Daily loop</h2>
          <p className="text-2xl font-semibold text-slate-900 tracking-tight">Today</p>
        </div>
        <Card className="border-slate-200/60 shadow-none rounded-2xl bg-white">
          <CardContent className="p-2">
            {loop.length === 0 && (
              <p className="px-6 py-10 text-sm text-slate-400">Select a project to see today&apos;s loop.</p>
            )}
            {loop.map((step) => (
              <Link
                key={step.id}
                href={step.href || "/dashboard"}
                className="flex items-center gap-4 rounded-xl px-4 py-3.5 hover:bg-slate-50 transition-colors group"
              >
                {step.done ? (
                  <CheckCircle2 className="w-5 h-5 text-emerald-500 shrink-0" />
                ) : (
                  <Circle className="w-5 h-5 text-slate-300 shrink-0" />
                )}
                <div className="min-w-0 flex-1">
                  <p className={`text-sm font-semibold tracking-tight ${step.done ? "text-slate-400" : "text-slate-900"}`}>
                    {step.title}
                  </p>
                  {step.detail && (
                    <p className="text-xs text-slate-400 mt-0.5 truncate">{step.detail}</p>
                  )}
                </div>
                <ChevronRight className="w-4 h-4 text-slate-300 group-hover:text-slate-500 shrink-0" />
              </Link>
            ))}
          </CardContent>
        </Card>
      </div>

      <div className="lg:col-span-5 space-y-6">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Vendor cache</h2>
          <p className="text-2xl font-semibold text-slate-900 tracking-tight">DataForSEO</p>
        </div>
        <p className="text-xs text-slate-400 -mt-3">
          {brief?.message || "Cached DataForSEO lookups. Not Search Console. Overview never calls the vendor."}
        </p>
        <div className="space-y-3">
          <VendorLookupCard
            href={domain.href || "/seo/site-explorer"}
            icon={Globe}
            title="Domain explorer"
            stale={!!domain.stale}
            available={!!domain.available}
            stats={[
              { label: "Est. traffic", value: lookupValue(domain, "organic_etv") },
              { label: "Keywords", value: lookupValue(domain, "organic_count") },
              { label: "Rank", value: lookupValue(domain, "rank") },
            ]}
          />
          <VendorLookupCard
            href={ai.href || "/seo/ai-visibility"}
            icon={Sparkles}
            title="AI visibility"
            stale={!!ai.stale}
            available={!!ai.available}
            stats={[
              { label: "Mentions", value: lookupValue(ai, "mentions") },
              { label: "Citations", value: lookupValue(ai, "citations") },
            ]}
          />
          <VendorLookupCard
            href={backlinks.href || "/seo/backlinks"}
            icon={LinkIcon}
            title="Backlink explorer"
            stale={!!backlinks.stale}
            available={!!backlinks.available}
            stats={[
              { label: "Backlinks", value: lookupValue(backlinks, "backlinks") },
              { label: "Ref. domains", value: lookupValue(backlinks, "referring_domains") },
              { label: "Rank", value: lookupValue(backlinks, "rank") },
            ]}
          />
        </div>
      </div>
    </div>
  );
}

function VendorLookupCard({
  href,
  icon: Icon,
  title,
  stale,
  available,
  stats,
}: {
  href: string;
  icon: typeof Globe;
  title: string;
  stale: boolean;
  available: boolean;
  stats: { label: string; value: string }[];
}) {
  return (
    <Link href={href} className="block group">
      <Card className="border-slate-200/60 shadow-none rounded-2xl bg-white hover:shadow-lg hover:shadow-slate-200/30 transition-all">
        <CardContent className="p-5">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <div className="p-2 rounded-lg bg-slate-50 border border-slate-200/60 text-slate-400 group-hover:text-slate-900">
                <Icon className="w-4 h-4" />
              </div>
              <p className="text-sm font-semibold text-slate-900">{title}</p>
            </div>
            <div className="flex items-center gap-2">
              {stale && (
                <Badge variant="outline" className="text-[10px] font-semibold text-amber-600 border-amber-100 bg-amber-50">
                  Stale
                </Badge>
              )}
              <Badge variant="outline" className="text-[10px] font-semibold text-slate-400 border-slate-100">
                {available ? "Cached" : "Look up"}
              </Badge>
            </div>
          </div>
          <div className={stats.length === 2 ? "grid grid-cols-2 gap-3" : "grid grid-cols-3 gap-3"}>
            {stats.map((s) => (
              <div key={s.label}>
                <p className="text-lg font-semibold text-slate-900 tabular-nums">{s.value}</p>
                <p className="text-[10px] font-medium text-slate-400 uppercase tracking-wider">{s.label}</p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
