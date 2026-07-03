package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// PlanRepository reads plan rows. Phase 2 keeps it read-only — admins
// will mutate these rows in a future phase via the admin panel.
type PlanRepository interface {
	FindByCode(code string) (*models.Plan, error)
	ListActive() ([]models.Plan, error)
}

type gormPlanRepository struct {
	db *gorm.DB
}

// NewPlanRepository returns a GORM-backed PlanRepository.
func NewPlanRepository(db *gorm.DB) PlanRepository {
	return &gormPlanRepository{db: db}
}

// FindByCode returns the plan with the given code, or
// gorm.ErrRecordNotFound if absent.
func (r *gormPlanRepository) FindByCode(code string) (*models.Plan, error) {
	var plan models.Plan
	err := r.db.Where("code = ?", code).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// ListActive returns all plans where is_active = true, ordered by tier_rank
// asc. The pricing page uses this.
func (r *gormPlanRepository) ListActive() ([]models.Plan, error) {
	var plans []models.Plan
	err := r.db.Where("is_active = ?", true).
		Order("tier_rank ASC, code ASC").
		Find(&plans).Error
	return plans, err
}
