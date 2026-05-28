"use client";

import { motion, AnimatePresence } from "framer-motion";
import {
  Globe, Search, ShieldCheck, AlertCircle,
  Zap, Loader2, CheckCircle2, RefreshCw,
  ExternalLink, Clock, AlertTriangle, Info,
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
              setAuditResult({ score: status.score, health: status.health });
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
          {auditResult?.load_time_ms && (
            <p className="text-xs text-slate-400 mt-3 flex items-center gap-1">
              <Clock className="w-3.5 h-3.5" /> {auditResult.load_time_ms}ms load time
            </p>
          )}
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
                            <td className="px-6 py-4 text-xs text-slate-500 max-w-xs">{check.detail}</td>
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
            Click "Run Technical Audit" to crawl your site and get a real SEO health report with 13 checks.
          </p>
        </div>
      )}
    </div>
  );
}
