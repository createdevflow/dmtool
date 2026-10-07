import { describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  usePathname: () => "/dashboard",
}));

vi.mock("next/link", () => ({
  default: ({ children }: { children: unknown }) => children,
}));

import { navGroups } from "./sidebar";

describe("navGroups honesty", () => {
  it("does not list Visual AI or Profile Discovery", () => {
    const hrefs = navGroups.flatMap((g) => g.items.map((i) => i.href));
    expect(hrefs).toContain("/seo/backlinks");
    expect(hrefs).toContain("/seo/ai-visibility");
    expect(hrefs).not.toContain("/ai/visual");
    expect(hrefs).not.toContain("/social/competitors");
  });
});
