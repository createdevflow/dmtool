package models

import "time"

// DomainOverviewRequest is one billed Domain Overview Search for a user.
// Semrush counts daily requests per subscription; cache hits after Search
// still count because the user used a lookup.
type DomainOverviewRequest struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Domain    string    `gorm:"not null;size:255" json:"domain"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (DomainOverviewRequest) TableName() string { return "domain_overview_requests" }
