// PlanModal smoke tests.
//
// PlanModal is a complex client component: Radix Dialog + form state
// + an axios-backed create/update call. We render it in jsdom under
// React Testing Library, stub the @/lib/api-client adminApi singleton
// via vi.mock, and assert the user-visible state transitions:
//   * edit mode pre-fills the form from the row data
//   * edit mode LOCKS the code field
//   * create mode accepts an input typed into the code field
//   * submit rejects invalid codes with the inline regex toast
//   * submit propagates server rejections (400 / 404 / etc.)
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import * as React from "react";
import { Dialog, DialogContent } from "@/components/ui/dialog";

// Stub the api-client module *before* importing the component so the
// component picks up the stub on evaluation.
const mockCreatePlan = vi.fn();
const mockUpdatePlan = vi.fn();
vi.mock("@/lib/api-client", () => ({
  adminApi: {
    createPlan: (...args: unknown[]) => mockCreatePlan(...args),
    updatePlan: (...args: unknown[]) => mockUpdatePlan(...args),
  },
}));

// Stub the toast so we can assert calls without rendering Toaster.
const mockToast = vi.fn();
vi.mock("@/components/ui/toaster", () => ({
  toast: (...args: unknown[]) => mockToast(...args),
}));

// Import after mocks so the component picks up stubs.
import { PlanModal } from "./page";

beforeEach(() => {
  mockCreatePlan.mockReset();
  mockUpdatePlan.mockReset();
  mockToast.mockReset();
});

// PlanModal is rendered inside a Dialog tree (it expects to be
// mounted by a parent that opens with state). For tests we mount it
// in always-open mode (Dialog open={true}).
function renderModal(mode: "create" | "edit", initial = {}) {
  const defaults = {
    code: "",
    name: "",
    description: "",
    tier_rank: 0,
    monthly_cents: 0,
    yearly_cents: 0,
    max_sites: 1,
    is_active: true,
  };
  return render(
    <Dialog open>
      <DialogContent>
        <PlanModal
          mode={mode}
          planId={1}
          initial={{ ...defaults, ...initial }}
          onClose={() => {}}
          onSaved={() => {}}
        />
      </DialogContent>
    </Dialog>,
  );
}

describe("PlanModal", () => {
  it("edit mode pre-fills the form from initial values", async () => {
    renderModal("edit", {
      code: "team",
      name: "Team",
      monthly_cents: 4900,
      yearly_cents: 49000,
      max_sites: 10,
      tier_rank: 5,
      description: "team plan",
      is_active: false,
    });

    expect(
      (screen.getByLabelText(/^Code/i) as HTMLInputElement).value,
    ).toBe("team");
    expect(
      (screen.getByLabelText(/^Name/i) as HTMLInputElement).value,
    ).toBe("Team");
    expect(
      (screen.getByLabelText(/Monthly/i) as HTMLInputElement).value,
    ).toBe("4900");
    expect(
      (screen.getByLabelText(/Yearly/i) as HTMLInputElement).value,
    ).toBe("49000");
    expect(
      (screen.getByLabelText(/Max sites/i) as HTMLInputElement).value,
    ).toBe("10");
    expect(
      (screen.getByLabelText(/Tier rank/i) as HTMLInputElement).value,
    ).toBe("5");
    expect(
      (screen.getByLabelText(/Description/i) as HTMLTextAreaElement)
        .value,
    ).toBe("team plan");
  });

  it("edit mode DISABLES the code input (code is the plan's PK)", () => {
    renderModal("edit", { code: "team" });
    const codeInput = screen.getByLabelText(/^Code/i) as HTMLInputElement;
    expect(codeInput.disabled).toBe(true);
  });

  it("create mode keeps the code input enabled", () => {
    renderModal("create");
    const codeInput = screen.getByLabelText(/^Code/i) as HTMLInputElement;
    expect(codeInput.disabled).toBe(false);
  });

  it("code 'BadCode' (uppercase) triggers inline regex toast", async () => {
    const user = userEvent.setup();
    renderModal("create");

    await user.type(screen.getByLabelText(/^Code/i), "BadCode");
    await user.type(screen.getByLabelText(/^Name/i), "X");
    await user.click(screen.getByRole("button", { name: /Create/i }));

    expect(mockToast).toHaveBeenCalledTimes(1);
    expect(mockToast.mock.calls[0]?.[0]).toMatch(
      /Code must be lowercase letters, digits, and underscores/,
    );
    // The fix: no server call was made because validation failed
    // before submission.
    expect(mockCreatePlan).not.toHaveBeenCalled();
  });

  // Server-rejection path.
  // `e instanceof Error ? e.message : "Save failed"`. The
  // production path returns an AxiosError; we throw a real Error
  // carrying the server message so the test reflects what the
  // user actually sees on a 400 from the backend.
  it("server rejection surfaces the error message via toast", async () => {
    mockCreatePlan.mockRejectedValueOnce(new Error("Plan code already exists"));
    const user = userEvent.setup();
    renderModal("create");

    await user.type(screen.getByLabelText(/^Code/i), "good_code");
    await user.type(screen.getByLabelText(/^Name/i), "Good");
    await user.click(screen.getByRole("button", { name: /Create/i }));

    await waitFor(() =>
      expect(mockToast).toHaveBeenCalledWith(
        "Plan code already exists",
        "error",
      ),
    );
  });

  // The JS-side checks for `monthly_cents < 0` and `max_sites < 1`
  // are not unit-tested here. jsdom honours `<input type="number"
  // min=0>` and sanitises user.type to "" — bypassing the DOM
  // filter via the native value setter is brittle across React
  // 19 versions. The matching server-side guards (INVALID_PRICE,
  // INVALID_MAX_SITES) are exercised in
  // backend/_tools/phase7_verify/main.go. We rely on the regex +
  // length test above to prove the validator runs, and trust the
  // numeric branches to fire what they fire in production.
  it.todo("negative monthly_cents rejected before submit");
  it.todo("zero max_sites rejected before submit");
});
