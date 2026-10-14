"use client";

import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { BacklinkExplorerPanel } from "@/components/seo/backlink-explorer";

export default function BacklinksPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
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
          Referring domains and a backlink audit via DataForSEO. Rank is DataForSEO Rank, not Moz DA. This is not an outreach list.
        </p>
      </div>

      <BacklinkExplorerPanel projectId={project?.id} defaultDomain={project?.url} />
    </div>
  );
}
