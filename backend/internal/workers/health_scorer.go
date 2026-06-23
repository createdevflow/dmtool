
// Package workers contains the HealthScorer background worker.
// It runs every 24 hours, crawls each project URL with the SEO crawler,
// and updates Project.HealthScore with the real audit result.
package workers

import (
	"log"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"

	"gorm.io/gorm"
)

// StartHealthScorer runs every 24 hours and re-scores every project's SEO health.
func StartHealthScorer(
	db *gorm.DB,
	projectRepo repository.ProjectRepository,
	seoRepo repository.SEORepository,
) {
	crawler := services.NewSEOCrawlerService()

	log.Println("[worker:HealthScorer] started — interval: 24h")
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run immediately on startup so scores are populated from first launch
	runHealthScorer(db, projectRepo, seoRepo, crawler)

	for range ticker.C {
		runHealthScorer(db, projectRepo, seoRepo, crawler)
	}
}

func runHealthScorer(
	db *gorm.DB,
	projectRepo repository.ProjectRepository,
	seoRepo repository.SEORepository,
	crawler services.SEOCrawlerService,
) {
	log.Println("[worker:HealthScorer] scoring all projects...")

	projects, err := projectRepo.FindAll()
	if err != nil {
		log.Printf("[worker:HealthScorer] failed to fetch projects: %v", err)
		return
	}

	for _, project := range projects {
		if project.URL == "" {
			continue
		}

		// Normalize URL
		targetURL := project.URL
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
			targetURL = "https://" + targetURL
		}

		log.Printf("[worker:HealthScorer] crawling project %d: %s", project.ID, targetURL)

		result, err := crawler.Crawl(targetURL)
		if err != nil {
			log.Printf("[worker:HealthScorer] crawl failed for project %d: %v", project.ID, err)
			continue
		}

		// Update project health score
		if err := db.Model(&models.Project{}).Where("id = ?", project.ID).
			Update("health_score", result.Score).Error; err != nil {
			log.Printf("[worker:HealthScorer] failed to update health score for project %d: %v", project.ID, err)
			continue
		}

		// Persist SEO issues found — clear all open unresolved issues for this project
		// (auto-scorer replaces stale findings on each run), then insert fresh ones.
		if len(result.Checks) > 0 {
			db.Where("project_id = ? AND resolved_at IS NULL", project.ID).
				Delete(&models.SEOIssue{})

			for _, check := range result.Checks {
				if check.Status == services.CheckPass {
					continue // only store failures and warnings
				}
				issue := models.SEOIssue{
					ProjectID: project.ID,
					URL:       targetURL,
					Severity:  mapSeverity(check.Severity),
					Category:  check.Category,
					Detail:    check.Label + ": " + check.Detail,
				}
				seoRepo.CreateIssue(&issue)
			}
		}

		log.Printf("[worker:HealthScorer] project %d (%s) scored %d/100", project.ID, project.URL, result.Score)
	}

	log.Println("[worker:HealthScorer] scoring complete")
}

// mapSeverity converts the crawler's severity strings to the model's constants.
func mapSeverity(s string) string {
	switch strings.ToLower(s) {
	case "high":
		return models.SeverityHigh
	case "medium", "med":
		return models.SeverityMed
	default:
		return models.SeverityLow
	}
}
