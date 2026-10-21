import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("next/link", () => ({
  default: ({ href, children, ...rest }: { href: string; children: unknown }) => (
    <a href={href} {...rest}>{children as never}</a>
  ),
}));

import { StoredWorkList } from "./stored-work-list";

describe("StoredWorkList honesty", () => {
  it("links stored work and does not invent a growth score", () => {
    render(
      <StoredWorkList
        work={[
          { id: "connect_gsc", kind: "loop", title: "Connect Google Search Console", href: "/integrations", done: false, detail: "Overview traffic comes from GSC only." },
          { id: "issue-3", kind: "issue", title: "blocked", href: "/seo/site-explorer", done: false, detail: "high · robots" },
        ]}
      />
    );
    expect(screen.getByText("Connect Google Search Console")).toBeInTheDocument();
    expect(screen.getByText("blocked")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Connect Google Search Console/i })).toHaveAttribute("href", "/integrations");
    expect(screen.queryByText(/growth index/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Batch Content Creation/i)).not.toBeInTheDocument();
  });

  it("shows an empty label when there is no stored work", () => {
    render(<StoredWorkList work={[]} emptyLabel="No stored work yet." />);
    expect(screen.getByText("No stored work yet.")).toBeInTheDocument();
  });
});
