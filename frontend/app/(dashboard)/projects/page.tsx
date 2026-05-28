"use client";

import { motion } from "framer-motion";
import { 
  Plus, Globe, MoreHorizontal, ArrowRight, Loader2, 
  Edit2, Trash2, Settings, ExternalLink, 
  Instagram, Twitter, Linkedin, Facebook 
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { 
  DropdownMenu, 
  DropdownMenuContent, 
  DropdownMenuItem, 
  DropdownMenuTrigger,
  DropdownMenuSeparator
} from "@/components/ui/dropdown-menu";
import Link from "next/link";
import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { dashboardApi } from "@/lib/api-client";
// Removed sonner toast as it's not installed

export default function ProjectsPage() {
  const router = useRouter();
  const [projects, setProjects] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchProjects = async () => {
    setLoading(true);
    try {
      const res = await dashboardApi.getProjects();
      const data = res.data?.data || res.data;
      setProjects(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error("Failed to fetch projects", err);
      setProjects([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  const handleDelete = async (id: number) => {
    if (!confirm("Are you sure you want to delete this project? All associated data will be lost.")) return;
    
    try {
      // In project_handlers.go, Delete is DELETE /api/projects/:id
      // But lib/api-client.ts doesn't have a direct deleteProject method, let's check
      await dashboardApi.deleteProject(id);
      fetchProjects();
    } catch (err) {
      console.error("Failed to delete project", err);
      alert("Failed to delete project");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-900" />
      </div>
    );
  }

  return (
    <div className="space-y-8 max-w-6xl mx-auto pb-20">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
          <h1 className="text-3xl font-bold tracking-tight text-slate-900">Projects</h1>
          <p className="text-sm text-slate-500 font-medium">Manage your connected websites and digital properties.</p>
        </motion.div>
        
        <motion.div initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} transition={{ duration: 0.3, delay: 0.1 }}>
          <Button 
            className="gap-2 bg-slate-900 hover:bg-slate-800 text-white rounded-xl shadow-lg shadow-slate-900/10 h-10 px-6 font-semibold"
            onClick={() => router.push("/projects/create")}
          >
            <Plus className="w-4 h-4" /> Add Project
          </Button>
        </motion.div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {projects.length === 0 ? (
          <div className="col-span-full py-20 text-center border-2 border-dashed border-slate-100 rounded-[32px]">
            <Globe className="w-12 h-12 text-slate-200 mx-auto mb-4" />
            <h3 className="text-lg font-bold text-slate-900">No projects yet</h3>
            <p className="text-slate-500 mb-6">Create your first project to start monitoring.</p>
            <Button variant="outline" className="rounded-xl px-8" onClick={() => router.push("/projects/create")}>
              Create Project
            </Button>
          </div>
        ) : (
          projects.map((project, i) => (
            <motion.div 
              key={project.id}
              initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4, delay: i * 0.1 }}
            >
              <Card className="border-slate-200 shadow-xl shadow-slate-200/20 hover:shadow-2xl hover:shadow-slate-200/40 transition-all rounded-[24px] flex flex-col group cursor-pointer bg-white">
                <CardHeader className="flex flex-row items-start justify-between pb-2 space-y-0 p-6">
                    <div className="flex items-center gap-3" onClick={() => {
                       localStorage.setItem("dmtool_selected_project", JSON.stringify(project));
                       router.push("/dashboard");
                    }}>
                        <div className="w-12 h-12 rounded-2xl bg-slate-50 flex items-center justify-center text-slate-900 border border-slate-100">
                          <Globe className="w-6 h-6" />
                        </div>
                        <div className="overflow-hidden">
                          <CardTitle className="text-lg font-bold truncate">{project.name}</CardTitle>
                          <div className="flex items-center gap-1 text-xs text-slate-400 font-medium">
                            <span className="truncate max-w-[150px]">{project.url}</span>
                            <ExternalLink className="w-3 h-3" />
                          </div>
                        </div>
                    </div>
                    
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <button className="w-8 h-8 rounded-lg flex items-center justify-center text-slate-400 hover:text-slate-900 hover:bg-slate-50 transition-all">
                          <MoreHorizontal className="w-5 h-5" />
                        </button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-48 rounded-xl p-1.5 shadow-2xl border-slate-100">
                        <DropdownMenuItem className="rounded-lg gap-2 cursor-pointer" onClick={() => router.push(`/projects/settings?project_id=${project.id}`)}>
                          <Settings className="w-4 h-4" /> Settings
                        </DropdownMenuItem>
                        <DropdownMenuItem className="rounded-lg gap-2 cursor-pointer" onClick={() => router.push(`/projects/settings?project_id=${project.id}&tab=general`)}>
                          <Edit2 className="w-4 h-4" /> Edit Details
                        </DropdownMenuItem>
                        <DropdownMenuSeparator className="bg-slate-50" />
                        <DropdownMenuItem 
                          className="rounded-lg gap-2 cursor-pointer text-rose-600 focus:text-rose-600 focus:bg-rose-50" 
                          onClick={() => handleDelete(project.id)}
                        >
                          <Trash2 className="w-4 h-4" /> Delete Project
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                </CardHeader>
                <CardContent className="p-6 pt-2 flex-1 flex flex-col" onClick={() => {
                    localStorage.setItem("dmtool_selected_project", JSON.stringify(project));
                    router.push("/dashboard");
                }}>
                    <div className="grid grid-cols-2 gap-4 text-sm mb-6 bg-slate-50/50 p-4 rounded-2xl border border-slate-100">
                        <div>
                          <div className="text-slate-400 text-[10px] uppercase font-bold tracking-widest mb-1">Health</div>
                          <div className="font-bold text-slate-900 text-base">{project.health_score || 0}%</div>
                        </div>
                        <div>
                          <div className="text-slate-400 text-[10px] uppercase font-bold tracking-widest mb-1">Status</div>
                          <div className="font-bold text-slate-900 flex items-center gap-1.5 text-base capitalize">
                              <div className={`w-2 h-2 rounded-full ${project.status === 'active' ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]' : 'bg-slate-300'}`} />
                              {project.status}
                          </div>
                        </div>
                    </div>
                    
                    <div className="flex items-center gap-2 mb-6">
                      {[
                        { val: project.ig_handle, icon: Instagram, color: "text-rose-500 bg-rose-50 border-rose-100" },
                        { val: project.twitter_handle, icon: Twitter, color: "text-slate-900 bg-slate-50 border-slate-200" },
                        { val: project.facebook_handle, icon: Facebook, color: "text-blue-600 bg-blue-50 border-blue-100" },
                        { val: project.linkedin_handle, icon: Linkedin, color: "text-sky-700 bg-sky-50 border-sky-100" }
                      ].filter(s => s.val).map((s, idx) => (
                        <div key={idx} className={`w-8 h-8 rounded-lg flex items-center justify-center border transition-all ${s.color}`}>
                          <s.icon className="w-4 h-4" />
                        </div>
                      ))}
                      {!project.ig_handle && !project.twitter_handle && !project.facebook_handle && !project.linkedin_handle && (
                        <span className="text-[10px] text-slate-400 font-bold uppercase tracking-widest">No Connections</span>
                      )}
                    </div>

                    <div className="mt-auto flex items-center justify-between group-hover:text-slate-900">
                      <span className="text-xs font-bold text-slate-400 uppercase tracking-widest group-hover:text-slate-900 transition-colors">Enter Dashboard</span>
                      <div className="w-8 h-8 rounded-full bg-slate-900 text-white flex items-center justify-center -translate-x-2 opacity-0 group-hover:translate-x-0 group-hover:opacity-100 transition-all duration-300">
                        <ArrowRight className="w-4 h-4" />
                      </div>
                    </div>
                </CardContent>
              </Card>
            </motion.div>
          ))
        )}
      </div>
    </div>
  );
}
