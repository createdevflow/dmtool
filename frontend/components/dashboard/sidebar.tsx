"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  LayoutDashboard,
  FolderOpen,
  PlusSquare,
  Settings,
  Search,
  BarChart3,
  TrendingUp,
  Users,
  Target,
  MessageSquare,
  PenTool,
  ImageIcon,
  Link as LinkIcon,
  Crosshair,
  ChevronDown,
  ChevronRight,
  Globe,
  CreditCard,
  Blocks,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useDashboardMode, type Mode } from "./dashboard-mode-context";
import { ModeSwitcher } from "./mode-switcher";

// ────────────────────────────────────────────────────────────────────────
// Nav data — single source of truth for both the desktop Sidebar, the
// mobile SidebarNav (Sheet), and the CommandMenu. Exported so other
// components can derive their routes from the same shape (phase 8
// completeness check, plan 9 tests).
// ────────────────────────────────────────────────────────────────────────

// Mode visibility arrays. Single source of truth for nav inclusion.
const ALL: Mode[] = ["search", "social", "combined"];
const SEARCH_OR_COMBINED: Mode[] = ["search", "combined"];
const SOCIAL_OR_COMBINED: Mode[] = ["social", "combined"];

export type NavItem = {
  name: string;
  href: string;
  icon: typeof LayoutDashboard;
  // Items always render when their group is visible. modeExtras
  // narrows visibility to specific modes (when the containing group's
  // modes array is broader than the item's allowed subset).
  modeExtras?: Mode[];
};

export type NavGroup = {
  title: string;
  modes: Mode[];
  defaultOpen: boolean;
  items: NavItem[];
};

// Curated-subset design (the agreed spec):
//
//   SEARCHMODE  →  Dashboard (1) + SEO Intelligence (4) +
//                  AI Engine (3) + Account (3)    = 11 items, 4 groups
//   SOCIALMODE  →  Dashboard (1) + Social Media (4) +
//                  AI Engine (3) + Account (3)    = 11 items, 4 groups
//   COMBINED    →  Dashboard (3 — drop Backlinks) +
//                  Social (3 — drop Profile Discovery) +
//                  AI Engine (3) + Account (3)    = 13 items, 5 groups
//
// Items hidden from the sidebar move into per-mode overview
// SectionTabs drill-downs (built in a later phase):
//   * Backlink Analysis       → Site Explorer drill-down
//   * Profile Discovery       → Profile Analyzer drill-down
//   * AI Insights / Action Center / Alerts / Custom Reports
//                              → overview pages per mode
//
// Notes:
//   * Settings points at /projects/settings — that's the only Settings
//     page that exists; the Sidebar's prior hard-coded `/settings`
//     link 404'd for everyone. Phase 8 fix.
//   * Node 22 module-shape: nn uses unused-import warnings, so unused
//     icons pruned (BrainCircuit, FileText, etc).
export const navGroups: NavGroup[] = [
  {
    title: "Dashboard",
    modes: ALL,
    defaultOpen: true,
    items: [
      { name: "Overview", href: "/dashboard", icon: LayoutDashboard },
    ],
  },
  {
    title: "SEO Intelligence",
    modes: SEARCH_OR_COMBINED,
    defaultOpen: true,
    items: [
      { name: "Site Explorer", href: "/seo/site-explorer", icon: Globe },
      { name: "Keyword Research", href: "/seo/keywords", icon: Search },
      {
        name: "Backlink Analysis",
        href: "/seo/backlinks",
        icon: LinkIcon,
        // Search only: Backlink Analysis moves to Site Explorer
        // drill-down / SectionTabs for Combined users.
        modeExtras: ["search"],
      },
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
      {
        name: "Profile Discovery",
        href: "/social/competitors",
        icon: Crosshair,
        // Social only: moves to Profile Analyzer drill-down for Combined.
        modeExtras: ["social"],
      },
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
    title: "Account",
    modes: ALL,
    defaultOpen: false,
    items: [
      { name: "All Projects", href: "/projects", icon: FolderOpen },
      { name: "Create Project", href: "/projects/create", icon: PlusSquare },
      { name: "Project Settings", href: "/projects/settings", icon: Settings },
      { name: "Billing", href: "/billing", icon: CreditCard },
      { name: "Integrations", href: "/integrations", icon: Blocks },
    ],
  },
];

// ────────────────────────────────────────────────────────────────────────
// SidebarNav — renders the nav content only (no outer container). Both
// the fixed desktop Sidebar and the mobile-Sheet Sidebar wrap this.
//
// Tracks the open/closed state of each collapsible group internally;
// when the route changes (pathname), all groups briefly close on remount
// of the Sheet to keep the UX obvious. The Sheet won't normally unmount
// the inner content during a soft nav, so the visible "stay open"
// behaviour is intentional for the desktop Sidebar.
// ────────────────────────────────────────────────────────────────────────
export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  const { mode } = useDashboardMode();

  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {};
    for (const g of navGroups) {
      initial[g.title] = g.defaultOpen;
    }
    return initial;
  });

  const toggleGroup = (title: string) => {
    setOpenGroups((prev) => ({ ...prev, [title]: !prev[title] }));
  };

  // Two-level filter:
  //   1. Group must list this mode in `modes`.
  //   2. Item must pass: either no modeExtras (always-allow), or its
  //      modeExtras array contains this mode.
  const visibleGroups = navGroups
    .filter((group) => group.modes.includes(mode))
    .map((group) => ({
      ...group,
      items: group.items.filter(
        (item) => item.modeExtras === undefined || item.modeExtras.includes(mode)
      ),
    }))
    .filter((group) => group.items.length > 0);

  return (
    <div className="flex flex-1 flex-col overflow-y-auto subtle-scrollbar space-y-4">
      {visibleGroups.map((group) => {
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
                        onClick={onNavigate}
                        className={cn(
                          "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                          isActive
                            ? "bg-brand-50 text-brand-700"
                            : "text-slate-600 hover:text-slate-900 hover:bg-slate-50"
                        )}
                      >
                        <Icon className="w-4 h-4 shrink-0" />
                        <span className="flex-1 truncate">{item.name}</span>
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
  );
}

// ────────────────────────────────────────────────────────────────────────
// Sidebar — fixed-left desktop nav. Renders the brand header, mode
// switcher, and the same SidebarNav used by the mobile Sheet.
// ────────────────────────────────────────────────────────────────────────
export function Sidebar() {
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
      <div className="flex flex-1 flex-col px-3 py-4 overflow-hidden">
        <SidebarNav />
      </div>
    </div>
  );
}
