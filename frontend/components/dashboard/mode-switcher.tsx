"use client";

// ModeSwitcher — segmented 3-button control rendered inside the sidebar
// header. Reads/writes the active mode through the DashboardMode
// context; rendering is local (no global re-render of the sidebar tree
// on switch — React only re-renders consumers of the context value).

import { useDashboardMode, MODES, type Mode } from "./dashboard-mode-context";

const labels: Record<Mode, string> = {
  search: "Search",
  social: "Social",
  combined: "Combined",
};

const shortLabels: Record<Mode, string> = {
  search: "SEO",
  social: "Social",
  combined: "All",
};

export function ModeSwitcher() {
  const { mode, setMode } = useDashboardMode();

  return (
    <div
      className="flex items-center gap-1 rounded-lg bg-slate-100 dark:bg-slate-800 p-1"
      role="radiogroup"
      aria-label="Dashboard mode"
    >
      {MODES.map((m) => {
        const active = mode === m;
        return (
          <button
            key={m}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={`Switch to ${labels[m]} mode`}
            data-mode={m}
            onClick={() => {
              if (active) return;
              void setMode(m);
            }}
            className={
              "flex-1 rounded-md text-xs font-semibold py-1.5 px-2 transition-colors outline-none " +
              (active
                ? "bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 shadow-sm"
                : "text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200")
            }
          >
            {shortLabels[m]}
          </button>
        );
      })}
    </div>
  );
}
