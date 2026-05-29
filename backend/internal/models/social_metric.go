package models

import "time"

// SocialMetric stores point-in-time social platform snapshots for growth
// tracking.  A new row is inserted each time the MetricsSyncer runs.
type SocialMetric struct {
	ID        uint  `gorm:"primaryKey" json:"id"`
	ProjectID uint  `gorm:"not null;index" json:"project_id"`

	Platform        string  `gorm:"not null" json:"platform"` // facebook | instagram
	Followers       int64   `json:"followers"`
	FollowingCount  int64   `json:"following_count"`
	PostsCount      int64   `json:"posts_count"`
	Reach           int64   `json:"reach"`
	WeeklyReach     int64   `json:"weekly_reach"`
	MonthlyReach    int64   `json:"monthly_reach"`
	Biography       string  `gorm:"type:text" json:"biography"`
	Website         string  `json:"website"`
	DisplayName     string  `json:"display_name"`
	ProfilePictureURL string `json:"profile_picture_url"`
	Engagement      float64 `json:"engagement_rate"` // percentage
	EngagementCount int64 `json:"engagement"` // count
	Status          string  `json:"status"`         // growing | stable | dropping
	IsSimulated     bool    `json:"is_simulated"`
	TopContent      string  `gorm:"type:text" json:"top_content"`      // JSON of top posts
	ContentSplit    string  `gorm:"type:text" json:"content_split"`    // JSON of Reels/Posts split
	ActiveTimes     string  `gorm:"type:text" json:"active_times"`     // JSON of follower active times
	AudienceInsights string `gorm:"type:text" json:"audience_insights"` // JSON of audience distribution
	ProfileVisits    int64 `json:"profile_visits"`
	ExternalLinkTaps int64 `json:"external_link_taps"`

	// RecordedAt has no NOT NULL tag so SQLite can add this column to existing
	// tables via AutoMigrate.  Application code always sets this before insert.
	RecordedAt time.Time `gorm:"index" json:"recorded_at"`
}
