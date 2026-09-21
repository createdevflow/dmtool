package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services/entitlements"
)
// billingHarness spins up an isolated SQLite DB with the migration
// runner + plan seeds, then constructs a full BillingHandler over
// the in-process repositories and entitlements service. We
// populate c.MustGet("user_id") via a tiny middleware so we don't
// have to plumb JWT signing into a handler-level test.
type billingHarness struct {
	db *gorm.DB
	h  *BillingHandler
}
func newBillingHarness(t *testing.T) *billingHarness {
	t.Helper()
	tmp := t.TempDir()
	dsn := tmp + "/billing.db"
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

	userRepo := repository.NewUserRepository(database)
	subRepo := repository.NewSubscriptionRepository(database)
	planRepo := repository.NewPlanRepository(database)
	projRepo := repository.NewProjectRepository(database)
	ent := entitlements.New(database, subRepo, planRepo, projRepo)
	h := NewBillingHandler(database, userRepo, subRepo, planRepo, projRepo, ent)

	return &billingHarness{
		db: database,
		h:  h,
	}
}

// seedUser inserts a user with a unique email so the harness stays
// composable when multiple tests run sequentially.
func (bh *billingHarness) seedUser(id uint, email string) *models.User {
	u := &models.User{
		ID:           id,
		Name:         "Test",
		Email:        email,
		PasswordHash: "x",
		Role:         "owner",
	}
	if err := bh.db.Create(u).Error; err != nil {
		bh.db.Delete(&models.User{ID: id})
		bh.db.Create(u)
	}
	return u
}

// call is the helper that wires gin's test recorder and a minimal
// user-id-stamped context, then dispatches to the named handler.
func (bh *billingHarness) call(handlerName string, body []byte, userID uint) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// r.Use must run before the route handler is added — Gin
	// evaluates middleware in the order they're added. We register
	// the user_id injection first, then mount only the handler
	// under test.
	r.Use(func(c *gin.Context) {
		if userID != 0 {
			c.Set("user_id", userID)
		}
		c.Next()
	})
	h := bh.h
	switch handlerName {
	case "StartTrial":
		r.POST("/api/billing/trial", h.StartTrial)
	case "Subscribe":
		r.POST("/api/billing/subscribe", h.Subscribe)
	}

	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest("POST", "/api/billing/trial", bytes.NewReader(body))
	} else {
		req, _ = http.NewRequest("POST", "/api/billing/trial", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// callSubscribe is like `call` but targets the /subscribe endpoint.
func (bh *billingHarness) callSubscribe(body []byte, userID uint) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID != 0 {
			c.Set("user_id", userID)
		}
		c.Next()
	})
	r.POST("/api/billing/subscribe", bh.h.Subscribe)

	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest("POST", "/api/billing/subscribe", bytes.NewReader(body))
	} else {
		req, _ = http.NewRequest("POST", "/api/billing/subscribe", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── StartTrial tests ───────────────────────────────────────────────────

func TestStartTrial_HappyPath_ProMonthly(t *testing.T) {
	h := newBillingHarness(t)
	h.seedUser(1, "trial1@example.com")

	body, _ := json.Marshal(map[string]string{"plan_code": models.PlanCodeProMonthly})
	w := h.call("StartTrial", body, 1)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}

	var sub models.Subscription
	if err := json.Unmarshal(w.Body.Bytes(), &struct {
		Data *models.Subscription `json:"data"`
	}{Data: &sub}); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if sub.PlanCode != models.PlanCodeProMonthly {
		t.Errorf("plan_code = %q, want pro_monthly", sub.PlanCode)
	}
	if sub.Status != models.SubscriptionStatusTrialing {
		t.Errorf("status = %q, want trialing", sub.Status)
	}
	if sub.TrialEndsAt == nil {
		t.Errorf("TrialEndsAt = nil, want non-nil after a successful trial start")
	}

	// TrialUsedAt is set on the User row.
	var u models.User
	h.db.First(&u, 1)
	if u.TrialUsedAt == nil {
		t.Errorf("TrialUsedAt = nil, want non-nil after first trial")
	}
}
func TestStartTrial_RejectsNonProPlanCodes(t *testing.T) {
	h := newBillingHarness(t)
	h.seedUser(2, "trial2@example.com")

	body, _ := json.Marshal(map[string]string{"plan_code": "free"})
	w := h.call("StartTrial", body, 2)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d body = %s, want 400", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("INVALID_PLAN")) {
		t.Errorf("body should contain INVALID_PLAN: %s", w.Body.String())
	}
}

func TestStartTrial_TrialAlreadyUsed_409(t *testing.T) {
	h := newBillingHarness(t)
	// Seed with TrialUsedAt set to a non-null timestamp.
	usedAt := mustParseTime("2026-01-01T00:00:00Z")
	h.db.Create(&models.User{
		ID:           3,
		Email:        "trial3@example.com",
		PasswordHash: "x",
		Role:         "owner",
		TrialUsedAt:  &usedAt,
	})

	body, _ := json.Marshal(map[string]string{"plan_code": models.PlanCodeProMonthly})
	w := h.call("StartTrial", body, 3)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s, want 409", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("TRIAL_ALREADY_USED")) {
		t.Errorf("body should contain TRIAL_ALREADY_USED: %s", w.Body.String())
	}
}

// Second trial attempt on a fresh user — idempotency rationale
// comes from the user.TrialUsedAt flag, which is set on the first
// successful trial and never cleared. After the first trial, even
// if a stale "active subscription" gets somehow removed, the
// trial_used_at flag prevents a second trial.
func TestStartTrial_SecondTrial_RejectedEvenIfSubGone(t *testing.T) {
	h := newBillingHarness(t)
	usedAt := mustParseTime("2026-01-01T00:00:00Z")
	h.db.Create(&models.User{
		ID:           4,
		Email:        "trial4@example.com",
		PasswordHash: "x",
		Role:         "owner",
		TrialUsedAt:  &usedAt,
	})

	// No subscription row at all — but the flag alone is enough.
	body, _ := json.Marshal(map[string]string{"plan_code": models.PlanCodeProYearly})
	w := h.call("StartTrial", body, 4)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (trial_used_at alone must block)", w.Code)
	}
}

func TestStartTrial_Unauthorized_NoUserID(t *testing.T) {
	h := newBillingHarness(t)
	h.seedUser(5, "trial5@example.com")

	// userID = 0 — JWTAuth wouldn't allow this in production but
	// the handler should defend against the user_id being absent
	// from the context (defense in depth).
	body, _ := json.Marshal(map[string]string{"plan_code": models.PlanCodeProMonthly})
	w := h.call("StartTrial", body, 0)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// ── Subscribe tests ────────────────────────────────────────────────────

func TestSubscribe_HappyPath_NewSubscription(t *testing.T) {
	h := newBillingHarness(t)
	h.seedUser(10, "sub1@example.com")

	body, _ := json.Marshal(map[string]string{"plan_code": models.PlanCodeProYearly})
	w := h.callSubscribe(body, 10)
	// For a brand-new row, the handler returns 201 Created (vs 200
	// on the update-in-place path). The HTTP conventions line up
	// here even though there's no Stripe call yet; when Stripe is
	// wired later, the create branch becomes the checkout-session
	// opener and the update branch handles the webhook.
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s, want 201 (new subscription)", w.Code, w.Body.String())
	}
	var s models.Subscription
	if err := json.Unmarshal(w.Body.Bytes(), &struct {
		Data *models.Subscription `json:"data"`
	}{Data: &s}); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if s.PlanCode != models.PlanCodeProYearly {
		t.Errorf("plan_code = %q, want pro_yearly", s.PlanCode)
	}
	if s.Status != models.SubscriptionStatusActive {
		t.Errorf("status = %q, want active", s.Status)
	}
}
// Subscribe on a user with a previously-canceled subscription row
// inserts a fresh active row alongside the canceled one, because
// Subscribe's lookup uses FindCurrentByUser (filters to
// active|trialing) which hides canceled. This is a known phase 6
// quirk: canceled rows are kept on disk for audit but aren't reused
// when a user re-subscribes. Phase 7+ admin tooling (FindLatestByUser)
// shows them in the audit log; this test documents the behaviour.
func TestSubscribe_CanceledRowIsNotReused_NewRowCreated(t *testing.T) {
	h := newBillingHarness(t)
	h.seedUser(11, "sub2@example.com")

	// Existing canceled row.
	h.db.Create(&models.Subscription{
		UserID:   11,
		PlanCode: models.PlanCodeProMonthly,
		Status:   models.SubscriptionStatusCanceled,
	})

	body, _ := json.Marshal(map[string]string{"plan_code": models.PlanCodeProYearly})
	w := h.callSubscribe(body, 11)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}

	// After subscribe: there are now two rows for this user.
	var rows []models.Subscription
	h.db.Where("user_id = ?", 11).Order("starts_at ASC, id ASC").Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("subscriptions = %d, want 2 (canceled + new active)", len(rows))
	}

	// The new active row is the most-recent.
	newest := rows[len(rows)-1]
	if newest.Status != models.SubscriptionStatusActive {
		t.Errorf("newest.status = %q, want active", newest.Status)
	}
	if newest.PlanCode != models.PlanCodeProYearly {
		t.Errorf("newest.plan_code = %q, want pro_yearly", newest.PlanCode)
	}

	// The original canceled row is preserved as-is.
	if rows[0].Status != models.SubscriptionStatusCanceled {
		t.Errorf("oldest.status = %q, want canceled (preserved on disk)", rows[0].Status)
	}
}

func TestSubscribe_RejectsFreePlan(t *testing.T) {
	h := newBillingHarness(t)
	h.seedUser(12, "sub3@example.com")

	body, _ := json.Marshal(map[string]string{"plan_code": "free"})
	w := h.callSubscribe(body, 12)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (free plan not subscribable)", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("INVALID_PLAN")) {
		t.Errorf("body should contain INVALID_PLAN: %s", w.Body.String())
	}
}

// Reject path. Anything outside the pro_monthly / pro_yearly set is
// rejected at the regex gate before reaching the plan lookup.
// (An unknown-but-shape-wise-correct plan row can't actually appear
// in the wild — the admin Plans API is the only writer — but this
// pins the failure mode for any future code that bypasses the gate.)
func TestSubscribe_RejectsBogusPlanCode(t *testing.T) {
	h := newBillingHarness(t)
	h.seedUser(13, "sub4@example.com")

	body, _ := json.Marshal(map[string]string{"plan_code": "ghost_plan_no_such_row"})
	w := h.callSubscribe(body, 13)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (regex gate)", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("INVALID_PLAN")) {
		t.Errorf("body should contain INVALID_PLAN, got: %s", w.Body.String())
	}
}

func TestSubscribe_EmptyBody_Returns400(t *testing.T) {
	h := newBillingHarness(t)
	h.seedUser(14, "sub5@example.com")

	// body=nil so ShouldBindJSON sees an empty body — Gin will
	// reject with a validation error because PlanCode is required.
	w := h.callSubscribe(nil, 14)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// ── helpers ──────────────────────────────────────────────────────────

func mustParseTime(s string) (t time.Time) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return
}

// keep the unused-import linter happy if config is unused in this file
var _ = config.Config{}
