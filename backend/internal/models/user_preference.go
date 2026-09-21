package models

import (
	"time"

	"gorm.io/gorm"
)

// UserPreference is a 1:1 join table for per-user UI preferences. Stored
// separately from User so a future preference column doesn't bloat the User
// row and so unit tests can construct preferences without a User fixture.
type UserPreference struct {
	UserID        uint           `gorm:"primaryKey" json:"user_id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	DashboardMode string `gorm:"not null;default:combined" json:"dashboard_mode"`
}

// Dashboard mode values. The constant referenced from frontend code mirrors
// the values used by the brief.
const (
	DashboardModeSearch   = "search"
	DashboardModeSocial   = "social"
	DashboardModeCombined = "combined"
)
