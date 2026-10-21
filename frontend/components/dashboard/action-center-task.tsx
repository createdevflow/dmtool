"use client";

import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { isStoredTask } from "@/lib/action-work";

interface ActionCenterTaskProps {
  task: any;
  onToggle?: (id: number) => void;
}

export function ActionCenterTask({ task, onToggle }: ActionCenterTaskProps) {
  const stored = isStoredTask(task);
  const isDone = !!(task.isDone ?? task.completed ?? task.done);
  const description = typeof task.description === "string" ? task.description.trim() : "";

  return (
    <div className="flex items-start gap-4 group py-3 border-b border-slate-50 last:border-0">
      {stored ? (
        <button
          type="button"
          onClick={() => onToggle?.(task.id)}
          className={cn(
            "mt-0.5 h-5 w-5 rounded-md border-1.5 flex items-center justify-center transition-all shrink-0",
            isDone
              ? "bg-emerald-500 border-emerald-500 shadow-sm"
              : "border-slate-300 hover:border-slate-400 bg-white"
          )}
        >
          {isDone ? (
            <Check className="w-3 h-3 text-white stroke-[3]" />
          ) : (
            <div className="w-1.5 h-1.5 rounded-full bg-slate-200 opacity-0 group-hover:opacity-100 transition-opacity" />
          )}
        </button>
      ) : (
        <div className="mt-0.5 h-5 w-5" />
      )}
      <div className="flex-1 min-w-0">
        <p className={cn(
          "text-[13px] font-semibold leading-tight transition-colors",
          isDone ? "text-slate-400 line-through" : "text-slate-900"
        )}>
          {task.title}
        </p>
        {description && (
          <p className="text-[11px] text-slate-500 mt-1.5 font-medium line-clamp-1 opacity-80">{description}</p>
        )}
      </div>
      {stored && task.priority && (
        <div className={cn(
          "text-[9px] font-bold uppercase tracking-widest px-2 py-0.5 rounded self-center",
          task.priority === "High" ? "bg-rose-50 text-rose-500" :
          task.priority === "Medium" ? "bg-amber-50 text-amber-500" : "bg-slate-100 text-slate-400"
        )}>
          {task.priority}
        </div>
      )}
    </div>
  );
}
