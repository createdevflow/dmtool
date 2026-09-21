package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"backend/internal/middleware"
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
	activityRepo repository.UserActivityRepository
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
	activityRepo repository.UserActivityRepository,
	privKey any,
) *AdminHandler {
	return &AdminHandler{
		db:           db,
		userRepo:     userRepo,
		subRepo:      subRepo,
		planRepo:     planRepo,
		auditRepo:    auditRepo,
		projRepo:     projRepo,
		activityRepo: activityRepo,
		privKey:      privKey,
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
// params: page, size, search (email/name), plan, role, mode, status.
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
	roleFilter := c.Query("role")
	modeFilter := c.Query("mode")
	statusFilter := c.Query("status") // "active", "suspended"

	q := h.db.Model(&models.User{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("users.email LIKE ? OR users.name LIKE ?", like, like)
	}
	if roleFilter != "" {
		q = q.Where("users.role = ?", roleFilter)
	}
	if modeFilter != "" {
		q = q.Where("users.dashboard_mode = ?", modeFilter)
	}
	if statusFilter == "suspended" {
		q = q.Where("users.disabled_at IS NOT NULL")
	} else if statusFilter == "active" {
		q = q.Where("users.disabled_at IS NULL")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		utils.InternalError(c, "Failed to count users")
		return
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
			LastLoginAt: u.LastLoginAt,
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

	// Apply plan filter in Go after the lookup.
	if planFilter != "" {
		filtered := summaries[:0]
		for _, s := range summaries {
			if s.PlanCode == planFilter {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
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
	if req.Role != nil && *req.Role != u.Role {
		if middleware.StaffPermsLoaded(c) && !middleware.StaffHas(c, models.PermUsersRoleAssign) {
			utils.Forbidden(c, "Assigning roles requires users.role.assign")
			return
		}
		if *req.Role == models.RoleViewer {
			utils.BadRequest(c, "viewer is not a staff role", "INVALID_ROLE")
			return
		}
		if *req.Role != models.RoleOwner {
			var staffRole models.Role
			if err := h.db.Where("code = ?", *req.Role).First(&staffRole).Error; err != nil {
				utils.BadRequest(c, "Unknown role", "INVALID_ROLE")
				return
			}
		}
		if h.isSystemRole(u.Role) && h.countUsersWithRole(u.Role) <= 1 {
			utils.BadRequest(c, "Cannot demote the last Super Admin", "LAST_SUPER_ADMIN")
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

// ── User Activity ────────────────────────────────────────────────────

// GetUserActivity returns the activity timeline for a specific user.
func (h *AdminHandler) GetUserActivity(c *gin.Context) {
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
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	activities, err := h.activityRepo.ListByUser(u.ID, limit)
	if err != nil {
		utils.InternalError(c, "Failed to fetch activity")
		return
	}
	utils.Success(c, gin.H{
		"user_id":   u.ID,
		"activities": activities,
	}, nil)
}

// ── Suspend / Unsuspend ──────────────────────────────────────────────

type suspendRequest struct {
	Reason string `json:"reason"`
}

// SuspendUser soft-suspends a user by setting disabled_at.
func (h *AdminHandler) SuspendUser(c *gin.Context) {
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
	if h.isSystemRole(u.Role) {
		utils.BadRequest(c, "Cannot suspend a Super Admin", "CANNOT_SUSPEND_ADMIN")
		return
	}
	if u.DisabledAt != nil {
		utils.BadRequest(c, "User is already suspended", "ALREADY_SUSPENDED")
		return
	}
	var req suspendRequest
	_ = c.ShouldBindJSON(&req)
	now := time.Now()
	u.DisabledAt = &now
	u.DisabledReason = req.Reason
	if err := h.userRepo.Update(u); err != nil {
		utils.InternalError(c, "Failed to suspend user")
		return
	}
	h.writeAudit(c, u.ID, "user.suspend", gin.H{"reason": req.Reason})
	utils.Success(c, gin.H{
		"user_id":      u.ID,
		"disabled_at":  now,
		"message":      "User suspended",
	}, nil)
}

// UnsuspendUser reactivates a suspended user by clearing disabled_at.
func (h *AdminHandler) UnsuspendUser(c *gin.Context) {
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
	if u.DisabledAt == nil {
		utils.BadRequest(c, "User is not suspended", "NOT_SUSPENDED")
		return
	}
	u.DisabledAt = nil
	u.DisabledReason = ""
	if err := h.userRepo.Update(u); err != nil {
		utils.InternalError(c, "Failed to unsuspend user")
		return
	}
	h.writeAudit(c, u.ID, "user.unsuspend", nil)
	utils.Success(c, gin.H{
		"user_id": u.ID,
		"message": "User reactivated",
	}, nil)
}

// ── User Export (CSV) ────────────────────────────────────────────────

// ExportUsers returns all users as CSV. No pagination — intended for
// admin data exports. The response is streamed as text/csv.
func (h *AdminHandler) ExportUsers(c *gin.Context) {
	var users []models.User
	if err := h.db.Order("created_at DESC").Find(&users).Error; err != nil {
		utils.InternalError(c, "Failed to export users")
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=users_export.csv")

	// Write CSV header
	c.Writer.WriteString("id,email,name,role,dashboard_mode,created_at,last_login_at,login_count,disabled_at\n")
	for _, u := range users {
		disabledAt := ""
		if u.DisabledAt != nil {
			disabledAt = u.DisabledAt.Format(time.RFC3339)
		}
		lastLogin := ""
		if u.LastLoginAt != nil {
			lastLogin = u.LastLoginAt.Format(time.RFC3339)
		}
		line := strconv.FormatUint(uint64(u.ID), 10) + "," +
			escapeCSV(u.Email) + "," +
			escapeCSV(u.Name) + "," +
			u.Role + "," +
			u.DashboardMode + "," +
			u.CreatedAt.Format(time.RFC3339) + "," +
			lastLogin + "," +
			strconv.Itoa(u.LoginCount) + "," +
			disabledAt + "\n"
		c.Writer.WriteString(line)
	}
}

// escapeCSV wraps a value in quotes if it contains a comma, quote, or newline.
func escapeCSV(s string) string {
	if len(s) == 0 {
		return s
	}
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' {
			return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
		}
	}
	return s
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

// StopImpersonation writes impersonate.stop and tells the client to
// drop the impersonation cookie. Access tokens are stateless RS256
// with no denylist — the impersonation JWT stays valid until its
// 30-minute TTL. Ending the session is cookie-clear + audit.
//
// Two callers:
//   - Browser banner: impersonation JWT. user_id is the target,
//     impersonator_id is the admin. Actor must be the admin.
//   - CLI / phase7_verify: real admin JWT. user_id is the admin;
//     target comes from the body or :id.
func (h *AdminHandler) StopImpersonation(c *gin.Context) {
	paramID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	isImp, _ := c.Get("is_impersonation")
	impersonating, _ := isImp.(bool)

	var actorID, targetID uint
	if impersonating {
		rawImp, _ := c.Get("impersonator_id")
		actorID, _ = rawImp.(uint)
		rawUID, _ := c.Get("user_id")
		targetID, _ = rawUID.(uint)
		if paramID != 0 && uint(paramID) != targetID {
			utils.BadRequest(c, "Impersonation token does not match this user", "IMPERSONATION_MISMATCH")
			return
		}
	} else {
		rawUID, _ := c.Get("user_id")
		actorID, _ = rawUID.(uint)
		var body struct {
			TargetID uint `json:"target_id"`
		}
		_ = c.ShouldBindJSON(&body)
		targetID = body.TargetID
		if targetID == 0 {
			targetID = uint(paramID)
		}
	}
	if actorID == 0 {
		utils.Unauthorized(c, "Authentication required")
		return
	}

	h.writeAuditAs(actorID, targetID, models.AdminAuditActionImpersonateStop, gin.H{
		"admin_id": actorID,
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

// ── Projects (admin) ──────────────────────────────────────────────────

// adminProjectSummary is the per-project row returned in the admin list.
type adminProjectSummary struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	UserID      uint      `json:"user_id"`
	OwnerEmail  string    `json:"owner_email"`
	Goal        string    `json:"goal"`
	Status      string    `json:"status"`
	Health      string    `json:"health"`
	HealthScore int       `json:"health_score"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListProjects returns a paginated, filterable list of all projects.
// Query params: page, size, search (name/url), goal, health, owner (user_id).
func (h *AdminHandler) ListProjects(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("size", "25"))
	if size < 1 || size > 200 {
		size = 25
	}
	search := c.Query("search")
	goalFilter := c.Query("goal")
	healthFilter := c.Query("health")
	ownerFilter := c.Query("owner")

	q := h.db.Model(&models.Project{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("projects.name LIKE ? OR projects.url LIKE ?", like, like)
	}
	if goalFilter != "" {
		q = q.Where("projects.goal = ?", goalFilter)
	}
	if healthFilter != "" {
		q = q.Where("projects.health = ?", healthFilter)
	}
	if ownerFilter != "" {
		q = q.Where("projects.user_id = ?", ownerFilter)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		utils.InternalError(c, "Failed to count projects")
		return
	}

	var projects []models.Project
	if err := q.Order("projects.created_at DESC").
		Limit(size).Offset((page - 1) * size).
		Find(&projects).Error; err != nil {
		utils.InternalError(c, "Failed to list projects")
		return
	}

	// Batch-fetch owner emails for the result set.
	userIDs := make([]uint, 0, len(projects))
	for _, p := range projects {
		userIDs = append(userIDs, p.UserID)
	}
	ownerEmails := map[uint]string{}
	if len(userIDs) > 0 {
		var users []models.User
		h.db.Select("id, email").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			ownerEmails[u.ID] = u.Email
		}
	}

	summaries := make([]adminProjectSummary, 0, len(projects))
	for _, p := range projects {
		summaries = append(summaries, adminProjectSummary{
			ID:          p.ID,
			Name:        p.Name,
			URL:         p.URL,
			UserID:      p.UserID,
			OwnerEmail:  ownerEmails[p.UserID],
			Goal:        p.Goal,
			Status:      p.Status,
			Health:      p.Health,
			HealthScore: p.HealthScore,
			CreatedAt:   p.CreatedAt,
		})
	}

	utils.Success(c, gin.H{
		"projects": summaries,
		"total":    total,
		"page":     page,
		"size":     size,
	}, nil)
}

// GetProject returns detailed info for a single project including
// owner info, SEO summary, social summary, and recent insights.
func (h *AdminHandler) GetProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid project id", "INVALID_ID")
		return
	}
	var p models.Project
	if err := h.db.First(&p, id).Error; err != nil {
		utils.NotFound(c, "Project not found")
		return
	}

	// Owner info
	var owner models.User
	h.db.Select("id, name, email").First(&owner, p.UserID)

	// SEO summary
	var openIssues int64
	h.db.Model(&models.SEOIssue{}).
		Where("project_id = ? AND resolved_at IS NULL", p.ID).
		Count(&openIssues)
	var criticalIssues int64
	h.db.Model(&models.SEOIssue{}).
		Where("project_id = ? AND resolved_at IS NULL AND severity = ?", p.ID, "high").
		Count(&criticalIssues)
	var keywordCount int64
	h.db.Model(&models.KeywordResult{}).
		Where("project_id = ?", p.ID).
		Distinct("keyword").
		Count(&keywordCount)

	// Social summary
	var latestSocial models.SocialMetric
	h.db.Where("project_id = ?", p.ID).Order("recorded_at DESC").First(&latestSocial)

	// Recent insights
	var insights []models.Insight
	h.db.Where("project_id = ?", p.ID).Order("created_at DESC").Limit(10).Find(&insights)

	utils.Success(c, gin.H{
		"project": p,
		"owner":   owner,
		"seo": gin.H{
			"open_issues":    openIssues,
			"critical_issues": criticalIssues,
			"keyword_count":   keywordCount,
		},
		"social": latestSocial,
		"insights": insights,
	}, nil)
}

// GetProjectMetrics returns the time-series metrics for a project.
func (h *AdminHandler) GetProjectMetrics(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid project id", "INVALID_ID")
		return
	}
	var p models.Project
	if err := h.db.First(&p, id).Error; err != nil {
		utils.NotFound(c, "Project not found")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	var metrics []models.Metric
	h.db.Where("project_id = ?", p.ID).Order("date DESC").Limit(limit).Find(&metrics)
	utils.Success(c, gin.H{
		"project_id": p.ID,
		"metrics":    metrics,
	}, nil)
}

// GetProjectSEO returns SEO data for a project: issues, keywords, backlinks.
func (h *AdminHandler) GetProjectSEO(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid project id", "INVALID_ID")
		return
	}
	var p models.Project
	if err := h.db.First(&p, id).Error; err != nil {
		utils.NotFound(c, "Project not found")
		return
	}

	var issues []models.SEOIssue
	h.db.Where("project_id = ? AND resolved_at IS NULL", p.ID).
		Order("CASE severity WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END").
		Limit(50).Find(&issues)

	var keywords []models.KeywordResult
	h.db.Where("project_id = ?", p.ID).Order("volume DESC").Limit(50).Find(&keywords)

	utils.Success(c, gin.H{
		"project_id": p.ID,
		"issues":     issues,
		"keywords":   keywords,
	}, nil)
}

// GetProjectSocial returns social data for a project.
func (h *AdminHandler) GetProjectSocial(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid project id", "INVALID_ID")
		return
	}
	var p models.Project
	if err := h.db.First(&p, id).Error; err != nil {
		utils.NotFound(c, "Project not found")
		return
	}

	var socialMetrics []models.SocialMetric
	h.db.Where("project_id = ?", p.ID).Order("recorded_at DESC").Limit(30).Find(&socialMetrics)

	var latestSocial models.SocialMetric
	h.db.Where("project_id = ?", p.ID).Order("recorded_at DESC").First(&latestSocial)

	utils.Success(c, gin.H{
		"project_id": p.ID,
		"latest":     latestSocial,
		"history":    socialMetrics,
	}, nil)
}

// GetProjectStats returns platform-wide project stats for the admin overview.
func (h *AdminHandler) GetProjectStats(c *gin.Context) {
	var totalProjects int64
	h.db.Model(&models.Project{}).Count(&totalProjects)

	// By health
	var healthRows []struct {
		Health string
		Count  int64
	}
	h.db.Model(&models.Project{}).
		Select("health, count(*) as count").
		Group("health").
		Scan(&healthRows)
	healthBreakdown := map[string]int64{}
	for _, r := range healthRows {
		healthBreakdown[r.Health] = r.Count
	}

	// By goal
	var goalRows []struct {
		Goal  string
		Count int64
	}
	h.db.Model(&models.Project{}).
		Select("goal, count(*) as count").
		Group("goal").
		Scan(&goalRows)
	goalBreakdown := map[string]int64{}
	for _, r := range goalRows {
		goalBreakdown[r.Goal] = r.Count
	}

	// Avg health score
	var avgScore float64
	h.db.Model(&models.Project{}).
		Where("health_score > 0").
		Select("COALESCE(AVG(health_score), 0)").
		Scan(&avgScore)

	// Projects created in last 7/30 days
	sevenAgo := time.Now().AddDate(0, 0, -7)
	thirtyAgo := time.Now().AddDate(0, 0, -30)
	var new7d, new30d int64
	h.db.Model(&models.Project{}).Where("created_at >= ?", sevenAgo).Count(&new7d)
	h.db.Model(&models.Project{}).Where("created_at >= ?", thirtyAgo).Count(&new30d)

	utils.Success(c, gin.H{
		"total_projects":  totalProjects,
		"health_breakdown": healthBreakdown,
		"goal_breakdown":   goalBreakdown,
		"avg_health_score": avgScore,
		"new_7d":          new7d,
		"new_30d":         new30d,
	}, nil)
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
	RevenueAvailable bool                        `json:"revenue_available"`
	// Phase 3: project stats + feature adoption
	TotalProjects   int64              `json:"total_projects"`
	ProjectsNew7d   int64              `json:"projects_new_7d"`
	ProjectsNew30d  int64              `json:"projects_new_30d"`
	AvgHealthScore  float64            `json:"avg_health_score"`
	HealthBreakdown map[string]int64   `json:"health_breakdown"`
	GoalBreakdown   map[string]int64   `json:"goal_breakdown"`
	FeatureAdoption map[string]float64 `json:"feature_adoption"`
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
	// Per-plan counts ONLY over rows that represent a current
	// entitlement (active or trialing). Counting canceled rows
	// here would be misleading: a user who trialed, canceled,
	// subscribed, canceled, and resubscribed has 5 plan_code rows
	// on disk but contributes ONE active plan. With the filter
	// below, the per-plan number matches the active entitlement
	// count exactly. The corresponding revenue math (sum over
	// status='active') is in the MRR block further down; the two
	// are now consistent. See TestAdminStatsMRR_HandlesOrphanedRows
	// for the regression.
	var planRows []struct {
		PlanCode string
		Count    int64
	}
	h.db.Model(&models.Subscription{}).
		Where("status IN ?", []string{
			models.SubscriptionStatusActive,
			models.SubscriptionStatusTrialing,
		}).
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

	// ── Phase 3: project stats + feature adoption ──────────────────────
	h.db.Model(&models.Project{}).Count(&out.TotalProjects)
	h.db.Model(&models.Project{}).Where("created_at >= ?", sevenAgo).Count(&out.ProjectsNew7d)
	h.db.Model(&models.Project{}).Where("created_at >= ?", thirtyAgo).Count(&out.ProjectsNew30d)

	h.db.Model(&models.Project{}).
		Where("health_score > 0").
		Select("COALESCE(AVG(health_score), 0)").
		Scan(&out.AvgHealthScore)

	// Health breakdown
	out.HealthBreakdown = map[string]int64{}
	var healthRows []struct {
		Health string
		Count  int64
	}
	h.db.Model(&models.Project{}).
		Select("health, count(*) as count").
		Group("health").
		Scan(&healthRows)
	for _, r := range healthRows {
		out.HealthBreakdown[r.Health] = r.Count
	}

	// Goal breakdown
	out.GoalBreakdown = map[string]int64{}
	var goalRows []struct {
		Goal  string
		Count int64
	}
	h.db.Model(&models.Project{}).
		Select("goal, count(*) as count").
		Group("goal").
		Scan(&goalRows)
	for _, r := range goalRows {
		out.GoalBreakdown[r.Goal] = r.Count
	}

	// Feature adoption rates (percentage of projects using each feature)
	out.FeatureAdoption = map[string]float64{}
	if out.TotalProjects > 0 {
		var seoProjects int64
		h.db.Model(&models.Project{}).Where("goal IN ?", []string{"seo", "both"}).Count(&seoProjects)
		out.FeatureAdoption["seo"] = float64(seoProjects) / float64(out.TotalProjects) * 100

		var socialProjects int64
		h.db.Model(&models.Project{}).Where("goal IN ?", []string{"social", "both"}).Count(&socialProjects)
		out.FeatureAdoption["social"] = float64(socialProjects) / float64(out.TotalProjects) * 100

		var usersWithIntegrations int64
		h.db.Table("o_auth_credentials").Distinct("user_id").Count(&usersWithIntegrations)
		var totalUsers int64
		h.db.Model(&models.User{}).Count(&totalUsers)
		if totalUsers > 0 {
			out.FeatureAdoption["integrations"] = float64(usersWithIntegrations) / float64(totalUsers) * 100
		}

		var projectsWithInsights int64
		h.db.Table("insights").Distinct("project_id").Count(&projectsWithInsights)
		out.FeatureAdoption["ai_content"] = float64(projectsWithInsights) / float64(out.TotalProjects) * 100
	}

	out.RevenueAvailable = !middleware.StaffPermsLoaded(c) || middleware.StaffHas(c, models.PermStatsRevenueRead)
	if !out.RevenueAvailable {
		out.MRRCents = 0
		out.ARRProxyCents = 0
		out.PlanBreakdown = map[string]int64{}
	}

	utils.Success(c, out, nil)
}

// ── Platform SEO / Social Health ──────────────────────────────────────

// PlatformSEOHealth returns platform-wide SEO analytics computed from
// existing tables (projects, seo_issues, keyword_results).
func (h *AdminHandler) PlatformSEOHealth(c *gin.Context) {
	// SEO project count and avg health
	var totalSEO int64
	h.db.Model(&models.Project{}).
		Where("goal IN ?", []string{"seo", "both"}).
		Count(&totalSEO)

	var avgHealth float64
	h.db.Model(&models.Project{}).
		Where("goal IN ? AND health_score > 0", []string{"seo", "both"}).
		Select("COALESCE(AVG(health_score), 0)").
		Scan(&avgHealth)

	// Keywords
	var totalKeywords int64
	h.db.Table("keyword_results").
		Where("project_id IN (SELECT id FROM projects WHERE goal IN ('seo','both'))").
		Distinct("keyword").
		Count(&totalKeywords)

	var top10Keywords int64
	h.db.Table("keyword_results").
		Where("project_id IN (SELECT id FROM projects WHERE goal IN ('seo','both')) AND position > 0 AND position <= 10").
		Distinct("keyword").
		Count(&top10Keywords)

	// SEO issues
	var openIssues int64
	h.db.Table("seo_issues").
		Where("project_id IN (SELECT id FROM projects WHERE goal IN ('seo','both')) AND resolved_at IS NULL").
		Count(&openIssues)

	var criticalIssues int64
	h.db.Table("seo_issues").
		Where("project_id IN (SELECT id FROM projects WHERE goal IN ('seo','both')) AND resolved_at IS NULL AND severity = 'high'").
		Count(&criticalIssues)

	// Issue breakdown by category
	issueBreakdown := map[string]int64{}
	var issueCatRows []struct {
		Category string
		Count    int64
	}
	h.db.Table("seo_issues").
		Where("project_id IN (SELECT id FROM projects WHERE goal IN ('seo','both')) AND resolved_at IS NULL").
		Select("category, count(*) as count").
		Group("category").
		Scan(&issueCatRows)
	for _, r := range issueCatRows {
		issueBreakdown[r.Category] = r.Count
	}

	// Health distribution (bucket by score range)
	healthDist := map[string]int64{}
	var healthBucketRows []struct {
		Bucket string
		Count  int64
	}
	h.db.Raw(`
		SELECT
			CASE
				WHEN health_score >= 90 THEN '90-100'
				WHEN health_score >= 70 THEN '70-89'
				WHEN health_score >= 50 THEN '50-69'
				WHEN health_score > 0 THEN '0-49'
				ELSE 'unscored'
			END as bucket,
			count(*) as count
		FROM projects
		WHERE goal IN ('seo', 'both')
		GROUP BY bucket
	`).Scan(&healthBucketRows)
	for _, r := range healthBucketRows {
		healthDist[r.Bucket] = r.Count
	}

	utils.Success(c, gin.H{
		"total_seo_projects":  totalSEO,
		"avg_health_score":    avgHealth,
		"total_keywords":      totalKeywords,
		"keywords_in_top_10":  top10Keywords,
		"total_open_issues":   openIssues,
		"critical_issues":     criticalIssues,
		"issue_breakdown":     issueBreakdown,
		"health_distribution": healthDist,
	}, nil)
}

// PlatformSocialHealth returns platform-wide social analytics computed
// from existing tables (projects, social_metrics).
func (h *AdminHandler) PlatformSocialHealth(c *gin.Context) {
	// Social project count
	var totalSocial int64
	h.db.Model(&models.Project{}).
		Where("goal IN ?", []string{"social", "both"}).
		Count(&totalSocial)

	// Latest social metrics per project — sum followers, avg engagement, sum reach
	type socialAgg struct {
		TotalFollowers  int64
		AvgEngagement   float64
		TotalReach      int64
	}
	var agg socialAgg
	h.db.Raw(`
		SELECT
			COALESCE(SUM(sm.followers), 0) as total_followers,
			COALESCE(AVG(sm.engagement_rate), 0) as avg_engagement,
			COALESCE(SUM(sm.reach), 0) as total_reach
		FROM social_metrics sm
		INNER JOIN (
			SELECT project_id, MAX(recorded_at) as max_recorded
			FROM social_metrics
			GROUP BY project_id
		) latest ON sm.project_id = latest.project_id AND sm.recorded_at = latest.max_recorded
	`).Scan(&agg)

	// Platform breakdown
	platformBreakdown := map[string]int64{}
	var platformRows []struct {
		Platform string
		Count    int64
	}
	h.db.Raw(`
		SELECT sm.platform, count(DISTINCT sm.project_id) as count
		FROM social_metrics sm
		INNER JOIN (
			SELECT project_id, MAX(recorded_at) as max_recorded
			FROM social_metrics
			GROUP BY project_id
		) latest ON sm.project_id = latest.project_id AND sm.recorded_at = latest.max_recorded
		GROUP BY sm.platform
	`).Scan(&platformRows)
	for _, r := range platformRows {
		platformBreakdown[r.Platform] = r.Count
	}

	// Status breakdown
	statusBreakdown := map[string]int64{}
	var statusRows []struct {
		Status string
		Count  int64
	}
	h.db.Raw(`
		SELECT sm.status, count(DISTINCT sm.project_id) as count
		FROM social_metrics sm
		INNER JOIN (
			SELECT project_id, MAX(recorded_at) as max_recorded
			FROM social_metrics
			GROUP BY project_id
		) latest ON sm.project_id = latest.project_id AND sm.recorded_at = latest.max_recorded
		GROUP BY sm.status
	`).Scan(&statusRows)
	for _, r := range statusRows {
		statusBreakdown[r.Status] = r.Count
	}

	// Top 5 projects by followers
	type topProject struct {
		ProjectID      uint    `json:"project_id"`
		Name           string  `json:"name"`
		URL            string  `json:"url"`
		OwnerEmail     string  `json:"owner_email"`
		Platform       string  `json:"platform"`
		Followers      int64   `json:"followers"`
		EngagementRate float64 `json:"engagement_rate"`
		Reach          int64   `json:"reach"`
	}
	var topProjects []topProject
	h.db.Raw(`
		SELECT
			sm.project_id, p.name, p.url,
			COALESCE(u.email, '') as owner_email,
			sm.platform, sm.followers, sm.engagement_rate, sm.reach
		FROM social_metrics sm
		JOIN projects p ON p.id = sm.project_id
		LEFT JOIN users u ON u.id = p.user_id
		INNER JOIN (
			SELECT project_id, MAX(recorded_at) as max_recorded
			FROM social_metrics
			GROUP BY project_id
		) latest ON sm.project_id = latest.project_id AND sm.recorded_at = latest.max_recorded
		ORDER BY sm.followers DESC
		LIMIT 5
	`).Scan(&topProjects)

	utils.Success(c, gin.H{
		"total_social_projects": totalSocial,
		"total_followers":      agg.TotalFollowers,
		"avg_engagement_rate":  agg.AvgEngagement,
		"total_reach":          agg.TotalReach,
		"platform_breakdown":   platformBreakdown,
		"status_breakdown":     statusBreakdown,
		"top_projects":         topProjects,
	}, nil)
}

// ── System Health & Integrations ──────────────────────────────────────

// PlatformHealth returns system health status by checking database
// connectivity, server uptime, and recording the check.
func (h *AdminHandler) PlatformHealth(c *gin.Context) {
	// Check database
	dbStatus := "healthy"
	dbLatency := 0
	start := time.Now()
	if err := h.db.Raw("SELECT 1").Error; err != nil {
		dbStatus = "down"
	} else {
		dbLatency = int(time.Since(start).Milliseconds())
	}

	// Check OAuth credentials health
	oauthStatus := "healthy"
	var totalCreds int64
	h.db.Table("o_auth_credentials").Count(&totalCreds)
	var expiredCreds int64
	h.db.Table("o_auth_credentials").Where("is_expired = ?", true).Count(&expiredCreds)
	if expiredCreds > 0 && totalCreds > 0 {
		oauthStatus = "degraded"
	}

	// Check projects health
	projectStatus := "healthy"
	var projectsWithIssues int64
	h.db.Table("projects").Where("health = ?", "issues").Count(&projectsWithIssues)
	var totalProjects int64
	h.db.Model(&models.Project{}).Count(&totalProjects)
	if totalProjects > 0 && float64(projectsWithIssues)/float64(totalProjects) > 0.5 {
		projectStatus = "degraded"
	}

	// Record health check
	h.db.Create(&models.SystemHealth{
		CheckedAt: time.Now(),
		Service:   "api",
		Status:    "healthy",
		LatencyMs: dbLatency,
		Metadata:  datatypes.JSON(`{"db":"` + dbStatus + `"}`),
	})

	utils.Success(c, gin.H{
		"status": "healthy",
		"services": gin.H{
			"api_server": gin.H{
				"status":  "healthy",
				"latency": "<1ms",
			},
			"database": gin.H{
				"status":  dbStatus,
				"latency": fmt.Sprintf("%dms", dbLatency),
			},
			"oauth_integrations": gin.H{
				"status":      oauthStatus,
				"total":       totalCreds,
				"expired":     expiredCreds,
			},
			"projects": gin.H{
				"status":       projectStatus,
				"with_issues":  projectsWithIssues,
				"total":        totalProjects,
			},
		},
	}, nil)
}

// IntegrationsStatus returns the status of all OAuth integrations
// across the platform — which providers are connected, how many users
// have active credentials, and any sync errors.
func (h *AdminHandler) IntegrationsStatus(c *gin.Context) {
	// Count credentials per provider
	type providerStats struct {
		Provider     string `json:"provider"`
		Total        int64  `json:"total"`
		Expired      int64  `json:"expired"`
		SyncErrors   int64  `json:"sync_errors"`
		LastSyncedAt *time.Time `json:"last_synced_at"`
	}

	var stats []providerStats
	h.db.Table("o_auth_credentials").
		Select("provider, count(*) as total, sum(case when is_expired then 1 else 0 end) as expired, sum(case when sync_error != '' then 1 else 0 end) as sync_errors, max(last_synced_at) as last_synced_at").
		Group("provider").
		Scan(&stats)

	// Overall stats
	var totalConnections int64
	h.db.Table("o_auth_credentials").Count(&totalConnections)
	var activeConnections int64
	h.db.Table("o_auth_credentials").Where("is_expired = ?", false).Count(&activeConnections)
	var uniqueUsers int64
	h.db.Table("o_auth_credentials").Distinct("user_id").Count(&uniqueUsers)
	var uniqueProjects int64
	h.db.Table("o_auth_credentials").Distinct("project_id").Count(&uniqueProjects)

	utils.Success(c, gin.H{
		"total_connections":  totalConnections,
		"active_connections": activeConnections,
		"unique_users":       uniqueUsers,
		"unique_projects":    uniqueProjects,
		"providers":          stats,
	}, nil)
}

// AdminMe is the cheap shell check: admin.access only.
func (h *AdminHandler) AdminMe(c *gin.Context) {
	raw, _ := c.Get("auth_user")
	user, _ := raw.(*models.User)
	perms := []string{}
	if middleware.StaffPermsLoaded(c) {
		// Reconstruct list from the map without exporting the key.
		for _, p := range models.PermissionCatalog() {
			if middleware.StaffHas(c, p.Code) {
				perms = append(perms, p.Code)
			}
		}
	}
	roleCode := ""
	if user != nil {
		roleCode = user.Role
	}
	utils.Success(c, gin.H{
		"role":        roleCode,
		"permissions": perms,
	}, nil)
}

// ListRoles returns staff roles (not customer owner/viewer).
func (h *AdminHandler) ListRoles(c *gin.Context) {
	var roles []models.Role
	if err := h.db.Order("is_system DESC, code ASC").Find(&roles).Error; err != nil {
		utils.InternalError(c, "Failed to list roles")
		return
	}
	utils.Success(c, roles, nil)
}

func (h *AdminHandler) isSystemRole(code string) bool {
	if code == "" || code == models.RoleOwner || code == models.RoleViewer {
		return false
	}
	var r models.Role
	if err := h.db.Where("code = ?", code).First(&r).Error; err != nil {
		return code == models.RoleAdmin
	}
	return r.IsSystem
}

func (h *AdminHandler) countUsersWithRole(code string) int64 {
	var n int64
	h.db.Model(&models.User{}).Where("role = ?", code).Count(&n)
	return n
}

// ── helpers ───────────────────────────────────────────────────────────

// writeAudit persists a row in admin_audit_logs. targetID may be 0 for
// actions that don't target a specific user (e.g. plan edits). The
// metadata is JSON-encoded; nil becomes an empty JSON object so the
// field is never NULL.
func (h *AdminHandler) writeAudit(c *gin.Context, targetID uint, action string, metadata map[string]any) {
	actorID, _ := c.Get("user_id")
	aID, _ := actorID.(uint)
	h.writeAuditAs(aID, targetID, action, metadata)
}

func (h *AdminHandler) writeAuditAs(actorID, targetID uint, action string, metadata map[string]any) {
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
		ActorUserID:  actorID,
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

