"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { AlertList } from "@/components/dashboard/alert-list";
import { Card, CardContent } from "@/components/ui/card";
import { parseAlertsPayload, type StoredAlert } from "@/lib/alerts";

export default function AlertsPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [project, setProject] = useState<any>(null);
  const [projects, setProjects] = useState<any[]>([]);
  const [alerts, setAlerts] = useState<StoredAlert[]>([]);

  const fetchData = async (targetProjectId?: number) => {
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data?.data ?? pRes.data ?? [];
      setProjects(Array.isArray(allProjects) ? allProjects : []);

      if (allProjects.length > 0) {
        let savedProjectId = 0;
        try { savedProjectId = parseInt(localStorage.getItem("dmtool_active_project_id") || "0"); } catch {}
        const p = targetProjectId
          ? allProjects.find((x: any) => x.id === targetProjectId) || allProjects[allProjects.length - 1]
          : allProjects.find((x: any) => x.id === savedProjectId) || allProjects[allProjects.length - 1];
        setProject(p);
        if (p?.id) localStorage.setItem("dmtool_active_project_id", p.id.toString());

        const aRes = await dashboardApi.getAlerts(p.id);
        setAlerts(parseAlertsPayload(aRes.data));
      } else {
        setAlerts([]);
      }
    } catch (err) {
      console.error("Failed to fetch alerts", err);
      setAlerts([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  return (
    <div className="space-y-12 max-w-7xl mx-auto pb-40 pt-4">
      <DashboardHeader
        project={project}
        projects={projects}
        onProjectChange={(next) => {
          setLoading(true);
          fetchData(next.id);
        }}
        onAddSource={() => router.push("/integrations")}
      />

      <div className="space-y-1">
        <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Signals</h2>
        <p className="text-3xl font-semibold text-slate-900 tracking-tight">Alerts</p>
      </div>

      <Card className="border-slate-200/60 shadow-none bg-white rounded-3xl overflow-hidden">
        <CardContent className="p-8">
          <p className="text-xs text-slate-400 mb-6">
            From stored state: GSC, high-severity homepage issues, and stale DataForSEO cache. This page does not auto-fetch the vendor.
          </p>
          <AlertList
            alerts={alerts}
            emptyLabel={project ? "No alerts from stored state." : "Select a project to see alerts."}
          />
        </CardContent>
      </Card>
    </div>
  );
}
