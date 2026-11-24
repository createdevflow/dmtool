"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import Link from "next/link";
import {
  Bar, BarChart, CartesianGrid, Cell, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from "recharts";
import {
  AlertCircle, ChevronDown, Download, Globe, Loader2, Lock, MapPin, Monitor,
  RefreshCw, Search, Settings2, Smartphone, Sparkles, Tag, Users,
} from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";

const ORANGE = "#ff642d";
const PAGE_BG = "#f4f5f7";
const COMP_COLORS = ["#ff642d", "#2563eb", "#7c3aed", "#059669", "#d97706", "#0ea5e9"];

type SetupSection = "settings" | "keywords" | "competitors";
type MetricKey = "visibility" | "avg" | "etv" | "keywords";

function dash(v: unknown, digits = 0): string {
  if (v == null || v === "") return "—";
  const n = Number(v);
  if (!Number.isFinite(n)) return "—";
  return digits > 0 ? n.toFixed(digits) : String(Math.round(n));
}

function posTone(pos: number) {
  if (pos <= 0) return { color: "#94a3b8", bg: "#f1f5f9" };
  if (pos <= 3) return { color: "#047857", bg: "#d1fae5" };
  if (pos <= 10) return { color: "#c2410c", bg: "#ffedd5" };
  if (pos <= 30) return { color: "#b45309", bg: "#fef3c7" };
  return { color: "#475569", bg: "#f1f5f9" };
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

function uniqueTags(keywords: any[]): string[] {
  const seen = new Set<string>();
  for (const k of keywords || []) {
    for (const t of k.tags || []) {
      const s = String(t).trim();
      if (s) seen.add(s);
    }
  }
  return [...seen];
}

function visOf(pos: number) {
  if (!(pos > 0)) return 0;
  return Math.max(0, 100 - pos);
}

function domainStats(landscape: any[], domain: string, isYou: boolean) {
  let visSum = 0;
  let posSum = 0;
  let ranked = 0;
  let top3 = 0;
  let top10 = 0;
  let top30 = 0;
  let top100 = 0;
  const tracked = landscape.length;
  for (const row of landscape) {
    const p = isYou ? Number(row.target_position) || 0 : Number(row.by_domain?.[domain]) || 0;
    if (p > 0) {
      visSum += visOf(p);
      posSum += p;
      ranked += 1;
      top100 += 1;
      if (p <= 3) top3 += 1;
      if (p <= 10) top10 += 1;
      if (p <= 30) top30 += 1;
    }
  }
  return {
    domain,
    visibility: ranked ? visSum / ranked : null as number | null,
    avgPosition: ranked ? posSum / ranked : null as number | null,
    ranked,
    tracked,
    top3,
    top10,
    top30,
    top100,
  };
}

function ToolBtn({
  children, onClick, active, disabled, title,
}: {
  children: ReactNode;
  onClick?: () => void;
  active?: boolean;
  disabled?: boolean;
  title?: string;
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className={`inline-flex h-8 items-center gap-1.5 px-2.5 text-[12px] font-medium border disabled:opacity-50 ${
        active
          ? "border-[#ff642d] text-[#c2410c] bg-[#fff4f0]"
          : "border-slate-200 bg-white text-slate-700 hover:bg-slate-50"
      }`}
    >
      {children}
    </button>
  );
}

function CountChip({ label, count, icon, onClick }: { label: string; count: number; icon: ReactNode; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={`${label} ${count}`}
      className="inline-flex h-8 items-center gap-1.5 pl-2.5 pr-1.5 text-[12px] font-medium border border-slate-200 bg-white text-slate-700 hover:bg-slate-50"
    >
      {icon}
      {label}
      <span className="ml-0.5 min-w-[18px] h-[18px] px-1 inline-flex items-center justify-center text-[11px] font-semibold bg-slate-100 text-slate-700">
        {count}
      </span>
    </button>
  );
}

function PosCell({ pos }: { pos: number }) {
  if (!(pos > 0)) return <span className="text-slate-400">—</span>;
  const t = posTone(pos);
  return (
    <span
      className="inline-flex min-w-[28px] h-6 px-1.5 items-center justify-center text-[12px] font-semibold tabular-nums"
      style={{ color: t.color, background: t.bg }}
    >
      {pos}
    </span>
  );
}

function SerpChips({ features }: { features?: string[] }) {
  const feats = (features || []).map(featureLabel).filter(Boolean).slice(0, 5);
  if (!feats.length) return <span className="text-slate-400">—</span>;
  return (
    <span className="inline-flex gap-0.5 flex-wrap">
      {feats.map((f) => (
        <span
          key={f}
          className="inline-flex h-[18px] min-w-[22px] items-center justify-center px-1 text-[9px] font-bold tracking-wide text-[#c2410c] bg-[#fff4f0] border border-[#ffd8c8]"
        >
          {f}
        </span>
      ))}
    </span>
  );
}

function Locked({ plan, reason }: { plan?: string; reason?: string }) {
  return (
    <div className="relative overflow-hidden border border-slate-200 bg-white min-h-[320px]">
      <div
        className="absolute inset-0 opacity-50 pointer-events-none"
        style={{
          backgroundImage:
            "linear-gradient(#e8eaed 1px, transparent 1px), linear-gradient(90deg, #e8eaed 1px, transparent 1px)",
          backgroundSize: "22px 26px",
        }}
      />
      <div className="relative flex flex-col items-center justify-center py-24 px-6 text-center">
        <div className="w-10 h-10 bg-white border border-slate-200 flex items-center justify-center mb-3 shadow-sm">
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
  columns: { key: string; label: string; align?: "left" | "right" | "center"; width?: string }[];
  rows: any[];
  empty: string;
  render: (row: any, col: string) => ReactNode;
}) {
  return (
    <div className="border border-slate-200 bg-white">
      {rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-14 text-center">
          <AlertCircle className="w-4 h-4 text-slate-300 mb-2" />
          <p className="text-[13px] text-slate-500">{empty}</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-[13px] border-collapse">
            <thead className="sticky top-0 z-10">
              <tr className="bg-[#f8f9fb] border-b border-slate-200">
                {columns.map((c) => (
                  <th
                    key={c.key}
                    style={c.width ? { width: c.width } : undefined}
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
                <tr key={i} className="border-b border-slate-100 hover:bg-[#fff7f3]">
                  {columns.map((c) => (
                    <td
                      key={c.key}
                      className={`px-3 py-[7px] align-middle ${
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

function FilterBar({
  search, setSearch, posFilter, setPosFilter, count,
}: {
  search: string;
  setSearch: (v: string) => void;
  posFilter: "all" | "top3" | "top10" | "top30" | "lost";
  setPosFilter: (v: "all" | "top3" | "top10" | "top30" | "lost") => void;
  count: number;
}) {
  return (
    <div className="flex flex-wrap items-center gap-1.5 px-3 py-2 bg-white border border-b-0 border-slate-200">
      <div className="relative">
        <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400" />
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Filter keywords"
          className="pl-7 pr-2 h-7 w-52 text-[12px] border border-slate-200 bg-white outline-none focus:border-[#ff642d]"
        />
      </div>
      {([
        ["all", "All"],
        ["top3", "Top 3"],
        ["top10", "Top 10"],
        ["top30", "Top 30"],
        ["lost", "Not ranking"],
      ] as const).map(([id, label]) => (
        <button
          key={id}
          type="button"
          onClick={() => setPosFilter(id)}
          className={`h-7 px-2 text-[12px] border ${
            posFilter === id ? "border-[#ff642d] text-[#c2410c] bg-[#fff4f0]" : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
          }`}
        >
          {label}
        </button>
      ))}
      <span className="ml-auto text-[11px] text-slate-500 tabular-nums">{count} keywords</span>
    </div>
  );
}

export function PositionTrackingPanel({
  projectId,
  defaultDomain,
  projects = [],
  project,
  onProjectChange,
}: {
  projectId?: number;
  defaultDomain?: string;
  projects?: any[];
  project?: any;
  onProjectChange?: (p: any) => void;
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
  const [setupSection, setSetupSection] = useState<SetupSection>("settings");
  const [metric, setMetric] = useState<MetricKey>("visibility");
  const [chartDomains, setChartDomains] = useState<string[]>([]);

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
        setChartDomains([camp.domain, ...(camp.competitors || []).slice(0, 4)]);
      } else {
        setDomain(defaultDomain || "");
        setShowSetup(true);
        setSetupSection("settings");
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
  const landscape = reports.landscape ?? [];
  const tags = uniqueTags(campaign?.keywords || []);

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

  const statsByDomain = useMemo(() => {
    const rows: Record<string, ReturnType<typeof domainStats>> = {};
    if (campaign?.domain) rows[campaign.domain] = domainStats(landscape, campaign.domain, true);
    for (const d of competitors) rows[d] = domainStats(landscape, d, false);
    return rows;
  }, [landscape, campaign?.domain, competitors]);

  const distChart = useMemo(() => {
    const you = campaign?.domain ? statsByDomain[campaign.domain] : null;
    const rows = [
      { bucket: "Top 3", You: you?.top3 ?? 0 },
      { bucket: "Top 10", You: you?.top10 ?? 0 },
      { bucket: "Top 30", You: you?.top30 ?? 0 },
      { bucket: "Top 100", You: you?.top100 ?? 0 },
    ] as any[];
    for (const d of competitors) {
      rows[0][d] = statsByDomain[d]?.top3 ?? 0;
      rows[1][d] = statsByDomain[d]?.top10 ?? 0;
      rows[2][d] = statsByDomain[d]?.top30 ?? 0;
      rows[3][d] = statsByDomain[d]?.top100 ?? 0;
    }
    return rows;
  }, [statsByDomain, competitors, campaign?.domain]);

  const snapshotChart = useMemo(() => {
    const names = chartDomains.length ? chartDomains : (campaign?.domain ? [campaign.domain, ...competitors.slice(0, 4)] : []);
    return names.map((d) => {
      const s = statsByDomain[d];
      let value: number | null = null;
      if (metric === "visibility") value = s?.visibility ?? null;
      else if (metric === "avg") value = s?.avgPosition ?? null;
      else if (metric === "keywords") value = s?.ranked ?? 0;
      else value = null;
      return { name: d, value: value == null ? 0 : Number(value), missing: value == null && metric !== "keywords" };
    });
  }, [chartDomains, statsByDomain, metric, campaign?.domain, competitors]);

  const discoveryMap = useMemo(() => {
    const by: Record<string, { domain: string; keywords: number; posSum: number; ranked: number }> = {};
    for (const kw of reports.rankings ?? []) {
      const seen = new Set<string>();
      const add = (r: any) => {
        const d = String(r?.domain || "").toLowerCase();
        if (!d || seen.has(d)) return;
        seen.add(d);
        if (!by[d]) by[d] = { domain: d, keywords: 0, posSum: 0, ranked: 0 };
        by[d].keywords += 1;
        if (r.position > 0) {
          by[d].posSum += r.position;
          by[d].ranked += 1;
        }
      };
      (kw.discovered || []).forEach(add);
      (kw.competitors || []).forEach(add);
    }
    return Object.values(by).sort((a, b) => b.keywords - a.keywords).slice(0, 20);
  }, [reports.rankings]);

  const openSetup = (section: SetupSection) => {
    setSetupSection(section);
    setShowSetup(true);
  };

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

  const toggleChartDomain = (d: string) => {
    setChartDomains((prev) => {
      const has = prev.includes(d);
      if (has) {
        if (prev.length === 1) return prev;
        return prev.filter((x) => x !== d);
      }
      if (prev.length >= 5) return prev;
      return [...prev, d];
    });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[50vh]">
        <Loader2 className="w-6 h-6 animate-spin" style={{ color: ORANGE }} />
      </div>
    );
  }

  const DeviceIcon = device === "mobile" ? Smartphone : Monitor;
  const vis = overview.visibility == null ? null : Number(overview.visibility);
  const kwCount = (campaign?.keywords || []).length;

  const metricCards: { key: MetricKey; label: string; value: string; hint: string }[] = [
    { key: "visibility", label: "Visibility", value: vis == null ? "—" : `${vis.toFixed(1)}%`, hint: "Avg of max(0, 100 − position)" },
    { key: "avg", label: "Avg. position", value: dash(overview.avg_position, 1), hint: "Ranked keywords in this snapshot" },
    { key: "etv", label: "Estimated traffic", value: "—", hint: "Semrush ETV is not computed" },
    { key: "keywords", label: "Keywords", value: `${overview.ranked ?? 0} / ${overview.tracked ?? 0}`, hint: "Ranked / tracked" },
  ];

  return (
    <div className="min-h-[calc(100vh-4rem)]" style={{ background: PAGE_BG }}>
      <div className="sticky top-16 z-20 bg-white border-b border-slate-200">
        <div className="px-4 lg:px-5 py-2.5 flex flex-col xl:flex-row xl:items-center gap-2.5">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="inline-block w-1.5 h-4" style={{ background: ORANGE }} />
              <h1 className="text-[16px] font-semibold text-slate-900 leading-none">Position Tracking</h1>
              <span className="text-[10px] font-semibold uppercase tracking-wider px-1.5 py-0.5 border border-slate-200 text-slate-500">
                {limits.plan_label || "Free"}
              </span>
              {projects.length > 0 && (
                <label className="relative inline-flex items-center">
                  <select
                    value={project?.id || ""}
                    onChange={(e) => {
                      const p = projects.find((x: any) => String(x.id) === e.target.value);
                      if (p) onProjectChange?.(p);
                    }}
                    className="h-7 appearance-none pl-2 pr-6 text-[12px] font-medium border border-slate-200 bg-white text-slate-700"
                  >
                    {projects.map((p: any) => (
                      <option key={p.id} value={p.id}>{p.name || p.url || `Project ${p.id}`}</option>
                    ))}
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-1.5 w-3 h-3 text-slate-400" />
                </label>
              )}
            </div>
            {campaign ? (
              <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-[12px] text-slate-600">
                <span className="font-semibold text-slate-900">{campaign.domain}</span>
                <span className="inline-flex items-center gap-1"><Globe className="w-3 h-3 text-slate-400" />{campaign.engine}</span>
                <span className="inline-flex items-center gap-1"><MapPin className="w-3 h-3 text-slate-400" />{campaign.location_name}</span>
                <span className="inline-flex items-center gap-1"><DeviceIcon className="w-3 h-3 text-slate-400" />{campaign.device}</span>
                <span className="text-slate-400">
                  {campaign.snapshot_at ? `Updated ${new Date(campaign.snapshot_at).toLocaleString()}` : "No SERP snapshot yet"}
                </span>
              </div>
            ) : (
              <p className="mt-1.5 text-[12px] text-slate-500">Set up a campaign to track Google or Bing rankings.</p>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            {campaign && (
              <>
                <CountChip label="Keywords" count={kwCount} icon={<Search className="w-3 h-3" />} onClick={() => openSetup("keywords")} />
                <CountChip label="Competitors" count={competitors.length} icon={<Users className="w-3 h-3" />} onClick={() => openSetup("competitors")} />
                <CountChip label="Tags" count={tags.length} icon={<Tag className="w-3 h-3" />} onClick={() => openSetup("keywords")} />
                <ToolBtn onClick={() => openSetup("settings")}><Settings2 className="w-3.5 h-3.5" /> Settings</ToolBtn>
                <ToolBtn onClick={exportCSV} disabled={!(reports.rankings || []).length}><Download className="w-3.5 h-3.5" /> Export</ToolBtn>
                <button
                  type="button"
                  onClick={handleRefresh}
                  disabled={refreshing || !projectId}
                  className="inline-flex h-8 items-center gap-1.5 px-3 text-[12px] font-semibold text-white disabled:opacity-50"
                  style={{ background: ORANGE }}
                >
                  {refreshing ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RefreshCw className="w-3.5 h-3.5" />}
                  Update
                </button>
              </>
            )}
          </div>
        </div>

        {campaign && !showSetup && (
          <div className="px-2 lg:px-3 flex overflow-x-auto">
            {TABS.map((t) => {
              const locked = isLocked(reports, (t as any).lockKey);
              const active = tab === t.id;
              return (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => setTab(t.id)}
                  className={`shrink-0 px-3 py-2.5 text-[13px] font-medium border-b-2 -mb-px ${
                    active
                      ? "border-[#ff642d] text-[#ff642d]"
                      : "border-transparent text-slate-500 hover:text-slate-800"
                  }`}
                >
                  <span className="inline-flex items-center gap-1 whitespace-nowrap">
                    {t.label}
                    {locked && <Lock className="w-3 h-3 opacity-60" />}
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </div>

      {serp?.message && campaign && !showSetup && (
        <div className="px-4 lg:px-5 py-1.5 text-[12px] text-slate-600 bg-[#fff8f5] border-b border-orange-100 flex items-start gap-2">
          <Globe className="w-3.5 h-3.5 mt-0.5 text-orange-500 shrink-0" />
          <span>
            {serp.message}
            {gsc.used_as_fallback ? " GSC is filling empty ranks until a SERP lookup runs." : ""}
          </span>
        </div>
      )}

      {(!campaign || showSetup) && (
        <div className="p-4 lg:p-5">
          <div className="max-w-[880px] border border-slate-200 bg-white">
            <div className="px-4 py-2.5 border-b border-slate-200 flex items-center justify-between bg-[#fafbfc]">
              <div>
                <p className="text-[13px] font-semibold text-slate-900">
                  {campaign
                    ? setupSection === "keywords" ? "Keywords" : setupSection === "competitors" ? "Competitors" : "Campaign settings"
                    : "Set up Position Tracking"}
                </p>
                <p className="text-[11px] text-slate-500 mt-0.5">
                  {limits.plan_label}: {limits.keywords} keywords · {limits.websites} site{limits.websites === 1 ? "" : "s"} · {limits.competitors_per_campaign} competitors / campaign
                </p>
              </div>
              {campaign && (
                <button type="button" className="text-[12px] text-slate-500 hover:text-slate-800" onClick={() => setShowSetup(false)}>Close</button>
              )}
            </div>
            <div className="p-4 grid grid-cols-1 md:grid-cols-2 gap-3">
              {(setupSection === "settings" || !campaign) && (
                <>
                  <label className="text-[11px] font-semibold text-slate-600 space-y-1">
                    <span>Domain</span>
                    <input value={domain} onChange={(e) => setDomain(e.target.value)} placeholder="example.com" className="h-8 w-full px-2.5 text-[13px] border border-slate-200 outline-none focus:border-[#ff642d]" />
                  </label>
                  <label className="text-[11px] font-semibold text-slate-600 space-y-1">
                    <span>Location</span>
                    <input value={locationName} onChange={(e) => setLocationName(e.target.value)} className="h-8 w-full px-2.5 text-[13px] border border-slate-200 outline-none focus:border-[#ff642d]" />
                  </label>
                  <label className="text-[11px] font-semibold text-slate-600 space-y-1">
                    <span>Device</span>
                    <select value={device} onChange={(e) => setDevice(e.target.value)} className="h-8 w-full px-2.5 text-[13px] border border-slate-200 bg-white">
                      <option value="desktop">Desktop</option>
                      <option value="mobile">Mobile</option>
                    </select>
                  </label>
                  <label className="text-[11px] font-semibold text-slate-600 space-y-1">
                    <span>Search engine</span>
                    <select value={engine} onChange={(e) => setEngine(e.target.value)} className="h-8 w-full px-2.5 text-[13px] border border-slate-200 bg-white">
                      <option value="google">Google</option>
                      <option value="bing">Bing</option>
                    </select>
                  </label>
                </>
              )}
              {(setupSection === "keywords" || setupSection === "settings" || !campaign) && (
                <label className="md:col-span-2 text-[11px] font-semibold text-slate-600 space-y-1">
                  <span>Keywords (one per line{limits.tags ? ", optional | tag1, tag2" : ""})</span>
                  <textarea value={keywordText} onChange={(e) => setKeywordText(e.target.value)} rows={7} className="w-full px-2.5 py-2 text-[13px] font-mono border border-slate-200 outline-none focus:border-[#ff642d]" placeholder={"best coffee beans\ncoffee grinder"} />
                </label>
              )}
              {(setupSection === "competitors" || setupSection === "settings" || !campaign) && (
                <label className="md:col-span-2 text-[11px] font-semibold text-slate-600 space-y-1">
                  <span>Competitors (one domain per line)</span>
                  <textarea value={competitorText} onChange={(e) => setCompetitorText(e.target.value)} rows={3} className="w-full px-2.5 py-2 text-[13px] font-mono border border-slate-200 outline-none focus:border-[#ff642d]" placeholder="rival.com" />
                </label>
              )}
            </div>
            <div className="px-4 py-2.5 border-t border-slate-200 flex justify-end gap-2 bg-[#fafbfc]">
              {campaign && (
                <button type="button" className="h-8 px-3 text-[12px] border border-slate-200 bg-white" onClick={() => setShowSetup(false)}>Cancel</button>
              )}
              <button
                type="button"
                onClick={handleSave}
                disabled={saving || !projectId}
                className="h-8 px-3 text-[12px] font-semibold text-white disabled:opacity-50 inline-flex items-center gap-1.5"
                style={{ background: ORANGE }}
              >
                {saving ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : null}
                {campaign ? "Save changes" : "Start tracking"}
              </button>
            </div>
          </div>
        </div>
      )}

      {campaign && !showSetup && (
        <div className="p-3 lg:p-4 space-y-3">
          {tab === "overview" && (
            <>
              <div className="border border-slate-200 bg-white">
                <div className="grid grid-cols-2 xl:grid-cols-4 border-b border-slate-200">
                  {metricCards.map((m) => (
                    <button
                      key={m.key}
                      type="button"
                      onClick={() => m.key !== "etv" && setMetric(m.key)}
                      className={`text-left px-4 py-3 border-r border-slate-200 last:border-r-0 ${
                        metric === m.key ? "bg-white" : "bg-[#fafbfc] hover:bg-white"
                      }`}
                    >
                      <p className="text-[11px] font-medium text-slate-500 uppercase tracking-wide">{m.label}</p>
                      <p className="mt-1 text-[28px] leading-none font-semibold tabular-nums" style={{ color: metric === m.key ? ORANGE : "#0f172a" }}>
                        {m.value}
                      </p>
                      <p className="mt-1.5 text-[11px] text-slate-400">{m.hint}</p>
                      <div className="mt-2 h-0.5" style={{ background: metric === m.key ? ORANGE : "transparent" }} />
                    </button>
                  ))}
                </div>
                <div className="px-4 py-2.5 flex flex-wrap items-center gap-2 border-b border-slate-100">
                  <span className="text-[11px] font-semibold text-slate-500 uppercase tracking-wide">Competitors</span>
                  {[campaign.domain, ...competitors].map((d, i) => {
                    const on = chartDomains.includes(d);
                    return (
                      <button
                        key={d}
                        type="button"
                        onClick={() => toggleChartDomain(d)}
                        className={`inline-flex items-center gap-1.5 h-7 px-2 text-[12px] border ${
                          on ? "border-slate-300 bg-white" : "border-slate-200 bg-slate-50 text-slate-400"
                        }`}
                      >
                        <span className="w-2 h-2 rounded-full" style={{ background: on ? COMP_COLORS[i % COMP_COLORS.length] : "#cbd5e1" }} />
                        {d}
                      </button>
                    );
                  })}
                  <span className="ml-auto text-[11px] text-slate-400">Current snapshot · daily history is not stored yet</span>
                </div>
                <div className="h-[280px] px-2 py-3">
                  {metric === "etv" ? (
                    <div className="h-full flex items-center justify-center text-[13px] text-slate-500">
                      Estimated traffic is not computed. Visibility, average position, and ranked keywords use this snapshot.
                    </div>
                  ) : (
                    <ResponsiveContainer width="100%" height="100%">
                      <BarChart data={snapshotChart} layout="vertical" margin={{ top: 4, right: 16, left: 8, bottom: 4 }}>
                        <CartesianGrid stroke="#eef0f3" horizontal={false} />
                        <XAxis
                          type="number"
                          tick={{ fontSize: 11, fill: "#6b7280" }}
                          axisLine={{ stroke: "#e5e7eb" }}
                          tickLine={false}
                          domain={metric === "visibility" ? [0, 100] : ["auto", "auto"]}
                        />
                        <YAxis
                          type="category"
                          dataKey="name"
                          width={120}
                          tick={{ fontSize: 11, fill: "#334155" }}
                          axisLine={false}
                          tickLine={false}
                        />
                        <Tooltip
                          cursor={{ fill: "rgba(255,100,45,0.06)" }}
                          contentStyle={{ border: "1px solid #e5e7eb", fontSize: 12, borderRadius: 0 }}
                          formatter={(v: any) => {
                            if (metric === "visibility") return [`${Number(v).toFixed(1)}%`, "Visibility"];
                            if (metric === "avg") return [Number(v).toFixed(1), "Avg. position"];
                            return [v, "Ranked keywords"];
                          }}
                        />
                        <Bar dataKey="value" maxBarSize={18} radius={[0, 1, 1, 0]}>
                          {snapshotChart.map((row, i) => (
                            <Cell key={row.name} fill={COMP_COLORS[i % COMP_COLORS.length]} />
                          ))}
                        </Bar>
                      </BarChart>
                    </ResponsiveContainer>
                  )}
                </div>
              </div>

              {reports.share_of_voice && !reports.share_of_voice.locked && (
                <div className="border border-slate-200 bg-white">
                  <div className="px-4 py-2 border-b border-slate-200 flex items-center justify-between">
                    <p className="text-[13px] font-semibold text-slate-800">Share of voice</p>
                    <p className="text-[11px] text-slate-400">{reports.share_of_voice.honesty}</p>
                  </div>
                  <div className="p-4 space-y-2">
                    {Object.keys(reports.share_of_voice.shares || {}).length === 0 ? (
                      <p className="text-[13px] text-slate-500">No ranked keywords in this snapshot.</p>
                    ) : Object.entries(reports.share_of_voice.shares as Record<string, number>).map(([d, v]) => (
                      <div key={d} className="flex items-center gap-3 text-[13px]">
                        <span className="w-44 truncate font-medium text-slate-700">{d}</span>
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
                <div className="px-0.5 pb-1.5 flex items-center justify-between">
                  <p className="text-[13px] font-semibold text-slate-800">Rankings Overview</p>
                  <button type="button" className="text-[12px] font-medium" style={{ color: ORANGE }} onClick={() => setTab("rankings")}>
                    Open Rankings →
                  </button>
                </div>
                <FilterBar search={search} setSearch={setSearch} posFilter={posFilter} setPosFilter={setPosFilter} count={rankingRows.length} />
                <DenseTable
                  empty="No ranking rows yet. Click Update to query Google or Bing."
                  columns={[
                    { key: "keyword", label: "Keyword" },
                    { key: "pos", label: "Pos.", align: "center", width: "64px" },
                    { key: "url", label: "Landing URL" },
                    { key: "features", label: "SERP features" },
                    ...competitors.map((d) => ({ key: `c:${d}`, label: d, align: "center" as const })),
                  ]}
                  rows={(reports.landscape || []).filter((row: any) => {
                    const q = search.trim().toLowerCase();
                    if (q && !String(row.keyword || "").toLowerCase().includes(q)) return false;
                    const p = Number(row.target_position) || 0;
                    if (posFilter === "top3") return p > 0 && p <= 3;
                    if (posFilter === "top10") return p > 0 && p <= 10;
                    if (posFilter === "top30") return p > 0 && p <= 30;
                    if (posFilter === "lost") return p <= 0;
                    return true;
                  }).slice(0, 50)}
                  render={(row, col) => {
                    if (col === "keyword") {
                      const match = (reports.rankings || []).find((k: any) => k.keyword === row.keyword);
                      return (
                        <span>
                          <span className="font-medium text-slate-900">{row.keyword}</span>
                          {(match?.tags || []).length > 0 && (
                            <span className="ml-1.5 text-[10px] text-slate-400">{match.tags.join(", ")}</span>
                          )}
                        </span>
                      );
                    }
                    if (col === "pos") return <PosCell pos={row.target_position} />;
                    if (col === "url") {
                      const match = (reports.rankings || []).find((k: any) => k.keyword === row.keyword);
                      const url = match?.url;
                      return url ? <span className="text-[#2563eb] truncate max-w-[280px] inline-block">{url}</span> : <span className="text-slate-400">—</span>;
                    }
                    if (col === "features") {
                      const match = (reports.rankings || []).find((k: any) => k.keyword === row.keyword);
                      return <SerpChips features={match?.serp_features} />;
                    }
                    if (col.startsWith("c:")) {
                      const d = col.slice(2);
                      return <PosCell pos={row.by_domain?.[d] || 0} />;
                    }
                    return "—";
                  }}
                />
                <p className="mt-1.5 text-[11px] text-slate-400">{overview.honesty}</p>
              </div>
            </>
          )}

          {tab === "rankings" && (
            <>
              <FilterBar search={search} setSearch={setSearch} posFilter={posFilter} setPosFilter={setPosFilter} count={rankingRows.length} />
              <DenseTable
                empty="No keyword positions in this view. Save keywords and click Update."
                columns={[
                  { key: "keyword", label: "Keyword" },
                  { key: "pos", label: "Pos.", align: "center", width: "64px" },
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
                          <span className="ml-1.5 text-[10px] text-slate-400">{kw.tags.join(", ")}</span>
                        )}
                      </span>
                    );
                  }
                  if (col === "pos") return <PosCell pos={kw.position} />;
                  if (col === "url") return kw.url ? <span className="text-[#2563eb] truncate max-w-[320px] inline-block">{kw.url}</span> : <span className="text-slate-400">—</span>;
                  if (col === "source") return <span className="text-[12px] text-slate-500">{kw.source || "—"}</span>;
                  if (col === "impr") return <span className="tabular-nums">{kw.gsc_impressions > 0 ? Number(kw.gsc_impressions).toLocaleString() : "—"}</span>;
                  if (col === "clicks") return <span className="tabular-nums">{kw.gsc_clicks > 0 ? Number(kw.gsc_clicks).toLocaleString() : "—"}</span>;
                  if (col === "features") return <SerpChips features={kw.serp_features} />;
                  return "—";
                }}
              />
              <p className="text-[11px] text-slate-400">Position 0 is —. GSC impressions are not search volume.</p>
            </>
          )}

          {tab === "distribution" && (
            <>
              <div className="border border-slate-200 bg-white">
                <div className="px-4 py-2 border-b border-slate-200 flex items-center justify-between">
                  <p className="text-[13px] font-semibold text-slate-800">Keywords in Top 3 / 10 / 30 / 100</p>
                  <p className="text-[11px] text-slate-400">Current snapshot vs competitors in this campaign</p>
                </div>
                <div className="h-64 px-2 py-3">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={distChart} barGap={2} barCategoryGap="22%">
                      <CartesianGrid stroke="#eef0f3" vertical={false} />
                      <XAxis dataKey="bucket" tick={{ fontSize: 11, fill: "#6b7280" }} axisLine={{ stroke: "#e5e7eb" }} tickLine={false} />
                      <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: "#6b7280" }} axisLine={false} tickLine={false} width={28} />
                      <Tooltip contentStyle={{ border: "1px solid #e5e7eb", fontSize: 12, borderRadius: 0 }} />
                      <Legend wrapperStyle={{ fontSize: 12 }} />
                      <Bar dataKey="You" fill={ORANGE} maxBarSize={22} />
                      {competitors.map((d, i) => (
                        <Bar key={d} dataKey={d} fill={COMP_COLORS[(i + 1) % COMP_COLORS.length]} maxBarSize={22} />
                      ))}
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>
              <DenseTable
                empty="No distribution rows yet."
                columns={[
                  { key: "domain", label: "Domain" },
                  { key: "vis", label: "Visibility", align: "right" },
                  { key: "avg", label: "Avg. pos.", align: "right" },
                  { key: "top3", label: "Top 3", align: "right" },
                  { key: "top10", label: "Top 10", align: "right" },
                  { key: "top30", label: "Top 30", align: "right" },
                  { key: "top100", label: "Top 100", align: "right" },
                ]}
                rows={[
                  campaign.domain,
                  ...competitors,
                ].map((d) => {
                  const s = statsByDomain[d];
                  return {
                    domain: d,
                    vis: s?.visibility,
                    avg: s?.avgPosition,
                    top3: s?.top3 ?? 0,
                    top10: s?.top10 ?? 0,
                    top30: s?.top30 ?? 0,
                    top100: s?.top100 ?? 0,
                    isYou: d === campaign.domain,
                  };
                })}
                render={(row, col) => {
                  if (col === "domain") return <span className={`font-medium ${row.isYou ? "text-slate-900" : "text-slate-700"}`}>{row.domain}{row.isYou ? " (you)" : ""}</span>;
                  if (col === "vis") return <span className="tabular-nums">{row.vis == null ? "—" : `${row.vis.toFixed(1)}%`}</span>;
                  if (col === "avg") return <span className="tabular-nums">{row.avg == null ? "—" : row.avg.toFixed(1)}</span>;
                  return <span className="tabular-nums font-medium">{row[col] ?? 0}</span>;
                }}
              />
            </>
          )}

          {tab === "pages" && (
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
          )}

          {tab === "tags" && (
            reports.tags?.locked ? (
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
            )
          )}

          {tab === "devices" && (
            reports.devices_locations?.locked ? (
              <Locked plan={reports.devices_locations.plan_needed} reason={reports.devices_locations.reason} />
            ) : (
              <div className="border border-slate-200 bg-white">
                <div className="px-4 py-2 border-b border-slate-200">
                  <p className="text-[13px] font-semibold text-slate-800">Devices & locations</p>
                  <p className="text-[11px] text-slate-400">{reports.devices_locations?.note}</p>
                </div>
                <table className="w-full text-[13px]">
                  <thead>
                    <tr className="bg-[#f8f9fb] border-b border-slate-200">
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
            )
          )}

          {tab === "landscape" && (
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
          )}

          {tab === "discovery" && (
            <>
              <div className="border border-slate-200 bg-white">
                <div className="px-4 py-2 border-b border-slate-200 flex items-center justify-between">
                  <p className="text-[13px] font-semibold text-slate-800">Competition map</p>
                  <p className="text-[11px] text-slate-400">Keyword count vs average position in this snapshot</p>
                </div>
                {discoveryMap.length === 0 ? (
                  <div className="py-14 text-center text-[13px] text-slate-500">No other domains observed in this snapshot.</div>
                ) : (
                  <div className="relative h-[280px] m-3 border border-dashed border-slate-200 bg-[#fafbfc]">
                    <span className="absolute left-2 top-2 text-[10px] text-slate-400">Better avg. position ↑</span>
                    <span className="absolute right-2 bottom-2 text-[10px] text-slate-400">More keywords →</span>
                    {discoveryMap.map((d, i) => {
                      const maxKw = Math.max(...discoveryMap.map((x) => x.keywords), 1);
                      const avg = d.ranked ? d.posSum / d.ranked : 50;
                      const x = 8 + (d.keywords / maxKw) * 78;
                      const y = 8 + ((Math.min(avg, 100) - 1) / 99) * 78;
                      const size = 18 + (d.keywords / maxKw) * 22;
                      return (
                        <div
                          key={d.domain}
                          title={`${d.domain} · ${d.keywords} kw · avg ${avg.toFixed(1)}`}
                          className="absolute rounded-full flex items-center justify-center text-[9px] font-semibold text-white overflow-hidden"
                          style={{
                            left: `${x}%`,
                            top: `${y}%`,
                            width: size,
                            height: size,
                            marginLeft: -size / 2,
                            marginTop: -size / 2,
                            background: COMP_COLORS[i % COMP_COLORS.length],
                            opacity: 0.88,
                          }}
                        >
                          {d.domain.slice(0, 2)}
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
              <DenseTable
                empty="No other domains observed in this snapshot."
                columns={[
                  { key: "n", label: "#", width: "40px" },
                  { key: "domain", label: "Discovered domain" },
                  { key: "keywords", label: "Keywords", align: "right" },
                  { key: "avg", label: "Avg. pos.", align: "right" },
                ]}
                rows={discoveryMap.map((d, i) => ({
                  n: i + 1,
                  domain: d.domain,
                  keywords: d.keywords,
                  avg: d.ranked ? d.posSum / d.ranked : null,
                }))}
                render={(row, col) => {
                  if (col === "n") return <span className="text-slate-400">{row.n}</span>;
                  if (col === "domain") return <span className="font-medium">{row.domain}</span>;
                  if (col === "avg") return <span className="tabular-nums">{row.avg == null ? "—" : row.avg.toFixed(1)}</span>;
                  return <span className="tabular-nums">{row.keywords}</span>;
                }}
              />
              <p className="text-[11px] text-slate-400">{reports.competitors_discovery?.honesty}</p>
            </>
          )}

          {tab === "cannibalization" && (
            reports.cannibalization?.locked ? (
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
            )
          )}

          {tab === "snippets" && (
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
          )}

          {tab === "ai" && (
            reports.ai_search?.locked ? (
              <Locked plan={reports.ai_search.plan_needed} reason={reports.ai_search.reason} />
            ) : (
              <div className="border border-slate-200 bg-white px-6 py-16 text-center">
                <Sparkles className="w-6 h-6 mx-auto mb-3" style={{ color: ORANGE }} />
                <p className="text-[13px] font-semibold text-slate-800">AI Search prompt tracking is not connected</p>
                <p className="text-[12px] text-slate-500 mt-1 max-w-lg mx-auto">
                  {reports.ai_search?.reason || data?.honesty?.ai_search}
                </p>
                <Link href="/seo/ai-visibility" className="inline-flex mt-4 h-8 items-center px-4 text-xs font-semibold text-white" style={{ background: ORANGE }}>
                  Open AI Visibility
                </Link>
              </div>
            )
          )}
        </div>
      )}
    </div>
  );
}
