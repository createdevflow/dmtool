"use client";

import * as React from "react";
import { Bell, Search, Moon, Sun, Menu } from "lucide-react";
import { useTheme } from "@/components/theme-provider";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { LogOut, User, Settings as SettingsIcon } from "lucide-react";
import { dashboardApi, authApi, billingApi } from "@/lib/api-client";
import { readCookie, COOKIE_USER, clearAuth } from "@/lib/auth-cookie";
import { useRouter } from "next/navigation";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useSidebarDrawer } from "./sidebar-drawer-context";
import { PlanBadge } from "./plan-badge";
export function Topbar() {
  const router = useRouter();
  const { setTheme, theme } = useTheme();
  const { setOpen: setDrawerOpen } = useSidebarDrawer();
  const [user, setUser] = React.useState<any>(null);
  const [alertCount, setAlertCount] = React.useState(0);
  const [searchQuery, setSearchQuery] = React.useState("");
  const [planCode, setPlanCode] = React.useState<string | null>(null);
  const [planStatus, setPlanStatus] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (typeof window !== "undefined") {
      const storedUser = readCookie(COOKIE_USER);
      if (storedUser) {
        try {
          setUser(JSON.parse(storedUser));
        } catch {
          // Malformed cookie payload — fall back to no user. The proxy
          // has already gated this route; we just skip the user render.
        }
      }
    }

    const fetchAlertsAndProject = async () => {
      try {
        const res = await dashboardApi.getProjects();
        const allProjects = res.data?.data || [];
        if (allProjects.length > 0) {
          const latest = allProjects[allProjects.length - 1];
          const aRes = await dashboardApi.getAlerts(latest.id);
          const alerts = aRes.data?.data ?? [];
          const unread = Array.isArray(alerts) ? alerts.filter((a: any) => !a.is_read).length : 0;
          setAlertCount(unread);
        }
      } catch (err) {
        console.error(err);
      }
    };
    fetchAlertsAndProject();

    // Phase 8: fetch the real plan from /api/billing/me. Cookies
    // outlive a login but the plan can change server-side (admin
    // changed it, user upgraded, trial ended). The endpoint
    // resolves through entitlements so a user with no subscription
    // row still gets the default free plan.
    const fetchPlan = async () => {
      try {
        const r = await billingApi.me();
        const planCode = r.data?.data?.plan?.code ?? null;
        const subStatus = r.data?.data?.subscription?.status ?? null;
        setPlanCode(planCode);
        setPlanStatus(subStatus);
      } catch {
        // Anonymous user, network error, etc. — leave the badge in
        // its loading state (null code → renders "FREE" fallback).
      }
    };
    fetchPlan();
  }, []);

  const handleLogout = async () => {
    try {
      // Best-effort server-side logout (blacklists refresh token).
      await authApi.logout();
    } finally {
      // Always clear client-side cookies regardless of server response.
      clearAuth();
    }
    router.push("/login");
  };

  const handleSearch = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && searchQuery.trim()) {
      router.push(`/seo/keywords?q=${encodeURIComponent(searchQuery.trim())}`);
      setSearchQuery("");
    }
  };

  const userInitials = user?.name
    ? user.name.split(" ").map((n: string) => n[0]).join("").toUpperCase().slice(0, 2)
    : "AD";

  return (
    <header className="sticky top-0 z-40 flex h-16 shrink-0 items-center border-b border-slate-100 bg-white/80 backdrop-blur-xl px-4 sm:px-6 lg:px-8">

      <Button
        variant="ghost"
        size="icon"
        aria-label="Open navigation"
        onClick={() => setDrawerOpen(true)}
        className="mr-2 lg:hidden rounded-xl text-slate-500"
      >
        <Menu className="w-5 h-5" />
      </Button>

      <div className="flex flex-1 gap-x-4 lg:gap-x-6 items-center">
        <div className="ml-auto flex items-center gap-x-3 sm:gap-x-5 shrink-0">
          {/* Plan Badge — backed by GET /api/billing/me.
              Click navigates to /billing. */}
          <PlanBadge
            code={planCode}
            status={planStatus}
            onClick={() => router.push("/billing")}
            className="hidden sm:inline-flex"
          />

          <div className="hidden md:block w-px h-6 bg-slate-100" />

          <div className="flex items-center gap-1 sm:gap-1.5">
            {/* Theme Toggle */}
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
              className="rounded-lg text-slate-400 hover:text-slate-900 h-9 w-9"
            >
              <Sun className="h-4 h-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
              <Moon className="absolute h-4 h-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
            </Button>

            {/* Notifications */}
            <Button
              variant="ghost"
              size="icon"
              onClick={() => router.push('/alerts')}
              className="rounded-lg text-slate-400 hover:text-slate-900 h-9 w-9 relative"
            >
              {alertCount > 0 && (
                <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-rose-500 rounded-full ring-2 ring-white" />
              )}
              <Bell className="h-4 h-4" />
            </Button>
          </div>

          {/* User Profile */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="flex items-center gap-x-2 pl-2 group outline-none">
                <Avatar className="h-9 w-9 border border-slate-200 group-hover:border-slate-300 transition-all shadow-sm">
                  <AvatarFallback className="bg-slate-900 text-white font-bold text-[11px]">
                    {userInitials}
                  </AvatarFallback>
                </Avatar>
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56 rounded-xl p-2 shadow-xl border-slate-100">
              <DropdownMenuLabel className="px-2 py-1.5">
                <p className="text-sm font-semibold text-slate-900">{user?.name || "User"}</p>
                <p className="text-xs text-slate-400 font-medium truncate">{user?.email || ""}</p>
              </DropdownMenuLabel>
              <DropdownMenuSeparator className="bg-slate-50" />
              <DropdownMenuItem
                className="rounded-lg gap-2 cursor-pointer py-2.5"
                onClick={() => router.push('/projects/settings')}
              >
                <User className="w-4 h-4 text-slate-400" />
                <span className="font-medium">Profile Details</span>
              </DropdownMenuItem>
              <DropdownMenuItem
                className="rounded-lg gap-2 cursor-pointer py-2.5"
                onClick={() => router.push('/projects/settings')}
              >
                <SettingsIcon className="w-4 h-4 text-slate-400" />
                <span className="font-medium">Settings</span>
              </DropdownMenuItem>
              <DropdownMenuSeparator className="bg-slate-50" />
              <DropdownMenuItem
                onClick={handleLogout}
                className="rounded-lg gap-2 cursor-pointer py-2.5 text-rose-600 focus:text-rose-600 focus:bg-rose-50"
              >
                <LogOut className="w-4 h-4" />
                <span className="font-bold">Logout</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

        </div>
      </div>
    </header>
  );
}
