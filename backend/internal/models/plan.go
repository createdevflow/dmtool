package models

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/datatypes"
)

// Plan represents a billable subscription tier. Limits (e.g. max_sites) and
// prices live on this row at request time so adding a new tier doesn't
// require a code change.
type Plan struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	Code          string         `gorm:"uniqueIndex;not null" json:"code"`
	Name          string         `gorm:"not null" json:"name"`
	Description   string         `json:"description"`
	TierRank      int            `gorm:"not null;default:0" json:"tier_rank"`
	MonthlyCents  int            `gorm:"not null;default:0" json:"monthly_cents"`
	YearlyCents   int            `gorm:"not null;default:0" json:"yearly_cents"`
	Currency      string         `gorm:"not null;default:usd" json:"currency"`
	StripePriceID string         `json:"stripe_price_id"`
	MaxSites      int            `gorm:"not null;default:1" json:"max_sites"`
	Features      datatypes.JSON `json:"features"`
	IsActive      bool           `gorm:"not null;default:true" json:"is_active"`
}

// PlanCode constants used across the codebase.
const (
	PlanCodeFree        = "free"
	PlanCodeProMonthly  = "pro_monthly"
	PlanCodeProYearly   = "pro_yearly"
)
