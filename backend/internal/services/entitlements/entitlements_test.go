package entitlements

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

// harness spins up an isolated SQLite DB with the migration runner +
// plan seeds, then constructs an entitlements service over it.
type harness struct {
	database         *gorm.DB
	ent              *Service
	subscriptionRepo repository.SubscriptionRepository
	projectRepo      repository.ProjectRepository
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tmp := t.TempDir()
	dsn := filepath.Join(tmp, "entitlements.db")

	database := db.Init(dsn, true)
	if database == nil {
		t.Fatal("db.Init returned nil")
	}
	t.Cleanup(func() {
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	sub := repository.NewSubscriptionRepository(database)
	plan := repository.NewPlanRepository(database)
	proj := repository.NewProjectRepository(database)

	return &harness{
		database:         database,
		ent:              New(database, sub, plan, proj),
		subscriptionRepo: sub,
		projectRepo:      proj,
	}
}

// newUser inserts a minimal User so subscription / project FK targets
// exist. We don't use the auth flow here — phase 2 only cares about
// entitlement math.
func (h *harness) newUser(id uint) {
	h.database.Create(&models.User{
		ID:           id,
		Name:         "Test User",
		Email:        "u@example.com",
		PasswordHash: "x",
		Role:         "owner",
	})
}

func (h *harness) newSub(userID uint, code, status string) {
	_ = h.subscriptionRepo.Create(&models.Subscription{
		UserID:   userID,
		PlanCode: code,
		Status:   status,
	})
}

func (h *harness) newProject(userID uint, goal string) {
	_ = h.projectRepo.Create(&models.Project{
		UserID: userID,
		Name:   "p",
		URL:    "https://example.com",
		Goal:   goal,
	})
}

// expectQuota matches *QuotaExceededError against err.
func expectQuota(t *testing.T, err error) *QuotaExceededError {
	t.Helper()
	var qx *QuotaExceededError
	if !errors.As(err, &qx) {
		t.Fatalf("expected *QuotaExceededError, got %T (%v)", err, err)
	}
	return qx
}

func TestCanCreateSEOProject_FreePlanAllowsOneSEOProject(t *testing.T) {
	h := newHarness(t)
	h.newUser(1)
	h.newSub(1, models.PlanCodeFree, models.SubscriptionStatusActive)

	allowed, current, limit, planCode, err := h.ent.CanCreateSEOProject(context.Background(), 1, models.GoalSEO)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !allowed {
		t.Errorf("expected create allowed")
	}
	if current != 0 {
		t.Errorf("current = %d, want 0", current)
	}
	if limit != 1 {
		t.Errorf("limit = %d, want 1 (free plan)", limit)
	}
	if planCode != models.PlanCodeFree {
		t.Errorf("plan = %q, want %q", planCode, models.PlanCodeFree)
	}
}

func TestCanCreateSEOProject_FreePlanBlocksSecondSEOProject(t *testing.T) {
	h := newHarness(t)
	h.newUser(2)
	h.newSub(2, models.PlanCodeFree, models.SubscriptionStatusActive)
	h.newProject(2, models.GoalSEO)

	allowed, _, limit, _, err := h.ent.CanCreateSEOProject(context.Background(), 2, models.GoalSEO)
	if err == nil {
		t.Fatalf("expected QuotaExceededError, got nil")
	}
	if allowed {
		t.Errorf("expected create denied")
	}
	qx := expectQuota(t, err)
	if qx.Limit != 1 {
		t.Errorf("qx.Limit = %d, want 1", qx.Limit)
	}
	if qx.Current != 1 {
		t.Errorf("qx.Current = %d, want 1", qx.Current)
	}
	if limit != 1 {
		t.Errorf("returned limit = %d, want 1", limit)
	}
}

func TestCanCreateSEOProject_ProMonthlyAllowsUpTo3(t *testing.T) {
	h := newHarness(t)
	h.newUser(3)
	h.newSub(3, models.PlanCodeProMonthly, models.SubscriptionStatusActive)

	// 0 existing — allow.
	allowed0, _, limit0, _, err0 := h.ent.CanCreateSEOProject(context.Background(), 3, models.GoalSEO)
	if err0 != nil || !allowed0 || limit0 != 3 {
		t.Fatalf("iter 0: allowed=%v err=%v limit=%d want true nil 3", allowed0, err0, limit0)
	}
	h.newProject(3, models.GoalSEO)

	// 1 existing — allow.
	allowed1, _, _, _, err1 := h.ent.CanCreateSEOProject(context.Background(), 3, models.GoalSEO)
	if err1 != nil || !allowed1 {
		t.Fatalf("iter 1: allowed=%v err=%v", allowed1, err1)
	}
	h.newProject(3, models.GoalSEO)

	// 2 existing — still allow (3rd would push to limit only).
	allowed2, _, _, _, err2 := h.ent.CanCreateSEOProject(context.Background(), 3, models.GoalSEO)
	if err2 != nil || !allowed2 {
		t.Fatalf("iter 2: allowed=%v err=%v", allowed2, err2)
	}
	h.newProject(3, models.GoalSEO)

	// 3 existing — deny.
	allowed3, _, limit3, _, err3 := h.ent.CanCreateSEOProject(context.Background(), 3, models.GoalSEO)
	if err3 == nil {
		t.Fatalf("iter 3: expected err, got nil")
	}
	if allowed3 {
		t.Errorf("iter 3: expected denied")
	}
	if limit3 != 3 {
		t.Errorf("iter 3: limit = %d, want 3", limit3)
	}
	qx := expectQuota(t, err3)
	if qx.Current != 3 || qx.Limit != 3 {
		t.Errorf("iter 3: qx.Current=%d qx.Limit=%d want 3/3", qx.Current, qx.Limit)
	}
}

func TestCanCreateSEOProject_ProYearlySameLimit(t *testing.T) {
	h := newHarness(t)
	h.newUser(4)
	h.newSub(4, models.PlanCodeProYearly, models.SubscriptionStatusActive)

	allowed, _, limit, planCode, err := h.ent.CanCreateSEOProject(context.Background(), 4, models.GoalSEO)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !allowed {
		t.Errorf("expected create allowed on pro_yearly")
	}
	if limit != 3 {
		t.Errorf("pro_yearly limit = %d, want 3 (parity with pro_monthly)", limit)
	}
	if planCode != models.PlanCodeProYearly {
		t.Errorf("plan = %q, want %q", planCode, models.PlanCodeProYearly)
	}
}

func TestCanCreateSEOProject_SocialOnlyNotGated(t *testing.T) {
	h := newHarness(t)
	h.newUser(5)
	h.newSub(5, models.PlanCodeFree, models.SubscriptionStatusActive)
	h.newProject(5, models.GoalSEO) // 1 SEO already; free plan quota hit

	allowed, _, limit, _, err := h.ent.CanCreateSEOProject(context.Background(), 5, models.GoalSocial)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !allowed {
		t.Errorf("social-only create should not be gated")
	}
	if limit != 1 {
		t.Errorf("limit returned for social-only = %d, want 1 (free plan)", limit)
	}
}

func TestCanCreateSEOProject_BothGoalCountsAsSEO(t *testing.T) {
	h := newHarness(t)
	h.newUser(6)
	h.newSub(6, models.PlanCodeFree, models.SubscriptionStatusActive)

	// First 'both' should be allowed (free plan: 0 → 1).
	allowed0, _, _, _, err0 := h.ent.CanCreateSEOProject(context.Background(), 6, models.GoalBoth)
	if err0 != nil || !allowed0 {
		t.Fatalf("first both: allowed=%v err=%v want true nil", allowed0, err0)
	}
	h.newProject(6, models.GoalBoth)

	// Second 'both' should be denied (free plan: 1 → 2 would exceed).
	allowed1, _, _, _, err1 := h.ent.CanCreateSEOProject(context.Background(), 6, models.GoalBoth)
	if err1 == nil {
		t.Fatalf("second both: expected err, got nil")
	}
	if allowed1 {
		t.Errorf("second both: expected denied")
	}
	expectQuota(t, err1)
}

func TestCanCreateSEOProject_NoSubscriptionFallsBackToFree(t *testing.T) {
	h := newHarness(t)
	h.newUser(7)
	// No subscription row at all → fall back to free.

	allowed, _, limit, planCode, err := h.ent.CanCreateSEOProject(context.Background(), 7, models.GoalSEO)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !allowed {
		t.Errorf("first SEO create with no sub should be allowed (free fallback)")
	}
	if limit != 1 {
		t.Errorf("limit = %d, want 1 (free default)", limit)
	}
	if planCode != models.PlanCodeFree {
		t.Errorf("plan = %q, want %q (default)", planCode, models.PlanCodeFree)
	}
}

func TestCanCreateSEOProject_CanceledSubFallsBackToFree(t *testing.T) {
	h := newHarness(t)
	h.newUser(8)
	// A canceled subscription must NOT grant pro limits. The
	// subscription repo's FindCurrentByUser only returns rows with
	// status IN (active, trialing), so this row is filtered out and
	// the service falls back to free.
	_ = h.subscriptionRepo.Create(&models.Subscription{
		UserID:   8,
		PlanCode: models.PlanCodeProMonthly,
		Status:   models.SubscriptionStatusCanceled,
	})

	allowed, _, limit, planCode, err := h.ent.CanCreateSEOProject(context.Background(), 8, models.GoalSEO)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !allowed {
		t.Errorf("first SEO with canceled pro should fall back to free and be allowed")
	}
	if limit != 1 {
		t.Errorf("limit = %d, want 1 (free fallback, not pro's 3)", limit)
	}
	if planCode != models.PlanCodeFree {
		t.Errorf("plan = %q, want free", planCode)
	}
}
