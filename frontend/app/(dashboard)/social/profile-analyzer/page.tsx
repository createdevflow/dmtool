"use client";

import { motion } from "framer-motion";
import { 
  Users, TrendingUp, Activity, BarChart3, 
  Instagram, Twitter, Linkedin, Facebook,
  Loader2, ArrowUpRight, ArrowDownRight,
  MessageSquare, Heart, Eye, Image as ImageIcon
} from "lucide-react";
import { useState, useEffect } from "react";
import { dashboardApi, socialApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

const platformIcons: Record<string, any> = {
  Instagram, Twitter, LinkedIn: Linkedin, Facebook,
  "Twitter/X": Twitter,
};

export default function ProfileAnalyzerPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [socialMetrics, setSocialMetrics] = useState<any[]>([]);
  const [socialHistory, setSocialHistory] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [activePlatform, setActivePlatform] = useState<string>("");
  const [syncing, setSyncing] = useState(false);

  const [publicProfile, setPublicProfile] = useState<any>(null);
  const [lookingUp, setLookingUp] = useState(false);

  const handleSync = async () => {
    if (!project?.id) return;
    setSyncing(true);
    try {
      const res = await dashboardApi.refreshSocial(project.id);
      if (res.data.success) {
        toast(res.data.meta?.message || "Profile metrics updated successfully", "success");
        // Also try to fetch public profile data
        if (project.ig_handle) {
          const ppRes = await dashboardApi.getPublicProfile(project.ig_handle, "instagram", project.id);
          setPublicProfile(ppRes.data?.data ?? null);
        }
        fetchData(project.id);
      }
    } catch (err: any) {
      const msg = err.response?.data?.error?.message || err.response?.data?.message || "Failed to sync profiles";
      toast(msg, "error");
    } finally {
      setSyncing(false);
    }
  };



  const fetchData = async (targetProjectId?: number) => {
    setLoading(true);
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data.data ?? [];
      setProjects(allProjects);

      if (allProjects.length > 0) {
        const selected = targetProjectId 
          ? allProjects.find((p: any) => p.id === targetProjectId) || allProjects[allProjects.length - 1]
          : allProjects[allProjects.length - 1];
        
        setProject(selected);

        const [sRes, hRes] = await Promise.all([
          dashboardApi.getSocialInsights(selected.id),
          dashboardApi.getSocialHistoryForDelta(selected.id, 30),
        ]);
        const rawMetrics = sRes.data?.data;
        const rawHistory = hRes.data?.data;
        const metrics = Array.isArray(rawMetrics) ? rawMetrics : [];
        const history = Array.isArray(rawHistory) ? rawHistory : [];
        setSocialMetrics(metrics);
        setSocialHistory(history);

        if (metrics.length > 0) setActivePlatform(metrics[0].platform);

      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleProjectChange = (newProj: any) => {
    fetchData(newProj.id);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[60vh]">
        <Loader2 className="w-8 h-8 animate-spin text-slate-300" />
      </div>
    );
  }

  const activeSocial = socialMetrics.find(s => s.platform === activePlatform) ?? null;
  const totalReach = socialMetrics.reduce((sum, s) => sum + (s.reach ?? 0), 0);
  const totalEngagement = socialMetrics.reduce((sum, s) => sum + (s.engagement ?? 0), 0);
  const totalFollowers = socialMetrics.reduce((sum, s) => sum + (s.followers ?? 0), 0);
  const engagementRate = totalReach > 0 ? ((totalEngagement / totalReach) * 100).toFixed(2) : "0.00";

  // Derive per-platform handle from project
  const handleForPlatform = (platform: string) => {
    if (!project) return "username";
    if (platform.toLowerCase().includes("instagram")) return project.ig_handle || "username";
    if (platform.toLowerCase().includes("twitter")) return project.twitter_handle || "username";
    if (platform.toLowerCase().includes("linkedin")) return project.linkedin_handle || "username";
    if (platform.toLowerCase().includes("facebook")) return project.facebook_handle || "username";
    return "username";
  };

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
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Social Intelligence</h2>
          <div className="flex items-center gap-4">
             <p className="text-3xl font-semibold text-slate-900 tracking-tight">Profile Analyzer</p>
             <Button 
               variant="outline" 
               size="sm" 
               onClick={handleSync}
               disabled={syncing}
               className="rounded-lg h-8 px-3 text-[11px] font-bold uppercase tracking-wider bg-slate-50 border-slate-200 hover:bg-white transition-all gap-2"
             >
               {syncing ? <Loader2 className="w-3 h-3 animate-spin" /> : <TrendingUp className="w-3 h-3" />}
               {syncing ? "Syncing..." : "Sync Live Data"}
             </Button>
          </div>
        </div>
        
        {socialMetrics.length > 0 && (
          <Tabs value={activePlatform} onValueChange={setActivePlatform} className="w-auto">
            <TabsList className="rounded-xl bg-slate-100/80 p-1 h-11 border border-slate-200/50">
              {socialMetrics.map(s => {
                const Icon = platformIcons[s.platform] ?? BarChart3;
                return (
                  <TabsTrigger key={s.id} value={s.platform} className="rounded-lg font-semibold text-[11px] px-6 data-[state=active]:bg-white data-[state=active]:shadow-sm data-[state=active]:text-slate-900 text-slate-500 gap-2">
                    <Icon className="w-3.5 h-3.5" /> {s.platform}
                  </TabsTrigger>
                );
              })}
            </TabsList>
          </Tabs>
        )}
      </div>

      {socialMetrics.length === 0 ? (
        <div className="rounded-3xl border-2 border-dashed border-slate-100 flex flex-col items-center justify-center py-32 text-center">
          <Users className="w-12 h-12 text-slate-200 mb-4" />
          {(project?.ig_handle || project?.twitter_handle || project?.linkedin_handle || project?.fb_handle) ? (
            <>
              <p className="text-slate-500 font-semibold text-lg">Profiles ready for analysis</p>
              <p className="text-slate-400 text-sm mt-1">We found social handles linked to this project. Sync data to generate insights.</p>
              <Button disabled={syncing} className="mt-8 rounded-xl bg-brand-600 hover:bg-brand-500 text-white px-8 h-12 font-semibold" onClick={handleSync}>
                {syncing ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <TrendingUp className="w-4 h-4 mr-2" />}
                Sync Profiles Now
              </Button>
            </>
          ) : (
            <>
              <p className="text-slate-500 font-semibold text-lg">No social profiles connected</p>
              <p className="text-slate-400 text-sm mt-1">Add your social handles in project settings to see analytics here.</p>
              <Button className="mt-8 rounded-xl bg-slate-900 text-white px-8 h-12 font-semibold" onClick={() => window.location.href="/projects/settings"}>
                Go to Settings
              </Button>
            </>
          )}
        </div>
      ) : (
        <>
          {activeSocial?.is_simulated && (
            <motion.div 
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              className="bg-amber-50/50 border border-amber-100 rounded-3xl p-6 flex flex-col md:flex-row md:items-center justify-between gap-6"
            >
              <div className="flex items-start gap-4">
                <div className="w-10 h-10 rounded-xl bg-amber-100 flex items-center justify-center shrink-0">
                  <Activity className="w-5 h-5 text-amber-600" />
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-bold text-amber-900 uppercase tracking-wider">Simulated Mode Active</p>
                  <p className="text-sm text-amber-700/80 font-medium">
                    The metrics for <span className="font-bold">@{handleForPlatform(activePlatform)}</span> are currently estimated. Connect your account for real-time accuracy.
                  </p>
                </div>
              </div>
              <Button 
                onClick={() => window.location.href="/integrations"}
                className="rounded-xl bg-amber-600 hover:bg-amber-700 text-white px-6 h-11 font-bold text-xs uppercase tracking-widest shadow-sm shadow-amber-200"
              >
                Connect Live API
              </Button>
            </motion.div>
          )}

          {/* Profile Info & Quick Stats */}
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
            
            {/* Profile Card */}
            <Card className="lg:col-span-4 border-slate-100 shadow-xl shadow-slate-200/20 rounded-3xl p-8 bg-white text-center flex flex-col items-center">
               <div className="relative mb-6">
                  <div className="w-24 h-24 rounded-3xl bg-slate-50 border border-slate-100 flex items-center justify-center overflow-hidden shadow-inner">
                     <Users className="w-10 h-10 text-slate-200" />
                  </div>
                  <div className="absolute -bottom-2 -right-2 w-8 h-8 rounded-xl bg-slate-900 flex items-center justify-center text-white border-2 border-white">
                     {(() => { const Icon = platformIcons[activePlatform] ?? BarChart3; return <Icon className="w-4 h-4" />; })()}
                  </div>
               </div>
               <h3 className="text-xl font-bold text-slate-900">@{handleForPlatform(activePlatform)}</h3>
               <div className="text-sm text-slate-500 font-medium mt-1 flex items-center justify-center gap-2">
                 {activePlatform}
                 {activeSocial?.is_simulated && (
                    <Badge variant="outline" className="bg-amber-50 text-amber-700 border-amber-200 text-[10px] px-1.5 py-0 h-4 rounded-md font-bold uppercase tracking-tight">
                      Simulated
                    </Badge>
                 )}
               </div>
               
               <div className="grid grid-cols-3 gap-4 w-full mt-8 pt-8 border-t border-slate-50">
                  <div>
                     <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Followers</p>
                     <p className="text-lg font-bold text-slate-900 mt-1">{activeSocial?.followers?.toLocaleString() ?? "—"}</p>
                  </div>
                  <div>
                     <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Reach</p>
                     <p className="text-lg font-bold text-slate-900 mt-1">{activeSocial?.reach?.toLocaleString() ?? "—"}</p>
                  </div>
                  <div>
                     <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Engage</p>
                     <p className="text-lg font-bold text-slate-900 mt-1">{activeSocial?.engagement?.toLocaleString() ?? "—"}</p>
                  </div>
               </div>

               <Button className="w-full mt-8 rounded-xl bg-slate-900 text-white font-semibold h-12">
                 Detailed Audit
               </Button>
            </Card>

           {/* Engagement Stats with REAL deltas */}
           <div className="lg:col-span-8 grid grid-cols-1 sm:grid-cols-2 gap-6">
              {(() => {
                // Compute real deltas from history per platform
                const histForPlatform = socialHistory.filter(
                  (h: any) => h.platform?.toLowerCase() === activePlatform?.toLowerCase()
                ).sort((a: any, b: any) => new Date(a.recorded_at).getTime() - new Date(b.recorded_at).getTime());

                const getDelta = (field: string) => {
                  if (histForPlatform.length < 2) return { label: "—", trend: "up" };
                  const prev = histForPlatform[Math.max(0, histForPlatform.length - 8)][field] ?? 0;
                  const curr = histForPlatform[histForPlatform.length - 1][field] ?? 0;
                  if (prev === 0) return { label: "—", trend: "up" };
                  const pct = ((curr - prev) / prev) * 100;
                  return { label: `${pct >= 0 ? "+" : ""}${pct.toFixed(1)}%`, trend: pct >= 0 ? "up" : "down" };
                };

                const stats = [
                  { label: "Total Followers", value: totalFollowers.toLocaleString(), delta: getDelta("followers"), icon: Users },
                  { label: "Engagement Rate", value: `${engagementRate}%`, delta: getDelta("engagement"), icon: Activity },
                  { label: "Total Reach", value: totalReach.toLocaleString(), delta: getDelta("reach"), icon: Eye },
                  { label: "Total Engagement", value: totalEngagement.toLocaleString(), delta: getDelta("engagement_count"), icon: Heart },
                ];

                return stats.map((stat, i) => (
                  <motion.div
                    key={i}
                    initial={{ opacity: 0, y: 8 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.05 }}
                  >
                    <Card className="border-slate-100 shadow-none bg-slate-50/50 rounded-2xl p-6 hover:bg-white hover:border-slate-200 transition-all group">
                       <div className="flex justify-between items-start mb-6">
                          <div className="p-2.5 rounded-xl bg-white border border-slate-100 text-slate-400 group-hover:text-slate-900 transition-colors">
                             <stat.icon className="w-5 h-5" />
                          </div>
                          <div className={`flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded-md ${stat.delta.trend === 'up' ? 'text-emerald-600 bg-emerald-50' : 'text-rose-600 bg-rose-50'}`}>
                             {stat.delta.trend === 'up' ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                             {stat.delta.label}
                          </div>
                       </div>
                       <div>
                          <p className="text-[11px] font-bold text-slate-400 uppercase tracking-widest">{stat.label}</p>
                          <p className="text-3xl font-bold text-slate-900 mt-1 tracking-tight">{stat.value}</p>
                       </div>
                    </Card>
                  </motion.div>
                ));
              })()}
           </div>
          </div>

          {/* Platform Summary */}
          <div className="space-y-8">
            <div className="space-y-1">
              <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Channel Overview</h2>
              <p className="text-2xl font-semibold text-slate-900 tracking-tight">All Platforms</p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {socialMetrics.map((s: any, i: number) => {
                const Icon = platformIcons[s.platform] ?? BarChart3;
                return (
                  <Card key={s.id ?? i} className="border-slate-100 shadow-none rounded-2xl overflow-hidden hover:shadow-lg hover:shadow-slate-200/30 transition-all">
                    <CardContent className="p-6">
                      <div className="flex items-center gap-3 mb-6">
                        <div className="w-10 h-10 rounded-xl bg-slate-900 flex items-center justify-center text-white">
                          <Icon className="w-5 h-5" />
                        </div>
                        <div>
                          <p className="font-bold text-slate-900 text-sm">{s.platform}</p>
                          <p className="text-[10px] text-slate-400 font-semibold">@{handleForPlatform(s.platform)}</p>
                        </div>
                      </div>
                      <div className="grid grid-cols-3 gap-4">
                        <div>
                          <p className="text-xl font-bold text-slate-900">{s.followers?.toLocaleString() ?? 0}</p>
                          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-1">Fans</p>
                        </div>
                        <div>
                          <p className="text-xl font-bold text-slate-900">{s.reach?.toLocaleString()}</p>
                          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-1">Reach</p>
                        </div>
                        <div>
                          <p className="text-xl font-bold text-emerald-600">{s.engagement?.toLocaleString()}</p>
                          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mt-1">Engage</p>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
