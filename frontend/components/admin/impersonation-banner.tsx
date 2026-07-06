"use client";

import { useEffect, useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { ShieldOff, ShieldAlert } from "lucide-react";
import { adminApi } from "@/lib/api-client";
import {
  COOKIE_IMPERSONATION_TOKEN,
  readImpersonationTarget,
  clearImpersonation,
} from "@/lib/auth-cookie";

type Target = { id: number; email: string; name: string };

// ImpersonationBanner is rendered on admin pages. It detects the
// dmtool_impersonation_token cookie (set when the admin hits
// "Impersonate" on /admin/users/:id) and surfaces a "you are
// impersonating X" strip with a Stop button.
//
// The check is cookie-based (not localStorage) so:
//   - the banner is server-renderable and shows on first paint;
//   - a full-page navigation while impersonating does not cause a
//     flash of "no banner" before the client mounts;
//   - the cookie path is consistent with the regular auth token
//     and the CSRF defense in proxy.ts.
//
// When the admin stops, the cookie is cleared and the admin's own
// dmtool_token cookie resumes carrying the real token on subsequent
// requests.
export function ImpersonationBanner() {
  const router = useRouter();
  const [target, setTarget] = useState<Target | null>(null);
  const [stopping, setStopping] = useState(false);
  const [, startTransition] = useTransition();

  useEffect(() => {
    // Re-read in case a sibling page set or cleared the cookie.
    setTarget(readImpersonationTarget());
  }, []);

  if (!target) return null;

  const stop = async () => {
    setStopping(true);
    try {
      await adminApi.stopImpersonation(target.id);
    } catch {
      // Even if the audit-write fails, the client-side cookie clear
      // is the primary effect; the next request will carry the
      // admin's real token.
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
      className="rounded-xl border-2 border-amber-300 bg-amber-50 p-4 flex items-center gap-3"
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
