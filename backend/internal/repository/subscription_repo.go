package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// SubscriptionRepository defines subscription persistence. Today only one
// active subscription per user is tracked; history rows will use a separate
// event table in a future phase.
type SubscriptionRepository interface {
	Create(sub *models.Subscription) error
	FindCurrentByUser(userID uint) (*models.Subscription, error)
	UpdatePlanAndStatus(userID uint, planCode, status string) error
	// FindLatestByUser returns the user's most recently-started subscription
	// row regardless of status. Used by the billing endpoints when the
	// status filter on FindCurrentByUser (active|trialing) would hide a
	// just-canceled row.
	FindLatestByUser(userID uint) (*models.Subscription, error)

}

type gormSubscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository returns a GORM-backed SubscriptionRepository.
func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &gormSubscriptionRepository{db: db}
}

// Create inserts a new subscription row.
func (r *gormSubscriptionRepository) Create(sub *models.Subscription) error {
	return r.db.Create(sub).Error
}

// FindCurrentByUser returns the most recent active or trialing subscription
// for the user, ordered by StartsAt desc. Returns gorm.ErrRecordNotFound
// when no row exists (treated as "no entitlement, default to free").
func (r *gormSubscriptionRepository) FindCurrentByUser(userID uint) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.Where("user_id = ? AND status IN ?", userID, []string{
		models.SubscriptionStatusActive,
		models.SubscriptionStatusTrialing,
	}).
		Order("starts_at DESC").
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// UpdatePlanAndStatus changes the active subscription's plan and status.
// Used by the future admin-panel endpoint and the (stubbed) Stripe webhook.
// Today kept on the interface so phase 2 doesn't introduce a second repo.
func (r *gormSubscriptionRepository) UpdatePlanAndStatus(userID uint, planCode, status string) error {
	return r.db.Model(&models.Subscription{}).
		Where("user_id = ?", userID).
		Order("starts_at DESC").
		Limit(1).
		Updates(map[string]any{
			"plan_code": planCode,
			"status":    status,
		}).Error
}

// FindLatestByUser returns the user's most recently-started subscription
// row regardless of status. Used by the billing endpoints when the
// status filter on FindCurrentByUser (active|trialing) would hide a
// just-canceled row.
func (r *gormSubscriptionRepository) FindLatestByUser(userID uint) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.Where("user_id = ?", userID).
		Order("starts_at DESC, id DESC").
		First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}
