package models

import "time"

// PositionTrackingCampaign is one Semrush-style rank campaign per project.
type PositionTrackingCampaign struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ProjectID       uint       `gorm:"not null;uniqueIndex" json:"project_id"`
	Domain          string     `gorm:"not null;size:255" json:"domain"`
	LocationName    string     `gorm:"not null;size:128;default:United States" json:"location_name"`
	LocationCode    int        `gorm:"not null;default:2840" json:"location_code"`
	LanguageCode    string     `gorm:"not null;size:8;default:en" json:"language_code"`
	Device          string     `gorm:"not null;size:16;default:desktop" json:"device"`
	Engine          string     `gorm:"not null;size:32;default:google" json:"engine"`
	KeywordsJSON    string     `gorm:"type:text" json:"-"`
	CompetitorsJSON string     `gorm:"type:text" json:"-"`
	TargetsJSON     string     `gorm:"type:text" json:"-"`
	SnapshotJSON    string     `gorm:"type:text" json:"-"`
	SnapshotSource  string     `gorm:"size:32" json:"snapshot_source"`
	SnapshotAt      *time.Time `json:"snapshot_at,omitempty"`
	SnapshotError   string     `gorm:"size:500" json:"snapshot_error,omitempty"`
}

func (PositionTrackingCampaign) TableName() string { return "position_tracking_campaigns" }

// PTKeyword is a tracked keyword in a campaign (JSON payload, not a table).
type PTKeyword struct {
	Keyword string   `json:"keyword"`
	Tags    []string `json:"tags,omitempty"`
}

// PTTarget is a multitargeting variation (location/device/engine).
type PTTarget struct {
	LocationName string `json:"location_name"`
	LocationCode int    `json:"location_code"`
	LanguageCode string `json:"language_code"`
	Device       string `json:"device"`
	Engine       string `json:"engine"`
}
