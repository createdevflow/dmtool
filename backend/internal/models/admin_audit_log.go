package models

import (
	"time"

	"gorm.io/datatypes"
)

// AdminAuditLog records every mutating action an admin performs: password
// resets, plan changes, impersonation start/stop, plan edits. Append-only.
type AdminAuditLog struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time      `gorm:"index" json:"created_at"`

	ActorUserID  uint           `gorm:"not null;index" json:"actor_user_id"`
	TargetUserID uint           `gorm:"not null;index" json:"target_user_id"`
	Action       string         `gorm:"not null" json:"action"`
	Metadata     datatypes.JSON `json:"metadata"`
}

// AdminAuditLog action constants.
const (
	AdminAuditActionImpersonateStart = "impersonate.start"
	AdminAuditActionImpersonateStop  = "impersonate.stop"
	AdminAuditActionUserPlanChange   = "user.plan.change"
	AdminAuditActionUserPasswordReset = "user.password.reset"
)
