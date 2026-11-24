import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import * as React from "react";

vi.mock("next/navigation", () => ({
  usePathname: () => "/seo/site-audit",
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...props
  }: {
    children: React.ReactNode;
    href: string;
    onClick?: () => void;
    className?: string;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

import { Sidebar } from "./sidebar";
import { SidebarDrawerProvider } from "./sidebar-drawer-context";

function renderNav() {
  return render(
    <SidebarDrawerProvider>
      <Sidebar />
    </SidebarDrawerProvider>
  );
}

describe("Semrush two-pane user nav", () => {
  it("shows SEO tools for /seo/site-audit and omits ModeSwitcher", () => {
    renderNav();

    expect(screen.getByRole("heading", { name: "SEO" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Site Audit" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Position Tracking" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Home" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "SEO" })).toBeInTheDocument();
    expect(screen.queryByRole("radiogroup", { name: "Dashboard mode" })).not.toBeInTheDocument();
    expect(screen.queryByText("Keyword Magic Tool")).not.toBeInTheDocument();
    expect(screen.queryByText("Search")).not.toBeInTheDocument();
    expect(screen.queryByText("Combined")).not.toBeInTheDocument();
  });

  it("switches the tool list when a rail is clicked", async () => {
    const user = userEvent.setup();
    renderNav();

    await user.click(screen.getByRole("button", { name: "Social" }));
    expect(screen.getByRole("heading", { name: "Social" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Profile Analyzer" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Site Audit" })).not.toBeInTheDocument();
  });
});
