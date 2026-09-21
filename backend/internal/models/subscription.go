package models

import (
	"time"

	"gorm.io/gorm"
)

// Subscription represents one user's current plan. Append-only history rows
// (e.g. plan changes, cancellations) can be added in a future phase by
// introducing a SubscriptionEvent model — not needed for the entitlement
// guard introduced in phase 2.
type Subscription struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	UserID      uint    `gorm:"not null;index" json:"user_id"`
	PlanCode    string  `gorm:"not null" json:"plan_code"`
	Status      string  `gorm:"not null;default:active" json:"status"`
	TrialEndsAt *time.Time `json:"trial_ends_at"`
	StartsAt    time.Time `gorm:"not null" json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	CanceledAt  *time.Time `json:"canceled_at"`
	StripeSubID string  `json:"stripe_sub_id"`
}

// Subscription status values.
const (
	SubscriptionStatusTrialing  = "trialing"
	SubscriptionStatusActive   = "active"
	SubscriptionStatusPastDue  = "past_due"
	SubscriptionStatusCanceled = "canceled"
)
