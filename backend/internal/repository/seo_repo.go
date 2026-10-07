// Package repository provides SEO and keyword persistence operations.
package repository

import (
	"errors"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

// SEORepository manages SEO issues and keyword results.
type SEORepository interface {
	CreateIssue(issue *models.SEOIssue) error
	ReplaceOpenIssues(projectID uint, issues []models.SEOIssue) error
	FindOpenIssues(projectID uint, severity string) ([]models.SEOIssue, error)
	ResolveIssue(id, projectID uint) error

	UpsertKeywords(results []models.KeywordResult) error
	FindKeywords(projectID uint, seed string) ([]models.KeywordResult, bool, error)

	ReplaceDimension(projectID uint, dimension string, rows []models.GSCBreakdown) error
	FindBreakdowns(projectID uint, dimension string) ([]models.GSCBreakdown, error)

	FindVendorDomainSnapshot(domain string, locationCode int, languageCode string) (*models.VendorDomainSnapshot, error)
	SaveVendorDomainSnapshot(row *models.VendorDomainSnapshot) error

	FindVendorAIVisibilitySnapshot(domain string, locationCode int, languageCode string) (*models.VendorAIVisibilitySnapshot, error)
	SaveVendorAIVisibilitySnapshot(row *models.VendorAIVisibilitySnapshot) error
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

// ReplaceOpenIssues deletes open issues for the project and inserts this run's
// findings so a re-audit does not inflate the open count.
func (r *gormSEORepository) ReplaceOpenIssues(projectID uint, issues []models.SEOIssue) error {
	if err := r.db.Where("project_id = ? AND resolved_at IS NULL", projectID).
		Delete(&models.SEOIssue{}).Error; err != nil {
		return err
	}
	if len(issues) == 0 {
		return nil
	}
	return r.db.Create(&issues).Error
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

// FindKeywords returns cached keyword results for a project.
// An empty seed returns every keyword for the project (used by Rank Tracking).
// The bool return indicates whether the cache is still valid (< 24 hours old).
func (r *gormSEORepository) FindKeywords(projectID uint, seed string) ([]models.KeywordResult, bool, error) {
	var results []models.KeywordResult
	q := r.db.Where("project_id = ?", projectID)
	if seed != "" {
		q = q.Where("seed = ?", seed)
	}
	err := q.Find(&results).Error
	if err != nil || len(results) == 0 {
		return nil, false, err
	}
	fresh := time.Since(results[0].UpdatedAt) < 24*time.Hour
	return results, fresh, nil
}

// ReplaceDimension deletes stored GSC rows for one dimension and inserts this fetch.
// An empty rows slice clears that dimension (successful empty GSC response).
func (r *gormSEORepository) ReplaceDimension(projectID uint, dimension string, rows []models.GSCBreakdown) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ? AND dimension = ?", projectID, dimension).
			Delete(&models.GSCBreakdown{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for i := range rows {
			rows[i].ProjectID = projectID
			rows[i].Dimension = dimension
		}
		return tx.Create(&rows).Error
	})
}

// FindBreakdowns returns GSC dimension rows, highest clicks first.
// Empty dimension returns every stored dimension for the project.
func (r *gormSEORepository) FindBreakdowns(projectID uint, dimension string) ([]models.GSCBreakdown, error) {
	var rows []models.GSCBreakdown
	q := r.db.Where("project_id = ?", projectID)
	if dimension != "" {
		q = q.Where("dimension = ?", dimension)
	}
	err := q.Order("clicks DESC").Find(&rows).Error
	return rows, err
}

func (r *gormSEORepository) FindVendorDomainSnapshot(domain string, locationCode int, languageCode string) (*models.VendorDomainSnapshot, error) {
	var row models.VendorDomainSnapshot
	err := r.db.Where("domain = ? AND location_code = ? AND language_code = ?", domain, locationCode, languageCode).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *gormSEORepository) SaveVendorDomainSnapshot(row *models.VendorDomainSnapshot) error {
	if row == nil {
		return nil
	}
	var existing models.VendorDomainSnapshot
	err := r.db.Where("domain = ? AND location_code = ? AND language_code = ?", row.Domain, row.LocationCode, row.LanguageCode).
		First(&existing).Error
	if err == nil {
		existing.Source = row.Source
		existing.Payload = row.Payload
		existing.FetchedAt = row.FetchedAt
		return r.db.Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.Create(row).Error
}

func (r *gormSEORepository) FindVendorAIVisibilitySnapshot(domain string, locationCode int, languageCode string) (*models.VendorAIVisibilitySnapshot, error) {
	var row models.VendorAIVisibilitySnapshot
	err := r.db.Where("domain = ? AND location_code = ? AND language_code = ?", domain, locationCode, languageCode).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *gormSEORepository) SaveVendorAIVisibilitySnapshot(row *models.VendorAIVisibilitySnapshot) error {
	if row == nil {
		return nil
	}
	var existing models.VendorAIVisibilitySnapshot
	err := r.db.Where("domain = ? AND location_code = ? AND language_code = ?", row.Domain, row.LocationCode, row.LanguageCode).
		First(&existing).Error
	if err == nil {
		existing.Source = row.Source
		existing.Payload = row.Payload
		existing.FetchedAt = row.FetchedAt
		return r.db.Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.Create(row).Error
}
