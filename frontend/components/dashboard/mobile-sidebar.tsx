"use client";

import * as React from "react";
import Link from "next/link";
import { Activity } from "lucide-react";
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";
import { ModeSwitcher } from "./mode-switcher";
import { SidebarNav } from "./sidebar";
import { useSidebarDrawer } from "./sidebar-drawer-context";

// MobileSidebar — Sheet-anchored nav for `lg-` viewports.
//
// Mounted once in the dashboard layout, alongside <Sidebar>. The
// desktop <Sidebar> uses `hidden lg:flex` so it doesn't show on
// mobile; this is the mobile counterpart, anchored to the left edge
// and slid in when the Topbar hamburger clicks.
//
// On a successful nav, the SidebarNav's <Link onClick> fires
// onNavigate which closes the Sheet so the user sees the new page
// immediately, not the empty drawer over the new page.
export function MobileSidebar() {
  const { open, setOpen } = useSidebarDrawer();

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetContent
        side="left"
        widthClass="w-72 sm:w-80"
        // Match the desktop sidebar's outer chrome so the items
        // line up; we drop the right padding to let the nav scroll
        // content reach the edge.
        className="p-0"
      >
        <SheetTitle className="sr-only">Navigation</SheetTitle>
        <div className="flex h-full flex-col">
          {/* Brand header mirrors the desktop sidebar */}
          <div className="flex h-16 shrink-0 items-center px-6 border-b border-slate-100">
            <Link
              href="/"
              onClick={() => setOpen(false)}
              className="flex items-center gap-2 text-slate-900 font-bold text-lg tracking-tight"
            >
              <div className="bg-brand-600 rounded p-1.5 text-white shadow-sm">
                <Activity className="w-4 h-4" />
              </div>
              <span>DMTool</span>
            </Link>
          </div>

          {/* Mode switcher — same control as desktop, just reflowed
              into the drawer's narrower column */}
          <div className="px-4 py-3 border-b border-slate-100">
            <ModeSwitcher />
          </div>

          {/* The nav itself */}
          <div className="flex-1 px-3 py-4 overflow-y-auto subtle-scrollbar">
            <SidebarNav onNavigate={() => setOpen(false)} />
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
