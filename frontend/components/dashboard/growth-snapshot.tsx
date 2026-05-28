"use client";

import { motion } from "framer-motion";
import { 
  ArrowUpRight, ArrowDownRight, Activity, Globe, 
  Users, Target, Zap, MousePointer2, TrendingUp,
  BarChart3, Sparkles
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface GrowthSnapshotProps {
  viewMode: 'website' | 'social' | 'combined';
  setViewMode: (val: 'website' | 'social' | 'combined') => void;
  project: any;
  snapshotData?: {
    websiteStats?: any[];
    socialStats?: any[];
    combinedStats?: any[];
  } | null;
}

// Map icon name strings from API to actual icon components
const iconMap: Record<string, any> = {
  Target, Globe, TrendingUp, MousePointer2,
  Activity, Users, Zap, BarChart3, Sparkles,
};

// Fallback stats shown while API data loads
const fallbackWebsite = [
  { label: "SEO Health", value: "—", change: "—", trend: "up", icon: Target },
  { label: "Organic Traffic", value: "—", change: "—", trend: "up", icon: Globe },
  { label: "Ranking Keywords", value: "—", change: "—", trend: "up", icon: TrendingUp },
  { label: "Total Backlinks", value: "—", change: "—", trend: "up", icon: MousePointer2 },
];
const fallbackSocial = [
  { label: "Engagement Rate", value: "—", change: "—", trend: "up", icon: Activity },
  { label: "New Followers", value: "—", change: "—", trend: "up", icon: Users },
  { label: "Average Reach", value: "—", change: "—", trend: "up", icon: Zap },
  { label: "Content Score", value: "—", change: "—", trend: "up", icon: BarChart3 },
];
const fallbackCombined = [
  { label: "Growth Index", value: "—", change: "—", trend: "up", icon: TrendingUp },
  { label: "Aggregate Reach", value: "—", change: "—", trend: "up", icon: Zap },
  { label: "Conversion Rate", value: "—", change: "—", trend: "up", icon: Target },
  { label: "Efficiency Index", value: "—", change: "—", trend: "up", icon: Activity },
];

function mapStats(raw?: any[]) {
  if (!raw || raw.length === 0) return null;
  return raw.map((s: any) => ({
    ...s,
    icon: iconMap[s.icon] ?? Activity,
  }));
}

export function GrowthSnapshot({ viewMode, setViewMode, project, snapshotData }: GrowthSnapshotProps) {
  const websiteStats  = mapStats(snapshotData?.websiteStats)  ?? fallbackWebsite;
  const socialStats   = mapStats(snapshotData?.socialStats)   ?? fallbackSocial;
  const combinedStats = mapStats(snapshotData?.combinedStats) ?? fallbackCombined;

  const stats = viewMode === 'website' ? websiteStats : viewMode === 'social' ? socialStats : combinedStats;

  return (
    <div className="space-y-8">
      <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-widest">Performance Metrics</h2>
          <p className="text-2xl font-semibold text-slate-900 tracking-tight">Growth Snapshot</p>
        </div>
        
        <Tabs value={viewMode} onValueChange={(val: any) => setViewMode(val)} className="w-auto">
          <TabsList className="rounded-xl bg-slate-100/80 p-1 h-10 border border-slate-200/50">
            <TabsTrigger value="website" className="rounded-lg font-medium text-[11px] px-5 data-[state=active]:bg-white data-[state=active]:shadow-sm data-[state=active]:text-slate-900 text-slate-500">Website</TabsTrigger>
            <TabsTrigger value="social" className="rounded-lg font-medium text-[11px] px-5 data-[state=active]:bg-white data-[state=active]:shadow-sm data-[state=active]:text-slate-900 text-slate-500">Social</TabsTrigger>
            <TabsTrigger value="combined" className="rounded-lg font-medium text-[11px] px-5 data-[state=active]:bg-white data-[state=active]:shadow-sm data-[state=active]:text-slate-900 text-slate-500">Combined</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {stats.map((stat: any, i: number) => (
          <motion.div 
            key={`${viewMode}-${stat.label}`} 
            initial={{ opacity: 0, y: 8 }} 
            animate={{ opacity: 1, y: 0 }} 
            transition={{ duration: 0.3, delay: i * 0.05 }}
          >
            <Card className="border-slate-200/60 shadow-none hover:shadow-lg hover:shadow-slate-200/30 transition-all group bg-white">
               <CardContent className="p-6">
                  <div className="flex justify-between items-start mb-6">
                     <div className="p-2.5 rounded-lg bg-slate-50 border border-slate-200/60 text-slate-400 group-hover:text-slate-900 transition-colors">
                        <stat.icon className="w-4 h-4" />
                     </div>
                     <div className={`flex items-center gap-1 text-[11px] font-semibold px-2 py-0.5 rounded-md ${stat.trend === 'up' ? 'text-emerald-600 bg-emerald-50' : 'text-rose-600 bg-rose-50'}`}>
                        {stat.trend === 'up' ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                        {stat.change}
                     </div>
                  </div>
                  <div className="space-y-1">
                    <p className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">{stat.label}</p>
                    <p className="text-3xl font-semibold text-slate-900 tracking-tight tabular-nums">{stat.value}</p>
                  </div>
               </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>
    </div>
  );
}
