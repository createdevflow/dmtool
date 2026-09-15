package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

func normalizeHandle(raw string) string {
	return utils.NormalizeSocialHandle(raw)
}

func resolveTargetAccount(accounts []services.MetaAccount, handle string) (string, string) {
	if len(accounts) == 0 {
		return "", ""
	}
	if handle != "" && handle != "auto" {
		norm := normalizeHandle(handle)
		for _, acc := range accounts {
			if acc.Username == norm || acc.Name == norm {
				return acc.ID, acc.AccessToken
			}
		}
	}
	return accounts[0].ID, accounts[0].AccessToken
}

type SocialHandler struct {
	projectRepo         repository.ProjectRepository
	metricRepo          repository.MetricRepository
	oauthRepo           repository.OAuthRepository
	rapidAPISvc         services.RapidAPIService
	scraperService      services.SocialScraperService
	metaService         services.MetaService
	linkedinService     services.LinkedinService
	metaPageAccessToken string
	linkedinAccessToken string
	encKey              []byte
}

func NewSocialHandler(
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	rapidAPISvc services.RapidAPIService,
	scraperService services.SocialScraperService,
	metaService services.MetaService,
	linkedinService services.LinkedinService,
	metaPageAccessToken string,
	linkedinAccessToken string,
	encKey []byte,
) *SocialHandler {
	return &SocialHandler{
		projectRepo:         projectRepo,
		metricRepo:          metricRepo,
		oauthRepo:           oauthRepo,
		rapidAPISvc:         rapidAPISvc,
		scraperService:      scraperService,
		metaService:         metaService,
		linkedinService:     linkedinService,
		metaPageAccessToken: metaPageAccessToken,
		linkedinAccessToken: linkedinAccessToken,
		encKey:              encKey,
	}
}

func (h *SocialHandler) resolveMetaAccessToken(userID uint) string {
	if metaCred, err := h.oauthRepo.FindByUserAndProvider(userID, "meta"); err == nil && metaCred != nil {
		if accessToken, decErr := utils.Decrypt(metaCred.AccessTokenEnc, h.encKey); decErr == nil && accessToken != "" {
			return accessToken
		}
	}
	return h.metaPageAccessToken
}

func (h *SocialHandler) resolveLinkedinAccessToken(userID uint) string {
	if linkedinCred, err := h.oauthRepo.FindByUserAndProvider(userID, "linkedin"); err == nil && linkedinCred != nil {
		if accessToken, decErr := utils.Decrypt(linkedinCred.AccessTokenEnc, h.encKey); decErr == nil && accessToken != "" {
			return accessToken
		}
	}
	return h.linkedinAccessToken
}

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

	needsRealData := len(metrics) == 0
	if len(metrics) > 0 {
		allSimulated := true
		for _, m := range metrics {
			if !m.IsSimulated {
				allSimulated = false
				break
			}
		}
		needsRealData = allSimulated
		if !needsRealData {
			latest := metrics[0]
			emptyLiveSnapshot := latest.Reach == 0 && latest.ProfileVisits == 0 && latest.ExternalLinkTaps == 0 && latest.EngagementCount == 0
			if emptyLiveSnapshot {
				needsRealData = true
			}
		}
	}

	// Try real Meta API automatically if we only have simulated data
	if needsRealData {
		accessToken := h.resolveMetaAccessToken(userID)
		if accessToken != "" {
			if project.IGHandle != "" {
				accounts, _ := h.metaService.GetIGUserAccounts(accessToken)
				if len(accounts) > 0 {
					targetID, targetToken := resolveTargetAccount(accounts, project.IGHandle)
					if targetID != "" {
						if targetToken == "" {
							targetToken = accessToken
						}
						sm, fetchErr := h.metaService.FetchInstagramMetrics(project.ID, targetID, targetToken)
						if fetchErr == nil && sm != nil {
							h.metricRepo.CreateSocialMetric(sm)
							metrics = []models.SocialMetric{*sm}
						}
					}
				}
			}
			if len(metrics) == 0 && project.FBHandle != "" {
				accounts, _ := h.metaService.GetFacebookPageAccounts(accessToken)
				if len(accounts) > 0 {
					targetID, targetToken := resolveTargetAccount(accounts, project.FBHandle)
					if targetID != "" {
						if targetToken == "" {
							targetToken = accessToken
						}
						sm, fetchErr := h.metaService.FetchFacebookPageMetrics(project.ID, targetID, targetToken)
						if fetchErr == nil && sm != nil {
							h.metricRepo.CreateSocialMetric(sm)
							metrics = []models.SocialMetric{*sm}
						}
					}
				}
			}
		}
	}

	// Try LinkedIn automatically when credentials exist and LinkedIn metrics are not already present
	linkedinToken := h.resolveLinkedinAccessToken(userID)
	hasLinkedinMetric := false
	for _, m := range metrics {
		if strings.EqualFold(m.Platform, "linkedin") || strings.EqualFold(m.Platform, "LinkedIn") {
			hasLinkedinMetric = true
			break
		}
	}
	if !hasLinkedinMetric && linkedinToken != "" {
		if sm, err := h.linkedinService.FetchLinkedInMetrics(project.ID, project.LinkedinHandle, linkedinToken); err == nil && sm != nil {
			h.metricRepo.CreateSocialMetric(sm)
			metrics = append(metrics, *sm)
		}
	}

	// Public Instagram scrape only when the fetch is real (not a hash fallback).
	if len(metrics) == 0 && project.IGHandle != "" {
		scraped, err := h.scraperService.FetchInstagramPublic(project.IGHandle)
		if err == nil && scraped != nil && !scraped.IsSimulated && scraped.Followers > 0 {
			sm := &models.SocialMetric{
				ProjectID:         pid,
				Platform:          "Instagram",
				Followers:         scraped.Followers,
				FollowingCount:    scraped.Following,
				PostsCount:        scraped.PostCount,
				DisplayName:       scraped.FullName,
				Biography:         scraped.Bio,
				ProfilePictureURL: scraped.ProfilePic,
				Status:            "stable",
				IsSimulated:       false,
				RecordedAt:        time.Now(),
			}
			h.metricRepo.CreateSocialMetric(sm)
			metrics = []models.SocialMetric{*sm}
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
		case "LinkedIn", "linkedin":
			show = true
		}
		if show && !m.IsSimulated {
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

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 {
			days = parsed
		}
	}

	history, err := h.metricRepo.FindSocialMetricsByProject(uint(projectID), days)
	if err != nil {
		utils.InternalError(c, "Failed to fetch social history")
		return
	}

	real := make([]models.SocialMetric, 0, len(history))
	for _, m := range history {
		if !m.IsSimulated {
			real = append(real, m)
		}
	}

	utils.Success(c, real, nil)
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
	accessToken := h.resolveMetaAccessToken(userID)
	if accessToken != "" {

		if project.IGHandle != "" {
			accounts, err := h.metaService.GetIGUserAccounts(accessToken)
			if err == nil && len(accounts) > 0 {
				targetID, targetToken := resolveTargetAccount(accounts, project.IGHandle)

				if targetID != "" {
					if targetToken == "" {
						targetToken = accessToken
					}
					sm, err := h.metaService.FetchInstagramMetrics(project.ID, targetID, targetToken)
					if err == nil {
						h.metricRepo.CreateSocialMetric(sm)
						utils.Success(c, sm, &utils.ResponseMeta{Message: "Live metrics synced from Meta API"})
						return
					}
				}
			}
		}

		if project.FBHandle != "" {
			accounts, err := h.metaService.GetFacebookPageAccounts(accessToken)
			if err == nil && len(accounts) > 0 {
				targetID, targetToken := resolveTargetAccount(accounts, project.FBHandle)

				if targetID != "" {
					if targetToken == "" {
						targetToken = accessToken
					}
					sm, err := h.metaService.FetchFacebookPageMetrics(project.ID, targetID, targetToken)
					if err == nil {
						h.metricRepo.CreateSocialMetric(sm)
						utils.Success(c, sm, &utils.ResponseMeta{Message: "Live Facebook Page metrics synced from Meta API"})
						return
					}
				}
			}
		}
	}

	// 1.5. Try LinkedIn API if handle exists
	linkedinToken := h.resolveLinkedinAccessToken(userID)
	if linkedinToken != "" {
		fmt.Printf("[RefreshSocial] Calling FetchLinkedInMetrics with token (len=%d)\n", len(linkedinToken))
		sm, err := h.linkedinService.FetchLinkedInMetrics(project.ID, project.LinkedinHandle, linkedinToken)
		if err == nil && sm != nil {
			fmt.Printf("[RefreshSocial] LinkedIn fetch succeeded: %d followers\n", sm.Followers)
			h.metricRepo.CreateSocialMetric(sm)
			utils.Success(c, sm, &utils.ResponseMeta{Message: "Live LinkedIn metrics synced"})
			return
		}
		if err != nil {
			fmt.Printf("[RefreshSocial] LinkedIn fetch failed: %v\n", err)
			utils.IntegrationError(c, "LinkedIn metrics refresh failed. Reconnect LinkedIn to restore live analytics.")
			return
		}
	} else {
		fmt.Printf("[RefreshSocial] No LinkedIn token found for user %d\n", userID)
	}

	if project.IGHandle != "" {
		scraped, err := h.scraperService.FetchInstagramPublic(project.IGHandle)
		if err == nil && scraped != nil && !scraped.IsSimulated && scraped.Followers > 0 {
			sm := &models.SocialMetric{
				ProjectID:         project.ID,
				Platform:          "Instagram",
				Followers:         scraped.Followers,
				FollowingCount:    scraped.Following,
				PostsCount:        scraped.PostCount,
				DisplayName:       scraped.FullName,
				Biography:         scraped.Bio,
				ProfilePictureURL: scraped.ProfilePic,
				Status:            "stable",
				IsSimulated:       false,
				RecordedAt:        time.Now(),
			}
			h.metricRepo.CreateSocialMetric(sm)
			utils.Success(c, sm, &utils.ResponseMeta{Message: "Metrics fetched from public Instagram profile"})
			return
		}
	}

	utils.Success(c, gin.H{}, &utils.ResponseMeta{
		Message: "Connect Meta or LinkedIn, or a public Instagram handle, to load social metrics.",
	})
}

// PublicProfile returns public social profile data (follower count, bio, avatar, etc.).
func (h *SocialHandler) PublicProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	handle := c.Query("handle")
	platform := c.DefaultQuery("platform", "instagram")
	var project *models.Project

	if handle == "" && projectIDStr != "" {
		projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
		found, err := h.projectRepo.FindByIDAndUser(uint(projectID), userID)
		if err == nil && found != nil {
			project = found
			switch platform {
			case "facebook":
				handle = found.FBHandle
			default:
				handle = found.IGHandle
			}
		}
	}

	if handle == "" {
		utils.BadRequest(c, "handle is required", "MISSING_HANDLE")
		return
	}

	accessToken := h.resolveMetaAccessToken(userID)

	switch platform {
	case "instagram":
		data, err := h.scraperService.FetchInstagramPublic(handle)
		if err != nil {
			utils.InternalError(c, "Failed to fetch profile: "+err.Error())
			return
		}
		utils.Success(c, data, nil)
	case "facebook":
		if accessToken == "" {
			utils.BadRequest(c, "Meta access token is required for Facebook page lookup", "MISSING_META_TOKEN")
			return
		}
		accounts, err := h.metaService.GetFacebookPageAccounts(accessToken)
		if err != nil || len(accounts) == 0 {
			utils.InternalError(c, "Failed to resolve Facebook page")
			return
		}
		targetID, targetToken := resolveTargetAccount(accounts, handle)
		if targetID == "" {
			utils.NotFound(c, "Facebook page not found")
			return
		}
		if targetToken == "" {
			targetToken = accessToken
		}
		profileProjectID := uint(0)
		if project != nil {
			profileProjectID = project.ID
		}
		profile, err := h.metaService.FetchFacebookPageMetrics(profileProjectID, targetID, targetToken)
		if err != nil {
			utils.InternalError(c, "Failed to fetch Facebook page profile")
			return
		}
		utils.Success(c, profile, nil)
	default:
		utils.BadRequest(c, "Unsupported platform. Supported: instagram, facebook", "UNSUPPORTED_PLATFORM")
	}
}

// RelatedProfilesData is the response struct for recommended profiles
type RelatedProfilesData struct {
	Handle      string   `json:"handle"`
	Name        string   `json:"name"`
	Avatar      string   `json:"avatar"`
	Followers   int64    `json:"followers"`
	Following   int64    `json:"following"`
	Posts       int64    `json:"posts"`
	MatchScore  int      `json:"match_score"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

// RelatedProfiles is not implemented. Returns an empty list.
func (h *SocialHandler) RelatedProfiles(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	project, err := h.projectRepo.FindByIDAndUser(uint(projectID), userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	utils.Success(c, []RelatedProfilesData{}, &utils.ResponseMeta{
		Message: "Profile discovery is not available.",
	})
}

