package models

import "time"

// VendorDomainSnapshot caches a DataForSEO Labs + Backlinks lookup for a domain.
// Payload is the JSON DomainExplorerResult. Shared across users — this is
// public domain metrics, not private GSC data.
type VendorDomainSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Domain       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_domain_loc" json:"domain"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_domain_loc" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_domain_loc" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"` // dataforseo
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorDomainSnapshot) TableName() string { return "vendor_domain_snapshots" }
