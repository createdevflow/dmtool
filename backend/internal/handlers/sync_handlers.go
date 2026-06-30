// Package handlers — SyncHandler provides on-demand data sync for a project.
// It immediately triggers GSC + Meta + SEO crawl for a project, using any
// existing OAuth credentials the user has connected.
package handlers

import (
	"context"
	"log"
	"strconv"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
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
	gscService     services.GSCService
	metaService    services.MetaService
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

// WithGSCService attaches the GSC service so SyncProject can prefer live GSC
// data over DataForSEO when Google is connected.
func (h *SyncHandler) WithGSCService(gsc services.GSCService) *SyncHandler {
	h.gscService = gsc
	return h
}

// WithMetaService attaches the Meta service so SyncProject can pull live
// Instagram/Facebook metrics when Meta is connected.
func (h *SyncHandler) WithMetaService(meta services.MetaService) *SyncHandler {
	h.metaService = meta
	return h
}

// SyncProject triggers an immediate full sync for a project.
// Priority for traffic: GSC (if Google connected) → DataForSEO (estimated).
// Priority for social:  Meta API (if connected) → RapidAPI (estimated).
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
		"project_id": project.ID,
		"synced_at":  time.Now().Format(time.RFC3339),
		"traffic":    gin.H{"status": "skipped", "records": 0},
		"social":     gin.H{"status": "skipped", "records": 0},
		"seo":        gin.H{"status": "skipped", "score": 0},
	}

	// ── 1. Traffic Sync ──────────────────────────────────────────────────────
	// Prefer live GSC data when the user has connected Google; fall back to
	// DataForSEO estimated traffic otherwise.
	if project.URL != "" {
		gscSynced := false

		if h.gscService != nil {
			googleCred, credErr := h.oauthRepo.FindByUserAndProvider(userID, "google")
			if credErr == nil && googleCred != nil {
				accessToken, err1 := utils.Decrypt(googleCred.AccessTokenEnc, h.encKey)
				refreshToken, err2 := utils.Decrypt(googleCred.RefreshTokenEnc, h.encKey)
				if err1 == nil && err2 == nil {
					token := &oauth2.Token{
						AccessToken:  accessToken,
						RefreshToken: refreshToken,
						Expiry:       googleCred.ExpiresAt,
					}
					metrics, gscErr := h.gscService.FetchMetrics(context.Background(), project.URL, token)
					if gscErr == nil {
						for _, m := range metrics {
							m.ProjectID = project.ID
							h.metricRepo.UpsertMetric(&m)
						}
						result["traffic"] = gin.H{
							"status":  "success",
							"source":  "gsc",
							"records": len(metrics),
						}
						log.Printf("[sync] Traffic (GSC): synced %d records for project %d", len(metrics), project.ID)
						gscSynced = true
					} else {
						log.Printf("[sync] GSC fetch error for project %d: %v — falling back to DataForSEO", project.ID, gscErr)
					}
				}
			}
		}

		// Fall back to DataForSEO estimated traffic
		if !gscSynced {
			metrics, dfErr := h.dataForSEOSvc.FetchEstimatedTraffic(project.URL)
			if dfErr == nil {
				for _, m := range metrics {
					m.ProjectID = project.ID
					h.metricRepo.UpsertMetric(&m)
				}
				result["traffic"] = gin.H{
					"status":  "success",
					"source":  "dataforseo",
					"records": len(metrics),
				}
				log.Printf("[sync] Traffic (DataForSEO): synced %d records for project %d", len(metrics), project.ID)
			} else {
				result["traffic"] = gin.H{"status": "error", "message": dfErr.Error()}
				log.Printf("[sync] Traffic error for project %d: %v", project.ID, dfErr)
			}
		}
	}

	// ── 2. Social Sync ───────────────────────────────────────────────────────
	// Prefer live Meta API when connected; fall back to RapidAPI otherwise.
	if project.IGHandle != "" || project.FBHandle != "" {
		metaSynced := false

		if h.metaService != nil {
			var metaToken string
			if metaCred, credErr := h.oauthRepo.FindByUserAndProvider(userID, "meta"); credErr == nil && metaCred != nil {
				if tok, decErr := utils.Decrypt(metaCred.AccessTokenEnc, h.encKey); decErr == nil && tok != "" {
					metaToken = tok
				}
			}

			if metaToken != "" && project.IGHandle != "" {
				accounts, accErr := h.metaService.GetIGUserAccounts(metaToken)
				if accErr == nil && len(accounts) > 0 {
					targetID, targetToken := accounts[0].ID, accounts[0].AccessToken
					if targetToken == "" {
						targetToken = metaToken
					}
					sm, smErr := h.metaService.FetchInstagramMetrics(project.ID, targetID, targetToken)
					if smErr == nil && sm != nil {
						h.metricRepo.CreateSocialMetric(sm)
						result["social"] = gin.H{
							"status":    "success",
							"source":    "meta",
							"simulated": false,
							"followers": sm.Followers,
						}
						metaSynced = true
					}
				}
			}

			if !metaSynced && metaToken != "" && project.FBHandle != "" {
				accounts, accErr := h.metaService.GetFacebookPageAccounts(metaToken)
				if accErr == nil && len(accounts) > 0 {
					targetID, targetToken := accounts[0].ID, accounts[0].AccessToken
					if targetToken == "" {
						targetToken = metaToken
					}
					sm, smErr := h.metaService.FetchFacebookPageMetrics(project.ID, targetID, targetToken)
					if smErr == nil && sm != nil {
						h.metricRepo.CreateSocialMetric(sm)
						result["social"] = gin.H{
							"status":    "success",
							"source":    "meta",
							"simulated": false,
							"followers": sm.Followers,
						}
						metaSynced = true
					}
				}
			}
		}

		// Fall back to RapidAPI estimated profile
		if !metaSynced && project.IGHandle != "" {
			sm, rpErr := h.rapidAPISvc.FetchInstagramProfile(project.IGHandle)
			if rpErr == nil && sm != nil && sm.Followers > 0 {
				sm.ProjectID = project.ID
				h.metricRepo.CreateSocialMetric(sm)
				result["social"] = gin.H{
					"status":    "success",
					"source":    "rapidapi",
					"simulated": sm.IsSimulated,
					"followers": sm.Followers,
				}
			} else if rpErr != nil {
				result["social"] = gin.H{"status": "error", "message": rpErr.Error()}
			}
		}
	}

	// ── 3. SEO Crawl ─────────────────────────────────────────────────────────
	if project.URL != "" {
		crawlResult, crawlErr := h.crawlerService.Crawl(project.URL)
		if crawlErr == nil {
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
			project.HealthScore = crawlResult.Score
			if crawlResult.Score >= 75 {
				project.Health = "healthy"
			} else if crawlResult.Score >= 50 {
				project.Health = "issues"
			} else {
				project.Health = "critical"
			}
			h.projectRepo.Update(project)
			result["seo"] = gin.H{
				"status":       "success",
				"score":        crawlResult.Score,
				"issues_found": len(crawlResult.Checks),
			}
		} else {
			result["seo"] = gin.H{"status": "error", "message": crawlErr.Error()}
		}
	}

	utils.Success(c, result, &utils.ResponseMeta{
		Message: "Sync complete.",
	})
}
