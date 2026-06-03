package models

import "time"

// TaskSource distinguishes AI-created tasks from user-created ones
const (
	TaskSourceAI     = "ai"
	TaskSourceManual = "manual"
)

// Task represents an actionable item for a project.
// It can be created by the InsightGenerator worker (Source=ai) or by the user.
type Task struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ProjectID uint   `gorm:"not null;index" json:"project_id"`
	Title     string `gorm:"not null" json:"title"`
	Platform   string `gorm:"type:text;default:instagram" json:"platform"`
	ContentType string `gorm:"type:text;default:post" json:"content_type"`
	AssetName     string `gorm:"type:text" json:"asset_name"`
	AssetURL      string `gorm:"type:text" json:"asset_url"`
	AssetPath     string `gorm:"type:text" json:"asset_path"`
	AssetMime     string `gorm:"type:text" json:"asset_mime"`
	ThumbnailName string `gorm:"type:text" json:"thumbnail_name"`
	ThumbnailURL  string `gorm:"type:text" json:"thumbnail_url"`
	ThumbnailPath string `gorm:"type:text" json:"thumbnail_path"`
	ThumbnailMime string `gorm:"type:text" json:"thumbnail_mime"`
	Caption       string `gorm:"type:text" json:"caption"`
	Location    string `gorm:"type:text" json:"location"`
	Music       string `gorm:"type:text" json:"music"`
	Tags        string `gorm:"type:text" json:"tags"`
	Completed   bool      `gorm:"default:false" json:"completed"`
	PublishStatus string   `gorm:"type:text;default:scheduled" json:"publish_status"`
	PublishError  string   `gorm:"type:text" json:"publish_error"`
	PublishedAt   *time.Time `json:"published_at"`
	DueDate       *time.Time `json:"due_date"`
	Source        string   `gorm:"default:manual" json:"source"` // ai | manual
}
