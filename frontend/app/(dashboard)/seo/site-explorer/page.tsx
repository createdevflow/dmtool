"use client";

import { motion, AnimatePresence } from "framer-motion";
import {
  Globe, Search, ShieldCheck, AlertCircle,
  Zap, Loader2, CheckCircle2, RefreshCw,
  ExternalLink, Clock, AlertTriangle, Info, FileText, Lock, Gauge, ListTree, ScanSearch, Timer,
} from "lucide-react";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

const severityConfig: Record<string, { color: string; bg: string; border: string; icon: any }> = {
  high:   { color: "text-rose-600",  bg: "bg-rose-50",   border: "border-rose-100", icon: AlertCircle },
  medium: { color: "text-amber-600", bg: "bg-amber-50",  border: "border-amber-100", icon: AlertTriangle },
  low:    { color: "text-blue-600",  bg: "bg-blue-50",   border: "border-blue-100", icon: Info },
};

const statusConfig: Record<string, { dot: string; label: string }> = {
  pass:    { dot: "bg-emerald-500", label: "Pass" },
  warning: { dot: "bg-amber-400",  label: "Warning" },
  fail:    { dot: "bg-rose-500",   label: "Fail" },
};

function httpsStatusLabel(https: any): string {
  if (https?.label) return https.label;
  if (https?.status === "pass") return "Pass";
  if (https?.status === "fail") return "Fail";
  if (https?.status === "warning") return "Warning";
  return "";
}

function httpsHeadline(https: any): string {
  const status = httpsStatusLabel(https);
  if (typeof https?.score === "number") {
    return status ? `${status} · ${https.score}/100` : `${https.score}/100`;
  }
  return status || "Not checked";
}

function httpsSummary(https: any): string {
  const parts: string[] = [];
  if (https?.serves_https === true) {
    parts.push("This page is served over HTTPS.");
    if (https.tls_valid) {
      const cert = ["Certificate valid"];
      if (https.cert_issuer) cert.push(https.cert_issuer);
      if (https.cert_expiry) cert.push(`expires ${https.cert_expiry}`);
      parts.push(cert.join(" · "));
    } else if (https.tls_error) {
      parts.push(`Certificate check failed: ${https.tls_error}`);
    }
    if (https.redirects_http) {
      parts.push("HTTP redirects to HTTPS.");
    } else if (https.redirect_note) {
      parts.push(https.redirect_note);
    }
    if (https.mixed_active?.length) {
      parts.push(`Active mixed content on this page: ${https.mixed_active.join(", ")}`);
    } else if (https.mixed_passive?.length) {
      parts.push(`Passive mixed content on this page: ${https.mixed_passive.join(", ")}`);
    } else if (https.serves_https) {
      parts.push("No http:// resources found on this page.");
    }
    if (https.hsts) {
      parts.push("HSTS header present.");
    }
  } else if (https?.serves_https === false) {
    parts.push("This page is served over HTTP.");
    if (https.redirect_note) parts.push(https.redirect_note);
  } else {
    parts.push("HTTPS details from the last technical audit. Re-run the audit for certificate, redirect, and mixed-content findings.");
  }
  parts.push("Checked on this page only — not a sitewide HTTP inventory.");
  return parts.join(" ");
}

function cwvHeadline(cwv: any): string {
  if (cwv?.overall_label) return cwv.overall_label;
  if (cwv?.label) return cwv.label;
  if (cwv?.status === "pass") return "Good";
  if (cwv?.status === "fail") return "Poor";
  if (cwv?.status === "warning") return "Needs work";
  return "Not checked";
}

function cwvMetricLine(m: any): string {
  if (!m) return "";
  const bits = [m.name || "", m.display || ""].filter(Boolean);
  if (m.rating === "good") bits.push("good");
  else if (m.rating === "needs_improvement") bits.push("needs improvement");
  else if (m.rating === "poor") bits.push("poor");
  return bits.join(" ");
}

function cwvSummary(cwv: any): string {
  const parts: string[] = [];
  if (cwv?.field_available) {
    const metrics = [cwvMetricLine(cwv.lcp), cwvMetricLine(cwv.inp), cwvMetricLine(cwv.cls)].filter(Boolean);
    if (metrics.length) parts.push(metrics.join(" · "));
    if (cwv.field_scope === "origin") {
      parts.push("Chrome UX Report is origin-level (not enough data for this specific URL).");
    } else {
      parts.push("Chrome UX Report field data for this URL (mobile, ~28 days).");
    }
  } else if (typeof cwv?.lab_performance === "number") {
    parts.push("Not enough CrUX field data to score Core Web Vitals. Lighthouse lab is listed under PageSpeed Lab and is not CWV.");
  } else if (cwv?.error) {
    parts.push(`PageSpeed Insights unavailable: ${cwv.error}`);
  } else {
    parts.push("Core Web Vitals from the last technical audit. Re-run the audit for LCP, INP, and CLS from Chrome UX Report.");
  }
  if (cwv?.field_available && typeof cwv.lab_performance === "number") {
    parts.push("Lighthouse lab is listed under PageSpeed Lab (simulated, not CWV).");
  }
  parts.push("Mobile Chrome UX Report for this URL only.");
  return parts.join(" ");
}

function labHeadline(cwv: any, lighthouse: any): string {
  if (typeof cwv?.lab_performance === "number") return `${cwv.lab_performance}/100 lab`;
  if (lighthouse?.label) return lighthouse.label;
  if (lighthouse?.status === "pass") return "Pass";
  if (lighthouse?.status === "fail") return "Fail";
  if (lighthouse?.status === "warning") return "Warning";
  if (cwv?.lab_fcp || cwv?.lab_ttfb) return "Lab metrics";
  return "Not checked";
}

function labSummary(cwv: any): string {
  const parts: string[] = [];
  if (typeof cwv?.lab_performance === "number") {
    parts.push(`Lighthouse mobile performance ${cwv.lab_performance}/100.`);
  }
  if (cwv?.lab_fcp) parts.push(`Lab FCP ${cwv.lab_fcp}.`);
  if (cwv?.lab_ttfb) parts.push(`Lab TTFB ${cwv.lab_ttfb}.`);
  if (cwv?.lab_lcp) parts.push(`Lab LCP ${cwv.lab_lcp}.`);
  if (cwv?.lab_cls) parts.push(`Lab CLS ${cwv.lab_cls}.`);
  const opps = Array.isArray(cwv?.opportunities) ? cwv.opportunities : [];
  if (opps.length) {
    const lines = opps.slice(0, 8).map((o: any) => {
      const title = o.title || o.id;
      if (o.display) return `${title} (${o.display})`;
      if (o.savings_ms > 0) return `${title} (~${o.savings_ms}ms)`;
      return title;
    }).filter(Boolean);
    if (lines.length) parts.push(`Opportunities: ${lines.join("; ")}.`);
  }
  if (typeof cwv?.desktop_performance === "number") {
    const desk = [`Desktop lab ${cwv.desktop_performance}/100`];
    if (cwv.desktop_fcp) desk.push(`FCP ${cwv.desktop_fcp}`);
    if (cwv.desktop_ttfb) desk.push(`TTFB ${cwv.desktop_ttfb}`);
    parts.push(`${desk.join(" · ")}.`);
  } else if (cwv?.desktop_error) {
    parts.push(`Desktop PageSpeed Insights unavailable: ${cwv.desktop_error}.`);
  }
  if (parts.length === 0) {
    parts.push("PageSpeed lab from the last technical audit. Re-run the audit for Lighthouse FCP, TTFB, opportunities, and desktop.");
  }
  parts.push("Lab is a simulated Lighthouse test on this URL — not Core Web Vitals and not a sitewide crawl.");
  return parts.join(" ");
}

function sitemapHeadline(sm: any): string {
  if (typeof sm?.url_count === "number" && sm.url_count > 0) return `${sm.url_count} URLs listed`;
  if (sm?.label) return sm.label;
  if (sm?.status === "pass") return "Pass";
  if (sm?.status === "fail") return "Fail";
  if (sm?.status === "warning") return "Warning";
  return "Not checked";
}

function sitemapSummary(sm: any): string {
  const parts: string[] = [];
  if (sm?.declared_from_robots) {
    const declared = (sm.declared || []).join(", ");
    parts.push(declared ? `Declared in robots.txt: ${declared}.` : "Using Sitemap: URL(s) from robots.txt.");
  } else if (sm?.fallback_url) {
    parts.push(`No Sitemap: in robots.txt; fetched ${sm.fallback_url}.`);
  }
  if (typeof sm?.url_count === "number" && sm.url_count > 0) {
    parts.push(`Counted ${sm.url_count} <loc> URL(s) in fetched sitemap XML.`);
  } else if (sm?.status === "pass" || sm?.status === "warning" || sm?.status === "fail") {
    parts.push("No <loc> URLs counted in fetched sitemap XML.");
  }
  if (sm?.index_children) {
    parts.push(`Sitemap index listed ${sm.index_children} child sitemap(s).`);
  }
  if (sm?.truncated) {
    parts.push("Fetch was capped (file size / child sitemaps) — this is not a full site inventory.");
  }
  if (sm?.sample?.length) {
    parts.push(`Sample: ${sm.sample.join(", ")}.`);
  }
  parts.push("Listed URLs are not crawled. This is not a sitewide inventory.");
  return parts.join(" ");
}

function indexabilityHeadline(idx: any): string {
  if (idx?.noindex) return "noindex";
  if (idx?.indexable) return "Indexable";
  if (idx?.label) return idx.label;
  if (idx?.status === "pass") return "Indexable";
  if (idx?.status === "fail") return "Blocked";
  if (idx?.status === "warning") return "Warning";
  return "Not checked";
}

function indexabilitySummary(idx: any): string {
  const parts: string[] = [];
  if (idx?.meta_robots) parts.push(`Meta robots: ${idx.meta_robots}.`);
  else parts.push("No meta robots tag (default is index).");
  if (idx?.x_robots_tag) parts.push(`X-Robots-Tag: ${idx.x_robots_tag}.`);
  else parts.push("No X-Robots-Tag header.");
  if (idx?.nofollow && !idx?.noindex) parts.push("nofollow is set; the page may still be indexed.");
  if (idx?.canonical) parts.push(`Canonical: ${idx.canonical}.`);
  else parts.push("No canonical tag on this page.");
  parts.push("Checked this page only — not a sitewide indexability crawl.");
  return parts.join(" ");
}

export default function SiteExplorerPage() {
  const [project, setProject] = useState<any>(null);
  const [auditResult, setAuditResult] = useState<any>(null);
  const [issues, setIssues] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [activeTab, setActiveTab] = useState<"checks" | "issues">("checks");

  useEffect(() => {
    (async () => {
      try {
        const pRes = await dashboardApi.getProjects();
        const projects = pRes.data?.data ?? pRes.data ?? [];
        if (projects.length > 0) {
          const p = projects[projects.length - 1];
          setProject(p);
          // Load existing issues
          const iRes = await dashboardApi.getSeoIssues(p.id);
          const issueData = iRes.data?.data ?? iRes.data ?? [];
          setIssues(Array.isArray(issueData) ? issueData : []);
          // Load audit status
          try {
            const aRes = await dashboardApi.getSeoAudit(p.id);
            const status = aRes.data?.data ?? aRes.data;
            if (status?.score != null) {
              setAuditResult({
                score: status.score,
                health: status.health,
                crawled_at: status.last_updated,
                robots: status.robots,
                https: status.https,
                cwv: status.cwv,
                lighthouse: status.lighthouse,
                sitemap: status.sitemap,
                indexability: status.indexability,
              });
            }
          } catch {}
        }
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const handleRunAudit = async () => {
    if (!project) return;
    setRunning(true);
    setAuditResult(null);
    try {
      const res = await dashboardApi.runSeoAudit(project.id, project.url);
      const result = res.data?.data ?? res.data;
      setAuditResult(result);
      // Reload issues after audit
      const iRes = await dashboardApi.getSeoIssues(project.id);
      const issueData = iRes.data?.data ?? iRes.data ?? [];
      setIssues(Array.isArray(issueData) ? issueData : []);
      setActiveTab("checks");
    } catch (err) {
      console.error(err);
    } finally {
      setRunning(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  const score = auditResult?.score ?? project?.health_score ?? 0;
  const checks: any[] = auditResult?.checks ?? [];
  const healthLabel = score >= 75 ? "Healthy" : score >= 50 ? "Needs Work" : score > 0 ? "Critical" : "Not Audited";
  const healthColor = score >= 75 ? "emerald" : score >= 50 ? "amber" : score > 0 ? "rose" : "slate";
  const circumference = 2 * Math.PI * 54;

  const passCount = checks.filter((c) => c.status === "pass").length;
  const warnCount = checks.filter((c) => c.status === "warning").length;
  const failCount = checks.filter((c) => c.status === "fail").length;

  return (
    <div className="space-y-10 max-w-7xl mx-auto pb-40 pt-4">
      <DashboardHeader project={project} onAddSource={() => {}} />

      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Technical SEO</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Site Explorer</p>
          {project?.url && (
            <a href={project.url} target="_blank" rel="noopener noreferrer"
              className="text-sm text-slate-400 hover:text-slate-600 flex items-center gap-1.5 transition-colors">
              {project.url} <ExternalLink className="w-3.5 h-3.5" />
            </a>
          )}
        </div>

        <Button
          onClick={handleRunAudit}
          disabled={running || !project}
          className="rounded-xl bg-slate-900 text-white px-6 h-11 font-semibold gap-2 hover:bg-slate-800 disabled:opacity-60 shrink-0"
        >
          {running ? (
            <><RefreshCw className="w-4 h-4 animate-spin" /> Analyzing...</>
          ) : (
            <><Search className="w-4 h-4" /> Run Technical Audit</>
          )}
        </Button>
      </div>

      {/* Score card + stats */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <Card className="lg:col-span-1 border-slate-100 shadow-xl shadow-slate-200/20 rounded-3xl p-8 flex flex-col items-center justify-center text-center bg-white relative overflow-hidden">
          <div className={`absolute top-0 left-0 w-full h-1.5 bg-${healthColor}-500`} />
          <p className="text-[11px] font-bold text-slate-400 uppercase tracking-[0.2em] mb-6">Overall Health</p>
          <div className="relative mb-6">
            <svg className="w-28 h-28 transform -rotate-90">
              <circle cx="56" cy="56" r="54" stroke="currentColor" strokeWidth="7" fill="transparent" className="text-slate-50" />
              <circle
                cx="56" cy="56" r="54"
                stroke="currentColor" strokeWidth="7" fill="transparent"
                strokeDasharray={circumference}
                strokeDashoffset={circumference * (1 - score / 100)}
                className={`text-${healthColor}-500 transition-all duration-1000`}
              />
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-4xl font-bold text-slate-900 tracking-tighter">{score}</span>
              <span className="text-[10px] font-bold text-slate-400">/ 100</span>
            </div>
          </div>
          <Badge variant="outline" className={`bg-${healthColor}-50 text-${healthColor}-600 border-0 font-bold px-3 py-1`}>
            {healthLabel}
          </Badge>
          {auditResult?.load_time_ms ? (
            <p className="text-xs text-slate-400 mt-3 flex items-center gap-1">
              <Clock className="w-3.5 h-3.5" /> {auditResult.load_time_ms}ms load time
            </p>
          ) : null}
          {auditResult?.crawled_at ? (
            <p className="text-xs text-slate-400 mt-1">
              Last checked {new Date(auditResult.crawled_at).toLocaleString()}
            </p>
          ) : null}
        </Card>

        <div className="lg:col-span-2 grid grid-cols-2 gap-5">
          {[
            { label: "Checks Passed", value: checks.length > 0 ? `${passCount}/${checks.length}` : "—", icon: ShieldCheck, color: "emerald", warn: false },
            { label: "Warnings", value: warnCount > 0 ? String(warnCount) : "0", icon: AlertTriangle, color: "amber", warn: warnCount > 0 },
            { label: "Critical Issues", value: failCount > 0 ? String(failCount) : "0", icon: AlertCircle, color: "rose", warn: failCount > 0 },
            { label: "Open Issues (DB)", value: String(issues.length), icon: Globe, color: "slate", warn: issues.length > 5 },
          ].map((stat, i) => (
            <Card key={i} className="border-slate-100 shadow-none bg-slate-50/50 rounded-2xl p-5 hover:bg-white hover:border-slate-200 transition-all flex items-center gap-5">
              <div className={`w-11 h-11 rounded-xl bg-white border border-slate-100 flex items-center justify-center ${stat.warn ? `text-${stat.color}-500` : "text-slate-400"}`}>
                <stat.icon className="w-5 h-5" />
              </div>
              <div>
                <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">{stat.label}</p>
                <p className={`text-2xl font-bold ${stat.warn ? `text-${stat.color}-600` : "text-slate-900"} mt-0.5`}>{stat.value}</p>
              </div>
            </Card>
          ))}
        </div>
      </div>

      {(auditResult?.robots || auditResult?.https || auditResult?.cwv || auditResult?.lighthouse || auditResult?.sitemap || auditResult?.indexability) && (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-5">
          {auditResult?.robots && (
            <Card className="border-slate-100 rounded-2xl p-6 bg-white">
              <div className="flex items-start gap-4">
                <div className="w-11 h-11 rounded-xl bg-slate-50 border border-slate-100 flex items-center justify-center text-slate-500 shrink-0">
                  <FileText className="w-5 h-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">robots.txt</p>
                  <p className="text-xl font-semibold text-slate-900 mt-0.5">
                    {auditResult.robots.label
                      ?? (auditResult.robots.status === "pass" ? "Pass"
                        : auditResult.robots.status === "fail" ? "Fail"
                        : auditResult.robots.status === "warning" ? "Warning"
                        : auditResult.robots.available ? "Found" : "Not found")}
                  </p>
                  <p className="text-sm text-slate-500 mt-2">
                    {auditResult.robots.blocks_all
                      ? "Disallow: / blocks the entire site."
                      : auditResult.robots.blocked_important?.length
                        ? `Blocked: ${auditResult.robots.blocked_important.join(", ")}`
                        : auditResult.robots.sitemap_declared
                          ? `Sitemap declared: ${(auditResult.robots.sitemaps || []).join(", ") || "yes"}`
                          : auditResult.robots.available === false
                            ? "File missing or not readable. Search engines treat a 404 as allow-all."
                            : "Parsed on the last technical audit."}
                  </p>
                  {(auditResult.robots.checked_at || auditResult.crawled_at) && (
                    <p className="text-xs text-slate-400 mt-2">
                      Last checked {new Date(auditResult.robots.checked_at || auditResult.crawled_at).toLocaleString()}
                    </p>
                  )}
                </div>
              </div>
            </Card>
          )}
          {auditResult?.https && (
            <Card className="border-slate-100 rounded-2xl p-6 bg-white">
              <div className="flex items-start gap-4">
                <div className="w-11 h-11 rounded-xl bg-slate-50 border border-slate-100 flex items-center justify-center text-slate-500 shrink-0">
                  <Lock className="w-5 h-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">HTTPS</p>
                  <p className="text-xl font-semibold text-slate-900 mt-0.5">
                    {httpsHeadline(auditResult.https)}
                  </p>
                  <p className="text-sm text-slate-500 mt-2">
                    {httpsSummary(auditResult.https)}
                  </p>
                  {(auditResult.https.checked_at || auditResult.crawled_at) && (
                    <p className="text-xs text-slate-400 mt-2">
                      Last checked {new Date(auditResult.https.checked_at || auditResult.crawled_at).toLocaleString()}
                    </p>
                  )}
                </div>
              </div>
            </Card>
          )}
          {auditResult?.cwv && (
            <Card className="border-slate-100 rounded-2xl p-6 bg-white">
              <div className="flex items-start gap-4">
                <div className="w-11 h-11 rounded-xl bg-slate-50 border border-slate-100 flex items-center justify-center text-slate-500 shrink-0">
                  <Gauge className="w-5 h-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Core Web Vitals</p>
                  <p className="text-xl font-semibold text-slate-900 mt-0.5">
                    {cwvHeadline(auditResult.cwv)}
                  </p>
                  <p className="text-sm text-slate-500 mt-2">
                    {cwvSummary(auditResult.cwv)}
                  </p>
                  {(auditResult.cwv.checked_at || auditResult.crawled_at) && (
                    <p className="text-xs text-slate-400 mt-2">
                      Last checked {new Date(auditResult.cwv.checked_at || auditResult.crawled_at).toLocaleString()}
                    </p>
                  )}
                </div>
              </div>
            </Card>
          )}
          {(auditResult?.cwv || auditResult?.lighthouse) && (
            <Card className="border-slate-100 rounded-2xl p-6 bg-white">
              <div className="flex items-start gap-4">
                <div className="w-11 h-11 rounded-xl bg-slate-50 border border-slate-100 flex items-center justify-center text-slate-500 shrink-0">
                  <Timer className="w-5 h-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">PageSpeed Lab</p>
                  <p className="text-xl font-semibold text-slate-900 mt-0.5">
                    {labHeadline(auditResult.cwv, auditResult.lighthouse)}
                  </p>
                  <p className="text-sm text-slate-500 mt-2">
                    {labSummary(auditResult.cwv)}
                  </p>
                  {(auditResult.cwv?.checked_at || auditResult.lighthouse?.checked_at || auditResult.crawled_at) && (
                    <p className="text-xs text-slate-400 mt-2">
                      Last checked {new Date(auditResult.cwv?.checked_at || auditResult.lighthouse?.checked_at || auditResult.crawled_at).toLocaleString()}
                    </p>
                  )}
                </div>
              </div>
            </Card>
          )}
          {auditResult?.sitemap?.status && (
            <Card className="border-slate-100 rounded-2xl p-6 bg-white">
              <div className="flex items-start gap-4">
                <div className="w-11 h-11 rounded-xl bg-slate-50 border border-slate-100 flex items-center justify-center text-slate-500 shrink-0">
                  <ListTree className="w-5 h-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">XML Sitemap</p>
                  <p className="text-xl font-semibold text-slate-900 mt-0.5">
                    {sitemapHeadline(auditResult.sitemap)}
                  </p>
                  <p className="text-sm text-slate-500 mt-2">
                    {sitemapSummary(auditResult.sitemap)}
                  </p>
                  {(auditResult.sitemap.checked_at || auditResult.crawled_at) && (
                    <p className="text-xs text-slate-400 mt-2">
                      Last checked {new Date(auditResult.sitemap.checked_at || auditResult.crawled_at).toLocaleString()}
                    </p>
                  )}
                </div>
              </div>
            </Card>
          )}
          {auditResult?.indexability?.status && (
            <Card className="border-slate-100 rounded-2xl p-6 bg-white">
              <div className="flex items-start gap-4">
                <div className="w-11 h-11 rounded-xl bg-slate-50 border border-slate-100 flex items-center justify-center text-slate-500 shrink-0">
                  <ScanSearch className="w-5 h-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Indexability</p>
                  <p className="text-xl font-semibold text-slate-900 mt-0.5">
                    {indexabilityHeadline(auditResult.indexability)}
                  </p>
                  <p className="text-sm text-slate-500 mt-2">
                    {indexabilitySummary(auditResult.indexability)}
                  </p>
                  {(auditResult.indexability.checked_at || auditResult.crawled_at) && (
                    <p className="text-xs text-slate-400 mt-2">
                      Last checked {new Date(auditResult.indexability.checked_at || auditResult.crawled_at).toLocaleString()}
                    </p>
                  )}
                </div>
              </div>
            </Card>
          )}
        </div>
      )}

      {/* Tabs: Checks | Issues */}
      {(checks.length > 0 || issues.length > 0) && (
        <div className="space-y-6">
          <div className="flex gap-1 bg-slate-100 rounded-xl p-1 w-fit">
            {([
              { id: "checks", label: `Checks (${checks.length})` },
              { id: "issues", label: `Saved Issues (${issues.length})` },
            ] as const).map((tab) => (
              <button key={tab.id} onClick={() => setActiveTab(tab.id)}
                className={`px-5 py-2 rounded-lg text-sm font-semibold transition-all ${
                  activeTab === tab.id
                    ? "bg-white text-slate-900 shadow-sm"
                    : "text-slate-500 hover:text-slate-700"
                }`}>
                {tab.label}
              </button>
            ))}
          </div>

          <AnimatePresence mode="wait">
            {activeTab === "checks" && checks.length > 0 && (
              <motion.div key="checks" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }}
                className="bg-white border border-slate-100 rounded-3xl overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full text-left">
                    <thead>
                      <tr className="border-b border-slate-50 bg-slate-50/60">
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Check</th>
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Category</th>
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Status</th>
                        <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Detail</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-50">
                      {checks.map((check: any, i: number) => {
                        const s = statusConfig[check.status] ?? statusConfig.fail;
                        return (
                          <motion.tr key={i}
                            initial={{ opacity: 0, y: 4 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.03 }}
                            className="group hover:bg-slate-50/50 transition-colors">
                            <td className="px-6 py-4 font-semibold text-sm text-slate-900">{check.label}</td>
                            <td className="px-6 py-4">
                              <Badge variant="outline" className="text-[10px] capitalize text-slate-500 bg-slate-50 border-slate-100">{check.category}</Badge>
                            </td>
                            <td className="px-6 py-4">
                              <span className="flex items-center gap-2">
                                <span className={`w-2 h-2 rounded-full ${s.dot}`} />
                                <span className={`text-xs font-bold ${
                                  check.status === "pass" ? "text-emerald-600" :
                                  check.status === "warning" ? "text-amber-600" : "text-rose-600"
                                }`}>{s.label}</span>
                              </span>
                            </td>
                            <td className="px-6 py-4 text-xs text-slate-500 max-w-md">
                              <p>{check.detail}</p>
                              {check.recommendation ? (
                                <p className="text-slate-400 mt-1">{check.recommendation}</p>
                              ) : null}
                            </td>
                          </motion.tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </motion.div>
            )}

            {activeTab === "issues" && (
              <motion.div key="issues" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }}
                className="bg-white border border-slate-100 rounded-3xl overflow-hidden">
                {issues.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-20 text-center">
                    <CheckCircle2 className="w-12 h-12 text-emerald-400 mb-4" />
                    <p className="text-slate-900 font-semibold text-lg">No open issues</p>
                    <p className="text-slate-400 text-sm mt-1">Run an audit to find and save technical issues.</p>
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-left">
                      <thead>
                        <tr className="border-b border-slate-50 bg-slate-50/60">
                          <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Severity</th>
                          <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Category</th>
                          <th className="px-6 py-4 text-[10px] font-bold text-slate-400 uppercase tracking-widest">Detail</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-50">
                        {issues.map((issue: any, i: number) => {
                          const sc = severityConfig[issue.severity] ?? severityConfig.low;
                          const Icon = sc.icon;
                          return (
                            <motion.tr key={issue.id ?? i}
                              initial={{ opacity: 0, y: 4 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.03 }}
                              className="hover:bg-slate-50/50 transition-colors">
                              <td className="px-6 py-4">
                                <span className={`inline-flex items-center gap-1.5 ${sc.bg} ${sc.border} border rounded-full px-2.5 py-1`}>
                                  <Icon className={`w-3.5 h-3.5 ${sc.color}`} />
                                  <span className={`text-[10px] font-bold capitalize ${sc.color}`}>{issue.severity}</span>
                                </span>
                              </td>
                              <td className="px-6 py-4">
                                <Badge variant="outline" className="text-[10px] capitalize text-slate-500 bg-slate-50 border-slate-100">{issue.category}</Badge>
                              </td>
                              <td className="px-6 py-4 text-xs text-slate-600 max-w-md">{issue.detail}</td>
                            </motion.tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      )}

      {/* Empty state */}
      {checks.length === 0 && issues.length === 0 && !running && (
        <div className="bg-white border border-slate-100 rounded-3xl flex flex-col items-center justify-center py-24 text-center">
          <div className="w-16 h-16 bg-slate-50 rounded-2xl flex items-center justify-center mb-6">
            <Search className="w-8 h-8 text-slate-300" />
          </div>
          <p className="text-slate-900 font-semibold text-lg">No audit data yet</p>
          <p className="text-slate-400 text-sm mt-2 max-w-sm">
            Click "Run Technical Audit" to crawl this page and get a real SEO health report.
          </p>
        </div>
      )}
    </div>
  );
}
