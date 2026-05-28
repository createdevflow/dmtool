"use client";

import { motion } from "framer-motion";
import { 
  CheckSquare, CheckCircle2, Loader2, Plus, ArrowUpRight, 
  Zap, Target, TrendingUp
} from "lucide-react";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { ActionCenterTask } from "@/components/dashboard/action-center-task";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Card, CardContent } from "@/components/ui/card";

export default function ActionCenterPage() {
  const [tasks, setTasks] = useState<any[]>([]);
  const [insights, setInsights] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [project, setProject] = useState<any>(null);
  const [projects, setProjects] = useState<any[]>([]);
  const [filter, setFilter] = useState<string>("all");
  const [isCreatingTask, setIsCreatingTask] = useState(false);
  const [newTaskTitle, setNewTaskTitle] = useState("");

  const fetchData = async (targetProjectId?: number) => {
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data?.data ?? pRes.data ?? [];
      setProjects(Array.isArray(allProjects) ? allProjects : []);

      if (allProjects.length > 0) {
        const p = targetProjectId
          ? allProjects.find((x: any) => x.id === targetProjectId) || allProjects[allProjects.length - 1]
          : allProjects[allProjects.length - 1];
        setProject(p);

        const [tRes, iRes] = await Promise.all([
          dashboardApi.getTasks(p.id),
          dashboardApi.getInsights(p.id),
        ]);

        const rawTasks = tRes.data?.data ?? tRes.data ?? [];
        setTasks(Array.isArray(rawTasks) ? rawTasks.map((t: any) => ({
          ...t,
          isDone: t.completed,
          priority: t.source === 'ai' ? 'High' : 'Medium',
        })) : []);

        const rawInsights = iRes.data?.data ?? [];
        setInsights(Array.isArray(rawInsights) ? rawInsights : []);
      }
    } catch (err) {
      console.error("Failed to fetch tasks", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleProjectChange = (newProj: any) => {
    setLoading(true);
    fetchData(newProj.id);
  };

  const handleToggleTask = async (id: number) => {
    const prev = [...tasks];
    setTasks(tasks.map(t => t.id === id ? { ...t, isDone: !t.isDone } : t));
    try {
      await dashboardApi.toggleTask(id, project?.id);
    } catch (err) {
      setTasks(prev); // revert on failure
    }
  };

  const handleCreateTask = async () => {
    if (!newTaskTitle.trim() || !project) return;
    setIsCreatingTask(true);
    try {
      const res = await dashboardApi.createTask({
        project_id: project.id,
        title: newTaskTitle.trim()
      });
      const createdTask = res.data.data;
      setTasks(prev => [...prev, { ...createdTask, isDone: false, priority: 'Medium' }]);
      setNewTaskTitle("");
    } catch (err) {
      alert("Failed to create task.");
    } finally {
      setIsCreatingTask(false);
    }
  };

  const doneCount = tasks.filter(t => t.isDone).length;
  const progressPercent = tasks.length > 0 ? (doneCount / tasks.length) * 100 : 0;

  // Compute velocity: ratio of AI tasks done vs total
  const aiTasksDone = tasks.filter(t => t.isDone && t.priority === 'High').length;
  const aiTotal = tasks.filter(t => t.priority === 'High').length;
  const velocityPct = aiTotal > 0 ? Math.round((aiTasksDone / aiTotal) * 100) : progressPercent;
  const velocityLabel = velocityPct >= 80 ? "High" : velocityPct >= 50 ? "Moderate" : "Building";

  const filteredTasks = tasks.filter(t => {
    if (filter === "all") return true;
    if (filter === "done") return t.isDone;
    if (filter === "pending") return !t.isDone;
    return t.priority?.toLowerCase() === filter.toLowerCase();
  });

  // Derive strategy tips from real AI insights (type=opportunity)
  const opportunityInsights = insights
    .filter(i => i.type === "opportunity" || i.type === "warning")
    .slice(0, 2);

  const fallbackTips = [
    { title: "Batch Content Creation", desc: "Create 5 content pieces this week to stay ahead of your growth goals.", icon: Zap },
    { title: "SEO Low-Hanging Fruit", desc: "Optimize meta titles for pages that rank on page 2 to capture quick wins.", icon: Target },
  ];

  const strategyTips = opportunityInsights.length > 0
    ? opportunityInsights.map((i: any) => ({
        title: i.title,
        desc: i.body || i.description || "Review this insight to unlock growth.",
        icon: i.type === "warning" ? Target : Zap,
      }))
    : fallbackTips;

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
        onProjectChange={handleProjectChange}
        onAddSource={() => {}} 
      />

      <div className="flex flex-col md:flex-row md:items-end justify-between gap-8">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Execution Loop</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Action Center</p>
        </div>
        
        <div className="flex flex-col items-end gap-3">
          <div className="flex items-center gap-2">
            {["all", "pending", "done", "high"].map((f) => (
              <Button
                key={f}
                variant={filter === f ? "default" : "ghost"}
                onClick={() => setFilter(f)}
                className={`rounded-xl h-10 px-5 text-xs font-semibold capitalize transition-all ${
                  filter === f 
                    ? "bg-slate-900 text-white shadow-lg shadow-slate-900/10" 
                    : "text-slate-500 hover:bg-slate-50"
                }`}
              >
                {f}
              </Button>
            ))}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-12">
        
        {/* Main Task List */}
        <div className="lg:col-span-8 space-y-8">
          <Card className="border-slate-200/60 shadow-xl shadow-slate-200/20 bg-white rounded-3xl overflow-hidden">
            <div className="p-8 border-b border-slate-50 bg-slate-50/30 flex items-center justify-between">
               <h3 className="text-sm font-bold text-slate-900 flex items-center gap-2">
                 <CheckSquare className="w-4 h-4 text-brand-600" />
                 Active Tasks
               </h3>
               <span className="text-[11px] font-bold text-slate-400 uppercase tracking-widest">{filteredTasks.length} items</span>
            </div>
            <CardContent className="p-8 space-y-2">
              {filteredTasks.length > 0 ? (
                filteredTasks.map((task) => (
                  <ActionCenterTask key={task.id} task={task} onToggle={handleToggleTask} />
                ))
              ) : (
                <div className="py-20 text-center text-slate-400">
                  <CheckCircle2 className="w-12 h-12 mx-auto mb-4 opacity-10" />
                  <p className="text-sm font-medium">No tasks match your current filter</p>
                </div>
              )}
              
              <div className="mt-8 flex gap-2">
                <input 
                  type="text"
                  placeholder="New execution task..." 
                  value={newTaskTitle}
                  onChange={(e) => setNewTaskTitle(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleCreateTask()}
                  className="flex h-14 w-full rounded-2xl border border-slate-200 bg-white px-4 py-2 text-sm placeholder:text-slate-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                />
                <Button 
                  onClick={handleCreateTask}
                  disabled={isCreatingTask || !newTaskTitle.trim()}
                  className="h-14 rounded-2xl bg-slate-900 text-white px-6 font-semibold shrink-0"
                >
                  {isCreatingTask ? <Loader2 className="w-5 h-5 animate-spin" /> : <Plus className="w-5 h-5" />}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Action Sidebar */}
        <div className="lg:col-span-4 space-y-8">
          
          {/* Progress Card — Real Data */}
          <Card className="border-slate-200/60 shadow-none bg-slate-900 text-white rounded-3xl p-8 relative overflow-hidden">
             <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,_var(--tw-gradient-stops))] from-white/10 via-transparent to-transparent opacity-40" />
             <div className="relative z-10 space-y-6">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-bold uppercase tracking-widest text-slate-400">Velocity</h3>
                  <div className="flex items-center gap-1 text-emerald-400 font-bold text-xs">
                    <TrendingUp className="w-3 h-3" /> {velocityLabel}
                  </div>
                </div>
                <div className="space-y-4">
                  <div className="flex items-end justify-between">
                    <p className="text-4xl font-semibold tracking-tight">{progressPercent.toFixed(0)}<span className="text-lg opacity-40">%</span></p>
                    <p className="text-[11px] font-bold text-slate-400 uppercase">{doneCount}/{tasks.length} Completed</p>
                  </div>
                  <Progress value={progressPercent} className="h-1.5 bg-white/10" indicatorClassName="bg-emerald-400" />
                </div>
                <p className="text-xs text-slate-400 font-medium leading-relaxed opacity-80">
                  {doneCount === 0
                    ? "Start completing tasks to build execution momentum."
                    : doneCount === tasks.length
                    ? "All tasks complete! Great work — add new tasks to keep growing."
                    : `${tasks.length - doneCount} tasks remaining. Keep going to hit your goals.`}
                </p>
             </div>
          </Card>

          {/* Strategy Tips — Real from Insights */}
          <div className="space-y-6">
            <h4 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 px-2">
              {opportunityInsights.length > 0 ? "AI Growth Tips" : "Growth Strategy Tips"}
            </h4>
            <div className="space-y-4">
              {strategyTips.map((tip, i) => (
                <div key={i} className="p-5 rounded-2xl border border-slate-100 bg-white hover:border-slate-200 transition-all flex gap-4">
                  <div className="w-8 h-8 rounded-lg bg-brand-50 flex items-center justify-center text-brand-600 shrink-0">
                    <tip.icon className="w-4 h-4" />
                  </div>
                  <div>
                    <p className="text-[13px] font-semibold text-slate-900">{tip.title}</p>
                    <p className="text-[11px] text-slate-500 font-medium mt-1 leading-relaxed line-clamp-3">{tip.desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

        </div>
      </div>

    </div>
  );
}
