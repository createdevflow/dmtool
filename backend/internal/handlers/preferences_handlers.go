package handlers

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PreferencesHandler exposes the per-user UI preference rows. Today
// these include only `dashboard_mode`; new keys extend the field set.
type PreferencesHandler struct {
	prefRepo repository.UserPreferenceRepository
}

// NewPreferencesHandler builds the handler.
func NewPreferencesHandler(prefRepo repository.UserPreferenceRepository) *PreferencesHandler {
	return &PreferencesHandler{prefRepo: prefRepo}
}

const validModesCSV = "search,social,combined"

func isValidMode(mode string) bool {
	for _, m := range strings.Split(validModesCSV, ",") {
		if m == mode {
			return true
		}
	}
	return false
}

// defaultPref is the response shape when no row exists yet for the user.
func defaultPref(userID uint) models.UserPreference {
	return models.UserPreference{UserID: userID, DashboardMode: models.DashboardModeCombined}
}

// Get returns the current preferences row for the calling user,
// falling back to the default row (no DB write) when none exists yet.
//
// Why no insert-on-miss: phase 3 should be safe against a fresh user
// whose preferences row has never been materialised. Resolving to
// "combined" in memory keeps DB write volume down and makes GET
// idempotent.
func (h *PreferencesHandler) Get(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	pref, err := h.prefRepo.Get(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.Success(c, defaultPref(userID), nil)
			return
		}
		utils.InternalError(c, "Failed to load preferences")
		return
	}
	utils.Success(c, pref, nil)
}

// UpdateModeRequest is the body shape for PATCH /api/users/me/preferences.
type UpdateModeRequest struct {
	Mode string `json:"dashboard_mode" binding:"required"`
}

// UpdateMode writes the dashboard_mode for the calling user.
func (h *PreferencesHandler) UpdateMode(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var req UpdateModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if !isValidMode(req.Mode) {
		utils.BadRequest(c, "dashboard_mode must be one of: "+validModesCSV, "VALIDATION_ERROR")
		return
	}
	if err := h.prefRepo.UpdateMode(userID, req.Mode); err != nil {
		utils.InternalError(c, "Failed to update preferences")
		return
	}
	utils.Success(c, gin.H{
		"user_id":        userID,
		"dashboard_mode": req.Mode,
	}, nil)
}
