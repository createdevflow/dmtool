package models

import (
	"time"

	"gorm.io/gorm"
)

// Goal constants for project type
const (
	GoalSEO    = "seo"
	GoalSocial = "social"
	GoalBoth   = "both"
)

// Project represents a client website / brand tracked inside DMTool.
// Every child record (Metrics, Insights, Tasks, etc.) cascades on delete.
type Project struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID      uint   `gorm:"not null;index" json:"user_id"`
	Name        string `gorm:"not null" json:"name"`
	URL         string `gorm:"not null" json:"url"`
	Goal        string `gorm:"default:both" json:"goal"` // seo | social | both
	Status      string `gorm:"default:active" json:"status"` // active | paused | archiving
	Health      string `gorm:"default:scanning" json:"health"` // scanning | healthy | issues
	HealthScore int    `gorm:"default:0" json:"health_score"`

	// Social handles (all optional)
	IGHandle       string `json:"ig_handle"`
	FBHandle       string `json:"facebook_handle"`
	TwitterHandle  string `json:"twitter_handle"`
	LinkedinHandle string `json:"linkedin_handle"`
}
