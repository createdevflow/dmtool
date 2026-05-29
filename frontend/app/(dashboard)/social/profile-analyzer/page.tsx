"use client";

import { motion } from "framer-motion";
import { 
  Users, TrendingUp, Activity, BarChart3, 
  Instagram, Twitter, Linkedin, Facebook,
  Loader2, ArrowUpRight, ArrowDownRight,
  MessageSquare, Heart, Eye, Image as ImageIcon
} from "lucide-react";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Legend, Tooltip as RechartsTooltip } from 'recharts';

const mockAudience = [
  { name: 'Followers', value: 75, color: '#0ea5e9' },
  { name: 'Non-followers', value: 25, color: '#e2e8f0' }
];

const mockContentSplit = [
  { type: 'Reels', percentage: 98.5, color: 'bg-rose-500' },
  { type: 'Stories', percentage: 1.5, color: 'bg-amber-500' },
  { type: 'Posts', percentage: 0, color: 'bg-sky-500' }
];

const mockTopContent = [
  { id: 1, type: 'Reel', views: '12.4K', interactions: 845, date: 'Oct 12', img: 'https://images.unsplash.com/photo-1611162617213-7d7a39e9b1d7?w=150&h=150&fit=crop' },
  { id: 2, type: 'Reel', views: '8.2K', interactions: 520, date: 'Oct 10', img: 'https://images.unsplash.com/photo-1515378791036-0648a3ef77b2?w=150&h=150&fit=crop' },
  { id: 3, type: 'Reel', views: '5.1K', interactions: 310, date: 'Oct 05', img: 'https://images.unsplash.com/photo-1522202176988-66273c2fd55f?w=150&h=150&fit=crop' },
  { id: 4, type: 'Story', views: '3.4K', interactions: 120, date: 'Oct 04', img: 'https://images.unsplash.com/photo-1542204165-65bf26472b9b?w=150&h=150&fit=crop' },
  { id: 5, type: 'Reel', views: '2.8K', interactions: 95, date: 'Oct 01', img: 'https://images.unsplash.com/photo-1518770660439-4636190af475?w=150&h=150&fit=crop' },
];

const mockActiveTimes = [
  { day: 'Mon', active: 65 },
  { day: 'Tue', active: 70 },
  { day: 'Wed', active: 85 },
  { day: 'Thu', active: 90 },
  { day: 'Fri', active: 100 },
  { day: 'Sat', active: 80 },
  { day: 'Sun', active: 60 },
];

const platformIcons: Record<string, any> = {
  Instagram, Twitter, LinkedIn: Linkedin, Facebook,
  "Twitter/X": Twitter,
};

const normalizePlatform = (platform: string) => platform.trim().toLowerCase();

const dedupeLatestMetrics = (items: any[]) => {
  const ordered = [...items].sort((a, b) => {
    const aSimulated = a?.is_simulated ? 1 : 0;
    const bSimulated = b?.is_simulated ? 1 : 0;
    if (aSimulated !== bSimulated) return aSimulated - bSimulated;

    const aRecorded = new Date(a?.recorded_at ?? a?.recordedAt ?? 0).getTime();
    const bRecorded = new Date(b?.recorded_at ?? b?.recordedAt ?? 0).getTime();
    return bRecorded - aRecorded;
  });

  const seen = new Set<string>();
  const deduped: any[] = [];

  for (const item of ordered) {
    const key = normalizePlatform(item?.platform ?? "");
    if (!key || seen.has(key)) continue;
    seen.add(key);
    deduped.push(item);
  }

  return deduped;
};

export default function ProfileAnalyzerPage() {
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [socialMetrics, setSocialMetrics] = useState<any[]>([]);
  const [socialHistory, setSocialHistory] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [activePlatform, setActivePlatform] = useState<string>("");
  const [syncing, setSyncing] = useState(false);
  const [dateRange, setDateRange] = useState("30");

  const [publicProfile, setPublicProfile] = useState<any>(null);
  const [lookingUp, setLookingUp] = useState(false);

  const syncLiveSocialInsights = async (projectId: number) => {
    try {
      await dashboardApi.refreshSocial(projectId);
    } catch (err) {
      console.error("Failed to refresh live social insights", err);
    }
  };

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
        } else if (project.fb_handle || project.facebook_handle) {
          const fbHandle = project.fb_handle || project.facebook_handle;
          const ppRes = await dashboardApi.getPublicProfile(fbHandle, "facebook", project.id);
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
        let savedProjectId = null;
        try { savedProjectId = parseInt(localStorage.getItem("dmtool_active_project_id") || "0"); } catch (e) {}
        const defaultProject = allProjects.find((p: any) => p.id === savedProjectId) || allProjects[allProjects.length - 1];
        const selected = targetProjectId
          ? allProjects.find((p: any) => p.id === targetProjectId) || defaultProject
          : defaultProject;
        if (selected) localStorage.setItem("dmtool_active_project_id", selected.id.toString());
        
        setProject(selected);

        const [sRes, hRes] = await Promise.all([
          dashboardApi.getSocialInsights(selected.id),
          dashboardApi.getSocialHistoryForDelta(selected.id, 90),
        ]);
        const rawMetrics = sRes.data?.data;
        const rawHistory = hRes.data?.data;
        const metrics = Array.isArray(rawMetrics) ? dedupeLatestMetrics(rawMetrics) : [];
        const history = Array.isArray(rawHistory) ? rawHistory : [];

        const hasMetaSocial = !!(selected?.ig_handle || selected?.fb_handle || selected?.facebook_handle);
        const needsLiveRefresh = hasMetaSocial && (metrics.length === 0 || metrics.every((item: any) => item.is_simulated));

        if (needsLiveRefresh) {
          await syncLiveSocialInsights(selected.id);
          const [refreshedInsights, refreshedHistory] = await Promise.all([
            dashboardApi.getSocialInsights(selected.id),
            dashboardApi.getSocialHistoryForDelta(selected.id, 90),
          ]);
          const refreshedMetrics = Array.isArray(refreshedInsights.data?.data) ? dedupeLatestMetrics(refreshedInsights.data.data) : [];
          const refreshedHistoryData = Array.isArray(refreshedHistory.data?.data) ? refreshedHistory.data.data : [];
          setSocialMetrics(refreshedMetrics);
          setSocialHistory(refreshedHistoryData);

          if (refreshedMetrics.length > 0) {
            const saved = localStorage.getItem("dmtool_active_platform");
            const target = refreshedMetrics.some((m: any) => m.platform === saved) ? saved : refreshedMetrics[0].platform;
            setActivePlatform(target);
            localStorage.setItem("dmtool_active_platform", target);
          }
          return;
        }

        setSocialMetrics(metrics);
        setSocialHistory(history);

        if (metrics.length > 0) {
          const saved = localStorage.getItem("dmtool_active_platform");
          const target = metrics.some((m: any) => m.platform === saved) ? saved : metrics[0].platform;
          setActivePlatform(target);
          localStorage.setItem("dmtool_active_platform", target);
        }

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

  const activeSocial = socialMetrics.find(s => normalizePlatform(s.platform) === normalizePlatform(activePlatform)) ?? null;
  const totalReach = activeSocial?.reach ?? 0;
  const totalEngagement = activeSocial?.engagement_count ?? activeSocial?.engagementCount ?? 0;
  const totalFollowers = activeSocial?.followers ?? 0;
  const engagementRate = activeSocial?.engagement ?? 0;

  // Derive per-platform handle from project
  const handleForPlatform = (platform: string) => {
    if (!project) return "username";
    const platformName = normalizePlatform(platform);
    if (platformName.includes("instagram")) return project.ig_handle || "username";
    if (platformName.includes("twitter")) return project.twitter_handle || "username";
    if (platformName.includes("linkedin")) return project.linkedin_handle || "username";
    if (platformName.includes("facebook")) return project.facebook_handle || project.fb_handle || "username";
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
             <div className="flex items-center gap-3">
               <Select value={dateRange} onValueChange={setDateRange}>
                 <SelectTrigger className="w-[140px] h-8 text-[11px] font-bold uppercase tracking-widest bg-white border-slate-200 shadow-sm rounded-lg hover:bg-slate-50 transition-colors focus:ring-0 focus:ring-offset-0">
                   <SelectValue />
                 </SelectTrigger>
                 <SelectContent>
                   <SelectItem value="7">Last Week</SelectItem>
                   <SelectItem value="30">Last Month</SelectItem>
                   <SelectItem value="90">Last 3 Months</SelectItem>
                 </SelectContent>
               </Select>
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
        </div>
        
        {socialMetrics.length > 0 && (
          <Tabs 
            value={activePlatform} 
            onValueChange={(val) => {
              setActivePlatform(val);
              localStorage.setItem("dmtool_active_platform", val);
            }} 
            className="w-auto"
          >
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
          {(project?.ig_handle || project?.twitter_handle || project?.linkedin_handle || project?.facebook_handle || project?.fb_handle) ? (
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
                  {activeSocial?.profile_picture_url ? (
                       <Avatar className="h-24 w-24 rounded-3xl">
                         <AvatarImage src={activeSocial.profile_picture_url} alt={handleForPlatform(activePlatform)} className="object-cover" />
                         <AvatarFallback className="rounded-3xl bg-slate-100">
                           <Users className="w-10 h-10 text-slate-200" />
                         </AvatarFallback>
                       </Avatar>
                     ) : (
                       <Users className="w-10 h-10 text-slate-200" />
                     )}
                  </div>
                  <div className="absolute -bottom-2 -right-2 w-8 h-8 rounded-xl bg-slate-900 flex items-center justify-center text-white border-2 border-white">
                     {(() => { const Icon = platformIcons[activePlatform] ?? BarChart3; return <Icon className="w-4 h-4" />; })()}
                  </div>
               </div>
               <h3 className="text-xl font-bold text-slate-900">
                 {activeSocial?.display_name ? activeSocial.display_name : `@${handleForPlatform(activePlatform)}`}
               </h3>
               {activeSocial?.username && (
                 <p className="text-sm text-slate-500 mt-1">@{activeSocial.username}</p>
               )}
               <div className="text-sm text-slate-500 font-medium mt-1 flex items-center justify-center gap-2">
                 {activePlatform}
                 {activeSocial?.is_simulated && (
                    <Badge variant="outline" className="bg-amber-50 text-amber-700 border-amber-200 text-[10px] px-1.5 py-0 h-4 rounded-md font-bold uppercase tracking-tight">
                      Simulated
                    </Badge>
                 )}
               </div>

               {activeSocial && (activePlatform.toLowerCase().includes("instagram") || activePlatform.toLowerCase().includes("facebook")) && (
                 <div className="w-full mt-6 rounded-2xl bg-slate-50/70 border border-slate-100 p-4 text-left space-y-3">
                    <div className="grid grid-cols-2 gap-3 text-center">
                     <div className="rounded-xl bg-white border border-slate-100 p-3">
                       <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Followers</p>
                       <p className="text-base font-bold text-slate-900 mt-1">{activeSocial.followers?.toLocaleString?.() ?? "—"}</p>
                     </div>
                     <div className="rounded-xl bg-white border border-slate-100 p-3">
                       <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Following</p>
                       <p className="text-base font-bold text-slate-900 mt-1">{activeSocial.following_count?.toLocaleString?.() ?? activeSocial.followingCount?.toLocaleString?.() ?? "—"}</p>
                     </div>
                     <div className="rounded-xl bg-white border border-slate-100 p-3">
                       <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Posts</p>
                       <p className="text-base font-bold text-slate-900 mt-1">{activeSocial.posts_count?.toLocaleString?.() ?? activeSocial.postsCount?.toLocaleString?.() ?? "—"}</p>
                     </div>
                     <div className="rounded-xl bg-white border border-slate-100 p-3">
                       <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Weekly Reach</p>
                       <p className="text-base font-bold text-slate-900 mt-1">{activeSocial.weekly_reach?.toLocaleString?.() ?? activeSocial.weeklyReach?.toLocaleString?.() ?? "—"}</p>
                     </div>
                   </div>

                   {activeSocial.display_name && (
                     <div>
                       <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mb-1">{activePlatform.toLowerCase().includes("facebook") ? "Page Name" : "Display Name"}</p>
                       <p className="text-sm font-semibold text-slate-900">{activeSocial.display_name}</p>
                     </div>
                   )}

                   {activeSocial.biography && (
                     <div>
                       <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mb-1">Bio</p>
                       <p className="text-xs leading-5 text-slate-600 whitespace-pre-line">{activeSocial.biography}</p>
                     </div>
                   )}

                   {activeSocial.website && (
                     <div>
                       <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mb-1">{activePlatform.toLowerCase().includes("facebook") ? "Page Link" : "Website"}</p>
                       <a
                         href={activeSocial.website}
                         target="_blank"
                         rel="noreferrer"
                         className="text-xs font-semibold text-slate-900 underline underline-offset-4 break-all"
                       >
                         {activeSocial.website}
                       </a>
                     </div>
                   )}
                 </div>
               )}
               

            </Card>

           {/* Detailed Instagram-Style Insights */}
           <div className="lg:col-span-8 space-y-6">
              {(() => {
                const histForPlatform = socialHistory.filter(
                  (h: any) => h.platform?.toLowerCase() === activePlatform?.toLowerCase()
                ).sort((a: any, b: any) => new Date(a.recorded_at).getTime() - new Date(b.recorded_at).getTime());

                const rangeDays = parseInt(dateRange);
                const cutoffDate = new Date();
                cutoffDate.setDate(cutoffDate.getDate() - rangeDays);

                const getDelta = (field: string) => {
                  if (histForPlatform.length < 2) return { diff: 0, pct: 0 };
                  const currRecord = histForPlatform[histForPlatform.length - 1];
                  const curr = currRecord[field] ?? 0;
                  const rangeRecords = histForPlatform.filter((h: any) => new Date(h.recorded_at) >= cutoffDate);
                  const prevRecord = rangeRecords.length > 0 ? rangeRecords[0] : histForPlatform[0];
                  const prev = prevRecord[field] ?? 0;
                  if (prev === 0) return { diff: curr, pct: 0 };
                  const diff = curr - prev;
                  return { diff, pct: (diff / prev) * 100 };
                };

                const followersDelta = getDelta("followers");
                const reachDelta = getDelta("reach");
                
                // --- Parse Dynamic Data ---
                let dynamicTopContent = mockTopContent;
                if (activeSocial?.top_content) {
                  try {
                    const parsed = JSON.parse(activeSocial.top_content);
                    if (Array.isArray(parsed) && parsed.length > 0) dynamicTopContent = parsed;
                  } catch (e) {}
                }

                // sort by interactions/engagement desc and take top 5
                try {
                  if (Array.isArray(dynamicTopContent)) {
                    dynamicTopContent = [...dynamicTopContent].sort((a: any, b: any) => {
                      const aScore = (a.interactions ?? a.interaction ?? a.engagement ?? 0);
                      const bScore = (b.interactions ?? b.interaction ?? b.engagement ?? 0);
                      return bScore - aScore;
                    }).slice(0, 5);
                  }
                } catch (e) {}

                let dynamicContentSplit = mockContentSplit;
                if (activeSocial?.content_split) {
                  try {
                    const parsed = JSON.parse(activeSocial.content_split);
                    if (Array.isArray(parsed) && parsed.length > 0) dynamicContentSplit = parsed;
                  } catch (e) {}
                }

                // active_times may be stored as JSON array [{day, active}] or as an object mapping
                let dynamicActiveTimes = mockActiveTimes;
                if (activeSocial?.active_times) {
                  try {
                    const parsed = JSON.parse(activeSocial.active_times);
                    if (Array.isArray(parsed) && parsed.length > 0) {
                      dynamicActiveTimes = parsed;
                    } else if (parsed && typeof parsed === 'object') {
                      // if object like { Mon: 10, Tue: 20 }
                      const arr: any[] = [];
                      for (const k of Object.keys(parsed)) {
                        arr.push({ day: k, active: parsed[k] });
                      }
                      if (arr.length > 0) dynamicActiveTimes = arr;
                    }
                  } catch (e) {
                    // keep mock
                  }
                }

                // Audience insights parsing
                // We will build: pie data for audience split, ageGender for grouped bar, and country/city lists
                let dynamicAudience: any[] = mockAudience;
                let ageGenderData: any[] = [];
                let topCountries: [string, number][] = [];
                let topCities: [string, number][] = [];

                if (activeSocial?.audience_insights) {
                  try {
                    const parsed = JSON.parse(activeSocial.audience_insights);
                    // countries
                    if (parsed.countries && typeof parsed.countries === 'object') {
                      topCountries = Object.entries(parsed.countries).sort((a: any, b: any) => b[1] - a[1]).slice(0, 6);
                    }
                    if (parsed.cities && typeof parsed.cities === 'object') {
                      topCities = Object.entries(parsed.cities).sort((a: any, b: any) => b[1] - a[1]).slice(0, 6);
                    }
                    // age_gender -> expected shape: { "18-24": { male: X, female: Y }, ... }
                    if (parsed.age_gender && typeof parsed.age_gender === 'object') {
                      ageGenderData = Object.keys(parsed.age_gender).map(age => {
                        const obj = parsed.age_gender[age];
                        return {
                          age,
                          male: obj?.male ?? 0,
                          female: obj?.female ?? 0
                        };
                      }).sort((a: any, b: any) => {
                        // try to sort by age range start
                        const aNum = parseInt(a.age.split('-')[0] ?? '0');
                        const bNum = parseInt(b.age.split('-')[0] ?? '0');
                        return aNum - bNum;
                      });
                    }

                    // For pie chart, create a simple split: top country vs others OR use audience breakdown if provided
                    if (topCountries.length > 0) {
                      const top = topCountries[0];
                      const topValue = top[1];
                      const total = topCountries.reduce((s: number, c: any) => s + c[1], 0);
                      dynamicAudience = [
                        { name: top[0], value: Math.round((topValue / (total || 1)) * 100), color: '#0ea5e9' },
                        { name: 'Other', value: Math.max(100 - Math.round((topValue / (total || 1)) * 100), 0), color: '#e2e8f0' }
                      ];
                    }
                  } catch (e) {
                    // keep defaults
                  }
                }
                
                return (
                  <>
                    {/* Insight Summary Banner */}
                    <div className="w-full bg-slate-900 rounded-2xl p-6 text-white flex flex-col sm:flex-row gap-6 items-center justify-between shadow-xl shadow-slate-900/10">
                       <div className="flex items-center gap-4">
                         <div className="w-12 h-12 rounded-xl bg-white/10 flex items-center justify-center">
                           <Activity className="w-6 h-6 text-sky-400" />
                         </div>

                          {/* AI / Performance Insights (heuristic summaries) */}
                          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-4">
                            {(() => {
                              const insights: { title: string; text: string }[] = [];
                              // growth
                              const status = activeSocial?.status ?? 'stable';
                              insights.push({ title: 'Growth', text: status === 'growing' ? 'Your profile is growing steadily.' : status === 'dropping' ? 'Profile growth has slowed recently.' : 'Your profile is growing steadily.' });
                              // content type
                              try {
                                const reels = dynamicContentSplit.find((c: any) => c.type.toLowerCase().includes('reel'))?.percentage ?? dynamicContentSplit.find((c: any) => c.type.toLowerCase().includes('reels'))?.percentage ?? 0;
                                const topType = dynamicContentSplit.reduce((acc: any, cur: any) => (cur.percentage > (acc.percentage ?? 0) ? cur : acc), {} as any);
                                if (topType && topType.type) {
                                  insights.push({ title: 'Top Content', text: `${topType.type} are generating the highest performance.` });
                                } else {
                                  insights.push({ title: 'Top Content', text: 'Content performance is mixed.' });
                                }
                              } catch (e) {
                                insights.push({ title: 'Top Content', text: 'Content performance is mixed.' });
                              }

                              // best day
                              try {
                                const best = [...dynamicActiveTimes].sort((a: any, b: any) => (b.active ?? 0) - (a.active ?? 0))[0];
                                if (best) insights.push({ title: 'Best Day', text: `${best.day} has the highest follower activity.` });
                                else insights.push({ title: 'Best Day', text: `Follower activity data is limited.` });
                              } catch (e) {
                                insights.push({ title: 'Best Day', text: `Follower activity data is limited.` });
                              }

                              // top country
                              if (topCountries && topCountries.length > 0) {
                                insights.push({ title: 'Top Country', text: `Most audience comes from ${topCountries[0][0]}.` });
                              }

                              // ensure exactly 3 cards show
                              return insights.slice(0, 3).map((ins, i) => (
                                <Card key={i} className="p-4 rounded-xl border-slate-100 shadow-sm">
                                  <p className="text-[11px] font-bold text-slate-400 uppercase tracking-widest mb-2">{ins.title}</p>
                                  <p className="text-sm font-semibold text-slate-900">{ins.text}</p>
                                </Card>
                              ));
                            })()}
                          </div>
                         <div>
                           <p className="text-sm font-semibold text-slate-300">Profile Highlight</p>
                           <p className="text-lg font-bold">Your Reels are driving 98% of new reach.</p>
                         </div>
                       </div>
                       <Button className="bg-white hover:bg-slate-100 text-slate-900 font-bold rounded-xl whitespace-nowrap">
                         View Top Post
                       </Button>
                    </div>

                    {/* Audience Insights: Age/Gender + Top Locations */}
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                      <Card className="lg:col-span-2 border-slate-100 shadow-none rounded-2xl p-6">
                        <p className="text-sm font-bold text-slate-900 mb-4">Age & Gender Breakdown</p>
                        {ageGenderData.length > 0 ? (
                          <div className="h-[200px] w-full">
                            <ResponsiveContainer width="100%" height="100%">
                              <BarChart data={ageGenderData} margin={{ top: 10, right: 20, left: 0, bottom: 5 }}>
                                <XAxis dataKey="age" axisLine={false} tickLine={false} />
                                <YAxis />
                                <RechartsTooltip />
                                <Legend />
                                <Bar dataKey="male" fill="#3b82f6" name="Male" />
                                <Bar dataKey="female" fill="#ec4899" name="Female" />
                              </BarChart>
                            </ResponsiveContainer>
                          </div>
                        ) : (
                          <p className="text-sm text-slate-500">Age & gender data not available for this profile.</p>
                        )}
                      </Card>

                      <Card className="border-slate-100 shadow-none rounded-2xl p-6">
                        <p className="text-sm font-bold text-slate-900 mb-4">Top Countries</p>
                        {topCountries.length > 0 ? (
                          <div className="space-y-3">
                            {topCountries.map((c, i) => (
                              <div key={i} className="flex items-center justify-between">
                                <div className="text-sm font-semibold text-slate-700">{c[0]}</div>
                                <div className="text-sm font-bold text-slate-900">{c[1].toLocaleString()}</div>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <p className="text-sm text-slate-500">No country breakdown available.</p>
                        )}

                        <div className="mt-6">
                          <p className="text-sm font-bold text-slate-900 mb-2">Top Cities</p>
                          {topCities.length > 0 ? (
                            <div className="space-y-2">
                              {topCities.map((c, i) => (
                                <div key={i} className="flex items-center justify-between">
                                  <div className="text-sm text-slate-700">{c[0]}</div>
                                  <div className="text-sm font-bold text-slate-900">{c[1].toLocaleString()}</div>
                                </div>
                              ))}
                            </div>
                          ) : (
                            <p className="text-sm text-slate-500">No city data available.</p>
                          )}
                        </div>
                      </Card>
                    </div>

                    {/* Account Overview Grid */}
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                      {[
                        { label: "Total Followers", value: (activeSocial.followers ?? activeSocial.followers_count ?? 0).toLocaleString(), diff: followersDelta.diff },
                        { label: "Following", value: (activeSocial.following_count ?? activeSocial.followingCount ?? 0).toLocaleString(), diff: 0 },
                        { label: "Total Posts/Reels", value: (activeSocial.posts_count ?? activeSocial.postsCount ?? 0).toLocaleString(), diff: 0 },
                        { label: "Daily Reach", value: (activeSocial.profile_visits ? (activeSocial.reach ?? 0).toLocaleString() : (activeSocial.reach ?? 0).toLocaleString()), diff: reachDelta.diff },
                        { label: "Weekly Reach", value: (activeSocial.weekly_reach ?? activeSocial.weeklyReach ?? 0).toLocaleString(), diff: 0 },
                        { label: "Monthly Reach", value: (activeSocial.monthly_reach ?? activeSocial.monthlyReach ?? 0).toLocaleString(), diff: 0 },
                        { label: "Profile Visits", value: (activeSocial.profile_visits ?? activeSocial.profileVisits ?? "—").toLocaleString?.() ?? (activeSocial.profile_visits ?? activeSocial.profileVisits ?? "—") , diff: 0 },
                        { label: "Link Taps", value: (activeSocial.external_link_taps ?? activeSocial.externalLinkTaps ?? 0).toLocaleString(), diff: 0 },
                        { label: "Engagement", value: (activeSocial.engagement_count ?? activeSocial.engagementCount ?? 0).toLocaleString(), diff: 0 },
                        { label: "Engagement Rate", value: (activeSocial.engagement ?? activeSocial.engagement_rate ?? engagementRate).toLocaleString ? `${(activeSocial.engagement ?? activeSocial.engagement_rate ?? engagementRate).toFixed(1)}%` : `${(activeSocial.engagement ?? activeSocial.engagement_rate ?? engagementRate)}%`, diff: 0 }
                      ].map((stat, i) => (
                        <Card key={i} className="border-slate-100 shadow-none rounded-2xl p-5 hover:border-slate-200 transition-colors">
                          <p className="text-[11px] font-bold text-slate-400 uppercase tracking-widest mb-2">{stat.label}</p>
                          <p className="text-2xl font-bold text-slate-900">{stat.value}</p>
                          <div className={`mt-2 text-[11px] font-bold inline-flex items-center gap-1 ${stat.diff >= 0 ? 'text-emerald-600' : 'text-rose-600'}`}>
                            {stat.diff >= 0 ? <TrendingUp className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                            {stat.diff > 0 ? '+' : ''}{stat.diff.toLocaleString ? stat.diff.toLocaleString() : stat.diff} vs {dateRange} days
                          </div>
                        </Card>
                      ))}
                    </div>

                    {/* Split Insights Row */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      {/* Audience Split */}
                      <Card className="border-slate-100 shadow-none rounded-2xl p-6">
                        <p className="text-sm font-bold text-slate-900 mb-6">Audience Distribution</p>
                        <div className="flex items-center justify-between">
                          <div className="w-[120px] h-[120px]">
                            <ResponsiveContainer width="100%" height="100%">
                              <PieChart>
                                <Pie data={dynamicAudience} cx="50%" cy="50%" innerRadius={35} outerRadius={55} dataKey="value" stroke="none">
                                  {dynamicAudience.map((entry, index) => <Cell key={`cell-${index}`} fill={entry.color} />)}
                                </Pie>
                                <RechartsTooltip />
                              </PieChart>
                            </ResponsiveContainer>
                          </div>
                          <div className="space-y-4 flex-1 ml-6">
                            {dynamicAudience.map((a, i) => (
                              <div key={i} className="flex items-center justify-between">
                                <div className="flex items-center gap-2">
                                  <div className="w-2 h-2 rounded-full" style={{ backgroundColor: a.color }}></div>
                                  <span className="text-xs font-semibold text-slate-600">{a.name}</span>
                                </div>
                                <span className="text-sm font-bold text-slate-900">{a.value}%</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      </Card>

                      {/* Content Split */}
                      <Card className="border-slate-100 shadow-none rounded-2xl p-6">
                        <p className="text-sm font-bold text-slate-900 mb-6">Content Performance by Type</p>
                        <div className="space-y-5">
                          {dynamicContentSplit.map((c: any, i: number) => (
                            <div key={i}>
                              <div className="flex items-center justify-between mb-2">
                                <span className="text-xs font-semibold text-slate-600">{c.type}</span>
                                <span className="text-sm font-bold text-slate-900">{c.percentage.toFixed(1)}%</span>
                              </div>
                              <div className="w-full h-2 rounded-full bg-slate-100 overflow-hidden">
                                <motion.div 
                                  initial={{ width: 0 }} 
                                  animate={{ width: `${c.percentage}%` }} 
                                  className={`h-full ${c.color}`} 
                                />
                              </div>
                            </div>
                          ))}
                        </div>
                      </Card>
                    </div>

                    {/* Top Content */}
                    <Card className="border-slate-100 shadow-none rounded-2xl p-6 overflow-hidden">
                      <p className="text-sm font-bold text-slate-900 mb-6">Top Performing Content</p>
                      <div className="flex gap-4 overflow-x-auto pb-4 scrollbar-hide">
                        {dynamicTopContent.map((post: any) => (
                          <div key={post.id} className="min-w-[140px] space-y-3 group cursor-pointer" onClick={() => post.permalink && window.open(post.permalink, '_blank')}>
                            <div className="w-[140px] h-[180px] rounded-2xl overflow-hidden relative bg-slate-100">
                              {post.img ? (
                                <img src={post.img} alt="Post thumbnail" className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500" />
                              ) : (
                                <div className="w-full h-full flex items-center justify-center text-slate-400">
                                  <ImageIcon className="w-8 h-8 opacity-50" />
                                </div>
                              )}
                              <div className="absolute top-2 right-2 px-2 py-1 bg-black/50 backdrop-blur-md text-white text-[10px] font-bold rounded-lg uppercase tracking-wider">
                                {post.type}
                              </div>
                            </div>
                            <div>
                              <p className="text-sm font-bold text-slate-900">{post.views} <span className="text-xs text-slate-400 font-medium">Views</span></p>
                              <p className="text-[10px] text-slate-500 font-semibold">{post.interactions} Interactions • {post.date}</p>
                            </div>
                          </div>
                        ))}
                      </div>
                    </Card>

                    {/* Follower Analytics & Profile Activity */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      <Card className="border-slate-100 shadow-none rounded-2xl p-6">
                        <p className="text-sm font-bold text-slate-900 mb-6">Most Active Times</p>
                        <div className="h-[150px] w-full">
                          <ResponsiveContainer width="100%" height="100%">
                            <BarChart data={dynamicActiveTimes}>
                              <XAxis dataKey="day" axisLine={false} tickLine={false} tick={{ fontSize: 10, fill: '#94a3b8' }} />
                              <RechartsTooltip cursor={{ fill: '#f8fafc' }} contentStyle={{ borderRadius: '12px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }} />
                              <Bar dataKey="active" fill="#0ea5e9" radius={[4, 4, 4, 4]} />
                            </BarChart>
                          </ResponsiveContainer>
                        </div>
                      </Card>
                      <Card className="border-slate-100 shadow-none rounded-2xl p-6 flex flex-col justify-center space-y-6">
                        <p className="text-sm font-bold text-slate-900 mb-2">Profile Activity</p>
                        <div className="flex items-center justify-between border-b border-slate-50 pb-4">
                          <div className="flex items-center gap-3">
                            <div className="p-2 rounded-lg bg-sky-50 text-sky-600"><Users className="w-4 h-4" /></div>
                            <span className="text-sm font-semibold text-slate-600">Profile Visits</span>
                          </div>
                          <span className="text-lg font-bold text-slate-900">{(activeSocial.profile_visits ?? activeSocial.profileVisits ?? 0).toLocaleString()}</span>
                        </div>
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <div className="p-2 rounded-lg bg-emerald-50 text-emerald-600"><ArrowUpRight className="w-4 h-4" /></div>
                            <span className="text-sm font-semibold text-slate-600">External Link Taps</span>
                          </div>
                          <span className="text-lg font-bold text-slate-900">{(activeSocial.external_link_taps ?? activeSocial.externalLinkTaps ?? 0).toLocaleString()}</span>
                        </div>
                      </Card>
                    </div>

                  </>
                );
              })()}
           </div>
          </div>

        </>
      )}
    </div>
  );
}
