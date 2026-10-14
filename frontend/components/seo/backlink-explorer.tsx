"use client";

import { useEffect, useState } from "react";
import {
  Globe, Loader2, RefreshCw, Search, Link as LinkIcon,
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

function mapEntries(raw: unknown): [string, number][] {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return [];
  return Object.entries(raw as Record<string, unknown>)
    .map(([k, v]) => [k, Number(v)] as [string, number])
    .filter(([k, v]) => k && Number.isFinite(v));
}

type ReferringRow = {
  domain?: string;
  rank?: number;
  backlinks?: number;
  referring_pages?: number;
  spam_score?: number;
  first_seen?: string;
  broken_backlinks?: number;
  referring_pages_nofollow?: number;
};

export function BacklinkExplorerPanel({
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
        const res = await dashboardApi.getBacklinkExplorer(projectId, defaultDomain, {});
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
      const res = await dashboardApi.getBacklinkExplorer(
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

  const referring: ReferringRow[] = Array.isArray(data?.referring) ? data.referring : [];
  const summaryOK = !!data?.summary_ok;
  const referringOK = !!data?.referring_ok;
  const audit = data?.audit ?? {};
  const total = Number(data?.total_count);
  const sampleLabel = Number.isFinite(total) && total > referring.length
    ? `Showing ${referring.length} of ${total.toLocaleString()}`
    : referring.length
      ? `${referring.length} referring domain${referring.length === 1 ? "" : "s"}`
      : "";
  const linkTypes = mapEntries(audit.link_types);
  const linkAttrs = mapEntries(audit.link_attributes);

  return (
    <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
      <CardHeader className="px-8 pt-8 pb-4">
        <CardTitle className="text-base font-semibold">Backlink explorer</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          Referring domains and a backlink audit from DataForSEO. Rank is DataForSEO Rank, not Moz Domain Authority. Not Search Console and not an outreach list. Lookup spends vendor credits; results cache 24 hours.
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
          <p className="text-sm text-slate-500">Enter a domain, then look it up. Nothing is fetched until you do.</p>
        )}

        {data && !data.configured && (
          <p className="text-sm text-slate-600">
            {data.message || "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD."}
          </p>
        )}

        {data?.configured && !data.available && (
          <div className="space-y-1">
            <p className="text-sm text-slate-600">{data.message || "No backlink data for this domain."}</p>
            {data.summary_error && <p className="text-xs text-amber-600">{data.summary_error}</p>}
            {data.referring_error && <p className="text-xs text-amber-600">{data.referring_error}</p>}
          </div>
        )}

        {data?.available && (
          <div className="space-y-6">
            <p className="text-[11px] font-medium text-slate-400">
              {data.domain} · {data.cached ? "cached" : "live"} · {data.message}
            </p>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {[
                { label: "Backlinks", value: fmtNum(audit.backlinks, summaryOK) },
                { label: "Referring domains", value: fmtNum(audit.referring_domains, summaryOK) },
                { label: "Referring main domains", value: fmtNum(audit.referring_main_domains, summaryOK) },
                { label: "DataForSEO Rank", value: fmtNum(audit.rank, summaryOK), hint: "Not Moz DA" },
                { label: "Moz DA", value: "—" },
                { label: "Spam score", value: fmtNum(audit.spam_score, summaryOK) },
                { label: "Broken backlinks", value: fmtNum(audit.broken_backlinks, summaryOK) },
                { label: "Broken pages", value: fmtNum(audit.broken_pages, summaryOK) },
                { label: "Nofollow referring domains", value: fmtNum(audit.referring_domains_nofollow, summaryOK) },
                { label: "Referring pages", value: fmtNum(audit.referring_pages, summaryOK) },
                { label: "Referring IPs", value: fmtNum(audit.referring_ips, summaryOK) },
                { label: "Referring subnets", value: fmtNum(audit.referring_subnets, summaryOK) },
              ].map((s) => (
                <div key={s.label} className="rounded-xl border border-slate-100 p-4">
                  <p className="text-xl font-bold text-slate-900 tabular-nums">{s.value}</p>
                  <p className="text-[11px] font-bold uppercase tracking-widest text-slate-400 mt-1">{s.label}</p>
                  {"hint" in s && s.hint ? <p className="text-[10px] text-slate-400 mt-0.5">{s.hint}</p> : null}
                </div>
              ))}
            </div>

            {summaryOK && audit.first_seen ? (
              <p className="text-xs text-slate-500">First seen {fmtDate(audit.first_seen)}.</p>
            ) : null}

            {(linkTypes.length > 0 || linkAttrs.length > 0) && (
              <div className="grid md:grid-cols-2 gap-4">
                {linkTypes.length > 0 && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-900 mb-2">Link types</h3>
                    <div className="flex flex-wrap gap-2">
                      {linkTypes.map(([k, v]) => (
                        <span key={k} className="text-xs rounded-full bg-slate-50 border border-slate-100 px-2.5 py-1 text-slate-600">
                          {k} · {v.toLocaleString()}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
                {linkAttrs.length > 0 && (
                  <div>
                    <h3 className="text-sm font-semibold text-slate-900 mb-2">Link attributes</h3>
                    <div className="flex flex-wrap gap-2">
                      {linkAttrs.map(([k, v]) => (
                        <span key={k} className="text-xs rounded-full bg-slate-50 border border-slate-100 px-2.5 py-1 text-slate-600">
                          {k} · {v.toLocaleString()}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            <div>
              <h3 className="text-sm font-semibold text-slate-900 mb-1 flex items-center gap-2">
                <LinkIcon className="w-4 h-4 text-slate-400" />
                Referring domains
              </h3>
              <p className="text-xs text-slate-400 mb-3">
                Up to 15 referring domains, ordered by DataForSEO Rank. {sampleLabel ? `${sampleLabel}. ` : ""}This is not a recommendation to contact these sites.
              </p>
              {referringOK && referring.length === 0 ? (
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
                        <th className="text-right py-2">Broken</th>
                        <th className="text-right py-2">First seen</th>
                      </tr>
                    </thead>
                    <tbody>
                      {referring.map((row) => (
                        <tr key={row.domain} className="border-t border-slate-50">
                          <td className="py-2 font-medium text-slate-900">{row.domain || "—"}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.rank, referringOK)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.backlinks, referringOK)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.referring_pages, referringOK)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.spam_score, referringOK)}</td>
                          <td className="py-2 text-right tabular-nums">{fmtNum(row.broken_backlinks, referringOK)}</td>
                          <td className="py-2 text-right tabular-nums text-slate-600">{fmtDate(row.first_seen)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
            {data.summary_error && <p className="text-xs text-amber-600">{data.summary_error}</p>}
            {data.referring_error && <p className="text-xs text-amber-600">{data.referring_error}</p>}
            <p className="text-[10px] text-slate-400">Vendor data is not mixed into Overview GSC tiles. Failed lookups show — instead of zeros. Moz Domain Authority is not provided.</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
