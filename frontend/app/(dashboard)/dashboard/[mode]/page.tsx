import { redirect } from "next/navigation";
import type { Metadata } from "next";
import { ModeOverviewClient } from "./overview-client";

type Mode = "search" | "social" | "combined";

const MODES: ReadonlySet<string> = new Set(["search", "social", "combined"]);

interface ModePageProps {
  params: Promise<{ mode: string }>;
}

export async function generateMetadata({ params }: ModePageProps): Promise<Metadata> {
  const { mode: m } = await params;
  const label = m === "search" ? "Search" : m === "social" ? "Social" : m === "combined" ? "Combined" : "Overview";
  return { title: `${label} · DMTool` };
}

export default async function ModeOverviewPage({ params }: ModePageProps) {
  const { mode: raw } = await params;
  if (!MODES.has(raw)) {
    redirect("/dashboard");
  }
  const mode = raw as Mode;
  return <ModeOverviewClient mode={mode} />;
}
