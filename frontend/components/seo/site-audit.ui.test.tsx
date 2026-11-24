import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

vi.mock("@/components/ui/toaster", () => ({ toast: vi.fn() }));

const listSiteAudit = vi.fn();
const getSiteAudit = vi.fn();
vi.mock("@/lib/api-client", () => ({
  dashboardApi: {
    listSiteAudit: (...args: unknown[]) => listSiteAudit(...args),
    getSiteAudit: (...args: unknown[]) => getSiteAudit(...args),
    saveSiteAudit: vi.fn(),
    runSiteAudit: vi.fn(),
  },
}));

import { SiteAuditPanel } from "./site-audit";

const emptyList = {
  data: {
    limits: { plan_label: "Free", pages_per_audit: 100, websites: 1, run_page_cap: 8, js_rendering: false },
    campaigns: [],
    honesty: "This run crawled 0 HTML pages",
  },
  meta: { page: 1, page_size: 10, total: 0 },
};

const listed = {
  data: {
    limits: { plan_label: "Free", pages_per_audit: 100, websites: 1, run_page_cap: 8 },
    campaigns: [{
      project_id: 1,
      project_name: "Acme",
      domain: "example.com",
      last_update: "2026-11-24T12:00:00Z",
      pages_crawled: 2,
      page_limit: 100,
      site_health: { value: 80, status: "ok" },
      ai_search_health: { status: "not_implemented", label: "Not implemented" },
      errors: { value: 1, status: "ok" },
      warnings: { value: 3, status: "ok" },
      crawlability: { value: 90, status: "ok" },
      https: { value: 100, status: "ok" },
      international: { status: "empty" },
      performance: { value: 70, status: "ok" },
      linking: { value: 50, status: "ok" },
      markups: { status: "empty" },
      core_web_vitals: { status: "not_implemented", label: "Not implemented" },
    }],
    honesty: "This run crawled 2 HTML pages",
  },
  meta: { page: 1, page_size: 10, total: 1 },
};

describe("Site Audit Semrush list", () => {
  beforeEach(() => {
    listSiteAudit.mockReset();
    getSiteAudit.mockReset();
    listSiteAudit.mockResolvedValue({ data: emptyList });
  });

  it("renders campaign list chrome, Create Project, and empty dashes", async () => {
    render(<SiteAuditPanel projects={[{ id: 1, name: "Acme", url: "https://example.com" }]} />);

    await waitFor(() => expect(screen.getByRole("heading", { name: "Site Audit" })).toBeInTheDocument());
    expect(screen.getAllByRole("button", { name: /Create Project/ }).length).toBeGreaterThan(0);
    expect(screen.getByText("No Site Audit campaigns yet")).toBeInTheDocument();
    expect(screen.queryByText("91%")).not.toBeInTheDocument();
    expect(screen.queryByText("438")).not.toBeInTheDocument();
  });

  it("shows Semrush columns, honest Not implemented, and real page counts", async () => {
    listSiteAudit.mockResolvedValue({ data: listed });
    getSiteAudit.mockResolvedValue({
      data: {
        data: {
          limits: { plan_label: "Free" },
          campaign: { project_id: 1, domain: "example.com", page_limit: 100 },
          reports: {
            overview: {
              site_health: { value: 80, status: "ok" },
              pages_crawled: 2,
              errors: { value: 1, status: "ok" },
              warnings: { value: 3, status: "ok" },
              notices: { value: 0, status: "ok" },
              thematic: {
                crawlability: { value: 90, status: "ok" },
                https: { value: 100, status: "ok" },
                international: { status: "empty" },
                performance: { value: 70, status: "ok" },
                linking: { value: 50, status: "ok" },
                markups: { status: "empty" },
                core_web_vitals: { status: "not_implemented", label: "Not implemented" },
                ai_search_health: { status: "not_implemented", label: "Not implemented" },
              },
              honesty: "This run crawled 2 HTML pages",
            },
            issues: [],
            pages: [{ url: "https://example.com/", status_code: 200, title: "Home", score: 80, errors: 0, warnings: 0, inlinks: 0, indexable: true }],
            statistics: { pages: 2, indexable: 2, blocked: 0, status_codes: { "200": 2 } },
            compare: { locked: true, plan_needed: "SEO", reason: "Compare Crawls is included on SEO and above." },
          },
        },
      },
    });
    const user = userEvent.setup();
    render(<SiteAuditPanel projects={[{ id: 1, name: "Acme", url: "https://example.com" }]} />);

    await waitFor(() => expect(screen.getByText("Project")).toBeInTheDocument());
    expect(screen.getByText("Last Update")).toBeInTheDocument();
    expect(screen.getByText("Pages Crawled")).toBeInTheDocument();
    expect(screen.getByText("Site Health")).toBeInTheDocument();
    expect(screen.getByText("AI Search Health")).toBeInTheDocument();
    expect(screen.getByText("Errors")).toBeInTheDocument();
    expect(screen.getByText("Warnings")).toBeInTheDocument();
    expect(screen.getByText("Crawlability")).toBeInTheDocument();
    expect(screen.getByText("HTTPS")).toBeInTheDocument();
    expect(screen.getByText("Int. SEO")).toBeInTheDocument();
    expect(screen.getByText("Site Performance")).toBeInTheDocument();
    expect(screen.getByText("Internal Linking")).toBeInTheDocument();
    expect(screen.getByText("Markups")).toBeInTheDocument();
    expect(screen.getByText("Core Web Vitals")).toBeInTheDocument();
    expect(screen.getByText("Acme")).toBeInTheDocument();
    expect(screen.getByText("2 / 100")).toBeInTheDocument();
    expect(screen.getAllByText("Not implemented").length).toBeGreaterThanOrEqual(2);
    expect(screen.queryByText("91%")).not.toBeInTheDocument();

    await user.click(screen.getByText("Acme"));
    await waitFor(() => expect(screen.getByRole("button", { name: /Start Audit/ })).toBeInTheDocument());
    expect(screen.getByRole("button", { name: /^Overview$/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Issues$/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Crawled Pages$/ })).toBeInTheDocument();
    expect(screen.getAllByText("Not implemented").length).toBeGreaterThanOrEqual(2);
  });
});
