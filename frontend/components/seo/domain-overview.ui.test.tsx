import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("@/components/ui/toaster", () => ({ toast: vi.fn() }));

const getDomainOverview = vi.fn();
vi.mock("@/lib/api-client", () => ({
  dashboardApi: {
    getDomainOverview: (...args: unknown[]) => getDomainOverview(...args),
  },
}));

import { DomainOverviewPanel } from "./domain-overview";

const empty = {
  data: {
    limits: { plan_label: "Free", requests_per_day: 10, growth: false, compare_countries: false, export_csv: true },
    used_today: 0,
    remaining: 10,
    available: false,
    configured: true,
    domain: "example.com",
    reports: {
      overview: {
        ai_visibility: { status: "not_implemented", label: "Not implemented" },
        authority_score: { status: "not_implemented", label: "Not implemented" },
        organic_traffic: { status: "empty" },
        paid_traffic: { status: "empty" },
        referring_domains: { status: "empty" },
        traffic_share: { status: "empty" },
        organic_keywords: { status: "empty" },
        paid_keywords: { status: "empty" },
        backlinks: { status: "empty" },
      },
      top_keywords: [],
      competitors: [],
      honesty: "Empty cells stay —.",
    },
  },
};

const filled = {
  data: {
    limits: { plan_label: "Free", requests_per_day: 10, growth: false, compare_countries: false, export_csv: true },
    used_today: 1,
    remaining: 9,
    available: true,
    configured: true,
    domain: "example.com",
    reports: {
      overview: {
        ai_visibility: { status: "not_implemented", label: "Not implemented" },
        authority_score: { status: "not_implemented", label: "Not implemented" },
        organic_traffic: { value: 1234.5, status: "ok", hint: "Labs ETV" },
        paid_traffic: { value: 10, status: "ok" },
        referring_domains: { value: 12, status: "ok" },
        traffic_share: { value: 60.6, status: "ok" },
        organic_keywords: { value: 40, status: "ok" },
        paid_keywords: { value: 3, status: "ok" },
        backlinks: { value: 99, status: "ok" },
        dfs_rank: { value: 371, status: "ok" },
      },
      top_keywords: [{ keyword: "buy widgets", volume: 2200, cpc: 1.5, kd: 44, position: 3, intent: "transactional" }],
      competitors: [{ domain: "rival.com", intersections: 12, organic_etv: 800 }],
      honesty: "Estimated from DataForSEO Labs.",
    },
  },
};

describe("Domain Overview Semrush workspace", () => {
  beforeEach(() => {
    getDomainOverview.mockReset();
    getDomainOverview.mockResolvedValue({ data: empty });
  });

  it("renders chrome, Search, and honest empty / Not implemented cells", async () => {
    render(<DomainOverviewPanel projectId={1} defaultDomain="example.com" />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Domain Overview" })).toBeInTheDocument());
    expect(screen.getByRole("button", { name: /Search/ })).toBeInTheDocument();
    expect(screen.getAllByText("Not implemented").length).toBeGreaterThan(0);
    expect(screen.queryByText("91%")).not.toBeInTheDocument();
    expect(screen.queryByText("Keyword Magic Tool")).not.toBeInTheDocument();
  });

  it("shows vendor traffic after Search and keeps Authority Score unimplemented", async () => {
    getDomainOverview
      .mockResolvedValueOnce({ data: empty })
      .mockResolvedValueOnce({ data: filled });
    const user = userEvent.setup();
    render(<DomainOverviewPanel projectId={1} defaultDomain="example.com" />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Domain Overview" })).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: /Search/ }));
    await waitFor(() => expect(screen.getByText("1.2k")).toBeInTheDocument());
    expect(screen.getByText("buy widgets")).toBeInTheDocument();
    expect(screen.getAllByText("Not implemented").length).toBeGreaterThan(0);
    expect(getDomainOverview.mock.calls.some((c) => c[2]?.fetch === true)).toBe(true);
  });
});
