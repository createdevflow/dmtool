package models

import "time"

// VendorDomainCompareSnapshot caches a DataForSEO Labs + backlinks
// side-by-side lookup for a target/competitor pair. Shared across users —
// public Labs estimates, not GSC.
type VendorDomainCompareSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Target       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_domain_cmp" json:"target"`
	Competitor   string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_domain_cmp" json:"competitor"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_domain_cmp" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_domain_cmp" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"`
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorDomainCompareSnapshot) TableName() string { return "vendor_domain_compare_snapshots" }
