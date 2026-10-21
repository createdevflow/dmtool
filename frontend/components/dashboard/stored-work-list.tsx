"use client";

import Link from "next/link";
import { CheckCircle2, Circle, ChevronRight } from "lucide-react";
import type { ActionWorkItem } from "@/lib/action-work";

export function StoredWorkList({
  work,
  emptyLabel = "No stored work yet.",
}: {
  work: ActionWorkItem[];
  emptyLabel?: string;
}) {
  if (!work.length) {
    return <p className="px-2 py-8 text-sm text-slate-400 text-center">{emptyLabel}</p>;
  }
  return (
    <div>
      {work.map((step) => (
        <Link
          key={step.id}
          href={step.href || "/seo/site-explorer"}
          className="flex items-center gap-4 rounded-xl px-2 py-3 hover:bg-slate-50 transition-colors group"
        >
          {step.done ? (
            <CheckCircle2 className="w-5 h-5 text-emerald-500 shrink-0" />
          ) : (
            <Circle className="w-5 h-5 text-slate-300 shrink-0" />
          )}
          <div className="min-w-0 flex-1">
            <p className={`text-sm font-semibold tracking-tight ${step.done ? "text-slate-400" : "text-slate-900"}`}>
              {step.title}
            </p>
            {step.detail && (
              <p className="text-xs text-slate-400 mt-0.5 truncate">{step.detail}</p>
            )}
          </div>
          <ChevronRight className="w-4 h-4 text-slate-300 group-hover:text-slate-500 shrink-0" />
        </Link>
      ))}
    </div>
  );
}
