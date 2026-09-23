"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { SectionTabs, type SectionTab } from "@/components/dashboard/section-tabs";
import { useDashboardMode, type Mode } from "@/components/dashboard/dashboard-mode-context";
import { dashboardApi } from "@/lib/api-client";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ChevronRight, AlertCircle, Loader2 } from "lucide-react";

interface ModeOverviewClientProps {
  mode: Mode;
}

interface ProjectSummary {
  id: number;
  name: string;
  goal: string;
}

// Tab definitions per mode. Each tab routes to a real page (already
// present in the codebase) and appears in SectionTabs above the
// overview cards. The same `href` is reused in the per-card Open
// link at the bottom of each card.
const SEARCH_TABS: SectionTab[] = [
  { label: "Overview", href: "/dashboard", match: (p) => p === "/dashboard" },
  { label: "Site Explorer", href: "/seo/site-explorer" },
  { label: "Keyword Research", href: "/seo/keywords" },
  { label: "Rank Tracking", href: "/seo/rank-tracking" },
  { label: "AI Insights", href: "/ai-insights" },
  { label: "Action Center", href: "/action-center" },
];

const SOCIAL_TABS: SectionTab[] = [
  { label: "Overview", href: "/dashboard", match: (p) => p === "/dashboard" },
  { label: "Profile Analyzer", href: "/social/profile-analyzer" },
  { label: "Content Analytics", href: "/social/insights" },
  { label: "Growth Tracking", href: "/social/growth" },
  { label: "AI Insights", href: "/ai-insights" },
  { label: "Action Center", href: "/action-center" },
];

const COMBINED_TABS: SectionTab[] = [
  { label: "Overview", href: "/dashboard", match: (p) => p === "/dashboard" },
  { label: "Site Explorer", href: "/seo/site-explorer" },
  { label: "Keyword Research", href: "/seo/keywords" },
  { label: "Rank Tracking", href: "/seo/rank-tracking" },
  { label: "Profile Analyzer", href: "/social/profile-analyzer" },
  { label: "Content Analytics", href: "/social/insights" },
  { label: "Growth Tracking", href: "/social/growth" },
  { label: "AI Insights", href: "/ai-insights" },
  { label: "Action Center", href: "/action-center" },
  { label: "Traffic", href: "/analytics/traffic" },
  { label: "Custom Reports", href: "/analytics/custom" },
];

const TABS_BY_MODE: Record<Mode, SectionTab[]> = {
  search: SEARCH_TABS,
  social: SOCIAL_TABS,
  combined: COMBINED_TABS,
};

const MODE_LABELS: Record<Mode, string> = {
  search: "Search",
  social: "Social",
  combined: "Combined",
};

// Tab descriptions — short, written to match the existing dashboard
// page tone. The descriptions are static strings; live data comes
// when the user clicks through to the dedicated page.
const TAB_DESCRIPTIONS: Record<string, string> = {
  "/dashboard": "Daily snapshot of clicks, impressions, and AI insights for the active project.",
  "/seo/site-explorer": "Audit the live site: meta tags, headers, indexing, and speed.",
  "/seo/keywords": "Discover keyword opportunities and track positions over time.",
  "/seo/rank-tracking": "Positions from Google Search Console, not a daily rank tracker.",
  "/social/profile-analyzer": "Reach, engagement, and audience growth for the connected social profile.",
  "/social/insights": "Post-level performance and audience interaction patterns.",
  "/social/growth": "Follower growth, content cadence, and engagement velocity.",
  "/ai-insights": "AI-generated recommendations ranked by impact and recency.",
  "/action-center": "Tasks the system thinks you should do next.",
  "/analytics/traffic": "Time-series of traffic, engagement, and source mix.",
  "/analytics/custom": "Build ad-hoc reports from any project metric.",
};

export function ModeOverviewClient({ mode }: ModeOverviewClientProps) {
  // Sync the URL-derived mode with the context so other consumers
  // (sidebar, topbar) see the same value. The proxy already wrote
  // the cookie before this page rendered; this is belt-and-braces.
  const { setMode } = useDashboardMode();
  useEffect(() => {
    void setMode(mode);
  }, [mode, setMode]);

  const tabs = TABS_BY_MODE[mode];

  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [activeProject, setActiveProject] = useState<ProjectSummary | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await dashboardApi.getProjects();
        if (cancelled) return;
        const all = (res.data?.data ?? []) as ProjectSummary[];
        setProjects(all);

        if (all.length === 0) {
          setActiveProject(null);
          return;
        }

        let savedId = 0;
        try {
          savedId = parseInt(
            window.localStorage.getItem("dmtool_active_project_id") ?? "0",
            10
          );
        } catch {
          savedId = 0;
        }
        const saved = all.find((p) => p.id === savedId) ?? all[all.length - 1];
        setActiveProject(saved);
        window.localStorage.setItem("dmtool_active_project_id", String(saved.id));
      } catch (err) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : "Failed to load projects");
        }
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const activeHref = useMemo(() => {
    if (!activeProject) return null;
    return (href: string) =>
      activeProject ? `${href}?project_id=${activeProject.id}` : href;
  }, [activeProject]);

  return (
    <div className="space-y-8 pb-12">
      <header className="space-y-1">
        <h1 className="text-3xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
          {MODE_LABELS[mode]} overview
        </h1>
        <p className="text-sm text-slate-500 dark:text-slate-400">
          Jump into any {MODE_LABELS[mode].toLowerCase()} tool below. Each
          card is a direct link to its full page.
        </p>
      </header>

      <SectionTabs tabs={tabs} />

      {isLoading ? (
        <Card>
          <CardContent className="p-6 flex items-center gap-3 text-sm text-slate-500">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading project context…
          </CardContent>
        </Card>
      ) : loadError ? (
        <Card>
          <CardContent className="p-6 flex items-center gap-3 text-sm text-rose-600">
            <AlertCircle className="h-4 w-4" />
            {loadError}
          </CardContent>
        </Card>
      ) : projects.length === 0 ? (
        <EmptyState mode={mode} />
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {tabs.map((tab) => (
            <TabCard
              key={tab.href}
              tab={tab}
              activeProject={activeProject}
              activeHref={activeHref}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface TabCardProps {
  tab: SectionTab;
  activeProject: ProjectSummary | null;
  activeHref: ((href: string) => string) | null;
}

function TabCard({ tab, activeProject, activeHref }: TabCardProps) {
  const description = TAB_DESCRIPTIONS[tab.href] ?? "Open this tool.";
  const href = activeHref ? activeHref(tab.href) : tab.href;
  return (
    <Card className="group hover:border-brand-300 dark:hover:border-brand-700 transition-colors">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-2">
          <CardTitle className="text-base font-semibold">
            {tab.label}
          </CardTitle>
          {activeProject ? (
            <Badge variant="secondary" className="text-[10px]">
              {activeProject.name}
            </Badge>
          ) : null}
        </div>
        <CardDescription className="text-xs leading-relaxed">
          {description}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex items-center justify-between pt-0">
        <span className="text-xs text-slate-400">Click to open</span>
        <Link
          href={href}
          prefetch
          className="inline-flex items-center justify-center gap-1 rounded-lg px-3 h-8 text-xs font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 transition-colors"
        >
          Open
          <ChevronRight className="h-3.5 w-3.5" />
        </Link>
      </CardContent>
    </Card>
  );
}

function EmptyState({ mode }: { mode: Mode }) {
  const projectHref = "/projects/create";
  return (
    <Card>
      <CardContent className="p-8 text-center space-y-3">
        <p className="text-base font-semibold text-slate-900 dark:text-slate-100">
          No projects yet
        </p>
        <p className="text-sm text-slate-500 max-w-md mx-auto">
          Connect a website or social profile to start using the{" "}
          {MODE_LABELS[mode].toLowerCase()} mode. Once connected, the
          overview cards fill in with project context.
        </p>
        <Link
          href={projectHref}
          className="inline-flex items-center justify-center h-10 rounded-xl px-4 py-2 bg-foreground text-background text-sm font-medium shadow hover:bg-foreground/90 transition-colors"
        >
          Create your first project
        </Link>
      </CardContent>
    </Card>
  );
}
