"use client";

import * as React from "react";
import { Sparkles } from "lucide-react";
import { cn } from "@/lib/utils";

// PlanBadge — small badge that renders the user's current billing plan.
// Source of truth: GET /api/billing/me returns
//   { plan: { code, name, ... }, subscription: { status } }
// The Topbar fetches that on mount and passes `code` + `status` here.
//
// `code` is the canonical "free" | "pro_monthly" | "pro_yearly" |
// (admin-created custom codes) identifier. We render the uppercase
// display name (e.g. "FREE", "PRO · MONTHLY") since that matches what
// /pricing shows the user and keeps the badge short.
//
// `status` is the subscription row status — "active" | "trialing" |
// "canceled" | "past_due". When not active, we dim the badge and
// append the status so the user can see why their "Pro" entitlements
// may be paused without it looking identical to a healthy badge.

export type PlanBadgeProps = {
  code: string | null;
  status?: string | null;
  className?: string;
  onClick?: () => void;
};

// displayName converts "pro_monthly" → "PRO · MONTHLY" for the badge
// label. Falls back to uppercase passthrough so admin-created codes
// still render meaningfully without code changes.
function displayName(code: string | null): string {
  if (!code) return "FREE";
  const cleaned = code.replace(/_/g, " ");
  return cleaned.toUpperCase();
}

function statusAdornment(status?: string | null): string | null {
  if (!status) return null;
  if (status === "active" || status === "trialing") return null;
  // Show the non-trivial statuses so the user can tell at a glance
  // that their entitlements may be limited.
  if (status === "canceled") return "CANCELED";
  if (status === "past_due") return "PAST DUE";
  return status.toUpperCase();
}

export function PlanBadge({ code, status, className, onClick }: PlanBadgeProps) {
  const adornment = statusAdornment(status);
  const healthy = !adornment;
  const Tag = onClick ? "button" : "div";
  return (
    // Using a button when clickable (click → /billing) gives proper
    // keyboard / focus behaviour. When not clickable it's a plain div.
    <Tag
      type={onClick ? "button" : undefined}
      onClick={onClick}
      aria-label={code ? `Current plan: ${displayName(code)}` : "Current plan: unknown"}
      className={cn(
        "inline-flex items-center gap-2 px-3 py-1.5 rounded-lg border transition-colors",
        healthy
          ? "bg-slate-50 border-slate-100 text-slate-600"
          : "bg-amber-50 border-amber-200 text-amber-800",
        onClick && "hover:border-slate-300 hover:bg-white cursor-pointer",
        className
      )}
    >
      <Sparkles className="w-3.5 h-3.5" />
      <span className="text-[11px] font-bold tracking-tight">
        {displayName(code)}
        {adornment && (
          <>
            {" "}
            <span aria-hidden="true">·</span> {adornment}
          </>
        )}
        {!code && !adornment && (
          <>
            {" "}
            <span aria-hidden="true">·</span> UNKNOWN
          </>
        )}
      </span>
    </Tag>
  );
}
