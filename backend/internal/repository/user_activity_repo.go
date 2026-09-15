package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// UserActivityRepository writes and reads user activity log entries.
type UserActivityRepository interface {
	Create(row *models.UserActivity) error
	ListByUser(userID uint, limit int) ([]models.UserActivity, error)
}

type gormUserActivityRepository struct {
	db *gorm.DB
}

func NewUserActivityRepository(db *gorm.DB) UserActivityRepository {
	return &gormUserActivityRepository{db: db}
}

func (r *gormUserActivityRepository) Create(row *models.UserActivity) error {
	return r.db.Create(row).Error
}

func (r *gormUserActivityRepository) ListByUser(userID uint, limit int) ([]models.UserActivity, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.UserActivity
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}
