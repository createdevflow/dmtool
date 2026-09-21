// Package models contains all GORM database models for DMTool v2.
// Every model follows the repository pattern — no direct DB access in handlers.
package models

import (
	"time"

	"gorm.io/gorm"
)

// Role constants for RBAC
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleViewer = "viewer"
)
// User represents a registered account on DMTool.
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name         string `gorm:"not null" json:"name"`
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	Phone        string `json:"phone"`
	PasswordHash string `gorm:"not null" json:"-"`
	Role         string `gorm:"default:owner" json:"role"` // owner | admin | viewer

	// DashboardMode controls which nav groups render. search|social|combined.
	// Phase 4 added this. Existing users will see the column added by
	// db.migrate via a forward-only ALTER on first boot after deploy.
	DashboardMode string `gorm:"not null;default:combined;size:16" json:"dashboard_mode"`

	// TrialUsedAt is the one-way "this account has consumed its trial"
	// flag, set the first time the user starts a trial and never
	// cleared. Phase 6 introduced this after the prior
	// status-derived check was shown to allow trial→cancel→trial
	// abuse (the subscription status resets to "canceled", so the
	// eligibility check saw a fresh user).
	TrialUsedAt *time.Time `json:"trial_used_at,omitempty"`

	// Phase 1 admin enhancements: suspension and login tracking.
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
	DisabledReason string     `gorm:"size:500" json:"disabled_reason,omitempty"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	LoginCount     int        `gorm:"default:0" json:"login_count"`

	Projects []Project `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"projects,omitempty"`
}

// Mode constants are defined in user_preference.go; constants reused
// here to keep callers from importing both files. (Future cleanup:
// pick one canonical home.)
