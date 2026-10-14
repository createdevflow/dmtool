package models

import "time"

// VendorBacklinkExplorerSnapshot caches a DataForSEO backlinks summary plus
// referring-domains sample for a target. Separate from VendorDomainSnapshot
// so an explorer lookup cannot overwrite Labs cache. Shared across users —
// public referring-domain data, not GSC.
type VendorBacklinkExplorerSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Domain       string    `gorm:"not null;size:255;uniqueIndex:idx_vendor_backlink_explorer" json:"domain"`
	LocationCode int       `gorm:"not null;uniqueIndex:idx_vendor_backlink_explorer" json:"location_code"`
	LanguageCode string    `gorm:"not null;size:8;uniqueIndex:idx_vendor_backlink_explorer" json:"language_code"`
	Source       string    `gorm:"not null;size:32" json:"source"`
	Payload      string    `gorm:"type:text" json:"-"`
	FetchedAt    time.Time `json:"fetched_at"`
}

func (VendorBacklinkExplorerSnapshot) TableName() string {
	return "vendor_backlink_explorer_snapshots"
}
