package models

import "time"

// KeywordResult caches keyword research results for 24 hours.
// The Seed field stores the original seed keyword used for the lookup.
type KeywordResult struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ProjectID uint   `gorm:"not null;index" json:"project_id"`
	Seed      string `gorm:"not null" json:"seed"`    // original query keyword
	Keyword   string `gorm:"not null" json:"keyword"` // suggested keyword
	Volume    int    `json:"volume"`                   // monthly search volume
	KD        int    `json:"kd"`                       // keyword difficulty 0-100
	Position  float64 `json:"position"`                // current SERP position (GSC)
}
