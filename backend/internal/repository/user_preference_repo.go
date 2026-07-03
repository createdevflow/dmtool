package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// UserPreferenceRepository reads + writes the per-user UI preferences
// (today: dashboard_mode). Phase 4 will read this on every protected
// route render; for now the frontend proxy uses its own cookie until
// the preference GET endpoint is wired.
type UserPreferenceRepository interface {
	Get(userID uint) (*models.UserPreference, error)
	Upsert(pref *models.UserPreference) error
	UpdateMode(userID uint, mode string) error
}

type gormUserPreferenceRepository struct {
	db *gorm.DB
}

// NewUserPreferenceRepository returns a GORM-backed preference repo.
func NewUserPreferenceRepository(db *gorm.DB) UserPreferenceRepository {
	return &gormUserPreferenceRepository{db: db}
}

// Get returns the preference row for a user, or gorm.ErrRecordNotFound
// when none exists yet (so callers can fall back to defaults).
func (r *gormUserPreferenceRepository) Get(userID uint) (*models.UserPreference, error) {
	var pref models.UserPreference
	err := r.db.Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

// Upsert inserts or updates a preference row keyed by user_id.
func (r *gormUserPreferenceRepository) Upsert(pref *models.UserPreference) error {
	return r.db.Save(pref).Error
}

// UpdateMode sets dashboard_mode for a user, creating the row if missing.
// Phase 3 minimises this by deferring the "create on demand" to the GET
// endpoint; phase 4 will introduce a "auto-create defaults" pattern.
func (r *gormUserPreferenceRepository) UpdateMode(userID uint, mode string) error {
	return r.db.Save(&models.UserPreference{
		UserID:        userID,
		DashboardMode: mode,
	}).Error
}
