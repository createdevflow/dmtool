"use client";

import { cn } from "@/lib/utils";
import { motion } from "framer-motion";
import { 
  Bell, CheckCircle2, AlertCircle, Info, Clock,
  Loader2, Trash2, MailOpen
} from "lucide-react";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

const typeIcon = (type: string) => {
  switch (type) {
    case "SUCCESS": return <CheckCircle2 className="w-6 h-6" />;
    case "WARNING": return <AlertCircle className="w-6 h-6" />;
    case "DANGER": return <AlertCircle className="w-6 h-6" />;
    default: return <Info className="w-6 h-6" />;
  }
};

const typeStyles = (type: string) => {
  switch (type) {
    case "SUCCESS": return "bg-emerald-50 text-emerald-600";
    case "WARNING": return "bg-amber-50 text-amber-600";
    case "DANGER": return "bg-rose-50 text-rose-600";
    default: return "bg-blue-50 text-blue-600";
  }
};

function timeAgo(dateStr: string) {
  const date = new Date(dateStr);
  const secs = Math.floor((Date.now() - date.getTime()) / 1000);
  if (secs < 60) return "just now";
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export default function AlertsPage() {
  const [alerts, setAlerts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);

  const fetchData = async (targetProjectId?: number) => {
    setLoading(true);
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data.data || [];
      setProjects(allProjects);

      if (allProjects.length > 0) {
        const selected = targetProjectId 
          ? allProjects.find((p: any) => p.id === targetProjectId) || allProjects[allProjects.length - 1]
          : allProjects[allProjects.length - 1];
        
        setProject(selected);

        const aRes = await dashboardApi.getAlerts(selected.id);
        setAlerts(aRes.data.data ?? []);
      }

    } catch (err) {
      console.error("Failed to fetch alerts", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchData(); }, []);

  const handleProjectChange = (newProj: any) => {
    fetchData(newProj.id);
  };

  const markAllRead = () => {
    setAlerts(prev => prev.map(a => ({ ...a, is_read: true })));
  };

  const clearAll = () => setAlerts([]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  const unreadCount = alerts.filter(a => !a.is_read).length;

  return (
    <div className="space-y-12 max-w-7xl mx-auto pb-40 pt-4">
      
      <DashboardHeader 
        project={project} 
        projects={projects}
        onProjectChange={handleProjectChange}
        onAddSource={() => {}} 
      />

      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">System Events</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">
            Alerts & Notifications
            {unreadCount > 0 && (
              <span className="ml-3 text-sm font-bold text-rose-500 bg-rose-50 px-2.5 py-0.5 rounded-full">{unreadCount} new</span>
            )}
          </p>
        </div>
        
        <div className="flex items-center gap-3">
          <Button onClick={markAllRead} variant="ghost" className="rounded-xl h-10 px-4 text-xs font-semibold text-slate-500 hover:bg-slate-50 gap-2">
            <MailOpen className="w-4 h-4" /> Mark all as read
          </Button>
          <Button onClick={clearAll} variant="ghost" className="rounded-xl h-10 px-4 text-xs font-semibold text-rose-500 hover:bg-rose-50 gap-2">
            <Trash2 className="w-4 h-4" /> Clear all
          </Button>
        </div>
      </div>

      <div className="space-y-4">
        {alerts.length > 0 ? (
          alerts.map((alert, i) => (
            <motion.div
              key={alert.id ?? i}
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: i * 0.05 }}
            >
              <Card className={cn(
                "border-slate-100 shadow-none rounded-2xl transition-all group overflow-hidden",
                !alert.is_read ? "bg-white ring-1 ring-slate-200" : "bg-slate-50/50 opacity-75"
              )}>
                <CardContent className="p-0">
                  <div className="flex items-center gap-6 p-6">
                    <div className={cn("w-12 h-12 rounded-xl flex items-center justify-center shrink-0", typeStyles(alert.type))}>
                      {typeIcon(alert.type)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between mb-1">
                        <h4 className="font-semibold text-slate-900 truncate pr-4">{alert.title}</h4>
                        <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest flex items-center gap-1.5 whitespace-nowrap">
                          <Clock className="w-3 h-3" />
                          {alert.created_at ? timeAgo(alert.created_at) : "recently"}
                        </span>
                      </div>
                      <p className="text-sm text-slate-500 font-medium line-clamp-1">{alert.message}</p>
                    </div>
                    <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button
                        variant="ghost" size="icon"
                        className="rounded-lg h-10 w-10 text-slate-400 hover:text-slate-900"
                        onClick={() => setAlerts(prev => prev.map(a => a.id === alert.id ? { ...a, is_read: true } : a))}
                      >
                        <MailOpen className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost" size="icon"
                        className="rounded-lg h-10 w-10 text-slate-400 hover:text-rose-500"
                        onClick={() => setAlerts(prev => prev.filter(a => a.id !== alert.id))}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          ))
        ) : (
          <div className="rounded-3xl border-2 border-dashed border-slate-100 flex flex-col items-center justify-center p-24 text-center">
            <div className="w-16 h-16 rounded-2xl bg-slate-50 flex items-center justify-center mb-6">
              <Bell className="w-8 h-8 text-slate-200" />
            </div>
            <h3 className="text-lg font-semibold text-slate-900 mb-2">Inbox zero</h3>
            <p className="text-slate-500 max-w-sm mx-auto text-sm leading-relaxed">
              No new notifications. We'll alert you when something significant happens with your growth.
            </p>
          </div>
        )}
      </div>

      {/* Recent Activity — derived from alerts history */}
      {alerts.length > 0 && (
        <div className="pt-8 space-y-8">
          <div className="space-y-1">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">History</h2>
            <p className="text-2xl font-semibold text-slate-900 tracking-tight">Recent Activity</p>
          </div>
          
          <div className="bg-white border border-slate-100 rounded-2xl divide-y divide-slate-50">
            {alerts.slice(0, 5).map((alert, i) => (
              <div key={alert.id ?? i} className="flex items-center justify-between p-5 hover:bg-slate-50/50 transition-colors">
                <div className="flex items-center gap-4">
                  <div className={cn(
                    "w-2 h-2 rounded-full",
                    alert.type === "SUCCESS" ? "bg-emerald-500" :
                    alert.type === "DANGER" ? "bg-rose-500" : "bg-amber-500"
                  )} />
                  <p className="text-sm font-medium text-slate-700">{alert.message}</p>
                </div>
                <p className="text-[11px] font-semibold text-slate-400 whitespace-nowrap">
                  {alert.created_at ? timeAgo(alert.created_at) : "recently"}
                </p>
              </div>
            ))}
          </div>
        </div>
      )}

    </div>
  );
}
