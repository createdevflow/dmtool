package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type SEOHandler struct {
	projectRepo    repository.ProjectRepository
	seoRepo        repository.SEORepository
	oauthRepo      repository.OAuthRepository
	dataForSEOSvc  services.DataForSEOService
	crawlerService services.SEOCrawlerService
	keywordService services.KeywordService
	encKey         []byte
}

func NewSEOHandler(
	projectRepo repository.ProjectRepository,
	seoRepo repository.SEORepository,
	oauthRepo repository.OAuthRepository,
	dataForSEOSvc services.DataForSEOService,
	crawler services.SEOCrawlerService,
	keywords services.KeywordService,
	encKey []byte,
) *SEOHandler {
	return &SEOHandler{
		projectRepo:    projectRepo,
		seoRepo:        seoRepo,
		oauthRepo:      oauthRepo,
		dataForSEOSvc:  dataForSEOSvc,
		crawlerService: crawler,
		keywordService: keywords,
		encKey:         encKey,
	}
}

// AuditRun performs a real SEO crawl and saves findings to the DB.
func (h *SEOHandler) AuditRun(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		ProjectID uint   `json:"project_id" binding:"required"`
		URL       string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "project_id is required", "VALIDATION_ERROR")
		return
	}

	project, err := h.projectRepo.FindByIDAndUser(req.ProjectID, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	targetURL := req.URL
	if targetURL == "" {
		targetURL = project.URL
	}

	log.Printf("[seo] Running audit for project %d: %s", project.ID, targetURL)

	result, err := h.crawlerService.Crawl(targetURL)
	if err != nil {
		utils.InternalError(c, "Crawler failed: "+err.Error())
		return
	}

	issues := issuesFromChecks(project.ID, targetURL, result.Checks)
	if err := h.seoRepo.ReplaceOpenIssues(project.ID, issues); err != nil {
		log.Printf("[seo] failed to replace issues for project %d: %v", project.ID, err)
	}

	// Update project health score
	project.HealthScore = result.Score
	if result.Score >= 75 {
		project.Health = "healthy"
	} else if result.Score >= 50 {
		project.Health = "issues"
	} else {
		project.Health = "critical"
	}
	h.projectRepo.Update(project)

	log.Printf("[seo] Audit complete for project %d: score=%d, %d checks, %dms",
		project.ID, result.Score, len(result.Checks), result.LoadTimeMs)

	issuesSummary := countIssues(result.Checks)

	utils.Success(c, gin.H{
		"url":          result.URL,
		"score":        result.Score,
		"checks":       result.Checks,
		"load_time_ms": result.LoadTimeMs,
		"crawled_at":   result.CrawledAt,
		"issues_found": issuesSummary,
	}, nil)
}

// Issues returns all open SEO issues, optionally filtered by severity.
func (h *SEOHandler) Issues(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	severity := c.Query("severity")
	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)

	issues, err := h.seoRepo.FindOpenIssues(uint(projectID), severity)
	if err != nil {
		utils.InternalError(c, "Failed to fetch SEO issues")
		return
	}

	utils.Success(c, issues, nil)
}

// Keywords returns keyword suggestions using GSC (if connected) or Google autocomplete.
func (h *SEOHandler) Keywords(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	seed := c.Query("seed")

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

	if seed == "" {
		seed = extractDomainKeyword(project.URL)
	}

	// Check cache
	cached, fresh, _ := h.seoRepo.FindKeywords(pid, seed)
	if fresh && len(cached) > 0 {
		utils.Success(c, gin.H{"keywords": cached, "source": "cache", "seed": seed}, nil)
		return
	}

	keywords, source := h.resolveKeywords(pid, seed, project.URL, userID)

	if len(keywords) > 0 {
		h.seoRepo.UpsertKeywords(keywords)
	}

	utils.Success(c, gin.H{
		"keywords": keywords,
		"source":   source,
		"seed":     seed,
		"count":    len(keywords),
	}, nil)
}

// KeywordsPost handles POST /seo/keywords with body {project_id, seed}
func (h *SEOHandler) KeywordsPost(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		ProjectID uint   `json:"project_id" binding:"required"`
		Seed      string `json:"seed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "project_id is required", "VALIDATION_ERROR")
		return
	}

	project, err := h.projectRepo.FindByIDAndUser(req.ProjectID, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	seed := req.Seed
	if seed == "" {
		seed = extractDomainKeyword(project.URL)
	}

	cached, fresh, _ := h.seoRepo.FindKeywords(req.ProjectID, seed)
	if fresh && len(cached) > 0 {
		utils.Success(c, gin.H{"keywords": cached, "source": "cache", "seed": seed}, nil)
		return
	}

	keywords, source := h.resolveKeywords(req.ProjectID, seed, project.URL, userID)

	if len(keywords) > 0 {
		h.seoRepo.UpsertKeywords(keywords)
	}

	utils.Success(c, gin.H{
		"keywords": keywords,
		"source":   source,
		"seed":     seed,
		"count":    len(keywords),
	}, nil)
}

// resolveKeywords tries GSC first, then falls back to autocomplete.
func (h *SEOHandler) resolveKeywords(projectID uint, seed, siteURL string, userID uint) ([]models.KeywordResult, string) {
	// Try GSC if integration connected (DEPRECATED - use DataForSEO or Autocomplete)
	// We skip OAuth now and rely on our data providers.

	// Fall back to Google autocomplete
	autoKeywords, err := h.keywordService.FetchAutocompleteKeywords(seed)
	if err == nil {
		for i := range autoKeywords {
			autoKeywords[i].ProjectID = projectID
		}
		return autoKeywords, "autocomplete"
	}

	return nil, "none"
}

// Report returns a structured audit summary for a project.
func (h *SEOHandler) Report(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)

	project, _ := h.projectRepo.FindByIDAndUser(pid, userID)
	if project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	issues, _ := h.seoRepo.FindOpenIssues(pid, "")
	keywords, _, _ := h.seoRepo.FindKeywords(pid, "")

	highCount, medCount, lowCount := 0, 0, 0
	for _, iss := range issues {
		switch iss.Severity {
		case "high":
			highCount++
		case "medium":
			medCount++
		default:
			lowCount++
		}
	}

	utils.Success(c, gin.H{
		"project_name":   project.Name,
		"url":            project.URL,
		"score":          project.HealthScore,
		"health":         project.Health,
		"issues_count":   len(issues),
		"issues_high":    highCount,
		"issues_medium":  medCount,
		"issues_low":     lowCount,
		"keywords_count": len(keywords),
		"last_updated":   project.UpdatedAt.Format(time.RFC3339),
	}, nil)
}

// ResolveIssue marks a specific SEO issue as resolved.
func (h *SEOHandler) ResolveIssue(c *gin.Context) {
	issueIDStr := c.Param("id")
	projectIDStr := c.Query("project_id")

	issueID, _ := strconv.ParseUint(issueIDStr, 10, 32)
	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)

	if err := h.seoRepo.ResolveIssue(uint(issueID), uint(projectID)); err != nil {
		utils.InternalError(c, "Failed to resolve issue")
		return
	}

	utils.Success(c, gin.H{"resolved": true}, nil)
}

// GetAuditStatus returns the current audit status for a project.
func (h *SEOHandler) GetAuditStatus(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)

	project, _ := h.projectRepo.FindByIDAndUser(pid, userID)
	if project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	issues, _ := h.seoRepo.FindOpenIssues(pid, "")

	utils.Success(c, gin.H{
		"score":        project.HealthScore,
		"health":       project.Health,
		"issues_count": len(issues),
		"url":          project.URL,
		"last_updated": project.UpdatedAt.Format(time.RFC3339),
	}, nil)
}

// PublicAudit allows unauthenticated SEO audits (for landing page demo).
func (h *SEOHandler) PublicAudit(c *gin.Context) {
	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	result, err := h.crawlerService.Crawl(targetURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "crawl failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url":    result.URL,
			"score":  result.Score,
			"checks": result.Checks,
		},
	})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func countIssues(checks []services.AuditCheck) gin.H {
	high, med, low := 0, 0, 0
	for _, ch := range checks {
		if ch.Status == services.CheckFail || ch.Status == services.CheckWarning {
			switch ch.Severity {
			case "high":
				high++
			case "medium":
				med++
			default:
				low++
			}
		}
	}
	return gin.H{"high": high, "medium": med, "low": low}
}

func issuesFromChecks(projectID uint, targetURL string, checks []services.AuditCheck) []models.SEOIssue {
	var issues []models.SEOIssue
	for _, check := range checks {
		if check.Status == services.CheckFail || check.Status == services.CheckWarning {
			issues = append(issues, models.SEOIssue{
				ProjectID: projectID,
				URL:       targetURL,
				Severity:  check.Severity,
				Category:  check.Category,
				Detail:    check.Label + ": " + check.Detail,
			})
		}
	}
	return issues
}

func extractDomainKeyword(rawURL string) string {
	s := rawURL
	for _, pfx := range []string{"https://", "http://", "www."} {
		if len(s) > len(pfx) && s[:len(pfx)] == pfx {
			s = s[len(pfx):]
		}
	}
	for i, ch := range s {
		if ch == '/' || ch == '.' {
			return s[:i]
		}
	}
	return s
}

// RankTracking returns keyword position data from stored keywords (GSC-based when connected).
func (h *SEOHandler) RankTracking(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)

	project, _ := h.projectRepo.FindByIDAndUser(pid, userID)
	if project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	isGSCConnected := false
	googleCred, err := h.oauthRepo.FindByUserAndProvider(userID, "google")
	if err == nil && googleCred != nil {
		isGSCConnected = true
	}

	// Rank Tracking is GSC-only. Without a Google connection, leftover
	// autocomplete/seed rows must not appear as rankings.
	ranked := make([]models.KeywordResult, 0)
	if isGSCConnected {
		stored, _, _ := h.seoRepo.FindKeywords(pid, "")
		for _, kw := range stored {
			if kw.Position > 0 {
				ranked = append(ranked, kw)
			}
		}
	}

	var visibilityScore float64
	if len(ranked) > 0 {
		for _, kw := range ranked {
			score := 100.0 - kw.Position
			if score < 0 {
				score = 0
			}
			visibilityScore += score
		}
		visibilityScore = visibilityScore / float64(len(ranked))
	}

	top3, top10, top30, beyond := 0, 0, 0, 0
	for _, kw := range ranked {
		switch {
		case kw.Position <= 3:
			top3++
		case kw.Position <= 10:
			top10++
		case kw.Position <= 30:
			top30++
		default:
			beyond++
		}
	}

	utils.Success(c, gin.H{
		"keywords":      ranked,
		"total":         len(ranked),
		"visibility":    visibilityScore,
		"gsc_connected": isGSCConnected,
		"health_score":  project.HealthScore,
		"buckets": gin.H{
			"top3":   top3,
			"top10":  top10,
			"top30":  top30,
			"beyond": beyond,
		},
		"last_updated": project.UpdatedAt.Format(time.RFC3339),
	}, nil)
}

// Backlinks does not have a link index. Returns zeros rather than estimated counts.
func (h *SEOHandler) Backlinks(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)

	project, _ := h.projectRepo.FindByIDAndUser(pid, userID)
	if project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	utils.Success(c, gin.H{
		"total_backlinks":   0,
		"referring_domains": 0,
		"do_follow":         0,
		"no_follow":         0,
		"domain_authority":  0,
		"gsc_connected":     false,
		"is_estimated":      false,
		"available":         false,
		"top_domains":       []gin.H{},
		"last_updated":      project.UpdatedAt.Format(time.RFC3339),
		"message":           "Backlink index is not connected.",
	}, nil)
}
