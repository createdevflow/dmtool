package models

import "time"

// VendorCitationGapSnapshot caches a DataForSEO LLM Mentions citation
// comparison for a target/competitor pair. Shared across users — public
// LLM citation counts, not GSC.
type VendorCitationGapSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Target       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_citation_gap" json:"target"`
	Competitor   string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_citation_gap" json:"competitor"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_citation_gap" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_citation_gap" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"`
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorCitationGapSnapshot) TableName() string { return "vendor_citation_gap_snapshots" }
