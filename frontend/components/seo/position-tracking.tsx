"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  AlertCircle, Award, BarChart2, Download, Eye, Globe, Loader2,
  Lock, RefreshCw, Search, Sparkles, TrendingUp,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";

function dash(v: unknown, digits = 0): string {
  if (v == null || v === "") return "—";
  const n = Number(v);
  if (!Number.isFinite(n)) return "—";
  return digits > 0 ? n.toFixed(digits) : String(Math.round(n));
}

function posColor(pos: number) {
  if (pos <= 0) return "text-slate-500 bg-slate-50";
  if (pos <= 3) return "text-emerald-600 bg-emerald-50";
  if (pos <= 10) return "text-blue-600 bg-blue-50";
  if (pos <= 30) return "text-amber-600 bg-amber-50";
  return "text-slate-500 bg-slate-50";
}

const TABS = [
  { id: "overview", label: "Overview" },
  { id: "rankings", label: "Rankings" },
  { id: "distribution", label: "Ranking Distribution" },
  { id: "pages", label: "Pages" },
  { id: "tags", label: "Tags" },
  { id: "devices", label: "Devices & Locations" },
  { id: "landscape", label: "Landscape" },
  { id: "discovery", label: "Competitors Discovery" },
  { id: "cannibalization", label: "Cannibalization" },
  { id: "snippets", label: "Featured Snippets" },
  { id: "ai", label: "AI Search" },
] as const;

function Locked({ plan, reason }: { plan?: string; reason?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
      <Lock className="w-8 h-8 text-slate-300 mb-3" />
      <p className="text-sm font-semibold text-slate-700">
        {plan ? `Included on ${plan} and above` : "Not available on this plan"}
      </p>
      <p className="text-xs text-slate-500 mt-1 max-w-md">{reason || "Upgrade to unlock this Semrush-equivalent report."}</p>
      <Link href="/billing" className="mt-4 inline-flex h-8 items-center rounded-xl bg-foreground px-3 text-xs font-medium text-background">
        Upgrade
      </Link>
    </div>
  );
}

function Stat({ label, value, sub, icon: Icon }: { label: string; value: string; sub: string; icon: typeof Eye }) {
  return (
    <Card className="border-slate-100 shadow-none rounded-2xl">
      <CardContent className="p-6">
        <div className="w-9 h-9 rounded-xl bg-slate-50 flex items-center justify-center mb-4">
          <Icon className="w-5 h-5 text-slate-500" />
        </div>
        <p className="text-2xl font-bold text-slate-900">{value}</p>
        <p className="text-[11px] font-bold text-slate-400 uppercase tracking-widest mt-1">{label}</p>
        <p className="text-[10px] text-slate-300 mt-0.5">{sub}</p>
      </CardContent>
    </Card>
  );
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
  const rankings = (reports.rankings ?? []).filter((k: any) =>
    search ? String(k.keyword || "").toLowerCase().includes(search.toLowerCase()) : true
  );

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

  const competitors = campaign?.competitors ?? [];

  const setupForm = (
    <Card className="border-slate-100 shadow-none rounded-2xl">
      <CardHeader className="px-8 pt-8 pb-4">
        <CardTitle className="text-base font-semibold">Campaign setup</CardTitle>
        <p className="text-xs text-slate-400 mt-1">
          {limits.plan_label} plan: {limits.keywords} keywords, {limits.websites} website{limits.websites === 1 ? "" : "s"}, {limits.competitors_per_campaign} competitors per campaign.
          Extra locations and devices count toward the keyword cap.
        </p>
      </CardHeader>
      <CardContent className="px-8 pb-8 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <label className="space-y-1 text-sm">
            <span className="text-xs font-semibold text-slate-500">Domain</span>
            <Input value={domain} onChange={(e) => setDomain(e.target.value)} placeholder="example.com" className="rounded-xl" />
          </label>
          <label className="space-y-1 text-sm">
            <span className="text-xs font-semibold text-slate-500">Location</span>
            <Input value={locationName} onChange={(e) => setLocationName(e.target.value)} className="rounded-xl" />
          </label>
          <label className="space-y-1 text-sm">
            <span className="text-xs font-semibold text-slate-500">Device</span>
            <select value={device} onChange={(e) => setDevice(e.target.value)} className="h-9 w-full rounded-xl border border-slate-200 px-3 text-sm">
              <option value="desktop">Desktop</option>
              <option value="mobile">Mobile</option>
            </select>
          </label>
          <label className="space-y-1 text-sm">
            <span className="text-xs font-semibold text-slate-500">Search engine</span>
            <select value={engine} onChange={(e) => setEngine(e.target.value)} className="h-9 w-full rounded-xl border border-slate-200 px-3 text-sm">
              <option value="google">Google</option>
              <option value="bing">Bing</option>
            </select>
          </label>
        </div>
        <label className="block space-y-1 text-sm">
          <span className="text-xs font-semibold text-slate-500">Keywords (one per line{limits.tags ? ", optional | tag1, tag2" : ""})</span>
          <Textarea value={keywordText} onChange={(e) => setKeywordText(e.target.value)} rows={6} className="rounded-xl" placeholder="best coffee beans" />
        </label>
        <label className="block space-y-1 text-sm">
          <span className="text-xs font-semibold text-slate-500">Competitors (one domain per line, max {limits.competitors_per_campaign})</span>
          <Textarea value={competitorText} onChange={(e) => setCompetitorText(e.target.value)} rows={3} className="rounded-xl" placeholder="rival.com" />
        </label>
        <div className="flex gap-3">
          <Button onClick={handleSave} disabled={saving || !projectId} className="rounded-xl">
            {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
            Save campaign
          </Button>
          {campaign && (
            <Button variant="outline" className="rounded-xl" onClick={() => setShowSetup(false)}>Cancel</Button>
          )}
        </div>
      </CardContent>
    </Card>
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[40vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">SEO Intelligence</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Position Tracking</p>
          <p className="text-sm text-slate-500">
            Daily-style rank campaign for Google and Bing. Visibility is a simple average of current positions. Estimated traffic is —.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="outline" className="rounded-lg">{limits.plan_label || "Free"}</Badge>
          {campaign && (
            <>
              <Button variant="outline" className="rounded-xl h-10 text-sm" onClick={() => setShowSetup(true)}>Edit campaign</Button>
              <Button variant="outline" className="rounded-xl h-10 text-sm gap-2" onClick={exportCSV} disabled={!(reports.rankings || []).length}>
                <Download className="w-4 h-4" /> Export CSV
              </Button>
              <Button className="rounded-xl h-10 text-sm gap-2" onClick={handleRefresh} disabled={refreshing || !projectId}>
                {refreshing ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
                Update rankings
              </Button>
            </>
          )}
        </div>
      </div>

      {serp?.message && (
        <div className="bg-slate-50 border border-slate-100 rounded-2xl p-4 text-sm text-slate-600 flex items-start gap-3">
          <Globe className="w-4 h-4 mt-0.5 text-slate-400" />
          <div>
            <p>{serp.message}</p>
            {gsc.used_as_fallback && (
              <p className="text-xs text-slate-400 mt-1">Google Search Console is filling empty ranks until a SERP lookup runs. That is not a daily crawl.</p>
            )}
          </div>
        </div>
      )}

      {(!campaign || showSetup) && setupForm}

      {campaign && !showSetup && (
        <Tabs value={tab} onValueChange={setTab} className="space-y-4">
          <TabsList className="flex flex-wrap h-auto gap-1 w-full justify-start bg-slate-100 p-1 rounded-xl">
            {TABS.map((t) => (
              <TabsTrigger key={t.id} value={t.id} className="rounded-lg text-xs">
                {t.label}
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="overview">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <Stat label="Visibility" value={dash(overview.visibility, 1)} sub="avg of max(0, 100 − pos)" icon={Eye} />
              <Stat label="Avg position" value={dash(overview.avg_position, 1)} sub="ranked keywords only" icon={TrendingUp} />
              <Stat label="Estimated traffic" value="—" sub="not computed" icon={BarChart2} />
              <Stat label="Ranked / tracked" value={`${overview.ranked ?? 0} / ${overview.tracked ?? 0}`} sub="current snapshot" icon={Award} />
            </div>
            <p className="text-xs text-slate-400 mt-3">{overview.honesty}</p>
            {reports.historical?.locked ? null : (
              <p className="text-xs text-slate-400 mt-2">{reports.historical?.honesty}</p>
            )}
          </TabsContent>

          <TabsContent value="rankings">
            <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
              <CardHeader className="px-8 pt-8 pb-4 flex flex-row items-center justify-between gap-4">
                <div>
                  <CardTitle className="text-base font-semibold">Keyword rankings</CardTitle>
                  <p className="text-xs text-slate-400 mt-1">Position 0 or missing is —. GSC impressions are not search volume.</p>
                </div>
                <div className="relative w-64">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                  <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search keywords..." className="pl-9 h-9 rounded-xl" />
                </div>
              </CardHeader>
              <CardContent className="px-0 pb-0">
                {rankings.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-16">
                    <AlertCircle className="w-8 h-8 text-slate-200 mb-3" />
                    <p className="text-sm text-slate-500">No ranking rows yet. Save keywords and click Update rankings.</p>
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-t border-slate-50">
                          {["Keyword", "Pos", "URL", "Source", "GSC impr.", "Features"].map((h) => (
                            <th key={h} className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3 first:pl-8">{h}</th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {rankings.map((kw: any, i: number) => (
                          <tr key={kw.keyword ?? i} className="border-t border-slate-50">
                            <td className="px-8 py-3 font-semibold text-slate-900">
                              {kw.keyword}
                              {(kw.tags || []).length > 0 && (
                                <span className="ml-2 text-[10px] text-slate-400">{kw.tags.join(", ")}</span>
                              )}
                            </td>
                            <td className="px-4 py-3">
                              {kw.position > 0 ? (
                                <span className={`inline-flex items-center justify-center w-10 h-8 rounded-lg text-xs font-bold ${posColor(kw.position)}`}>#{kw.position}</span>
                              ) : "—"}
                            </td>
                            <td className="px-4 py-3 text-xs text-slate-500 max-w-xs truncate">{kw.url || "—"}</td>
                            <td className="px-4 py-3 text-xs text-slate-500">{kw.source || "—"}</td>
                            <td className="px-4 py-3 tabular-nums">{kw.gsc_impressions > 0 ? Number(kw.gsc_impressions).toLocaleString() : "—"}</td>
                            <td className="px-4 py-3 text-xs text-slate-400">{(kw.serp_features || []).slice(0, 3).join(", ") || "—"}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="distribution">
            <Card className="border-slate-100 shadow-none rounded-2xl">
              <CardContent className="p-8">
                <h3 className="text-sm font-bold text-slate-400 uppercase tracking-widest mb-6">Ranking distribution</h3>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  {[
                    { label: "Positions 1–3", count: reports.distribution?.top3 ?? 0, color: "bg-emerald-500" },
                    { label: "Positions 4–10", count: reports.distribution?.top10 ?? 0, color: "bg-blue-500" },
                    { label: "Positions 11–30", count: reports.distribution?.top30 ?? 0, color: "bg-amber-400" },
                    { label: "Position 31+", count: reports.distribution?.beyond ?? 0, color: "bg-slate-200" },
                  ].map((b) => (
                    <div key={b.label} className="space-y-3">
                      <div className="flex justify-between">
                        <p className="text-xs font-semibold text-slate-500">{b.label}</p>
                        <p className="text-sm font-bold">{b.count}</p>
                      </div>
                      <div className="h-2 bg-slate-100 rounded-full overflow-hidden">
                        <div className={`h-full ${b.color}`} style={{ width: `${Math.min(100, (b.count / Math.max(1, overview.tracked || 1)) * 100)}%` }} />
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="pages">
            <ReportTable
              empty="No ranking URLs in this snapshot."
              rows={reports.pages ?? []}
              columns={["URL", "Keywords", "Best pos"]}
              render={(row: any) => [row.url, row.keywords, row.best_position > 0 ? row.best_position : "—"]}
            />
          </TabsContent>

          <TabsContent value="tags">
            {reports.tags?.locked ? (
              <Locked plan={reports.tags.plan_needed} reason={reports.tags.reason} />
            ) : (
              <ReportTable
                empty="No tags on tracked keywords. Add tags as keyword | tag on save."
                rows={reports.tags?.rows ?? []}
                columns={["Tag", "Keywords", "Ranked"]}
                render={(row: any) => [row.tag, row.keywords, row.ranked]}
              />
            )}
          </TabsContent>

          <TabsContent value="devices">
            {reports.devices_locations?.locked ? (
              <Locked plan={reports.devices_locations.plan_needed} reason={reports.devices_locations.reason} />
            ) : (
              <Card className="border-slate-100 shadow-none rounded-2xl">
                <CardContent className="p-8 space-y-3">
                  <p className="text-sm text-slate-600">{reports.devices_locations?.note}</p>
                  <p className="text-sm"><span className="font-semibold">Primary:</span> {campaign.location_name} · {campaign.device} · {campaign.engine}</p>
                  {(reports.devices_locations?.targets || []).length === 0 ? (
                    <p className="text-sm text-slate-500">No extra targets. Multitargeting is available on this plan; extra targets count toward the keyword cap.</p>
                  ) : (
                    (reports.devices_locations.targets as any[]).map((t, i) => (
                      <p key={i} className="text-sm text-slate-700">{t.location_name} · {t.device} · {t.engine}</p>
                    ))
                  )}
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="landscape">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr>
                    <th className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest py-3">Keyword</th>
                    <th className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest py-3">You</th>
                    {competitors.map((d: string) => (
                      <th key={d} className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest py-3">{d}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {(reports.landscape || []).length === 0 ? (
                    <tr><td className="py-8 text-slate-500" colSpan={2 + competitors.length}>No landscape rows yet.</td></tr>
                  ) : (reports.landscape as any[]).map((row) => (
                    <tr key={row.keyword} className="border-t border-slate-50">
                      <td className="py-3 font-medium">{row.keyword}</td>
                      <td className="py-3">{row.target_position > 0 ? row.target_position : "—"}</td>
                      {competitors.map((d: string) => (
                        <td key={d} className="py-3">{row.by_domain?.[d] > 0 ? row.by_domain[d] : "—"}</td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </TabsContent>

          <TabsContent value="discovery">
            <Card className="border-slate-100 shadow-none rounded-2xl">
              <CardContent className="p-8 space-y-3">
                <p className="text-xs text-slate-400">{reports.competitors_discovery?.honesty}</p>
                {(reports.competitors_discovery?.domains || []).length === 0 ? (
                  <p className="text-sm text-slate-500">No other domains observed in this snapshot.</p>
                ) : (
                  <ul className="grid grid-cols-1 md:grid-cols-2 gap-2">
                    {(reports.competitors_discovery.domains as string[]).map((d) => (
                      <li key={d} className="text-sm font-medium text-slate-800">{d}</li>
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="cannibalization">
            {reports.cannibalization?.locked ? (
              <Locked plan={reports.cannibalization.plan_needed} reason={reports.cannibalization.reason} />
            ) : (
              <ReportTable
                empty="No keyword is ranking with more than one URL in this snapshot."
                rows={reports.cannibalization?.rows ?? []}
                columns={["Keyword", "URLs"]}
                render={(row: any) => [row.keyword, (row.urls || []).map((u: any) => `${u.url} (#${u.position})`).join(" · ") || "—"]}
              />
            )}
          </TabsContent>

          <TabsContent value="snippets">
            <ReportTable
              empty="No featured snippets in this snapshot."
              rows={reports.featured_snippets ?? []}
              columns={["Keyword", "Owner", "URL"]}
              render={(row: any) => [row.keyword, row.owner, row.url || "—"]}
            />
          </TabsContent>

          <TabsContent value="ai">
            <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
              <Sparkles className="w-8 h-8 text-slate-300 mb-3" />
              <p className="text-sm font-semibold text-slate-700">AI Search prompt tracking is not connected</p>
              <p className="text-xs text-slate-500 mt-1 max-w-lg">
                {reports.ai_search?.reason || data?.honesty?.ai_search}
              </p>
              <div className="flex gap-3 mt-4">
                {reports.ai_search?.locked && (
                  <Link href="/billing" className="inline-flex h-8 items-center rounded-xl border border-border px-3 text-xs font-medium">
                    Upgrade to {reports.ai_search.plan_needed || "Pro+"}
                  </Link>
                )}
                <Link href="/seo/ai-visibility" className="inline-flex h-8 items-center rounded-xl bg-foreground px-3 text-xs font-medium text-background">
                  Open AI Visibility
                </Link>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      )}

      {campaign && reports.share_of_voice && !reports.share_of_voice.locked && tab === "overview" && (
        <Card className="border-slate-100 shadow-none rounded-2xl">
          <CardHeader className="px-8 pt-8 pb-2">
            <CardTitle className="text-base font-semibold">Share of voice</CardTitle>
            <p className="text-xs text-slate-400">{reports.share_of_voice.honesty}</p>
          </CardHeader>
          <CardContent className="px-8 pb-8">
            {Object.keys(reports.share_of_voice.shares || {}).length === 0 ? (
              <p className="text-sm text-slate-500">No ranked keywords in this snapshot.</p>
            ) : (
              <ul className="space-y-2">
                {Object.entries(reports.share_of_voice.shares as Record<string, number>).map(([d, v]) => (
                  <li key={d} className="flex justify-between text-sm">
                    <span>{d}</span>
                    <span className="tabular-nums">{(v * 100).toFixed(1)}%</span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function ReportTable({
  rows, columns, render, empty,
}: {
  rows: any[];
  columns: string[];
  render: (row: any) => any[];
  empty: string;
}) {
  return (
    <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
      <CardContent className="px-0 pb-0 pt-0">
        {rows.length === 0 ? (
          <p className="px-8 py-12 text-sm text-slate-500">{empty}</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-t border-slate-50">
                {columns.map((c) => (
                  <th key={c} className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest px-4 py-3 first:pl-8">{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr key={i} className="border-t border-slate-50">
                  {render(row).map((cell, j) => (
                    <td key={j} className={`py-3 px-4 ${j === 0 ? "pl-8 font-medium text-slate-900" : "text-slate-600"}`}>{cell}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  );
}
