// Package repository provides the ProjectRepository interface and its GORM
// implementation.  All queries are scoped by UserID to enforce multi-tenancy.
package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// ProjectRepository defines all project persistence operations.
// Every method accepts a userID to enforce tenant isolation.
type ProjectRepository interface {
	Create(project *models.Project) error
	FindAllByUser(userID uint) ([]models.Project, error)
	FindByIDAndUser(id, userID uint) (*models.Project, error)
	Update(project *models.Project) error
	Delete(id, userID uint) error
	FindAll() ([]models.Project, error)
}

type gormProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository returns a new GORM-backed ProjectRepository.
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &gormProjectRepository{db: db}
}

// Create inserts a new project record.
func (r *gormProjectRepository) Create(project *models.Project) error {
	return r.db.Create(project).Error
}

// FindAllByUser retrieves all projects owned by a user (excludes soft-deleted).
func (r *gormProjectRepository) FindAllByUser(userID uint) ([]models.Project, error) {
	var projects []models.Project
	err := r.db.Where("user_id = ?", userID).Find(&projects).Error
	return projects, err
}

// FindByIDAndUser retrieves a single project, verifying ownership.
// Returns gorm.ErrRecordNotFound if the project doesn't belong to the user.
func (r *gormProjectRepository) FindByIDAndUser(id, userID uint) (*models.Project, error) {
	var project models.Project
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}


// Update saves changes to an existing project.
func (r *gormProjectRepository) Update(project *models.Project) error {
	return r.db.Save(project).Error
}

// Delete soft-deletes a project, cascading to child records via DB constraints.
func (r *gormProjectRepository) Delete(id, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Project{}).Error
}
// FindAll retrieves all projects in the system.
func (r *gormProjectRepository) FindAll() ([]models.Project, error) {
	var projects []models.Project
	err := r.db.Find(&projects).Error
	return projects, err
}
