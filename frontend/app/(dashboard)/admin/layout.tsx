"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";

// AdminGuard is the client-side half of role enforcement. The backend
// already returns 403 for non-admins (RequireRole middleware), but
// doing a client-side check avoids the "load page → see 403" flash
// and makes the route behavior obvious. We ping GET /admin/me (admin.access)
// on mount; if it 403s, redirect to /dashboard.
function AdminGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [state, setState] = useState<"loading" | "ok" | "denied">("loading");

  useEffect(() => {
    let cancelled = false;
    adminApi
      .me()
      .then(() => {
        if (!cancelled) setState("ok");
      })
      .catch((e) => {
        if (cancelled) return;
        if (e?.response?.status === 403) {
          setState("denied");
          toast("Admin access required", "error");
          router.replace("/dashboard/combined");
        } else if (e?.response?.status === 401) {
          router.replace("/login");
        } else {
          // Network error etc — show the page anyway; the backend
          // will 403 the actual data calls.
          setState("ok");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [router]);

  if (state === "denied") {
    return null;
  }
  if (state === "loading") {
    return (
      <div className="flex items-center justify-center min-h-[300px] text-slate-400">
        Checking admin access…
      </div>
    );
  }
  return <>{children}</>;
}

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <AdminGuard>{children}</AdminGuard>;
}
