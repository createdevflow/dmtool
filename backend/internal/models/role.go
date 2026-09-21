package models

import "time"

// Role is a staff role. Code is the value stored on users.role for
// staff accounts (Super Admin code is "admin"). Customer accounts
// keep role=owner and have no row here.
type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Code        string `gorm:"uniqueIndex;not null;size:64" json:"code"`
	Name        string `gorm:"not null;size:120" json:"name"`
	Description string `gorm:"size:500" json:"description"`
	IsSystem    bool   `gorm:"not null;default:false" json:"is_system"`

	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// RolePermission is the roles ↔ permissions join.
type RolePermission struct {
	RoleID       uint `gorm:"primaryKey"`
	PermissionID uint `gorm:"primaryKey"`
}

const RoleCodeSuperAdmin = RoleAdmin // "admin" — keep in lockstep with User.Role / cookie / bootstrap
