"use client";

import Link from "next/link";
import { Bell, ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { StoredAlert } from "@/lib/alerts";

export function AlertList({
  alerts,
  emptyLabel = "No alerts from stored state.",
}: {
  alerts: StoredAlert[];
  emptyLabel?: string;
}) {
  if (!alerts.length) {
    return (
      <div className="py-16 text-center text-slate-400">
        <Bell className="w-10 h-10 mx-auto mb-4 opacity-20" />
        <p className="text-sm font-medium">{emptyLabel}</p>
      </div>
    );
  }
  return (
    <div>
      {alerts.map((alert) => (
        <Link
          key={alert.id}
          href={alert.href || "/dashboard"}
          className="flex items-start gap-4 rounded-xl px-2 py-4 hover:bg-slate-50 transition-colors group"
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <p className="text-sm font-semibold text-slate-900 tracking-tight">{alert.title}</p>
              {alert.severity && (
                <Badge
                  variant="outline"
                  className={`text-[10px] font-semibold uppercase tracking-wider ${
                    alert.severity === "high"
                      ? "text-rose-600 border-rose-100 bg-rose-50"
                      : "text-amber-600 border-amber-100 bg-amber-50"
                  }`}
                >
                  {alert.severity}
                </Badge>
              )}
            </div>
            {alert.body && (
              <p className="text-xs text-slate-400 mt-1 leading-relaxed">{alert.body}</p>
            )}
          </div>
          <ChevronRight className="w-4 h-4 text-slate-300 group-hover:text-slate-500 shrink-0 mt-1" />
        </Link>
      ))}
    </div>
  );
}
