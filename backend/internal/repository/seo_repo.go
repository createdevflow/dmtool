// Package repository provides SEO and keyword persistence operations.
package repository

import (
	"backend/internal/models"
	"time"

	"gorm.io/gorm"
)

// SEORepository manages SEO issues and keyword results.
type SEORepository interface {
	CreateIssue(issue *models.SEOIssue) error
	FindOpenIssues(projectID uint, severity string) ([]models.SEOIssue, error)
	ResolveIssue(id, projectID uint) error

	UpsertKeywords(results []models.KeywordResult) error
	FindKeywords(projectID uint, seed string) ([]models.KeywordResult, bool, error)
}

type gormSEORepository struct {
	db *gorm.DB
}

// NewSEORepository returns a new GORM-backed SEORepository.
func NewSEORepository(db *gorm.DB) SEORepository {
	return &gormSEORepository{db: db}
}

func (r *gormSEORepository) CreateIssue(issue *models.SEOIssue) error {
	return r.db.Create(issue).Error
}

// FindOpenIssues returns all unresolved SEO issues, optionally filtered by severity.
func (r *gormSEORepository) FindOpenIssues(projectID uint, severity string) ([]models.SEOIssue, error) {
	var issues []models.SEOIssue
	q := r.db.Where("project_id = ? AND resolved_at IS NULL", projectID)
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	err := q.Order("CASE severity WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END").Find(&issues).Error
	return issues, err
}

// ResolveIssue sets resolved_at to now for an issue owned by the given project.
func (r *gormSEORepository) ResolveIssue(id, projectID uint) error {
	now := time.Now()
	return r.db.Model(&models.SEOIssue{}).
		Where("id = ? AND project_id = ?", id, projectID).
		Update("resolved_at", &now).Error
}

// UpsertKeywords batch-inserts keyword results (replaces on same project+keyword).
func (r *gormSEORepository) UpsertKeywords(results []models.KeywordResult) error {
	return r.db.Save(&results).Error
}

// FindKeywords returns cached keyword results for a project+seed pair.
// The bool return indicates whether the cache is still valid (< 24 hours old).
func (r *gormSEORepository) FindKeywords(projectID uint, seed string) ([]models.KeywordResult, bool, error) {
	var results []models.KeywordResult
	err := r.db.Where("project_id = ? AND seed = ?", projectID, seed).Find(&results).Error
	if err != nil || len(results) == 0 {
		return nil, false, err
	}
	fresh := time.Since(results[0].UpdatedAt) < 24*time.Hour
	return results, fresh, nil
}
