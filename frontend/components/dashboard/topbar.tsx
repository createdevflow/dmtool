"use client";

import * as React from "react";
import { Bell, Search, Moon, Sun, Sparkles, Menu } from "lucide-react";
import { useTheme } from "next-themes";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { 
  DropdownMenu, 
  DropdownMenuContent, 
  DropdownMenuItem, 
  DropdownMenuLabel, 
  DropdownMenuSeparator, 
  DropdownMenuTrigger 
} from "@/components/ui/dropdown-menu";
import { LogOut, User, Settings as SettingsIcon } from "lucide-react";
import { dashboardApi } from "@/lib/api-client";
import { useRouter } from "next/navigation";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

export function Topbar() {
  const router = useRouter();
  const { setTheme, theme } = useTheme();
  const [user, setUser] = React.useState<any>(null);
  const [alertCount, setAlertCount] = React.useState(0);
  const [searchQuery, setSearchQuery] = React.useState("");

  React.useEffect(() => {
    if (typeof window !== "undefined") {
      const storedUser = localStorage.getItem("dmtool_user");
      if (storedUser) setUser(JSON.parse(storedUser));
    }

    const fetchAlertsAndProject = async () => {
      try {
        const res = await dashboardApi.getProjects();
        const allProjects = res.data?.data || [];
        if (allProjects.length > 0) {
          const latest = allProjects[allProjects.length - 1];
          const aRes = await dashboardApi.getAlerts(latest.id);
          const alerts = aRes.data?.data ?? [];
          // Count unread alerts
          const unread = Array.isArray(alerts) ? alerts.filter((a: any) => !a.is_read).length : 0;
          setAlertCount(unread);
        }
      } catch (err) {
        console.error(err);
      }
    };
    fetchAlertsAndProject();
  }, []);

  const handleLogout = () => {
    localStorage.removeItem("dmtool_token");
    localStorage.removeItem("dmtool_user");
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

  const userPlan = user?.plan || "Pro";

  return (
    <header className="sticky top-0 z-40 flex h-16 shrink-0 items-center border-b border-slate-100 bg-white/80 backdrop-blur-xl px-4 sm:px-6 lg:px-8">
      
      <Button variant="ghost" size="icon" className="mr-2 lg:hidden rounded-xl text-slate-500">
        <Menu className="w-5 h-5" />
      </Button>

      <div className="flex flex-1 gap-x-4 lg:gap-x-6 items-center">
        
        {/* Global Command Search */}
        <div className="relative flex flex-1 items-center max-w-xl group">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 group-focus-within:text-slate-900 transition-colors" />
          <Input 
            className="pl-10 pr-12 h-10 w-full bg-slate-50 border-transparent focus-visible:ring-1 focus-visible:ring-slate-200 focus-visible:bg-white transition-all rounded-xl text-[13px] placeholder:text-slate-400 font-medium" 
            placeholder="Search keywords, insights, or pages… (Enter to search)" 
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={handleSearch}
          />
          <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1 pointer-events-none">
             <kbd className="hidden sm:inline-flex h-5 items-center gap-1 rounded border border-slate-200 bg-white px-1.5 font-mono text-[9px] font-bold text-slate-400">
                ↵
             </kbd>
          </div>
        </div>

        <div className="flex items-center gap-x-3 sm:gap-x-5 ml-auto shrink-0">
          
          {/* Plan Badge */}
          <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-50 border border-slate-100">
             <Sparkles className="w-3.5 h-3.5 text-slate-400" />
             <span className="text-[11px] font-bold text-slate-600 tracking-tight">{userPlan.toUpperCase()} PLAN</span>
          </div>

          <div className="hidden md:block w-px h-6 bg-slate-100" />
          
          <div className="flex items-center gap-1 sm:gap-1.5">
            {/* Theme Toggle */}
            <Button 
              variant="ghost" 
              size="icon"
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
              className="rounded-lg text-slate-400 hover:text-slate-900 h-9 w-9"
            >
              <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
              <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
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
              <Bell className="h-4 w-4" />
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
                onClick={() => router.push('/settings')}
              >
                <User className="w-4 h-4 text-slate-400" />
                <span className="font-medium">Profile Details</span>
              </DropdownMenuItem>
              <DropdownMenuItem 
                className="rounded-lg gap-2 cursor-pointer py-2.5"
                onClick={() => router.push('/settings')}
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
