import { cookies } from "next/headers";
import { Topbar } from "@/components/dashboard/topbar";
import { CommandMenu } from "@/components/dashboard/command-menu";
import { MobileSidebar } from "@/components/dashboard/mobile-sidebar";
import { SidebarDrawerProvider } from "@/components/dashboard/sidebar-drawer-context";
import { Toaster } from "@/components/ui/toaster";
import { TooltipProvider } from "@/components/ui/tooltip";
import { DashboardModeProvider } from "@/components/dashboard/dashboard-mode-context";
import { COOKIE_MODE } from "@/lib/auth-cookie";
import { SidebarSwitch } from "@/components/admin/sidebar-switch";

export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const jar = await cookies();
  const rawMode = jar.get(COOKIE_MODE)?.value;
  const initialMode =
    rawMode === "search" || rawMode === "social" || rawMode === "combined"
      ? rawMode
      : "combined";

  return (
    <DashboardModeProvider initialMode={initialMode}>
      <SidebarDrawerProvider>
        <TooltipProvider delayDuration={150}>
          <div className="h-full bg-background min-h-screen font-sans">
            <SidebarSwitch />
            <div className="lg:pl-64 flex flex-col min-h-screen">
              <Topbar />

              <main className="flex-1 p-6 lg:p-8 max-w-7xl w-full mx-auto relative">
                <CommandMenu />
                <MobileSidebar />
                <Toaster />
                {children}
              </main>
            </div>
          </div>
        </TooltipProvider>
      </SidebarDrawerProvider>
    </DashboardModeProvider>
  );
}
