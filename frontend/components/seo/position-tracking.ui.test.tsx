import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("next/link", () => ({
  default: ({ href, children, ...rest }: { href: string; children: unknown }) => (
    <a href={href} {...rest}>{children as never}</a>
  ),
}));

vi.mock("@/components/ui/toaster", () => ({ toast: vi.fn() }));

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: unknown }) => <div data-testid="chart">{children as never}</div>,
  BarChart: ({ children }: { children: unknown }) => <div>{children as never}</div>,
  Bar: () => null,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
  Legend: () => null,
  Cell: () => null,
}));

const getPositionTracking = vi.fn();
vi.mock("@/lib/api-client", () => ({
  dashboardApi: {
    getPositionTracking: (...args: unknown[]) => getPositionTracking(...args),
    savePositionTracking: vi.fn(),
    refreshPositionTracking: vi.fn(),
  },
}));

import { PositionTrackingPanel } from "./position-tracking";

const payload = {
  limits: { plan_label: "Free", keywords: 10, websites: 1, competitors_per_campaign: 20, tags: false },
  campaign: {
    domain: "example.com",
    engine: "google",
    location_name: "United States",
    device: "desktop",
    snapshot_at: "2026-11-24T12:00:00Z",
    keywords: [{ keyword: "best coffee", tags: [] }, { keyword: "coffee grinder", tags: [] }],
    competitors: ["rival.com"],
  },
  gsc: { used_as_fallback: false },
  serp: { message: "Cached SERP snapshot." },
  reports: {
    overview: {
      visibility: 88,
      avg_position: 4.5,
      etv: null,
      ranked: 2,
      tracked: 2,
      honesty: "Visibility is the average of max(0, 100 − position) over ranked keywords in this snapshot. Estimated traffic is not computed.",
    },
    rankings: [
      { keyword: "best coffee", position: 2, url: "https://example.com/coffee", source: "serp", serp_features: ["featured_snippet"], gsc_impressions: 12, gsc_clicks: 3, tags: [] },
      { keyword: "coffee grinder", position: 7, url: "https://example.com/grinder", source: "serp", serp_features: [], gsc_impressions: 0, gsc_clicks: 0, tags: [] },
    ],
    landscape: [
      { keyword: "best coffee", target_position: 2, by_domain: { "rival.com": 5 } },
      { keyword: "coffee grinder", target_position: 7, by_domain: { "rival.com": 3 } },
    ],
    pages: [{ url: "https://example.com/coffee", keywords: 1, best_position: 2 }],
    tags: { locked: true, plan_needed: "Pro+", reason: "Included on Pro+ and above." },
    devices_locations: { locked: true, plan_needed: "Pro+", reason: "Included on Pro+ and above." },
    cannibalization: { locked: true, plan_needed: "Pro+", reason: "Included on Pro+ and above." },
    ai_search: { locked: true, plan_needed: "Pro+", reason: "ChatGPT tracking is not connected." },
    competitors_discovery: { domains: ["rival.com"], honesty: "Domains observed in this SERP snapshot." },
    featured_snippets: [{ keyword: "best coffee", owner: "example.com", url: "https://example.com/coffee", is_target: true }],
    share_of_voice: { locked: true },
  },
};

describe("Position Tracking Semrush chrome", () => {
  beforeEach(() => {
    getPositionTracking.mockReset();
    getPositionTracking.mockResolvedValue({ data: { data: payload } });
  });

  it("renders tool chrome, metric strip, and honest empty ETV", async () => {
    render(<PositionTrackingPanel projectId={1} defaultDomain="example.com" />);

    await waitFor(() => expect(screen.getByRole("heading", { name: "Position Tracking" })).toBeInTheDocument());

    expect(screen.getByRole("button", { name: /^Keywords 2$/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Competitors 1$/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Tags 0$/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Update/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Overview$/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Rankings Distribution$/ })).toBeInTheDocument();
    expect(screen.getByText("Visibility")).toBeInTheDocument();
    expect(screen.getByText("88.0%")).toBeInTheDocument();
    expect(screen.getByText("Estimated traffic")).toBeInTheDocument();
    expect(screen.getByText("Semrush ETV is not computed")).toBeInTheDocument();
    expect(screen.getByText("Rankings Overview")).toBeInTheDocument();
    expect(screen.getByText("Current snapshot · daily history is not stored yet")).toBeInTheDocument();
    expect(screen.queryByText("Marketing Properties")).not.toBeInTheDocument();
    expect(screen.queryByText("Add Source")).not.toBeInTheDocument();
  });

  it("switches to Rankings and keeps GSC impressions unlabeled as volume", async () => {
    const user = userEvent.setup();
    render(<PositionTrackingPanel projectId={1} defaultDomain="example.com" />);
    await waitFor(() => expect(screen.getByRole("button", { name: /^Rankings$/ })).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: /^Rankings$/ }));
    expect(screen.getByText("GSC impr.")).toBeInTheDocument();
    expect(screen.getByText(/GSC impressions are not search volume/)).toBeInTheDocument();
    expect(screen.queryByText("Search volume")).not.toBeInTheDocument();
  });
});
