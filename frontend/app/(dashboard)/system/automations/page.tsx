"use client";

import { motion } from "framer-motion";
import { 
  Zap, Plus, Play, MoreHorizontal, 
  Clock, ArrowRight, Shield, Loader2
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";

export default function AutomationsPage() {
  const [automations, setAutomations] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data?.data || [];
      setProjects(allProjects);
      
      if (allProjects.length > 0) {
        const p = allProjects[allProjects.length - 1];
        setProject(p);
        const res = await dashboardApi.getAutomations(p.id);
        const tasks = res.data?.data || [];
        const mappedAutomations = tasks.map((t: any) => ({
          id: t.id,
          name: t.title,
          trigger: "Every Friday",
          action: "Generate PDF Summary",
          status: !t.completed, // completed = false means active for automations
          lastRun: "Never"
        }));
        setAutomations(mappedAutomations);
      }
    } catch (err) {
      console.error("Failed to fetch automations", err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateWorkflow = async () => {
    if (!project) return;
    const title = prompt("Enter automation description (e.g. Weekly Report):");
    if (!title) return;
    
    try {
      await dashboardApi.createAutomation({
        project_id: project.id,
        title: title
      });
      fetchData();
    } catch (err) {
      alert("Failed to create workflow.");
    }
  };

  const toggleAutomation = async (id: number) => {
    setAutomations(prev => prev.map(a => 
      a.id === id ? { ...a, status: !a.status } : a
    ));
    
    try {
      await dashboardApi.toggleAutomation(id);
    } catch (err) {
      console.error("Toggle failed", err);
      fetchData();
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  return (
    <div className="space-y-8 max-w-6xl mx-auto">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }}>
          <h1 className="text-3xl font-bold tracking-tight">Automations</h1>
          <p className="text-slate-500 mt-1">Deploy autonomous workflows that work while you sleep.</p>
        </motion.div>
        
        <Button onClick={handleCreateWorkflow} className="rounded-xl bg-brand-600 hover:bg-brand-500 text-white gap-2 h-11 px-6 shadow-lg shadow-brand-500/20 font-semibold">
          <Plus className="w-5 h-5" /> Create Workflow
        </Button>
      </div>

      <div className="grid grid-cols-1 gap-4">
        {automations.map((workflow, i) => (
          <motion.div 
            key={workflow.id}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: i * 0.05 }}
          >
            <Card className="border-border/50 hover:border-brand-200 transition-all rounded-2xl overflow-hidden bg-white dark:bg-slate-900 group">
              <CardContent className="p-0">
                <div className="flex flex-col md:flex-row items-center p-6 gap-6">
                  <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${workflow.status ? 'bg-brand-50 text-brand-600' : 'bg-slate-100 text-slate-400'}`}>
                    <Zap className={`w-6 h-6 ${workflow.status ? 'animate-pulse' : ''}`} />
                  </div>
                  
                  <div className="flex-1 space-y-1">
                    <div className="flex items-center gap-3">
                      <h3 className="font-bold text-lg text-slate-900 dark:text-white">{workflow.name}</h3>
                      {workflow.status && <Badge className="bg-emerald-500 hover:bg-emerald-600 rounded-md h-5 text-[10px]">Active</Badge>}
                    </div>
                    <div className="flex items-center gap-3 text-sm text-slate-500">
                      <span className="flex items-center gap-1.5"><Clock className="w-3.5 h-3.5" /> If {workflow.trigger}</span>
                      <ArrowRight className="w-3 h-3 text-slate-300" />
                      <span className="flex items-center gap-1.5 font-medium text-slate-700 dark:text-slate-300">Then {workflow.action}</span>
                    </div>
                  </div>

                  <div className="flex items-center gap-8 md:border-l border-slate-100 dark:border-slate-800 md:pl-8">
                    <div className="text-right hidden lg:block">
                      <p className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-0.5">Last Execution</p>
                      <p className="text-sm font-semibold">{workflow.lastRun}</p>
                    </div>
                    
                    <div className="flex items-center gap-4">
                      <Switch 
                        checked={workflow.status} 
                        onChange={() => toggleAutomation(workflow.id)} 
                      />
                      <Button variant="ghost" size="icon" className="rounded-xl h-10 w-10 text-slate-400">
                        <MoreHorizontal className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </motion.div>
        ))}
        {automations.length === 0 && (
          <div className="py-20 text-center border-2 border-dashed border-slate-100 rounded-3xl">
            <Zap className="w-12 h-12 mx-auto text-slate-200 mb-4" />
            <p className="text-slate-400 font-medium">No automations found. Start by creating your first workflow.</p>
          </div>
        )}
      </div>
    </div>
  );
}
