"use client";

// SectionTabs — thin wrapper around the existing Tabs primitive that
// gives drill-down sections a consistent visual style. Today it's a
// thin pass-through; future passes can add icons, badges, or counts.

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

export interface SectionTab {
  label: string;
  href: string;
  match?: (pathname: string) => boolean;
}

interface SectionTabsProps {
  tabs: SectionTab[];
  className?: string;
}

export function SectionTabs({ tabs, className }: SectionTabsProps) {
  const pathname = usePathname();

  return (
    <div
      className={cn(
        "border-b border-slate-100 dark:border-slate-800 mb-6",
        className
      )}
    >
      <nav
        className="flex items-center gap-1 -mb-px overflow-x-auto subtle-scrollbar"
        aria-label="Section tabs"
      >
        {tabs.map((tab) => {
          const active = tab.match
            ? tab.match(pathname)
            : pathname === tab.href || (tab.href !== "/" && pathname.startsWith(tab.href));
          return (
            <Link
              key={tab.href}
              href={tab.href}
              aria-current={active ? "page" : undefined}
              className={cn(
                "px-3 py-2.5 text-sm font-medium border-b-2 transition-colors whitespace-nowrap",
                active
                  ? "border-brand-600 text-brand-700 dark:text-brand-300 dark:border-brand-400"
                  : "border-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100"
              )}
            >
              {tab.label}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
