package models

import "time"

// SiteAuditCampaign is one Semrush-style Site Audit campaign per project.
type SiteAuditCampaign struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ProjectID     uint   `gorm:"not null;uniqueIndex" json:"project_id"`
	Domain        string `gorm:"not null;size:255" json:"domain"`
	Scope         string `gorm:"not null;size:32;default:domain" json:"scope"` // domain | subdomain | subfolder
	PageLimit     int    `gorm:"not null;default:100" json:"page_limit"`
	CrawlSource   string `gorm:"not null;size:32;default:website" json:"crawl_source"` // website | sitemap
	Schedule      string `gorm:"not null;size:32;default:manual" json:"schedule"`      // manual | daily | weekly
	UserAgent     string `gorm:"not null;size:32;default:desktop" json:"user_agent"`
	JSRendering   bool   `gorm:"not null;default:false" json:"js_rendering"`
	SubfolderPath string `gorm:"size:512" json:"subfolder_path"`

	SnapshotJSON      string     `gorm:"type:text" json:"-"`
	PreviousJSON      string     `gorm:"type:text" json:"-"`
	SnapshotAt        *time.Time `json:"snapshot_at,omitempty"`
	SnapshotError     string     `gorm:"size:500" json:"snapshot_error,omitempty"`
	PagesCrawled      int        `json:"pages_crawled"`
	LastStoppedReason string     `gorm:"size:255" json:"last_stopped_reason,omitempty"`
}

func (SiteAuditCampaign) TableName() string { return "site_audit_campaigns" }
