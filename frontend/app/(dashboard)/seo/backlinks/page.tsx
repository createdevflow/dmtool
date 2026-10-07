"use client";

import { useEffect, useState } from "react";
import { Loader2, Link as LinkIcon } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { DomainExplorerPanel } from "@/components/seo/domain-explorer";

function fmtNum(n: unknown): string {
  const v = Number(n);
  if (!Number.isFinite(v) || v <= 0) return "—";
  return v.toLocaleString();
}

export default function BacklinksPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const fetchData = async (targetProjectId?: number) => {
    setLoading(true);
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data.data ?? [];
      setProjects(allProjects);
      if (allProjects.length > 0) {
        let savedProjectId = 0;
        try { savedProjectId = parseInt(localStorage.getItem("dmtool_active_project_id") || "0"); } catch {}
        const defaultProject = allProjects.find((p: any) => p.id === savedProjectId) || allProjects[allProjects.length - 1];
        const selected = targetProjectId
          ? allProjects.find((p: any) => p.id === targetProjectId) || defaultProject
          : defaultProject;
        if (selected) localStorage.setItem("dmtool_active_project_id", selected.id.toString());
        setProject(selected);
        const res = await dashboardApi.getBacklinks(selected.id);
        setData(res.data?.data ?? null);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  return (
    <div className="space-y-10 max-w-7xl mx-auto pb-32 pt-4">
      <DashboardHeader
        project={project}
        projects={projects}
        onProjectChange={(p: any) => fetchData(p.id)}
        onAddSource={() => {}}
      />

      <div className="space-y-1">
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">SEO Intelligence</h2>
        <p className="text-3xl font-semibold text-slate-900 tracking-tight">Backlinks</p>
        <p className="text-sm text-slate-500">
          DataForSEO Backlinks index when a domain lookup has run. Rank is not Moz Domain Authority.
        </p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: "Backlinks", value: data?.available ? fmtNum(data.total_backlinks) : "—" },
          { label: "Referring domains", value: data?.available ? fmtNum(data.referring_domains) : "—" },
          { label: "DataForSEO Rank", value: data?.available && data.rank > 0 ? fmtNum(data.rank) : "—" },
          { label: "Moz DA", value: "—" },
        ].map((s) => (
          <Card key={s.label} className="border-slate-100 shadow-none rounded-2xl">
            <CardContent className="p-6">
              <p className="text-2xl font-bold text-slate-900">{s.value}</p>
              <p className="text-[11px] font-bold uppercase tracking-widest text-slate-400 mt-1">{s.label}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="border-slate-100 shadow-none rounded-2xl">
        <CardHeader className="px-8 pt-8 pb-4">
          <CardTitle className="text-base font-semibold flex items-center gap-2">
            <LinkIcon className="w-4 h-4 text-slate-400" />
            Index status
          </CardTitle>
        </CardHeader>
        <CardContent className="px-8 pb-8">
          <p className="text-sm text-slate-600">{data?.message || "Backlink index is not connected."}</p>
          {data?.domain && (
            <p className="text-xs text-slate-400 mt-2">{data.domain}{data.cached ? " · cached" : ""}</p>
          )}
        </CardContent>
      </Card>

      <DomainExplorerPanel projectId={project?.id} defaultDomain={project?.url || ""} />
    </div>
  );
}
