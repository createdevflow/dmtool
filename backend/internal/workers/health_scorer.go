package workers

import (
	"log"
	"time"

	"backend/internal/repository"

	"gorm.io/gorm"
)

// StartHealthScorer runs every 24 hours and re-runs the SEO audit for each
// project, updating Project.HealthScore.  Full implementation in Phase 3.
func StartHealthScorer(
	db *gorm.DB,
	projectRepo repository.ProjectRepository,
	seoRepo repository.SEORepository,
) {
	log.Println("[worker:HealthScorer] started — interval: 24h")
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("[worker:HealthScorer] scoring all projects...")
		// Phase 3: iterate projects, call seo/analyzer.go, update HealthScore
		_ = db
		_ = projectRepo
		_ = seoRepo
		log.Println("[worker:HealthScorer] scoring complete")
	}
}
