import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("next/link", () => ({
  default: ({ href, children, ...rest }: { href: string; children: unknown }) => (
    <a href={href} {...rest}>{children as never}</a>
  ),
}));

import { DailyLoop } from "./daily-loop";

describe("DailyLoop honesty", () => {
  it("shows em dash when vendor cache is empty and labels DataForSEO", () => {
    render(
      <DailyLoop
        snapshot={{
          today_loop: [
            { id: "connect_gsc", title: "Connect Google Search Console", href: "/integrations", done: false, detail: "Overview traffic comes from GSC only." },
            { id: "lookup_domain", title: "Look up domain", href: "/seo/site-explorer", done: false, detail: "DataForSEO Labs. Overview does not auto-fetch." },
          ],
          vendor_brief: {
            source: "dataforseo",
            message: "Cached DataForSEO lookups. Not Search Console. Overview never calls the vendor.",
            domain_explorer: { available: false, stale: false, href: "/seo/site-explorer" },
            ai_visibility: { available: false, stale: false, href: "/seo/ai-visibility" },
            backlink_explorer: { available: false, stale: false, href: "/seo/backlinks" },
          },
        }}
      />
    );

    expect(screen.getByText("DataForSEO")).toBeInTheDocument();
    expect(screen.getByText(/Not Search Console/)).toBeInTheDocument();
    expect(screen.getByText("Connect Google Search Console")).toBeInTheDocument();
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
    expect(screen.queryByText("50")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: /domain explorer/i })).toHaveAttribute("href", "/seo/site-explorer");
  });

  it("renders cached vendor numbers beside the loop without inventing a score", () => {
    render(
      <DailyLoop
        snapshot={{
          today_loop: [
            { id: "lookup_domain", title: "Look up domain", href: "/seo/site-explorer", done: true, detail: "DataForSEO Labs. Overview does not auto-fetch." },
          ],
          vendor_brief: {
            source: "dataforseo",
            message: "Cached DataForSEO lookups. Not Search Console. Overview never calls the vendor.",
            domain_explorer: {
              available: true, stale: true, href: "/seo/site-explorer",
              organic_etv: 50, organic_count: 3, rank: 10,
            },
            ai_visibility: { available: true, stale: false, href: "/seo/ai-visibility", mentions: 4, citations: 15 },
            backlink_explorer: { available: false, stale: false, href: "/seo/backlinks" },
          },
        }}
      />
    );

    expect(screen.getByText("50")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
    expect(screen.getByText("15")).toBeInTheDocument();
    expect(screen.getByText("Stale")).toBeInTheDocument();
    expect(screen.queryByText(/AI visibility score/i)).not.toBeInTheDocument();
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
  });
});
