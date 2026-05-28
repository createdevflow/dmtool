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
	Completed bool   `gorm:"default:false" json:"completed"`
	DueDate   *time.Time `json:"due_date"`
	Source    string `gorm:"default:manual" json:"source"` // ai | manual
}
