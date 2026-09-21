package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services/entitlements"
)

// TestAdminStatsMRR_HandlesOrphanedRows pins the revenue-math
// invariant: a user who trialled → canceled → subscribed → canceled
// → resubscribed accumulates several subscription rows but
// contributes ONE active subscription to MRR. The fix alongside
// this test ensures the AdminHandler.Stats query:
//
//   1. Sums monthly_cents / yearly_cents only over status='active'
//      rows, so historical canceled rows never inflate MRR.
//   2. The PlanBreakdown map is restricted to active|trialing
//      rows, not all plan_code rows — so the per-plan total
//      matches the active entitlement count exactly.
//
// What this test does NOT cover: ensuring the subscriptions table
// itself enforces at-most-one-active-per-user (DEFER-007). The
// math here is the dashboard's correct view of whatever is
// currently in the table; data hygiene is a separate fix.
func TestAdminStatsMRR_HandlesOrphanedRows(t *testing.T) {
	harness := newRevenueHarness(t)
	now := time.Now()

	seedRevUser(t, harness.db, 1, "a@x")
	seedRevUser(t, harness.db, 2, "b@x")
	seedRevUser(t, harness.db, 3, "c@x")
	seedRevUser(t, harness.db, 4, "d@x")

	// User A: single active pro_monthly.
	seedRevSub(t, harness.db, 1, "pro_monthly", "active", now.Add(-72*time.Hour))
	// User B: single active pro_yearly.
	seedRevSub(t, harness.db, 2, "pro_yearly", "active", now.Add(-72*time.Hour))
	// User C: history of canceled pro_monthly rows + 1 active
	// pro_monthly. The two canceled rows must NOT contribute.
	seedRevSub(t, harness.db, 3, "pro_monthly", "canceled", now.Add(-720*time.Hour))
	seedRevSub(t, harness.db, 3, "pro_monthly", "canceled", now.Add(-360*time.Hour))
	seedRevSub(t, harness.db, 3, "pro_monthly", "active", now.Add(-1*time.Hour))
	// User D: free, $0.
	seedRevSub(t, harness.db, 4, "free", "active", now.Add(-1*time.Hour))

	got := callStatsViaHarness(t, harness.h)

	// Expected MRR:
	//   User A: pro_monthly active, 2900.
	//   User B: pro_yearly active, 29000/12 = 2416 (integer div on
	//     cents; the storage math is whole-rounded per row, not
	//     per-currency-unit). pre-fix this assertion was 2075
	//     because the seed tables in the spec document used 24900
	//     but the actual seed in migrate.go is 29000.
	//   User C: pro_monthly active, 2900. The two canceled rows
	//     must NOT contribute.
	//   User D: free active, 0.
	// Total: 2900 + 2416 + 2900 = 8216.
	// Post-fix: still 8216 (MRR was already filtered to
	//   status='active'). The PlanBreakdown regression below
	//   proves the per-plan number is now correct.
	if got.MRRCents != 8216 {
		t.Errorf("MRRCents = %d, want 8216 (User A 2900 + User B 2416 + User C 2900)", got.MRRCents)
	}
	// 4 active subs: A, B, C, D. The per-status count itself was
	// never inflated — the bug was on PlanBreakdown and (if the
	// user has multiple active rows) on MRR if a future bug
	// double-counts active rows. Pin both.
	if got.ActiveSubs != 4 {
		t.Errorf("ActiveSubs = %d, want 4 (A + B + C + D)", got.ActiveSubs)
	}
	if got.CanceledSubs != 2 {
		t.Errorf("CanceledSubs = %d, want 2 (User C's two history rows)", got.CanceledSubs)
	}
	if got.PlanBreakdown["pro_monthly"] != 2 {
		// Pre-fix this was 4 (counting User C's two canceled rows).
		t.Errorf("planBreakdown.pro_monthly = %d, want 2 (post-fix)", got.PlanBreakdown["pro_monthly"])
	}
	if got.PlanBreakdown["free"] != 1 {
		t.Errorf("planBreakdown.free = %d, want 1", got.PlanBreakdown["free"])
	}
	if got.PlanBreakdown["pro_yearly"] != 1 {
		t.Errorf("planBreakdown.pro_yearly = %d, want 1", got.PlanBreakdown["pro_yearly"])
	}
}

// ── test harness ─────────────────────────────────────────────────────

type revenueHarness struct {
	db *gorm.DB
	h  *AdminHandler
}

func newRevenueHarness(t *testing.T) *revenueHarness {
	t.Helper()
	tmp := t.TempDir()
	dsn := tmp + "/admin_stats.db"
	database := db.Init(dsn, true)
	t.Cleanup(func() {
		sqlDB, _ := database.DB()
		_ = sqlDB.Close()
	})

	userRepo := repository.NewUserRepository(database)
	subRepo := repository.NewSubscriptionRepository(database)
	planRepo := repository.NewPlanRepository(database)
	auditRepo := repository.NewAdminAuditLogRepository(database)
	projRepo := repository.NewProjectRepository(database)
	ent := entitlements.New(database, subRepo, planRepo, projRepo)
	// privKey is `any`; nil is fine — Stats doesn't mint tokens.
	h := NewAdminHandler(database, userRepo, subRepo, planRepo, auditRepo, projRepo, nil, nil)
	_ = ent
	return &revenueHarness{db: database, h: h}
}

func seedRevUser(t *testing.T, database *gorm.DB, id uint, email string) {
	t.Helper()
	if err := database.Create(&models.User{
		ID:           id,
		Name:         "Test",
		Email:        email,
		PasswordHash: "x",
		Role:         models.RoleOwner,
	}).Error; err != nil {
		t.Fatalf("seed user %d: %v", id, err)
	}
}

func seedRevSub(t *testing.T, database *gorm.DB, userID uint, code, status string, startsAt time.Time) {
	t.Helper()
	if err := database.Create(&models.Subscription{
		UserID:   userID,
		PlanCode: code,
		Status:   status,
		StartsAt: startsAt,
	}).Error; err != nil {
		t.Fatalf("seed sub user=%d code=%s status=%s: %v", userID, code, status, err)
	}
}

// callStatsViaHarness mounts AdminHandler.Stats on a fresh gin
// router with no auth context — we test the SQL math here, not
// the role/cookie gate (that lives in phase7_verify).
func callStatsViaHarness(t *testing.T, h *AdminHandler) adminStatsResponse {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin/stats", h.Stats)
	req, _ := http.NewRequest("GET", "/admin/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Data adminStatsResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode stats body: %v (raw=%s)", err, w.Body.String())
	}
	return resp.Data
}
