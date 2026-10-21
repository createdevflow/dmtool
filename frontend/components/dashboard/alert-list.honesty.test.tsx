import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("next/link", () => ({
  default: ({ href, children, ...rest }: { href: string; children: unknown }) => (
    <a href={href} {...rest}>{children as never}</a>
  ),
}));

import { AlertList } from "./alert-list";

describe("AlertList honesty", () => {
  it("renders stored alerts as links and does not invent a traffic drop", () => {
    render(
      <AlertList
        alerts={[
          { id: "gsc_missing", title: "Connect Google Search Console", href: "/integrations", body: "Overview traffic stays empty until GSC is connected.", severity: "high" },
        ]}
      />
    );
    expect(screen.getByText("Connect Google Search Console")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Connect Google Search Console/i })).toHaveAttribute("href", "/integrations");
    expect(screen.queryByText(/No alert engine is wired/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/traffic drop/i)).not.toBeInTheDocument();
  });

  it("shows an empty label when there are no stored alerts", () => {
    render(<AlertList alerts={[]} emptyLabel="No alerts from stored state." />);
    expect(screen.getByText("No alerts from stored state.")).toBeInTheDocument();
  });
});
