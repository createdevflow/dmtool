"use client";

import { useEffect, useState } from "react";
import {
  Globe, Loader2, RefreshCw, Search, Link as LinkIcon, Users, BarChart3,
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
  return Number.isInteger(v) ? v.toLocaleString() : v.toFixed(1);
}

function fmtMoney(n: unknown): string {
  const v = Number(n);
  if (!Number.isFinite(v) || v <= 0) return "—";
  return `$${v.toFixed(2)}`;
}

export function DomainExplorerPanel({
  projectId,
  defaultDomain,
}: {
  projectId?: number;
  defaultDomain?: string;
}) {
  const [domain, setDomain] = useState(defaultDomain || "");
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (defaultDomain && !domain) setDomain(defaultDomain);
  }, [defaultDomain, domain]);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    (async () => {
      try {
        const res = await dashboardApi.getDomainExplorer(projectId, defaultDomain, {});
        if (cancelled) return;
        const payload = res.data?.data ?? res.data;
        if (payload?.available) setData(payload);
      } catch {
        /* cache peek only */
      }
    })();
    return () => { cancelled = true; };
  }, [projectId, defaultDomain]);

  const lookup = async (refresh = false) => {
    if (!projectId) return;
    setLoading(true);
    setError("");
    try {
      const res = await dashboardApi.getDomainExplorer(
        projectId,
        domain.trim() || defaultDomain,
        refresh ? { refresh: true } : { fetch: true }
      );
      const payload = res.data?.data ?? res.data;
      setData(payload);
    } catch (err: any) {
      setError(err?.response?.data?.error?.message || "Lookup failed.");
      setData(null);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
      <CardHeader className="px-8 pt-8 pb-4">
        <CardTitle className="text-base font-semibold">Domain intelligence</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          Any domain via DataForSEO Labs (Google United States). Estimated traffic, not Search Console. Lookup spends vendor credits; results cache 24 hours.
        </p>
      </CardHeader>
      <CardContent className="px-8 pb-8 space-y-6">
        <form
          className="flex flex-col sm:flex-row gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            lookup(false);
          }}
        >
          <div className="relative flex-1">
            <Globe className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <Input
              value={domain}
              onChange={(e) => setDomain(e.target.value)}
              placeholder="example.com"
              className="pl-9 h-11 rounded-xl border-slate-200"
            />
          </div>
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
        </form>

        {error && <p className="text-sm text-rose-600">{error}</p>}

        {!data && !loading && (
          <p className="text-sm text-slate-500">Enter a domain and look it up. Nothing is fetched until you do.</p>
        )}

        {data && !data.configured && (
          <p className="text-sm text-slate-600">
            {data.message || "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD."}
          </p>
        )}

        {data?.configured && !data.available && (
          <p className="text-sm text-slate-600">{data.message || "No DataForSEO data for this domain."}</p>
        )}

        {data?.available && (
          <div className="space-y-6">
            <p className="text-[11px] font-medium text-slate-400">
              {data.domain} · {data.location_name || "United States"} · {data.cached ? "cached" : "live"} · {data.message}
            </p>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {[
                { label: "Est. organic traffic", value: data.organic_ok ? fmtNum(data.organic?.etv) : "—", hint: "Labs ETV / month" },
                { label: "Ranking keywords", value: data.organic_ok ? fmtNum(data.organic?.count) : "—", hint: "Google US" },
                { label: "Backlinks", value: data.backlinks_ok ? fmtNum(data.backlinks?.backlinks) : "—", hint: "DataForSEO index" },
                { label: "Referring domains", value: data.backlinks_ok ? fmtNum(data.backlinks?.referring_domains) : "—", hint: "Not Moz DA" },
              ].map((s) => (
                <div key={s.label} className="rounded-xl border border-slate-100 bg-slate-50/60 p-4">
                  <p className="text-xl font-bold text-slate-900">{s.value}</p>
                  <p className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mt-1">{s.label}</p>
                  <p className="text-[10px] text-slate-400 mt-0.5">{s.hint}</p>
                </div>
              ))}
            </div>

            {data.organic_ok && (
              <p className="text-xs text-slate-500">
                Positions 1 / 2–3 / 4–10: {fmtNum(data.organic?.pos_1)} / {fmtNum(data.organic?.pos_2_3)} / {fmtNum(data.organic?.pos_4_10)}
                {data.backlinks_ok && data.backlinks?.rank > 0 ? ` · DataForSEO Rank ${data.backlinks.rank}` : ""}
              </p>
            )}
            {data.organic_error && <p className="text-xs text-amber-600">{data.organic_error}</p>}
            {data.backlinks_error && <p className="text-xs text-amber-600">{data.backlinks_error}</p>}

            {data.competitors_ok && Array.isArray(data.competitors) && data.competitors.length > 0 && (
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <Users className="w-4 h-4 text-slate-400" />
                  <h3 className="text-sm font-semibold text-slate-900">Organic competitors</h3>
                </div>
                <p className="text-xs text-slate-400 mb-3">Domains sharing Google US keywords with this target. Not social competitors.</p>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-[10px] uppercase tracking-widest text-slate-400">
                        <th className="text-left py-2">Domain</th>
                        <th className="text-right py-2">Keyword overlap</th>
                        <th className="text-right py-2">Est. traffic</th>
                        <th className="text-right py-2">Keywords</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.competitors.map((row: any) => (
                        <tr key={row.domain} className="border-t border-slate-50">
                          <td className="py-2 font-medium text-slate-900">{row.domain}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.intersections)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.organic_etv)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.organic_count)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
            {data.competitors_error && <p className="text-xs text-amber-600">{data.competitors_error}</p>}

            {data.keywords_ok && Array.isArray(data.keywords) && data.keywords.length > 0 && (
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <BarChart3 className="w-4 h-4 text-slate-400" />
                  <h3 className="text-sm font-semibold text-slate-900">Top ranked keywords</h3>
                </div>
                <p className="text-xs text-slate-400 mb-3">Search volume and CPC from DataForSEO Labs. Not GSC impressions.</p>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-[10px] uppercase tracking-widest text-slate-400">
                        <th className="text-left py-2">Keyword</th>
                        <th className="text-right py-2">Volume</th>
                        <th className="text-right py-2">CPC</th>
                        <th className="text-right py-2">KD</th>
                        <th className="text-right py-2">Pos</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.keywords.map((row: any) => (
                        <tr key={row.keyword} className="border-t border-slate-50">
                          <td className="py-2 font-medium text-slate-900">{row.keyword}</td>
                          <td className="py-2 text-right tabular-nums">{row.volume > 0 ? fmtNum(row.volume) : "—"}</td>
                          <td className="py-2 text-right tabular-nums">{fmtMoney(row.cpc)}</td>
                          <td className="py-2 text-right tabular-nums">{row.kd > 0 ? row.kd : "—"}</td>
                          <td className="py-2 text-right tabular-nums">{row.position > 0 ? row.position : "—"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
            {data.keywords_error && <p className="text-xs text-amber-600">{data.keywords_error}</p>}
            <p className="text-[10px] text-slate-400 flex items-center gap-1">
              <LinkIcon className="w-3 h-3" /> Vendor data is not mixed into Overview GSC tiles.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
