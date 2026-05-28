import { Sidebar } from "@/components/dashboard/sidebar";
import { Topbar } from "@/components/dashboard/topbar";
import { CommandMenu } from "@/components/dashboard/command-menu";
import { Toaster } from "@/components/ui/toaster";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="h-full bg-background min-h-screen font-sans">
      <Sidebar />
      <div className="lg:pl-64 flex flex-col min-h-screen">
        <Topbar />
        
        <main className="flex-1 p-6 lg:p-8 max-w-7xl w-full mx-auto relative">
          <CommandMenu />
          <Toaster />
          {children}
        </main>
      </div>
    </div>
  );
}
