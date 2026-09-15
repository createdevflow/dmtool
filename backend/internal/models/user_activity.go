package models

import (
	"time"

	"gorm.io/datatypes"
)

// UserActivity records significant user actions for the admin activity
// timeline. Append-only — rows are never updated or deleted.
type UserActivity struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Action    string         `gorm:"not null;size:100" json:"action"`
	Metadata  datatypes.JSON `json:"metadata"`
	IPAddress string         `gorm:"size:45" json:"ip_address"`
	UserAgent string         `gorm:"size:500" json:"user_agent"`
}

// UserActivity action constants.
const (
	UserActivityLogin         = "login"
	UserActivityRegister      = "register"
	UserActivityProjectCreate = "project.create"
	UserActivityAuditRun      = "audit.run"
	UserActivityContentGen    = "content.generate"
	UserActivityPasswordReset = "password.reset"
	UserActivityProfileUpdate = "profile.update"
)
