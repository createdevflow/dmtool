package models

import "time"

// VendorAIVisibilityGapSnapshot caches a DataForSEO LLM Mentions
// comparison (search_scope any) for a target/competitor pair. Distinct
// from citation-gap (sources) and from a single-domain AI visibility
// lookup. Shared across users — public mention counts, not GSC. Not a
// composite AI visibility score.
type VendorAIVisibilityGapSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Target       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_ai_vis_gap" json:"target"`
	Competitor   string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_ai_vis_gap" json:"competitor"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_ai_vis_gap" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_ai_vis_gap" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"`
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorAIVisibilityGapSnapshot) TableName() string {
	return "vendor_ai_visibility_gap_snapshots"
}
