package models

import "time"

// SocialMetric stores point-in-time social platform snapshots for growth
// tracking.  A new row is inserted each time the MetricsSyncer runs.
type SocialMetric struct {
	ID        uint  `gorm:"primaryKey" json:"id"`
	ProjectID uint  `gorm:"not null;index" json:"project_id"`

	Platform   string  `gorm:"not null" json:"platform"` // facebook | instagram
	Followers  int64   `json:"followers"`
	Reach      int64   `json:"reach"`
	Engagement float64 `json:"engagement_rate"` // percentage
	EngagementCount int64 `json:"engagement"` // count
	Status     string  `json:"status"`         // growing | stable | dropping
	IsSimulated bool    `json:"is_simulated"`
	// RecordedAt has no NOT NULL tag so SQLite can add this column to existing
	// tables via AutoMigrate.  Application code always sets this before insert.
	RecordedAt time.Time `gorm:"index" json:"recorded_at"`
}
