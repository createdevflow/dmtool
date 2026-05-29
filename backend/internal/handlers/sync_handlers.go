// Package handlers — SyncHandler provides on-demand data sync for a project.
// It immediately triggers GSC + Meta + SEO crawl for a project, using any
// existing OAuth credentials the user has connected.
package handlers

import (
	"log"
	"strconv"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// SyncHandler orchestrates on-demand full data sync for a project.
type SyncHandler struct {
	projectRepo    repository.ProjectRepository
	metricRepo     repository.MetricRepository
	oauthRepo      repository.OAuthRepository
	seoRepo        repository.SEORepository
	insightRepo    repository.InsightRepository
	dataForSEOSvc  services.DataForSEOService
	rapidAPISvc    services.RapidAPIService
	crawlerService services.SEOCrawlerService
	encKey         []byte
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	seoRepo repository.SEORepository,
	insightRepo repository.InsightRepository,
	dataForSEOSvc services.DataForSEOService,
	rapidAPISvc services.RapidAPIService,
	crawler services.SEOCrawlerService,
	encKey []byte,
) *SyncHandler {
	return &SyncHandler{
		projectRepo:    projectRepo,
		metricRepo:     metricRepo,
		oauthRepo:      oauthRepo,
		seoRepo:        seoRepo,
		insightRepo:    insightRepo,
		dataForSEOSvc:  dataForSEOSvc,
		rapidAPISvc:    rapidAPISvc,
		crawlerService: crawler,
		encKey:         encKey,
	}
}

// SyncProject triggers an immediate full sync for a project.
// POST /projects/:id/sync
func (h *SyncHandler) SyncProject(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid project ID", "INVALID_ID")
		return
	}

	project, err := h.projectRepo.FindByIDAndUser(uint(id), userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	result := gin.H{
		"project_id":   project.ID,
		"synced_at":    time.Now().Format(time.RFC3339),
		"traffic":      gin.H{"status": "skipped", "records": 0},
		"social":       gin.H{"status": "skipped", "records": 0},
		"seo":          gin.H{"status": "skipped", "score": 0},
	}

	// ── 1. Traffic Sync (DataForSEO) ────────────────────────────────────────
	if project.URL != "" {
		metrics, err := h.dataForSEOSvc.FetchEstimatedTraffic(project.URL)
		if err == nil {
			for _, m := range metrics {
				m.ProjectID = project.ID
				h.metricRepo.UpsertMetric(&m)
			}
			result["traffic"] = gin.H{"status": "success", "records": len(metrics)}
			log.Printf("[sync] Traffic: synced %d metric records for project %d", len(metrics), project.ID)
		} else {
			result["traffic"] = gin.H{"status": "error", "message": err.Error()}
			log.Printf("[sync] Traffic error for project %d: %v", project.ID, err)
		}
	}

	// ── 2. Social Sync (RapidAPI) ─────────────────────────────────────────────
	if project.IGHandle != "" {
		sm, err := h.rapidAPISvc.FetchInstagramProfile(project.IGHandle)
		if err == nil && sm != nil && sm.Followers > 0 {
			sm.ProjectID = project.ID
			h.metricRepo.CreateSocialMetric(sm)
			result["social"] = gin.H{"status": "success", "simulated": sm.IsSimulated, "followers": sm.Followers}
		} else if err != nil {
			result["social"] = gin.H{"status": "error", "message": err.Error()}
		}
	}

	// ── 4. SEO crawl (if URL is set) ─────────────────────────────────────────
	if project.URL != "" {
		crawlResult, err := h.crawlerService.Crawl(project.URL)
		if err == nil {
			// Save issues
			for _, check := range crawlResult.Checks {
				if check.Status == services.CheckFail || check.Status == services.CheckWarning {
					issue := &models.SEOIssue{
						ProjectID: project.ID,
						URL:       project.URL,
						Severity:  check.Severity,
						Category:  check.Category,
						Detail:    check.Label + ": " + check.Detail,
					}
					h.seoRepo.CreateIssue(issue)
				}
			}
			// Update health score
			project.HealthScore = crawlResult.Score
			if crawlResult.Score >= 75 {
				project.Health = "healthy"
			} else if crawlResult.Score >= 50 {
				project.Health = "issues"
			} else {
				project.Health = "critical"
			}
			h.projectRepo.Update(project)
			result["seo"] = gin.H{"status": "success", "score": crawlResult.Score, "issues_found": len(crawlResult.Checks)}
		} else {
			result["seo"] = gin.H{"status": "error", "message": err.Error()}
		}
	}

	utils.Success(c, result, &utils.ResponseMeta{
		Message: "Sync complete. Data fetched from central providers.",
	})
}
