package repository

import (
	"time"

	"backend/internal/models"
	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(task *models.Task) error
	FindAllByProject(projectID uint) ([]models.Task, error)
	FindDueScheduledTasks(before time.Time) ([]models.Task, error)
	FindByID(id uint) (*models.Task, error)
	Update(task *models.Task) error
	Delete(id uint) error
}

type gormTaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &gormTaskRepository{db: db}
}

func (r *gormTaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *gormTaskRepository) FindAllByProject(projectID uint) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Where("project_id = ?", projectID).Order("created_at desc").Find(&tasks).Error
	return tasks, err
}

func (r *gormTaskRepository) FindDueScheduledTasks(before time.Time) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.
		Where("due_date IS NOT NULL AND due_date <= ? AND completed = ? AND publish_status = ?", before, false, "scheduled").
		Order("due_date asc").
		Find(&tasks).Error
	return tasks, err
}

func (r *gormTaskRepository) FindByID(id uint) (*models.Task, error) {
	var task models.Task
	if err := r.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *gormTaskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *gormTaskRepository) Delete(id uint) error {
	return r.db.Delete(&models.Task{}, id).Error
}
