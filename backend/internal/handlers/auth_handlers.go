package handlers

import (
	"crypto/rsa"
	"encoding/json"
	"log"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db               *gorm.DB
	userRepo         repository.UserRepository
	tokenRepo        repository.RefreshTokenRepository
	projectRepo      repository.ProjectRepository
	subscriptionRepo repository.SubscriptionRepository
	activityRepo     repository.UserActivityRepository
	privKey          *rsa.PrivateKey
	pubKey           *rsa.PublicKey
	encKey           []byte
	cfg              *config.Config
}
func NewAuthHandler(
	db *gorm.DB,
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	projectRepo repository.ProjectRepository,
	subscriptionRepo repository.SubscriptionRepository,
	activityRepo repository.UserActivityRepository,
	privKey *rsa.PrivateKey,
	pubKey *rsa.PublicKey,
	encKey []byte,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		db:               db,
		userRepo:         userRepo,
		tokenRepo:        tokenRepo,
		projectRepo:      projectRepo,
		subscriptionRepo: subscriptionRepo,
		activityRepo:     activityRepo,
		privKey:          privKey,
		pubKey:           pubKey,
		encKey:           encKey,
		cfg:              cfg,
	}
}
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	// Check if user exists
	existing, _ := h.userRepo.FindByEmail(req.Email)
	if existing != nil {
		utils.BadRequest(c, "Email already registered", "EMAIL_EXISTS")
		return
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		utils.InternalError(c, "Failed to process security")
		return
	}

	user := &models.User{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hashed),
		Role:         "owner",
	}

	if err := h.userRepo.Create(user); err != nil {
		utils.InternalError(c, "Failed to create account")
		return
	}
	// Auto-subscribe the new user to the default (free) plan. This row is
	// the source of truth for entitlements — entitlements.ResolveSubscription
	// falls back to the default if the row is missing, but we create it
	// explicitly here so admin views and the user management surface are
	// simpler.
	sub := &models.Subscription{
		UserID:   user.ID,
		PlanCode: "free",
		Status:   models.SubscriptionStatusActive,
		StartsAt: time.Now(),
	}
	if err := h.subscriptionRepo.Create(sub); err != nil {
		log.Printf("[auth] WARN: failed to auto-subscribe user %d: %v", user.ID, err)
	}

	// Issue tokens
	h.issueTokens(c, user)
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil || user == nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}
	if user.DisabledAt != nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}

	// Log login event and update login tracking fields.
	now := time.Now()
	h.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"last_login_at": now,
		"login_count":   gorm.Expr("login_count + 1"),
	})
	if h.activityRepo != nil {
		meta, _ := json.Marshal(map[string]any{"email": user.Email})
		_ = h.activityRepo.Create(&models.UserActivity{
			UserID:    user.ID,
			Action:    models.UserActivityLogin,
			Metadata:  meta,
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
	}

	h.issueTokens(c, user)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		utils.Unauthorized(c, "Session expired")
		return
	}

	// Validate refresh token
	token, err := h.tokenRepo.FindByRawToken(cookie)
	if err != nil || token == nil {
		utils.Unauthorized(c, "Session expired")
		return
	}

	user, err := h.userRepo.FindByID(token.UserID)
	if err != nil || user == nil {
		utils.Unauthorized(c, "User not found")
		return
	}
	if user.DisabledAt != nil {
		_ = h.tokenRepo.Delete(token.ID)
		utils.Unauthorized(c, "Session expired")
		return
	}

	// Rotate token
	h.tokenRepo.Delete(token.ID)
	h.issueTokens(c, user)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	cookie, err := c.Cookie("refresh_token")
	if err == nil {
		// Revoke in DB
		token, _ := h.tokenRepo.FindByRawToken(cookie)
		if token != nil {
			h.tokenRepo.Delete(token.ID)
		}
	}

	// Clear cookie
	c.SetCookie("refresh_token", "", -1, "/", "", h.cfg.AppEnv == "production", true)
	utils.NoContent(c)
}


func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "Not authenticated")
		return
	}

	user, err := h.userRepo.FindByID(userID.(uint))
	if err != nil || user == nil {
		utils.NotFound(c, "User not found")
		return
	}

	utils.Success(c, user, nil)
}

// UpdateMeBody is the body shape for PATCH /api/auth/me.
type UpdateMeBody struct {
	DashboardMode string `json:"dashboard_mode" binding:"required"`
}

// validDashboardMode reports whether v is one of the allowed mode
// literals: "search" | "social" | "combined".
func validDashboardMode(v string) bool {
	switch v {
	case models.DashboardModeSearch, models.DashboardModeSocial, models.DashboardModeCombined:
		return true
	}
	return false
}

// UpdateMe writes mutable fields on the calling user. Phase 4 only
// exposes dashboard_mode; future fields will share this handler.
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "Not authenticated")
		return
	}
	var req UpdateMeBody
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if !validDashboardMode(req.DashboardMode) {
		utils.BadRequest(c, "dashboard_mode must be one of: search, social, combined", "VALIDATION_ERROR")
		return
	}

	user, err := h.userRepo.FindByID(userID.(uint))
	if err != nil || user == nil {
		utils.NotFound(c, "User not found")
		return
	}
	user.DashboardMode = req.DashboardMode
	if err := h.userRepo.Update(user); err != nil {
		utils.InternalError(c, "Failed to update user")
		return
	}
	utils.Success(c, gin.H{
		"user_id":        user.ID,
		"dashboard_mode": user.DashboardMode,
	}, nil)
}

func (h *AuthHandler) issueTokens(c *gin.Context, user *models.User) {
	// Generate JWT Access Token (RS256)
	accessToken, err := utils.GenerateAccessToken(h.privKey, user.ID, user.Email, user.Role)
	if err != nil {
		utils.InternalError(c, "Failed to issue access token")
		return
	}

	// Generate Refresh Token (Opaque)
	refreshToken := utils.GenerateOpaqueToken()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	rt := &models.RefreshToken{
		UserID:     user.ID,
		TokenHash:  utils.HashToken(refreshToken),
		ExpiresAt:  expiresAt,
		DeviceInfo: c.Request.UserAgent(),
	}

	if err := h.tokenRepo.Create(rt); err != nil {
		utils.InternalError(c, "Failed to issue refresh token")
		return
	}

	// Set HttpOnly Cookie
	c.SetCookie("refresh_token", refreshToken, int(7*24*3600), "/", "", h.cfg.AppEnv == "production", true)

	utils.Success(c, gin.H{
		"token": accessToken,
		"user":  user,
	}, nil)
}
