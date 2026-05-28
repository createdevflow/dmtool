"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { dashboardApi } from "@/lib/api-client";
import { 
  Activity, LayoutDashboard, BrainCircuit, CheckSquare, Bell,
  FolderOpen, PlusSquare, Settings, 
  Search, BarChart3, Link as LinkIcon, TrendingUp,
  Share2, Users, Target, Repeat, MessageSquare, 
  PenTool, Image as ImageIcon, Crosshair, PieChart,
  FileText, Calendar, Zap, Blocks, CreditCard, ChevronDown, ChevronRight, Globe
} from "lucide-react";
import { cn } from "@/lib/utils";

// Define the comprehensive navigation structure with proper routing
const navGroups = [
  {
    title: "Overview",
    defaultOpen: true,
    items: [
      { name: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
      { name: "AI Insights", href: "/ai-insights", icon: BrainCircuit, badge: "New" },
      { name: "Action Center", href: "/action-center", icon: CheckSquare },
      { name: "Alerts & Notifications", href: "/alerts", icon: Bell },
    ]
  },
  {
    title: "Project Management",
    defaultOpen: false,
    items: [
      { name: "All Projects", href: "/projects", icon: FolderOpen },
      { name: "Create Project", href: "/projects/create", icon: PlusSquare },
      { name: "Project Settings", href: "/projects/settings", icon: Settings },
    ]
  },
  {
    title: "SEO Intelligence",
    defaultOpen: true,
    items: [
      { name: "Site Explorer", href: "/seo/site-explorer", icon: Globe },
      { name: "Keyword Research", href: "/seo/keywords", icon: Search },
      { name: "Backlink Analysis", href: "/seo/backlinks", icon: LinkIcon },
      { name: "Rank Tracking", href: "/seo/rank-tracking", icon: TrendingUp },
    ]
  },
  {
    title: "Social Media",
    defaultOpen: true,
    items: [
      { name: "Profile Analyzer", href: "/social/profile-analyzer", icon: Users },
      { name: "Content Analytics", href: "/social/insights", icon: BarChart3 },
      { name: "Growth Tracking", href: "/social/growth", icon: Target },
      { name: "Competitor Social", href: "/social/competitors", icon: Crosshair },
    ]
  },
  {
    title: "AI Engine",
    defaultOpen: false,
    items: [
      { name: "AI Chat Assistant", href: "/ai/chat", icon: MessageSquare },
      { name: "Content Generator", href: "/ai/content", icon: PenTool },
      { name: "Visual AI", href: "/ai/visual", icon: ImageIcon },
    ]
  },
  {
    title: "Analytics & Reports",
    defaultOpen: false,
    items: [
      { name: "Traffic Dashboard", href: "/analytics/traffic", icon: PieChart },
      { name: "Custom Reports", href: "/analytics/custom", icon: FileText },
    ]
  },
  {
    title: "System",
    defaultOpen: false,
    items: [
      { name: "Content Calendar", href: "/system/calendar", icon: Calendar },
      { name: "Automations", href: "/system/automations", icon: Zap },
      { name: "Integrations", href: "/integrations", icon: Blocks },
      { name: "Billing", href: "/billing", icon: CreditCard },
      { name: "Settings", href: "/settings", icon: Settings },
    ]
  }
];

export function Sidebar() {
  const pathname = usePathname();
  const [projectGoal, setProjectGoal] = useState<string>("both");
  
  // Keep track of which groups are open
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {};
    navGroups.forEach(g => {
      initial[g.title] = g.defaultOpen;
    });
    return initial;
  });

  useEffect(() => {
    const fetchProjectContext = async () => {
      try {
        const res = await dashboardApi.getProjects();
        if (res.data && res.data.length > 0) {
          // Use the most recent project as context
          const latest = res.data[res.data.length - 1];
          setProjectGoal(latest.goal || "both");
        }
      } catch (err) {
        console.error("Failed to fetch project context", err);
      }
    };
    fetchProjectContext();
  }, []);

  const toggleGroup = (title: string) => {
    setOpenGroups(prev => ({ ...prev, [title]: !prev[title] }));
  };

  // Filter groups based on project goal
  const filteredGroups = navGroups.filter(group => {
    if (group.title === "SEO Intelligence" && projectGoal === "social") return false;
    if (group.title === "Social Media" && projectGoal === "seo") return false;
    return true;
  });

  return (
    <div className="fixed inset-y-0 left-0 z-50 w-64 flex-col bg-background border-r border-border hidden lg:flex">
      
      {/* Brand Header */}
      <div className="flex h-16 shrink-0 items-center px-6 border-b border-border">
        <Link href="/" className="flex items-center gap-2 text-foreground font-bold text-lg tracking-tight group">
          <div className="bg-brand-600 rounded p-1.5 text-white group-hover:bg-brand-500 transition-colors shadow-sm">
            <Activity className="w-4 h-4" />
          </div>
          <span>DMTool</span>
        </Link>
      </div>

      {/* Scrollable Navigation Area */}
      <div className="flex flex-1 flex-col overflow-y-auto px-3 py-4 subtle-scrollbar space-y-4">
        
        {filteredGroups.map((group) => {
          const isOpen = openGroups[group.title];
          return (
            <div key={group.title} className="space-y-1">
              <button 
                onClick={() => toggleGroup(group.title)}
                className="w-full flex items-center justify-between px-3 py-1.5 text-xs font-semibold text-slate-400 uppercase tracking-wider hover:text-foreground transition-colors group outline-none"
              >
                <span>{group.title}</span>
                {isOpen ? (
                  <ChevronDown className="w-3.5 h-3.5 opacity-50 group-hover:opacity-100" />
                ) : (
                  <ChevronRight className="w-3.5 h-3.5 opacity-50 group-hover:opacity-100" />
                )}
              </button>
              
              {isOpen && (
                <nav className="space-y-0.5 text-sm font-medium">
                  {group.items.map((item) => {
                    const isActive = pathname === item.href;
                    return (
                      <Link
                        key={item.href}
                        href={item.href}
                        className={cn(
                          "flex items-center gap-3 rounded-md px-3 py-2 transition-all",
                          isActive
                            ? "bg-brand-50/50 text-brand-700 dark:bg-brand-500/10 dark:text-brand-400"
                            : "text-slate-500 hover:bg-slate-50 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800/50 dark:hover:text-white"
                        )}
                      >
                        <item.icon className={cn("w-4 h-4 shrink-0", isActive ? "text-brand-600 dark:text-brand-400" : "text-slate-400")} />
                        <span className="flex-1 truncate">{item.name}</span>
                        {item.badge && (
                          <span className="px-1.5 py-0.5 rounded text-[10px] bg-brand-100 text-brand-700 dark:bg-brand-900 dark:text-brand-300 font-bold tracking-wide">
                            {item.badge}
                          </span>
                        )}
                      </Link>
                    );
                  })}
                </nav>
              )}
            </div>
          );
        })}
        
      </div>
      
    </div>
  );
}
