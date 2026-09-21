// Package handlers — SyncHandler provides on-demand data sync for a project.
// It pulls GSC + Meta + LinkedIn + SEO crawl using connected OAuth credentials.
// Nothing connected → 200 with skipped statuses, no simulated rows.
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
	projectRepo     repository.ProjectRepository
	metricRepo      repository.MetricRepository
	oauthRepo       repository.OAuthRepository
	seoRepo         repository.SEORepository
	gscService      services.GSCService
	metaService     services.MetaService
	linkedinService services.LinkedinService
	keywordService  services.KeywordService
	crawlerService  services.SEOCrawlerService
	encKey          []byte
	metaPageToken   string
	linkedinToken   string
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	seoRepo repository.SEORepository,
	gscService services.GSCService,
	metaService services.MetaService,
	linkedinService services.LinkedinService,
	keywordService services.KeywordService,
	crawler services.SEOCrawlerService,
	encKey []byte,
	metaPageToken string,
	linkedinToken string,
) *SyncHandler {
	return &SyncHandler{
		projectRepo:     projectRepo,
		metricRepo:      metricRepo,
		oauthRepo:       oauthRepo,
		seoRepo:         seoRepo,
		gscService:      gscService,
		metaService:     metaService,
		linkedinService: linkedinService,
		keywordService:  keywordService,
		crawlerService:  crawler,
		encKey:          encKey,
		metaPageToken:   metaPageToken,
		linkedinToken:   linkedinToken,
	}
}

func skipped() gin.H {
	return gin.H{"status": "skipped", "records": 0}
}

func (h *SyncHandler) persistSocial(sm *models.SocialMetric) bool {
	if sm == nil || sm.IsSimulated {
		return false
	}
	if err := h.metricRepo.CreateSocialMetric(sm); err != nil {
		log.Printf("[sync] persist social metric: %v", err)
		return false
	}
	return true
}

func (h *SyncHandler) resolveMetaAccessToken(userID uint) string {
	if metaCred, err := h.oauthRepo.FindByUserAndProvider(userID, "meta"); err == nil && metaCred != nil {
		if accessToken, decErr := utils.Decrypt(metaCred.AccessTokenEnc, h.encKey); decErr == nil && accessToken != "" {
			return accessToken
		}
	}
	return h.metaPageToken
}

func (h *SyncHandler) resolveLinkedinAccessToken(userID uint) string {
	if linkedinCred, err := h.oauthRepo.FindByUserAndProvider(userID, "linkedin"); err == nil && linkedinCred != nil {
		if accessToken, decErr := utils.Decrypt(linkedinCred.AccessTokenEnc, h.encKey); decErr == nil && accessToken != "" {
			return accessToken
		}
	}
	return h.linkedinToken
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
		"project_id": project.ID,
		"synced_at":  time.Now().Format(time.RFC3339),
		"traffic":    skipped(),
		"rankings":   skipped(),
		"social":     skipped(),
		"seo":        skipped(),
	}

	ctx := context.Background()

	// ── 1. GSC traffic + query positions (only if Google is connected) ──────
	googleCred, gErr := h.oauthRepo.FindByUserAndProvider(userID, "google")
	if gErr == nil && googleCred != nil && project.URL != "" && h.gscService != nil {
		accessToken, err1 := utils.Decrypt(googleCred.AccessTokenEnc, h.encKey)
		refreshToken, err2 := utils.Decrypt(googleCred.RefreshTokenEnc, h.encKey)
		if err1 != nil || err2 != nil {
			result["traffic"] = gin.H{"status": "error", "message": "failed to decrypt Google credentials"}
			log.Printf("[sync] GSC decrypt error for project %d", project.ID)
		} else {
			token := &oauth2.Token{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				Expiry:       googleCred.ExpiresAt,
			}

			metrics, mErr := h.gscService.FetchMetrics(ctx, project.URL, token)
			if mErr != nil {
				result["traffic"] = gin.H{"status": "error", "message": mErr.Error()}
				log.Printf("[sync] GSC traffic error for project %d: %v", project.ID, mErr)
			} else {
				for i := range metrics {
					metrics[i].ProjectID = project.ID
					if metrics[i].Source == "" {
						metrics[i].Source = models.MetricSourceGSC
					}
					h.metricRepo.UpsertMetric(&metrics[i])
				}
				result["traffic"] = gin.H{"status": "success", "records": len(metrics)}
				log.Printf("[sync] GSC traffic: %d records for project %d", len(metrics), project.ID)
			}

			if h.keywordService != nil {
				kws, kErr := h.keywordService.FetchGSCKeywords(ctx, project.URL, token, h.gscService.OAuthConfig())
				if kErr != nil {
					result["rankings"] = gin.H{"status": "error", "message": kErr.Error()}
					log.Printf("[sync] GSC rankings error for project %d: %v", project.ID, kErr)
				} else {
					for i := range kws {
						kws[i].ProjectID = project.ID
					}
					if len(kws) > 0 {
						h.seoRepo.UpsertKeywords(kws)
					}
					result["rankings"] = gin.H{"status": "success", "records": len(kws)}
					log.Printf("[sync] GSC rankings: %d queries for project %d", len(kws), project.ID)
				}
			}
		}
	}

	// ── 2. Social (Meta / LinkedIn only — no RapidAPI, no simulated rows) ──
	socialRecords := 0
	socialErr := ""

	accessToken := h.resolveMetaAccessToken(userID)
	if accessToken != "" && h.metaService != nil {
		if project.IGHandle != "" {
			accounts, accErr := h.metaService.GetIGUserAccounts(accessToken)
			if accErr != nil {
				socialErr = accErr.Error()
			} else if len(accounts) > 0 {
				targetID, targetToken := resolveTargetAccount(accounts, project.IGHandle)
				if targetID != "" {
					if targetToken == "" {
						targetToken = accessToken
					}
					sm, fetchErr := h.metaService.FetchInstagramMetrics(project.ID, targetID, targetToken)
					if fetchErr != nil {
						socialErr = fetchErr.Error()
					} else if h.persistSocial(sm) {
						socialRecords++
					}
				}
			}
		}
		if project.FBHandle != "" {
			accounts, accErr := h.metaService.GetFacebookPageAccounts(accessToken)
			if accErr != nil {
				socialErr = accErr.Error()
			} else if len(accounts) > 0 {
				targetID, targetToken := resolveTargetAccount(accounts, project.FBHandle)
				if targetID != "" {
					if targetToken == "" {
						targetToken = accessToken
					}
					sm, fetchErr := h.metaService.FetchFacebookPageMetrics(project.ID, targetID, targetToken)
					if fetchErr != nil {
						socialErr = fetchErr.Error()
					} else if h.persistSocial(sm) {
						socialRecords++
					}
				}
			}
		}
	}

	linkedinToken := h.resolveLinkedinAccessToken(userID)
	if linkedinToken != "" && h.linkedinService != nil {
		sm, liErr := h.linkedinService.FetchLinkedInMetrics(project.ID, project.LinkedinHandle, linkedinToken)
		if liErr != nil {
			socialErr = liErr.Error()
		} else if h.persistSocial(sm) {
			socialRecords++
		}
	}

	if socialRecords > 0 {
		result["social"] = gin.H{"status": "success", "records": socialRecords}
	} else if socialErr != "" {
		result["social"] = gin.H{"status": "error", "message": socialErr}
	}

	// ── 3. SEO crawl (if URL is set) ────────────────────────────────────────
	if project.URL != "" && h.crawlerService != nil {
		crawlResult, crawlErr := h.crawlerService.Crawl(project.URL)
		if crawlErr != nil {
			result["seo"] = gin.H{"status": "error", "message": crawlErr.Error()}
			log.Printf("[sync] SEO crawl error for project %d: %v", project.ID, crawlErr)
		} else {
			_ = h.seoRepo.ReplaceOpenIssues(project.ID, issuesFromChecks(project.ID, project.URL, crawlResult.Checks))
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
		}
	}

	utils.Success(c, result, &utils.ResponseMeta{
		Message: "Sync complete.",
	})
}
