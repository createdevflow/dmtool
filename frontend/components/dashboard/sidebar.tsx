"use client";

import { useEffect, useState, type ReactNode } from "react";
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
  Globe,
  Sparkles,
  Eye,
  GitCompare,
  Scale,
  Map as MapIcon,
  FileSearch,
  Quote,
  Unlink,
  Link as LinkIcon,
  ClipboardCheck,
  CreditCard,
  Blocks,
  Shield,
  Bell,
  Home,
  LineChart,
  ChevronsLeft,
  ChevronsRight,
  ListTodo,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { type Mode } from "./dashboard-mode-context";
import { readCookie, COOKIE_USER } from "@/lib/auth-cookie";
import { useSidebarDrawer } from "./sidebar-drawer-context";

const ALL: Mode[] = ["search", "social", "combined"];

export const RAIL_PX = 64;
export const PANEL_PX = 232;
export const SIDEBAR_PX = RAIL_PX + PANEL_PX;

export type NavItem = {
  name: string;
  href: string;
  icon?: typeof LayoutDashboard;
};

export type NavSection = {
  title?: string;
  items: NavItem[];
};

export type NavGroup = {
  id: string;
  title: string;
  icon: typeof LayoutDashboard;
  modes: Mode[];
  defaultOpen: boolean;
  defaultHref: string;
  adminOnly?: boolean;
  sections: NavSection[];
};

// Honesty sprint: Visual AI and Profile Discovery stay omitted.
// Semrush-only tools we do not have (Keyword Magic, Organic Rankings 2.0,
// Local, Ad, AI PR, Apps, Traffic & Market) stay omitted.
//
// phase8 nav_audit.js parses the navGroups literal below and checks every
// href against app/(dashboard) pages. Keep real page hrefs in this array.
export const navGroups: NavGroup[] = [
  {
    id: "home",
    title: "Home",
    icon: Home,
    modes: ALL,
    defaultOpen: true,
    defaultHref: "/dashboard",
    sections: [
      {
        items: [
          { name: "Overview", href: "/dashboard", icon: LayoutDashboard },
          { name: "Alerts", href: "/alerts", icon: Bell },
          { name: "Action Center", href: "/action-center", icon: ListTodo },
          { name: "AI Insights", href: "/ai-insights", icon: Sparkles },
        ],
      },
    ],
  },
  {
    id: "seo",
    title: "SEO",
    icon: LineChart,
    modes: ALL,
    defaultOpen: true,
    defaultHref: "/seo/site-audit",
    sections: [
      {
        title: "Site Performance",
        items: [
          { name: "Site Audit", href: "/seo/site-audit", icon: ClipboardCheck },
          { name: "Site Explorer", href: "/seo/site-explorer", icon: Globe },
          { name: "Domain Overview", href: "/seo/domain-overview", icon: Globe },
        ],
      },
      {
        title: "Rank tracking",
        items: [
          { name: "Position Tracking", href: "/seo/position-tracking", icon: TrendingUp },
        ],
      },
      {
        title: "Competitive Analysis",
        items: [
          { name: "Domain Compare", href: "/seo/domain-compare", icon: Scale },
          { name: "Keyword Gap", href: "/seo/keyword-gap", icon: GitCompare },
          { name: "Content Gap", href: "/seo/content-gap", icon: FileSearch },
          { name: "Citation Gap", href: "/seo/citation-gap", icon: Quote },
          { name: "Backlink Gap", href: "/seo/backlink-gap", icon: Unlink },
          { name: "Positioning Map", href: "/seo/positioning-map", icon: MapIcon },
        ],
      },
      {
        title: "Keyword Research",
        items: [
          { name: "Keyword Research", href: "/seo/keywords", icon: Search },
        ],
      },
      {
        title: "Links & AI",
        items: [
          { name: "Backlinks", href: "/seo/backlinks", icon: LinkIcon },
          { name: "AI Visibility", href: "/seo/ai-visibility", icon: Sparkles },
          { name: "AI Visibility Gap", href: "/seo/ai-visibility-gap", icon: Eye },
        ],
      },
    ],
  },
  {
    id: "social",
    title: "Social",
    icon: Users,
    modes: ALL,
    defaultOpen: true,
    defaultHref: "/social/profile-analyzer",
    sections: [
      {
        items: [
          { name: "Profile Analyzer", href: "/social/profile-analyzer", icon: Users },
          { name: "Content Analytics", href: "/social/insights", icon: BarChart3 },
          { name: "Growth Tracking", href: "/social/growth", icon: Target },
        ],
      },
    ],
  },
  {
    id: "ai",
    title: "AI",
    icon: Sparkles,
    modes: ALL,
    defaultOpen: true,
    defaultHref: "/ai/chat",
    sections: [
      {
        items: [
          { name: "AI Chat Assistant", href: "/ai/chat", icon: MessageSquare },
          { name: "Content Generator", href: "/ai/content", icon: PenTool },
        ],
      },
    ],
  },
  {
    id: "projects",
    title: "Projects",
    icon: FolderOpen,
    modes: ALL,
    defaultOpen: true,
    defaultHref: "/projects",
    sections: [
      {
        items: [
          { name: "All Projects", href: "/projects", icon: FolderOpen },
          { name: "Create Project", href: "/projects/create", icon: PlusSquare },
          { name: "Project Settings", href: "/projects/settings", icon: Settings },
          { name: "Billing", href: "/billing", icon: CreditCard },
          { name: "Integrations", href: "/integrations", icon: Blocks },
        ],
      },
    ],
  },
  {
    id: "admin",
    title: "Admin",
    icon: Shield,
    modes: ALL,
    defaultOpen: true,
    defaultHref: "/admin",
    adminOnly: true,
    sections: [
      {
        items: [
          { name: "Admin Panel", href: "/admin", icon: Shield },
        ],
      },
    ],
  },
];

export function pathRailId(pathname: string): string {
  if (pathname.startsWith("/admin")) return "admin";
  if (pathname.startsWith("/seo/")) return "seo";
  if (pathname.startsWith("/social/")) return "social";
  if (pathname.startsWith("/ai/")) return "ai";
  if (
    pathname.startsWith("/projects") ||
    pathname.startsWith("/billing") ||
    pathname.startsWith("/integrations")
  ) {
    return "projects";
  }
  return "home";
}

function getUserRole(): string | null {
  if (typeof document === "undefined") return null;
  const raw = readCookie(COOKIE_USER);
  if (!raw) return null;
  try {
    const user = JSON.parse(raw);
    return user?.role ?? null;
  } catch {
    return null;
  }
}

function visibleRails(isAdmin: boolean): NavGroup[] {
  return navGroups.filter((g) => !g.adminOnly || isAdmin);
}

export function DashboardShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { collapsed } = useSidebarDrawer();
  const adminRoute = pathname.startsWith("/admin");
  const pad = adminRoute
    ? "lg:pl-64"
    : collapsed
      ? "lg:pl-16"
      : "lg:pl-[296px]";
  return <div className={cn("flex flex-col min-h-screen", pad)}>{children}</div>;
}

export function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  return <NavChrome onNavigate={onNavigate} mobile />;
}

function NavChrome({
  onNavigate,
  mobile = false,
}: {
  onNavigate?: () => void;
  mobile?: boolean;
}) {
  const pathname = usePathname();
  const { collapsed, setCollapsed } = useSidebarDrawer();
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const isAdmin = mounted && getUserRole() === "admin";
  const rails = visibleRails(isAdmin);
  const pathId = pathRailId(pathname);
  const [pickedId, setPickedId] = useState(pathId);
  useEffect(() => {
    setPickedId(pathRailId(pathname));
  }, [pathname]);

  const activeId = rails.some((r) => r.id === pickedId) ? pickedId : rails[0]?.id;
  const active = rails.find((r) => r.id === activeId) || rails[0];
  const hidePanel = !mobile && collapsed;

  const pickRail = (id: string) => {
    setPickedId(id);
    if (!mobile && collapsed) setCollapsed(false);
  };

  return (
    <div className={cn("flex h-full", mobile ? "min-h-0" : "")}>
      <div
        className="shrink-0 flex flex-col items-center border-r border-[#e8eaed] bg-[#f4f5f7]"
        style={{ width: RAIL_PX }}
      >
        <Link
          href="/dashboard"
          onClick={onNavigate}
          aria-label="DMTool home"
          className="flex h-14 w-full items-center justify-center text-slate-800"
        >
          <span className="flex h-8 w-8 items-center justify-center bg-slate-900 text-white">
            <Activity className="w-4 h-4" />
          </span>
        </Link>
        <div className="flex-1 w-full overflow-y-auto subtle-scrollbar py-1 px-1.5 space-y-0.5">
          {rails.map((rail) => {
            const Icon = rail.icon;
            const on = rail.id === activeId;
            return (
              <button
                key={rail.id}
                type="button"
                onClick={() => pickRail(rail.id)}
                aria-current={on ? "page" : undefined}
                aria-label={rail.title}
                className={cn(
                  "w-full flex flex-col items-center gap-1 py-2 px-1 text-[10px] font-medium leading-tight",
                  on ? "bg-[#e8eaed] text-slate-900" : "text-slate-500 hover:bg-[#eceef1] hover:text-slate-800"
                )}
              >
                <Icon className="w-5 h-5 shrink-0" />
                <span className="text-center px-0.5">{rail.title}</span>
              </button>
            );
          })}
        </div>
        {!mobile ? (
          <button
            type="button"
            onClick={() => setCollapsed(!collapsed)}
            aria-label={collapsed ? "Expand navigation" : "Collapse navigation"}
            className="flex h-10 w-full items-center justify-center text-slate-500 hover:text-slate-800 hover:bg-[#eceef1]"
          >
            {collapsed ? <ChevronsRight className="w-4 h-4" /> : <ChevronsLeft className="w-4 h-4" />}
          </button>
        ) : null}
      </div>

      {!hidePanel && active ? (
        <div
          className="flex min-w-0 flex-col bg-white"
          style={{ width: PANEL_PX }}
        >
          <div className={cn("px-4 pt-4 pb-2", mobile && "pr-10")}>
            <h2 className="text-[15px] font-semibold text-slate-900">{active.title}</h2>
          </div>
          <div className="flex-1 overflow-y-auto subtle-scrollbar px-2 pb-4">
            {active.sections.map((section, i) => (
              <div key={`${active.id}-${section.title || i}`} className={section.title ? "mt-3" : "mt-1"}>
                {section.title ? (
                  <p className="px-2 pt-2 pb-1 text-[12px] text-slate-400">{section.title}</p>
                ) : null}
                <ul>
                  {section.items.map((item) => {
                    const isActive = pathname === item.href;
                    return (
                      <li key={item.href}>
                        <Link
                          href={item.href}
                          onClick={onNavigate}
                          className={cn(
                            "flex items-center px-2 py-[7px] text-[13.5px] leading-snug",
                            isActive
                              ? "bg-[#f2f3f5] text-slate-900 font-medium"
                              : "text-slate-700 hover:bg-[#f6f7f8]"
                          )}
                        >
                          {item.name}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function Sidebar() {
  const { collapsed } = useSidebarDrawer();
  return (
    <div
      className="fixed inset-y-0 left-0 z-50 flex-col border-r border-[#e8eaed] bg-white hidden lg:flex"
      style={{ width: collapsed ? RAIL_PX : SIDEBAR_PX }}
    >
      <NavChrome />
    </div>
  );
}
