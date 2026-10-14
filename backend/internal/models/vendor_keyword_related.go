package models

import "time"

// VendorKeywordRelatedSnapshot caches DataForSEO Labs related keywords
// for a seed (Google US/en). Shared across users — public Labs data, not GSC.
type VendorKeywordRelatedSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Seed         string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_keyword_related" json:"seed"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_keyword_related" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_keyword_related" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"`
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorKeywordRelatedSnapshot) TableName() string { return "vendor_keyword_related_snapshots" }
