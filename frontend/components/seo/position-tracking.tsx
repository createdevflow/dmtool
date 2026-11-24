"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import Link from "next/link";
import {
  Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from "recharts";
import {
  AlertCircle, Download, Globe, Loader2, Lock, MapPin, Monitor,
  RefreshCw, Search, Settings2, Smartphone, Sparkles,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";

const ORANGE = "#ff622d";
const COMP_COLORS = ["#2563eb", "#7c3aed", "#059669", "#d97706", "#0ea5e9"];

function dash(v: unknown, digits = 0): string {
  if (v == null || v === "") return "—";
  const n = Number(v);
  if (!Number.isFinite(n)) return "—";
  return digits > 0 ? n.toFixed(digits) : String(Math.round(n));
}

function posTone(pos: number) {
  if (pos <= 0) return "text-slate-400";
  if (pos <= 3) return "text-emerald-600";
  if (pos <= 10) return "text-orange-600";
  if (pos <= 30) return "text-amber-600";
  return "text-slate-600";
}

function featureLabel(raw: string) {
  const k = String(raw || "").toLowerCase();
  if (k.includes("featured_snippet") || k === "featured snippet") return "FS";
  if (k.includes("ai_overview") || k.includes("ai overview")) return "AIO";
  if (k.includes("people_also_ask") || k.includes("paa")) return "PAA";
  if (k.includes("sitelink")) return "SL";
  if (k.includes("local")) return "Local";
  if (k.includes("image")) return "Image";
  if (k.includes("video")) return "Video";
  if (k.includes("shopping")) return "Shop";
  if (k === "organic") return "";
  return raw.replace(/_/g, " ").slice(0, 12);
}

const TABS = [
  { id: "overview", label: "Overview" },
  { id: "rankings", label: "Rankings" },
  { id: "distribution", label: "Rankings Distribution" },
  { id: "pages", label: "Pages" },
  { id: "tags", label: "Tags", lockKey: "tags" },
  { id: "devices", label: "Devices & Locations", lockKey: "devices_locations" },
  { id: "landscape", label: "Landscape" },
  { id: "discovery", label: "Competitors Discovery" },
  { id: "cannibalization", label: "Cannibalization", lockKey: "cannibalization" },
  { id: "snippets", label: "Featured Snippets" },
  { id: "ai", label: "AI Search", lockKey: "ai_search" },
] as const;

function isLocked(reports: any, key?: string) {
  if (!key) return false;
  return !!reports?.[key]?.locked;
}

function Kpi({
  label, value, hint, bar,
}: {
  label: string;
  value: string;
  hint: string;
  bar?: number | null;
}) {
  const pct = bar == null || !Number.isFinite(bar) ? 0 : Math.max(0, Math.min(100, bar));
  return (
    <div className="border border-slate-200 bg-white px-4 py-3 min-w-0">
      <p className="text-[11px] font-medium text-slate-500 uppercase tracking-wide">{label}</p>
      <p className="mt-1 text-[28px] leading-none font-semibold text-slate-900 tabular-nums">{value}</p>
      <div className="mt-3 h-1.5 bg-slate-100">
        <div className="h-full" style={{ width: `${pct}%`, background: ORANGE }} />
      </div>
      <p className="mt-2 text-[11px] text-slate-400">{hint}</p>
    </div>
  );
}

function Locked({ plan, reason }: { plan?: string; reason?: string }) {
  return (
    <div className="relative overflow-hidden border border-slate-200 bg-[#f7f8fa] min-h-[280px]">
      <div className="absolute inset-0 opacity-40 pointer-events-none" style={{
        backgroundImage:
          "linear-gradient(#e5e7eb 1px, transparent 1px), linear-gradient(90deg, #e5e7eb 1px, transparent 1px)",
        backgroundSize: "24px 28px",
      }} />
      <div className="relative flex flex-col items-center justify-center py-20 px-6 text-center">
        <div className="w-10 h-10 rounded-full bg-white border border-slate-200 flex items-center justify-center mb-3 shadow-sm">
          <Lock className="w-4 h-4 text-slate-500" />
        </div>
        <p className="text-sm font-semibold text-slate-800">
          {plan ? `Available on ${plan}` : "Upgrade to unlock this report"}
        </p>
        <p className="text-xs text-slate-500 mt-1 max-w-md">{reason}</p>
        <Link
          href="/billing"
          className="mt-4 inline-flex h-8 items-center px-4 text-xs font-semibold text-white"
          style={{ background: ORANGE }}
        >
          Upgrade
        </Link>
      </div>
    </div>
  );
}

function DenseTable({
  columns,
  rows,
  empty,
  render,
}: {
  columns: { key: string; label: string; align?: "left" | "right" | "center" }[];
  rows: any[];
  empty: string;
  render: (row: any, col: string) => ReactNode;
}) {
  return (
    <div className="border border-slate-200 bg-white overflow-hidden">
      {rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <AlertCircle className="w-5 h-5 text-slate-300 mb-2" />
          <p className="text-sm text-slate-500">{empty}</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-[13px]">
            <thead>
              <tr className="bg-[#f4f5f7] border-b border-slate-200">
                {columns.map((c) => (
                  <th
                    key={c.key}
                    className={`px-3 py-2 text-[11px] font-semibold text-slate-500 uppercase tracking-wide whitespace-nowrap ${
                      c.align === "right" ? "text-right" : c.align === "center" ? "text-center" : "text-left"
                    }`}
                  >
                    {c.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr key={i} className="border-b border-slate-100 hover:bg-[#fff4f0]">
                  {columns.map((c) => (
                    <td
                      key={c.key}
                      className={`px-3 py-[9px] align-middle ${
                        c.align === "right" ? "text-right" : c.align === "center" ? "text-center" : "text-left"
                      }`}
                    >
                      {render(row, c.key)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function PosCell({ pos }: { pos: number }) {
  if (!(pos > 0)) return <span className="text-slate-400">—</span>;
  return <span className={`font-semibold tabular-nums ${posTone(pos)}`}>{pos}</span>;
}

export function PositionTrackingPanel({
  projectId,
  defaultDomain,
}: {
  projectId?: number;
  defaultDomain?: string;
}) {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [tab, setTab] = useState("overview");
  const [search, setSearch] = useState("");
  const [posFilter, setPosFilter] = useState<"all" | "top3" | "top10" | "top30" | "lost">("all");
  const [domain, setDomain] = useState(defaultDomain || "");
  const [keywordText, setKeywordText] = useState("");
  const [competitorText, setCompetitorText] = useState("");
  const [locationName, setLocationName] = useState("United States");
  const [device, setDevice] = useState("desktop");
  const [engine, setEngine] = useState("google");
  const [showSetup, setShowSetup] = useState(false);

  const load = async () => {
    if (!projectId) {
      setData(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await dashboardApi.getPositionTracking(projectId);
      const payload = res.data?.data ?? null;
      setData(payload);
      const camp = payload?.campaign;
      if (camp) {
        setDomain(camp.domain || defaultDomain || "");
        setLocationName(camp.location_name || "United States");
        setDevice(camp.device || "desktop");
        setEngine(camp.engine || "google");
        setKeywordText((camp.keywords || []).map((k: any) => {
          const tags = (k.tags || []).join(", ");
          return tags ? `${k.keyword} | ${tags}` : k.keyword;
        }).join("\n"));
        setCompetitorText((camp.competitors || []).join("\n"));
        setShowSetup(false);
      } else {
        setDomain(defaultDomain || "");
        setShowSetup(true);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, [projectId]);

  const limits = data?.limits ?? {};
  const reports = data?.reports ?? {};
  const campaign = data?.campaign;
  const gsc = data?.gsc ?? {};
  const serp = data?.serp ?? {};
  const overview = reports.overview ?? {};
  const competitors: string[] = campaign?.competitors ?? [];

  const rankingRows = useMemo(() => {
    return (reports.rankings ?? []).filter((k: any) => {
      const q = search.trim().toLowerCase();
      if (q && !String(k.keyword || "").toLowerCase().includes(q)) return false;
      const p = Number(k.position) || 0;
      if (posFilter === "top3") return p > 0 && p <= 3;
      if (posFilter === "top10") return p > 0 && p <= 10;
      if (posFilter === "top30") return p > 0 && p <= 30;
      if (posFilter === "lost") return p <= 0;
      return true;
    });
  }, [reports.rankings, search, posFilter]);

  const distChart = useMemo(() => {
    const youTop3 = (reports.rankings ?? []).filter((k: any) => k.position > 0 && k.position <= 3).length;
    const youTop10 = (reports.rankings ?? []).filter((k: any) => k.position > 0 && k.position <= 10).length;
    const youTop30 = (reports.rankings ?? []).filter((k: any) => k.position > 0 && k.position <= 30).length;
    const youTop100 = (reports.rankings ?? []).filter((k: any) => k.position > 0).length;
    const rows = [
      { bucket: "Top 3", You: youTop3 },
      { bucket: "Top 10", You: youTop10 },
      { bucket: "Top 30", You: youTop30 },
      { bucket: "Top 100", You: youTop100 },
    ] as any[];
    for (const d of competitors) {
      const count = (posMax: number) =>
        (reports.landscape ?? []).filter((row: any) => {
          const p = Number(row.by_domain?.[d]) || 0;
          return p > 0 && p <= posMax;
        }).length;
      rows[0][d] = count(3);
      rows[1][d] = count(10);
      rows[2][d] = count(30);
      rows[3][d] = count(100);
    }
    return rows;
  }, [reports.rankings, reports.landscape, competitors]);

  const handleSave = async () => {
    if (!projectId) return;
    setSaving(true);
    try {
      await dashboardApi.savePositionTracking({
        project_id: projectId,
        domain,
        location_name: locationName,
        language_code: "en",
        device,
        engine,
        keyword_text: keywordText,
        competitor_text: competitorText,
      });
      toast("Campaign saved.", "success");
      await load();
    } catch (err: any) {
      toast(err.response?.data?.error?.message || "Could not save campaign.", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleRefresh = async () => {
    if (!projectId) return;
    setRefreshing(true);
    try {
      const res = await dashboardApi.refreshPositionTracking(projectId, { fetch: true });
      const payload = res.data?.data ?? null;
      if (payload) setData(payload);
      toast(payload?.serp?.message || "Rankings updated.", "success");
      await load();
    } catch (err: any) {
      toast(err.response?.data?.error?.message || "Refresh failed.", "error");
    } finally {
      setRefreshing(false);
    }
  };

  const exportCSV = () => {
    const rows = reports.rankings ?? [];
    const header = "keyword,position,url,source,gsc_impressions,gsc_clicks";
    const lines = rows.map((r: any) =>
      [r.keyword, r.position || "", r.url || "", r.source || "", r.gsc_impressions || "", r.gsc_clicks || ""]
        .map((x: any) => `"${String(x).replace(/"/g, '""')}"`)
        .join(",")
    );
    const blob = new Blob([[header, ...lines].join("\n")], { type: "text/csv" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = "position-tracking.csv";
    a.click();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[50vh] bg-white border border-slate-200">
        <Loader2 className="w-6 h-6 animate-spin text-orange-400" />
      </div>
    );
  }

  const DeviceIcon = device === "mobile" ? Smartphone : Monitor;
  const vis = overview.visibility == null ? null : Number(overview.visibility);

  return (
    <div className="bg-white border border-slate-200 -mx-1">
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3 px-4 py-3 border-b border-slate-200 bg-[#fafbfc]">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="inline-block w-1.5 h-5 rounded-sm" style={{ background: ORANGE }} />
            <h1 className="text-lg font-semibold text-slate-900">Position Tracking</h1>
            <span className="text-[10px] font-semibold uppercase tracking-wider px-1.5 py-0.5 border border-slate-200 text-slate-500 bg-white">
              {limits.plan_label || "Free"}
            </span>
          </div>
          {campaign ? (
            <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-600">
              <span className="font-semibold text-slate-800">{campaign.domain}</span>
              <span className="flex items-center gap-1"><Globe className="w-3 h-3" />{campaign.engine}</span>
              <span className="flex items-center gap-1"><MapPin className="w-3 h-3" />{campaign.location_name}</span>
              <span className="flex items-center gap-1"><DeviceIcon className="w-3 h-3" />{campaign.device}</span>
              <span className="text-slate-400">
                {campaign.snapshot_at ? `Updated ${new Date(campaign.snapshot_at).toLocaleString()}` : "No SERP snapshot yet"}
              </span>
            </div>
          ) : (
            <p className="mt-1 text-xs text-slate-500">Set up a campaign to track Google or Bing rankings.</p>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {campaign && (
            <>
              <Button variant="outline" className="h-8 rounded-md text-xs gap-1.5" onClick={() => setShowSetup(true)}>
                <Settings2 className="w-3.5 h-3.5" /> Campaign settings
              </Button>
              <Button variant="outline" className="h-8 rounded-md text-xs gap-1.5" onClick={exportCSV} disabled={!(reports.rankings || []).length}>
                <Download className="w-3.5 h-3.5" /> Export
              </Button>
              <Button
                className="h-8 rounded-md text-xs gap-1.5 text-white hover:opacity-90"
                style={{ background: ORANGE }}
                onClick={handleRefresh}
                disabled={refreshing || !projectId}
              >
                {refreshing ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RefreshCw className="w-3.5 h-3.5" />}
                Update
              </Button>
            </>
          )}
        </div>
      </div>

      {serp?.message && campaign && !showSetup && (
        <div className="px-4 py-2 text-xs text-slate-600 bg-[#fff8f5] border-b border-orange-100 flex items-start gap-2">
          <Globe className="w-3.5 h-3.5 mt-0.5 text-orange-500 shrink-0" />
          <span>
            {serp.message}
            {gsc.used_as_fallback ? " GSC is filling empty ranks until a SERP lookup runs." : ""}
          </span>
        </div>
      )}

      {(!campaign || showSetup) && (
        <div className="p-4 md:p-6 bg-[#f7f8fa] border-b border-slate-200">
          <div className="max-w-4xl border border-slate-200 bg-white">
            <div className="px-5 py-3 border-b border-slate-200 flex items-center justify-between">
              <div>
                <p className="text-sm font-semibold text-slate-900">{campaign ? "Campaign settings" : "Set up Position Tracking"}</p>
                <p className="text-[11px] text-slate-500 mt-0.5">
                  {limits.plan_label}: {limits.keywords} keywords · {limits.websites} site{limits.websites === 1 ? "" : "s"} · {limits.competitors_per_campaign} competitors / campaign
                </p>
              </div>
              {campaign && (
                <button type="button" className="text-xs text-slate-500 hover:text-slate-800" onClick={() => setShowSetup(false)}>Close</button>
              )}
            </div>
            <div className="p-5 grid grid-cols-1 md:grid-cols-2 gap-4">
              <label className="text-xs font-semibold text-slate-600 space-y-1">
                <span>Domain</span>
                <Input value={domain} onChange={(e) => setDomain(e.target.value)} placeholder="example.com" className="h-9 rounded-md" />
              </label>
              <label className="text-xs font-semibold text-slate-600 space-y-1">
                <span>Location</span>
                <Input value={locationName} onChange={(e) => setLocationName(e.target.value)} className="h-9 rounded-md" />
              </label>
              <label className="text-xs font-semibold text-slate-600 space-y-1">
                <span>Device</span>
                <select value={device} onChange={(e) => setDevice(e.target.value)} className="h-9 w-full rounded-md border border-slate-200 px-3 text-sm">
                  <option value="desktop">Desktop</option>
                  <option value="mobile">Mobile</option>
                </select>
              </label>
              <label className="text-xs font-semibold text-slate-600 space-y-1">
                <span>Search engine</span>
                <select value={engine} onChange={(e) => setEngine(e.target.value)} className="h-9 w-full rounded-md border border-slate-200 px-3 text-sm">
                  <option value="google">Google</option>
                  <option value="bing">Bing</option>
                </select>
              </label>
              <label className="md:col-span-2 text-xs font-semibold text-slate-600 space-y-1">
                <span>Keywords (one per line{limits.tags ? ", optional | tag1, tag2" : ""})</span>
                <Textarea value={keywordText} onChange={(e) => setKeywordText(e.target.value)} rows={7} className="rounded-md font-mono text-[13px]" placeholder={"best coffee beans\ncoffee grinder"} />
              </label>
              <label className="md:col-span-2 text-xs font-semibold text-slate-600 space-y-1">
                <span>Competitors (one domain per line)</span>
                <Textarea value={competitorText} onChange={(e) => setCompetitorText(e.target.value)} rows={3} className="rounded-md font-mono text-[13px]" placeholder="rival.com" />
              </label>
            </div>
            <div className="px-5 py-3 border-t border-slate-200 flex justify-end gap-2 bg-[#fafbfc]">
              {campaign && (
                <Button variant="outline" className="h-8 rounded-md text-xs" onClick={() => setShowSetup(false)}>Cancel</Button>
              )}
              <Button
                onClick={handleSave}
                disabled={saving || !projectId}
                className="h-8 rounded-md text-xs text-white"
                style={{ background: ORANGE }}
              >
                {saving ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : null}
                {campaign ? "Save changes" : "Start tracking"}
              </Button>
            </div>
          </div>
        </div>
      )}

      {campaign && !showSetup && (
        <Tabs value={tab} onValueChange={setTab} className="space-y-0">
          <TabsList className="flex w-full h-auto flex-wrap justify-start gap-0 rounded-none bg-white p-0 border-b border-slate-200">
            {TABS.map((t) => {
              const locked = isLocked(reports, (t as any).lockKey);
              return (
                <TabsTrigger
                  key={t.id}
                  value={t.id}
                  className="rounded-none border-b-2 border-transparent bg-transparent px-3 py-2.5 text-[13px] font-medium text-slate-500 shadow-none data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:text-[#ff622d] data-[state=active]:border-[#ff622d] hover:text-slate-800"
                >
                  <span className="inline-flex items-center gap-1">
                    {t.label}
                    {locked && <Lock className="w-3 h-3 opacity-60" />}
                  </span>
                </TabsTrigger>
              );
            })}
          </TabsList>

          <TabsContent value="overview" className="mt-0 p-4 space-y-4 bg-[#f7f8fa]">
            <div className="grid grid-cols-2 xl:grid-cols-4 gap-px bg-slate-200 border border-slate-200">
              <Kpi label="Visibility" value={vis == null ? "—" : `${vis.toFixed(1)}%`} hint="Avg of max(0, 100 − position)" bar={vis} />
              <Kpi label="Avg. position" value={dash(overview.avg_position, 1)} hint="Ranked keywords in this snapshot" bar={overview.avg_position ? Math.max(0, 100 - Number(overview.avg_position)) : null} />
              <Kpi label="Estimated traffic" value="—" hint="Semrush ETV is not computed" bar={null} />
              <Kpi label="Keywords" value={`${overview.ranked ?? 0} / ${overview.tracked ?? 0}`} hint="Ranked / tracked" bar={overview.tracked ? ((overview.ranked || 0) / overview.tracked) * 100 : null} />
            </div>

            <div className="border border-slate-200 bg-white">
              <div className="px-4 py-2.5 border-b border-slate-200 flex items-center justify-between">
                <p className="text-sm font-semibold text-slate-800">Rankings distribution</p>
                <p className="text-[11px] text-slate-400">Current snapshot · daily history is not stored yet</p>
              </div>
              <div className="h-56 px-2 py-3">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={distChart} barGap={2} barCategoryGap="28%">
                    <CartesianGrid stroke="#eef0f3" vertical={false} />
                    <XAxis dataKey="bucket" tick={{ fontSize: 11, fill: "#6b7280" }} axisLine={{ stroke: "#e5e7eb" }} tickLine={false} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: "#6b7280" }} axisLine={false} tickLine={false} width={28} />
                    <Tooltip
                      cursor={{ fill: "rgba(255,98,45,0.06)" }}
                      contentStyle={{ border: "1px solid #e5e7eb", fontSize: 12, borderRadius: 0 }}
                    />
                    <Bar dataKey="You" fill={ORANGE} radius={[1, 1, 0, 0]} maxBarSize={22} />
                    {competitors.map((d, i) => (
                      <Bar key={d} dataKey={d} fill={COMP_COLORS[i % COMP_COLORS.length]} radius={[1, 1, 0, 0]} maxBarSize={22} />
                    ))}
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>

            {reports.share_of_voice && !reports.share_of_voice.locked && (
              <div className="border border-slate-200 bg-white">
                <div className="px-4 py-2.5 border-b border-slate-200">
                  <p className="text-sm font-semibold text-slate-800">Share of voice</p>
                  <p className="text-[11px] text-slate-400">{reports.share_of_voice.honesty}</p>
                </div>
                <div className="p-4 space-y-2">
                  {Object.keys(reports.share_of_voice.shares || {}).length === 0 ? (
                    <p className="text-sm text-slate-500">No ranked keywords in this snapshot.</p>
                  ) : Object.entries(reports.share_of_voice.shares as Record<string, number>).map(([d, v]) => (
                    <div key={d} className="flex items-center gap-3 text-sm">
                      <span className="w-40 truncate font-medium text-slate-700">{d}</span>
                      <div className="flex-1 h-2 bg-slate-100">
                        <div className="h-full" style={{ width: `${Math.min(100, v * 100)}%`, background: ORANGE }} />
                      </div>
                      <span className="w-14 text-right tabular-nums text-slate-600">{(v * 100).toFixed(1)}%</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div>
              <div className="px-1 pb-2 flex items-center justify-between">
                <p className="text-sm font-semibold text-slate-800">Rankings overview</p>
                <button type="button" className="text-xs font-medium" style={{ color: ORANGE }} onClick={() => setTab("rankings")}>
                  Open Rankings →
                </button>
              </div>
              <DenseTable
                empty="No ranking rows yet. Click Update to query Google or Bing."
                columns={[
                  { key: "keyword", label: "Keyword" },
                  { key: "pos", label: "Pos.", align: "center" },
                  { key: "url", label: "Landing URL" },
                  { key: "features", label: "SERP features" },
                  ...competitors.map((d) => ({ key: `c:${d}`, label: d, align: "center" as const })),
                ]}
                rows={(reports.landscape || []).slice(0, 25)}
                render={(row, col) => {
                  if (col === "keyword") return <span className="font-medium text-slate-900">{row.keyword}</span>;
                  if (col === "pos") return <PosCell pos={row.target_position} />;
                  if (col === "url") {
                    const match = (reports.rankings || []).find((k: any) => k.keyword === row.keyword);
                    const url = match?.url;
                    return url ? <span className="text-[#2563eb] truncate max-w-[280px] inline-block">{url}</span> : <span className="text-slate-400">—</span>;
                  }
                  if (col === "features") {
                    const match = (reports.rankings || []).find((k: any) => k.keyword === row.keyword);
                    const feats = (match?.serp_features || []).map(featureLabel).filter(Boolean).slice(0, 4);
                    if (!feats.length) return <span className="text-slate-400">—</span>;
                    return (
                      <span className="inline-flex gap-1 flex-wrap">
                        {feats.map((f: string) => (
                          <span key={f} className="text-[10px] font-semibold px-1 py-0.5 border border-orange-200 text-orange-700 bg-orange-50">{f}</span>
                        ))}
                      </span>
                    );
                  }
                  if (col.startsWith("c:")) {
                    const d = col.slice(2);
                    return <PosCell pos={row.by_domain?.[d] || 0} />;
                  }
                  return "—";
                }}
              />
            </div>
            <p className="text-[11px] text-slate-400">{overview.honesty}</p>
          </TabsContent>

          <TabsContent value="rankings" className="mt-0 p-4 bg-[#f7f8fa] space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <div className="relative">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400" />
                <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Filter keywords" className="pl-8 h-8 w-56 rounded-md text-xs bg-white" />
              </div>
              {(["all", "top3", "top10", "top30", "lost"] as const).map((f) => (
                <button
                  key={f}
                  type="button"
                  onClick={() => setPosFilter(f)}
                  className={`h-8 px-2.5 text-xs border ${
                    posFilter === f ? "border-orange-400 text-orange-700 bg-orange-50" : "border-slate-200 bg-white text-slate-600"
                  }`}
                >
                  {f === "all" ? "All" : f === "lost" ? "Not ranking" : f === "top3" ? "Top 3" : f === "top10" ? "Top 10" : "Top 30"}
                </button>
              ))}
              <span className="ml-auto text-[11px] text-slate-400">{rankingRows.length} keywords</span>
            </div>
            <DenseTable
              empty="No keyword positions in this view. Save keywords and click Update."
              columns={[
                { key: "keyword", label: "Keyword" },
                { key: "pos", label: "Pos.", align: "center" },
                { key: "url", label: "Landing URL" },
                { key: "source", label: "Source" },
                { key: "impr", label: "GSC impr.", align: "right" },
                { key: "clicks", label: "GSC clicks", align: "right" },
                { key: "features", label: "SERP features" },
              ]}
              rows={rankingRows}
              render={(kw, col) => {
                if (col === "keyword") {
                  return (
                    <span>
                      <span className="font-medium text-slate-900">{kw.keyword}</span>
                      {(kw.tags || []).length > 0 && (
                        <span className="ml-2 text-[10px] text-slate-400">{kw.tags.join(", ")}</span>
                      )}
                    </span>
                  );
                }
                if (col === "pos") return <PosCell pos={kw.position} />;
                if (col === "url") return kw.url ? <span className="text-[#2563eb] truncate max-w-[320px] inline-block">{kw.url}</span> : <span className="text-slate-400">—</span>;
                if (col === "source") return <span className="text-xs text-slate-500">{kw.source || "—"}</span>;
                if (col === "impr") return <span className="tabular-nums">{kw.gsc_impressions > 0 ? Number(kw.gsc_impressions).toLocaleString() : "—"}</span>;
                if (col === "clicks") return <span className="tabular-nums">{kw.gsc_clicks > 0 ? Number(kw.gsc_clicks).toLocaleString() : "—"}</span>;
                if (col === "features") {
                  const feats = (kw.serp_features || []).map(featureLabel).filter(Boolean).slice(0, 5);
                  if (!feats.length) return <span className="text-slate-400">—</span>;
                  return (
                    <span className="inline-flex gap-1 flex-wrap">
                      {feats.map((f: string) => (
                        <span key={f} className="text-[10px] font-semibold px-1 py-0.5 border border-orange-200 text-orange-700 bg-orange-50">{f}</span>
                      ))}
                    </span>
                  );
                }
                return "—";
              }}
            />
            <p className="text-[11px] text-slate-400">Position 0 is —. GSC impressions are not search volume.</p>
          </TabsContent>

          <TabsContent value="distribution" className="mt-0 p-4 bg-[#f7f8fa] space-y-3">
            <div className="border border-slate-200 bg-white">
              <div className="px-4 py-2.5 border-b border-slate-200">
                <p className="text-sm font-semibold text-slate-800">Keywords in Top 3 / 10 / 30 / 100</p>
                <p className="text-[11px] text-slate-400">Compared with competitors in this campaign from the current snapshot.</p>
              </div>
              <div className="h-64 px-2 py-3">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={distChart} barGap={2}>
                    <CartesianGrid stroke="#eef0f3" vertical={false} />
                    <XAxis dataKey="bucket" tick={{ fontSize: 11, fill: "#6b7280" }} axisLine={{ stroke: "#e5e7eb" }} tickLine={false} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: "#6b7280" }} axisLine={false} tickLine={false} width={28} />
                    <Tooltip contentStyle={{ border: "1px solid #e5e7eb", fontSize: 12, borderRadius: 0 }} />
                    <Bar dataKey="You" fill={ORANGE} maxBarSize={26} />
                    {competitors.map((d, i) => (
                      <Bar key={d} dataKey={d} fill={COMP_COLORS[i % COMP_COLORS.length]} maxBarSize={26} />
                    ))}
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
            <DenseTable
              empty="No distribution rows yet."
              columns={[
                { key: "domain", label: "Domain" },
                { key: "top3", label: "Top 3", align: "right" },
                { key: "top10", label: "Top 10", align: "right" },
                { key: "top30", label: "Top 30", align: "right" },
                { key: "top100", label: "Top 100", align: "right" },
              ]}
              rows={[
                { domain: campaign.domain, top3: distChart[0]?.You, top10: distChart[1]?.You, top30: distChart[2]?.You, top100: distChart[3]?.You },
                ...competitors.map((d) => ({
                  domain: d,
                  top3: distChart[0]?.[d] ?? 0,
                  top10: distChart[1]?.[d] ?? 0,
                  top30: distChart[2]?.[d] ?? 0,
                  top100: distChart[3]?.[d] ?? 0,
                })),
              ]}
              render={(row, col) =>
                col === "domain"
                  ? <span className="font-medium">{row.domain}</span>
                  : <span className="tabular-nums">{row[col] ?? 0}</span>
              }
            />
          </TabsContent>

          <TabsContent value="pages" className="mt-0 p-4 bg-[#f7f8fa]">
            <DenseTable
              empty="No ranking URLs in this snapshot."
              columns={[
                { key: "url", label: "Landing page" },
                { key: "keywords", label: "Keywords", align: "right" },
                { key: "best", label: "Best pos.", align: "center" },
              ]}
              rows={reports.pages ?? []}
              render={(row, col) => {
                if (col === "url") return <span className="text-[#2563eb]">{row.url}</span>;
                if (col === "keywords") return <span className="tabular-nums">{row.keywords}</span>;
                return <PosCell pos={row.best_position} />;
              }}
            />
          </TabsContent>

          <TabsContent value="tags" className="mt-0 p-4 bg-[#f7f8fa]">
            {reports.tags?.locked ? (
              <Locked plan={reports.tags.plan_needed} reason={reports.tags.reason} />
            ) : (
              <DenseTable
                empty="No tags on tracked keywords. Add tags as keyword | tag on save."
                columns={[
                  { key: "tag", label: "Tag" },
                  { key: "keywords", label: "Keywords", align: "right" },
                  { key: "ranked", label: "Ranked", align: "right" },
                ]}
                rows={reports.tags?.rows ?? []}
                render={(row, col) => col === "tag" ? <span className="font-medium">{row.tag}</span> : <span className="tabular-nums">{row[col]}</span>}
              />
            )}
          </TabsContent>

          <TabsContent value="devices" className="mt-0 p-4 bg-[#f7f8fa]">
            {reports.devices_locations?.locked ? (
              <Locked plan={reports.devices_locations.plan_needed} reason={reports.devices_locations.reason} />
            ) : (
              <div className="border border-slate-200 bg-white">
                <div className="px-4 py-2.5 border-b border-slate-200">
                  <p className="text-sm font-semibold text-slate-800">Devices & locations</p>
                  <p className="text-[11px] text-slate-400">{reports.devices_locations?.note}</p>
                </div>
                <table className="w-full text-[13px]">
                  <thead>
                    <tr className="bg-[#f4f5f7] border-b border-slate-200">
                      <th className="text-left px-3 py-2 text-[11px] font-semibold text-slate-500 uppercase">Location</th>
                      <th className="text-left px-3 py-2 text-[11px] font-semibold text-slate-500 uppercase">Device</th>
                      <th className="text-left px-3 py-2 text-[11px] font-semibold text-slate-500 uppercase">Engine</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr className="border-b border-slate-100">
                      <td className="px-3 py-2 font-medium">{campaign.location_name}</td>
                      <td className="px-3 py-2 capitalize">{campaign.device}</td>
                      <td className="px-3 py-2 capitalize">{campaign.engine}</td>
                    </tr>
                    {(reports.devices_locations?.targets || []).map((t: any, i: number) => (
                      <tr key={i} className="border-b border-slate-100">
                        <td className="px-3 py-2">{t.location_name}</td>
                        <td className="px-3 py-2 capitalize">{t.device}</td>
                        <td className="px-3 py-2 capitalize">{t.engine}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </TabsContent>

          <TabsContent value="landscape" className="mt-0 p-4 bg-[#f7f8fa]">
            <DenseTable
              empty="No landscape rows yet."
              columns={[
                { key: "keyword", label: "Keyword" },
                { key: "you", label: campaign.domain || "You", align: "center" },
                ...competitors.map((d) => ({ key: d, label: d, align: "center" as const })),
              ]}
              rows={reports.landscape ?? []}
              render={(row, col) => {
                if (col === "keyword") return <span className="font-medium">{row.keyword}</span>;
                if (col === "you") return <PosCell pos={row.target_position} />;
                return <PosCell pos={row.by_domain?.[col] || 0} />;
              }}
            />
          </TabsContent>

          <TabsContent value="discovery" className="mt-0 p-4 bg-[#f7f8fa]">
            <DenseTable
              empty="No other domains observed in this snapshot."
              columns={[
                { key: "n", label: "#" },
                { key: "domain", label: "Discovered domain" },
              ]}
              rows={(reports.competitors_discovery?.domains || []).map((d: string, i: number) => ({ n: i + 1, domain: d }))}
              render={(row, col) => col === "n" ? <span className="text-slate-400">{row.n}</span> : <span className="font-medium">{row.domain}</span>}
            />
            <p className="text-[11px] text-slate-400 mt-2">{reports.competitors_discovery?.honesty}</p>
          </TabsContent>

          <TabsContent value="cannibalization" className="mt-0 p-4 bg-[#f7f8fa]">
            {reports.cannibalization?.locked ? (
              <Locked plan={reports.cannibalization.plan_needed} reason={reports.cannibalization.reason} />
            ) : (
              <DenseTable
                empty="No keyword is ranking with more than one URL in this snapshot."
                columns={[
                  { key: "keyword", label: "Keyword" },
                  { key: "urls", label: "Ranking URLs" },
                ]}
                rows={reports.cannibalization?.rows ?? []}
                render={(row, col) =>
                  col === "keyword"
                    ? <span className="font-medium">{row.keyword}</span>
                    : <span className="text-[12px] text-slate-600">{(row.urls || []).map((u: any) => `${u.url} (${u.position > 0 ? "#" + u.position : "—"})`).join("  ·  ") || "—"}</span>
                }
              />
            )}
          </TabsContent>

          <TabsContent value="snippets" className="mt-0 p-4 bg-[#f7f8fa]">
            <DenseTable
              empty="No featured snippets in this snapshot."
              columns={[
                { key: "keyword", label: "Keyword" },
                { key: "owner", label: "Owner" },
                { key: "url", label: "URL" },
                { key: "yours", label: "Yours?", align: "center" },
              ]}
              rows={reports.featured_snippets ?? []}
              render={(row, col) => {
                if (col === "keyword") return <span className="font-medium">{row.keyword}</span>;
                if (col === "owner") return row.owner || "—";
                if (col === "url") return <span className="text-[#2563eb]">{row.url || "—"}</span>;
                return row.is_target ? <span className="text-emerald-600 font-semibold">Yes</span> : <span className="text-slate-400">No</span>;
              }}
            />
          </TabsContent>

          <TabsContent value="ai" className="mt-0 p-4 bg-[#f7f8fa]">
            {reports.ai_search?.locked ? (
              <Locked plan={reports.ai_search.plan_needed} reason={reports.ai_search.reason} />
            ) : (
              <div className="border border-slate-200 bg-white px-6 py-16 text-center">
                <Sparkles className="w-6 h-6 text-orange-400 mx-auto mb-3" />
                <p className="text-sm font-semibold text-slate-800">AI Search prompt tracking is not connected</p>
                <p className="text-xs text-slate-500 mt-1 max-w-lg mx-auto">
                  {reports.ai_search?.reason || data?.honesty?.ai_search}
                </p>
                <Link href="/seo/ai-visibility" className="inline-flex mt-4 h-8 items-center px-4 text-xs font-semibold text-white" style={{ background: ORANGE }}>
                  Open AI Visibility
                </Link>
              </div>
            )}
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}
