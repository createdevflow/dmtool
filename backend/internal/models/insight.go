package models

import "time"

// InsightType constants map to badge colours in the UI
const (
	InsightTypeCritical    = "CRITICAL"
	InsightTypeOpportunity = "OPPORTUNITY"
	InsightTypeInfo        = "INFO"
)

// Insight stores AI-generated recommendations for a project.
// Records older than 24 hours are replaced by the InsightGenerator worker.
type Insight struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	ProjectID uint   `gorm:"not null;index" json:"project_id"`
	Type      string `gorm:"not null" json:"type"`  // CRITICAL | OPPORTUNITY | INFO
	Title     string `gorm:"not null" json:"title"`
	Body      string `gorm:"not null" json:"body"`
	Priority  int    `gorm:"default:5" json:"priority"` // 1 = highest
}
