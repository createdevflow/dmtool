"use client";

import * as React from "react";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";

// SidebarDrawerContext — tiny provider shared between the Topbar
// hamburger trigger and the MobileSidebar Sheet. Avoids prop-drilling
// the open state through the dashboard layout.
//
// Three surfaces read this:
//   1. MobileSidebar (Sheet wrapper, mounts in the layout) — reads
//      open + onOpenChange; renders the Sheet as either open or
//      closed, dismissible.
//   2. Topbar's hamburger button — opens the drawer.
//   3. SidebarNav's <Link onClick> (via prop) — closes the drawer
//      after navigation, so on mobile a tap doesn't leave the drawer
//      visible over the new page.

const COLLAPSED_KEY = "dmtool_nav_collapsed";

interface SidebarDrawerValue {
  open: boolean;
  setOpen: (v: boolean) => void;
  collapsed: boolean;
  setCollapsed: (v: boolean) => void;
}

const SidebarDrawerContext = createContext<SidebarDrawerValue | null>(null);

export function SidebarDrawerProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpenState] = useState(false);
  const [collapsed, setCollapsedState] = useState(false);
  const setOpen = useCallback((v: boolean) => setOpenState(v), []);
  const setCollapsed = useCallback((v: boolean) => {
    setCollapsedState(v);
    try {
      localStorage.setItem(COLLAPSED_KEY, v ? "1" : "0");
    } catch {}
  }, []);

  useEffect(() => {
    try {
      if (localStorage.getItem(COLLAPSED_KEY) === "1") setCollapsedState(true);
    } catch {}
  }, []);

  const value = useMemo(
    () => ({ open, setOpen, collapsed, setCollapsed }),
    [open, setOpen, collapsed, setCollapsed]
  );
  return <SidebarDrawerContext.Provider value={value}>{children}</SidebarDrawerContext.Provider>;
}

export function useSidebarDrawer(): SidebarDrawerValue {
  const ctx = useContext(SidebarDrawerContext);
  if (!ctx) {
    // Sensible default for callers that live outside the provider
    // (e.g. on public pages that haven't wrapped the layout). The
    // open action would no-op visually; we don't crash.
    return { open: false, setOpen: () => {}, collapsed: false, setCollapsed: () => {} };
  }
  return ctx;
}
