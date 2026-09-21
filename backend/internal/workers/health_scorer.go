package workers

import (
	"log"
	"time"

	"backend/internal/repository"
	"backend/internal/services"
)

// StartHealthScorer crawls each project URL and writes the real SEO health
// score. Runs once at startup, then every 24 hours. Projects without a URL
// are skipped. It does not invent scores.
func StartHealthScorer(
	projectRepo repository.ProjectRepository,
	crawler services.SEOCrawlerService,
) {
	log.Println("[worker:HealthScorer] started — interval: 24h")
	runHealthScorer(projectRepo, crawler)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		runHealthScorer(projectRepo, crawler)
	}
}

func runHealthScorer(projectRepo repository.ProjectRepository, crawler services.SEOCrawlerService) {
	if crawler == nil {
		log.Println("[worker:HealthScorer] no crawler configured — skipping")
		return
	}

	projects, err := projectRepo.FindAll()
	if err != nil {
		log.Printf("[worker:HealthScorer] failed to list projects: %v", err)
		return
	}

	scored := 0
	skipped := 0
	for i := range projects {
		p := &projects[i]
		if p.URL == "" {
			skipped++
			continue
		}
		result, crawlErr := crawler.Crawl(p.URL)
		if crawlErr != nil {
			log.Printf("[worker:HealthScorer] crawl failed for project %d (%s): %v", p.ID, p.URL, crawlErr)
			continue
		}
		p.HealthScore = result.Score
		if result.Score >= 75 {
			p.Health = "healthy"
		} else if result.Score >= 50 {
			p.Health = "issues"
		} else {
			p.Health = "critical"
		}
		if err := projectRepo.Update(p); err != nil {
			log.Printf("[worker:HealthScorer] update failed for project %d: %v", p.ID, err)
			continue
		}
		scored++
		log.Printf("[worker:HealthScorer] project %d (%s) score=%d", p.ID, p.URL, result.Score)
	}
	log.Printf("[worker:HealthScorer] scored %d project(s), skipped %d without URL", scored, skipped)
}
