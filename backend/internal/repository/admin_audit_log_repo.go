package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// AdminAuditLogRepository writes and reads admin action log entries.
type AdminAuditLogRepository interface {
	Create(row *models.AdminAuditLog) error
	// ListByTargetUser returns the most recent rows for a target user,
	// newest first. Limit is required; pass 0 for default (50).
	ListByTargetUser(targetUserID uint, limit int) ([]models.AdminAuditLog, error)
	// ListFiltered paginates over the log with optional filters. Any
	// filter passed as 0 / "" is treated as "no filter on this field".
	ListFiltered(actorID, targetID int, action string, page, size int) ([]models.AdminAuditLog, int64, error)
}

type gormAdminAuditLogRepository struct {
	db *gorm.DB
}

func NewAdminAuditLogRepository(db *gorm.DB) AdminAuditLogRepository {
	return &gormAdminAuditLogRepository{db: db}
}

func (r *gormAdminAuditLogRepository) Create(row *models.AdminAuditLog) error {
	return r.db.Create(row).Error
}

func (r *gormAdminAuditLogRepository) ListByTargetUser(targetUserID uint, limit int) ([]models.AdminAuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.AdminAuditLog
	err := r.db.Where("target_user_id = ?", targetUserID).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *gormAdminAuditLogRepository) ListFiltered(actorID, targetID int, action string, page, size int) ([]models.AdminAuditLog, int64, error) {
	q := r.db.Model(&models.AdminAuditLog{})
	if actorID > 0 {
		q = q.Where("actor_user_id = ?", actorID)
	}
	if targetID > 0 {
		q = q.Where("target_user_id = ?", targetID)
	}
	if action != "" {
		q = q.Where("action = ?", action)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 50
	}
	var rows []models.AdminAuditLog
	if err := q.Order("created_at DESC").
		Limit(size).Offset((page - 1) * size).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
