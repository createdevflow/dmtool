package handlers

import (
	"log"
	"strconv"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProjectHandler struct {
	projectRepo    repository.ProjectRepository
	metricRepo     repository.MetricRepository
	oauthRepo      repository.OAuthRepository
	seoRepo        repository.SEORepository
	dataForSEOSvc  services.DataForSEOService
	rapidAPISvc    services.RapidAPIService
	crawlerService services.SEOCrawlerService
	db             *gorm.DB
	encKey         []byte
}

func NewProjectHandler(
	projectRepo repository.ProjectRepository,
	database *gorm.DB,
	encKey []byte,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	seoRepo repository.SEORepository,
	dataForSEOSvc services.DataForSEOService,
	rapidAPISvc services.RapidAPIService,
	crawler services.SEOCrawlerService,
) *ProjectHandler {
	return &ProjectHandler{
		projectRepo:    projectRepo,
		metricRepo:     metricRepo,
		oauthRepo:      oauthRepo,
		seoRepo:        seoRepo,
		dataForSEOSvc:  dataForSEOSvc,
		rapidAPISvc:    rapidAPISvc,
		crawlerService: crawler,
		db:             database,
		encKey:         encKey,
	}
}

func (h *ProjectHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	projects, err := h.projectRepo.FindAllByUser(userID)
	if err != nil {
		utils.InternalError(c, "Failed to fetch projects")
		return
	}

	utils.Success(c, projects, nil)
}

type ProjectRequest struct {
	Name            string `json:"name"`

	URL             string `json:"url"`
	Goal            string `json:"goal" binding:"oneof=seo social both"`
	InstagramHandle string `json:"ig_handle"`
	TwitterHandle   string `json:"twitter_handle"`
	LinkedinHandle  string `json:"linkedin_handle"`
	FacebookHandle  string `json:"fb_handle"`
}

func (h *ProjectHandler) Create(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req ProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	project := &models.Project{
		UserID:          userID,
		Name:            req.Name,
		URL:             req.URL,
		Goal:            req.Goal,
		Status:          "active",
		Health:          "scanning",
		HealthScore:     0,
		IGHandle:        req.InstagramHandle,
		TwitterHandle:   req.TwitterHandle,
		LinkedinHandle:  req.LinkedinHandle,
		FBHandle:        req.FacebookHandle,
	}

	if project.Name == "" {
		project.Name = utils.DeriveNameFromURL(req.URL)
	}


	if err := h.projectRepo.Create(project); err != nil {
		utils.InternalError(c, "Failed to create project")
		return
	}

	// Trigger seeder for new project (Real-looking data for demo/initial state)
	go utils.SeedProject(h.db, project.ID)

	utils.Success(c, project, nil)
}

func (h *ProjectHandler) Update(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid project ID", "INVALID_ID")
		return
	}

	var req ProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	project, err := h.projectRepo.FindByIDAndUser(uint(id), userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	project.Name = req.Name
	project.URL = req.URL
	project.Goal = req.Goal
	project.IGHandle = req.InstagramHandle
	project.TwitterHandle = req.TwitterHandle
	project.LinkedinHandle = req.LinkedinHandle
	project.FBHandle = req.FacebookHandle

	if err := h.projectRepo.Update(project); err != nil {
		utils.InternalError(c, "Failed to update project")
		return
	}

	utils.Success(c, project, nil)
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid project ID", "INVALID_ID")
		return
	}

	if err := h.projectRepo.Delete(uint(id), userID); err != nil {
		utils.InternalError(c, "Failed to delete project")
		return
	}

	utils.NoContent(c)
}

// Onboard handles the specialized onboarding flow
func (h *ProjectHandler) Onboard(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req ProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	project := &models.Project{
		UserID:      userID,
		Name:        req.Name, // If name is empty, we could derive from URL
		URL:         req.URL,
		Goal:        req.Goal,
		Status:      "active",
		Health:      "analyzing",
		HealthScore: 75, // Starting score for demo
		IGHandle:    req.InstagramHandle,
		TwitterHandle:   req.TwitterHandle,
		LinkedinHandle:  req.LinkedinHandle,
		FBHandle:        req.FacebookHandle,
	}

	// Logic to derive name
	if project.Name == "" {
		if req.URL != "" {
			project.Name = utils.DeriveNameFromURL(req.URL)
		} else if req.InstagramHandle != "" {
			project.Name = "@" + req.InstagramHandle
		} else if req.TwitterHandle != "" {
			project.Name = "@" + req.TwitterHandle
		} else if req.FacebookHandle != "" {
			project.Name = "@" + req.FacebookHandle
		} else if req.LinkedinHandle != "" {
			project.Name = "@" + req.LinkedinHandle
		} else {
			project.Name = "New Project"
		}
	}

	if err := h.projectRepo.Create(project); err != nil {
		utils.InternalError(c, "Failed to start onboarding")
		return
	}

	// Trigger seeder for new project
	go utils.SeedProject(h.db, project.ID)

	// Auto-sync: If user already has OAuth credentials, trigger immediate data sync
	go h.autoSyncNewProject(project, userID)

	utils.Success(c, gin.H{
		"message": "Onboarding started",
		"project": project,
	}, nil)
}

// autoSyncNewProject triggers centralized data provider sync.
func (h *ProjectHandler) autoSyncNewProject(project *models.Project, userID uint) {
	log.Printf("[project] Auto-sync triggered for new project %d (%s)", project.ID, project.Name)

	// 1. Traffic Sync (DataForSEO)
	if project.URL != "" && h.dataForSEOSvc != nil {
		metrics, err := h.dataForSEOSvc.FetchEstimatedTraffic(project.URL)
		if err == nil {
			for _, m := range metrics {
				m.ProjectID = project.ID
				h.metricRepo.UpsertMetric(&m)
			}
			log.Printf("[project] Auto-synced %d Traffic records for project %d", len(metrics), project.ID)
		}
	}

	// 2. Social Sync (RapidAPI)
	if project.IGHandle != "" && h.rapidAPISvc != nil {
		sm, err := h.rapidAPISvc.FetchInstagramProfile(project.IGHandle)
		if err == nil && sm != nil && sm.Followers > 0 {
			sm.ProjectID = project.ID
			h.metricRepo.CreateSocialMetric(sm)
			log.Printf("[project] Auto-synced Social metrics for project %d: %d followers", project.ID, sm.Followers)
		}
	}

	// 4. Always run SEO crawl in background
	if project.URL != "" && h.crawlerService != nil {
		crawlResult, err := h.crawlerService.Crawl(project.URL)
		if err == nil {
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
			log.Printf("[project] Auto-crawled SEO for project %d: score=%d", project.ID, crawlResult.Score)
		}
	}
}
