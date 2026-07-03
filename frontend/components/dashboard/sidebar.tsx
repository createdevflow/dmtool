"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Activity, LayoutDashboard, BrainCircuit, CheckSquare, Bell,
  FolderOpen, PlusSquare, Settings,
  Search, BarChart3, Link as LinkIcon, TrendingUp,
  Share2, Users, Target, Repeat, MessageSquare,
  PenTool, ImageIcon, Crosshair, PieChart,
  FileText, Calendar, Zap, Blocks, CreditCard, ChevronDown, ChevronRight, Globe } from "lucide-react";
import { cn } from "@/lib/utils";
import { useDashboardMode, type Mode } from "./dashboard-mode-context";
import { ModeSwitcher } from "./mode-switcher";

// Modes a nav group belongs to. Items are filtered by the current mode.
// search + combined: SEO Intelligence.
// social + combined: Social Media.
// all three: the rest.
const ALL: Mode[] = ["search", "social", "combined"];
const SEARCH_OR_COMBINED: Mode[] = ["search", "combined"];
const SOCIAL_OR_COMBINED: Mode[] = ["social", "combined"];

// Define the comprehensive navigation structure with mode tags.
const navGroups = [
  {
    title: "Overview",
    modes: ALL,
    defaultOpen: true,
    items: [
      { name: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
      { name: "AI Insights", href: "/ai-insights", icon: BrainCircuit, badge: "New" },
      { name: "Action Center", href: "/action-center", icon: CheckSquare },
      { name: "Alerts & Notifications", href: "/alerts", icon: Bell },
    ],
  },
  {
    title: "Project Management",
    modes: ALL,
    defaultOpen: false,
    items: [
      { name: "All Projects", href: "/projects", icon: FolderOpen },
      { name: "Create Project", href: "/projects/create", icon: PlusSquare },
      { name: "Project Settings", href: "/projects/settings", icon: Settings },
    ],
  },
  {
    title: "SEO Intelligence",
    modes: SEARCH_OR_COMBINED,
    defaultOpen: true,
    items: [
      { name: "Site Explorer", href: "/seo/site-explorer", icon: Globe },
      { name: "Keyword Research", href: "/seo/keywords", icon: Search },
      { name: "Backlink Analysis", href: "/seo/backlinks", icon: LinkIcon },
      { name: "Rank Tracking", href: "/seo/rank-tracking", icon: TrendingUp },
    ],
  },
  {
    title: "Social Media",
    modes: SOCIAL_OR_COMBINED,
    defaultOpen: true,
    items: [
      { name: "Profile Analyzer", href: "/social/profile-analyzer", icon: Users },
      { name: "Content Analytics", href: "/social/insights", icon: BarChart3 },
      { name: "Growth Tracking", href: "/social/growth", icon: Target },
      { name: "Profile Discovery", href: "/social/competitors", icon: Crosshair },
    ],
  },
  {
    title: "AI Engine",
    modes: ALL,
    defaultOpen: false,
    items: [
      { name: "AI Chat Assistant", href: "/ai/chat", icon: MessageSquare },
      { name: "Content Generator", href: "/ai/content", icon: PenTool },
      { name: "Visual AI", href: "/ai/visual", icon: ImageIcon },
    ],
  },
  {
    title: "Analytics & Reports",
    modes: ALL,
    defaultOpen: false,
    items: [
      { name: "Traffic Dashboard", href: "/analytics/traffic", icon: PieChart },
      { name: "Custom Reports", href: "/analytics/custom", icon: FileText },
    ],
  },
  {
    title: "System",
    modes: ALL,
    defaultOpen: false,
    items: [
      { name: "Content Calendar", href: "/system/calendar", icon: Calendar },
      { name: "Automations", href: "/system/automations", icon: Zap },
      { name: "Integrations", href: "/integrations", icon: Blocks },
      { name: "Billing", href: "/billing", icon: CreditCard },
      { name: "Settings", href: "/settings", icon: Settings },
    ],
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const { mode } = useDashboardMode();

  // Track which groups are open. Initial state respects `defaultOpen`;
  // local toggle wins on user click.
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {};
    navGroups.forEach((g) => {
      initial[g.title] = g.defaultOpen;
    });
    return initial;
  });

  const toggleGroup = (title: string) => {
    setOpenGroups((prev) => ({ ...prev, [title]: !prev[title] }));
  };

  // Phase 4: filter by current mode. The legacy `hasSeoProject /
  // hasSocialProject` heuristic is replaced — modes are the source of
  // truth for what nav is visible.
  const filteredGroups = navGroups.filter((group) =>
    group.modes.includes(mode)
  );

  return (
    <div className="fixed inset-y-0 left-0 z-50 w-64 flex-col bg-background border-r border-border hidden lg:flex">

      {/* Brand Header */}
      <div className="flex h-16 shrink-0 items-center px-6 border-b border-border">
        <Link
          href="/"
          className="flex items-center gap-2 text-foreground font-bold text-lg tracking-tight group"
        >
          <div className="bg-brand-600 rounded p-1.5 text-white group-hover:bg-brand-500 transition-colors shadow-sm">
            <Activity className="w-4 h-4" />
          </div>
          <span>DMTool</span>
        </Link>
      </div>

      {/* Mode switcher sits below the brand header. Phase 4. */}
      <div className="px-4 py-3 border-b border-border">
        <ModeSwitcher />
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
                  <ChevronDown className="w-3.5 h-3.5" />
                ) : (
                  <ChevronRight className="w-3.5 h-3.5" />
                )}
              </button>
              {isOpen && (
                <ul className="space-y-0.5">
                  {group.items.map((item) => {
                    const Icon = item.icon;
                    const isActive = pathname === item.href;
                    return (
                      <li key={item.href}>
                        <Link
                          href={item.href}
                          className={cn(
                            "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                            isActive
                              ? "bg-brand-50 text-brand-700 dark:bg-brand-900/20 dark:text-brand-300"
                              : "text-slate-600 hover:text-slate-900 hover:bg-slate-50 dark:text-slate-400 dark:hover:text-slate-200 dark:hover:bg-slate-800/50"
                          )}
                        >
                          <Icon className="w-4 h-4 shrink-0" />
                          <span className="flex-1 truncate">{item.name}</span>
                          {item.badge && (
                            <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-brand-100 text-brand-700 dark:bg-brand-900/40 dark:text-brand-300">
                              {item.badge}
                            </span>
                          )}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
