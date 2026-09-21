"use client";

// DashboardMode — single source of truth for the user's current dashboard
// mode within the browser. The persisted source of truth is the
// `users.dashboard_mode` column on the backend; on every login we set
// the dmtool_mode cookie (same value) so the Next proxy can do
// /dashboard → /dashboard/<mode> rewrites without an extra round trip.
//
// Provider lifecycle:
//   1. cookie value is the synchronously-available default
//   2. on mount we PATCH-style-fetch GET /api/auth/me, but use the
//      cached value as the rendered state until the fetch lands
//   3. any explicit setMode() writes the cookie and PATCHes the backend
import type { ReactNode } from "react";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import apiClient from "@/lib/api-client";

export type Mode = "search" | "social" | "combined";

export const MODES: Mode[] = ["search", "social", "combined"];

export const COOKIE_MODE = "dmtool_mode";

interface DashboardModeContextValue {
  mode: Mode;
  ready: boolean;
  setMode: (next: Mode) => Promise<void>;
}

const DashboardModeContext = createContext<DashboardModeContextValue | null>(null);

function writeCookieMode(value: Mode): void {
  if (typeof document === "undefined") return;
  // Same shape as other auth cookies: SameSite=Lax, Path=/. 30-day TTL.
  const expires = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toUTCString();
  document.cookie = `${COOKIE_MODE}=${value}; Path=/; SameSite=Lax; Expires=${expires}`;
}

export function DashboardModeProvider({
  children,
  initialMode = "combined",
}: {
  children: ReactNode;
  initialMode?: Mode;
}) {
  // Must match SSR. Reading document.cookie here used to default the
  // server to "combined" while the client used dmtool_mode (e.g. search),
  // which hydrated the mode switcher and nav groups with different trees.
  const [mode, setModeState] = useState<Mode>(initialMode);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    // Fetch the server's authoritative value; use it to overwrite the
    // cookie if the server disagrees (server wins).
    let cancelled = false;
    (async () => {
      try {
        const res = await apiClient.get("/auth/me");
        const serverMode = res.data?.data?.dashboard_mode;
        if (
          !cancelled &&
          (serverMode === "search" ||
            serverMode === "social" ||
            serverMode === "combined") &&
          serverMode !== mode
        ) {
          writeCookieMode(serverMode);
          setModeState(serverMode);
        }
      } catch {
        // Network failure or auth — keep the cookie value.
      } finally {
        if (!cancelled) setReady(true);
      }
    })();
    return () => {
      cancelled = true;
    };
    // We intentionally bind to initial mount only; later updates go
    // through setMode.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const setMode = useCallback(async (next: Mode) => {
    writeCookieMode(next);
    setModeState(next);
    try {
      await apiClient.patch("/auth/me", { dashboard_mode: next });
    } catch {
      // The cookie is local-truth; the backend will reconcile on the
      // next /me fetch. We don't roll back the cookie on PATCH error
      // because that would create flicker on flaky connections.
    }
  }, []);

  const value = useMemo<DashboardModeContextValue>(
    () => ({ mode, ready, setMode }),
    [mode, ready, setMode]
  );

  return (
    <DashboardModeContext.Provider value={value}>
      {children}
    </DashboardModeContext.Provider>
  );
}

export function useDashboardMode(): DashboardModeContextValue {
  const ctx = useContext(DashboardModeContext);
  if (!ctx) {
    throw new Error("useDashboardMode called outside DashboardModeProvider");
  }
  return ctx;
}
