package models

import "time"

// VendorBacklinkGapSnapshot caches a DataForSEO backlinks domain
// intersection for a target/competitor pair. Shared across users —
// public referring-domain data, not GSC.
type VendorBacklinkGapSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Target       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_backlink_gap" json:"target"`
	Competitor   string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_backlink_gap" json:"competitor"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_backlink_gap" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_backlink_gap" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"`
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorBacklinkGapSnapshot) TableName() string { return "vendor_backlink_gap_snapshots" }
