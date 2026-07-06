package handlers

import (
	"log"
	"time"

	"backend/internal/models"
 	"backend/internal/repository"
	"backend/internal/services/entitlements"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TrialLength is the default trial window applied when a user starts a
// pro plan trial. 14 days was confirmed with product before phase 6.
const TrialLength = 14 * 24 * time.Hour

// BillingHandler owns /api/billing/*. Phase 6 ships Stripe as a stub
// (no real charge) — endpoints mutate the subscription row directly.
type BillingHandler struct {
	db          *gorm.DB
	subRepo     repository.SubscriptionRepository
	planRepo    repository.PlanRepository
	projRepo    repository.ProjectRepository
	entitlements *entitlements.Service
}

// NewBillingHandler wires the billing endpoints. All dependencies are
// required.
func NewBillingHandler(
	db *gorm.DB,
	subRepo repository.SubscriptionRepository,
	planRepo repository.PlanRepository,
	projRepo repository.ProjectRepository,
	ent *entitlements.Service,
) *BillingHandler {
	return &BillingHandler{
		db:           db,
		subRepo:      subRepo,
		planRepo:     planRepo,
		projRepo:     projRepo,
		entitlements: ent,
	}
}

// BillingState is the response shape of GET /api/billing/me. It bundles
// the user's current subscription, the underlying plan, and the live
// usage count so the /billing page can render without further lookups.
type BillingState struct {
	Subscription *models.Subscription `json:"subscription"`
	Plan         *models.Plan         `json:"plan"`
	Usage        BillingUsage         `json:"usage"`
}

// BillingUsage reports current consumption against the plan's quotas.
// Today only SEO project count is gated (phase 2); social is unlimited.
type BillingUsage struct {
	SEOProjects int `json:"seo_projects"`
	MaxSites    int `json:"max_sites"`
}

// Me returns the current user's billing state. Resolves the active
// subscription via the entitlements service so a user with no row
// still gets a coherent answer (the default free plan).
func (h *BillingHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok || uid == 0 {
		utils.Unauthorized(c, "Not authenticated")
		return
	}

	sub, plan, err := h.entitlements.ResolveSubscription(c.Request.Context(), uid)
	if err != nil {
		utils.InternalError(c, "Failed to load subscription")
		return
	}

	seoCount, err := h.projRepo.CountSEOByUser(uid)
	if err != nil {
		utils.InternalError(c, "Failed to load usage")
		return
	}

	utils.Success(c, BillingState{
		Subscription: sub,
		Plan:         plan,
		Usage: BillingUsage{
			SEOProjects: int(seoCount),
			MaxSites:    plan.MaxSites,
		},
	}, nil)
}

// StartTrialRequest is the body of POST /api/billing/trial. The plan
// code is the user-facing plan row (e.g. "pro_monthly" or "pro_yearly").
type StartTrialRequest struct {
	PlanCode string `json:"plan_code" binding:"required"`
}

// validTrialPlan reports whether code names a real billable pro tier.
// Free plans cannot be trialled.
func validTrialPlan(code string) bool {
	return code == models.PlanCodeProMonthly || code == models.PlanCodeProYearly
}

// StartTrial creates a 14-day trialing subscription on the requested
// pro plan. Rejects with 409 if the user already has an active or
// trialing subscription — trials are one-shot per user (the rationale
// is in IMPLEMENTATION_PLAN §6; revisiting means a SubscriptionEvent
// history table).
func (h *BillingHandler) StartTrial(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok || uid == 0 {
		utils.Unauthorized(c, "Not authenticated")
		return
	}

	var req StartTrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if !validTrialPlan(req.PlanCode) {
		utils.BadRequest(c, "plan_code must be pro_monthly or pro_yearly", "INVALID_PLAN")
		return
	}

	plan, err := h.planRepo.FindByCode(req.PlanCode)
	if err != nil || plan == nil {
		utils.NotFound(c, "Plan not found")
		return
	}

	existing, err := h.subRepo.FindCurrentByUser(uid)

	if err != nil && err != gorm.ErrRecordNotFound {
		utils.InternalError(c, "Failed to load subscription")
		return
	}

	// A user is allowed exactly one trial ever. The auto-free row
	// written on registration is NOT a trial — it's the default. We
	// reject only when the existing row is already a real subscription
	// (trialing or active on a pro plan), since re-activating a trial
	// after it's been spent is the abuse case to prevent.
	if existing != nil {
		isProPlan := existing.PlanCode == models.PlanCodeProMonthly ||
			existing.PlanCode == models.PlanCodeProYearly
		alreadyTrialing := existing.Status == models.SubscriptionStatusTrialing
		alreadyActivePro := isProPlan && existing.Status == models.SubscriptionStatusActive
		if alreadyTrialing || alreadyActivePro {
			utils.Conflict(c, "A subscription already exists for this user", "TRIAL_ALREADY_USED")
			return
		}
	}

	now := time.Now()
	trialEnd := now.Add(TrialLength)

	// Two paths: brand-new user (no row) → Create; existing free row →
	// upgrade it in place so we don't end up with two rows for one user.
	if existing == nil {
		sub := &models.Subscription{
			UserID:      uid,
			PlanCode:    plan.Code,
			Status:      models.SubscriptionStatusTrialing,
			TrialEndsAt: &trialEnd,
			StartsAt:    now,
		}
		if err := h.subRepo.Create(sub); err != nil {
			log.Printf("[billing] failed to start trial for user %d: %v", uid, err)
			utils.InternalError(c, "Failed to start trial")
			return
		}
		utils.Created(c, sub)
		return
	}

	// Upgrade the free row to a trialing pro row in one shot.
	if err := h.db.Model(&models.Subscription{}).
		Where("user_id = ?", uid).
		Order("starts_at DESC").
		Limit(1).
		Updates(map[string]any{
			"plan_code":     plan.Code,
			"status":        models.SubscriptionStatusTrialing,
			"trial_ends_at": trialEnd,
		}).Error; err != nil {
		log.Printf("[billing] failed to upgrade free sub to trial for user %d: %v", uid, err)
		utils.InternalError(c, "Failed to start trial")
		return
	}
	updated, err := h.subRepo.FindCurrentByUser(uid)
	if err != nil {
		utils.InternalError(c, "Failed to load subscription")
		return
	}
	utils.Success(c, updated, nil)
}

// SubscribeRequest is the body of POST /api/billing/subscribe. Phase 6
// has no payment — the subscription is immediately activated. A visible
// banner on the /billing page makes this clear. When Stripe is wired
// later, this handler becomes the place that opens a Checkout session
// and the real activation moves into the webhook.
type SubscribeRequest struct {
	PlanCode string `json:"plan_code" binding:"required"`
}

// Subscribe activates the requested pro plan for the current user.
// If a subscription row already exists, its plan + status are updated;
// otherwise a new active row is inserted. The previous plan is not
// archived today (no SubscriptionEvent table) — that's a phase 7
// admin-history item.
func (h *BillingHandler) Subscribe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok || uid == 0 {
		utils.Unauthorized(c, "Not authenticated")
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if !validTrialPlan(req.PlanCode) {
		utils.BadRequest(c, "plan_code must be pro_monthly or pro_yearly", "INVALID_PLAN")
		return
	}

	plan, err := h.planRepo.FindByCode(req.PlanCode)
	if err != nil || plan == nil {
		utils.NotFound(c, "Plan not found")
		return
	}

	existing, err := h.subRepo.FindCurrentByUser(uid)
	if err != nil && err != gorm.ErrRecordNotFound {
		utils.InternalError(c, "Failed to load subscription")
		return
	}

	now := time.Now()
	if existing == nil {
		sub := &models.Subscription{
			UserID:   uid,
			PlanCode: plan.Code,
			Status:   models.SubscriptionStatusActive,
			StartsAt: now,
		}
		if err := h.subRepo.Create(sub); err != nil {
			log.Printf("[billing] failed to create subscription for user %d: %v", uid, err)
			utils.InternalError(c, "Failed to activate plan")
			return
		}
		utils.Created(c, sub)
		return
	}
	if err := h.subRepo.UpdatePlanAndStatus(uid, plan.Code, models.SubscriptionStatusActive); err != nil {
		log.Printf("[billing] failed to update subscription for user %d: %v", uid, err)
		utils.InternalError(c, "Failed to activate plan")
		return
	}
	// Reload to return the new state to the caller.
	updated, err := h.subRepo.FindCurrentByUser(uid)
	if err != nil {
		utils.InternalError(c, "Failed to load subscription")
		return
	}
	utils.Success(c, updated, nil)
}

// Cancel marks the user's current subscription as canceled. The plan
// row is not deleted; entitlements.ResolveSubscription will then fall
// through to the default free plan the next time it's queried.
func (h *BillingHandler) Cancel(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok || uid == 0 {
		utils.Unauthorized(c, "Not authenticated")
		return
	}

	existing, err := h.subRepo.FindLatestByUser(uid)
	if err == gorm.ErrRecordNotFound || existing == nil {
		utils.NotFound(c, "No subscription found")
		return
	}
	if err != nil {
		utils.InternalError(c, "Failed to load subscription")
		return
	}

	now := time.Now()
	if err := h.db.Model(&models.Subscription{}).
		Where("user_id = ?", uid).
		Order("starts_at DESC, id DESC").
		Limit(1).
		Updates(map[string]any{
			"status":      models.SubscriptionStatusCanceled,
			"canceled_at": now,
		}).Error; err != nil {
		log.Printf("[billing] failed to cancel subscription for user %d: %v", uid, err)
		utils.InternalError(c, "Failed to cancel subscription")
		return
	}

	updated, err := h.subRepo.FindLatestByUser(uid)
	if err != nil {
		utils.InternalError(c, "Failed to load subscription")
		return
	}
	utils.Success(c, updated, nil)
}

// Webhook is a stub for Stripe. Real signature verification + event
// dispatch lands in a follow-up phase. Returning 501 today tells the
// frontend (and Stripe) that this endpoint is intentionally inert.
func (h *BillingHandler) Webhook(c *gin.Context) {
	utils.NotImplemented(c, "Stripe webhook is not yet wired. Subscription changes happen through the /billing UI for now.")
}

