"use client";

import dynamic from "next/dynamic";
import { 
  Plus, Search, Sparkles, Send, BarChart3, TrendingUp,
  Activity, Zap, Target, MousePointer2, ChevronRight,
  Loader2, AlertCircle, CheckCircle2, Info, ArrowUpRight,
  Briefcase, Lightbulb, UserCheck, ShieldCheck
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const ResponsiveContainer = dynamic(
  () => import("recharts").then((m) => m.ResponsiveContainer),
  { ssr: false, loading: () => <div className="h-[360px] w-full animate-pulse bg-slate-100 rounded" /> }
);
const AreaChart = dynamic(
  () => import("recharts").then((m) => m.AreaChart),
  { ssr: false }
);
const Area = dynamic(
  () => import("recharts").then((m) => m.Area),
  { ssr: false }
);
const XAxis = dynamic(() => import("recharts").then((m) => m.XAxis), { ssr: false });
const YAxis = dynamic(() => import("recharts").then((m) => m.YAxis), { ssr: false });
const CartesianGrid = dynamic(() => import("recharts").then((m) => m.CartesianGrid), { ssr: false });
const Tooltip = dynamic(() => import("recharts").then((m) => m.Tooltip), { ssr: false });

const MotionDiv = dynamic(
  () => import("framer-motion").then((m) => m.motion.div),
  { ssr: false }
);

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { GrowthSnapshot } from "@/components/dashboard/growth-snapshot";
import { AIInsightCard } from "@/components/dashboard/ai-insight-card";
import { ActionCenterTask } from "@/components/dashboard/action-center-task";
import { toast } from "@/components/ui/toaster";

export default function DashboardPage() {
  const router = useRouter();
  const [metrics, setMetrics] = useState<any[]>([]);
  const [insights, setInsights] = useState<any[]>([]);
  const [tasks, setTasks] = useState<any[]>([]);
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [snapshot, setSnapshot] = useState<any>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [viewMode, setViewMode] = useState<'website' | 'social' | 'combined'>('combined');

  const [isCreatingTask, setIsCreatingTask] = useState(false);
  const [newTaskTitle, setNewTaskTitle] = useState("");

  const fetchData = async (targetProjectId?: number) => {
    setIsLoading(true);
    try {
      const projRes = await dashboardApi.getProjects();
      const allProjects = projRes.data.data || [];
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

        const [mRes, iRes, tRes, sRes] = await Promise.all([
          dashboardApi.getMetrics(selected.id),
          dashboardApi.getInsights(selected.id),
          dashboardApi.getTasks(selected.id),
          dashboardApi.getSnapshot(selected.id),
        ]);

        
        const mData = mRes.data?.data;
        const iData = iRes.data?.data;
        const tData = tRes.data?.data;

        
        const formattedMetrics = (Array.isArray(mData) ? mData : []).map((m: any) => ({
          name: new Date(m.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
          traffic: m.clicks,
          engagement: m.engagement,
        }));
        setMetrics(formattedMetrics);
        const formattedInsights = (Array.isArray(iData) ? iData : []).map((i: any) => ({
          ...i,
          description: i.body,
          color: i.type === 'critical' ? 'rose' : i.type === 'opportunity' ? 'emerald' : i.type === 'warning' ? 'amber' : 'brand',
          actionText: i.type === 'critical' ? 'Fix Issue' : i.type === 'opportunity' ? 'Capitalize' : 'Learn More'
        }));
        setInsights(formattedInsights);
        const formattedTasks = (Array.isArray(tData) ? tData : []).map((t: any) => ({
          ...t,
          isDone: t.completed,
          priority: t.source === 'ai' ? 'High' : 'Medium'
        }));
        setTasks(formattedTasks);
        setSnapshot(sRes.data?.data || null);




        if (selected.goal === 'seo') setViewMode('website');
        else if (selected.goal === 'social') setViewMode('social');
      }
    } catch (err) {
      console.error("Failed to fetch dashboard data", err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleProjectChange = (newProj: any) => {
    fetchData(newProj.id);
  };

  const handleToggleTask = async (id: number) => {
    const updatedTasks = tasks.map(t => t.id === id ? { ...t, isDone: !t.isDone } : t);
    setTasks(updatedTasks);
    try {
      await dashboardApi.toggleTask(id, project?.id);
    } catch (err) {
      setTasks(tasks); // revert on failure
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
      setTasks([...tasks, { ...createdTask, isDone: false, priority: 'Medium' }]);
      setNewTaskTitle("");
    } catch (err) {
      toast("Failed to create task");
    } finally {
      setIsCreatingTask(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  const doneCount = tasks.filter(t => t.isDone).length;
  const progressPercent = tasks.length > 0 ? (doneCount / tasks.length) * 100 : 0;
  const isSimulated = snapshot?.is_simulated ?? false;

  return (
    <div className="space-y-16 max-w-7xl mx-auto pb-40 pt-4">
      
      {/* 🏙️ 1. DASHBOARD HEADER */}
      <DashboardHeader 
        project={project} 
        projects={projects}
        onProjectChange={handleProjectChange}
        onAddSource={() => router.push('/onboarding')} 
      />

      {isSimulated && (
        <MotionDiv 
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          className="bg-amber-50 border border-amber-100 rounded-2xl p-4 flex items-center justify-between gap-4"
        >
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-amber-100 flex items-center justify-center text-amber-600 shrink-0">
              <AlertCircle className="w-4 h-4" />
            </div>
            <p className="text-sm text-amber-800 font-medium">
              <span className="font-bold">Simulated Data Active:</span> Some social metrics are currently estimated. Connect your Meta API for live accuracy.
            </p>
          </div>
          <Button 
            variant="outline" 
            size="sm"
            className="rounded-lg bg-white border-amber-200 text-amber-700 hover:bg-amber-50"
            onClick={() => router.push('/integrations')}
          >
            Connect API
          </Button>
        </MotionDiv>
      )}

      {/* 🔥 "Today’s Focus" Banner - Refined */}
      <MotionDiv 
        initial={{ opacity: 0, y: -4 }} 
        animate={{ opacity: 1, y: 0 }}
        className="bg-slate-900 rounded-2xl p-8 text-white shadow-xl shadow-slate-200 relative overflow-hidden flex flex-col md:flex-row items-center justify-between gap-8"
      >
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,_var(--tw-gradient-stops))] from-white/10 via-transparent to-transparent opacity-40" />
        <div className="relative z-10 flex items-center gap-6">
          <div className="w-12 h-12 rounded-xl bg-white/10 flex items-center justify-center border border-white/10">
            <Sparkles className="w-5 h-5 text-brand-100" />
          </div>
          <div>
            <h3 className="text-[10px] font-bold tracking-[0.2em] uppercase text-slate-400 mb-1">Critical Objective</h3>
            <p className="text-xl font-semibold tracking-tight">
              {tasks.length > 0 ? tasks[0].title : insights.length > 0 ? insights[0].title : "Analyze your latest metrics to find growth opportunities."}
            </p>
          </div>
        </div>
        <div className="relative z-10 flex items-center gap-8 md:border-l border-white/10 md:pl-8">
          <div className="text-right">
            <p className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-1">Growth Score</p>
            <p className="text-3xl font-semibold tracking-tight tabular-nums">{project?.health_score || 0}<span className="text-sm opacity-40">/100</span></p>
          </div>
          <Button 
            className="rounded-xl bg-white text-slate-900 hover:bg-slate-50 font-semibold px-6 h-12 transition-all"
            onClick={() => router.push('/action-center')}
          >
            Execute Strategy
          </Button>
        </div>
      </MotionDiv>

      {/* 📊 2. PERFORMANCE SNAPSHOT */}
      <GrowthSnapshot 
        viewMode={viewMode} 
        setViewMode={setViewMode} 
        project={project} 
        snapshotData={snapshot}
      />

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-12">
        
        {/* LEFT COLUMN: Insights & Trends */}
        <div className="lg:col-span-8 space-y-16">
          
          {/* 📈 3. VISUAL TRENDS */}
          <div className="space-y-8">
            <div className="space-y-1">
              <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Analytics Velocity</h2>
              <p className="text-2xl font-semibold text-slate-900 tracking-tight">Growth Trends</p>
            </div>
            <Card className="border-slate-200/60 shadow-none rounded-2xl overflow-hidden bg-white">
              <CardContent className="p-8">
                 <div className="h-[360px] w-full">
                    <ResponsiveContainer width="100%" height="100%">
                       <AreaChart data={metrics}>
                          <defs>
                             <linearGradient id="colorPrimary" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#0f172a" stopOpacity={0.05}/>
                                <stop offset="95%" stopColor="#0f172a" stopOpacity={0}/>
                             </linearGradient>
                          </defs>
                          <CartesianGrid strokeDasharray="3 3" vertical={false} strokeOpacity={0.05} />
                          <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: '#94a3b8', fontWeight: 500 }} dy={15} />
                          <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: '#94a3b8', fontWeight: 500 }} />
                          <Tooltip 
                             contentStyle={{ borderRadius: '12px', border: '1px solid #f1f5f9', backgroundColor: '#ffffff', boxShadow: '0 10px 15px -3px rgb(0 0 0 / 0.1)' }}
                             itemStyle={{ fontSize: '12px', fontWeight: 600 }}
                          />
                          <Area type="monotone" dataKey="traffic" stroke="#0f172a" strokeWidth={2.5} fillOpacity={1} fill="url(#colorPrimary)" />
                          <Area type="monotone" dataKey="engagement" stroke="#94a3b8" strokeWidth={1.5} strokeDasharray="4 4" fill="none" />
                       </AreaChart>
                    </ResponsiveContainer>
                 </div>
              </CardContent>
            </Card>
          </div>

          {/* 🧠 4. AI INSIGHTS */}
          <div className="space-y-8">
            <div className="space-y-1">
              <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Intelligent Reports</h2>
              <p className="text-2xl font-semibold text-slate-900 tracking-tight">AI Insights</p>
            </div>
            <div className="grid gap-6">
              {insights.map((insight) => (
                <AIInsightCard key={insight.id} insight={insight} />
              ))}
              {insights.length === 0 && (
                <div className="rounded-2xl border-2 border-dashed border-slate-100 flex flex-col items-center justify-center p-16 text-center">
                  <Lightbulb className="w-10 h-10 mb-4 text-slate-200" />
                  <p className="text-slate-400 font-medium">Connect accounts to generate tailored growth insights</p>
                </div>
              )}
            </div>
          </div>

        </div>

        {/* RIGHT COLUMN: Execution & Context */}
        <div className="lg:col-span-4 space-y-16">
          
          {/* ✅ 5. ACTION CENTER */}
          <div className="space-y-8">
            <div className="space-y-1">
              <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Task Execution</h2>
              <p className="text-2xl font-semibold text-slate-900 tracking-tight">Action Center</p>
            </div>
            <Card className="rounded-2xl border-slate-200/60 shadow-xl shadow-slate-200/30 flex flex-col h-fit bg-white overflow-hidden">
              <CardHeader className="pb-6 pt-8 px-8 border-b border-slate-50">
                <div className="flex items-center justify-between mb-6">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="bg-emerald-50 text-emerald-600 border-0 font-bold px-2 py-0.5">
                      {doneCount}/{tasks.length} Complete
                    </Badge>
                  </div>
                  <span className="text-[11px] font-bold text-slate-400 uppercase tracking-widest">Velocity: Increasing</span>
                </div>
                <div className="space-y-3">
                  <Progress value={progressPercent} className="h-1.5 bg-slate-100" indicatorClassName="bg-emerald-500" />
                  <p className="text-[10px] text-slate-400 font-semibold uppercase tracking-wider">Execute tasks to move your growth index</p>
                </div>
              </CardHeader>
              <CardContent className="px-8 py-4 space-y-1">
                {tasks.map((task) => (
                  <ActionCenterTask key={task.id} task={task} onToggle={handleToggleTask} />
                ))}
                {tasks.length === 0 && (
                  <div className="py-12 text-center text-slate-300">
                    <p className="text-sm font-medium">No pending actions</p>
                  </div>
                )}
                
                <div className="mt-6 flex gap-2">
                  <Input 
                    placeholder="New custom task..." 
                    value={newTaskTitle}
                    onChange={(e: any) => setNewTaskTitle(e.target.value)}
                    onKeyDown={(e: any) => e.key === 'Enter' && handleCreateTask()}
                    className="h-10 rounded-xl bg-slate-50 border-slate-200"
                  />
                  <Button 
                    onClick={handleCreateTask}
                    disabled={isCreatingTask || !newTaskTitle.trim()}
                    className="h-10 rounded-xl bg-slate-900 text-white px-4 font-semibold shrink-0"
                  >
                    {isCreatingTask ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>


        </div>
      </div>

      {/* ⚡ 9. QUICK ACTION BAR (FLOATING) - Refined */}
      <div className="fixed bottom-10 left-1/2 -translate-x-1/2 z-50 px-6 w-full max-w-2xl">
        <div className="bg-white/90 backdrop-blur-xl border border-slate-200/60 rounded-2xl p-2.5 flex items-center justify-between shadow-[0_20px_50px_-12px_rgba(0,0,0,0.1)]">
          <div className="flex items-center gap-1">
             <Button 
               variant="ghost" 
               className="rounded-xl h-11 text-slate-600 font-semibold gap-2 px-4 hover:bg-slate-100 hover:text-slate-900"
               onClick={() => router.push("/seo/site-explorer")}
             >
               <Search className="w-4 h-4" />
               <span className="hidden sm:inline">Run Audit</span>
             </Button>
             <Button 
               variant="ghost" 
               className="rounded-xl h-11 text-slate-600 font-semibold gap-2 px-4 hover:bg-slate-100 hover:text-slate-900"
               onClick={() => router.push("/projects/settings")}
             >
               <UserCheck className="w-4 h-4" />
               <span className="hidden sm:inline">Compare</span>
             </Button>
             <Button 
               variant="ghost" 
               className="rounded-xl h-11 text-slate-600 font-semibold gap-2 px-4 hover:bg-slate-100 hover:text-slate-900"
               onClick={() => router.push("/content")}
             >
               <Briefcase className="w-4 h-4" />
               <span className="hidden sm:inline">Content</span>
             </Button>
          </div>
          <div className="h-6 w-px bg-slate-200 mx-3" />
          <Button 
            className="rounded-xl h-11 bg-slate-900 hover:bg-slate-800 text-white font-semibold px-6 transition-all shadow-lg shadow-slate-900/10"
            onClick={() => router.push("/projects/create")}
          >
            Analyze New URL
          </Button>
        </div>
      </div>

    </div>
  );
}
