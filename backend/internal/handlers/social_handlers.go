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
	projectRepo    repository.ProjectRepository
	metricRepo     repository.MetricRepository
	oauthRepo      repository.OAuthRepository
	rapidAPISvc    services.RapidAPIService
	scraperService services.SocialScraperService
	metaService    services.MetaService
	metaPageAccessToken string
	encKey         []byte
}

func NewSocialHandler(
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	rapidAPISvc services.RapidAPIService,
	scraperService services.SocialScraperService,
	metaService services.MetaService,
	metaPageAccessToken string,
	encKey []byte,
) *SocialHandler {
	return &SocialHandler{
		projectRepo:    projectRepo,
		metricRepo:     metricRepo,
		oauthRepo:      oauthRepo,
		rapidAPISvc:    rapidAPISvc,
		scraperService: scraperService,
		metaService:    metaService,
		metaPageAccessToken: metaPageAccessToken,
		encKey:         encKey,
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
		} else if project.FBHandle != "" {
			h := utils.HashString(project.FBHandle)
			followers := int64(1500 + (h % 70000))
			reach := int64(float64(followers) * (1.6 + float64(h%8)/10.0))
			metrics = append(metrics, models.SocialMetric{
				Platform: "Facebook", Followers: followers, Reach: reach,
				Engagement: 3.6, EngagementCount: int64(float64(reach) * 0.036),
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
	platform := "Instagram"
	handle := project.IGHandle
	if handle == "" {
		platform = "Facebook"
		handle = project.FBHandle
	}
	hash := utils.HashString(handle)
	followers := int64(45 + int(hash%120))
	if platform == "Facebook" {
		followers = int64(1500 + int(hash%70000))
	}
	reach := int64(float64(followers) * (1.2 + float64(hash%5)/10.0))
	if platform == "Facebook" {
		reach = int64(float64(followers) * (1.6 + float64(hash%8)/10.0))
	}
	engagementRate := 5.2 + float64(hash%30)/10.0
	if platform == "Facebook" {
		engagementRate = 3.6
	}

	sm := &models.SocialMetric{
		ProjectID:       project.ID,
		Platform:        platform,
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

// RelatedProfiles returns a curated list of similar profiles
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

	var related []RelatedProfilesData

	if project.IGHandle == "capturedbyaaryan" || project.IGHandle == "@capturedbyaaryan" {
		related = []RelatedProfilesData{
			{
				Handle:      "samkolder",
				Name:        "Sam Kolder",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.82787-19/702365691_18591275881014122_5434092266847647636_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=107&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=yid3STAv17MQ7kNvwFwqVFd&_nc_oc=Adr58OspGJisXbyG7KyRq2XUXxuXZ77x19kgwA6euNQrdSBNp2yAsCf9dhidla64T2EolpQ_8bLUvlru1izjTomI&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=wFKF7kkpCEAI8jFVRKyOBA&_nc_ss=7360f&oh=00_Af6tXsaauNM5ltXJS6JFCAv0ZKx6tYtcmFgxgay4xbSEjg&oe=6A1EFA17",
				Followers:   2000000,
				Following:   1012,
				Posts:       200,
				MatchScore:  99,
				Tags:        []string{"Vlog", "Travel"},
				Description: "Filmmaker / Director / @koldercreative",
			},
			{
				Handle:      "benn_tk",
				Name:        "Benn TK",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/311161940_785157549432920_570970524973205207_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=104&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=SOxBkbchRfgQ7kNvwFMLSC-&_nc_oc=AdqWtygprjQ99h1ZEii-_oEuRnhBT1IoCokU2QhE3x6QZoB8XwTnSFFBu30VJIUEAlQjqqidWij9YZjsJ72Yr5n8&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af6T_olN8eJy_PuBPasnGjwio0B5cIf_PHpbIDhqWARGzA&oe=6A1F18BA",
				Followers:   208000,
				Following:   1415,
				Posts:       596,
				MatchScore:  99,
				Tags:        []string{"Filmmaking", "Travel"},
				Description: "I make cinematic travel videos to inspire people.",
			},
			{
				Handle:      "mattih",
				Name:        "Matti Haapoja",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.82787-19/532229670_18514722631059431_7719408287347732919_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=101&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=BbfHtaxL7H4Q7kNvwEb-3wZ&_nc_oc=Adpj6PV_hDpj_y_QyiDVJgUarEAqdBgLEmGnxEjgfqHvs27pJVnNwO3TNTc66n-7Ga0DrXHXQFG4gy1Stm4mdehb&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=Q3hwvsvwa5SDbHA_NBixAA&_nc_ss=7360f&oh=00_Af5mt3TP-pLKY-MNHK5yf8s3A5R1aVODh6qkh-_wvfojVg&oe=6A1F25AB",
				Followers:   465000,
				Following:   1239,
				Posts:       898,
				MatchScore:  98,
				Tags:        []string{"Cinematic", "Vlog"},
				Description: "Director of Photography & YouTuber. Cinematic vibes.",
			},
			{
				Handle:      "petermckinnon",
				Name:        "Peter McKinnon",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.82787-19/649075979_18567194755033569_3002926412572366380_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=1&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=q_M3qaa7J7UQ7kNvwEuEr6N&_nc_oc=AdoPKaLksWxr-hDuXmyonjNJu9RC3WIxmKEyppmno1qriwojhoO-FgMgEJXT-wicZkqb1jkbGb73AnTpX7QWIDJi&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=PuboO49CabNEux5aO_5_IQ&_nc_ss=7360f&oh=00_Af6P9qy-dYCZQYG1bUvDrxfDzyMBJmOkF8dNeWauPvQn1A&oe=6A1F19B4",
				Followers:   3000000,
				Following:   400,
				Posts:       1063,
				MatchScore:  98,
				Tags:        []string{"Travel", "Cinematic"},
				Description: "Photographer & Filmmaker. Coffee enthusiast.",
			},
			{
				Handle:      "chrisburkard",
				Name:        "Chris Burkard",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.82787-19/590413759_18553170712013183_8058678629183092323_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=110&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=khopN0lUWIMQ7kNvwEwRQiZ&_nc_oc=Adqw0g61W-E3RpfHqHZLZ5luPVsBZ-_G-_txm5BeNo5dIo0WGSQ_QOjQYUNbsga_zTHwC4qkt4yzh1f0GsRS-tMW&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=yi6eOMdVLC5AG8LIo9FUEA&_nc_ss=7360f&oh=00_Af55hytzvBnc9B7jUUKgQreQvf4mCLCf7J2u4gEzD7Xh5g&oe=6A1F1CB8",
				Followers:   4000000,
				Following:   1953,
				Posts:       5150,
				MatchScore:  97,
				Tags:        []string{"Travel", "Filmmaking"},
				Description: "Explorer, Photographer, Creative Director.",
			},
			{
				Handle:      "jralli23",
				Name:        "JR Alli",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/72787276_411261643142266_7752995124145029120_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=100&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=9P-X3Wrty7IQ7kNvwHAefH9&_nc_oc=AdpLjxXGrUVxvzd_ho3ysIxCFQJX4i-PUYOV-6WJhRMzIu0xj0uXsPKVh5izjmxBVisg7ct9YdDcCCzTQvLyrtco&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af6QeOHnY25tKlqAvQQ7yCYpxz_GiyrYTMM34TC_lwD0Dw&oe=6A1EFF58",
				Followers:   79000,
				Following:   800,
				Posts:       407,
				MatchScore:  97,
				Tags:        []string{"Travel", "Cinematic"},
				Description: "Visual storytelling. Creating feelings through film.",
			},
			{
				Handle:      "beautifuldestinations",
				Name:        "Beautiful Destinations",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.75761-19/495088815_18518699668032189_7331358023860286024_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=1&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDAwLkMzIn0%3D&_nc_ohc=mVCJDoEBvS8Q7kNvwFfPFwc&_nc_oc=Adqy7ZxkzkqK7BJwsnSvqLOtAa4zi7up7vt3GI7jKQrdc9TrkqLsk09fl0RJ8xF6hiuK-D4ZEI5oB81FYkp5A2Tw&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=EI9pjzL23mhuxE_Ge8cKww&_nc_ss=7360f&oh=00_Af6EkOhbzqYu1nwHSCBd48lHf_urehl0fcBBZBsXcrXlQQ&oe=6A1F1A92",
				Followers:   24000000,
				Following:   45,
				Posts:       14000,
				MatchScore:  96,
				Tags:        []string{"Vlog", "Cinematic"},
				Description: "The largest travel community on social media.",
			},
			{
				Handle:      "travelandleisure",
				Name:        "Travel And Leisure",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/85012422_2916580841733708_573326190681522176_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=1&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy45NjAuQzMifQ%3D%3D&_nc_ohc=mkKr5y57h8YQ7kNvwGHTeti&_nc_oc=AdrowvlFKe3jmCrDnOmbuYXYp5qvZk4m_JzWD1N0iZo7wZ66YHrLqbubakacRCFXRJaFdtMm3T8ZMSZs_JwUoMJg&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af5PKPtlvGIv6FyjCgjPjaf8cKtTi4xxn-Dnu0HuQNtorw&oe=6A1F256D",
				Followers:   6000000,
				Following:   939,
				Posts:       14000,
				MatchScore:  96,
				Tags:        []string{"Vlog", "Cinematic"},
				Description: "Your ultimate guide to traveling the world.",
			},
			{
				Handle:      "earth",
				Name:        "Earth",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.82787-19/658704278_18575096848013203_288900753127483512_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=1&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy43NDkuQzMifQ%3D%3D&_nc_ohc=QR045ni6MawQ7kNvwHi-vX_&_nc_oc=Ado75lxXitA_KBd-lySq2Qy9e2ZfCkqovd50A8tuRBgUc0ZEJ_o1pOT05RJ6Zfmx7m-YgX1GaGfb8Kd1wGu9sF6z&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=4d-KDTTtPY8iyquyGaSi4Q&_nc_ss=7360f&oh=00_Af5DQWnkc03_GeutcxFAG13QDSnmmnEYSAAZk4BxHbosXA&oe=6A1F2CDE",
				Followers:   7000000,
				Following:   616,
				Posts:       3077,
				MatchScore:  95,
				Tags:        []string{"Vlog", "Cinematic"},
				Description: "Capturing the beautiful moments of our planet.",
			},
			{
				Handle:      "rorykramer",
				Name:        "Rory Kramer",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.82787-19/626213173_18554806372018844_7644555667306734993_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=104&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=pE0ROcj7DesQ7kNvwFoxMvA&_nc_oc=Adoi-H18EtBNL7GxwV2rjpXcodm-lW7y0Hy_wWNUNU6C_WtYFraoe38inbaNOO--NULF703TxwAagrh6E0uHcMr8&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=Xw9mTEapouFwa2E-s7dFjg&_nc_ss=7360f&oh=00_Af6Wi51oxUFE3oIlTG2rnpyQDByfDQHYDikUA75cVBz-gw&oe=6A1F0907",
				Followers:   678000,
				Following:   2193,
				Posts:       2491,
				MatchScore:  95,
				Tags:        []string{"Travel", "Cinematic"},
				Description: "Professional Life Liver. Videographer.",
			},
			{
				Handle:      "daniel.schiffer",
				Name:        "Daniel Schiffer",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.82787-19/581413342_18548861518020564_3371825090161279396_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=103&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=mOb2m0ZeLCcQ7kNvwFo16HT&_nc_oc=Adr5uHeyfFipSVNUeFQUjCDFPP6U4Ad67-0lxkp1tT5Agtu8LX1meAfB1ZCQUHITio4ljFt8ah6H9n2V8Rz2HzvW&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=F-lhD7IS0NIsi8IkeKeWjw&_nc_ss=7360f&oh=00_Af4lTy0ajflZXKOjtdhoAa5aQOD7T5qFVD0nS9F-8SgVaQ&oe=6A1F0E81",
				Followers:   518000,
				Following:   382,
				Posts:       271,
				MatchScore:  94,
				Tags:        []string{"Travel", "Filmmaking"},
				Description: "Commercial Director & B-Roll Master.",
			},
			{
				Handle:      "lizziepeirce",
				Name:        "Lizzie Peirce",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/315861934_1758369237880567_8097713063193833148_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=101&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy41NDEuQzMifQ%3D%3D&_nc_ohc=pyXOrud-3mkQ7kNvwExA6gp&_nc_oc=AdrWTwkpj4wlwLoiSvbFtazoWm1Rmhm3-gTSEnp9X_sQ3n9HQeahEWCDn24U1veLaFXSe_DA9MBk2sxKLY6Rn3Lx&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af5UAZwxUTu8zbMoSc8abVqrGRD1YpyTnFtN0GSm-jhHSw&oe=6A1F2B39",
				Followers:   120000,
				Following:   1458,
				Posts:       531,
				MatchScore:  94,
				Tags:        []string{"Filmmaking", "Travel"},
				Description: "Video producer & photographer from Toronto.",
			},
			{
				Handle:      "colinandsamir",
				Name:        "Colin And Samir",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/67410611_2174653209498818_3747814464221609984_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=108&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy41MDAuQzMifQ%3D%3D&_nc_ohc=azlay8jrmccQ7kNvwH-ZCOv&_nc_oc=AdphO5zFrpNuFX_d_xrHbTsEtNhg56txWdgwjwgCKUfnuElLDxB_LZSBnQrss_EkzDC4v5PlCmYY2JEv9QUkMWII&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af7GxCuH2QREAAv9HIXApn7pUqDI-LCgYHw8FvNAAlnzuQ&oe=6A1F18BB",
				Followers:   299000,
				Following:   1052,
				Posts:       427,
				MatchScore:  93,
				Tags:        []string{"Filmmaking", "Cinematic"},
				Description: "Decoding the creator economy.",
			},
			{
				Handle:      "taylorcutfilms",
				Name:        "Taylorcutfilms",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/184407919_784522695600312_8532203637821013516_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=110&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=VOHu71rYAesQ7kNvwHE_6RW&_nc_oc=Adq32A4E2LdtVhLpmYhddgQwzd22DkIGq3o9fcQUu_ApQ9E4RF_nFVTN731VJGlv-dy1INZHX5uBU_n0r2Y93vsP&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af7oYCv4UXIJ9-A_JnAa3YIsoFBXNVARnS_n5zEv5LGiWQ&oe=6A1F2685",
				Followers:   522000,
				Following:   362,
				Posts:       2539,
				MatchScore:  93,
				Tags:        []string{"Vlog", "Filmmaking"},
				Description: "Jordan Taylor Wright. Director & Creative.",
			},
			{
				Handle:      "lostleblanc",
				Name:        "Lost LeBlanc",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/39331420_1791267910991546_5379036506302185472_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=106&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=BFcadq9HYS8Q7kNvwFTc0_U&_nc_oc=AdphfeYhynnZtblkEEOWdDd3rcLItL3opHYmcRRjM22SGiGqkqsfZAZleH_oP9Gjx2sWZjoO8owtbPK9A_acTEkf&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af5stJam4eXRb2ECYXdeK8837aW92UDUDc4JBBtqcZFGSQ&oe=6A1F1977",
				Followers:   712000,
				Following:   1457,
				Posts:       1224,
				MatchScore:  92,
				Tags:        []string{"Cinematic", "Vlog"},
				Description: "Quit my job to travel the world. Founder of Lost Creator Academy.",
			},
			{
				Handle:      "jessedriftwood",
				Name:        "Jesse Driftwood",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/317574675_677152763989512_6367427859637979977_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=101&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy42MjYuQzMifQ%3D%3D&_nc_ohc=SSXfRbhBwncQ7kNvwEs34Wa&_nc_oc=Adr6uDCrgqK3lqd8eTWJQtRQCSFKc_3iR1skTKxoyp6a8mvnr_bV1mKrKSrd3A3eY5Rz-YyibtIfv5LNL72Fl5jX&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af6uBALwEm-KKt7meWGQe1mezB0Pc9ZDzltN72-XJr6SXw&oe=6A1EFC09",
				Followers:   143000,
				Following:   1231,
				Posts:       410,
				MatchScore:  91,
				Tags:        []string{"Filmmaking", "Vlog"},
				Description: "Travel Filmmaker documenting the world.",
			},
			{
				Handle:      "caseyneistat",
				Name:        "Casey Neistat",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/397515707_286651127675800_3761168994942547018_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=1&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDAwLkMzIn0%3D&_nc_ohc=2xw5mDtDgRsQ7kNvwHb4lXw&_nc_oc=Ado7rJENtfSP2REle5Pk2CsCjYCoM-mo6VPzan7sxS82Fh_hheprbqFBuP9H-R0l576VGhz4tcmI06miUEb9w0yA&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af6F8mt5vQrCBBDKLr2TWHCHSEF_Ij5fZPpSlv1L_diYBg&oe=6A1F22C3",
				Followers:   3000000,
				Following:   4443,
				Posts:       2029,
				MatchScore:  91,
				Tags:        []string{"Travel", "Filmmaking"},
				Description: "Filmmaker, YouTube pioneer, New Yorker.",
			},
			{
				Handle:      "jordi.koalitic",
				Name:        "Jordi Koalitic",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.75761-19/499659877_18504531427044665_2404796851135103569_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=1&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=Ra0kbGtyrfkQ7kNvwEcne3R&_nc_oc=Adp3yVh3M6AZNwGITbUo_i5bM5wa2YtltNlkrHpCz16Yvv2RTdykQpVkG-8VsvLY9MqW7H30h72mKf9psHp3HJCr&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=1IfkQS0k7cPtDNQi2RTyPQ&_nc_ss=7360f&oh=00_Af5_bcWf201_jYyEcXwdlhVEOQDrDKobz8kLChx4TIZxUQ&oe=6A1F1F3A",
				Followers:   7000000,
				Following:   1020,
				Posts:       1051,
				MatchScore:  90,
				Tags:        []string{"Travel", "Cinematic"},
				Description: "Creative Photography & Videography tricks.",
			},
			{
				Handle:      "mangostreetlab",
				Name:        "Mango Street",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/117295896_209597980502058_1686383511424285453_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=104&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=0OhdHg8cfEIQ7kNvwEewNLw&_nc_oc=AdoRQ3i_-SZ2doOehCq9lvyQoiVx8OteFTS6JH-FAub967iLVOHh3XLOOuOrdzz6aUiZtJkuuFNOU0Zz2NYsIrhV&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af6pkfLssnlWCR80V4xQH_LAo8IbsReABqUlq8M1tKVIxA&oe=6A1F0C91",
				Followers:   102000,
				Following:   7,
				Posts:       219,
				MatchScore:  90,
				Tags:        []string{"Travel", "Cinematic"},
				Description: "Photography and filmmaking tutorials that don't waste your time.",
			},
			{
				Handle:      "genenagata",
				Name:        "Potato Jet",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/106371931_317686256180527_1286707043061471260_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=105&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=1USyW0f2K3oQ7kNvwHZFvz8&_nc_oc=AdpF865sLsENLZ-9X9sqXMrUQW7c9-TEfbOIBqYYuSPtkXW17bDdKok9rVMUn1UDxm6Os-sFZklhLe1mKyQb1scb&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af5sUT8qfTE-P07YvzWCVWbyhtNFo5L9llAg1E4t4N82Uw&oe=6A1F1517",
				Followers:   100000,
				Following:   1674,
				Posts:       354,
				MatchScore:  89,
				Tags:        []string{"Cinematic", "Filmmaking"},
				Description: "Filmmaker testing the best camera gear in the world.",
			},
			{
				Handle:      "brandon_l_li",
				Name:        "Brandon Li",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/11282142_838415872900358_71171122_a.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=103&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xNTAuQzMifQ%3D%3D&_nc_ohc=1usRO6oWXqEQ7kNvwHbvjIJ&_nc_oc=Adrhx45k04kkeapk3gqn0XVm6PQ0UiWEWcl5XgNiLV7-OtsORjIFa9UW0s-jV7YnDCkuysYJ6KamgiuO3pLWti4d&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af70kQHQjlHxiqV3LTRofX72GdG5aYcv3e_ynAZgLsmJpQ&oe=6A1F2064",
				Followers:   212000,
				Following:   614,
				Posts:       687,
				MatchScore:  89,
				Tags:        []string{"Filmmaking", "Travel"},
				Description: "Nomadic filmmaker. Unscripted cinematic travel.",
			},
			{
				Handle:      "andyto",
				Name:        "Andy To",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.82787-19/554214136_18533511808047381_1138508400482628236_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=108&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=A4rYdqOYh24Q7kNvwFrXrrf&_nc_oc=AdpYji6VCLepctARXN2j9WnWpcqD68jK2pwwzM9SCPqTZZnUTHc07036Z7y5oF9q2vGT_JDPBF90CZoir07BJsef&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=_QMYOzndDl_HWrU3tZXKsg&_nc_ss=7360f&oh=00_Af5Qt7WiJlZdJj7HliCdzbbaZMiCnjPesgUiTrhypyTVfg&oe=6A1F1533",
				Followers:   289000,
				Following:   1792,
				Posts:       1809,
				MatchScore:  88,
				Tags:        []string{"Travel", "Vlog"},
				Description: "Filmmaker based in NYC. Apple partner.",
			},
			{
				Handle:      "jacob",
				Name:        "Jacob Riglin",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.75761-19/514012763_18503608285018977_7682039993554214869_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=1&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy4xMDgwLkMzIn0%3D&_nc_ohc=TkaQ2tj3IIQQ7kNvwGfR0Jd&_nc_oc=AdqEq1bnSdSonVqBW8R9V--r2rGfyIAwG5Xoq676uDmN8-dyt9NQ2-iCVyBwc0AccyLZ5j6ParlGwHkPGHWilTtw&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_gid=TD0j7QZQsxXb60QAtXoRqw&_nc_ss=7360f&oh=00_Af6aJ1tjAjG66xs1jjoBE_pk_rJP-QzjTgn3iZHWquN5zw&oe=6A1F2BBC",
				Followers:   1000000,
				Following:   2299,
				Posts:       2786,
				MatchScore:  88,
				Tags:        []string{"Cinematic", "Filmmaking"},
				Description: "Photography. Travel. NFTs. Beautifully captured.",
			},
			{
				Handle:      "emmett_sparling",
				Name:        "Emmett Sparling",
				Avatar:      "https://scontent.cdninstagram.com/v/t51.2885-19/136834190_149651426928400_5540861194396446085_n.jpg?stp=dst-jpg_s100x100_tt6&_nc_cat=105&ccb=7-5&_nc_sid=bf7eb4&efg=eyJ2ZW5jb2RlX3RhZyI6InByb2ZpbGVfcGljLnd3dy41NTkuQzMifQ%3D%3D&_nc_ohc=0AcI26GkfeYQ7kNvwFx4664&_nc_oc=Adow4jPmGfwPrBf7XFuL6anx-jXbgx31Dt7Oy3MO6xlu2g1KBJJ4lIzcPBEDCnDst__JhWkDmGoD7fGicDEHIJ9g&_nc_zt=24&_nc_ht=scontent.cdninstagram.com&_nc_ss=7360f&oh=00_Af6Zy-3UGOflVApp9oybQ0f_35KZotLjuHOTI58h5hDFTg&oe=6A1F2CAF",
				Followers:   1000000,
				Following:   1286,
				Posts:       2243,
				MatchScore:  87,
				Tags:        []string{"Filmmaking", "Travel"},
				Description: "Adventure & Travel Photographer.",
			},
		}
	} else if project.IGHandle == "aryanvishwakarma_01" || project.IGHandle == "@aryanvishwakarma_01" {
		related = []RelatedProfilesData{
			{
				Handle:      "peter_mckinnon",
				Name:        "Peter McKinnon",
				Avatar:      "https://ui-avatars.com/api/?name=Peter+McKinnon&background=0f172a&color=fff&size=256",
				Followers:   3200000,
				MatchScore:  98,
				Tags:        []string{"Photography", "Travel", "Vlog"},
				Description: "High match due to similar audience overlap in Street & Travel photography.",
			},
			{
				Handle:      "chrisburkard",
				Name:        "Chris Burkard",
				Avatar:      "https://ui-avatars.com/api/?name=Chris+Burkard&background=0284c7&color=fff&size=256",
				Followers:   3900000,
				MatchScore:  94,
				Tags:        []string{"Adventure", "Landscapes"},
				Description: "Engages a similar outdoor/travel demographic.",
			},
			{
				Handle:      "gregwilliamsphotography",
				Name:        "Greg Williams",
				Avatar:      "https://ui-avatars.com/api/?name=Greg+Williams&background=1e293b&color=fff&size=256",
				Followers:   1200000,
				MatchScore:  89,
				Tags:        []string{"Portraits", "Street"},
				Description: "Strong overlap in portrait style photography.",
			},
			{
				Handle:      "alexcruz",
				Name:        "Alex Cruz",
				Avatar:      "https://ui-avatars.com/api/?name=Alex+Cruz&background=b91c1c&color=fff&size=256",
				Followers:   245000,
				MatchScore:  85,
				Tags:        []string{"Street", "Urban"},
				Description: "Emerging street photographer with rapidly growing engagement.",
			},
		}
	} else {
		related = []RelatedProfilesData{
			{
				Handle:      "creator_hub",
				Name:        "Creator Hub",
				Avatar:      "https://ui-avatars.com/api/?name=Creator+Hub&background=0f172a&color=fff&size=256",
				Followers:   850000,
				MatchScore:  92,
				Tags:        []string{"Content Creation", "Tips"},
				Description: "Shares similar educational content style.",
			},
			{
				Handle:      "daily_growth",
				Name:        "Daily Growth",
				Avatar:      "https://ui-avatars.com/api/?name=Daily+Growth&background=059669&color=fff&size=256",
				Followers:   120000,
				MatchScore:  88,
				Tags:        []string{"Business", "Motivation"},
				Description: "Audience demographics closely align with yours.",
			},
			{
				Handle:      "marketing_pro",
				Name:        "Marketing Pro",
				Avatar:      "https://ui-avatars.com/api/?name=Marketing+Pro&background=ea580c&color=fff&size=256",
				Followers:   45000,
				MatchScore:  81,
				Tags:        []string{"Marketing", "SEO"},
				Description: "Competes for similar high-value industry keywords.",
			},
		}
	}
	// -------------------------------------------------------------
	// DYNAMIC REAL DATA FETCHING VIA META API BUSINESS DISCOVERY
	// -------------------------------------------------------------
	
	// Collect handles to fetch
	var handlesToFetch []string
	for _, r := range related {
		handlesToFetch = append(handlesToFetch, r.Handle)
	}

	accessToken := h.resolveMetaAccessToken(userID)
	if accessToken != "" && project.IGHandle != "" {
		accounts, err := h.metaService.GetIGUserAccounts(accessToken)
		if err == nil && len(accounts) > 0 {
			targetID, targetToken := resolveTargetAccount(accounts, project.IGHandle)
			if targetID != "" {
				if targetToken == "" {
					targetToken = accessToken
				}
				
				// Fetch real data for these handles
				liveData, err := h.metaService.FetchCompetitorData(targetID, targetToken, handlesToFetch)
				if err == nil && len(liveData) > 0 {
					// Overwrite the hardcoded data with REAL numbers!
					for i, r := range related {
						if data, ok := liveData[r.Handle]; ok {
							if followers, ok := data["followers"].(int); ok {
								related[i].Followers = int64(followers)
							}
							if posts, ok := data["posts"].(int); ok {
								related[i].Posts = int64(posts)
							}
							if name, ok := data["name"].(string); ok && name != "" {
								related[i].Name = name
							}
							if avatar, ok := data["profile_picture_url"].(string); ok && avatar != "" {
								related[i].Avatar = avatar
							}
							if bio, ok := data["biography"].(string); ok && bio != "" {
								related[i].Description = bio
							}
						}
					}
				}
			}
		}
	}

	utils.Success(c, related, nil)
}
