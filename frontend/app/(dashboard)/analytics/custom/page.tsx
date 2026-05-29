"use client";

import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import {
  BarChart2, Calendar, Download, RefreshCw, Loader2,
  TrendingUp, MousePointer2, Eye, Activity, ChevronDown
} from "lucide-react";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, LineChart, Line, Legend
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";

const RANGES = [
  { label: "Last 7 days", days: 7 },
  { label: "Last 14 days", days: 14 },
  { label: "Last 30 days", days: 30 },
  { label: "Last 90 days", days: 90 },
];

const METRICS = [
  { key: "clicks", label: "Clicks", color: "#0f172a" },
  { key: "impressions", label: "Impressions", color: "#3b82f6" },
  { key: "reach", label: "Reach", color: "#10b981" },
  { key: "engagement", label: "Engagement", color: "#f59e0b" },
];

export default function CustomAnalyticsPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [selectedRange, setSelectedRange] = useState(RANGES[2]);
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>(["clicks", "impressions"]);
  const [metrics, setMetrics] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [chartType, setChartType] = useState<"bar" | "line">("line");

  const fetchData = async (targetProjectId?: number, days = selectedRange.days) => {
    setLoading(true);
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data.data ?? [];
      setProjects(allProjects);

      if (allProjects.length > 0) {
        let savedProjectId = null;
        try { savedProjectId = parseInt(localStorage.getItem("dmtool_active_project_id") || "0"); } catch (e) {}
        const defaultProject = allProjects.find((p: any) => p.id === savedProjectId) || allProjects[allProjects.length - 1];
        const selected = targetProjectId
          ? allProjects.find((p: any) => p.id === targetProjectId) || defaultProject
          : defaultProject;
        if (selected) localStorage.setItem("dmtool_active_project_id", selected.id.toString());
        setProject(selected);

        const res = await dashboardApi.getMetrics(selected.id, days);
        const raw = res.data?.data;
        setMetrics(Array.isArray(raw) ? raw : []);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  const chartData = metrics.map((m: any) => ({
    date: m.date ? new Date(m.date).toLocaleDateString("en-US", { month: "short", day: "numeric" }) : (m.name ?? ""),
    clicks: m.clicks || 0,
    impressions: m.impressions || 0,
    reach: m.reach || 0,
    engagement: m.engagement || 0,
  }));

  const totals = selectedMetrics.reduce((acc: any, key) => {
    acc[key] = metrics.reduce((s: number, m: any) => s + (m[key] || 0), 0);
    return acc;
  }, {} as Record<string, number>);

  const exportCSV = () => {
    setExporting(true);
    const headers = ["date", ...selectedMetrics].join(",");
    const rows = chartData.map((row: any) =>
      [row.date, ...selectedMetrics.map((k) => row[k] || 0)].join(",")
    );
    const csv = [headers, ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${project?.name ?? "analytics"}_${selectedRange.label.replace(/ /g, "_")}.csv`;
    a.click();
    URL.revokeObjectURL(url);
    setTimeout(() => setExporting(false), 500);
  };

  const toggleMetric = (key: string) => {
    setSelectedMetrics((prev) =>
      prev.includes(key)
        ? prev.length > 1 ? prev.filter((k) => k !== key) : prev
        : [...prev, key]
    );
  };

  return (
    <div className="space-y-10 max-w-7xl mx-auto pb-32 pt-4">
      <DashboardHeader
        project={project}
        projects={projects}
        onProjectChange={(p: any) => fetchData(p.id)}
        onAddSource={() => {}}
      />

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Custom Reports</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Analytics Builder</p>
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <Button
            variant="outline"
            className="rounded-xl h-10 border-slate-200 font-semibold gap-2 text-sm"
            onClick={exportCSV}
            disabled={exporting || metrics.length === 0}
          >
            <Download className="w-4 h-4" />
            Export CSV
          </Button>
          <Button
            variant="outline"
            className="rounded-xl h-10 border-slate-200 font-semibold gap-2 text-sm"
            onClick={() => fetchData(project?.id)}
            disabled={loading}
          >
            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
            Refresh
          </Button>
        </div>
      </div>

      {/* Controls */}
      <div className="flex flex-col md:flex-row gap-4">
        {/* Date range selector */}
        <div className="flex bg-slate-50 rounded-2xl p-1.5 border border-slate-100 gap-1">
          {RANGES.map((r) => (
            <button
              key={r.days}
              onClick={() => { setSelectedRange(r); fetchData(project?.id, r.days); }}
              className={`px-4 py-2 rounded-xl text-xs font-bold uppercase tracking-wider transition-all ${
                selectedRange.days === r.days
                  ? "bg-white text-slate-900 shadow-sm border border-slate-200"
                  : "text-slate-400 hover:text-slate-700"
              }`}
            >
              {r.label}
            </button>
          ))}
        </div>

        {/* Chart type toggle */}
        <div className="flex bg-slate-50 rounded-2xl p-1.5 border border-slate-100 gap-1">
          {(["line", "bar"] as const).map((type) => (
            <button
              key={type}
              onClick={() => setChartType(type)}
              className={`px-4 py-2 rounded-xl text-xs font-bold uppercase tracking-wider transition-all ${
                chartType === type
                  ? "bg-white text-slate-900 shadow-sm border border-slate-200"
                  : "text-slate-400 hover:text-slate-700"
              }`}
            >
              {type === "line" ? "Line" : "Bar"}
            </button>
          ))}
        </div>
      </div>

      {/* Metric Selector */}
      <div className="flex flex-wrap gap-3">
        {METRICS.map((m) => {
          const isActive = selectedMetrics.includes(m.key);
          return (
            <button
              key={m.key}
              onClick={() => toggleMetric(m.key)}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-bold border transition-all ${
                isActive
                  ? "border-slate-900 bg-slate-900 text-white"
                  : "border-slate-200 text-slate-500 hover:border-slate-300"
              }`}
            >
              <div className="w-2 h-2 rounded-full" style={{ backgroundColor: isActive ? "white" : m.color }} />
              {m.label}
            </button>
          );
        })}
      </div>

      {/* Totals Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {METRICS.filter((m) => selectedMetrics.includes(m.key)).map((m, i) => (
          <motion.div key={m.key} initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}>
            <Card className="border-slate-100 shadow-none rounded-2xl">
              <CardContent className="p-5">
                <div className="w-3 h-3 rounded-full mb-4" style={{ backgroundColor: m.color }} />
                <p className="text-2xl font-bold text-slate-900">{(totals[m.key] ?? 0).toLocaleString()}</p>
                <p className="text-[11px] font-bold text-slate-400 uppercase tracking-widest mt-1">Total {m.label}</p>
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      {/* Chart */}
      <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
        <CardHeader className="px-8 pt-8 pb-0">
          <CardTitle className="text-base font-semibold">
            {selectedMetrics.map((k) => METRICS.find((m) => m.key === k)?.label).join(" vs ")} — {selectedRange.label}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-8">
          {loading ? (
            <div className="h-72 flex items-center justify-center">
              <Loader2 className="w-7 h-7 animate-spin text-slate-300" />
            </div>
          ) : chartData.length === 0 ? (
            <div className="h-72 flex flex-col items-center justify-center text-center gap-3">
              <BarChart2 className="w-8 h-8 text-slate-200" />
              <p className="text-slate-400 font-semibold text-sm">No data for this period</p>
              <p className="text-xs text-slate-300">Try a longer date range or connect Google Search Console.</p>
            </div>
          ) : (
            <div className="h-[380px]">
              <ResponsiveContainer width="100%" height="100%">
                {chartType === "line" ? (
                  <LineChart data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                    <XAxis dataKey="date" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={10} />
                    <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} />
                    <Tooltip
                      contentStyle={{ borderRadius: "12px", border: "1px solid #f1f5f9", backgroundColor: "#fff", boxShadow: "0 10px 25px rgba(0,0,0,0.08)" }}
                      itemStyle={{ fontSize: "12px", fontWeight: 600 }}
                    />
                    <Legend iconType="circle" iconSize={8} />
                    {METRICS.filter((m) => selectedMetrics.includes(m.key)).map((m) => (
                      <Line key={m.key} type="monotone" dataKey={m.key} name={m.label} stroke={m.color} strokeWidth={2.5} dot={false} />
                    ))}
                  </LineChart>
                ) : (
                  <BarChart data={chartData} barGap={4} barSize={8}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                    <XAxis dataKey="date" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} dy={10} />
                    <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "#94a3b8" }} />
                    <Tooltip
                      contentStyle={{ borderRadius: "12px", border: "1px solid #f1f5f9", backgroundColor: "#fff" }}
                      itemStyle={{ fontSize: "12px", fontWeight: 600 }}
                    />
                    <Legend iconType="circle" iconSize={8} />
                    {METRICS.filter((m) => selectedMetrics.includes(m.key)).map((m) => (
                      <Bar key={m.key} dataKey={m.key} name={m.label} fill={m.color} radius={[4, 4, 0, 0]} />
                    ))}
                  </BarChart>
                )}
              </ResponsiveContainer>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Data Table */}
      {chartData.length > 0 && (
        <Card className="border-slate-100 shadow-none rounded-2xl overflow-hidden">
          <CardHeader className="px-8 pt-8 pb-4">
            <CardTitle className="text-base font-semibold">Raw Data</CardTitle>
          </CardHeader>
          <CardContent className="px-0 pb-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-t border-slate-50">
                    <th className="text-left text-[10px] font-bold text-slate-400 uppercase tracking-widest px-8 py-3">Date</th>
                    {METRICS.filter((m) => selectedMetrics.includes(m.key)).map((m) => (
                      <th key={m.key} className="text-right text-[10px] font-bold text-slate-400 uppercase tracking-widest px-6 py-3">{m.label}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {chartData.slice(-14).reverse().map((row: any, i: number) => (
                    <tr key={i} className="border-t border-slate-50 hover:bg-slate-50/50 transition-colors">
                      <td className="px-8 py-3 font-semibold text-slate-700">{row.date}</td>
                      {METRICS.filter((m) => selectedMetrics.includes(m.key)).map((m) => (
                        <td key={m.key} className="px-6 py-3 text-right font-semibold text-slate-900">{(row[m.key] ?? 0).toLocaleString()}</td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
