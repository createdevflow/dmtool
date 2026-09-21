"use client";

import { useEffect, useState, useTransition } from "react";
import { usePathname, useRouter } from "next/navigation";
import { ShieldOff, ShieldAlert } from "lucide-react";
import { adminApi } from "@/lib/api-client";
import {
  COOKIE_IMPERSONATION_TOKEN,
  IMPERSONATION_EVENT,
  readImpersonationTarget,
  clearImpersonation,
} from "@/lib/auth-cookie";

type Target = { id: number; email: string; name: string };

// ImpersonationBanner lives in the dashboard layout so it shows on
// every gated page (admin AND the impersonated user's dashboard)
// while dmtool_impersonation_target is set.
//
// Cookie + dmtool:impersonation event (fired by set/clearImpersonation)
// so starting impersonation on /admin/users/:id updates a banner that
// was already mounted in the layout.
export function ImpersonationBanner() {
  const router = useRouter();
  const pathname = usePathname();
  const [target, setTarget] = useState<Target | null>(null);
  const [stopping, setStopping] = useState(false);
  const [, startTransition] = useTransition();

  useEffect(() => {
    const sync = () => setTarget(readImpersonationTarget());
    sync();
    window.addEventListener(IMPERSONATION_EVENT, sync);
    return () => window.removeEventListener(IMPERSONATION_EVENT, sync);
  }, [pathname]);

  if (!target) return null;

  const stop = async () => {
    setStopping(true);
    try {
      await adminApi.stopImpersonation(target.id);
    } catch {
      // Cookie clear is still the client session end if the audit
      // write fails. Stop itself should 200 with an impersonation JWT.
    }
    clearImpersonation();
    setTarget(null);
    setStopping(false);
    startTransition(() => router.refresh());
  };

  return (
    <div
      data-testid="impersonation-banner"
      data-impersonation-cookie={COOKIE_IMPERSONATION_TOKEN}
      className="rounded-xl border-2 border-amber-300 bg-amber-50 p-4 flex items-center gap-3 mb-6"
    >
      <ShieldAlert className="w-5 h-5 text-amber-700 shrink-0" />
      <div className="flex-1 text-sm">
        <div className="font-semibold text-amber-900">
          Impersonating {target.name} &lt;{target.email}&gt;
        </div>
        <div className="text-amber-800 text-xs mt-0.5">
          Requests in this tab run as this user. The admin&apos;s real
          session is paused; click Stop to return to your own
          session. The token auto-expires in 30 minutes.
        </div>
      </div>
      <button
        onClick={stop}
        disabled={stopping}
        className="px-3 py-1.5 rounded-lg bg-amber-700 text-white text-sm font-medium hover:bg-amber-800 disabled:opacity-50 inline-flex items-center gap-1"
      >
        <ShieldOff className="w-4 h-4" />
        {stopping ? "Stopping…" : "Stop impersonation"}
      </button>
    </div>
  );
}
