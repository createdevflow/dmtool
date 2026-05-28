package repository

import (
	"backend/internal/models"
	"backend/internal/utils"
	"time"

	"gorm.io/gorm"
)

// RefreshTokenRepository manages refresh token persistence and rotation.
type RefreshTokenRepository interface {
	Create(token *models.RefreshToken) error
	FindByRawToken(rawToken string) (*models.RefreshToken, error)
	RevokeByHash(tokenHash string) error
	RevokeAllForUser(userID uint) error
	PurgeExpired() error
	Delete(id uint) error
}

type gormRefreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository returns a new GORM-backed RefreshTokenRepository.
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &gormRefreshTokenRepository{db: db}
}

// Create inserts a new refresh token record.
func (r *gormRefreshTokenRepository) Create(token *models.RefreshToken) error {
	return r.db.Create(token).Error
}

// FindByRawToken looks up a refresh token by hashing the raw value.
// Returns only valid (non-revoked, non-expired) tokens.
func (r *gormRefreshTokenRepository) FindByRawToken(rawToken string) (*models.RefreshToken, error) {
	hash := utils.HashToken(rawToken)
	var token models.RefreshToken
	err := r.db.Where("token_hash = ? AND revoked = ? AND expires_at > ?", hash, false, time.Now()).
		First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}


// RevokeByHash marks a token as revoked by its hash (immediate invalidation).
func (r *gormRefreshTokenRepository) RevokeByHash(tokenHash string) error {
	return r.db.Model(&models.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("revoked", true).Error
}

// RevokeAllForUser revokes all active refresh tokens for a user (logout all devices).
func (r *gormRefreshTokenRepository) RevokeAllForUser(userID uint) error {
	return r.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Update("revoked", true).Error
}

// PurgeExpired hard-deletes tokens that have expired or been revoked.
func (r *gormRefreshTokenRepository) PurgeExpired() error {
	return r.db.Where("expires_at < ? OR revoked = ?", time.Now(), true).
		Delete(&models.RefreshToken{}).Error
}

// Delete hard-deletes a token record by primary key.
func (r *gormRefreshTokenRepository) Delete(id uint) error {
	return r.db.Delete(&models.RefreshToken{}, id).Error
}
