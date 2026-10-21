"use client";

import {
  CheckSquare, CheckCircle2, Loader2, Plus, Lightbulb,
} from "lucide-react";
import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { dashboardApi } from "@/lib/api-client";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { ActionCenterTask } from "@/components/dashboard/action-center-task";
import { StoredWorkList } from "@/components/dashboard/stored-work-list";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Card, CardContent } from "@/components/ui/card";
import { formatTaskRow, parseTasksPayload, type ActionTask, type ActionWorkItem } from "@/lib/action-work";

export default function ActionCenterPage() {
  const router = useRouter();
  const [tasks, setTasks] = useState<ActionTask[]>([]);
  const [work, setWork] = useState<ActionWorkItem[]>([]);
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
        let savedProjectId = 0;
        try { savedProjectId = parseInt(localStorage.getItem("dmtool_active_project_id") || "0"); } catch {}
        const p = targetProjectId
          ? allProjects.find((x: any) => x.id === targetProjectId) || allProjects[allProjects.length - 1]
          : allProjects.find((x: any) => x.id === savedProjectId) || allProjects[allProjects.length - 1];
        setProject(p);
        if (p?.id) localStorage.setItem("dmtool_active_project_id", p.id.toString());

        const [tRes, iRes] = await Promise.all([
          dashboardApi.getTasks(p.id),
          dashboardApi.getInsights(p.id),
        ]);

        const parsed = parseTasksPayload(tRes.data);
        setTasks(parsed.tasks.map(formatTaskRow));
        setWork(parsed.work);

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
    setTasks(tasks.map((t) => t.id === id ? { ...t, isDone: !t.isDone, completed: !t.isDone } : t));
    try {
      await dashboardApi.toggleTask(id, project?.id);
    } catch {
      setTasks(prev);
    }
  };

  const handleCreateTask = async () => {
    if (!newTaskTitle.trim() || !project) return;
    setIsCreatingTask(true);
    try {
      const res = await dashboardApi.createTask({
        project_id: project.id,
        title: newTaskTitle.trim(),
      });
      const createdTask = res.data.data;
      setTasks((prev) => [...prev, formatTaskRow(createdTask)]);
      setNewTaskTitle("");
    } catch {
      alert("Failed to create task.");
    } finally {
      setIsCreatingTask(false);
    }
  };

  const doneCount = tasks.filter((t) => t.isDone).length;
  const progressPercent = tasks.length > 0 ? (doneCount / tasks.length) * 100 : null;
  const openWork = work.filter((w) => !w.done);

  const filteredTasks = tasks.filter((t) => {
    if (filter === "all") return true;
    if (filter === "done") return t.isDone;
    if (filter === "pending") return !t.isDone;
    return (t.priority || "").toLowerCase() === filter.toLowerCase();
  });

  const insightTips = insights.filter((i) => {
    const type = String(i.type || "").toLowerCase();
    return type === "opportunity" || type === "warning" || type === "critical";
  }).slice(0, 3);

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
        onAddSource={() => router.push("/integrations")}
      />

      <div className="flex flex-col md:flex-row md:items-end justify-between gap-8">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Execution loop</h2>
          <p className="text-3xl font-semibold text-slate-900 tracking-tight">Action Center</p>
        </div>

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

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-12">
        <div className="lg:col-span-8 space-y-8">
          <Card className="border-slate-200/60 shadow-none bg-white rounded-3xl overflow-hidden">
            <div className="p-8 border-b border-slate-50 bg-slate-50/30 flex items-center justify-between">
              <h3 className="text-sm font-bold text-slate-900">Today&apos;s work</h3>
              <span className="text-[11px] font-bold text-slate-400 uppercase tracking-widest">
                {openWork.length} open
              </span>
            </div>
            <CardContent className="p-6">
              <p className="text-xs text-slate-400 mb-4 px-2">
                From stored state. Links do not auto-fetch DataForSEO. Checking off happens on your task rows below.
              </p>
              <StoredWorkList work={work} emptyLabel="Select a project to see stored work." />
            </CardContent>
          </Card>

          <Card className="border-slate-200/60 shadow-xl shadow-slate-200/20 bg-white rounded-3xl overflow-hidden">
            <div className="p-8 border-b border-slate-50 bg-slate-50/30 flex items-center justify-between">
              <h3 className="text-sm font-bold text-slate-900 flex items-center gap-2">
                <CheckSquare className="w-4 h-4 text-brand-600" />
                Your tasks
              </h3>
              <span className="text-[11px] font-bold text-slate-400 uppercase tracking-widest">{filteredTasks.length} items</span>
            </div>
            <CardContent className="p-8 space-y-2">
              {filteredTasks.length > 0 ? (
                filteredTasks.map((task) => (
                  <ActionCenterTask key={task.id} task={task} onToggle={handleToggleTask} />
                ))
              ) : (
                <div className="py-16 text-center text-slate-400">
                  <CheckCircle2 className="w-12 h-12 mx-auto mb-4 opacity-10" />
                  <p className="text-sm font-medium">
                    {tasks.length === 0 ? "No task rows yet. Add one below." : "No tasks match this filter."}
                  </p>
                </div>
              )}

              <div className="mt-8 flex gap-2">
                <input
                  type="text"
                  placeholder="New task..."
                  value={newTaskTitle}
                  onChange={(e) => setNewTaskTitle(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleCreateTask()}
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

        <div className="lg:col-span-4 space-y-8">
          <Card className="border-slate-200/60 shadow-none bg-slate-900 text-white rounded-3xl p-8 relative overflow-hidden">
            <div className="relative z-10 space-y-6">
              <h3 className="text-sm font-bold uppercase tracking-widest text-slate-400">Progress</h3>
              <div className="space-y-4">
                <div className="flex items-end justify-between">
                  <p className="text-4xl font-semibold tracking-tight tabular-nums">
                    {progressPercent === null ? "—" : `${progressPercent.toFixed(0)}`}
                    {progressPercent !== null && <span className="text-lg opacity-40">%</span>}
                  </p>
                  <p className="text-[11px] font-bold text-slate-400 uppercase">
                    {tasks.length === 0 ? "No task rows" : `${doneCount}/${tasks.length} tasks`}
                  </p>
                </div>
                <Progress value={progressPercent ?? 0} className="h-1.5 bg-white/10" indicatorClassName="bg-emerald-400" />
              </div>
              <p className="text-xs text-slate-400 font-medium leading-relaxed">
                {openWork.length === 0
                  ? "No open work from stored state."
                  : `${openWork.length} open from stored state (GSC, audit, lookups, issues).`}
              </p>
            </div>
          </Card>

          <div className="space-y-6">
            <h4 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 px-2">Insights</h4>
            {insightTips.length > 0 ? (
              <div className="space-y-4">
                {insightTips.map((tip) => (
                  <div key={tip.id ?? tip.title} className="p-5 rounded-2xl border border-slate-100 bg-white flex gap-4">
                    <div className="w-8 h-8 rounded-lg bg-slate-50 flex items-center justify-center text-slate-500 shrink-0">
                      <Lightbulb className="w-4 h-4" />
                    </div>
                    <div>
                      <p className="text-[13px] font-semibold text-slate-900">{tip.title}</p>
                      <p className="text-[11px] text-slate-500 font-medium mt-1 leading-relaxed line-clamp-3">{tip.body}</p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="p-5 rounded-2xl border border-dashed border-slate-100 text-sm text-slate-400">
                No insights yet. Connect GSC and sync to generate them from real traffic.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
