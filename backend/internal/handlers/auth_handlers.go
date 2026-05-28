package handlers

import (
	"crypto/rsa"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo    repository.UserRepository
	tokenRepo   repository.RefreshTokenRepository
	projectRepo repository.ProjectRepository
	privKey     *rsa.PrivateKey
	pubKey      *rsa.PublicKey
	encKey      []byte
	cfg         *config.Config
}

func NewAuthHandler(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	projectRepo repository.ProjectRepository,
	privKey *rsa.PrivateKey,
	pubKey *rsa.PublicKey,
	encKey []byte,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		projectRepo: projectRepo,
		privKey:     privKey,
		pubKey:      pubKey,
		encKey:      encKey,
		cfg:         cfg,
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
