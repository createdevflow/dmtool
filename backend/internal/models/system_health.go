package models

import (
	"time"

	"gorm.io/datatypes"
)

// SystemHealth records periodic health-check snapshots for each service.
// Append-only — rows are never updated or deleted.
type SystemHealth struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CheckedAt time.Time      `gorm:"index" json:"checked_at"`
	Service   string         `gorm:"not null;size:50;index" json:"service"` // api, database, ai_engine, seo_crawler, social_api
	Status    string         `gorm:"not null;size:20" json:"status"`       // healthy, degraded, down
	LatencyMs int            `json:"latency_ms"`
	Metadata  datatypes.JSON `json:"metadata"`
}
