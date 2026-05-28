"use client";

import { motion } from "framer-motion";
import { 
  Calendar as CalendarIcon, ChevronLeft, ChevronRight, 
  Plus, Instagram, Twitter, Linkedin, 
  Facebook, Loader2, Sparkles, Filter
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useState, useEffect } from "react";
import { dashboardApi } from "@/lib/api-client";

export default function CalendarPage() {
  const [currentDate, setCurrentDate] = useState(new Date());
  const [posts, setPosts] = useState<any[]>([]);
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
        const res = await dashboardApi.getCalendar(p.id);
        const tasks = res.data?.data || [];
        const mappedPosts = tasks.map((t: any) => {
          const date = new Date(t.due_date || t.created_at);
          return {
            id: t.id,
            day: date.getDate(),
            month: date.getMonth(),
            year: date.getFullYear(),
            title: t.title,
            platform: t.source === "ai" ? "instagram" : "facebook",
            time: date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
          };
        });
        setPosts(mappedPosts);
      }
    } catch (err) {
      console.error("Failed to fetch calendar", err);
    } finally {
      setLoading(false);
    }
  };

  const handleNewPost = async () => {
    if (!project) return;
    const title = prompt("Enter post description/task:");
    if (!title) return;
    const dueDays = prompt("How many days from now?", "1");
    
    try {
      const dueDate = new Date();
      dueDate.setDate(dueDate.getDate() + parseInt(dueDays || "0"));
      
      await dashboardApi.createCalendarEvent({
        project_id: project.id,
        title: title,
        due_date: dueDate.toISOString(),
      });
      fetchData();
    } catch (err) {
      alert("Failed to create post.");
    }
  };


  const daysInMonth = (year: number, month: number) => new Date(year, month + 1, 0).getDate();
  const startDay = new Date(currentDate.getFullYear(), currentDate.getMonth(), 1).getDay();
  
  const days = Array.from({ length: daysInMonth(currentDate.getFullYear(), currentDate.getMonth()) }, (_, i) => i + 1);
  const padding = Array.from({ length: startDay }, (_, i) => i);

  const prevMonth = () => setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() - 1));
  const nextMonth = () => setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() + 1));

  const getPlatformIcon = (platform: string) => {
    switch (platform.toLowerCase()) {
      case "instagram": return <Instagram className="w-3 h-3 text-pink-500" />;
      case "twitter": return <Twitter className="w-3 h-3 text-sky-500" />;
      case "linkedin": return <Linkedin className="w-3 h-3 text-blue-600" />;
      default: return <Facebook className="w-3 h-3 text-indigo-600" />;
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
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }}>
          <h1 className="text-3xl font-bold tracking-tight">Content Calendar</h1>
          <p className="text-slate-500 mt-1">Plan and schedule your social strategy visually.</p>
        </motion.div>
        
        <div className="flex items-center gap-3">
          <Button variant="outline" className="rounded-xl h-11 border-slate-200 gap-2 font-semibold">
            <Filter className="w-4 h-4" /> Filters
          </Button>
          <Button onClick={handleNewPost} className="rounded-xl bg-slate-900 text-white gap-2 h-11 px-6 font-semibold shadow-lg shadow-slate-900/10">
            <Plus className="w-4 h-4" /> New Post
          </Button>
        </div>
      </div>

      <Card className="border-border/50 rounded-3xl overflow-hidden shadow-xl shadow-slate-200/20 bg-white">
        <div className="p-8 flex items-center justify-between border-b border-slate-50 bg-slate-50/30">
          <div className="flex items-center gap-6">
            <h2 className="text-2xl font-bold tracking-tight min-w-[200px]">
              {currentDate.toLocaleString('default', { month: 'long', year: 'numeric' })}
            </h2>
            <div className="flex items-center gap-1 bg-white border border-slate-200 p-1 rounded-xl shadow-sm">
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-lg" onClick={prevMonth}><ChevronLeft className="w-4 h-4" /></Button>
              <Button variant="ghost" className="h-8 px-3 rounded-lg text-xs font-bold uppercase tracking-widest text-slate-500" onClick={() => setCurrentDate(new Date())}>Today</Button>
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-lg" onClick={nextMonth}><ChevronRight className="w-4 h-4" /></Button>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {["Week", "Month", "List"].map((v) => (
              <Button key={v} variant={v === 'Month' ? 'default' : 'ghost'} className={`rounded-lg h-9 px-4 text-xs font-bold uppercase tracking-widest ${v === 'Month' ? 'bg-slate-900' : 'text-slate-400'}`}>
                {v}
              </Button>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-7 border-b border-slate-100 bg-slate-50/50">
          {["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map((day) => (
            <div key={day} className="py-4 text-center text-[10px] font-bold uppercase tracking-[0.2em] text-slate-400">
              {day}
            </div>
          ))}
        </div>

        <div className="grid grid-cols-7 min-h-[800px] bg-slate-50/20">
          {padding.map((p) => <div key={`p-${p}`} className="border-r border-b border-slate-50 bg-slate-50/10" />)}
          
          {days.map((day) => {
            const dayPosts = posts.filter(p => p.day === day && p.month === currentDate.getMonth() && p.year === currentDate.getFullYear());
            return (
              <div key={day} className="border-r border-b border-slate-100 p-3 min-h-[140px] hover:bg-white transition-colors group relative">
                <span className="text-sm font-bold text-slate-400 group-hover:text-slate-900 transition-colors">{day}</span>
                
                <div className="mt-2 space-y-1.5">
                  {dayPosts.map((post, pi) => (
                    <div key={pi} className="p-2 rounded-lg bg-white border border-slate-200 shadow-sm text-[10px] font-medium animate-in fade-in slide-in-from-top-1">
                      <div className="flex items-center justify-between mb-1">
                        {getPlatformIcon(post.platform)}
                        <span className="text-[9px] text-slate-400 uppercase font-bold">{post.time}</span>
                      </div>
                      <p className="truncate text-slate-700">{post.title}</p>
                    </div>
                  ))}
                </div>

                <Button variant="ghost" size="icon" onClick={() => handleNewPost()} className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity h-6 w-6 rounded-md bg-brand-50 text-brand-600">
                  <Plus className="w-3 h-3" />
                </Button>
              </div>
            );
          })}
        </div>
      </Card>
    </div>
  );
}
