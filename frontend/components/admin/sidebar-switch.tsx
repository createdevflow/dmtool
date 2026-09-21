"use client";

// SidebarSwitch — renders AdminSidebar when on /admin/* routes,
// otherwise renders the regular user Sidebar. Uses usePathname()
// which requires a Client Component boundary.

import { usePathname } from "next/navigation";
import { Sidebar } from "@/components/dashboard/sidebar";
import { AdminSidebar } from "./admin-sidebar";

export function SidebarSwitch() {
  const pathname = usePathname();
  const isAdmin = pathname.startsWith("/admin");
  return isAdmin ? <AdminSidebar /> : <Sidebar />;
}
