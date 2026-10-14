package models

import "time"

// VendorContentGapSnapshot caches a DataForSEO Labs page-intersection +
// relevant-pages lookup for a target/competitor pair. Shared across users —
// public Labs estimates, not GSC.
type VendorContentGapSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Target       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_content_gap" json:"target"`
	Competitor   string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_content_gap" json:"competitor"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_content_gap" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_content_gap" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"`
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorContentGapSnapshot) TableName() string { return "vendor_content_gap_snapshots" }
