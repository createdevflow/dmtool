"use client";

// AdminMode — controls which metric groups are visible in the admin
// sidebar and can be read by future admin pages to filter charts/data.
// Persisted to localStorage so the preference survives page reloads.

import type { ReactNode } from "react";
import { createContext, useCallback, useContext, useMemo, useState } from "react";

export type AdminMode = "overview" | "seo" | "social";

export const ADMIN_MODES: AdminMode[] = ["overview", "seo", "social"];

const STORAGE_KEY = "dmtool_admin_mode";

interface AdminModeContextValue {
  adminMode: AdminMode;
  setAdminMode: (next: AdminMode) => void;
}

const AdminModeContext = createContext<AdminModeContextValue | null>(null);

function readStoredMode(): AdminMode {
  if (typeof window === "undefined") return "overview";
  try {
    const v = window.localStorage.getItem(STORAGE_KEY);
    if (v === "overview" || v === "seo" || v === "social") return v;
  } catch {
    // localStorage unavailable — SSR or private browsing.
  }
  return "overview";
}

export function AdminModeProvider({ children }: { children: ReactNode }) {
  const [adminMode, setAdminModeState] = useState<AdminMode>(readStoredMode);

  const setAdminMode = useCallback((next: AdminMode) => {
    setAdminModeState(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // Non-fatal.
    }
  }, []);

  const value = useMemo<AdminModeContextValue>(
    () => ({ adminMode, setAdminMode }),
    [adminMode, setAdminMode]
  );

  return (
    <AdminModeContext.Provider value={value}>
      {children}
    </AdminModeContext.Provider>
  );
}

export function useAdminMode(): AdminModeContextValue {
  const ctx = useContext(AdminModeContext);
  if (!ctx) {
    throw new Error("useAdminMode called outside AdminModeProvider");
  }
  return ctx;
}
