"use client";

import { useEffect, useState } from "react";
import {
  Globe, Loader2, RefreshCw, Scale, Search,
} from "lucide-react";
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

function fmtMoney(n: unknown, ok: boolean): string {
  if (!ok) return "—";
  const v = Number(n);
  if (!Number.isFinite(v)) return "—";
  return `$${v.toFixed(2)}`;
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
    paid_etv?: number;
  };
  backlinks_ok?: boolean;
  backlinks_error?: string;
  backlinks?: {
    backlinks?: number;
    referring_domains?: number;
    rank?: number;
  };
};

export function DomainComparePanel({
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

  const rows: { label: string; you: string; them: string; hint: string }[] = data?.available
    ? [
        { label: "Organic keywords", you: fmtNum(you.organic?.count, !!you.organic_ok), them: fmtNum(them.organic?.count, !!them.organic_ok), hint: "Labs ranking keywords" },
        { label: "Organic ETV", you: fmtMoney(you.organic?.etv, !!you.organic_ok), them: fmtMoney(them.organic?.etv, !!them.organic_ok), hint: "Estimated traffic value" },
        { label: "Position 1", you: fmtNum(you.organic?.pos_1, !!you.organic_ok), them: fmtNum(them.organic?.pos_1, !!them.organic_ok), hint: "Keywords at rank 1" },
        { label: "Positions 2–3", you: fmtNum(you.organic?.pos_2_3, !!you.organic_ok), them: fmtNum(them.organic?.pos_2_3, !!them.organic_ok), hint: "Keywords at rank 2–3" },
        { label: "Positions 4–10", you: fmtNum(you.organic?.pos_4_10, !!you.organic_ok), them: fmtNum(them.organic?.pos_4_10, !!them.organic_ok), hint: "Keywords at rank 4–10" },
        { label: "Paid ETV", you: fmtMoney(you.organic?.paid_etv, !!you.organic_ok), them: fmtMoney(them.organic?.paid_etv, !!them.organic_ok), hint: "Paid search estimate" },
        { label: "Backlinks", you: fmtNum(you.backlinks?.backlinks, !!you.backlinks_ok), them: fmtNum(them.backlinks?.backlinks, !!them.backlinks_ok), hint: "DataForSEO index" },
        { label: "Referring domains", you: fmtNum(you.backlinks?.referring_domains, !!you.backlinks_ok), them: fmtNum(them.backlinks?.referring_domains, !!them.backlinks_ok), hint: "Not Moz DA" },
        { label: "DataForSEO Rank", you: fmtNum(you.backlinks?.rank, !!you.backlinks_ok), them: fmtNum(them.backlinks?.rank, !!them.backlinks_ok), hint: "Not Moz Domain Authority" },
      ]
    : [];

  return (
    <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
      <CardHeader className="px-8 pt-8 pb-4">
        <CardTitle className="text-base font-semibold">Domain comparison</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          Side-by-side organic Labs estimates and backlink index counts for two domains (United States, English). Not Search Console. Lookup spends vendor credits; results cache 24 hours.
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
          <p className="text-sm text-slate-500">Enter your domain and a competitor, then look it up. Nothing is fetched until you do.</p>
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
          <div className="space-y-4">
            <p className="text-[11px] font-medium text-slate-400">
              {data.target} vs {data.competitor} · {data.location_name || "United States"} · {data.cached ? "cached" : "live"} · {data.message}
            </p>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-[10px] uppercase tracking-widest text-slate-400">
                    <th className="text-left py-2">Metric</th>
                    <th className="text-right py-2">{data.target || "You"}</th>
                    <th className="text-right py-2">{data.competitor || "Them"}</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr key={row.label} className="border-t border-slate-50">
                      <td className="py-2">
                        <p className="font-medium text-slate-900">{row.label}</p>
                        <p className="text-[10px] text-slate-400">{row.hint}</p>
                      </td>
                      <td className="py-2 text-right tabular-nums font-medium text-slate-900">{row.you}</td>
                      <td className="py-2 text-right tabular-nums font-medium text-slate-900">{row.them}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {(you.organic_error || you.backlinks_error || them.organic_error || them.backlinks_error) && (
              <div className="space-y-1">
                {you.organic_error && <p className="text-xs text-amber-600">{you.domain} organic: {you.organic_error}</p>}
                {you.backlinks_error && <p className="text-xs text-amber-600">{you.domain} backlinks: {you.backlinks_error}</p>}
                {them.organic_error && <p className="text-xs text-amber-600">{them.domain} organic: {them.organic_error}</p>}
                {them.backlinks_error && <p className="text-xs text-amber-600">{them.domain} backlinks: {them.backlinks_error}</p>}
              </div>
            )}
            <p className="text-[10px] text-slate-400">Vendor data is not mixed into Overview GSC tiles. Failed sections show — instead of zeros.</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
