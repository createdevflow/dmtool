package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type IntegrationHandler struct {
	projectRepo  repository.ProjectRepository
	oauthRepo    repository.OAuthRepository
	googleConfig *oauth2.Config
	metaConfig   *oauth2.Config
	linkedinConfig *oauth2.Config
	encKey       []byte
	cfg          *config.Config
}

func NewIntegrationHandler(
	projectRepo repository.ProjectRepository,
	oauthRepo repository.OAuthRepository,
	googleConfig *oauth2.Config,
	metaConfig *oauth2.Config,
	linkedinConfig *oauth2.Config,
	encKey []byte,
	cfg *config.Config,
) *IntegrationHandler {
	return &IntegrationHandler{
		projectRepo:  projectRepo,
		oauthRepo:    oauthRepo,
		googleConfig: googleConfig,
		metaConfig:   metaConfig,
		linkedinConfig: linkedinConfig,
		encKey:       encKey,
		cfg:          cfg,
	}
}

func (h *IntegrationHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	creds, err := h.oauthRepo.FindAllByUser(userID)
	if err != nil {
		utils.InternalError(c, "Failed to fetch integrations")
		return
	}

	utils.Success(c, creds, nil)
}

func (h *IntegrationHandler) GoogleAuthURL(c *gin.Context) {
	if h.googleConfig == nil || h.googleConfig.ClientID == "" {
		utils.BadRequest(c, "Google OAuth not configured", "MISSING_CONFIG")
		return
	}

	url := h.googleConfig.AuthCodeURL("google", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	utils.Success(c, gin.H{"url": url}, nil)
}

func (h *IntegrationHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		utils.BadRequest(c, "Code is required", "MISSING_CODE")
		return
	}

	token, err := h.googleConfig.Exchange(context.Background(), code)
	if err != nil {
		utils.InternalError(c, "Failed to exchange token")
		return
	}

	userID := c.MustGet("user_id").(uint)

	// Encrypt tokens before storing
	accessToken, _ := utils.Encrypt(token.AccessToken, h.encKey)
	refreshToken, _ := utils.Encrypt(token.RefreshToken, h.encKey)

	cred := &models.OAuthCredential{
		UserID:          userID,
		Provider:        "google",
		AccessTokenEnc:  accessToken,
		RefreshTokenEnc: refreshToken,
		ExpiresAt:       token.Expiry,
	}

	if err := h.oauthRepo.Upsert(cred); err != nil {
		utils.InternalError(c, "Failed to save credentials")
		return
	}

	utils.Success(c, "Integration complete", nil)
}

func (h *IntegrationHandler) MetaAuthURL(c *gin.Context) {
	if h.metaConfig == nil || h.metaConfig.ClientID == "" {
		utils.BadRequest(c, "Meta OAuth not configured", "MISSING_CONFIG")
		return
	}

	url := h.metaConfig.AuthCodeURL("meta")
	utils.Success(c, gin.H{"url": url}, nil)
}

func (h *IntegrationHandler) MetaCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		utils.BadRequest(c, "Code is required", "MISSING_CODE")
		return
	}

	token, err := h.metaConfig.Exchange(context.Background(), code)
	if err != nil {
		utils.InternalError(c, "Failed to exchange token")
		return
	}

	userID := c.MustGet("user_id").(uint)

	// Encrypt tokens before storing
	accessToken, _ := utils.Encrypt(token.AccessToken, h.encKey)
	refreshToken, _ := utils.Encrypt(token.RefreshToken, h.encKey)

	cred := &models.OAuthCredential{
		UserID:          userID,
		Provider:        "meta",
		AccessTokenEnc:  accessToken,
		RefreshTokenEnc: refreshToken,
		ExpiresAt:       token.Expiry,
	}

	if err := h.oauthRepo.Upsert(cred); err != nil {
		utils.InternalError(c, "Failed to save credentials")
		return
	}

	utils.Success(c, "Integration complete", nil)
}

func (h *IntegrationHandler) LinkedinAuthURL(c *gin.Context) {
	if h.linkedinConfig == nil || h.linkedinConfig.ClientID == "" {
		utils.BadRequest(c, "LinkedIn OAuth not configured", "MISSING_CONFIG")
		return
	}

	// Generate a short random state to help detect CSRF and aid debugging.
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Printf("[linkedin] failed to generate state: %v", err)
		// fallback to static state
		b = []byte("linkedin")
	}
	state := "linkedin:" + base64.RawURLEncoding.EncodeToString(b)

	url := h.linkedinConfig.AuthCodeURL(state)
	utils.Success(c, gin.H{"url": url, "state": state}, nil)
}

func (h *IntegrationHandler) LinkedinCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		utils.BadRequest(c, "Code is required", "MISSING_CODE")
		return
	}

	token, err := h.linkedinConfig.Exchange(context.Background(), code)
	if err != nil {
		// If the token exchange fails, try to surface LinkedIn's response for debugging.
		if rerr, ok := err.(*oauth2.RetrieveError); ok {
			log.Printf("[linkedin] token exchange failed: %s", string(rerr.Body))
			utils.IntegrationError(c, fmt.Sprintf("Failed to exchange token: %s", string(rerr.Body)))
			return
		}
		log.Printf("[linkedin] token exchange error: %v", err)
		utils.IntegrationError(c, fmt.Sprintf("Failed to exchange token: %v", err))
		return
	}

	userID := c.MustGet("user_id").(uint)

	// Encrypt tokens before storing
	accessToken, _ := utils.Encrypt(token.AccessToken, h.encKey)
	refreshToken, _ := utils.Encrypt(token.RefreshToken, h.encKey)

	cred := &models.OAuthCredential{
		UserID:          userID,
		Provider:        "linkedin",
		AccessTokenEnc:  accessToken,
		RefreshTokenEnc: refreshToken,
		ExpiresAt:       token.Expiry,
	}

	if err := h.oauthRepo.Upsert(cred); err != nil {
		utils.InternalError(c, "Failed to save credentials")
		return
	}

	utils.Success(c, "Integration complete", nil)
}

func (h *IntegrationHandler) Disconnect(c *gin.Context) {
	provider := c.Param("provider")
	userID := c.MustGet("user_id").(uint)

	if err := h.oauthRepo.Delete(userID, provider); err != nil {
		utils.InternalError(c, "Failed to disconnect integration")
		return
	}
	
	utils.NoContent(c)
}

