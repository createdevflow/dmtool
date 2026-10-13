package models

import "time"

// VendorKeywordGapSnapshot caches a DataForSEO Labs domain-intersection
// lookup for a target/competitor pair. Shared across users — public Labs
// keyword overlap, not GSC.
type VendorKeywordGapSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Target       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_kw_gap" json:"target"`
	Competitor   string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_kw_gap" json:"competitor"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_kw_gap" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_kw_gap" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"`
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorKeywordGapSnapshot) TableName() string { return "vendor_keyword_gap_snapshots" }
