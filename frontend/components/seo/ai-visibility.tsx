"use client";

import { useEffect, useState } from "react";
import {
  Globe, Loader2, RefreshCw, Search, Sparkles,
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

export function AIVisibilityPanel({
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
        const res = await dashboardApi.getAIVisibility(projectId, defaultDomain, {});
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
      const res = await dashboardApi.getAIVisibility(
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
        <CardTitle className="text-base font-semibold">LLM mentions</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          How often this domain is mentioned or cited in Google AI Overviews and ChatGPT (United States, English). DataForSEO LLM Mentions — not Search Console. Lookup spends vendor credits; results cache 24 hours.
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
          <div className="space-y-1">
            <p className="text-sm text-slate-600">{data.message || "No LLM mention data for this domain."}</p>
            {data.mentions_error && <p className="text-xs text-amber-600">{data.mentions_error}</p>}
            {data.citations_error && <p className="text-xs text-amber-600">{data.citations_error}</p>}
          </div>
        )}

        {data?.available && (
          <div className="space-y-6">
            <p className="text-[11px] font-medium text-slate-400">
              {data.domain} · {data.location_name || "United States"} · {data.cached ? "cached" : "live"} · {data.message}
            </p>

            <div>
              <div className="flex items-center gap-2 mb-3">
                <Sparkles className="w-4 h-4 text-slate-400" />
                <h3 className="text-sm font-semibold text-slate-900">Citations as a source</h3>
              </div>
              <p className="text-xs text-slate-400 mb-3">Times the model actually cited this domain. Zero is a real count when the lookup succeeded.</p>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                {[
                  { label: "Google AI Overviews", value: data.citations_ok ? fmtNum(data.citations?.google?.mentions) : "—", hint: "Cited sources" },
                  { label: "ChatGPT", value: data.citations_ok ? fmtNum(data.citations?.chat_gpt?.mentions) : "—", hint: "US English only" },
                  { label: "Total citations", value: data.citations_ok ? fmtNum(data.citations?.mentions) : "—", hint: "Both platforms" },
                  { label: "AI search volume", value: data.citations_ok ? fmtNum(data.citations?.ai_search_volume) : "—", hint: "Vendor estimate" },
                ].map((s) => (
                  <div key={s.label} className="rounded-xl border border-slate-100 bg-slate-50/60 p-4">
                    <p className="text-xl font-bold text-slate-900">{s.value}</p>
                    <p className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mt-1">{s.label}</p>
                    <p className="text-[10px] text-slate-400 mt-0.5">{s.hint}</p>
                  </div>
                ))}
              </div>
              {data.citations_error && <p className="text-xs text-amber-600 mt-2">{data.citations_error}</p>}
            </div>

            <div>
              <h3 className="text-sm font-semibold text-slate-900 mb-3">Mentions in answers</h3>
              <p className="text-xs text-slate-400 mb-3">Any appearance of this domain in LLM responses, not only citations.</p>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                {[
                  { label: "Google mentions", value: data.mentions_ok ? fmtNum(data.mentions?.google?.mentions) : "—" },
                  { label: "ChatGPT mentions", value: data.mentions_ok ? fmtNum(data.mentions?.chat_gpt?.mentions) : "—" },
                  { label: "Total mentions", value: data.mentions_ok ? fmtNum(data.mentions?.mentions) : "—" },
                ].map((s) => (
                  <div key={s.label} className="rounded-xl border border-slate-100 bg-slate-50/60 p-4">
                    <p className="text-xl font-bold text-slate-900">{s.value}</p>
                    <p className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mt-1">{s.label}</p>
                  </div>
                ))}
              </div>
              {data.mentions_error && <p className="text-xs text-amber-600 mt-2">{data.mentions_error}</p>}
            </div>

            {data.citations_ok && Array.isArray(data.citations?.sources) && data.citations.sources.length > 0 && (
              <div>
                <h3 className="text-sm font-semibold text-slate-900 mb-3">Co-cited source domains</h3>
                <p className="text-xs text-slate-400 mb-3">Other domains cited in the same AI answers. Target domain is omitted.</p>
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
                      {data.citations.sources.map((row: any) => (
                        <tr key={row.domain} className="border-t border-slate-50">
                          <td className="py-2 font-medium text-slate-900">{row.domain}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.mentions)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.ai_search_volume)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
            <p className="text-[10px] text-slate-400">Vendor data is not mixed into Overview GSC tiles.</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
