package models

import "time"

// MetricSource identifies the origin of a metric data point
const (
	MetricSourceGSC  = "gsc"
	MetricSourceMeta = "meta"
	MetricSourceSeed = "seed"
)

// Metric is a time-series data point for a project.
// Date is stored as "YYYY-MM-DD" string for easy range queries.
// Source distinguishes live API data from auto-seeded fallback data.
type Metric struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ProjectID uint   `gorm:"not null;index:idx_metric_project_date" json:"project_id"`
	Date      string `gorm:"not null;index:idx_metric_project_date" json:"date"` // YYYY-MM-DD

	Clicks      int64  `json:"clicks"`
	Impressions int64  `json:"impressions"`
	Reach       int64  `json:"reach"`
	Engagement  int64  `json:"engagement"`
	Source      string `gorm:"default:seed" json:"source"` // gsc | meta | seed

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
