package models

import "time"

// SeverityHigh / Med / Low are the allowed severity values for SEO issues
const (
	SeverityHigh = "high"
	SeverityMed  = "medium"
	SeverityLow  = "low"
)

// SEOIssue records a single technical SEO finding from an audit run.
// ResolvedAt being nil means the issue is still open.
type SEOIssue struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	ProjectID  uint       `gorm:"not null;index" json:"project_id"`
	URL        string     `gorm:"not null" json:"url"`
	Severity   string     `gorm:"not null" json:"severity"` // high | medium | low
	Category   string     `gorm:"not null" json:"category"` // meta | heading | performance | etc.
	Detail     string     `gorm:"not null" json:"detail"`
	ResolvedAt *time.Time `json:"resolved_at"` // nil = open
}
