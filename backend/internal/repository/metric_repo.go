// Package repository provides the MetricRepository for time-series data access.
package repository

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

// MetricRepository defines metric and social metric persistence operations.
type MetricRepository interface {
	CreateMetric(m *models.Metric) error
	FindMetricsByProjectAndRange(projectID uint, from, to string) ([]models.Metric, error)
	FindLatestMetric(projectID uint) (*models.Metric, error)
	UpsertMetric(m *models.Metric) error

	CreateSocialMetric(sm *models.SocialMetric) error
	FindSocialMetricsByProject(projectID uint, days int) ([]models.SocialMetric, error)
	FindLatestSocialMetric(projectID uint, platform string) (*models.SocialMetric, error)
	FindLatestSocialMetrics(projectID uint) ([]models.SocialMetric, error)
}

type gormMetricRepository struct {
	db *gorm.DB
}

// NewMetricRepository returns a new GORM-backed MetricRepository.
func NewMetricRepository(db *gorm.DB) MetricRepository {
	return &gormMetricRepository{db: db}
}

// CreateMetric inserts a new metric data point.
func (r *gormMetricRepository) CreateMetric(m *models.Metric) error {
	return r.db.Create(m).Error
}

// FindMetricsByProjectAndRange retrieves metrics for a date range (inclusive).
// Date format: "YYYY-MM-DD".
func (r *gormMetricRepository) FindMetricsByProjectAndRange(projectID uint, from, to string) ([]models.Metric, error) {
	var metrics []models.Metric
	err := r.db.Where("project_id = ? AND date >= ? AND date <= ?", projectID, from, to).
		Order("date ASC").Find(&metrics).Error
	return metrics, err
}

// FindLatestMetric returns the most recent metric record for a project.
func (r *gormMetricRepository) FindLatestMetric(projectID uint) (*models.Metric, error) {
	var m models.Metric
	err := r.db.Where("project_id = ?", projectID).Order("date DESC").First(&m).Error
	return &m, err
}

// UpsertMetric updates an existing metric for the date or creates a new one.
func (r *gormMetricRepository) UpsertMetric(m *models.Metric) error {
	var existing models.Metric
	err := r.db.Where("project_id = ? AND date = ?", m.ProjectID, m.Date).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(m).Error
	}
	m.ID = existing.ID
	return r.db.Save(m).Error
}

// CreateSocialMetric inserts a new social metric snapshot.
func (r *gormMetricRepository) CreateSocialMetric(sm *models.SocialMetric) error {
	return r.db.Create(sm).Error
}

// FindSocialMetricsByProject retrieves social metrics for the last N days.
func (r *gormMetricRepository) FindSocialMetricsByProject(projectID uint, days int) ([]models.SocialMetric, error) {
	var metrics []models.SocialMetric
	err := r.db.Where("project_id = ?", projectID).
		Order("recorded_at DESC").
		Limit(days * 10). // accommodate multiple platforms per day
		Find(&metrics).Error
	return metrics, err
}

// FindLatestSocialMetric returns the most recent social metric for a given platform.
func (r *gormMetricRepository) FindLatestSocialMetric(projectID uint, platform string) (*models.SocialMetric, error) {
	var sm models.SocialMetric
	err := r.db.Where("project_id = ? AND LOWER(platform) = LOWER(?)", projectID, platform).
		Order("recorded_at DESC").First(&sm).Error
	return &sm, err
}
// FindLatestSocialMetrics returns the most recent social metric for each platform.
// Uses a single SQL query to pick the top row per LOWER(platform), preserving the
// prior ORDER BY (is_simulated ASC, recorded_at DESC, id DESC) tiebreak by
// selecting MAX(id) per platform group (id is monotonic on insert).
func (r *gormMetricRepository) FindLatestSocialMetrics(projectID uint) ([]models.SocialMetric, error) {
	var metrics []models.SocialMetric
	err := r.db.Raw(`
		SELECT * FROM social_metrics
		WHERE id IN (
			SELECT MAX(id) FROM social_metrics
			WHERE project_id = ?
			GROUP BY LOWER(platform)
		)
		ORDER BY LOWER(platform) ASC, is_simulated ASC, recorded_at DESC, id DESC
	`, projectID).Scan(&metrics).Error
	return metrics, err
}
