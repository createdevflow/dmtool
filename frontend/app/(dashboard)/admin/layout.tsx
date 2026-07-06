"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import Link from "next/link";
import { Shield, Users, CreditCard, ScrollText, ArrowLeft } from "lucide-react";
import { adminApi } from "@/lib/api-client";
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";

// AdminSubNav is the in-page tab strip for /admin/*. The full app
// sidebar doesn't expose admin items (admins are a tiny fraction of
// users) — this sub-nav is the entire admin chrome, plus a "back to
// app" link.
function AdminSubNav() {
  const pathname = usePathname();
  const links = [
    { href: "/admin", label: "Overview", icon: Shield, exact: true },
    { href: "/admin/users", label: "Users", icon: Users },
    { href: "/admin/plans", label: "Plans", icon: CreditCard },
    { href: "/admin/audit-log", label: "Audit Log", icon: ScrollText },
  ];
  return (
    <div className="flex items-center gap-1 mb-6 border-b border-slate-200 pb-3 flex-wrap">
      <Link
        href="/dashboard/combined"
        className="text-xs text-slate-500 hover:text-slate-900 inline-flex items-center gap-1 mr-3"
      >
        <ArrowLeft className="w-3 h-3" /> Back to app
      </Link>
      {links.map((l) => {
        const active = l.exact ? pathname === l.href : pathname.startsWith(l.href);
        const Icon = l.icon;
        return (
          <Link
            key={l.href}
            href={l.href}
            className={cn(
              "px-3 py-1.5 rounded-lg text-sm font-medium inline-flex items-center gap-2 transition-colors",
              active
                ? "bg-slate-900 text-white"
                : "text-slate-600 hover:bg-slate-100"
            )}
          >
            <Icon className="w-4 h-4" />
            {l.label}
          </Link>
        );
      })}
    </div>
  );
}

// AdminGuard is the client-side half of role enforcement. The backend
// already returns 403 for non-admins (RequireRole middleware), but
// doing a client-side check avoids the "load page → see 403" flash
// and makes the route behavior obvious. We ping /api/admin/stats on
// mount; if it 403s, redirect to /dashboard.
function AdminGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [state, setState] = useState<"loading" | "ok" | "denied">("loading");

  useEffect(() => {
    let cancelled = false;
    adminApi
      .stats()
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
  return (
    <div>
      <AdminSubNav />
      {children}
    </div>
  );
}

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <AdminGuard>{children}</AdminGuard>;
}
