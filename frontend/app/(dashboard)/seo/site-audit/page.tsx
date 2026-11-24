"use client";

import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { SiteAuditPanel } from "@/components/seo/site-audit";

export default function SiteAuditPage() {
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
      <div className="-mx-6 -mt-6 lg:-mx-8 lg:-mt-8 min-h-[calc(100vh-4rem)] bg-[#f4f5f7] flex items-center justify-center">
        <Loader2 className="w-6 h-6 animate-spin text-[#ff642d]" />
      </div>
    );
  }

  return (
    <div className="-mx-6 -mt-6 lg:-mx-8 lg:-mt-8 min-h-[calc(100vh-4rem)] bg-[#f4f5f7]">
      <SiteAuditPanel
        projects={projects}
        project={project}
        onProjectChange={(p) => fetchData(p.id)}
      />
    </div>
  );
}
