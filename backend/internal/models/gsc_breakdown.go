package models

import "time"

// GSC dimension names stored in GSCBreakdown.Dimension.
const (
	GSCDimPage             = "page"
	GSCDimCountry          = "country"
	GSCDimDevice           = "device"
	GSCDimSearchAppearance = "searchAppearance"
)

// GSCBreakdown is one Search Analytics row for a single dimension
// (page, country, device, or searchAppearance) over a fetched date window.
// Values come from Google Search Console only — never estimated.
type GSCBreakdown struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	ProjectID uint `gorm:"not null;index:idx_gsc_break_proj_dim" json:"project_id"`

	Dimension string `gorm:"not null;size:32;index:idx_gsc_break_proj_dim" json:"dimension"`
	Key       string `gorm:"not null;size:2048" json:"key"`

	Clicks      int64   `json:"clicks"`
	Impressions int64   `json:"impressions"`
	CTR         float64 `json:"ctr"`      // GSC ratio 0–1
	Position    float64 `json:"position"` // average position

	StartDate string    `gorm:"size:10" json:"start_date"`
	EndDate   string    `gorm:"size:10" json:"end_date"`
	FetchedAt time.Time `json:"fetched_at"`
}

func (GSCBreakdown) TableName() string { return "gsc_breakdowns" }
