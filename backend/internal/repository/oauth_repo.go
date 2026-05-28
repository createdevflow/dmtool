// Package repository provides the OAuthCredentialRepository for managing
// encrypted OAuth tokens for Google and Meta integrations.
package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// OAuthRepository defines CRUD operations for OAuth credentials.
type OAuthRepository interface {
	Upsert(cred *models.OAuthCredential) error
	FindByUserAndProvider(userID uint, provider string) (*models.OAuthCredential, error)
	FindByProjectAndProvider(projectID uint, provider string) (*models.OAuthCredential, error)
	FindAllByUser(userID uint) ([]models.OAuthCredential, error)
	FindAll() ([]models.OAuthCredential, error)
	Delete(userID uint, provider string) error
}

type gormOAuthRepository struct {
	db *gorm.DB
}

// NewOAuthRepository returns a new GORM-backed OAuthRepository.
func NewOAuthRepository(db *gorm.DB) OAuthRepository {
	return &gormOAuthRepository{db: db}
}

// Upsert inserts or updates an OAuth credential for a user+provider pair.
func (r *gormOAuthRepository) Upsert(cred *models.OAuthCredential) error {
	var existing models.OAuthCredential
	err := r.db.Where("user_id = ? AND provider = ?", cred.UserID, cred.Provider).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(cred).Error
	}
	cred.ID = existing.ID
	return r.db.Save(cred).Error
}

// FindByUserAndProvider retrieves a credential by user ID and provider name.
func (r *gormOAuthRepository) FindByUserAndProvider(userID uint, provider string) (*models.OAuthCredential, error) {
	var cred models.OAuthCredential
	if err := r.db.Where("user_id = ? AND provider = ?", userID, provider).First(&cred).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

// FindByProjectAndProvider retrieves a credential scoped to a specific project.
func (r *gormOAuthRepository) FindByProjectAndProvider(projectID uint, provider string) (*models.OAuthCredential, error) {
	var cred models.OAuthCredential
	if err := r.db.Where("project_id = ? AND provider = ?", projectID, provider).First(&cred).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}


// FindAllByUser retrieves all OAuth credentials for a user (for the integrations page).
func (r *gormOAuthRepository) FindAllByUser(userID uint) ([]models.OAuthCredential, error) {
	var creds []models.OAuthCredential
	err := r.db.Where("user_id = ?", userID).Find(&creds).Error
	return creds, err
}

// Delete removes an OAuth credential (disconnect integration).
func (r *gormOAuthRepository) Delete(userID uint, provider string) error {
	return r.db.Where("user_id = ? AND provider = ?", userID, provider).Delete(&models.OAuthCredential{}).Error
}
// FindAll retrieves all OAuth credentials in the system.
func (r *gormOAuthRepository) FindAll() ([]models.OAuthCredential, error) {
	var creds []models.OAuthCredential
	err := r.db.Find(&creds).Error
	return creds, err
}
