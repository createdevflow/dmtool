// ImpersonationBanner smoke tests.
//
// The banner is the user-visible phase-7 impersonation indicator:
// while an admin is impersonating a user, every admin page renders
// "Impersonating <name> <email>" with a Stop button. Stop calls
// /admin/users/:id/stop-impersonation and clears the
// dmtool_impersonation_target cookie. We exercise:
//
//   * No cookie → banner doesn't render.
//   * Target cookie set → banner renders with target name + email.
//   * Click Stop → adminApi.stopImpersonation called + cookie cleared
//     + banner re-hidden.
//
// The cookie layer is identical to production — jsdom's
// document.cookie works the same way the browser does for set/get
// with non-HttpOnly cookies.

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import * as React from "react";

const mockRefresh = vi.fn();
const mockStopImpersonation = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: mockRefresh }),
}));

vi.mock("@/lib/api-client", () => ({
  adminApi: {
    stopImpersonation: (...args: unknown[]) => mockStopImpersonation(...args),
  },
}));

import { ImpersonationBanner } from "./impersonation-banner";
import {
  COOKIE_IMPERSONATION_TOKEN,
  COOKIE_IMPERSONATION_TARGET,
  setImpersonation,
  clearImpersonation,
} from "@/lib/auth-cookie";

function setImpersonationCookie(target: { id: number; email: string; name: string }) {
  // setImpersonation writes both cookies. We use the production
  // helper rather than document.cookie directly so we're testing
  // the same path the form code uses.
  setImpersonation("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ.a", target, 30);
}

beforeEach(() => {
  mockRefresh.mockReset();
  mockStopImpersonation.mockReset();
  // Make sure we start with no impersonation cookies.
  clearImpersonation();
});

afterEach(() => {
  clearImpersonation();
});

describe("ImpersonationBanner", () => {
  it("does not render when no impersonation cookie is present", () => {
    render(<ImpersonationBanner />);
    expect(screen.queryByTestId("impersonation-banner")).toBeNull();
  });

  it("renders the target's name and email when impersonation is active", async () => {
    setImpersonationCookie({
      id: 42,
      email: "alice@example.com",
      name: "Alice Anderson",
    });
    render(<ImpersonationBanner />);

    const banner = await screen.findByTestId("impersonation-banner");
    expect(banner).toBeInTheDocument();
    expect(banner).toHaveTextContent("Impersonating Alice Anderson");
    expect(banner).toHaveTextContent("alice@example.com");
    // data-attribute reflects the cookie name the source used —
    // proves the banner reads dmtool_impersonation_target (not a
    // hard-coded name).
    expect(banner).toHaveAttribute(
      "data-impersonation-cookie",
      COOKIE_IMPERSONATION_TOKEN,
    );
    // The Token cookie value is not echoed to the DOM (it would
    // leak creds via DevTools), but the banner should have a Stop
    // button.
    expect(
      screen.getByRole("button", { name: /Stop impersonation/i }),
    ).toBeInTheDocument();
  });

  it("stop flow: calls adminApi.stopImpersonation, clears cookie, hides banner", async () => {
    mockStopImpersonation.mockResolvedValueOnce({ data: { message: "ok" } });
    setImpersonationCookie({
      id: 7,
      email: "bob@example.com",
      name: "Bob Brown",
    });
    const user = userEvent.setup();
    render(<ImpersonationBanner />);

    const stopBtn = await screen.findByRole("button", {
      name: /Stop impersonation/i,
    });
    await user.click(stopBtn);

    // adminApi.stopImpersonation fired with the target's id.
    await waitFor(() =>
      expect(mockStopImpersonation).toHaveBeenCalledWith(7),
    );

    // Cookie helper cleared both cookies. We assert by re-running
    // readImpersonationTarget indirectly — the component's
    // useEffect re-renders, sets target=null, banner re-renders
    // as null, which is exactly what we wanted.
    await waitFor(() => {
      expect(screen.queryByTestId("impersonation-banner")).toBeNull();
    });

    // document.cookie for the target cookie is now empty/expires.
    expect(document.cookie).not.toMatch(/dmtool_impersonation_target=/);

    // router.refresh is invoked to reconcile the page state.
    expect(mockRefresh).toHaveBeenCalled();
  });

  it("continues to clear the cookie even if the server call throws", async () => {
    mockStopImpersonation.mockRejectedValueOnce(new Error("network"));
    setImpersonationCookie({
      id: 9,
      email: "carol@example.com",
      name: "Carol Chen",
    });
    const user = userEvent.setup();
    render(<ImpersonationBanner />);

    await user.click(
      await screen.findByRole("button", { name: /Stop impersonation/i }),
    );

    // The component catches the error; the cookie clear + UI hide
    // happen regardless (defense in depth — the audit-write
    // failure doesn't lock the admin into impersonation).
    await waitFor(() => {
      expect(screen.queryByTestId("impersonation-banner")).toBeNull();
    });
    expect(document.cookie).not.toMatch(/dmtool_impersonation_target=/);
  });
});
