"use client";

import { useEffect, useState } from "react";
import {
  Globe, Loader2, Quote, RefreshCw, Search,
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
  return v.toLocaleString();
}

export function CitationGapPanel({
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
      const res = await dashboardApi.getCitationGap(projectId, {
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

  const you = data?.target_citations ?? {};
  const them = data?.competitor_citations ?? {};
  const youOK = !!data?.target_ok;
  const themOK = !!data?.competitor_ok;
  const unique = Array.isArray(data?.unique_sources) ? data.unique_sources : [];

  const rows = data?.available
    ? [
        { label: "Google AI Overviews", you: fmtNum(you.google?.mentions, youOK), them: fmtNum(them.google?.mentions, themOK), hint: "Cited as a source" },
        { label: "ChatGPT", you: fmtNum(you.chat_gpt?.mentions, youOK), them: fmtNum(them.chat_gpt?.mentions, themOK), hint: "US English only" },
        { label: "Total citations", you: fmtNum(you.mentions, youOK), them: fmtNum(them.mentions, themOK), hint: "Both platforms" },
        { label: "AI search volume", you: fmtNum(you.ai_search_volume, youOK), them: fmtNum(them.ai_search_volume, themOK), hint: "Vendor estimate" },
      ]
    : [];

  return (
    <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
      <CardHeader className="px-8 pt-8 pb-4">
        <CardTitle className="text-base font-semibold">Citation gap</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          How often each domain is cited as a source in Google AI Overviews and ChatGPT (United States, English). DataForSEO LLM Mentions — not Search Console and not an AI visibility score. Lookup spends vendor credits; results cache 24 hours.
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
              <Quote className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
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
            <p className="text-sm text-slate-600">{data.message || "No citation data for this pair."}</p>
            {data.target_error && <p className="text-xs text-amber-600">{data.target}: {data.target_error}</p>}
            {data.competitor_error && <p className="text-xs text-amber-600">{data.competitor}: {data.competitor_error}</p>}
          </div>
        )}

        {data?.available && (
          <div className="space-y-6">
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
            {(data.target_error || data.competitor_error) && (
              <div className="space-y-1">
                {data.target_error && <p className="text-xs text-amber-600">{data.target}: {data.target_error}</p>}
                {data.competitor_error && <p className="text-xs text-amber-600">{data.competitor}: {data.competitor_error}</p>}
              </div>
            )}

            {youOK && themOK && (
              <div>
                <h3 className="text-sm font-semibold text-slate-900 mb-1">Co-cited with them, not with you</h3>
                <p className="text-xs text-slate-400 mb-3">Other domains cited in the same AI answers as the competitor, and not in yours. Both lookups must succeed. This is not a recommendation list.</p>
                {unique.length === 0 ? (
                  <p className="text-sm text-slate-500">No unique co-cited domains in this sample.</p>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="text-[10px] uppercase tracking-widest text-slate-400">
                          <th className="text-left py-2">Domain</th>
                          <th className="text-right py-2">Citations</th>
                          <th className="text-right py-2">AI search volume</th>
                        </tr>
                      </thead>
                      <tbody>
                        {unique.map((row: any) => (
                          <tr key={row.domain} className="border-t border-slate-50">
                            <td className="py-2 font-medium text-slate-900">{row.domain || "—"}</td>
                            <td className="py-2 text-right tabular-nums">{fmtNum(row.mentions, true)}</td>
                            <td className="py-2 text-right tabular-nums">{fmtNum(row.ai_search_volume, true)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}
            <p className="text-[10px] text-slate-400">Vendor data is not mixed into Overview GSC tiles or AI Insights cards. Failed sections show — instead of zeros.</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
