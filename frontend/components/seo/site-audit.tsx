"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import {
  AlertCircle, ArrowLeft, ChevronDown, ChevronLeft, ChevronRight, ClipboardCheck,
  Download, Loader2, Lock, Plus, RefreshCw, Settings2,
} from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";

const ORANGE = "#ff642d";
const PAGE_BG = "#f4f5f7";

type Cell = {
  value?: number | null;
  delta?: number | null;
  status?: string;
  label?: string;
};

const LIST_COLUMNS: { key: string; label: string; sort?: string; align?: "left" | "right" | "center"; width?: string }[] = [
  { key: "project", label: "Project", sort: "project" },
  { key: "last_update", label: "Last Update", sort: "last_update" },
  { key: "pages_crawled", label: "Pages Crawled", sort: "pages_crawled", align: "right" },
  { key: "site_health", label: "Site Health", sort: "site_health", align: "right" },
  { key: "ai_search_health", label: "AI Search Health", align: "right" },
  { key: "errors", label: "Errors", sort: "errors", align: "right" },
  { key: "warnings", label: "Warnings", sort: "warnings", align: "right" },
  { key: "crawlability", label: "Crawlability", sort: "crawlability", align: "right" },
  { key: "https", label: "HTTPS", sort: "https", align: "right" },
  { key: "international", label: "Int. SEO", sort: "international", align: "right" },
  { key: "performance", label: "Site Performance", sort: "performance", align: "right" },
  { key: "linking", label: "Internal Linking", sort: "linking", align: "right" },
  { key: "markups", label: "Markups", sort: "markups", align: "right" },
  { key: "core_web_vitals", label: "Core Web Vitals", align: "right" },
];

const TABS = [
  { id: "overview", label: "Overview" },
  { id: "issues", label: "Issues" },
  { id: "pages", label: "Crawled Pages" },
  { id: "statistics", label: "Statistics" },
  { id: "compare", label: "Compare Crawls", lockKey: "compare" },
] as const;

function healthTone(n: number) {
  if (n >= 80) return { color: "#047857", bg: "#d1fae5" };
  if (n >= 50) return { color: "#b45309", bg: "#fef3c7" };
  return { color: "#be123c", bg: "#ffe4e6" };
}

function relTime(iso?: string | null) {
  if (!iso) return "—";
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return "—";
  const diff = Date.now() - t;
  const mins = Math.round(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.round(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.round(hrs / 24);
  if (days < 14) return `${days}d ago`;
  const weeks = Math.round(days / 7);
  if (weeks < 8) return `${weeks}w ago`;
  return new Date(iso).toLocaleDateString();
}

function Delta({ n }: { n?: number | null }) {
  if (n == null) return <span className="text-slate-300">—</span>;
  if (n === 0) {
    return <span className="text-[11px] text-slate-400 tabular-nums">0%</span>;
  }
  const up = n > 0;
  return (
    <span className={`text-[11px] tabular-nums ${up ? "text-emerald-600" : "text-rose-600"}`}>
      {up ? "+" : ""}{n}%
    </span>
  );
}

function CountDelta({ n }: { n?: number | null }) {
  if (n == null) return <span className="text-slate-300">—</span>;
  if (n === 0) return <span className="text-[11px] text-slate-400 tabular-nums">0</span>;
  const up = n > 0;
  return (
    <span className={`text-[11px] tabular-nums ${up ? "text-rose-600" : "text-emerald-600"}`}>
      {up ? "+" : ""}{n}
    </span>
  );
}

function ScoreCellView({
  cell,
  kind = "pct",
}: {
  cell?: Cell | null;
  kind?: "pct" | "count" | "health";
}) {
  if (cell?.status === "not_implemented") {
    return <span className="text-[12px] text-slate-400 whitespace-nowrap">Not implemented</span>;
  }
  if (!cell || cell.status === "empty" || cell.value == null) {
    return <span className="text-slate-400">—</span>;
  }
  const n = cell.value;
  if (kind === "count") {
    return (
      <span className="inline-flex items-baseline gap-1 justify-end">
        <span className="tabular-nums font-semibold text-slate-800">{n}</span>
        <CountDelta n={cell.delta} />
      </span>
    );
  }
  const tone = kind === "health" ? healthTone(n) : null;
  return (
    <span className="inline-flex items-baseline gap-1 justify-end">
      <span
        className="tabular-nums font-semibold"
        style={tone ? { color: tone.color } : { color: "#1e293b" }}
      >
        {n}%
      </span>
      <Delta n={cell.delta} />
    </span>
  );
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

function Locked({ plan, reason }: { plan?: string; reason?: string }) {
  return (
    <div className="relative overflow-hidden border border-slate-200 bg-white min-h-[280px]">
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
      </div>
    </div>
  );
}

function HealthRing({ value }: { value?: number | null }) {
  if (value == null) {
    return (
      <div className="w-[88px] h-[88px] rounded-full border-8 border-slate-200 flex items-center justify-center text-slate-400 text-lg font-semibold">
        —
      </div>
    );
  }
  const tone = healthTone(value);
  const r = 34;
  const c = 2 * Math.PI * r;
  const dash = (value / 100) * c;
  return (
    <div className="relative w-[88px] h-[88px]">
      <svg viewBox="0 0 88 88" className="w-[88px] h-[88px] -rotate-90">
        <circle cx="44" cy="44" r={r} fill="none" stroke="#e2e8f0" strokeWidth="8" />
        <circle
          cx="44" cy="44" r={r} fill="none" stroke={tone.color} strokeWidth="8"
          strokeDasharray={`${dash} ${c - dash}`} strokeLinecap="butt"
        />
      </svg>
      <div className="absolute inset-0 flex items-center justify-center text-[18px] font-bold tabular-nums" style={{ color: tone.color }}>
        {value}
      </div>
    </div>
  );
}

function ThematicCard({ title, cell }: { title: string; cell?: Cell }) {
  return (
    <div className="border border-slate-200 bg-white p-3 min-h-[92px]">
      <p className="text-[11px] font-semibold text-slate-500 uppercase tracking-wide">{title}</p>
      <div className="mt-2 text-[20px] font-semibold">
        <ScoreCellView cell={cell} kind="pct" />
      </div>
    </div>
  );
}

export function SiteAuditPanel({
  projects = [],
  project,
  onProjectChange,
}: {
  projects?: any[];
  project?: any;
  onProjectChange?: (p: any) => void;
}) {
  const [list, setList] = useState<any>(null);
  const [detail, setDetail] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [running, setRunning] = useState(false);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [tab, setTab] = useState("overview");
  const [page, setPage] = useState(1);
  const [sort, setSort] = useState("last_update");
  const [dir, setDir] = useState<"asc" | "desc">("desc");
  const [issueFilter, setIssueFilter] = useState<"all" | "error" | "warning" | "notice">("all");
  const [pageSearch, setPageSearch] = useState("");

  const [formProjectId, setFormProjectId] = useState<number>(project?.id || 0);
  const [domain, setDomain] = useState(project?.url || "");
  const [scope, setScope] = useState("domain");
  const [pageLimit, setPageLimit] = useState(8);
  const [crawlSource, setCrawlSource] = useState("website");
  const [schedule, setSchedule] = useState("manual");
  const [userAgent, setUserAgent] = useState("desktop");
  const [jsRendering, setJsRendering] = useState(false);
  const [subfolder, setSubfolder] = useState("");

  const loadList = async () => {
    setLoading(true);
    try {
      const res = await dashboardApi.listSiteAudit({ page, page_size: 10, sort, dir });
      setList(res.data ?? null);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const loadDetail = async (id: number) => {
    setLoading(true);
    try {
      const res = await dashboardApi.getSiteAudit(id);
      const payload = res.data?.data ?? null;
      setDetail(payload);
      const camp = payload?.campaign;
      if (camp) {
        setDomain(camp.domain || "");
        setScope(camp.scope || "domain");
        setPageLimit(camp.page_limit || 8);
        setCrawlSource(camp.crawl_source || "website");
        setSchedule(camp.schedule || "manual");
        setUserAgent(camp.user_agent || "desktop");
        setJsRendering(!!camp.js_rendering);
        setSubfolder(camp.subfolder_path || "");
        setFormProjectId(camp.project_id);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (selectedId) loadDetail(selectedId);
    else loadList();
  }, [selectedId, page, sort, dir]);

  const limits = (selectedId ? detail?.limits : list?.data?.limits) ?? {};
  const campaigns: any[] = list?.data?.campaigns ?? [];
  const meta = list?.meta ?? {};
  const total = Number(meta.total || 0);
  const pageSize = Number(meta.page_size || 10);
  const reports = detail?.reports ?? {};
  const campaign = detail?.campaign;
  const overview = reports.overview ?? {};
  const thematic = overview.thematic ?? {};

  const toggleSort = (key?: string) => {
    if (!key) return;
    if (sort === key) setDir((d) => (d === "asc" ? "desc" : "asc"));
    else {
      setSort(key);
      setDir(key === "project" ? "asc" : "desc");
    }
    setPage(1);
  };

  const openCreate = () => {
    const p = project || projects[0];
    setFormProjectId(p?.id || 0);
    setDomain(p?.url || "");
    setScope("domain");
    setPageLimit(Math.min(8, limits.pages_per_audit || 8));
    setCrawlSource("website");
    setSchedule("manual");
    setUserAgent("desktop");
    setJsRendering(false);
    setSubfolder("");
    setShowCreate(true);
  };

  const handleSave = async (andRun: boolean) => {
    const pid = formProjectId || project?.id;
    if (!pid) {
      toast("Pick a project first.", "error");
      return;
    }
    setSaving(true);
    try {
      await dashboardApi.saveSiteAudit({
        project_id: pid,
        domain,
        scope,
        page_limit: pageLimit,
        crawl_source: crawlSource,
        schedule,
        user_agent: userAgent,
        js_rendering: jsRendering,
        subfolder_path: subfolder,
      });
      toast("Campaign saved.", "success");
      setShowCreate(false);
      setShowSettings(false);
      setSelectedId(pid);
      if (andRun) {
        await handleRun(pid);
      } else {
        await loadDetail(pid);
      }
    } catch (err: any) {
      toast(err.response?.data?.error?.message || "Could not save campaign.", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleRun = async (pid = selectedId) => {
    if (!pid) return;
    setRunning(true);
    try {
      const res = await dashboardApi.runSiteAudit(pid, { fetch: true });
      const payload = res.data?.data ?? null;
      if (payload) setDetail(payload);
      toast(payload?.honesty ? "Audit finished." : "Audit updated.", "success");
    } catch (err: any) {
      toast(err.response?.data?.error?.message || "Audit failed.", "error");
    } finally {
      setRunning(false);
    }
  };

  const exportCSV = () => {
    const rows = selectedId ? (reports.pages ?? []) : campaigns;
    if (selectedId) {
      const header = "url,status,title,score,errors,warnings,inlinks,indexable";
      const lines = rows.map((r: any) =>
        [r.url, r.status_code, r.title, r.score, r.errors, r.warnings, r.inlinks, r.indexable]
          .map((x: any) => `"${String(x ?? "").replace(/"/g, '""')}"`).join(",")
      );
      const blob = new Blob([[header, ...lines].join("\n")], { type: "text/csv" });
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = "site-audit-pages.csv";
      a.click();
      return;
    }
    const header = "project,domain,last_update,pages_crawled,site_health,errors,warnings";
    const lines = campaigns.map((r: any) =>
      [r.project_name, r.domain, r.last_update || "", r.pages_crawled ?? "", r.site_health?.value ?? "", r.errors?.value ?? "", r.warnings?.value ?? ""]
        .map((x: any) => `"${String(x).replace(/"/g, '""')}"`).join(",")
    );
    const blob = new Blob([[header, ...lines].join("\n")], { type: "text/csv" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = "site-audit-campaigns.csv";
    a.click();
  };

  const issueRows = useMemo(() => {
    return (reports.issues ?? []).filter((i: any) => issueFilter === "all" || i.severity === issueFilter);
  }, [reports.issues, issueFilter]);

  const pageRows = useMemo(() => {
    const q = pageSearch.trim().toLowerCase();
    return (reports.pages ?? []).filter((p: any) => !q || String(p.url || "").toLowerCase().includes(q) || String(p.title || "").toLowerCase().includes(q));
  }, [reports.pages, pageSearch]);

  const from = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);
  const maxPage = Math.max(1, Math.ceil(total / pageSize) || 1);

  return (
    <div className="min-h-[calc(100vh-4rem)]" style={{ background: PAGE_BG }}>
      <div className="sticky top-0 z-20 bg-white border-b border-slate-200">
        <div className="flex flex-wrap items-center gap-2 px-4 py-2.5">
          {selectedId ? (
            <button
              type="button"
              onClick={() => { setSelectedId(null); setDetail(null); setTab("overview"); }}
              className="inline-flex h-8 items-center gap-1 text-[12px] text-slate-600 hover:text-slate-900"
            >
              <ArrowLeft className="w-3.5 h-3.5" /> All campaigns
            </button>
          ) : null}
          <span className="inline-flex items-center justify-center w-6 h-6" style={{ background: ORANGE }}>
            <ClipboardCheck className="w-3.5 h-3.5 text-white" />
          </span>
          <h1 className="text-[15px] font-semibold text-slate-900">Site Audit</h1>
          {limits.plan_label ? (
            <span className="h-5 px-1.5 inline-flex items-center text-[10px] font-semibold uppercase tracking-wide bg-slate-100 text-slate-600">
              {limits.plan_label}
            </span>
          ) : null}
          {selectedId && campaign ? (
            <span className="text-[12px] text-slate-500 truncate max-w-[240px]">{campaign.domain}</span>
          ) : null}
          <div className="ml-auto flex flex-wrap items-center gap-1.5">
            {selectedId ? (
              <>
                <ToolBtn onClick={() => setShowSettings(true)}><Settings2 className="w-3.5 h-3.5" /> Settings</ToolBtn>
                <ToolBtn onClick={exportCSV}><Download className="w-3.5 h-3.5" /> Export</ToolBtn>
                <button
                  type="button"
                  disabled={running}
                  onClick={() => handleRun()}
                  className="inline-flex h-8 items-center gap-1.5 px-3 text-[12px] font-semibold text-white disabled:opacity-60"
                  style={{ background: ORANGE }}
                >
                  {running ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RefreshCw className="w-3.5 h-3.5" />}
                  Start Audit
                </button>
              </>
            ) : (
              <>
                <ToolBtn onClick={exportCSV}><Download className="w-3.5 h-3.5" /> Export</ToolBtn>
                <button
                  type="button"
                  onClick={openCreate}
                  className="inline-flex h-8 items-center gap-1.5 px-3 text-[12px] font-semibold text-white"
                  style={{ background: ORANGE }}
                >
                  <Plus className="w-3.5 h-3.5" /> Create Project
                </button>
              </>
            )}
          </div>
        </div>
        {selectedId ? (
          <div className="flex gap-0 px-4 overflow-x-auto">
            {TABS.map((t) => {
              const locked = t.id === "compare" && reports.compare?.locked;
              return (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => setTab(t.id)}
                  className={`h-9 px-3 text-[13px] whitespace-nowrap border-b-2 ${
                    tab === t.id ? "border-[#ff642d] text-slate-900 font-semibold" : "border-transparent text-slate-500 hover:text-slate-800"
                  }`}
                >
                  {t.label}
                  {locked ? <Lock className="inline w-3 h-3 ml-1 text-slate-400" /> : null}
                </button>
              );
            })}
          </div>
        ) : null}
      </div>

      <div className="p-4">
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="w-6 h-6 animate-spin text-[#ff642d]" />
          </div>
        ) : selectedId ? (
          <CampaignBody
            tab={tab}
            campaign={campaign}
            reports={reports}
            overview={overview}
            thematic={thematic}
            honesty={detail?.honesty || overview.honesty}
            jsMessage={detail?.js_message}
            issueFilter={issueFilter}
            setIssueFilter={setIssueFilter}
            issueRows={issueRows}
            pageSearch={pageSearch}
            setPageSearch={setPageSearch}
            pageRows={pageRows}
          />
        ) : (
          <ListBody
            campaigns={campaigns}
            sort={sort}
            dir={dir}
            onSort={toggleSort}
            onOpen={(id) => setSelectedId(id)}
            from={from}
            to={to}
            total={total}
            page={page}
            maxPage={maxPage}
            setPage={setPage}
            onCreate={openCreate}
            honesty={list?.data?.honesty}
          />
        )}
      </div>

      {(showCreate || showSettings) && (
        <Wizard
          title={showSettings ? "Campaign settings" : "Create Project"}
          projects={projects}
          formProjectId={formProjectId}
          setFormProjectId={(id: number) => {
            setFormProjectId(id);
            const p = projects.find((x: any) => x.id === id);
            if (p?.url) setDomain(p.url);
            onProjectChange?.(p);
          }}
          domain={domain}
          setDomain={setDomain}
          scope={scope}
          setScope={setScope}
          pageLimit={pageLimit}
          setPageLimit={setPageLimit}
          crawlSource={crawlSource}
          setCrawlSource={setCrawlSource}
          schedule={schedule}
          setSchedule={setSchedule}
          userAgent={userAgent}
          setUserAgent={setUserAgent}
          jsRendering={jsRendering}
          setJsRendering={setJsRendering}
          subfolder={subfolder}
          setSubfolder={setSubfolder}
          limits={limits}
          saving={saving}
          running={running}
          onClose={() => { setShowCreate(false); setShowSettings(false); }}
          onSave={() => handleSave(false)}
          onStart={() => handleSave(true)}
        />
      )}
    </div>
  );
}

function ListBody({
  campaigns, sort, dir, onSort, onOpen, from, to, total, page, maxPage, setPage, onCreate, honesty,
}: {
  campaigns: any[];
  sort: string;
  dir: string;
  onSort: (k?: string) => void;
  onOpen: (id: number) => void;
  from: number;
  to: number;
  total: number;
  page: number;
  maxPage: number;
  setPage: (n: number) => void;
  onCreate: () => void;
  honesty?: string;
}) {
  return (
    <div>
      <div className="border border-slate-200 bg-white overflow-hidden">
        {campaigns.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-center px-6">
            <AlertCircle className="w-5 h-5 text-slate-300 mb-2" />
            <p className="text-[14px] font-semibold text-slate-800">No Site Audit campaigns yet</p>
            <p className="text-[13px] text-slate-500 mt-1 max-w-md">Create a project to configure crawl scope, then Start Audit. Empty cells stay — until a crawl exists.</p>
            <button
              type="button"
              onClick={onCreate}
              className="mt-4 inline-flex h-8 items-center gap-1.5 px-3 text-[12px] font-semibold text-white"
              style={{ background: ORANGE }}
            >
              <Plus className="w-3.5 h-3.5" /> Create Project
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-[13px] border-collapse min-w-[1280px]">
              <thead>
                <tr className="bg-[#f8f9fb] border-b border-slate-200">
                  {LIST_COLUMNS.map((c) => (
                    <th
                      key={c.key}
                      className={`px-3 py-2 text-[11px] font-semibold text-slate-500 uppercase tracking-wide whitespace-nowrap ${
                        c.align === "right" ? "text-right" : "text-left"
                      } ${c.sort ? "cursor-pointer select-none hover:text-slate-800" : ""}`}
                      onClick={() => onSort(c.sort)}
                    >
                      <span className="inline-flex items-center gap-0.5">
                        {c.label}
                        {c.sort && sort === c.sort ? (
                          <ChevronDown className={`w-3 h-3 ${dir === "asc" ? "rotate-180" : ""}`} />
                        ) : null}
                      </span>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {campaigns.map((row) => (
                  <tr
                    key={row.project_id}
                    className="border-b border-slate-100 hover:bg-[#fff7f3] cursor-pointer"
                    onClick={() => onOpen(row.project_id)}
                  >
                    <td className="px-3 py-[9px]">
                      <div className="font-medium text-slate-900">{row.project_name || row.domain}</div>
                      <div className="text-[11px] text-slate-500">{row.domain}</div>
                    </td>
                    <td className="px-3 py-[9px] text-slate-600 whitespace-nowrap">{relTime(row.last_update)}</td>
                    <td className="px-3 py-[9px] text-right tabular-nums">
                      {row.pages_crawled == null ? <span className="text-slate-400">—</span> : (
                        <span>{row.pages_crawled} / {row.page_limit}</span>
                      )}
                    </td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.site_health} kind="health" /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.ai_search_health} /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.errors} kind="count" /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.warnings} kind="count" /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.crawlability} /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.https} /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.international} /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.performance} /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.linking} /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.markups} /></td>
                    <td className="px-3 py-[9px] text-right"><ScoreCellView cell={row.core_web_vitals} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
      <div className="flex items-center justify-between mt-3 text-[12px] text-slate-500">
        <span>{total === 0 ? "0 campaigns" : `${from}–${to} of ${total}`}</span>
        <div className="flex items-center gap-1">
          <ToolBtn disabled={page <= 1} onClick={() => setPage(page - 1)}><ChevronLeft className="w-3.5 h-3.5" /></ToolBtn>
          <span className="px-2 tabular-nums">{page} / {maxPage}</span>
          <ToolBtn disabled={page >= maxPage} onClick={() => setPage(page + 1)}><ChevronRight className="w-3.5 h-3.5" /></ToolBtn>
        </div>
      </div>
      {honesty ? <p className="mt-3 text-[11px] text-slate-500 max-w-4xl">{honesty}</p> : null}
    </div>
  );
}

function CampaignBody({
  tab, campaign, reports, overview, thematic, honesty, jsMessage,
  issueFilter, setIssueFilter, issueRows, pageSearch, setPageSearch, pageRows,
}: any) {
  if (!campaign) {
    return (
      <div className="border border-slate-200 bg-white py-16 text-center text-[13px] text-slate-500">
        Save a campaign to run Site Audit.
      </div>
    );
  }
  if (tab === "issues") {
    return (
      <div>
        <div className="flex flex-wrap items-center gap-1.5 mb-2">
          {(["all", "error", "warning", "notice"] as const).map((id) => (
            <button
              key={id}
              type="button"
              onClick={() => setIssueFilter(id)}
              className={`h-7 px-2 text-[12px] border capitalize ${
                issueFilter === id ? "border-[#ff642d] text-[#c2410c] bg-[#fff4f0]" : "border-slate-200 bg-white text-slate-600"
              }`}
            >
              {id === "all" ? "All" : id + "s"}
            </button>
          ))}
          <span className="ml-auto text-[11px] text-slate-500 tabular-nums">{issueRows.length} issues</span>
        </div>
        <IssueTable rows={issueRows} />
      </div>
    );
  }
  if (tab === "pages") {
    return (
      <div>
        <div className="flex items-center gap-2 mb-2">
          <input
            value={pageSearch}
            onChange={(e) => setPageSearch(e.target.value)}
            placeholder="Filter URLs"
            className="h-8 px-2 w-64 text-[12px] border border-slate-200 bg-white outline-none focus:border-[#ff642d]"
          />
          <span className="ml-auto text-[11px] text-slate-500 tabular-nums">{pageRows.length} pages</span>
        </div>
        <PageTable rows={pageRows} />
      </div>
    );
  }
  if (tab === "statistics") {
    const stats = reports.statistics ?? {};
    const codes = Object.entries(stats.status_codes || {});
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
        <Stat label="Pages crawled" value={stats.pages ?? "—"} />
        <Stat label="Indexable" value={stats.indexable ?? "—"} />
        <Stat label="Not indexable" value={stats.blocked ?? "—"} />
        {codes.map(([code, n]) => (
          <Stat key={code} label={`HTTP ${code}`} value={n as number} />
        ))}
        <p className="col-span-full text-[11px] text-slate-500 mt-2">{stats.honesty}</p>
      </div>
    );
  }
  if (tab === "compare") {
    const cmp = reports.compare ?? {};
    if (cmp.locked) return <Locked plan={cmp.plan_needed} reason={cmp.reason} />;
    if (!cmp.current) {
      return <p className="text-[13px] text-slate-500">Start Audit to store a snapshot. Compare uses the last two crawls.</p>;
    }
    return (
      <div className="grid md:grid-cols-2 gap-3">
        <CompareCard title="Current crawl" snap={cmp.current} />
        <CompareCard title="Previous crawl" snap={cmp.previous} />
        <p className="md:col-span-2 text-[11px] text-slate-500">{cmp.honesty}</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-2">
        <div className="border border-slate-200 bg-white p-4 flex items-center gap-4 col-span-2 lg:col-span-1">
          <HealthRing value={overview.site_health?.value} />
          <div>
            <p className="text-[11px] font-semibold text-slate-500 uppercase">Site Health</p>
            <ScoreCellView cell={overview.site_health} kind="health" />
          </div>
        </div>
        <Metric title="Pages crawled" value={overview.pages_crawled == null ? "—" : String(overview.pages_crawled)} />
        <Metric title="Errors" extra={<ScoreCellView cell={overview.errors} kind="count" />} />
        <Metric title="Warnings" extra={<ScoreCellView cell={overview.warnings} kind="count" />} />
        <Metric title="Notices" extra={<ScoreCellView cell={overview.notices} kind="count" />} />
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
        <ThematicCard title="Crawlability" cell={thematic.crawlability} />
        <ThematicCard title="HTTPS" cell={thematic.https} />
        <ThematicCard title="Int. SEO" cell={thematic.international} />
        <ThematicCard title="Site Performance" cell={thematic.performance} />
        <ThematicCard title="Internal Linking" cell={thematic.linking} />
        <ThematicCard title="Markups" cell={thematic.markups} />
        <ThematicCard title="Core Web Vitals" cell={thematic.core_web_vitals} />
        <ThematicCard title="AI Search Health" cell={thematic.ai_search_health} />
      </div>
      {overview.stopped_reason ? (
        <p className="text-[12px] text-slate-500">Stopped: {overview.stopped_reason}</p>
      ) : null}
      {jsMessage ? <p className="text-[11px] text-slate-500">{jsMessage}</p> : null}
      {honesty ? <p className="text-[11px] text-slate-500 max-w-4xl">{honesty}</p> : null}
    </div>
  );
}

function Metric({ title, value, extra }: { title: string; value?: string; extra?: ReactNode }) {
  return (
    <div className="border border-slate-200 bg-white p-4">
      <p className="text-[11px] font-semibold text-slate-500 uppercase tracking-wide">{title}</p>
      <div className="mt-2 text-[22px] font-semibold text-slate-900 tabular-nums">{extra || value}</div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="border border-slate-200 bg-white p-3">
      <p className="text-[11px] font-semibold text-slate-500 uppercase">{label}</p>
      <p className="mt-1 text-[20px] font-semibold tabular-nums">{value}</p>
    </div>
  );
}

function CompareCard({ title, snap }: { title: string; snap?: any }) {
  if (!snap) {
    return (
      <div className="border border-slate-200 bg-white p-4 min-h-[160px] text-[13px] text-slate-500">
        {title}: —
      </div>
    );
  }
  return (
    <div className="border border-slate-200 bg-white p-4">
      <p className="text-[12px] font-semibold text-slate-700 mb-2">{title}</p>
      <dl className="grid grid-cols-2 gap-x-3 gap-y-1 text-[13px]">
        <dt className="text-slate-500">Pages</dt><dd className="tabular-nums">{snap.pages_crawled ?? "—"}</dd>
        <dt className="text-slate-500">Site Health</dt><dd className="tabular-nums">{snap.site_health ?? "—"}</dd>
        <dt className="text-slate-500">Errors</dt><dd className="tabular-nums">{snap.errors ?? "—"}</dd>
        <dt className="text-slate-500">Warnings</dt><dd className="tabular-nums">{snap.warnings ?? "—"}</dd>
      </dl>
    </div>
  );
}

function IssueTable({ rows }: { rows: any[] }) {
  if (!rows.length) {
    return (
      <div className="border border-slate-200 bg-white py-14 text-center text-[13px] text-slate-500">
        No issues in this snapshot.
      </div>
    );
  }
  return (
    <div className="border border-slate-200 bg-white overflow-x-auto">
      <table className="w-full text-[13px]">
        <thead>
          <tr className="bg-[#f8f9fb] border-b border-slate-200">
            <th className="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase">Severity</th>
            <th className="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase">Issue</th>
            <th className="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase">URL</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-b border-slate-100">
              <td className="px-3 py-[7px] capitalize whitespace-nowrap">
                <span className={`text-[11px] font-semibold px-1.5 py-0.5 ${
                  r.severity === "error" ? "bg-rose-50 text-rose-700" : r.severity === "warning" ? "bg-amber-50 text-amber-700" : "bg-slate-100 text-slate-600"
                }`}>{r.severity}</span>
              </td>
              <td className="px-3 py-[7px]">
                <div className="font-medium text-slate-800">{r.label}</div>
                <div className="text-[12px] text-slate-500">{r.detail}</div>
              </td>
              <td className="px-3 py-[7px] text-[12px] text-slate-600 max-w-[280px] truncate">{r.url}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PageTable({ rows }: { rows: any[] }) {
  if (!rows.length) {
    return (
      <div className="border border-slate-200 bg-white py-14 text-center text-[13px] text-slate-500">
        No crawled pages yet. Start Audit to fetch HTML.
      </div>
    );
  }
  return (
    <div className="border border-slate-200 bg-white overflow-x-auto">
      <table className="w-full text-[13px]">
        <thead>
          <tr className="bg-[#f8f9fb] border-b border-slate-200">
            {["URL", "Status", "Title", "Score", "Errors", "Warnings", "Inlinks", "Indexable"].map((h) => (
              <th key={h} className="px-3 py-2 text-left text-[11px] font-semibold text-slate-500 uppercase">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r: any) => (
            <tr key={r.url} className="border-b border-slate-100">
              <td className="px-3 py-[7px] max-w-[320px] truncate text-slate-800">{r.url}</td>
              <td className="px-3 py-[7px] tabular-nums">{r.status_code || "—"}</td>
              <td className="px-3 py-[7px] max-w-[220px] truncate">{r.title || "—"}</td>
              <td className="px-3 py-[7px] tabular-nums">{r.score}</td>
              <td className="px-3 py-[7px] tabular-nums">{r.errors}</td>
              <td className="px-3 py-[7px] tabular-nums">{r.warnings}</td>
              <td className="px-3 py-[7px] tabular-nums">{r.inlinks}</td>
              <td className="px-3 py-[7px]">{r.indexable ? "Yes" : "No"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="block text-[11px] font-semibold text-slate-500 uppercase tracking-wide mb-1">{label}</span>
      {children}
    </label>
  );
}

const inputCls = "h-8 w-full px-2 text-[13px] border border-slate-200 bg-white outline-none focus:border-[#ff642d]";

function Wizard({
  title, projects, formProjectId, setFormProjectId, domain, setDomain, scope, setScope,
  pageLimit, setPageLimit, crawlSource, setCrawlSource, schedule, setSchedule,
  userAgent, setUserAgent, jsRendering, setJsRendering, subfolder, setSubfolder,
  limits, saving, running, onClose, onSave, onStart,
}: any) {
  const jsLocked = !limits.js_rendering;
  const schedLocked = !limits.schedule;
  return (
    <div className="fixed inset-0 z-40 bg-black/30 flex items-start justify-center overflow-y-auto py-10">
      <div className="w-full max-w-xl bg-white border border-slate-200 shadow-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200">
          <h2 className="text-[15px] font-semibold">{title}</h2>
          <button type="button" onClick={onClose} className="text-[12px] text-slate-500 hover:text-slate-800">Close</button>
        </div>
        <div className="p-4 grid gap-3">
          <Field label="Project">
            <select className={inputCls} value={formProjectId || ""} onChange={(e) => setFormProjectId(Number(e.target.value))}>
              <option value="">Select a project</option>
              {projects.map((p: any) => (
                <option key={p.id} value={p.id}>{p.name} — {p.url}</option>
              ))}
            </select>
          </Field>
          <Field label="Domain">
            <input className={inputCls} value={domain} onChange={(e) => setDomain(e.target.value)} placeholder="example.com" />
          </Field>
          <Field label="Crawl scope">
            <select className={inputCls} value={scope} onChange={(e) => setScope(e.target.value)}>
              <option value="domain">Domain</option>
              <option value="subdomain">Subdomain</option>
              <option value="subfolder">Subfolder</option>
            </select>
          </Field>
          {scope === "subfolder" ? (
            <Field label="Subfolder path">
              <input className={inputCls} value={subfolder} onChange={(e) => setSubfolder(e.target.value)} placeholder="/blog" />
            </Field>
          ) : null}
          <Field label={`Limit per audit (plan max ${limits.pages_per_audit ?? "—"}; this run caps at ${limits.run_page_cap ?? 8})`}>
            <input className={inputCls} type="number" min={1} value={pageLimit} onChange={(e) => setPageLimit(Number(e.target.value))} />
          </Field>
          <Field label="Pages to crawl">
            <select className={inputCls} value={crawlSource} onChange={(e) => setCrawlSource(e.target.value)}>
              <option value="website">Website (follow links from the homepage)</option>
              <option value="sitemap">Sitemaps (URLs declared in sitemap XML)</option>
            </select>
          </Field>
          <Field label="Schedule">
            <select className={inputCls} value={schedule} onChange={(e) => setSchedule(e.target.value)} disabled={schedLocked}>
              <option value="manual">Manual</option>
              <option value="weekly">Weekly (stored, not auto-run)</option>
              <option value="daily">Daily (stored, not auto-run)</option>
            </select>
            {schedLocked ? <p className="text-[11px] text-slate-500 mt-1">Scheduled recrawls are included on SEO and above. Auto-run is not wired.</p> : <p className="text-[11px] text-slate-500 mt-1">Stored only. Auto-crawl is not run yet.</p>}
          </Field>
          <Field label="User agent">
            <select className={inputCls} value={userAgent} onChange={(e) => setUserAgent(e.target.value)}>
              <option value="desktop">Site Audit desktop</option>
              <option value="mobile">Mobile</option>
              <option value="googlebot">Googlebot</option>
              <option value="openai-search">OpenAI-Search</option>
            </select>
          </Field>
          <label className="flex items-start gap-2 text-[13px] text-slate-700">
            <input type="checkbox" checked={jsRendering && !jsLocked} disabled={jsLocked} onChange={(e) => setJsRendering(e.target.checked)} className="mt-0.5" />
            <span>
              JS rendering {jsLocked ? "(Pro+)" : ""}
              <span className="block text-[11px] text-slate-500">JavaScript is not executed. The crawler reads HTML only.</span>
            </span>
          </label>
        </div>
        <div className="flex items-center justify-end gap-2 px-4 py-3 border-t border-slate-200">
          <ToolBtn onClick={onClose}>Cancel</ToolBtn>
          <ToolBtn onClick={onSave} disabled={saving}>{saving ? "Saving…" : "Save"}</ToolBtn>
          <button
            type="button"
            disabled={saving || running}
            onClick={onStart}
            className="inline-flex h-8 items-center px-3 text-[12px] font-semibold text-white disabled:opacity-60"
            style={{ background: ORANGE }}
          >
            {running ? "Starting…" : "Start Audit"}
          </button>
        </div>
      </div>
    </div>
  );
}
