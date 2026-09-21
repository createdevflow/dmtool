import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const generateContent = vi.fn();
const getProjects = vi.fn();

vi.mock("next/link", () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

vi.mock("@/lib/api-client", () => ({
  dashboardApi: {
    getProjects: (...args: unknown[]) => getProjects(...args),
    generateContent: (...args: unknown[]) => generateContent(...args),
  },
}));

import AIChatPage from "./page";

describe("AI chat never sends project_id 0", () => {
  beforeEach(() => {
    generateContent.mockReset();
    getProjects.mockReset();
    localStorage.clear();
    Element.prototype.scrollIntoView = vi.fn();
  });

  it("does not call generateContent when there is no project", async () => {
    getProjects.mockResolvedValue({ data: { data: [] } });
    render(<AIChatPage />);

    await waitFor(() => {
      expect(screen.getByText(/Create a project first/i)).toBeInTheDocument();
    });

    expect(screen.getByRole("button", { name: /send/i })).toBeDisabled();
    expect(generateContent).not.toHaveBeenCalled();
  });

  it("sends the selected project id, never 0", async () => {
    getProjects.mockResolvedValue({
      data: { data: [{ id: 7, name: "Citc" }] },
    });
    generateContent.mockResolvedValue({
      data: { data: { variants: [{ content: "ok" }], source: "template" } },
    });

    render(<AIChatPage />);
    const input = await screen.findByPlaceholderText(/topic/i);
    await userEvent.type(input, "launch post");
    await userEvent.click(screen.getByRole("button", { name: /send/i }));

    await waitFor(() => {
      expect(generateContent).toHaveBeenCalled();
    });
    const payload = generateContent.mock.calls[0][0] as { project_id: number };
    expect(payload.project_id).toBe(7);
    expect(payload.project_id).not.toBe(0);
  });
});
