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

	Projects []Project `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"projects,omitempty"`
}
