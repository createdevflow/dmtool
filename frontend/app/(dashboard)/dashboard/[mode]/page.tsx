import { redirect } from "next/navigation";
import type { Metadata } from "next";
import { ModeOverviewClient } from "./overview-client";

type Mode = "search" | "social" | "combined";

const MODES: ReadonlySet<string> = new Set(["search", "social", "combined"]);

interface ModePageProps {
  params: { mode: string };
}

export function generateMetadata({ params }: ModePageProps): Metadata {
  const m = params.mode;
  const label = m === "search" ? "Search" : m === "social" ? "Social" : m === "combined" ? "Combined" : "Overview";
  return { title: `${label} · DMTool` };
}

export default function ModeOverviewPage({ params }: ModePageProps) {
  const raw = params.mode;
  if (!MODES.has(raw)) {
    // Unknown mode segment — redirect to the bare /dashboard so the
    // existing page renders. (The proxy already rewrites /dashboard
    // → /dashboard/<mode>, so this path is only reachable if someone
    // hand-types an invalid mode.)
    redirect("/dashboard");
  }
  const mode = raw as Mode;
  return <ModeOverviewClient mode={mode} />;
}
