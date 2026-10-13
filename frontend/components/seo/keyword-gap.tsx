"use client";

import { useEffect, useState } from "react";
import {
  GitCompare, Globe, Loader2, RefreshCw, Search,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { dashboardApi } from "@/lib/api-client";

function fmtNum(n: unknown): string {
  const v = Number(n);
  if (!Number.isFinite(v)) return "—";
  if (Math.abs(v) >= 1_000_000) return `${(v / 1_000_000).toFixed(1)}M`;
  if (Math.abs(v) >= 1_000) return `${(v / 1_000).toFixed(1)}k`;
  return v.toLocaleString();
}

function fmtCpc(n: unknown): string {
  const v = Number(n);
  if (!Number.isFinite(v)) return "—";
  return `$${v.toFixed(2)}`;
}

function fmtPos(n: unknown): string {
  const v = Number(n);
  if (!Number.isFinite(v) || v <= 0) return "—";
  return String(v);
}

function intentClass(intent: string) {
  switch (intent) {
    case "transactional":
      return "border-rose-200 text-rose-600 bg-rose-50";
    case "commercial":
      return "border-amber-200 text-amber-600 bg-amber-50";
    case "informational":
      return "border-blue-200 text-blue-600 bg-blue-50";
    case "local":
      return "border-emerald-200 text-emerald-600 bg-emerald-50";
    default:
      return "border-slate-200 text-slate-400 bg-slate-50";
  }
}

type GapRow = {
  keyword?: string;
  volume?: number;
  cpc?: number;
  kd?: number;
  target_position?: number;
  competitor_position?: number;
  intent?: string;
};

function GapTable({
  rows,
  emptyLabel,
  showTarget,
}: {
  rows: GapRow[];
  emptyLabel: string;
  showTarget: boolean;
}) {
  if (!rows.length) {
    return <p className="text-sm text-slate-500">{emptyLabel}</p>;
  }
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-[10px] uppercase tracking-widest text-slate-400">
            <th className="text-left py-2">Keyword</th>
            <th className="text-right py-2">Volume</th>
            <th className="text-right py-2">CPC</th>
            <th className="text-right py-2">KD</th>
            {showTarget && <th className="text-right py-2">You</th>}
            <th className="text-right py-2">Them</th>
            <th className="text-left py-2 pl-4">Intent</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.keyword} className="border-t border-slate-50">
              <td className="py-2 font-medium text-slate-900">{row.keyword || "—"}</td>
              <td className="py-2 text-right tabular-nums">{fmtNum(row.volume)}</td>
              <td className="py-2 text-right tabular-nums">{fmtCpc(row.cpc)}</td>
              <td className="py-2 text-right tabular-nums">{row.kd ? row.kd : "—"}</td>
              {showTarget && (
                <td className="py-2 text-right tabular-nums">{fmtPos(row.target_position)}</td>
              )}
              <td className="py-2 text-right tabular-nums">{fmtPos(row.competitor_position)}</td>
              <td className="py-2 pl-4">
                {row.intent ? (
                  <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-semibold ${intentClass(row.intent)}`}>
                    {row.intent}
                  </span>
                ) : "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function KeywordGapPanel({
  projectId,
  defaultDomain,
}: {
  projectId?: number;
  defaultDomain?: string;
}) {
  const [domain, setDomain] = useState(defaultDomain || "");
  const [competitor, setCompetitor] = useState("");
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (defaultDomain && !domain) setDomain(defaultDomain);
  }, [defaultDomain, domain]);

  const lookup = async (refresh = false) => {
    if (!projectId) return;
    setLoading(true);
    setError("");
    try {
      const res = await dashboardApi.getKeywordGap(projectId, {
        domain: domain.trim() || defaultDomain,
        competitor: competitor.trim(),
        ...(refresh ? { refresh: true } : { fetch: true }),
      });
      const payload = res.data?.data ?? res.data;
      setData(payload);
    } catch (err: any) {
      setError(err?.response?.data?.error?.message || "Lookup failed.");
      setData(null);
    } finally {
      setLoading(false);
    }
  };

  const missing: GapRow[] = Array.isArray(data?.missing) ? data.missing : [];
  const overlap: GapRow[] = Array.isArray(data?.overlap) ? data.overlap : [];

  return (
    <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
      <CardHeader className="px-8 pt-8 pb-4">
        <CardTitle className="text-base font-semibold">Keyword gap</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          Keywords a competitor ranks for that you do not, plus overlap where both rank. DataForSEO Labs domain intersection (United States, English) — not Search Console. Lookup spends vendor credits; results cache 24 hours.
        </p>
      </CardHeader>
      <CardContent className="px-8 pb-8 space-y-6">
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            lookup(false);
          }}
        >
          <div className="flex flex-col sm:flex-row gap-3">
            <div className="relative flex-1">
              <Globe className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <Input
                value={domain}
                onChange={(e) => setDomain(e.target.value)}
                placeholder="your domain"
                className="pl-9 h-11 rounded-xl border-slate-200"
              />
            </div>
            <div className="relative flex-1">
              <GitCompare className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <Input
                value={competitor}
                onChange={(e) => setCompetitor(e.target.value)}
                placeholder="competitor.com"
                className="pl-9 h-11 rounded-xl border-slate-200"
              />
            </div>
          </div>
          <div className="flex flex-col sm:flex-row gap-3">
            <Button
              type="submit"
              disabled={loading || !projectId}
              className="rounded-xl h-11 px-5 bg-slate-900 text-white gap-2"
            >
              {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
              Look up
            </Button>
            {data?.available && (
              <Button
                type="button"
                variant="outline"
                disabled={loading}
                onClick={() => lookup(true)}
                className="rounded-xl h-11 px-4 gap-2"
              >
                <RefreshCw className="w-4 h-4" />
                Refresh
              </Button>
            )}
          </div>
        </form>

        {error && <p className="text-sm text-rose-600">{error}</p>}

        {!data && !loading && (
          <p className="text-sm text-slate-500">Enter your domain and a competitor, then look it up. Nothing is fetched until you do.</p>
        )}

        {data && !data.configured && (
          <p className="text-sm text-slate-600">
            {data.message || "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD."}
          </p>
        )}

        {data?.configured && !data.available && (
          <div className="space-y-1">
            <p className="text-sm text-slate-600">{data.message || "No keyword gap data for this pair."}</p>
            {data.missing_error && <p className="text-xs text-amber-600">{data.missing_error}</p>}
            {data.overlap_error && <p className="text-xs text-amber-600">{data.overlap_error}</p>}
          </div>
        )}

        {data?.available && (
          <div className="space-y-6">
            <p className="text-[11px] font-medium text-slate-400">
              {data.target} vs {data.competitor} · {data.location_name || "United States"} · {data.cached ? "cached" : "live"} · {data.message}
            </p>

            <div>
              <h3 className="text-sm font-semibold text-slate-900 mb-1">Missing keywords</h3>
              <p className="text-xs text-slate-400 mb-3">They rank in organic results; you do not. Up to 15, ordered by search volume.</p>
              {data.missing_ok ? (
                <GapTable rows={missing} emptyLabel="No missing keywords in this sample." showTarget={false} />
              ) : (
                <p className="text-sm text-slate-500">{data.missing_error || "Missing-keyword lookup did not succeed."}</p>
              )}
            </div>

            <div>
              <h3 className="text-sm font-semibold text-slate-900 mb-1">Overlap</h3>
              <p className="text-xs text-slate-400 mb-3">Both domains rank. Positions are organic rank group, not GSC.</p>
              {data.overlap_ok ? (
                <GapTable rows={overlap} emptyLabel="No overlapping keywords in this sample." showTarget />
              ) : (
                <p className="text-sm text-slate-500">{data.overlap_error || "Overlap lookup did not succeed."}</p>
              )}
            </div>
            <p className="text-[10px] text-slate-400">Vendor data is not mixed into Overview GSC tiles. Volume, CPC, and KD are Labs estimates.</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
