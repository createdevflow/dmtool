// Package repository provides the UserRepository interface and its GORM
// implementation.  All user DB access goes through this layer.
package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// UserRepository defines all user persistence operations.
type UserRepository interface {
	Create(user *models.User) error
	FindByID(id uint) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
}

// gormUserRepository is the GORM-backed implementation.
type gormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository returns a new GORM-backed UserRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

// Create inserts a new user record.
func (r *gormUserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// FindByID retrieves a user by primary key.
func (r *gormUserRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by their unique email address.
func (r *gormUserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}


// Update saves changes to an existing user record.
func (r *gormUserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Delete soft-deletes the user by ID.
func (r *gormUserRepository) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}
