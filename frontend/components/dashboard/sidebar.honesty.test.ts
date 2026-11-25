import { describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  usePathname: () => "/dashboard",
}));

vi.mock("next/link", () => ({
  default: ({ children }: { children: unknown }) => children,
}));

import { navGroups } from "./sidebar";

function hrefs() {
  return navGroups.flatMap((g) => g.sections.flatMap((s) => s.items.map((i) => i.href)));
}

function names() {
  return navGroups.flatMap((g) => g.sections.flatMap((s) => s.items.map((i) => i.name)));
}

describe("navGroups honesty", () => {
  it("does not list Visual AI or Profile Discovery", () => {
    const all = hrefs();
    expect(all).toContain("/seo/backlinks");
    expect(all).toContain("/seo/ai-visibility");
    expect(all).toContain("/seo/ai-visibility-gap");
    expect(all).toContain("/seo/keyword-gap");
    expect(all).toContain("/seo/domain-compare");
    expect(all).toContain("/seo/positioning-map");
    expect(all).toContain("/seo/content-gap");
    expect(all).toContain("/seo/citation-gap");
    expect(all).toContain("/seo/backlink-gap");
    expect(all).toContain("/seo/position-tracking");
    expect(all).toContain("/seo/site-audit");
    expect(all).toContain("/seo/domain-overview");
    expect(all).toContain("/alerts");
    expect(all).not.toContain("/ai/visual");
    expect(all).not.toContain("/social/competitors");
  });

  it("does not list Semrush-only tools we do not ship", () => {
    const all = names();
    expect(all).not.toContain("Keyword Magic Tool");
    expect(all).not.toContain("Organic Rankings");
    expect(all).not.toContain("Organic Rankings 2.0");
    expect(all).not.toContain("Local");
    expect(all).not.toContain("AI PR");
    expect(all).not.toContain("Apps");
    expect(all).not.toContain("Traffic & Market");
    expect(all).not.toContain("Content Ideas");
  });
});
