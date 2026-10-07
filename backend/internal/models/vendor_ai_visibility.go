package models

import "time"

// VendorAIVisibilitySnapshot caches DataForSEO LLM Mentions target metrics
// for a domain. Separate from VendorDomainSnapshot so a visibility lookup
// cannot overwrite Labs/backlinks cache. Shared across users — this is
// public mention data, not GSC.
type VendorAIVisibilitySnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Domain       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_ai_vis" json:"domain"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_ai_vis" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_ai_vis" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"` // dataforseo
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorAIVisibilitySnapshot) TableName() string { return "vendor_ai_visibility_snapshots" }
