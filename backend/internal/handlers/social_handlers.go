package handlers

import (
	"math"
	"strconv"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type SocialHandler struct {
	projectRepo    repository.ProjectRepository
	metricRepo     repository.MetricRepository
	oauthRepo      repository.OAuthRepository
	rapidAPISvc    services.RapidAPIService
	encKey         []byte
}

func NewSocialHandler(
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	rapidAPISvc services.RapidAPIService,
	encKey []byte,
) *SocialHandler {
	return &SocialHandler{
		projectRepo:    projectRepo,
		metricRepo:     metricRepo,
		oauthRepo:      oauthRepo,
		rapidAPISvc:    rapidAPISvc,
		encKey:         encKey,
	}
}

// SocialInsights returns the latest social metrics for a project.
// Priority: real Meta API data → public scrape data → hash-estimated fallback.
func (h *SocialHandler) SocialInsights(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)

	project, err := h.projectRepo.FindByIDAndUser(pid, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	// Fetch latest metrics from DB
	metrics, err := h.metricRepo.FindLatestSocialMetrics(pid)
	if err != nil {
		utils.InternalError(c, "Failed to fetch social insights")
		return
	}

	// If no real data in DB and we have handles, try public scraping
	if len(metrics) == 0 && project.IGHandle != "" {
		scraped, err := h.scraperService.FetchInstagramPublic(project.IGHandle)
		if err == nil && scraped != nil {
			sm := &models.SocialMetric{
				ProjectID:       pid,
				Platform:        "Instagram",
				Followers:       scraped.Followers,
				Reach:           int64(float64(scraped.Followers) * 0.15), // estimated reach ~15% of followers
				Engagement:      3.2,
				EngagementCount: int64(float64(scraped.Followers) * 0.032),
				Status:          "stable",
				IsSimulated:     scraped.IsSimulated,
				RecordedAt:      time.Now(),
			}
			// Save to DB for history
			h.metricRepo.CreateSocialMetric(sm)
			metrics = []models.SocialMetric{*sm}
		}
	}

	// If still no data, use deterministic hash-based estimates clearly marked as simulated
	if len(metrics) == 0 {
		if project.IGHandle != "" {
			h := utils.HashString(project.IGHandle)
			followers := int64(1000 + (h % 50000))
			reach := int64(float64(followers) * (2.0 + float64(h%10)/5.0))
			metrics = append(metrics, models.SocialMetric{
				Platform: "Instagram", Followers: followers, Reach: reach,
				Engagement: 4.2, EngagementCount: int64(float64(reach) * 0.042),
				Status: "growing", IsSimulated: true,
			})
		}
	}

	// Filter to only show platforms that have handles configured
	var filtered []models.SocialMetric
	for _, m := range metrics {
		show := false
		switch m.Platform {
		case "Instagram", "instagram":
			show = project.IGHandle != ""
		case "Facebook", "facebook":
			show = project.FBHandle != ""
		}
		if show {
			filtered = append(filtered, m)
		}
	}

	utils.Success(c, filtered, nil)
}

// SocialHistory returns social metric snapshots over time for charts.
func (h *SocialHandler) SocialHistory(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)

	history, err := h.metricRepo.FindSocialMetricsByProject(uint(projectID), 30)
	if err != nil {
		utils.InternalError(c, "Failed to fetch social history")
		return
	}

	utils.Success(c, history, nil)
}

// RefreshSocial triggers an immediate sync of social metrics.
// Priority: Meta API (if OAuth connected) → public scrape → hash estimate.
func (h *SocialHandler) RefreshSocial(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	pid, _ := strconv.ParseUint(projectIDStr, 10, 32)
	userID := c.MustGet("user_id").(uint)

	project, err := h.projectRepo.FindByIDAndUser(uint(pid), userID)
	if err != nil {
		utils.Forbidden(c, "Project not found or access denied")
		return
	}

	// 1. Try real Meta API if connected
	metaCred, err := h.oauthRepo.FindByUserAndProvider(userID, "meta")
	if err == nil && metaCred != nil {
		accessToken, err := utils.Decrypt(metaCred.AccessTokenEnc, h.encKey)
		if err != nil {
			utils.InternalError(c, "Failed to decrypt access token")
			return
		}

		accounts, err := h.metaService.GetIGUserAccounts(accessToken)
		if err == nil && len(accounts) > 0 {
			var targetID string
			if id, ok := accounts[project.IGHandle]; ok {
				targetID = id
			} else {
				for _, id := range accounts {
					targetID = id
					break
				}
			}

			if targetID != "" {
				sm, err := h.metaService.FetchInstagramMetrics(project.ID, targetID, accessToken)
				if err == nil {
					h.metricRepo.CreateSocialMetric(sm)
					utils.Success(c, sm, &utils.ResponseMeta{Message: "Live metrics synced from Meta API"})
					return
				}
			}
		}
	}

	// 2. Fall back to public Instagram scrape
	if project.IGHandle != "" {
		scraped, err := h.scraperService.FetchInstagramPublic(project.IGHandle)
		if err == nil && scraped != nil && scraped.Followers > 0 {
			sm := &models.SocialMetric{
				ProjectID:       project.ID,
				Platform:        "Instagram",
				Followers:       scraped.Followers,
				Reach:           int64(float64(scraped.Followers) * 0.15),
				Engagement:      3.2,
				EngagementCount: int64(float64(scraped.Followers) * 0.032),
				Status:          "stable",
				IsSimulated:     scraped.IsSimulated,
				RecordedAt:      time.Now(),
			}
			h.metricRepo.CreateSocialMetric(sm)
			msg := "Metrics fetched from public Instagram profile"
			if scraped.IsSimulated {
				msg = "Using estimated data — profile may be private or Instagram rate-limited. Connect Meta API for live metrics."
			}
			utils.Success(c, sm, &utils.ResponseMeta{Message: msg})
			return
		}
	}

	// 3. Hash-based estimate fallback
	hash := utils.HashString(project.IGHandle)
	followers := int64(45 + int(hash%120))
	reach := int64(float64(followers) * (1.2 + float64(hash%5)/10.0))
	engagementRate := 5.2 + float64(hash%30)/10.0

	sm := &models.SocialMetric{
		ProjectID:       project.ID,
		Platform:        "Instagram",
		Followers:       followers,
		Reach:           reach,
		Engagement:      math.Round(engagementRate*100) / 100,
		EngagementCount: int64(float64(reach) * (engagementRate / 100.0)),
		Status:          "stable",
		IsSimulated:     true,
		RecordedAt:      time.Now(),
	}

	h.metricRepo.CreateSocialMetric(sm)
	utils.Success(c, sm, &utils.ResponseMeta{Message: "Using estimated mode. Connect Meta API for 100% accurate live metrics."})
}

// PublicProfile returns public social profile data (follower count, bio, etc.)
// by scraping the public Instagram profile page. No OAuth required.
func (h *SocialHandler) PublicProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	handle := c.Query("handle")
	platform := c.DefaultQuery("platform", "instagram")

	if handle == "" && projectIDStr != "" {
		projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
		project, err := h.projectRepo.FindByIDAndUser(uint(projectID), userID)
		if err == nil && project != nil {
			handle = project.IGHandle
		}
	}

	if handle == "" {
		utils.BadRequest(c, "handle is required", "MISSING_HANDLE")
		return
	}

	switch platform {
	case "instagram":
		data, err := h.scraperService.FetchInstagramPublic(handle)
		if err != nil {
			utils.InternalError(c, "Failed to fetch profile: "+err.Error())
			return
		}
		utils.Success(c, data, nil)
	default:
		utils.BadRequest(c, "Unsupported platform. Supported: instagram", "UNSUPPORTED_PLATFORM")
	}
}
