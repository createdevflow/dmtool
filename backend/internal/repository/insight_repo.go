// Package repository provides insight and task persistence operations.
package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// InsightRepository manages AI-generated insights and tasks.
type InsightRepository interface {
	ReplaceInsights(projectID uint, insights []models.Insight) error
	FindInsights(projectID uint, limit int) ([]models.Insight, error)

	CreateTask(task *models.Task) error
	FindOpenTasks(projectID uint, limit int) ([]models.Task, error)
	ToggleTask(id, projectID uint) (*models.Task, error)
}

type gormInsightRepository struct {
	db *gorm.DB
}

// NewInsightRepository returns a new GORM-backed InsightRepository.
func NewInsightRepository(db *gorm.DB) InsightRepository {
	return &gormInsightRepository{db: db}
}

func (r *gormInsightRepository) ReplaceInsights(projectID uint, insights []models.Insight) error {
	// Delete all existing insights for this project to prevent duplication
	if err := r.db.Where("project_id = ?", projectID).Delete(&models.Insight{}).Error; err != nil {
		return err
	}
	if len(insights) == 0 {
		return nil
	}
	return r.db.Create(&insights).Error
}

// FindInsights returns the most recent insights ordered by priority.
func (r *gormInsightRepository) FindInsights(projectID uint, limit int) ([]models.Insight, error) {
	var insights []models.Insight
	err := r.db.Where("project_id = ?", projectID).
		Order("priority ASC, created_at DESC").
		Limit(limit).Find(&insights).Error
	return insights, err
}

// CreateTask inserts a new task for a project.
func (r *gormInsightRepository) CreateTask(task *models.Task) error {
	return r.db.Create(task).Error
}

// FindOpenTasks returns the N most recent incomplete tasks.
func (r *gormInsightRepository) FindOpenTasks(projectID uint, limit int) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Where("project_id = ? AND completed = ?", projectID, false).
		Order("created_at DESC").Limit(limit).Find(&tasks).Error
	return tasks, err
}

// ToggleTask flips the completed state of a task and returns the updated record.
func (r *gormInsightRepository) ToggleTask(id, projectID uint) (*models.Task, error) {
	var task models.Task
	if err := r.db.Where("id = ? AND project_id = ?", id, projectID).First(&task).Error; err != nil {
		return nil, err
	}
	task.Completed = !task.Completed
	if err := r.db.Save(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}
