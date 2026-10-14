"use client";

import { useEffect, useState } from "react";
import {
  Globe, Loader2, RefreshCw, Search, Unlink,
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

function fmtDate(raw: unknown): string {
  if (typeof raw !== "string" || !raw.trim()) return "—";
  const s = raw.trim();
  if (/^\d{4}-\d{2}-\d{2}/.test(s)) return s.slice(0, 10);
  return s;
}

type ReferringRow = {
  domain?: string;
  rank?: number;
  backlinks?: number;
  referring_pages?: number;
  spam_score?: number;
  first_seen?: string;
};

export function BacklinkGapPanel({
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
      const res = await dashboardApi.getBacklinkGap(projectId, {
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

  const referring: ReferringRow[] = Array.isArray(data?.referring) ? data.referring : [];
  const ok = !!data?.intersection_ok;
  const total = Number(data?.total_count);
  const sampleLabel = Number.isFinite(total) && total > referring.length
    ? `Showing ${referring.length} of ${total.toLocaleString()}`
    : referring.length
      ? `${referring.length} referring domain${referring.length === 1 ? "" : "s"}`
      : "";

  return (
    <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
      <CardHeader className="px-8 pt-8 pb-4">
        <CardTitle className="text-base font-semibold">Backlink gap</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          Referring domains that link to the competitor and not to you. DataForSEO backlinks domain intersection — not Search Console and not an outreach list. Rank is DataForSEO Rank, not Moz Domain Authority. Lookup spends vendor credits; results cache 24 hours.
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
              <Unlink className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
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
            <p className="text-sm text-slate-600">{data.message || "No backlink gap data for this pair."}</p>
            {data.intersection_error && <p className="text-xs text-amber-600">{data.intersection_error}</p>}
          </div>
        )}

        {data?.available && (
          <div className="space-y-6">
            <p className="text-[11px] font-medium text-slate-400">
              {data.target} vs {data.competitor} · {data.cached ? "cached" : "live"} · {data.message}
            </p>
            <div>
              <h3 className="text-sm font-semibold text-slate-900 mb-1">Link to them, not to you</h3>
              <p className="text-xs text-slate-400 mb-3">
                Up to 15 referring domains, ordered by DataForSEO Rank. {sampleLabel ? `${sampleLabel}. ` : ""}This is not a recommendation to contact these sites.
              </p>
              {ok && referring.length === 0 ? (
                <p className="text-sm text-slate-500">No referring domains in this sample.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-[10px] uppercase tracking-widest text-slate-400">
                        <th className="text-left py-2">Referring domain</th>
                        <th className="text-right py-2">Rank</th>
                        <th className="text-right py-2">Backlinks</th>
                        <th className="text-right py-2">Pages</th>
                        <th className="text-right py-2">Spam</th>
                        <th className="text-right py-2">First seen</th>
                      </tr>
                    </thead>
                    <tbody>
                      {referring.map((row) => (
                        <tr key={row.domain} className="border-t border-slate-50">
                          <td className="py-2 font-medium text-slate-900">{row.domain || "—"}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.rank, ok)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.backlinks, ok)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.referring_pages, ok)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.spam_score, ok)}</td>
                          <td className="py-2 text-right tabular-nums text-slate-600">{fmtDate(row.first_seen)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
            {data.intersection_error && (
              <p className="text-xs text-amber-600">{data.intersection_error}</p>
            )}
            <p className="text-[10px] text-slate-400">Vendor data is not mixed into Overview GSC tiles or the Backlinks summary page. Failed lookups show — instead of zeros.</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
