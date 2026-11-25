"use client";

import { useEffect, useState, type ReactNode } from "react";
import { Download, Globe, Loader2, Lock, RefreshCw, Search } from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";

const ORANGE = "#ff642d";
const PAGE_BG = "#f4f5f7";

type Cell = { value?: number | null; status?: string; label?: string; hint?: string };

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

function CellView({ cell, money }: { cell?: Cell; money?: boolean }) {
  if (!cell || cell.status === "empty") return <span className="text-slate-400">—</span>;
  if (cell.status === "not_implemented" || cell.status === "locked") {
    return <span className="text-[12px] text-slate-400">{cell.label || "Not implemented"}</span>;
  }
  if (cell.value == null) return <span className="text-slate-400">—</span>;
  return <span className="tabular-nums">{money ? fmtMoney(cell.value) : fmtNum(cell.value)}</span>;
}

function ToolBtn({ onClick, children, disabled }: { onClick?: () => void; children: ReactNode; disabled?: boolean }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="inline-flex h-8 items-center gap-1.5 px-2.5 text-[12px] text-slate-700 border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-50"
    >
      {children}
    </button>
  );
}

const METRICS: { key: string; label: string; money?: boolean }[] = [
  { key: "ai_visibility", label: "AI Visibility" },
  { key: "authority_score", label: "Authority Score" },
  { key: "organic_traffic", label: "Organic Traffic" },
  { key: "paid_traffic", label: "Paid Traffic" },
  { key: "referring_domains", label: "Ref. Domains" },
  { key: "traffic_share", label: "Traffic Share" },
  { key: "organic_keywords", label: "Organic Keywords" },
  { key: "paid_keywords", label: "Paid Keywords" },
  { key: "backlinks", label: "Backlinks" },
];

export function DomainOverviewPanel({
  projectId,
  defaultDomain,
}: {
  projectId?: number;
  defaultDomain?: string;
  projects?: any[];
  project?: any;
  onProjectChange?: (p: any) => void;
}) {
  const [domain, setDomain] = useState(defaultDomain || "");
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [searching, setSearching] = useState(false);
  const [tab, setTab] = useState<"overview" | "growth" | "countries">("overview");

  useEffect(() => {
    if (defaultDomain && !domain) setDomain(defaultDomain);
  }, [defaultDomain, domain]);

  const load = async () => {
    if (!projectId) {
      setData(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await dashboardApi.getDomainOverview(projectId, domain.trim() || defaultDomain, {});
      setData(res.data?.data ?? res.data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [projectId]);

  const search = async (refresh = false) => {
    if (!projectId) return;
    setSearching(true);
    try {
      const res = await dashboardApi.getDomainOverview(
        projectId,
        domain.trim() || defaultDomain,
        refresh ? { refresh: true } : { fetch: true }
      );
      setData(res.data?.data ?? res.data);
    } catch (err: any) {
      const msg = err?.response?.data?.error?.message || "Search failed.";
      toast(msg);
    } finally {
      setSearching(false);
    }
  };

  const limits = data?.limits ?? {};
  const reports = data?.reports ?? {};
  const overview = reports.overview ?? {};
  const honesty = reports.honesty || data?.message || "";

  const exportCSV = () => {
    const rows = (reports.top_keywords ?? []) as any[];
    if (!rows.length) {
      toast("No keywords to export yet.");
      return;
    }
    const header = "keyword,volume,cpc,kd,position,intent";
    const lines = rows.map((r) =>
      [r.keyword, r.volume, r.cpc, r.kd, r.position, r.intent]
        .map((x) => `"${String(x ?? "").replace(/"/g, '""')}"`).join(",")
    );
    const blob = new Blob([[header, ...lines].join("\n")], { type: "text/csv" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `domain-overview-${data?.domain || "export"}.csv`;
    a.click();
  };

  if (loading) {
    return (
      <div className="min-h-[calc(100vh-4rem)] flex items-center justify-center" style={{ background: PAGE_BG }}>
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  return (
    <div className="min-h-[calc(100vh-4rem)]" style={{ background: PAGE_BG }}>
      <div className="sticky top-0 z-20 bg-white border-b border-slate-200">
        <div className="flex flex-wrap items-center gap-2 px-4 py-2.5">
          <span className="inline-flex items-center justify-center w-6 h-6" style={{ background: ORANGE }}>
            <Globe className="w-3.5 h-3.5 text-white" />
          </span>
          <h1 className="text-[15px] font-semibold text-slate-900">Domain Overview</h1>
          {limits.plan_label ? (
            <span className="h-5 px-1.5 inline-flex items-center text-[10px] font-semibold uppercase tracking-wide bg-slate-100 text-slate-600">
              {limits.plan_label}
            </span>
          ) : null}
          <span className="text-[11px] text-slate-400">
            {data?.used_today ?? 0}/{limits.requests_per_day ?? 10} today
          </span>
          <div className="ml-auto flex flex-wrap items-center gap-1.5">
            <ToolBtn onClick={exportCSV} disabled={!limits.export_csv}><Download className="w-3.5 h-3.5" /> Export</ToolBtn>
            <ToolBtn onClick={() => search(true)} disabled={searching}><RefreshCw className="w-3.5 h-3.5" /> Refresh</ToolBtn>
          </div>
        </div>
        <form
          className="flex flex-wrap items-center gap-2 px-4 pb-3"
          onSubmit={(e) => { e.preventDefault(); search(false); }}
        >
          <div className="relative flex-1 min-w-[220px] max-w-xl">
            <Globe className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400" />
            <input
              value={domain}
              onChange={(e) => setDomain(e.target.value)}
              placeholder="example.com"
              className="w-full h-9 pl-8 pr-3 text-[13px] border border-slate-200 bg-white"
            />
          </div>
          <select disabled className="h-9 px-2 text-[12px] border border-slate-200 bg-slate-50 text-slate-500">
            <option>Root domain</option>
          </select>
          <span className="text-[11px] text-slate-400">United States · English</span>
          <button
            type="submit"
            disabled={searching || !projectId}
            className="inline-flex h-9 items-center gap-1.5 px-4 text-[12px] font-semibold text-white disabled:opacity-60"
            style={{ background: ORANGE }}
          >
            {searching ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Search className="w-3.5 h-3.5" />}
            Search
          </button>
        </form>
        <div className="flex gap-4 px-4 text-[13px] border-t border-slate-100">
          {([
            { id: "overview", label: "Overview" },
            { id: "growth", label: "Growth", lock: !limits.growth },
            { id: "countries", label: "Compare by countries", lock: !limits.compare_countries },
          ] as const).map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => setTab(t.id)}
              className={`h-9 inline-flex items-center gap-1 border-b-2 ${tab === t.id ? "border-[#ff642d] text-slate-900 font-medium" : "border-transparent text-slate-500"}`}
            >
              {t.label}
              {"lock" in t && t.lock ? <Lock className="w-3 h-3" /> : null}
            </button>
          ))}
        </div>
      </div>

      <div className="px-4 py-4 space-y-4 max-w-[1200px]">
        {data?.configured === false ? (
          <p className="text-[13px] text-slate-600">{data.message}</p>
        ) : null}

        {tab === "growth" && (
          <EmptyLock
            title="Growth report"
            locked={reports.growth?.locked}
            text={reports.growth?.locked ? "Growth is included on Pro+ and above. We still have no monthly archive, so this stays Not implemented." : "Not implemented. Domain Overview does not store a monthly history yet."}
          />
        )}
        {tab === "countries" && (
          <EmptyLock
            title="Compare by countries"
            locked={reports.compare_countries?.locked}
            text={reports.compare_countries?.locked ? "Compare by countries is included on Pro+ and above. This lookup is Google United States only." : "Not implemented. This lookup is Google United States only."}
          />
        )}

        {tab === "overview" && (
          <>
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-9 gap-px bg-slate-200 border border-slate-200 bg-white">
              {METRICS.map((m) => {
                const cell: Cell = overview[m.key] ?? { status: "empty" };
                return (
                  <div key={m.key} className="bg-white px-3 py-3 min-h-[88px]">
                    <p className="text-[10px] uppercase tracking-wider text-slate-400">{m.label}</p>
                    <p className="mt-1 text-[18px] font-semibold text-slate-900 leading-tight">
                      {m.key === "traffic_share" && cell.status === "ok" && cell.value != null
                        ? `${fmtNum(cell.value)}%`
                        : <CellView cell={cell} />}
                    </p>
                    {cell.hint ? <p className="mt-1 text-[10px] text-slate-400 leading-snug">{cell.hint}</p> : null}
                  </div>
                );
              })}
            </div>

            <Section title="Organic Research">
              <div className="grid md:grid-cols-2 gap-4">
                <Box title="Top organic keywords">
                  <KeywordTable rows={reports.top_keywords ?? []} />
                </Box>
                <Box title="Keywords by intent">
                  {(reports.intent ?? []).length === 0 ? <Dash /> : (
                    <ul className="text-[13px] space-y-1">
                      {(reports.intent ?? []).map((r: any) => (
                        <li key={r.intent} className="flex justify-between">
                          <span className="capitalize">{r.intent}</span>
                          <span className="tabular-nums">{r.count}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </Box>
                <Box title="Organic position distribution">
                  {(reports.positions ?? []).length === 0 ? <Dash /> : (
                    <ul className="text-[13px] space-y-1">
                      {(reports.positions ?? []).map((r: any) => (
                        <li key={r.key} className="flex justify-between">
                          <span>Pos {r.key}</span>
                          <span className="tabular-nums">{r.count}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </Box>
                <Box title="Main organic competitors">
                  <CompetitorTable rows={reports.competitors ?? []} />
                </Box>
              </div>
              <Box title="Competitive positioning map (organic)" className="mt-4">
                {(reports.competitive_map ?? []).length === 0 ? <Dash /> : (
                  <table className="w-full text-[13px]">
                    <thead>
                      <tr className="text-[10px] uppercase tracking-wider text-slate-400">
                        <th className="text-left py-1">Domain</th>
                        <th className="text-right py-1">Keywords</th>
                        <th className="text-right py-1">Est. traffic</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(reports.competitive_map ?? []).map((r: any) => (
                        <tr key={r.domain} className="border-t border-slate-100">
                          <td className="py-1.5 font-medium">{r.domain}{r.self ? " (this domain)" : ""}</td>
                          <td className="py-1.5 text-right tabular-nums">{fmtNum(r.count)}</td>
                          <td className="py-1.5 text-right tabular-nums">{fmtNum(r.etv)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </Box>
            </Section>

            <Section title="Paid Research">
              <div className="grid md:grid-cols-2 gap-4">
                <NotImpl title="Top paid keywords" note="Paid keyword list is not implemented. The Paid Keywords tile uses domain_rank_overview count when that call succeeds." />
                <NotImpl title="Paid position distribution" />
                <NotImpl title="Main paid competitors" />
                <NotImpl title="Sample text ads" />
              </div>
            </Section>

            <Section title="Backlinks">
              <div className="grid md:grid-cols-2 gap-4">
                <Box title="Follow vs nofollow">
                  {reports.backlink_split?.ok ? (
                    <p className="text-[13px]">Follow {fmtNum(reports.backlink_split.follow)} · Nofollow {fmtNum(reports.backlink_split.nofollow)}</p>
                  ) : <Dash />}
                </Box>
                <Box title="Backlink types">
                  {(reports.link_types ?? []).length === 0 ? <Dash /> : (
                    <ul className="text-[13px] space-y-1">
                      {(reports.link_types ?? []).map((r: any) => (
                        <li key={r.key} className="flex justify-between">
                          <span className="capitalize">{r.key}</span>
                          <span className="tabular-nums">{fmtNum(r.count)}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </Box>
                <NotImpl title="Top anchors" />
                <NotImpl title="Referring domains list" />
                <NotImpl title="Indexed pages" />
                <Box title="DataForSEO Rank">
                  <p className="text-[18px] font-semibold"><CellView cell={overview.dfs_rank} /></p>
                  <p className="text-[11px] text-slate-400 mt-1">Not Moz DA and not Semrush Authority Score.</p>
                </Box>
              </div>
            </Section>

            <Section title="Traffic sources and trends">
              <div className="grid md:grid-cols-2 gap-4">
                <NotImpl title="Distribution by country" note="This lookup is Google United States only." />
                <NotImpl title="Branded vs non-branded" />
                <NotImpl title="Google SERP positions (AI Overviews)" />
                <NotImpl title="Organic traffic trend" note="No monthly archive is stored yet." />
                <NotImpl title="Key topics" />
                <NotImpl title="Top cited sources (AI)" />
              </div>
            </Section>
          </>
        )}

        <p className="text-[11px] text-slate-400 pb-8">{honesty}</p>
      </div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-3">
      <h2 className="text-[13px] font-semibold text-slate-900">{title}</h2>
      {children}
    </section>
  );
}

function Box({ title, children, className = "" }: { title: string; children: ReactNode; className?: string }) {
  return (
    <div className={`bg-white border border-slate-200 p-4 ${className}`}>
      <h3 className="text-[12px] font-semibold text-slate-700 mb-2">{title}</h3>
      {children}
    </div>
  );
}

function Dash() {
  return <p className="text-[13px] text-slate-400">—</p>;
}

function NotImpl({ title, note }: { title: string; note?: string }) {
  return (
    <Box title={title}>
      <p className="text-[13px] text-slate-400">Not implemented</p>
      {note ? <p className="text-[11px] text-slate-400 mt-1">{note}</p> : null}
    </Box>
  );
}

function EmptyLock({ title, locked, text }: { title: string; locked?: boolean; text: string }) {
  return (
    <div className="bg-white border border-slate-200 p-8 text-center">
      <h2 className="text-[15px] font-semibold text-slate-900">{title}</h2>
      {locked ? <Lock className="w-5 h-5 text-slate-300 mx-auto mt-3" /> : null}
      <p className="text-[13px] text-slate-500 mt-3 max-w-lg mx-auto">{text}</p>
    </div>
  );
}

function KeywordTable({ rows }: { rows: any[] }) {
  if (!rows.length) return <Dash />;
  return (
    <table className="w-full text-[13px]">
      <thead>
        <tr className="text-[10px] uppercase tracking-wider text-slate-400">
          <th className="text-left py-1">Keyword</th>
          <th className="text-right py-1">Vol</th>
          <th className="text-right py-1">Pos</th>
          <th className="text-right py-1">CPC</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr key={r.keyword} className="border-t border-slate-100">
            <td className="py-1.5 font-medium">{r.keyword}</td>
            <td className="py-1.5 text-right tabular-nums">{r.volume > 0 ? fmtNum(r.volume) : "—"}</td>
            <td className="py-1.5 text-right tabular-nums">{r.position > 0 ? r.position : "—"}</td>
            <td className="py-1.5 text-right tabular-nums">{fmtMoney(r.cpc)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function CompetitorTable({ rows }: { rows: any[] }) {
  if (!rows.length) return <Dash />;
  return (
    <table className="w-full text-[13px]">
      <thead>
        <tr className="text-[10px] uppercase tracking-wider text-slate-400">
          <th className="text-left py-1">Domain</th>
          <th className="text-right py-1">Common KW</th>
          <th className="text-right py-1">Est. traffic</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr key={r.domain} className="border-t border-slate-100">
            <td className="py-1.5 font-medium">{r.domain}</td>
            <td className="py-1.5 text-right tabular-nums">{fmtNum(r.intersections)}</td>
            <td className="py-1.5 text-right tabular-nums">{fmtNum(r.organic_etv)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
