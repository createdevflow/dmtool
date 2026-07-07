package entitlements

// ─────────────────────────────────────────────────────────────────────
// ResolveSubscription — phase 9 direct tests.
//
// Phase 2 covered ResolveSubscription transitively through
// CanCreateSEOProject. Phase 9 adds direct tests so a future change
// to the fallback behavior (e.g. "canceled rows no longer return the
// last plan code") shows up here rather than as an unexpected
// surprise in the SEO quota tests.
//
// Harness setup (harness, newUser, newSub) lives in
// entitlements_test.go — same package, so the symbols are visible
// here without an import.
// ─────────────────────────────────────────────────────────────────────

import (
	"context"
	"testing"

	"backend/internal/models"
)

// No subscription row at all → ResolveSubscription returns a nil
// subscription pointer and the *free* plan with the seeded defaults.
func TestResolveSubscription_NoSubscriptionReturnsFreePlan(t *testing.T) {
	h := newHarness(t)
	h.newUser(1)

	sub, plan, err := h.ent.ResolveSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("ResolveSubscription: %v", err)
	}
	if sub != nil {
		t.Errorf("sub = %+v, want nil (no row)", sub)
	}
	if plan == nil {
		t.Fatal("plan = nil, want free plan")
	}
	if plan.Code != models.PlanCodeFree {
		t.Errorf("plan.Code = %q, want %q", plan.Code, models.PlanCodeFree)
	}
	if plan.MaxSites != 1 {
		t.Errorf("plan.MaxSites = %d, want 1", plan.MaxSites)
	}
}

// Active subscription → ResolveSubscription returns the plan row that
// matches the subscription's PlanCode (pro_monthly).
func TestResolveSubscription_ActiveSubscriptionReturnsProPlan(t *testing.T) {
	h := newHarness(t)
	h.newUser(2)
	h.newSub(2, models.PlanCodeProMonthly, models.SubscriptionStatusActive)

	sub, plan, err := h.ent.ResolveSubscription(context.Background(), 2)
	if err != nil {
		t.Fatalf("ResolveSubscription: %v", err)
	}
	if sub == nil {
		t.Fatal("sub = nil, want active row")
	}
	if plan == nil {
		t.Fatal("plan = nil")
	}
	if plan.Code != models.PlanCodeProMonthly {
		t.Errorf("plan.Code = %q, want %q", plan.Code, models.PlanCodeProMonthly)
	}
	if plan.MaxSites != 3 {
		t.Errorf("plan.MaxSites = %d, want 3 (pro_monthly)", plan.MaxSites)
	}
}

// Trialing subscription → row is returned, plan is the trial plan.
// The handler layer (BillingHandler.Me) reads Status out of the row
// separately. The entitlements service intentionally does NOT
// enforce status here — the request-time guard is elsewhere —
// but the returned subscription must reflect what the DB holds.
func TestResolveSubscription_TrialingReturnsProPlan(t *testing.T) {
	h := newHarness(t)
	h.newUser(3)
	h.newSub(3, models.PlanCodeProYearly, models.SubscriptionStatusTrialing)

	sub, plan, err := h.ent.ResolveSubscription(context.Background(), 3)
	if err != nil {
		t.Fatalf("ResolveSubscription: %v", err)
	}
	if sub == nil || sub.Status != models.SubscriptionStatusTrialing {
		t.Errorf("sub.Status = %q, want %q", subStatusOrEmpty(sub), models.SubscriptionStatusTrialing)
	}
	if plan == nil || plan.Code != models.PlanCodeProYearly {
		t.Errorf("plan.Code = %q, want %q", planCodeOrEmpty(plan), models.PlanCodeProYearly)
	}
}

// Canceled subscription (no active/trialing sibling).
// ResolveSubscription uses FindCurrentByUser, which filters to
// active|trialing — a "canceled" row alone is treated as no
// entitlement, and the user falls back to the free plan. The
// canceled row is preserved on disk and the admin panel can read
// it via FindLatestByUser; entitlements.ResolveSubscription
// intentionally does not, so a confirmed-canceled user can't
// keep using Pro features under a stale plan_code.
//
// This test pins that contract. A future change here is the right
// place to argue about whether canceled rows should be treated
// as a grace-period fallback.
func TestResolveSubscription_CanceledFallsBackToFree(t *testing.T) {
	h := newHarness(t)
	h.newUser(4)
	h.newSub(4, models.PlanCodeProMonthly, models.SubscriptionStatusCanceled)

	sub, plan, err := h.ent.ResolveSubscription(context.Background(), 4)
	if err != nil {
		t.Fatalf("ResolveSubscription: %v", err)
	}
	// ResolveSubscription's repo filters out canceled rows, so the
	// returned sub is nil and the plan falls back to free.
	if sub != nil {
		t.Errorf("sub = %+v, want nil (canceled rows aren't 'current')", sub)
	}
	if plan.Code != models.PlanCodeFree {
		t.Errorf("plan.Code = %q, want %q", plan.Code, models.PlanCodeFree)
	}
}

// Canceled subscription + active sibling. The active row wins.
// This documents that the filter prefers the most recent active
// row over a stale canceled one — the order-by-starts_at-desc
// in FindCurrentByUser guarantees that the *newest* active
// plan_code is what's returned, even if the user has bounced
// between plans in the past.
func TestResolveSubscription_CanceledAndActivePrefersActive(t *testing.T) {
	h := newHarness(t)
	h.newUser(11)
	// Old canceled row.
	_ = h.subscriptionRepo.Create(&models.Subscription{
		UserID:   11,
		PlanCode: models.PlanCodeProMonthly,
		Status:   models.SubscriptionStatusCanceled,
	})
	// New active row (different PlanCode) — created after the
	// canceled one, so starts_at ordering picks this one.
	_ = h.subscriptionRepo.Create(&models.Subscription{
		UserID:   11,
		PlanCode: models.PlanCodeProYearly,
		Status:   models.SubscriptionStatusActive,
	})

	sub, plan, err := h.ent.ResolveSubscription(context.Background(), 11)
	if err != nil {
		t.Fatalf("ResolveSubscription: %v", err)
	}
	if sub == nil || sub.PlanCode != models.PlanCodeProYearly {
		t.Errorf("sub = %+v, want active pro_yearly", sub)
	}
	if plan.Code != models.PlanCodeProYearly {
		t.Errorf("plan.Code = %q, want pro_yearly (active, not the canceled monthly)",
			plan.Code)
	}
}

// Unknown plan code on the subscription row → ResolveSubscription
// returns an error (the default-plan fallback is only used when the
// row is missing entirely). This guards against silently masking
// data corruption: a subscription pointing at a deleted plan code
// must surface loudly so the admin can fix it.
func TestResolveSubscription_UnknownPlanCodeReturnsError(t *testing.T) {
	h := newHarness(t)
	h.newUser(5)
	// Insert a subscription referencing a plan_code that has no
	// row in the seeded plans table. We bypass the harness helper
	// because that helper only accepts valid seeded codes.
	if err := h.subscriptionRepo.Create(&models.Subscription{
		UserID:   5,
		PlanCode: "ghost_plan_no_such_row",
		Status:   models.SubscriptionStatusActive,
	}); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	_, _, err := h.ent.ResolveSubscription(context.Background(), 5)
	if err == nil {
		t.Fatal("ResolveSubscription: expected error for unknown plan_code, got nil")
	}
}

// stub helpers — Go's t.Errorf format doesn't handle optional pointer
// fields gracefully, so a tiny helper keeps the asserts readable.
func subStatusOrEmpty(s *models.Subscription) string {
	if s == nil {
		return ""
	}
	return s.Status
}
func planCodeOrEmpty(p *models.Plan) string {
	if p == nil {
		return ""
	}
	return p.Code
}
