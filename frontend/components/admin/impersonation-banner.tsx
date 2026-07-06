"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ShieldOff, ShieldAlert } from "lucide-react";
import { adminApi } from "@/lib/api-client";

type Stored = {
  token: string;
  target: { id: number; email: string; name: string };
  expires_at: number;
};

const STORAGE_KEY = "dmtool_admin_impersonation";

function readStored(): Stored | null {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    const v = JSON.parse(raw) as Stored;
    if (v.expires_at && v.expires_at < Date.now()) {
      localStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return v;
  } catch {
    return null;
  }
}

// ImpersonationBanner is rendered on admin pages. It detects the
// stored impersonation token and surfaces a "you are impersonating X"
// strip with a Stop button. When the admin stops, the stored token
// is removed and the admin's own auth cookie resumes carrying the
// real token on subsequent requests.
export function ImpersonationBanner() {
  const router = useRouter();
  const [stored, setStored] = useState<Stored | null>(null);
  const [stopping, setStopping] = useState(false);

  useEffect(() => {
    setStored(readStored());
    // Re-read on focus in case a child page wrote/cleared it.
    const onFocus = () => setStored(readStored());
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, []);

  if (!stored) return null;

  const stop = async () => {
    setStopping(true);
    try {
      await adminApi.stopImpersonation(stored.target.id);
    } catch {
      // Even if the audit-write fails, the client-side swap is the
      // primary effect; clear local state and let the next request
      // carry the admin's real token.
    }
    localStorage.removeItem(STORAGE_KEY);
    setStored(null);
    setStopping(false);
    router.refresh();
  };

  return (
    <div className="rounded-xl border-2 border-amber-300 bg-amber-50 p-4 flex items-center gap-3">
      <ShieldAlert className="w-5 h-5 text-amber-700 shrink-0" />
      <div className="flex-1 text-sm">
        <div className="font-semibold text-amber-900">
          Impersonating {stored.target.name} &lt;{stored.target.email}&gt;
        </div>
        <div className="text-amber-800 text-xs mt-0.5">
          Requests in this tab run as this user. The admin's real
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
