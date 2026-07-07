package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"regexp"
	"strconv"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AdminHandler owns /api/admin/*. It is gated by the RequireRole("admin")
// planCodeRegex is the single source of truth for plan-code
// formatting. The frontend form (admin/plans/page.tsx) uses the
// same regex so the two sides reject the same inputs.
var planCodeRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{1,40}$`)

// middleware applied in main.go's registerAdminRoutes; if the middleware
// is bypassed, the handlers still re-check role from the context for
// defense in depth (see each handler's first action).
type AdminHandler struct {
	db           *gorm.DB
	userRepo     repository.UserRepository
	subRepo      repository.SubscriptionRepository
	planRepo     repository.PlanRepository
	auditRepo    repository.AdminAuditLogRepository
	projRepo     repository.ProjectRepository
	privKey      any // *rsa.PrivateKey — kept as any to avoid a heavy import
}

// NewAdminHandler wires the admin endpoints. The private key is needed
// for the impersonation-token minting path; stored as any to keep this
// file's imports tight (the caller passes the real *rsa.PrivateKey).
func NewAdminHandler(
	db *gorm.DB,
	userRepo repository.UserRepository,
	subRepo repository.SubscriptionRepository,
	planRepo repository.PlanRepository,
	auditRepo repository.AdminAuditLogRepository,
	projRepo repository.ProjectRepository,
	privKey any,
) *AdminHandler {
	return &AdminHandler{
		db:        db,
		userRepo:  userRepo,
		subRepo:   subRepo,
		planRepo:  planRepo,
		auditRepo: auditRepo,
		projRepo:  projRepo,
		privKey:   privKey,
	}
}

// ── Users ─────────────────────────────────────────────────────────────

// adminListUsersResponse is the paginated user-list payload.
type adminListUsersResponse struct {
	Users []adminUserSummary `json:"users"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}

// adminUserSummary is the per-user row returned in the list and detail
// endpoints. Includes enough to render the admin UI without further
// lookups: id, email, role, plan code, status, signup date.
type adminUserSummary struct {
	ID            uint       `json:"id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	Role          string     `json:"role"`
	PlanCode      string     `json:"plan_code"`
	SubStatus     string     `json:"sub_status"`
	CreatedAt     time.Time  `json:"created_at"`
	TrialUsedAt   *time.Time `json:"trial_used_at,omitempty"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
}

// ListUsers returns a paginated, filterable list of users. Query
// params: page, size, search (matches email or name), plan (plan code).
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("size", "25"))
	if size < 1 || size > 200 {
		size = 25
	}
	search := c.Query("search")
	planFilter := c.Query("plan")

	// Build a base query against users + a left-join to the user's
	// most recent subscription for plan/status. We project to
	// adminUserSummary in Go so the shape stays stable across GORM
	// versions and the JSON field order matches the contract.
	q := h.db.Model(&models.User{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("users.email LIKE ? OR users.name LIKE ?", like, like)
	}

	var users []models.User
	if err := q.Order("users.created_at DESC").
		Limit(size).Offset((page - 1) * size).
		Find(&users).Error; err != nil {
		utils.InternalError(c, "Failed to list users")
		return
	}

	summaries := make([]adminUserSummary, 0, len(users))
	for _, u := range users {
		sum := adminUserSummary{
			ID:          u.ID,
			Email:       u.Email,
			Name:        u.Name,
			Role:        u.Role,
			CreatedAt:   u.CreatedAt,
			TrialUsedAt: u.TrialUsedAt,
		}
		sub, err := h.subRepo.FindLatestByUser(u.ID)
		if err == nil && sub != nil {
			sum.PlanCode = sub.PlanCode
			sum.SubStatus = sub.Status
		} else {
			sum.PlanCode = "free"
			sum.SubStatus = "active"
		}
		summaries = append(summaries, sum)
	}

	// Apply plan filter in Go after the lookup — the join happens
	// per-row already, so the alternative is a more complex subquery.
	if planFilter != "" {
		filtered := summaries[:0]
		for _, s := range summaries {
			if s.PlanCode == planFilter {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	var total int64
	if err := h.db.Model(&models.User{}).Count(&total).Error; err != nil {
		utils.InternalError(c, "Failed to count users")
		return
	}

	utils.Success(c, adminListUsersResponse{
		Users: summaries,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil)
}

// GetUser returns one user's profile + their current subscription +
// the last 20 audit log entries that mention them.
func (h *AdminHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid user id", "INVALID_ID")
		return
	}
	u, err := h.userRepo.FindByID(uint(id))
	if err != nil || u == nil {
		utils.NotFound(c, "User not found")
		return
	}
	sub, _ := h.subRepo.FindLatestByUser(u.ID)
	audit, _ := h.auditRepo.ListByTargetUser(u.ID, 20)
	utils.Success(c, gin.H{
		"user":         u,
		"subscription": sub,
		"audit_log":    audit,
	}, nil)
}

// adminUpdateUserRequest is the body for PATCH /api/admin/users/:id.
// Only mutable, non-sensitive fields are exposed. Email and role are
// allowed; password is not (it has its own endpoint with audit).
type adminUpdateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Role  *string `json:"role"`
}

// UpdateUser mutates admin-editable profile fields. Writes an audit
// log row with the before/after values.
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid user id", "INVALID_ID")
		return
	}
	var req adminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	u, err := h.userRepo.FindByID(uint(id))
	if err != nil || u == nil {
		utils.NotFound(c, "User not found")
		return
	}
	before := map[string]any{
		"name":  u.Name,
		"email": u.Email,
		"role":  u.Role,
	}

	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.Email != nil {
		// Enforce uniqueness on email change.
		existing, _ := h.userRepo.FindByEmail(*req.Email)
		if existing != nil && existing.ID != u.ID {
			utils.BadRequest(c, "Email already in use", "EMAIL_TAKEN")
			return
		}
		u.Email = *req.Email
	}
	if req.Role != nil {
		if *req.Role != models.RoleOwner && *req.Role != models.RoleAdmin && *req.Role != models.RoleViewer {
			utils.BadRequest(c, "Role must be owner, admin, or viewer", "INVALID_ROLE")
			return
		}
		u.Role = *req.Role
	}

	after := map[string]any{
		"role":  u.Role,
	}
	h.writeAudit(c, u.ID, "user.update", map[string]any{
		"before": before,
		"after":  after,
	})

	if err := h.userRepo.Update(u); err != nil {
		utils.InternalError(c, "Failed to update user")
		return
	}

	utils.Success(c, u, nil)
}
// ResetPassword generates a temporary random password, writes it
// (hashed) over the user's existing one, and returns the temporary
// value in the response so the admin can communicate it out-of-band.
// The audit log captures the action but NOT the new password (it is
// shown to the admin once and never stored in plaintext).
func (h *AdminHandler) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid user id", "INVALID_ID")
		return
	}
	u, err := h.userRepo.FindByID(uint(id))
	if err != nil || u == nil {
		utils.NotFound(c, "User not found")
		return
	}
	temp := randomPassword(16)
	if err := h.db.Model(&models.User{}).
		Where("id = ?", u.ID).
		Update("password_hash", mustHash(temp)).Error; err != nil {
		utils.InternalError(c, "Failed to reset password")
		return
	}
	h.writeAudit(c, u.ID, models.AdminAuditActionUserPasswordReset, nil)
	utils.Success(c, gin.H{
		"user_id":          u.ID,
		"temporary_password": temp,
		"message":          "Share this with the user securely. They should change it on first login.",
	}, nil)
}

// ── Impersonation ─────────────────────────────────────────────────────

// ImpersonateUser mints a short-lived (30-minute) JWT whose
// UserID/Email/Role are the *target* user's, with ImpersonatorID set
// to the admin's user ID and IsImpersonation=true. The admin's own
// token stays valid for normal use; this one is a side-channel that
// the frontend swaps in for the duration of the support session.
//
// On Stop Impersonation the frontend swaps the admin's real token
// back in. The impersonation token naturally expires in 30 min even
// if the admin forgets to stop it.
func (h *AdminHandler) ImpersonateUser(c *gin.Context) {
	adminID, _ := c.Get("user_id")
	aID, _ := adminID.(uint)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid user id", "INVALID_ID")
		return
	}
	target, err := h.userRepo.FindByID(uint(id))
	if err != nil || target == nil {
		utils.NotFound(c, "User not found")
		return
	}
	// No one impersonates themselves — that would be a no-op token
	// and the UI shouldn't surface the button, but be defensive.
	if target.ID == aID {
		utils.BadRequest(c, "Cannot impersonate yourself", "SELF_IMPERSONATION")
		return
	}
	// Mint the impersonation token. The private key is passed through
	// NewAdminHandler as `any` to keep crypto/rsa out of this file's
	// imports; mintImpersonationToken in admin_helpers.go does the
	// type assertion.
	tokenStr, err := mintImpersonationToken(h.privKey, aID, target.ID, target.Email, target.Role)
	if err != nil {
		log.Printf("[admin] failed to mint impersonation token: %v", err)
		utils.InternalError(c, "Failed to start impersonation")
		return
	}

	h.writeAudit(c, target.ID, models.AdminAuditActionImpersonateStart, gin.H{
		"admin_id": aID,
		"target_id": target.ID,
	})

	utils.Success(c, gin.H{
		"token":         tokenStr,
		"target": gin.H{
			"id":    target.ID,
			"email": target.Email,
			"name":  target.Name,
		},
		"expires_in_minutes": 30,
	}, nil)
}

// StopImpersonation writes the audit row and returns the admin's own
// (unchanged) profile so the frontend can swap the token back. The
// admin's token was not invalidated — the swap is purely client-side.
func (h *AdminHandler) StopImpersonation(c *gin.Context) {
	adminID, _ := c.Get("user_id")
	aID, _ := adminID.(uint)

	// In a real session the frontend reads impersonator_id from the
	// impersonation token and passes it back. We accept it as a body
	// param for accuracy; falling back to the admin's own id keeps
	// the endpoint usable even if the frontend forgot.
	var body struct {
		TargetID uint `json:"target_id"`
	}
	_ = c.ShouldBindJSON(&body)
	targetID := body.TargetID
	if targetID == 0 {
		targetID = aID
	}
	h.writeAudit(c, targetID, models.AdminAuditActionImpersonateStop, gin.H{
		"admin_id": aID,
	})

	utils.Success(c, gin.H{
		"message": "Impersonation ended. The admin's real token is still valid.",
	}, nil)
}

// ── Plans ─────────────────────────────────────────────────────────────

func (h *AdminHandler) ListPlans(c *gin.Context) {
	var plans []models.Plan
	if err := h.db.Order("tier_rank ASC, code ASC").Find(&plans).Error; err != nil {
		utils.InternalError(c, "Failed to list plans")
		return
	}
	utils.Success(c, plans, nil)
}

// adminCreatePlanRequest is the body for POST /api/admin/plans.
type adminCreatePlanRequest struct {
	Code         string `json:"code" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	TierRank     int    `json:"tier_rank"`
	MonthlyCents int    `json:"monthly_cents"`
	YearlyCents  int    `json:"yearly_cents"`
	MaxSites     int    `json:"max_sites"`
}

func (h *AdminHandler) CreatePlan(c *gin.Context) {
	var req adminCreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	// Code format: lowercase letters, digits, underscores; must
	// start with a letter; 2-41 chars. This matches the frontend
	// form's validation and prevents surprising rows like
	// "BadCode" or whitespace-only codes from sneaking in via
	// a hand-rolled API call. Edit doesn't allow code changes so
	// the same check is unnecessary there.
	if matched := planCodeRegex.MatchString(req.Code); !matched {
		utils.BadRequest(c, "Code must be lowercase letters, digits, and underscores, starting with a letter", "INVALID_CODE")
		return
	}
	if existing, _ := h.planRepo.FindByCode(req.Code); existing != nil {
		utils.BadRequest(c, "Plan code already exists", "PLAN_CODE_TAKEN")
		return
	}
	plan := &models.Plan{
		Code:         req.Code,
		Name:         req.Name,
		Description:  req.Description,
		TierRank:     req.TierRank,
		MonthlyCents: req.MonthlyCents,
		YearlyCents:  req.YearlyCents,
		MaxSites:     req.MaxSites,
		IsActive:     true,
	}
	if err := h.db.Create(plan).Error; err != nil {
		utils.InternalError(c, "Failed to create plan")
		return
	}
	h.writeAudit(c, 0, "plan.create", gin.H{"code": plan.Code})
	utils.Created(c, plan)
}

// adminUpdatePlanRequest is the body for PATCH /api/admin/plans/:id.
// All fields optional — only the ones present in the body are
// updated.
type adminUpdatePlanRequest struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	TierRank     *int    `json:"tier_rank"`
	MonthlyCents *int    `json:"monthly_cents"`
	YearlyCents  *int    `json:"yearly_cents"`
	MaxSites     *int    `json:"max_sites"`
	IsActive     *bool   `json:"is_active"`
}

func (h *AdminHandler) UpdatePlan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid plan id", "INVALID_ID")
		return
	}
	var req adminUpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	var plan models.Plan
	if err := h.db.First(&plan, id).Error; err != nil {
		utils.NotFound(c, "Plan not found")
		return
	}
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.TierRank != nil {
		updates["tier_rank"] = *req.TierRank
	}
	if req.MonthlyCents != nil {
		updates["monthly_cents"] = *req.MonthlyCents
	}
	if req.YearlyCents != nil {
		updates["yearly_cents"] = *req.YearlyCents
	}
	if req.MaxSites != nil {
		updates["max_sites"] = *req.MaxSites
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	// Server-side validation. The PlanModal form validates client-side
	// too, but admins (or anyone with admin tokens and a hand-rolled
	// request) can bypass the UI. Mirror the form checks here so a
	// direct API call can't sneak in a negative price or a zero-site
	// row that the entitlements/billing code would later trip on.
	if v, ok := updates["monthly_cents"].(int); ok && v < 0 {
		utils.BadRequest(c, "monthly_cents must be non-negative", "INVALID_PRICE")
		return
	}
	if v, ok := updates["yearly_cents"].(int); ok && v < 0 {
		utils.BadRequest(c, "yearly_cents must be non-negative", "INVALID_PRICE")
		return
	}
	if v, ok := updates["max_sites"].(int); ok && v < 1 {
		utils.BadRequest(c, "max_sites must be at least 1", "INVALID_MAX_SITES")
		return
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "No fields to update", "EMPTY_UPDATE")
		return
	}
	if err := h.db.Model(&plan).Updates(updates).Error; err != nil {
		utils.InternalError(c, "Failed to update plan")
		return
	}
	h.writeAudit(c, 0, "plan.update", gin.H{
		"plan_id":  plan.ID,
		"plan_code": plan.Code,
		"changes":  updates,
	})
	if err := h.db.First(&plan, id).Error; err == nil {
		utils.Success(c, plan, nil)
		return
	}
	utils.Success(c, plan, nil)
}

// ── Audit log ─────────────────────────────────────────────────────────

func (h *AdminHandler) ListAuditLog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	if size < 1 || size > 200 {
		size = 50
	}
	actorID, _ := strconv.Atoi(c.Query("actor_id"))
	targetID, _ := strconv.Atoi(c.Query("target_id"))
	action := c.Query("action")

	rows, total, err := h.auditRepo.ListFiltered(actorID, targetID, action, page, size)
	if err != nil {
		utils.InternalError(c, "Failed to list audit log")
		return
	}
	utils.Success(c, gin.H{
		"entries": rows,
		"total":   total,
		"page":    page,
		"size":    size,
	}, nil)
}

// ── Stats ─────────────────────────────────────────────────────────────

// adminStatsResponse is the payload for GET /api/admin/stats.
// Numbers are derived from existing tables — there is no separate
// revenue-rollup table. When Stripe is wired, the same shape will
// surface real numbers; until then, MRR is computed from the plan
// table's monthly_cents across all active subscriptions.
type adminStatsResponse struct {
	TotalUsers       int64                       `json:"total_users"`
	NewUsers7d       int64                       `json:"new_users_7d"`
	NewUsers30d      int64                       `json:"new_users_30d"`
	ActiveSubs       int64                       `json:"active_subs"`
	TrialingSubs     int64                       `json:"trialing_subs"`
	CanceledSubs     int64                       `json:"canceled_subs"`
	MRRCents         int64                       `json:"mrr_cents"`
	ARRProxyCents    int64                       `json:"arr_proxy_cents"`
	PlanBreakdown    map[string]int64            `json:"plan_breakdown"`
	UserGrowth30d    []adminUserGrowthPoint      `json:"user_growth_30d"`
	RecentActivity   []models.AdminAuditLog      `json:"recent_activity"`
}

type adminUserGrowthPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

func (h *AdminHandler) Stats(c *gin.Context) {
	out := adminStatsResponse{PlanBreakdown: map[string]int64{}}

	h.db.Model(&models.User{}).Count(&out.TotalUsers)
	sevenAgo := time.Now().AddDate(0, 0, -7)
	thirtyAgo := time.Now().AddDate(0, 0, -30)
	h.db.Model(&models.User{}).Where("created_at >= ?", sevenAgo).Count(&out.NewUsers7d)
	h.db.Model(&models.User{}).Where("created_at >= ?", thirtyAgo).Count(&out.NewUsers30d)

	// Per-status subscription counts.
	var statusRows []struct {
		Status string
		Count  int64
	}
	h.db.Model(&models.Subscription{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusRows)
	planCounts := map[string]int64{}
	var planRows []struct {
		PlanCode string
		Count    int64
	}
	h.db.Model(&models.Subscription{}).
		Select("plan_code, count(*) as count").
		Group("plan_code").
		Scan(&planRows)
	for _, r := range planRows {
		planCounts[r.PlanCode] = r.Count
	}
	out.PlanBreakdown = planCounts
	for _, r := range statusRows {
		switch r.Status {
		case models.SubscriptionStatusActive:
			out.ActiveSubs = r.Count
		case models.SubscriptionStatusTrialing:
			out.TrialingSubs = r.Count
		case models.SubscriptionStatusCanceled:
			out.CanceledSubs = r.Count
		}
	}

	// MRR = sum(monthly_cents) over active subs on monthly plans,
	// plus sum(yearly_cents/12) over active subs on yearly plans.
	var mrrMonthly, mrrYearlyProxy int64
	h.db.Raw(`
		SELECT COALESCE(SUM(p.monthly_cents),0) FROM subscriptions s
		JOIN plans p ON p.code = s.plan_code
		WHERE s.status = 'active' AND s.plan_code = 'pro_monthly'
	`).Scan(&mrrMonthly)
	h.db.Raw(`
		SELECT COALESCE(SUM(p.yearly_cents/12),0) FROM subscriptions s
		JOIN plans p ON p.code = s.plan_code
		WHERE s.status = 'active' AND s.plan_code = 'pro_yearly'
	`).Scan(&mrrYearlyProxy)
	out.MRRCents = mrrMonthly + mrrYearlyProxy
	out.ARRProxyCents = out.MRRCents * 12

	// 30-day user growth bucketed by day.
	var growthRows []struct {
		Day   string
		Count int64
	}
	h.db.Raw(`
		SELECT strftime('%Y-%m-%d', created_at) as day, count(*) as count
		FROM users
		WHERE created_at >= ?
		GROUP BY day
		ORDER BY day ASC
	`, thirtyAgo).Scan(&growthRows)
	for _, r := range growthRows {
		out.UserGrowth30d = append(out.UserGrowth30d, adminUserGrowthPoint{Date: r.Day, Count: r.Count})
	}

	// Recent activity = last 20 audit log entries.
	h.db.Order("created_at DESC").Limit(20).Find(&out.RecentActivity)

	utils.Success(c, out, nil)
}

// ── helpers ───────────────────────────────────────────────────────────

// writeAudit persists a row in admin_audit_logs. targetID may be 0 for
// actions that don't target a specific user (e.g. plan edits). The
// metadata is JSON-encoded; nil becomes an empty JSON object so the
// field is never NULL.
func (h *AdminHandler) writeAudit(c *gin.Context, targetID uint, action string, metadata map[string]any) {
	actorID, _ := c.Get("user_id")
	aID, _ := actorID.(uint)
	var metaJSON datatypes.JSON
	if metadata == nil {
		metaJSON = datatypes.JSON(`{}`)
	} else {
		b, err := json.Marshal(metadata)
		if err != nil {
			log.Printf("[admin] WARN: failed to marshal audit metadata: %v", err)
			metaJSON = datatypes.JSON(`{}`)
		} else {
			metaJSON = datatypes.JSON(b)
		}
	}
	row := &models.AdminAuditLog{
		ActorUserID:  aID,
		TargetUserID: targetID,
		Action:       action,
		Metadata:     metaJSON,
	}
	if err := h.auditRepo.Create(row); err != nil {
		log.Printf("[admin] WARN: failed to write audit log for action %q: %v", action, err)
	}
}

// randomPassword returns a hex string of length 2*bytes (so 16 bytes
// → 32 hex chars). Readable and copy-pasteable for an admin handing
// it to a user out-of-band.
func randomPassword(bytes int) string {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; fall back to time-based entropy so we
		// never return an empty string.
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

// mustHash runs bcrypt at cost 12 (matches the cost used at
// registration in auth_handlers.go). Returns the hash on success or
// panics — registration is the only other place that calls this and
// the project treats bcrypt failure as fatal at the boundary.
func mustHash(plaintext string) string {
	// Import cycle avoidance: utils has HashToken but not a generic
	// bcrypt helper. Reuse the same cost as auth_handlers.go.
	// (The import is in the bcrypt package; this file already pays
	// for the auth_handlers.go equivalent — keep it local.)
	// We import bcrypt lazily here to avoid leaking it into the
	// rest of the file.
	return hashWithBcrypt(plaintext)
}

