"use client";

import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";
import { SidebarNav } from "./sidebar";
import { useSidebarDrawer } from "./sidebar-drawer-context";

// MobileSidebar — Sheet-anchored two-pane nav for `lg-` viewports.
// Same rail + tool list as desktop; ModeSwitcher stays off this chrome.
export function MobileSidebar() {
  const { open, setOpen } = useSidebarDrawer();

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetContent
        side="left"
        widthClass="w-[296px]"
        className="p-0 gap-0"
      >
        <SheetTitle className="sr-only">Navigation</SheetTitle>
        <div className="flex h-full min-h-0 flex-col">
          <SidebarNav onNavigate={() => setOpen(false)} />
        </div>
      </SheetContent>
    </Sheet>
  );
}
