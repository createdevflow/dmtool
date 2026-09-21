"use client";

// AdminSidebar — dedicated left sidebar for /admin/* routes. Replaces
// the user-facing Sidebar when the pathname starts with /admin.
//
// Features a mode switcher (Overview | SEO | Social) that controls
// which nav groups are visible, plus collapsible groups matching the
// existing sidebar pattern.

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  LayoutDashboard,
  Users,
  Globe,
  CreditCard,
  TrendingUp,
  ScrollText,
  Search,
  Share2,
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Shield,
  BarChart3,
  HeartPulse,
  Blocks,
  Package,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { readCookie, COOKIE_USER } from "@/lib/auth-cookie";
import { AdminModeProvider, useAdminMode, ADMIN_MODES, type AdminMode } from "./admin-mode-context";

// ────────────────────────────────────────────────────────────────────────
// Admin nav data — single source of truth
// ────────────────────────────────────────────────────────────────────────

type AdminNavItem = {
  name: string;
  href: string;
  icon: typeof LayoutDashboard;
  modes?: AdminMode[];
};

type AdminNavGroup = {
  title: string;
  modes: AdminMode[];
  defaultOpen: boolean;
  items: AdminNavItem[];
};

const adminNavGroups: AdminNavGroup[] = [
  // ── Platform (always visible) ──
  {
    title: "Platform",
    modes: ["overview", "seo", "social"],
    defaultOpen: true,
    items: [
      { name: "Platform Overview", href: "/admin", icon: LayoutDashboard },
      { name: "User Management", href: "/admin/users", icon: Users },
      { name: "Roles", href: "/admin/roles", icon: Shield },
      { name: "All Projects", href: "/admin/projects", icon: Globe },
    ],
  },
  // ── SEO Health (overview + seo tabs) ──
  {
    title: "SEO Health",
    modes: ["overview", "seo"],
    defaultOpen: false,
    items: [
      { name: "SEO Overview", href: "/admin/seo-health", icon: Search },
      { name: "SEO Projects", href: "/admin/projects?goal=seo", icon: Globe },
    ],
  },
  // ── Social Health (overview + social tabs) ──
  {
    title: "Social Health",
    modes: ["overview", "social"],
    defaultOpen: false,
    items: [
      { name: "Social Overview", href: "/admin/social-health", icon: Share2 },
      { name: "Social Projects", href: "/admin/projects?goal=social", icon: Globe },
    ],
  },
  // ── Finance (overview only) ──
  {
    title: "Finance",
    modes: ["overview"],
    defaultOpen: false,
    items: [
      { name: "Plans", href: "/admin/plans", icon: Package },
      { name: "Revenue", href: "/admin/revenue", icon: TrendingUp },
    ],
  },
  // ── System (always visible) ──
  {
    title: "System",
    modes: ["overview", "seo", "social"],
    defaultOpen: false,
    items: [
      { name: "Audit Log", href: "/admin/audit-log", icon: ScrollText },
      { name: "System Health", href: "/admin/system", icon: HeartPulse },
      { name: "Integrations", href: "/admin/integrations", icon: Blocks },
    ],
  },
];

const adminModeLabels: Record<AdminMode, string> = {
  overview: "Overview",
  seo: "SEO",
  social: "Social",
};

// ────────────────────────────────────────────────────────────────────────
// AdminModeSwitcher — segmented pill identical in shape to ModeSwitcher
// ────────────────────────────────────────────────────────────────────────

function AdminModeSwitcher() {
  const { adminMode, setAdminMode } = useAdminMode();

  return (
    <div
      className="flex items-center gap-1 rounded-lg bg-slate-100 dark:bg-slate-800 p-1"
      role="radiogroup"
      aria-label="Admin view mode"
    >
      {ADMIN_MODES.map((m) => {
        const active = adminMode === m;
        return (
          <button
            key={m}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={`Switch to ${adminModeLabels[m]} view`}
            onClick={() => setAdminMode(m)}
            className={cn(
              "flex-1 rounded-md text-xs font-semibold py-1.5 px-2 transition-colors outline-none",
              active
                ? "bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 shadow-sm"
                : "text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
            )}
          >
            {adminModeLabels[m]}
          </button>
        );
      })}
    </div>
  );
}

// ────────────────────────────────────────────────────────────────────────
// Helper: read current user from cookie
// ────────────────────────────────────────────────────────────────────────

function getAdminUser(): { name: string; email: string } | null {
  if (typeof document === "undefined") return null;
  const raw = readCookie(COOKIE_USER);
  if (!raw) return null;
  try {
    const user = JSON.parse(raw);
    if (user && typeof user === "object") {
      return { name: user.name ?? "Admin", email: user.email ?? "" };
    }
  } catch {
    // Malformed cookie.
  }
  return null;
}

// ────────────────────────────────────────────────────────────────────────
// AdminNavContent — the scrollable nav list
// ────────────────────────────────────────────────────────────────────────

function AdminNavContent() {
  const pathname = usePathname();
  const { adminMode } = useAdminMode();

  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {};
    for (const g of adminNavGroups) {
      initial[g.title] = g.defaultOpen;
    }
    return initial;
  });

  const toggleGroup = (title: string) => {
    setOpenGroups((prev) => ({ ...prev, [title]: !prev[title] }));
  };

  // Filter groups based on active admin mode
  const visibleGroups = adminNavGroups
    .filter((group) => group.modes.includes(adminMode))
    .map((group) => ({
      ...group,
      items: group.items.filter(
        (item) => item.modes === undefined || item.modes.includes(adminMode)
      ),
    }))
    .filter((group) => group.items.length > 0);

  const user = getAdminUser();

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
                  // Exact match for /admin, prefix match for everything else
                  const isActive =
                    item.href === "/admin"
                      ? pathname === "/admin"
                      : pathname.startsWith(item.href.split("?")[0]);
                  return (
                    <li key={item.href}>
                      <Link
                        href={item.href}
                        className={cn(
                          "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                          isActive
                            ? "bg-slate-900 text-white"
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

      {/* Bottom section — always visible */}
      <div className="mt-auto pt-4 border-t border-slate-100 space-y-3">
        <Link
          href="/dashboard/combined"
          className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium text-slate-500 hover:text-slate-900 hover:bg-slate-50 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Back to App</span>
        </Link>

        {user && (
          <div className="px-3 py-2">
            <p className="text-xs font-medium text-slate-900 truncate">
              {user.name}
            </p>
            <p className="text-[11px] text-slate-400 truncate">{user.email}</p>
          </div>
        )}
      </div>
    </div>
  );
}

// ────────────────────────────────────────────────────────────────────────
// AdminSidebar — the exported component. Wraps everything in
// AdminModeProvider so the mode switcher and nav share state.
// ────────────────────────────────────────────────────────────────────────

export function AdminSidebar() {
  return (
    <AdminModeProvider>
      <div className="fixed inset-y-0 left-0 z-50 w-64 flex-col bg-background border-r border-border hidden lg:flex">
        {/* Brand Header */}
        <div className="flex flex-col shrink-0 border-b border-border">
          <div className="flex h-16 items-center px-6">
            <Link
              href="/admin"
              className="flex items-center gap-2 text-foreground font-bold text-lg tracking-tight group"
            >
              <div className="bg-brand-600 rounded p-1.5 text-white group-hover:bg-brand-500 transition-colors shadow-sm">
                <Activity className="w-4 h-4" />
              </div>
              <span>DMTool</span>
            </Link>
            <span className="ml-2 inline-flex items-center gap-1 rounded-md bg-slate-900 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-white">
              <Shield className="w-3 h-3" />
              Admin
            </span>
          </div>

          {/* Admin Mode Switcher */}
          <div className="px-4 pb-3">
            <AdminModeSwitcher />
          </div>
        </div>

        {/* Scrollable Navigation Area */}
        <div className="flex flex-1 flex-col px-3 py-4 overflow-hidden">
          <AdminNavContent />
        </div>
      </div>
    </AdminModeProvider>
  );
}
