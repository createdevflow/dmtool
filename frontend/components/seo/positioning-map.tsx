"use client";

import { useEffect, useState } from "react";
import {
  Globe, Loader2, Map as MapIcon, RefreshCw, Scale, Search,
} from "lucide-react";
import {
  Bar, BarChart, CartesianGrid, ResponsiveContainer, Scatter, ScatterChart,
  Tooltip, XAxis, YAxis, Legend,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { dashboardApi } from "@/lib/api-client";

function fmtNum(n: unknown, ok: boolean): string {
  if (!ok) return "—";
  const v = Number(n);
  if (!Number.isFinite(v)) return "—";
  if (Math.abs(v) >= 1_000_000) return `${(v / 1_000_000).toFixed(1)}M`;
  if (Math.abs(v) >= 1_000) return `${(v / 1_000).toFixed(1)}k`;
  return Number.isInteger(v) ? v.toLocaleString() : v.toFixed(1);
}

function finiteOrNull(n: unknown, ok: boolean): number | null {
  if (!ok) return null;
  const v = Number(n);
  return Number.isFinite(v) ? v : null;
}

type Side = {
  domain?: string;
  organic_ok?: boolean;
  organic_error?: string;
  organic?: {
    etv?: number;
    count?: number;
    pos_1?: number;
    pos_2_3?: number;
    pos_4_10?: number;
  };
  backlinks_ok?: boolean;
  backlinks_error?: string;
  backlinks?: {
    referring_domains?: number;
    rank?: number;
  };
};

type ScatterPoint = {
  name: string;
  keywords: number;
  rank: number;
};

export function PositioningMapPanel({
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
      const res = await dashboardApi.getDomainCompare(projectId, {
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

  const you: Side = data?.target_side ?? {};
  const them: Side = data?.competitor_side ?? {};
  const youLabel = data?.target || "You";
  const themLabel = data?.competitor || "Them";

  const youPoint: ScatterPoint | null =
    you.organic_ok && you.backlinks_ok && Number.isFinite(Number(you.organic?.count)) && Number.isFinite(Number(you.backlinks?.rank))
      ? { name: youLabel, keywords: Number(you.organic?.count), rank: Number(you.backlinks?.rank) }
      : null;
  const themPoint: ScatterPoint | null =
    them.organic_ok && them.backlinks_ok && Number.isFinite(Number(them.organic?.count)) && Number.isFinite(Number(them.backlinks?.rank))
      ? { name: themLabel, keywords: Number(them.organic?.count), rank: Number(them.backlinks?.rank) }
      : null;

  const positionRows = [
    { label: "Position 1", you: finiteOrNull(you.organic?.pos_1, !!you.organic_ok), them: finiteOrNull(them.organic?.pos_1, !!them.organic_ok) },
    { label: "Positions 2–3", you: finiteOrNull(you.organic?.pos_2_3, !!you.organic_ok), them: finiteOrNull(them.organic?.pos_2_3, !!them.organic_ok) },
    { label: "Positions 4–10", you: finiteOrNull(you.organic?.pos_4_10, !!you.organic_ok), them: finiteOrNull(them.organic?.pos_4_10, !!them.organic_ok) },
  ];
  const canPlotPositions = you.organic_ok || them.organic_ok;

  return (
    <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
      <CardHeader className="px-8 pt-8 pb-4">
        <CardTitle className="text-base font-semibold">Competitive positioning</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          Plot of Domain Compare Labs organic keywords vs DataForSEO Rank for two domains (United States, English). Same lookup and 24h cache as Domain Compare — not a new vendor call, not Search Console, not Moz Domain Authority, and not a composite score.
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
              <Scale className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
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
          <p className="text-sm text-slate-500">Enter your domain and a competitor, then look it up. Nothing is fetched until you do. If Domain Compare already ran for this pair, this page reads that 24h cache.</p>
        )}

        {data && !data.configured && (
          <p className="text-sm text-slate-600">
            {data.message || "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD."}
          </p>
        )}

        {data?.configured && !data.available && (
          <div className="space-y-1">
            <p className="text-sm text-slate-600">{data.message || "No comparison data for this pair."}</p>
            {you.organic_error && <p className="text-xs text-amber-600">{you.domain}: {you.organic_error}</p>}
            {you.backlinks_error && <p className="text-xs text-amber-600">{you.domain}: {you.backlinks_error}</p>}
            {them.organic_error && <p className="text-xs text-amber-600">{them.domain}: {them.organic_error}</p>}
            {them.backlinks_error && <p className="text-xs text-amber-600">{them.domain}: {them.backlinks_error}</p>}
          </div>
        )}

        {data?.available && (
          <div className="space-y-8">
            <p className="text-[11px] font-medium text-slate-400">
              {youLabel} vs {themLabel} · {data.location_name || "United States"} · {data.cached ? "cached" : "live"} · {data.message}
            </p>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {[
                { label: "Organic keywords", you: fmtNum(you.organic?.count, !!you.organic_ok), them: fmtNum(them.organic?.count, !!them.organic_ok) },
                { label: "DataForSEO Rank", you: fmtNum(you.backlinks?.rank, !!you.backlinks_ok), them: fmtNum(them.backlinks?.rank, !!them.backlinks_ok) },
                { label: "Referring domains", you: fmtNum(you.backlinks?.referring_domains, !!you.backlinks_ok), them: fmtNum(them.backlinks?.referring_domains, !!them.backlinks_ok) },
                { label: "Moz DA", you: "—", them: "—" },
              ].map((s) => (
                <div key={s.label} className="rounded-xl border border-slate-100 p-4">
                  <p className="text-[11px] font-bold uppercase tracking-widest text-slate-400">{s.label}</p>
                  <p className="text-sm font-semibold text-slate-900 mt-2 tabular-nums">{youLabel}: {s.you}</p>
                  <p className="text-sm font-semibold text-slate-500 tabular-nums">{themLabel}: {s.them}</p>
                </div>
              ))}
            </div>

            {youPoint || themPoint ? (
              <div>
                <h3 className="text-sm font-semibold text-slate-900 mb-1 flex items-center gap-2">
                  <MapIcon className="w-4 h-4 text-slate-400" />
                  Keywords vs Rank
                </h3>
                <p className="text-xs text-slate-400 mb-3">
                  A domain is plotted only when both Labs organic and Backlinks sections succeeded. Higher DataForSEO Rank is stronger. Axes are vendor counts, not a score.
                </p>
                <div className="h-72 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <ScatterChart margin={{ top: 8, right: 16, bottom: 8, left: 8 }}>
                      <CartesianGrid stroke="#f1f5f9" />
                      <XAxis
                        type="number"
                        dataKey="keywords"
                        name="Organic keywords"
                        tick={{ fill: "#94a3b8", fontSize: 11 }}
                        tickLine={false}
                        axisLine={{ stroke: "#e2e8f0" }}
                      />
                      <YAxis
                        type="number"
                        dataKey="rank"
                        name="DataForSEO Rank"
                        tick={{ fill: "#94a3b8", fontSize: 11 }}
                        tickLine={false}
                        axisLine={{ stroke: "#e2e8f0" }}
                      />
                      <Tooltip
                        cursor={{ strokeDasharray: "3 3" }}
                        formatter={(value: any, name: any) => [Number(value).toLocaleString(), name === "rank" ? "DataForSEO Rank" : "Organic keywords"]}
                        labelFormatter={(_, payload) => payload?.[0]?.payload?.name || ""}
                      />
                      <Legend />
                      {youPoint && (
                        <Scatter name={youLabel} data={[youPoint]} fill="#0f172a" />
                      )}
                      {themPoint && (
                        <Scatter name={themLabel} data={[themPoint]} fill="#64748b" />
                      )}
                    </ScatterChart>
                  </ResponsiveContainer>
                </div>
              </div>
            ) : (
              <p className="text-sm text-slate-500">Not enough complete sections to plot keywords vs Rank. A point needs both organic Labs and backlinks. Failed sections stay —.</p>
            )}

            {canPlotPositions ? (
              <div>
                <h3 className="text-sm font-semibold text-slate-900 mb-1">Organic position buckets</h3>
                <p className="text-xs text-slate-400 mb-3">
                  Keyword counts in each rank band. A side is omitted when its organic lookup failed — zeros are not invented.
                </p>
                <div className="h-64 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={positionRows} margin={{ top: 8, right: 16, bottom: 8, left: 8 }}>
                      <CartesianGrid stroke="#f1f5f9" vertical={false} />
                      <XAxis dataKey="label" tick={{ fill: "#94a3b8", fontSize: 11 }} tickLine={false} axisLine={{ stroke: "#e2e8f0" }} />
                      <YAxis tick={{ fill: "#94a3b8", fontSize: 11 }} tickLine={false} axisLine={{ stroke: "#e2e8f0" }} allowDecimals={false} />
                      <Tooltip formatter={(value: any, name: any) => [value == null ? "—" : Number(value).toLocaleString(), name]} />
                      <Legend />
                      {you.organic_ok && <Bar dataKey="you" name={youLabel} fill="#0f172a" radius={[6, 6, 0, 0]} />}
                      {them.organic_ok && <Bar dataKey="them" name={themLabel} fill="#94a3b8" radius={[6, 6, 0, 0]} />}
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>
            ) : (
              <p className="text-sm text-slate-500">Organic position buckets are unavailable until a Labs organic section succeeds.</p>
            )}

            {(you.organic_error || you.backlinks_error || them.organic_error || them.backlinks_error) && (
              <div className="space-y-1">
                {you.organic_error && <p className="text-xs text-amber-600">{you.domain} organic: {you.organic_error}</p>}
                {you.backlinks_error && <p className="text-xs text-amber-600">{you.domain} backlinks: {you.backlinks_error}</p>}
                {them.organic_error && <p className="text-xs text-amber-600">{them.domain} organic: {them.organic_error}</p>}
                {them.backlinks_error && <p className="text-xs text-amber-600">{them.domain} backlinks: {them.backlinks_error}</p>}
              </div>
            )}
            <p className="text-[10px] text-slate-400">This map does not invent a leader/laggard score. Vendor data is not mixed into Overview GSC tiles. Failed sections show — instead of zeros.</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
