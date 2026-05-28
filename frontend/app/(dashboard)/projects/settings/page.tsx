"use client";

import { motion } from "framer-motion";
import { Settings, Shield, Bell, Database, Save, Trash2, Loader2, CheckCircle2, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useEffect, useState } from "react";
import { dashboardApi } from "@/lib/api-client";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "@/components/ui/toaster";
import { Suspense } from "react";

function ProjectSettingsContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [project, setProject] = useState<any>(null);
  const [formData, setFormData] = useState<any>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // Notification preferences (local UI state — toggled by user)
  const [notifPrefs, setNotifPrefs] = useState({
    seo_alerts: true,
    social_alerts: true,
    weekly_report: true,
    task_reminders: false,
    competitor_updates: false,
  });

  useEffect(() => {
    const loadProject = async () => {
      try {
        const res = await dashboardApi.getProjects();
        const data = res.data?.data || res.data;
        const allProjects = Array.isArray(data) ? data : [];

        // Prefer project_id from URL query param, fall back to last project
        const queryProjectId = searchParams?.get("project_id");
        let p: any = null;
        if (queryProjectId) {
          p = allProjects.find((x: any) => String(x.id) === queryProjectId);
        }
        if (!p && allProjects.length > 0) {
          p = allProjects[allProjects.length - 1];
        }

        if (p) {
          setProject(p);
          setFormData({
            name: p.name || "",
            url: p.url || "",
            goal: p.goal || "both",
            ig_handle: p.ig_handle || "",
            twitter_handle: p.twitter_handle || "",
            linkedin_handle: p.linkedin_handle || "",
            fb_handle: p.fb_handle || "",
          });
        }
      } catch (err) {
        console.error("Settings load failed", err);
      } finally {
        setLoading(false);
      }
    };
    loadProject();
  }, [searchParams]);

  const handleSave = async () => {
    if (!project) return;
    setSaving(true);
    setSaveSuccess(false);
    try {
      await dashboardApi.updateProject(project.id, formData);
      setProject({ ...project, ...formData });
      setSaveSuccess(true);
      toast("Settings saved successfully.", "success");
      setTimeout(() => setSaveSuccess(false), 3000);
    } catch (err) {
      toast("Failed to save settings. Please try again.", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!project || !confirm("Are you sure you want to delete this project? This cannot be undone.")) return;
    try {
      await dashboardApi.deleteProject(project.id);
      toast("Project deleted successfully.", "success");
      router.push("/projects");
    } catch (err) {
      toast("Failed to delete project.", "error");
    }
  };

  const handleExportJSON = () => {
    if (!project) return;
    const exportData = {
      project: {
        id: project.id,
        name: project.name,
        url: project.url,
        goal: project.goal,
        status: project.status,
        health_score: project.health_score,
        ig_handle: project.ig_handle,
        twitter_handle: project.twitter_handle,
        linkedin_handle: project.linkedin_handle,
        fb_handle: project.fb_handle,
        created_at: project.created_at,
      },
      exported_at: new Date().toISOString(),
      tool: "DMTool v2.0",
    };
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${project.name || "project"}-dmtool-export.json`;
    a.click();
    URL.revokeObjectURL(url);
    toast("Project data exported successfully.", "success");
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-8">
      <motion.div initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }}>
        <h1 className="text-3xl font-bold tracking-tight">Project Settings</h1>
        <p className="text-slate-500 mt-2">Manage configurations for <span className="font-semibold text-slate-700">{project?.name || "your project"}</span>.</p>
      </motion.div>

      {!project ? (
        <Card className="p-12 text-center space-y-4 border-dashed">
          <div className="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mx-auto">
            <Settings className="w-8 h-8 text-slate-400" />
          </div>
          <h2 className="text-xl font-bold">No Project Selected</h2>
          <p className="text-slate-500 max-w-sm mx-auto">
            We couldn't find any active projects. Please create a project first to manage its settings.
          </p>
          <Button onClick={() => router.push("/projects/create")} className="bg-brand-600">
            Create Project
          </Button>
        </Card>
      ) : (
        <Tabs defaultValue="general" className="space-y-6">
          <TabsList className="bg-slate-100 dark:bg-slate-900 p-1 rounded-xl h-auto flex-wrap sm:flex-nowrap">
            <TabsTrigger value="general" className="rounded-lg py-2 px-4 data-[state=active]:bg-white dark:data-[state=active]:bg-slate-800 data-[state=active]:shadow-sm">
              <Settings className="w-4 h-4 mr-2" /> General
            </TabsTrigger>
            <TabsTrigger value="social" className="rounded-lg py-2 px-4 data-[state=active]:bg-white dark:data-[state=active]:bg-slate-800 data-[state=active]:shadow-sm">
              <Settings className="w-4 h-4 mr-2" /> Social Profiles
            </TabsTrigger>
            <TabsTrigger value="notifications" className="rounded-lg py-2 px-4 data-[state=active]:bg-white dark:data-[state=active]:bg-slate-800 data-[state=active]:shadow-sm">
              <Bell className="w-4 h-4 mr-2" /> Notifications
            </TabsTrigger>
            <TabsTrigger value="security" className="rounded-lg py-2 px-4 data-[state=active]:bg-white dark:data-[state=active]:bg-slate-800 data-[state=active]:shadow-sm">
              <Shield className="w-4 h-4 mr-2" /> Security
            </TabsTrigger>
            <TabsTrigger value="data" className="rounded-lg py-2 px-4 data-[state=active]:bg-white dark:data-[state=active]:bg-slate-800 data-[state=active]:shadow-sm">
              <Database className="w-4 h-4 mr-2" /> Data Export
            </TabsTrigger>
          </TabsList>

          {/* ── GENERAL ── */}
          <TabsContent value="general" className="space-y-6">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>Core Configuration</CardTitle>
                <CardDescription>Update your project identification and tracking parameters.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <Label>Internal Project Name</Label>
                    <Input 
                      value={formData.name || ""} 
                      onChange={e => setFormData({...formData, name: e.target.value})}
                      className="rounded-xl border-slate-200 dark:border-slate-800" 
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Primary Domain</Label>
                    <Input 
                      value={formData.url || ""} 
                      onChange={e => setFormData({...formData, url: e.target.value})}
                      className="rounded-xl border-slate-200 dark:border-slate-800" 
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>Tracking ID</Label>
                  <Input defaultValue={`DM-${project?.id || '0000'}-X90`} readOnly className="rounded-xl bg-slate-50 dark:bg-slate-900 font-mono text-sm" />
                </div>
                <div className="space-y-2">
                  <Label>Goal</Label>
                  <div className="flex gap-3">
                    {["seo", "social", "both"].map((g) => (
                      <button
                        key={g}
                        type="button"
                        onClick={() => setFormData({...formData, goal: g})}
                        className={`px-5 py-2 rounded-xl text-sm font-semibold border transition-all capitalize ${
                          formData.goal === g
                            ? "bg-slate-900 text-white border-slate-900"
                            : "bg-white text-slate-600 border-slate-200 hover:border-slate-400"
                        }`}
                      >
                        {g === "both" ? "Unified (SEO + Social)" : g === "seo" ? "SEO Only" : "Social Only"}
                      </button>
                    ))}
                  </div>
                </div>
                <div className="flex items-center justify-between p-4 rounded-xl bg-brand-50/50 dark:bg-brand-500/5 border border-brand-100 dark:border-brand-900/50">
                  <div className="space-y-0.5">
                    <p className="text-sm font-semibold text-brand-900 dark:text-brand-300">Live Monitoring</p>
                    <p className="text-xs text-brand-700/70 dark:text-brand-400/70">Keep background crawler active for real-time updates.</p>
                  </div>
                  <Switch defaultChecked />
                </div>
                <div className="flex justify-end pt-4">
                  <Button 
                    onClick={handleSave}
                    className={`rounded-xl px-8 transition-all ${saveSuccess ? 'bg-emerald-600 hover:bg-emerald-600' : 'bg-brand-600 hover:bg-brand-500'} text-white`}
                    disabled={saving}
                  >
                    {saving ? (
                      <><Loader2 className="w-4 h-4 mr-2 animate-spin" /> Saving...</>
                    ) : saveSuccess ? (
                      <><CheckCircle2 className="w-4 h-4 mr-2" /> Saved!</>
                    ) : (
                      <><Save className="w-4 h-4 mr-2" /> Save Changes</>
                    )}
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card className="border-red-200 dark:border-red-900/30 bg-red-50/20 dark:bg-red-950/10">
              <CardHeader>
                <CardTitle className="text-red-600 dark:text-red-400">Danger Zone</CardTitle>
                <CardDescription>Permanently delete this project and all associated data.</CardDescription>
              </CardHeader>
              <CardContent>
                <Button variant="destructive" className="rounded-xl bg-red-600 hover:bg-red-700" onClick={handleDelete}>
                  <Trash2 className="w-4 h-4 mr-2" /> Delete Project
                </Button>
              </CardContent>
            </Card>
          </TabsContent>

          {/* ── SOCIAL PROFILES ── */}
          <TabsContent value="social" className="space-y-6">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>Social Media Profiles</CardTitle>
                <CardDescription>Configure the public handles used for automated analytics and social monitoring.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <Label>Instagram Handle</Label>
                    <div className="relative">
                      <span className="absolute left-3 top-2.5 text-slate-400 text-sm">@</span>
                      <Input value={formData.ig_handle || ""} onChange={e => setFormData({...formData, ig_handle: e.target.value})} placeholder="username" className="pl-7 rounded-xl border-slate-200 dark:border-slate-800" />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>Twitter/X Handle</Label>
                    <div className="relative">
                      <span className="absolute left-3 top-2.5 text-slate-400 text-sm">@</span>
                      <Input value={formData.twitter_handle || ""} onChange={e => setFormData({...formData, twitter_handle: e.target.value})} placeholder="username" className="pl-7 rounded-xl border-slate-200 dark:border-slate-800" />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>Facebook Page URL</Label>
                    <Input value={formData.fb_handle || ""} onChange={e => setFormData({...formData, fb_handle: e.target.value})} placeholder="fb.com/page" className="rounded-xl border-slate-200 dark:border-slate-800" />
                  </div>
                  <div className="space-y-2">
                    <Label>LinkedIn Page URL</Label>
                    <Input value={formData.linkedin_handle || ""} onChange={e => setFormData({...formData, linkedin_handle: e.target.value})} placeholder="linkedin.com/company/..." className="rounded-xl border-slate-200 dark:border-slate-800" />
                  </div>
                </div>
                <div className="flex justify-end pt-4">
                  <Button onClick={handleSave} disabled={saving} className="bg-brand-600 hover:bg-brand-500 text-white rounded-xl px-8">
                    {saving ? <><Loader2 className="w-4 h-4 mr-2 animate-spin" /> Saving...</> : <><Save className="w-4 h-4 mr-2" /> Save Social Profiles</>}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* ── NOTIFICATIONS ── */}
          <TabsContent value="notifications" className="space-y-6">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>Notification Preferences</CardTitle>
                <CardDescription>Control which events trigger alerts and reports for this project.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-0 divide-y divide-slate-50">
                {[
                  { key: "seo_alerts", label: "SEO Issue Alerts", desc: "Get notified when new SEO issues are detected or resolved." },
                  { key: "social_alerts", label: "Social Metric Alerts", desc: "Alerts for engagement drops or follower milestones." },
                  { key: "weekly_report", label: "Weekly Summary Report", desc: "Receive a digest of your project's weekly performance." },
                  { key: "task_reminders", label: "Task Due Reminders", desc: "Reminders when action center tasks are approaching their deadline." },
                  { key: "competitor_updates", label: "Competitor Activity", desc: "Alerts when tracked competitors show significant changes." },
                ].map((item) => (
                  <div key={item.key} className="flex items-center justify-between py-5">
                    <div>
                      <p className="text-sm font-semibold text-slate-900">{item.label}</p>
                      <p className="text-xs text-slate-500 mt-0.5">{item.desc}</p>
                    </div>
                    <Switch
                      checked={notifPrefs[item.key as keyof typeof notifPrefs]}
                      onChange={(e) => setNotifPrefs(prev => ({ ...prev, [item.key]: e.target.checked }))}
                    />
                  </div>
                ))}
              </CardContent>
            </Card>
          </TabsContent>

          {/* ── SECURITY ── */}
          <TabsContent value="security" className="space-y-6">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>Access Control & Activity</CardTitle>
                <CardDescription>Project access log and security overview.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="space-y-3">
                  <h4 className="text-xs font-bold uppercase tracking-widest text-slate-400">Project Info</h4>
                  <div className="bg-slate-50 rounded-xl divide-y divide-slate-100">
                    {[
                      { label: "Project ID", value: `DM-${project?.id}-X90` },
                      { label: "Status", value: project?.status ? project.status.charAt(0).toUpperCase() + project.status.slice(1) : "Active" },
                      { label: "Goal", value: project?.goal ? project.goal.toUpperCase() : "—" },
                      { label: "Health Score", value: `${project?.health_score || 0}/100` },
                      { label: "Created", value: project?.created_at ? new Date(project.created_at).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' }) : "—" },
                    ].map((row) => (
                      <div key={row.label} className="flex items-center justify-between px-4 py-3">
                        <span className="text-xs font-semibold text-slate-500">{row.label}</span>
                        <span className="text-sm font-semibold text-slate-900 font-mono">{row.value}</span>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="space-y-3">
                  <h4 className="text-xs font-bold uppercase tracking-widest text-slate-400">Access</h4>
                  <div className="flex items-center justify-between p-4 rounded-xl bg-slate-50 border border-slate-100">
                    <div>
                      <p className="text-sm font-semibold text-slate-900">Private Project</p>
                      <p className="text-xs text-slate-500 mt-0.5">Only you can view and edit this project's data.</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-emerald-500" />
                      <span className="text-xs font-bold text-emerald-700">Secured</span>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* ── DATA EXPORT ── */}
          <TabsContent value="data" className="space-y-6">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>Data Export</CardTitle>
                <CardDescription>Download your project data for backup or external analysis.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="p-6 rounded-2xl border border-slate-200 bg-slate-50/50 flex flex-col gap-4">
                    <div>
                      <h4 className="font-semibold text-slate-900">Project Configuration</h4>
                      <p className="text-xs text-slate-500 mt-1">Export project settings, handles, and metadata as JSON.</p>
                    </div>
                    <Button
                      onClick={handleExportJSON}
                      variant="outline"
                      className="rounded-xl border-slate-200 gap-2 w-full"
                    >
                      <Download className="w-4 h-4" /> Export JSON
                    </Button>
                  </div>
                  <div className="p-6 rounded-2xl border border-dashed border-slate-200 flex flex-col gap-4 opacity-60">
                    <div>
                      <h4 className="font-semibold text-slate-900">Full Analytics Report</h4>
                      <p className="text-xs text-slate-500 mt-1">Export complete metrics history as CSV (available on Pro+).</p>
                    </div>
                    <Button
                      variant="outline"
                      className="rounded-xl border-slate-200 gap-2 w-full"
                      disabled
                    >
                      <Download className="w-4 h-4" /> Export CSV — Pro+
                    </Button>
                  </div>
                </div>
                <div className="p-4 rounded-xl bg-amber-50 border border-amber-100 text-xs text-amber-800 font-medium">
                  ⚠️ Exported data may contain sensitive handles and API-linked information. Store securely.
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}

export default function ProjectSettingsPage() {
  return (
    <Suspense fallback={<div className="flex items-center justify-center h-[60vh]"><Loader2 className="w-8 h-8 animate-spin text-slate-300" /></div>}>
      <ProjectSettingsContent />
    </Suspense>
  );
}
