"use client";

import { motion } from "framer-motion";
import { 
  Calendar as CalendarIcon, ChevronLeft, ChevronRight, 
  Plus, Instagram, Twitter, Linkedin, 
  Facebook, Loader2, Sparkles, Filter, Clock, Edit2, Trash2
} from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useState, useEffect, useMemo, useCallback, useRef } from "react";
import { dashboardApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";

const playSuccessSound = () => {
  try {
    const ctx = new (window.AudioContext || (window as any).webkitAudioContext)();
    const osc = ctx.createOscillator();
    const gainNode = ctx.createGain();
    
    osc.type = 'sine';
    osc.frequency.setValueAtTime(523.25, ctx.currentTime);
    osc.frequency.exponentialRampToValueAtTime(1046.50, ctx.currentTime + 0.1);
    
    gainNode.gain.setValueAtTime(0, ctx.currentTime);
    gainNode.gain.linearRampToValueAtTime(0.3, ctx.currentTime + 0.05);
    gainNode.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.5);
    
    osc.connect(gainNode);
    gainNode.connect(ctx.destination);
    
    osc.start();
    osc.stop(ctx.currentTime + 0.5);
  } catch (e) {
    console.log("Audio play blocked", e);
  }
};

type ScheduledDraft = {
  title: string;
  platform: string;
  contentType: string;
  time: string;
  caption: string;
  location: string;
  music: string;
  musicPreviewUrl: string;
  tags: string;
  assetName: string;
  file: File | null;
  thumbnail: File | null;
  thumbnailName: string;
  thumbnailUrl?: string;
  assetUrl?: string;
  previewUrl?: string;
};

const parseTimeStr = (timeStr: string) => {
  const [h24Str, mStr] = (timeStr || "10:00").split(':');
  const h = parseInt(h24Str, 10);
  const isPM = h >= 12;
  const h12 = h % 12 || 12;
  return { 
    h12: h12.toString().padStart(2, '0'), 
    m: mStr?.padStart(2, '0') || '00', 
    ampm: isPM ? 'PM' : 'AM' 
  };
};

const buildTimeStr = (h12Str: string, mStr: string, ampm: string) => {
  let h = parseInt(h12Str, 10);
  if (ampm === 'PM' && h !== 12) h += 12;
  if (ampm === 'AM' && h === 12) h = 0;
  return `${h.toString().padStart(2, '0')}:${mStr.padStart(2, '0')}`;
};

export default function CalendarPage() {
  const [currentDate, setCurrentDate] = useState(new Date());
  const [posts, setPosts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [projects, setProjects] = useState<any[]>([]);
  const [project, setProject] = useState<any>(null);
  const [selectedDate, setSelectedDate] = useState<Date | null>(null);
  const [selectedDay, setSelectedDay] = useState<Date>(new Date());
  const [postFilter, setPostFilter] = useState<string>("All");
  const [drafts, setDrafts] = useState<ScheduledDraft[]>([{ title: "", platform: "instagram", contentType: "post", time: "10:00", caption: "", location: "", music: "", musicPreviewUrl: "", tags: "", assetName: "", file: null, thumbnail: null, thumbnailName: "" }]);
  const [saving, setSaving] = useState(false);
  const [showDayDetails, setShowDayDetails] = useState(true);
  const [editingTask, setEditingTask] = useState<any | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [currentTime, setCurrentTime] = useState<Date>(new Date());
  const previousPostsRef = useRef<any[]>([]);

  useEffect(() => {
    if (previousPostsRef.current.length > 0 && posts.length > 0) {
      posts.forEach(post => {
        const prev = previousPostsRef.current.find(p => p.id === post.id);
        if (prev && prev.publishStatus !== 'published' && post.publishStatus === 'published') {
          playSuccessSound();
          toast(`Successfully published: ${post.title}`, "success");
        }
      });
    }
    previousPostsRef.current = posts;
  }, [posts]);

  useEffect(() => {
    const interval = setInterval(() => setCurrentTime(new Date()), 1000);
    return () => clearInterval(interval);
  }, []);

  const fetchData = useCallback(async (targetProjectId?: number) => {
    try {
      const pRes = await dashboardApi.getProjects();
      const allProjects = pRes.data?.data || [];
      setProjects(allProjects);

      if (allProjects.length > 0) {
        const savedProjectId = typeof window !== "undefined" ? Number(window.localStorage.getItem("dmtool_active_project_id") || "0") : 0;
        let selectedProject = null;

        if (targetProjectId) {
          selectedProject = allProjects.find((p: any) => p.id === targetProjectId);
        }
        if (!selectedProject && savedProjectId) {
          selectedProject = allProjects.find((p: any) => p.id === savedProjectId);
        }
        if (!selectedProject) {
          selectedProject = allProjects[allProjects.length - 1];
        }

        if (selectedProject) {
          setProject(selectedProject);
          if (typeof window !== "undefined") {
            window.localStorage.setItem("dmtool_active_project_id", String(selectedProject.id));
          }

          const res = await dashboardApi.getCalendar(selectedProject.id);
          const tasks = res.data?.data || [];
          const mappedPosts = tasks.map((t: any) => {
            const parsedDate = t.due_date ? new Date(t.due_date) : null;
            const isValidDate = parsedDate && !Number.isNaN(parsedDate.getTime());
            const date = isValidDate
              ? parsedDate
              : new Date(t.year ?? t.created_at?.split('T')?.[0], (t.month ? t.month - 1 : 0), t.day || 1);

            return {
              id: t.id,
              day: isValidDate ? parsedDate.getDate() : (t.day ? t.day : date.getDate()),
              month: isValidDate ? parsedDate.getMonth() : (t.month ? t.month - 1 : date.getMonth()),
              year: isValidDate ? parsedDate.getFullYear() : (t.year ? t.year : date.getFullYear()),
              title: t.title,
              platform: t.platform || (t.source === "ai" ? "instagram" : "facebook"),
              contentType: t.content_type || "post",
              assetName: t.asset_name || "",
              assetUrl: t.asset_url || "",
              thumbnailUrl: t.thumbnail_url || "",
              caption: t.caption || "",
              location: t.location || "",
              music: t.music || "",
              tags: t.tags || "",
              dueDate: t.due_date,
              publishStatus: t.publish_status || "scheduled",
              publishError: t.publish_error || "",
              done: t.completed || t.done || t.publish_status === "published" || false,
              time: date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: true })
            };
          });
          setPosts(mappedPosts);
        }
      }
    } catch (err) {
      console.error("Failed to fetch calendar", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      fetchData();
    }, 15000);

    return () => window.clearInterval(interval);
  }, [fetchData]);

  const filteredPosts = useMemo(() => {
    if (postFilter === "All") return posts;
    return posts.filter(p => p.publishStatus === postFilter.toLowerCase());
  }, [posts, postFilter]);

  const getProjectPlatform = (project: any) => {
    if (!project) return "instagram";
    if (project.IGHandle || project.ig_handle) return "instagram";
    if (project.FBHandle || project.facebook_handle || project.fb_handle) return "facebook";
    if (project.twitter_handle) return "twitter";
    if (project.linkedin_handle) return "linkedin";
    return "instagram";
  };

  const openScheduler = (date?: Date) => {
    const initialDate = date ?? new Date(currentDate.getFullYear(), currentDate.getMonth(), 1);
    setSelectedDate(initialDate);
    setSelectedDay(initialDate);
    setDrafts([{ title: "", platform: getProjectPlatform(project), contentType: "post", time: "10:00", caption: "", location: "", music: "", musicPreviewUrl: "", tags: "", assetName: "", file: null, thumbnail: null, thumbnailName: "" }]);
    setEditingTask(null);
  };

  const openDayDetails = (date: Date) => {
    setSelectedDay(date);
    setShowDayDetails(true);
  };

  const taskListForSelectedDay = useMemo(() => {
    if (!selectedDay) return [];
    return filteredPosts.filter((post) => {
      if (typeof post.year === "number" && typeof post.month === "number" && typeof post.day === "number") {
        return post.year === selectedDay.getFullYear() && post.month === selectedDay.getMonth() && post.day === selectedDay.getDate();
      }
      const dueDate = new Date(post.dueDate);
      return dueDate.getFullYear() === selectedDay.getFullYear() && dueDate.getMonth() === selectedDay.getMonth() && dueDate.getDate() === selectedDay.getDate();
    });
  }, [selectedDay, filteredPosts]);

  const openEditTask = (task: any) => {
    setEditingTask(task);
    const taskDate = typeof task.year === "number" && typeof task.month === "number" && typeof task.day === "number"
      ? new Date(task.year, task.month, task.day)
      : new Date(task.dueDate);
    setSelectedDate(taskDate);
    setSelectedDay(taskDate);
    setDrafts([{ title: task.title, platform: getProjectPlatform(project), contentType: task.contentType, time: new Date(task.dueDate).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }), caption: task.caption, location: task.location, music: task.music, musicPreviewUrl: "", tags: task.tags || "", assetName: task.assetName, assetUrl: task.assetUrl, file: null, thumbnail: null, thumbnailName: task.thumbnailName || "", thumbnailUrl: task.thumbnailUrl || "" }]);
    setShowDayDetails(false);
  };

  const closeEdit = () => {
    setEditingTask(null);
    setDrafts([{ title: "", platform: "instagram", contentType: "post", time: "10:00", caption: "", location: "", music: "", musicPreviewUrl: "", tags: "", assetName: "", file: null, thumbnail: null, thumbnailName: "" }]);
  };

  const handleFileChange = (index: number, field: "file" | "thumbnail", file: File | null) => {
    setDrafts(currentDrafts => currentDrafts.map((draft, draftIndex) => {
      if (draftIndex !== index) return draft;
      if (field === "file") {
        return { 
          ...draft, 
          file, 
          assetName: file?.name ?? draft.assetName,
          previewUrl: file ? URL.createObjectURL(file) : undefined
        };
      }
      return { ...draft, thumbnail: file, thumbnailName: file?.name ?? draft.thumbnailName };
    }));
  };

  const updateDraft = (index: number, field: keyof ScheduledDraft, value: string) => {
    setDrafts(currentDrafts => currentDrafts.map((draft, draftIndex) => draftIndex === index ? { ...draft, [field]: value } : draft));
  };

  const addDraftRow = () => {
    setDrafts(currentDrafts => [...currentDrafts, { title: "", platform: getProjectPlatform(project), contentType: "post", time: "10:00", caption: "", location: "", music: "", musicPreviewUrl: "", tags: "", assetName: "", file: null, thumbnail: null, thumbnailName: "" }]);
  };

  const removeDraftRow = (index: number) => {
    setDrafts(currentDrafts => currentDrafts.length === 1 ? currentDrafts : currentDrafts.filter((_, draftIndex) => draftIndex !== index));
  };

  const saveScheduledContent = async () => {
    if (!project) {
      toast("No project selected", "error");
      return;
    }
    if (!selectedDate) {
      toast("No date selected", "error");
      return;
    }

    const validDrafts = drafts.filter(draft => draft.contentType === "story" || draft.caption.trim() || draft.file || draft.assetUrl);
    if (validDrafts.length === 0) {
      alert("Please provide content (file or caption) for your scheduled item.");
      return;
    }

	const missingFile = validDrafts.some(draft => !draft.file && !editingTask);
	if (missingFile) {
		alert("Select a local file for each scheduled item so it can be uploaded to social media.");
		return;
	}

	const hasInvalidFacebookPhotoStory = validDrafts.some(draft => {
		const isFb = draft.platform === "facebook";
		const isStory = draft.contentType === "story";
		let isPhoto = false;
		if (draft.file) {
			isPhoto = draft.file.type.startsWith("image/");
		} else if (draft.assetUrl) {
			isPhoto = !draft.assetUrl.match(/\.(mp4|webm|mov)$/i);
		}
		return isFb && isStory && isPhoto;
	});

	if (hasInvalidFacebookPhotoStory) {
		alert("Facebook Photo Stories are currently unsupported. Please switch the post type, use a video, or use an Instagram project instead.");
		return;
	}

    setSaving(true);
    try {
      for (const draft of validDrafts) {
        const [hours, minutes] = (draft.time || "10:00").split(":").map((value) => parseInt(value || "0", 10));
        const dueDate = new Date(selectedDate);
        dueDate.setHours(Number.isFinite(hours) ? hours : 10, Number.isFinite(minutes) ? minutes : 0, 0, 0);

        if (dueDate.getTime() <= new Date().getTime()) {
          toast(`Scheduled time cannot be in the past. Pick a future time.`, "error");
          setSaving(false);
          return;
        }

        const platform = getProjectPlatform(project);
        const formData = new FormData();
        formData.append("project_id", String(project.id));
        formData.append("title", draft.caption.trim().slice(0, 75) || "Scheduled Instagram Post");
        formData.append("platform", platform);
        formData.append("content_type", draft.contentType);
        formData.append("caption", draft.caption);
        formData.append("tags", draft.tags || "");
        formData.append("due_date", dueDate.toISOString());
        if (draft.file) {
          formData.append("file", draft.file);
        }

        if (editingTask) {
          await dashboardApi.updateCalendarEvent(editingTask.id, formData);
          setSuccessMessage("Changes saved successfully!");
        } else {
          await dashboardApi.createCalendarEvent(formData);
          setSuccessMessage("Post scheduled successfully!");
        }
      }

      setTimeout(() => setSuccessMessage(null), 3000);

      setSelectedDate(null);
      setEditingTask(null);
      setDrafts([{ title: "", platform: "instagram", contentType: "post", time: "10:00", caption: "", location: "", music: "", musicPreviewUrl: "", tags: "", assetName: "", file: null, thumbnail: null, thumbnailName: "" }]);
      await fetchData();
    } catch (err: any) {
      console.error("saveScheduledContent error:", err);
      const msg = err?.response?.data?.error?.message || err?.message || "Failed to save scheduled content.";
      toast(msg, "error");
    } finally {
      setSaving(false);
    }
  };

  const deleteTask = async (task: any) => {
    if (!window.confirm("Delete this scheduled post?")) return;
    try {
      await dashboardApi.deleteCalendarEvent(task.id);
      await fetchData();
      setShowDayDetails(true);
    } catch (err) {
      alert("Failed to delete scheduled content.");
    }
  };

  const retryTask = async (task: any) => {
    try {
      // Reset publish_status back to "scheduled" so the CalendarPublisher picks it up again
      const formData = new FormData();
      formData.append("project_id", String(task.projectId || project?.id));
      formData.append("title", task.title);
      formData.append("platform", task.platform);
      formData.append("content_type", task.contentType);
      formData.append("caption", task.caption || "");
      formData.append("tags", task.tags || "");
      // Schedule 2 minutes from now to give the worker time to pick it up
      const retryDate = new Date(Date.now() + 2 * 60 * 1000);
      formData.append("due_date", retryDate.toISOString());
      await dashboardApi.updateCalendarEvent(task.id, formData);
      toast("Post rescheduled — will retry in 2 minutes", "success");
      await fetchData();
    } catch (err) {
      toast("Failed to retry post", "error");
    }
  };

  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const daysInMonth = (year: number, month: number) => new Date(year, month + 1, 0).getDate();
  const startDay = new Date(currentDate.getFullYear(), currentDate.getMonth(), 1).getDay();
  
  const days = Array.from({ length: daysInMonth(currentDate.getFullYear(), currentDate.getMonth()) }, (_, i) => i + 1);
  const padding = Array.from({ length: startDay }, (_, i) => i);

  const prevMonth = () => setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() - 1));
  const nextMonth = () => setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() + 1));

  // Auto-refresh when a task is due
  useEffect(() => {
    if (!showDayDetails || !selectedDay) return;
    
    let needsRefresh = false;
    posts.forEach(p => {
      if (!p.done && p.dueDate) {
        const diff = new Date(p.dueDate).getTime() - currentTime.getTime();
        // If it crossed 0 in the last second
        if (diff <= 0 && diff > -1000) {
          needsRefresh = true;
        }
      }
    });

    if (needsRefresh) {
      // Refresh after a small delay to give the backend worker time to run and update DB
      setTimeout(() => fetchData(project?.id), 2500);
    }
  }, [currentTime, posts, showDayDetails, selectedDay, project?.id, fetchData]);

  const selectedDateLabel = selectedDate
    ? selectedDate.toLocaleDateString("default", { weekday: "long", month: "long", day: "numeric", year: "numeric" })
    : "";

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
          <div className="mt-2 flex flex-col sm:flex-row sm:items-center gap-3 text-sm text-slate-500">
            <span>Plan and schedule your social strategy visually.</span>
            {project && (
              <span className="rounded-full border border-slate-200 bg-slate-100 px-3 py-1 font-medium text-slate-700">
                Project: {project.name || `#${project.id}`}
              </span>
            )}
          </div>
          <div className="mt-3 sm:mt-0 flex items-center gap-3">
            <label htmlFor="calendar-project" className="text-sm font-semibold text-slate-500">Project</label>
            <select
              id="calendar-project"
              value={project?.id ?? ""}
              onChange={async (event) => {
                const projectId = Number(event.target.value);
                if (projectId) {
                  setLoading(true);
                  await fetchData(projectId);
                  setLoading(false);
                }
              }}
              className="rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm outline-none focus:border-slate-400"
            >
              {projects.map((proj) => (
                <option key={proj.id} value={proj.id}>{proj.name || `Project ${proj.id}`}</option>
              ))}
            </select>
          </div>
        </motion.div>
        
        <div className="flex items-center gap-3">
          <Button onClick={() => openScheduler(new Date())} className="rounded-xl bg-slate-900 text-white gap-2 h-11 px-6 font-semibold shadow-lg shadow-slate-900/10">
            <Plus className="w-4 h-4" /> New Post
          </Button>
        </div>
      </div>

      <Card className="border-border/50 rounded-3xl overflow-hidden shadow-xl shadow-slate-200/20 bg-white">
        <div className="p-8 flex items-center justify-between border-b border-slate-50 bg-slate-50/30">
          <div className="flex items-center gap-6">
            <div className="flex flex-col">
              <span className="text-[11px] font-bold uppercase tracking-widest text-brand-600 mb-1">
                Today is {new Date().toLocaleDateString('default', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })}
              </span>
              <h2 className="text-2xl font-bold tracking-tight min-w-50 text-slate-900">
                {currentDate.toLocaleString('default', { month: 'long', year: 'numeric' })}
              </h2>
            </div>
            <div className="flex items-center gap-1 bg-white border border-slate-200 p-1 rounded-xl shadow-sm">
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-lg" onClick={prevMonth}><ChevronLeft className="w-4 h-4" /></Button>
              <Button variant="ghost" className="h-8 px-3 rounded-lg text-xs font-bold uppercase tracking-widest text-slate-500 hover:text-slate-900" onClick={() => setCurrentDate(new Date())}>Today</Button>
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-lg" onClick={nextMonth}><ChevronRight className="w-4 h-4" /></Button>
            </div>
          </div>
          <div className="flex items-center gap-2">
             <input 
               type="month" 
               value={`${currentDate.getFullYear()}-${String(currentDate.getMonth() + 1).padStart(2, '0')}`}
               onChange={(e) => {
                 if(e.target.value) {
                   const [y, m] = e.target.value.split('-');
                   setCurrentDate(new Date(parseInt(y), parseInt(m) - 1, 1));
                 }
               }}
               className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 outline-none focus:border-slate-400 cursor-pointer shadow-sm hover:bg-slate-50 transition-all"
             />
          </div>
        </div>

        <div className="grid grid-cols-7 border-b border-slate-100 bg-slate-50/50">
          {["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map((day) => (
            <div key={day} className="py-4 text-center text-[10px] font-bold uppercase tracking-[0.2em] text-slate-400">
              {day}
            </div>
          ))}
        </div>

        <div className="grid grid-cols-7 min-h-200 bg-slate-50/20">
          {padding.map((p) => <div key={`p-${p}`} className="border-r border-b border-slate-50 bg-slate-50/10" />)}
          
          {days.map((day) => {
            const dayPosts = filteredPosts.filter((p) => {
              const hasExplicitDate = typeof p.day === 'number' && typeof p.month === 'number' && typeof p.year === 'number';
              if (hasExplicitDate) {
                return p.day === day && p.month === currentDate.getMonth() && p.year === currentDate.getFullYear();
              }
              const dueDate = p.dueDate ? new Date(p.dueDate) : null;
              return dueDate
                && dueDate.getFullYear() === currentDate.getFullYear()
                && dueDate.getMonth() === currentDate.getMonth()
                && dueDate.getDate() === day;
            });
            const isSelected = selectedDay && selectedDay.getDate() === day && selectedDay.getMonth() === currentDate.getMonth() && selectedDay.getFullYear() === currentDate.getFullYear();
            const cellDate = new Date(currentDate.getFullYear(), currentDate.getMonth(), day);
            const isPast = cellDate < today;
            return (
              <div
                key={day}
                className={`border-r border-b border-slate-100 p-3 min-h-35 transition-colors group relative cursor-pointer ${isSelected ? "bg-sky-600 text-white" : "hover:bg-slate-100"}`}
                onClick={() => openDayDetails(cellDate)}
              >
                <span className={`text-sm font-bold ${isSelected ? "text-white" : "text-slate-400 group-hover:text-slate-900"} transition-colors`}>{day}</span>
                {dayPosts.length > 0 && (
                  <span className={`absolute top-2 right-2 inline-flex h-6 min-w-6 items-center justify-center rounded-full px-2 text-[10px] font-bold uppercase tracking-[0.18em] ${isSelected ? "bg-white text-sky-600" : "bg-sky-600 text-white"}`}>
                    {dayPosts.length}
                  </span>
                )}

                {!isPast && (
                  <Button variant="ghost" size="icon" onClick={(event) => { event.stopPropagation(); openScheduler(cellDate); }} className="absolute bottom-2 right-2 rounded-full bg-brand-50 text-brand-600 hover:bg-brand-100">
                    <Plus className="w-3 h-3" />
                  </Button>
                )}
              </div>
            );
          })}
        </div>
      </Card>

      {selectedDay && (
        <div className="mt-6 grid gap-4 lg:grid-cols-[1fr,360px]">
          <div className="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-widest text-slate-400">Selected Date</p>
                <h2 className="text-xl font-bold text-slate-900 mt-1">{selectedDay.toLocaleDateString('default', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })}</h2>
              </div>
              <Button variant="outline" onClick={() => setShowDayDetails((current) => !current)}>
                {showDayDetails ? 'Hide scheduled posts' : `View ${taskListForSelectedDay.length} scheduled post${taskListForSelectedDay.length === 1 ? '' : 's'}`}
              </Button>
            </div>
            <p className="mt-4 text-sm text-slate-500">Click any day on the calendar to schedule new content or review posts already planned for that date.</p>
          </div>

          <div className="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
            <p className="text-xs font-semibold uppercase tracking-widest text-slate-400">Quick actions</p>
            <div className="mt-4 space-y-3">
              {selectedDay >= today && (
                <Button className="w-full" onClick={() => openScheduler(selectedDay)}>Schedule new item</Button>
              )}
              {taskListForSelectedDay.length > 0 && (
                <Button variant="outline" className="w-full" onClick={() => setShowDayDetails(true)}>
                  Manage scheduled content
                </Button>
              )}
            </div>
          </div>
        </div>
      )}

      {showDayDetails && selectedDay && (
        <div className="mt-6 rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between mb-4">
            <div>
              <h3 className="text-lg font-bold text-slate-900">Scheduled content for {selectedDay.toLocaleDateString('default', { month: 'long', day: 'numeric', year: 'numeric' })}</h3>
              <p className="text-sm text-slate-500">Review, edit, delete, or reschedule the content you have planned.</p>
            </div>
            <div className="flex flex-wrap items-center gap-3 text-sm text-slate-600">
              <span className="rounded-full bg-sky-100 px-3 py-1 text-sky-700 font-medium">{taskListForSelectedDay.length} item{taskListForSelectedDay.length === 1 ? '' : 's'}</span>
              
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="h-7 rounded-full border-slate-200 gap-1.5 text-xs font-semibold bg-white shadow-sm hover:bg-slate-50 transition-all">
                    <Filter className="w-3 h-3 text-slate-500" /> 
                    {postFilter === "All" ? "Filter" : postFilter}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-40 rounded-xl p-2 shadow-xl border-slate-200">
                  <DropdownMenuItem className="rounded-lg cursor-pointer text-xs font-medium" onClick={() => setPostFilter("All")}>All Posts</DropdownMenuItem>
                  <DropdownMenuItem className="rounded-lg cursor-pointer text-xs text-blue-600 font-medium" onClick={() => setPostFilter("Scheduled")}>Scheduled</DropdownMenuItem>
                  <DropdownMenuItem className="rounded-lg cursor-pointer text-xs text-emerald-600 font-medium" onClick={() => setPostFilter("Published")}>Published</DropdownMenuItem>
                  <DropdownMenuItem className="rounded-lg cursor-pointer text-xs text-rose-600 font-medium" onClick={() => setPostFilter("Failed")}>Failed</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>

            </div>
            <Button variant="ghost" onClick={() => setShowDayDetails(false)}>Close</Button>
          </div>
          {taskListForSelectedDay.length === 0 ? (
            <div className="rounded-3xl border border-dashed border-slate-200 p-8 text-center text-slate-500">No content scheduled for this date yet.</div>
          ) : (
            <div className="space-y-4">
              {taskListForSelectedDay.map((task) => {
                let statusLabel = task.done ? 'Published' : 'Scheduled';
                let statusClass = task.done ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700';

                // Map publish_status field to label and style
                if (task.publishStatus === 'published') {
                  statusLabel = 'Published';
                  statusClass = 'bg-emerald-100 text-emerald-700';
                } else if (task.publishStatus === 'failed') {
                  statusLabel = 'Failed';
                  statusClass = 'bg-rose-100 text-rose-700';
                } else if (task.publishStatus === 'scheduled') {
                  statusLabel = 'Scheduled';
                  statusClass = 'bg-amber-100 text-amber-700';
                }
                
                let countdownText = "";
                if (!task.done && task.dueDate) {
                  const diff = new Date(task.dueDate).getTime() - currentTime.getTime();
                  if (diff > 0) {
                    const hours = Math.floor(diff / (1000 * 60 * 60));
                    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
                    const seconds = Math.floor((diff % (1000 * 60)) / 1000);
                    if (hours > 0) countdownText = `in ${hours}h ${minutes}m`;
                    else if (minutes > 0) countdownText = `in ${minutes}m ${seconds}s`;
                    else countdownText = `in ${seconds}s`;
                  } else {
                    statusLabel = "Publishing...";
                    statusClass = "bg-sky-100 text-sky-700 animate-pulse";
                  }
                }

                return (
                  <div key={task.id} className="group relative rounded-2xl border border-slate-200/60 bg-white p-5 shadow-sm transition-all duration-200 hover:shadow-md hover:border-slate-300">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex gap-4">
                        <div className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl ${task.platform === 'instagram' ? 'bg-pink-50 text-pink-600' : 'bg-brand-50 text-brand-600'}`}>
                          {task.platform === 'instagram' ? <Instagram className="h-6 w-6" /> : <CalendarIcon className="h-6 w-6" />}
                        </div>
                        <div>
                          <div className="flex items-center gap-2 mb-1.5">
                            <span className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-semibold uppercase tracking-wider ring-1 ring-inset ${task.done ? 'bg-emerald-50 text-emerald-700 ring-emerald-600/20' : (statusLabel === 'Publishing...' ? 'bg-sky-50 text-sky-700 ring-sky-600/20 animate-pulse' : 'bg-amber-50 text-amber-700 ring-amber-600/20')}`}>
                              {statusLabel}
                            </span>
                            {countdownText && (
                              <span className="flex items-center gap-1.5 text-xs font-medium text-slate-500 bg-slate-100 px-2 py-0.5 rounded-md">
                                <Clock className="h-3 w-3 text-slate-400" />
                                {countdownText}
                              </span>
                            )}
                          </div>
                          <h4 className="text-base font-semibold text-slate-900 leading-tight">{task.title}</h4>
                          <div className="mt-2 flex items-center gap-2.5 text-xs font-medium text-slate-500">
                            <span className="capitalize text-slate-700 bg-slate-100 px-2 py-0.5 rounded-full">
                              {task.contentType}
                            </span>
                            <span className="text-slate-300">•</span>
                            <span>{task.time}</span>
                          </div>
                          {task.caption && <p className="mt-3 text-sm text-slate-600 line-clamp-2 leading-relaxed">{task.caption}</p>}
                          {task.publishStatus === 'failed' && task.publishError && (
                            <div className="mt-3 flex items-start gap-2 rounded-xl bg-rose-50 border border-rose-100 px-3 py-2">
                              <span className="text-xs text-rose-700 font-medium leading-relaxed">{task.publishError}</span>
                            </div>
                          )}
                        </div>
                      </div>
                      <div className="flex shrink-0 flex-col gap-1 transition-opacity">
                        {task.publishStatus === 'failed' && (
                          <Button size="sm" variant="outline" className="h-8 text-xs font-semibold text-rose-600 border-rose-200 hover:bg-rose-50" onClick={() => retryTask(task)}>
                            Retry
                          </Button>
                        )}
                        {task.publishStatus !== 'published' && (
                          <Button size="icon" variant="ghost" className="h-8 w-8 text-slate-400 hover:text-brand-600 hover:bg-brand-50" onClick={() => openEditTask(task)}>
                            <Edit2 className="h-4 w-4" />
                          </Button>
                        )}
                        <Button size="icon" variant="ghost" className="h-8 w-8 text-slate-400 hover:text-rose-600 hover:bg-rose-50" onClick={() => deleteTask(task)}>
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {selectedDate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4" onClick={() => setSelectedDate(null)}>
          <div className="w-full max-w-2xl rounded-3xl bg-white shadow-2xl shadow-slate-900/20 border border-slate-100 overflow-hidden" onClick={(event) => event.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-slate-100 px-6 py-4">
              <div>
                <p className="text-xs font-bold uppercase tracking-widest text-slate-400">Schedule Content</p>
                <h3 className="text-xl font-bold text-slate-900 mt-1">{selectedDateLabel}</h3>
              </div>
              <Button variant="ghost" onClick={() => setSelectedDate(null)} className="rounded-xl">
                Close
              </Button>
            </div>

            <div className="p-6 space-y-4 max-h-[70vh] overflow-y-auto">
              {drafts.map((draft, index) => (
                <div key={index} className="grid grid-cols-1 md:grid-cols-12 gap-3 rounded-2xl border border-slate-100 bg-slate-50/60 p-4">
                  <div className="md:col-span-2">
                    <label className="block text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-2">Post Type</label>
                    <select
                      value={draft.contentType}
                      onChange={(event) => updateDraft(index, "contentType", event.target.value)}
                      className="w-full rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm outline-none focus:border-slate-400"
                    >
                      <option value="post">Post</option>
                      <option value="story">Story</option>
                      <option value="reel">Reel</option>
                    </select>
                  </div>
                  <div className="md:col-span-5">
                    <label className="block text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-2">Content File</label>
                    <div className="flex gap-3 items-start">
                      {draft.assetUrl || draft.file ? (
                        <div className={`shrink-0 relative rounded-lg overflow-hidden border border-slate-200 bg-slate-100 flex items-center justify-center ${draft.contentType === "reel" ? "aspect-[9/16] h-14" : "aspect-square h-14"}`}>
                          {draft.file ? (
                            draft.file.type.startsWith("video/") ? (
                              <video src={draft.previewUrl} className="w-full h-full object-cover" muted />
                            ) : (
                              <img src={draft.previewUrl} alt="Preview" className="w-full h-full object-cover" />
                            )
                          ) : (
                            draft.assetUrl?.match(/\.(mp4|webm|mov)$/i) || draft.contentType === "reel" ? (
                              <video src={draft.assetUrl} className="w-full h-full object-cover" muted />
                            ) : (
                              <img src={draft.assetUrl} alt="Preview" className="w-full h-full object-cover" />
                            )
                          )}
                        </div>
                      ) : null}
                      <div className="flex-1 min-w-0">
                        <input
                          type="file"
                          accept={draft.contentType === "reel" ? "video/*" : "image/*,video/*"}
                          onChange={(event) => handleFileChange(index, "file", event.target.files?.[0] ?? null)}
                          className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm outline-none file:mr-3 file:rounded-lg file:border-0 file:bg-slate-900 file:px-3 file:py-1.5 file:text-xs file:font-bold file:text-white"
                        />
                        {!(draft.assetUrl || draft.file) && (
                          <p className="mt-2 text-[10px] text-slate-400 truncate">{draft.assetName || "Select an image or video from your computer"}</p>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="md:col-span-3">
                    <label className="block text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-2">Time</label>
                    {(() => {
                      const { h12, m, ampm } = parseTimeStr(draft.time);
                      return (
                        <div className="flex items-center gap-1.5">
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <div className="w-[62px] flex items-center justify-center rounded-xl border border-slate-200 bg-white px-1.5 py-2.5 text-[11px] font-semibold text-center outline-none focus:border-slate-400 cursor-pointer hover:border-slate-300 transition-colors shadow-sm">
                                {h12}
                              </div>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="center" className="min-w-0 w-16 max-h-[250px] overflow-y-auto rounded-xl p-1 shadow-xl">
                              {Array.from({length: 12}).map((_, i) => {
                                const val = (i + 1).toString().padStart(2, '0');
                                return (
                                  <DropdownMenuItem key={val} className="text-xs justify-center cursor-pointer rounded-lg" onClick={() => updateDraft(index, "time", buildTimeStr(val, m, ampm))}>
                                    {val}
                                  </DropdownMenuItem>
                                );
                              })}
                            </DropdownMenuContent>
                          </DropdownMenu>

                          <span className="text-slate-300 font-bold">:</span>
                          
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <div className="w-[62px] flex items-center justify-center rounded-xl border border-slate-200 bg-white px-1.5 py-2.5 text-[11px] font-semibold text-center outline-none focus:border-slate-400 cursor-pointer hover:border-slate-300 transition-colors shadow-sm">
                                {m}
                              </div>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="center" className="min-w-0 w-16 max-h-[250px] overflow-y-auto rounded-xl p-1 shadow-xl">
                              {Array.from({length: 60}).map((_, i) => {
                                const val = i.toString().padStart(2, '0');
                                return (
                                  <DropdownMenuItem key={val} className="text-xs justify-center cursor-pointer rounded-lg" onClick={() => updateDraft(index, "time", buildTimeStr(h12, val, ampm))}>
                                    {val}
                                  </DropdownMenuItem>
                                );
                              })}
                            </DropdownMenuContent>
                          </DropdownMenu>

                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <div className="w-[60px] ml-1 flex items-center justify-center rounded-xl border border-slate-200 bg-white px-1.5 py-2.5 text-[11px] font-semibold text-center outline-none focus:border-slate-400 cursor-pointer hover:border-slate-300 transition-colors shadow-sm">
                                {ampm}
                              </div>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="center" className="min-w-0 w-16 rounded-xl p-1 shadow-xl">
                              <DropdownMenuItem className="text-xs justify-center cursor-pointer rounded-lg" onClick={() => updateDraft(index, "time", buildTimeStr(h12, m, "AM"))}>AM</DropdownMenuItem>
                              <DropdownMenuItem className="text-xs justify-center cursor-pointer rounded-lg" onClick={() => updateDraft(index, "time", buildTimeStr(h12, m, "PM"))}>PM</DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </div>
                      );
                    })()}
                  </div>
                  <div className="md:col-span-2 flex items-end justify-end">
                    {drafts.length > 1 && (
                      <Button variant="ghost" onClick={() => removeDraftRow(index)} className="rounded-xl text-rose-600 hover:bg-rose-50 px-3">
                        Remove
                      </Button>
                    )}
                  </div>
                  {draft.contentType !== "story" && (
                    <>
                      <div className="md:col-span-12">
                        <div className="flex justify-between items-center mb-2">
                          <label className="block text-[10px] font-bold uppercase tracking-widest text-slate-400">Caption</label>
                          <span className={`text-[10px] font-bold ${draft.caption.length >= 2200 ? 'text-rose-500' : 'text-slate-400'}`}>
                            {draft.caption.length}/2200
                          </span>
                        </div>
                        <textarea
                          maxLength={2200}
                          value={draft.caption}
                          onChange={(event) => updateDraft(index, "caption", event.target.value)}
                          placeholder="Write the caption"
                          rows={4}
                          className="w-full min-h-[130px] rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm outline-none focus:border-slate-400 resize-none"
                        />
                      </div>
                      <div className="md:col-span-12">
                        <label className="block text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-2">Hashtags (Max 5)</label>
                        <input
                          type="text"
                          value={draft.tags}
                          onChange={(event) => {
                            const val = event.target.value;
                            const tagCount = val.split(/[\s,]+/).filter(t => t.trim() !== "").length;
                            // Allow typing if under limit, or if they are deleting characters
                            if (tagCount <= 5 || val.length < draft.tags.length) {
                              updateDraft(index, "tags", val);
                            }
                          }}
                          placeholder="e.g. #marketing #growth"
                          className="w-full rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm outline-none focus:border-slate-400"
                        />
                      </div>
                    </>
                  )}
                </div>
              ))}

              <Button variant="outline" onClick={addDraftRow} className="w-full rounded-2xl border-dashed border-slate-300 py-6 font-semibold">
                <Plus className="w-4 h-4 mr-2" /> Add another scheduled item
              </Button>

            </div>

            <div className="flex items-center justify-end gap-3 border-t border-slate-100 px-6 py-4 bg-slate-50/50">
              <Button variant="ghost" onClick={() => setSelectedDate(null)} className="rounded-xl">
                Cancel
              </Button>
              <Button onClick={saveScheduledContent} disabled={saving} className="rounded-xl bg-slate-900 text-white px-6">
                {saving ? "Saving..." : "Schedule Content"}
              </Button>
            </div>
          </div>
        </div>
      )}

      {successMessage && (
        <motion.div
          initial={{ opacity: 0, y: 50 }}
          animate={{ opacity: 1, y: 0 }}
          className="fixed bottom-6 right-6 z-[100] flex items-center gap-3 rounded-xl bg-slate-900 px-5 py-4 text-sm font-semibold text-white shadow-xl shadow-slate-900/20"
        >
          <Sparkles className="h-4 w-4 text-emerald-400" />
          {successMessage}
        </motion.div>
      )}
    </div>
  );
}
