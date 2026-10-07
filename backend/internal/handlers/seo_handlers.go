package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
		"url":           result.URL,
		"score":         result.Score,
		"checks":        result.Checks,
		"load_time_ms":  result.LoadTimeMs,
		"crawled_at":    result.CrawledAt,
		"robots":        result.Robots,
		"https":         result.HTTPS,
		"cwv":           result.CWV,
		"sitemap":       result.Sitemap,
		"indexability":  result.Indexability,
		"international": result.International,
		"markup":        result.Markup,
		"issues_found":  issuesSummary,
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
		utils.Success(c, gin.H{"keywords": keywordPublicList(cached), "ideas": ideasForKeywordSource(cached, "cache"), "source": "cache", "seed": seed}, nil)
		return
	}

	keywords, source := h.resolveKeywords(pid, seed, project.URL, userID)

	if len(keywords) > 0 {
		h.seoRepo.UpsertKeywords(keywords)
	}

	utils.Success(c, gin.H{
		"keywords": keywordPublicList(keywords),
		"ideas":    ideasForKeywordSource(keywords, source),
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
		utils.Success(c, gin.H{"keywords": keywordPublicList(cached), "ideas": ideasForKeywordSource(cached, "cache"), "source": "cache", "seed": seed}, nil)
		return
	}

	keywords, source := h.resolveKeywords(req.ProjectID, seed, project.URL, userID)

	if len(keywords) > 0 {
		h.seoRepo.UpsertKeywords(keywords)
	}

	utils.Success(c, gin.H{
		"keywords": keywordPublicList(keywords),
		"ideas":    ideasForKeywordSource(keywords, source),
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
		"score":         project.HealthScore,
		"health":        project.Health,
		"issues_count":  len(issues),
		"url":           project.URL,
		"last_updated":  project.UpdatedAt.Format(time.RFC3339),
		"robots":        robotsStatusFromIssues(issues, project.HealthScore, project.UpdatedAt),
		"https":         httpsStatusFromIssues(issues, project.HealthScore, project.UpdatedAt),
		"cwv":           cwvStatusFromIssues(issues, project.HealthScore, project.UpdatedAt),
		"lighthouse":    lighthouseStatusFromIssues(issues, project.HealthScore, project.UpdatedAt),
		"sitemap":       sitemapStatusFromIssues(issues, project.HealthScore, project.UpdatedAt),
		"indexability":  indexabilityStatusFromIssues(issues, project.HealthScore, project.UpdatedAt),
		"international": internationalStatusFromIssues(issues, project.HealthScore, project.UpdatedAt),
		"markup":        markupStatusFromIssues(issues, project.HealthScore, project.UpdatedAt),
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
			"url":           result.URL,
			"score":         result.Score,
			"checks":        result.Checks,
			"crawled_at":    result.CrawledAt,
			"robots":        result.Robots,
			"https":         result.HTTPS,
			"cwv":           result.CWV,
			"sitemap":       result.Sitemap,
			"indexability":  result.Indexability,
			"international": result.International,
			"markup":        result.Markup,
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

func robotsStatusFromIssues(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryStatusFromIssues(issues, "robots", healthScore, updated)
}

func httpsStatusFromIssues(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryStatusFromIssues(issues, "https", healthScore, updated)
}

func cwvStatusFromIssues(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	info := categoryStatusFromIssues(issues, "cwv", healthScore, updated)
	switch info["label"] {
	case "Pass":
		info["label"] = "Good"
	case "Warning":
		info["label"] = "Needs work"
	case "Fail":
		info["label"] = "Poor"
	}
	return info
}

func lighthouseStatusFromIssues(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryStatusFromIssues(issues, "lighthouse", healthScore, updated)
}

func sitemapStatusFromIssues(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryStatusFromIssues(issues, "sitemap", healthScore, updated)
}

func indexabilityStatusFromIssues(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryStatusFromIssues(issues, "indexability", healthScore, updated)
}

func internationalStatusFromIssues(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryStatusFromIssues(issues, "international", healthScore, updated)
}

func markupStatusFromIssues(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryStatusFromIssues(issues, "markup", healthScore, updated)
}

func keywordPublicList(kws []models.KeywordResult) []gin.H {
	out := make([]gin.H, 0, len(kws))
	for _, k := range kws {
		out = append(out, gin.H{
			"id":          k.ID,
			"keyword":     k.Keyword,
			"seed":        k.Seed,
			"volume":      k.Volume,
			"impressions": k.Volume,
			"clicks":      k.Clicks,
			"kd":          k.KD,
			"position":    k.Position,
			"intent":      services.GuessSearchIntent(k.Keyword),
		})
	}
	return out
}

func ideasForKeywordSource(kws []models.KeywordResult, source string) []gin.H {
	switch source {
	case "gsc":
		return gscContentIdeas(kws)
	case "cache":
		for _, k := range kws {
			if k.Position > 0 || k.Clicks > 0 {
				return gscContentIdeas(kws)
			}
		}
		return []gin.H{}
	default:
		return []gin.H{}
	}
}

func gscContentIdeas(kws []models.KeywordResult) []gin.H {
	clicksKnown := false
	for _, k := range kws {
		if k.Clicks > 0 {
			clicksKnown = true
			break
		}
	}
	out := make([]gin.H, 0)
	for _, k := range kws {
		if len(out) >= 8 {
			break
		}
		intent := services.GuessSearchIntent(k.Keyword)
		if clicksKnown && k.Clicks == 0 && k.Volume >= 20 {
			out = append(out, gin.H{
				"type":        "zero_click",
				"keyword":     k.Keyword,
				"impressions": k.Volume,
				"intent":      intent,
				"reason":      "GSC impressions with no clicks in this window. Not search volume and not a competitor gap.",
			})
			continue
		}
		if intent == "informational" {
			out = append(out, gin.H{
				"type":        "question",
				"keyword":     k.Keyword,
				"impressions": k.Volume,
				"intent":      intent,
				"reason":      "Question-style query from your GSC. Directional idea only.",
			})
		}
	}
	return out
}

func categoryStatusFromIssues(issues []models.SEOIssue, category string, healthScore int, updated time.Time) gin.H {
	if healthScore == 0 && len(issues) == 0 {
		return gin.H{
			"status":     "unknown",
			"label":      "—",
			"checked_at": "",
		}
	}
	fail, warn := false, false
	for _, issue := range issues {
		if !strings.EqualFold(issue.Category, category) {
			continue
		}
		if issue.Severity == models.SeverityHigh {
			fail = true
		} else {
			warn = true
		}
	}
	status, label := "pass", "Pass"
	if fail {
		status, label = "fail", "Fail"
	} else if warn {
		status, label = "warning", "Warning"
	}
	checked := ""
	if !updated.IsZero() {
		checked = updated.UTC().Format("2006-01-02 15:04 UTC")
	}
	return gin.H{
		"status":     status,
		"label":      label,
		"checked_at": checked,
	}
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

	pages, countries, devices, appearance := []models.GSCBreakdown{}, []models.GSCBreakdown{}, []models.GSCBreakdown{}, []models.GSCBreakdown{}
	windowStart, windowEnd := "", ""
	if isGSCConnected {
		pages, _ = h.seoRepo.FindBreakdowns(pid, models.GSCDimPage)
		countries, _ = h.seoRepo.FindBreakdowns(pid, models.GSCDimCountry)
		devices, _ = h.seoRepo.FindBreakdowns(pid, models.GSCDimDevice)
		appearance, _ = h.seoRepo.FindBreakdowns(pid, models.GSCDimSearchAppearance)
		if len(pages) > 0 {
			windowStart, windowEnd = pages[0].StartDate, pages[0].EndDate
		} else if len(devices) > 0 {
			windowStart, windowEnd = devices[0].StartDate, devices[0].EndDate
		} else if len(countries) > 0 {
			windowStart, windowEnd = countries[0].StartDate, countries[0].EndDate
		}
	}

	utils.Success(c, gin.H{
		"keywords":      keywordPublicList(ranked),
		"ideas":         gscContentIdeas(ranked),
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
		"pages":             gscBreakdownJSON(pages, displayPageKey),
		"countries":         gscBreakdownJSON(countries, displayCountryKey),
		"devices":           gscBreakdownJSON(devices, displayDeviceKey),
		"search_appearance": gscBreakdownJSON(appearance, displayAppearanceKey),
		"window": gin.H{
			"start": windowStart,
			"end":   windowEnd,
		},
		"last_updated": project.UpdatedAt.Format(time.RFC3339),
	}, nil)
}

const vendorSnapshotTTL = 24 * time.Hour

func (h *SEOHandler) dataForSEOReady() bool {
	return h.dataForSEOSvc != nil && h.dataForSEOSvc.Configured()
}

func emptyBacklinksPayload(updated time.Time, message string) gin.H {
	return gin.H{
		"total_backlinks":   0,
		"referring_domains": 0,
		"do_follow":         0,
		"no_follow":         0,
		"domain_authority":  0,
		"rank":              0,
		"gsc_connected":     false,
		"is_estimated":      false,
		"available":         false,
		"configured":        false,
		"source":            "",
		"top_domains":       []gin.H{},
		"last_updated":      updated.Format(time.RFC3339),
		"message":           message,
	}
}

// Backlinks returns DataForSEO Backlinks index data when a cached domain
// lookup exists. It does not invent counts. Moz Domain Authority is not provided.
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

	if !h.dataForSEOReady() {
		utils.Success(c, emptyBacklinksPayload(project.UpdatedAt, "Backlink index is not connected."), nil)
		return
	}

	host := services.NormalizeDomain(project.URL)
	result, ok := h.cachedDomainExplorer(host)
	if !ok || !result.BacklinksOK {
		payload := emptyBacklinksPayload(project.UpdatedAt, "Look up this domain in Site Explorer to load DataForSEO backlinks.")
		payload["configured"] = true
		payload["domain"] = host
		utils.Success(c, payload, nil)
		return
	}

	utils.Success(c, gin.H{
		"total_backlinks":   result.Backlinks.Backlinks,
		"referring_domains": result.Backlinks.ReferringDomains,
		"do_follow":         0,
		"no_follow":         0,
		"domain_authority":  0,
		"rank":              result.Backlinks.Rank,
		"spam_score":        result.Backlinks.SpamScore,
		"gsc_connected":     false,
		"is_estimated":      false,
		"available":         true,
		"configured":        true,
		"cached":            true,
		"source":            "dataforseo",
		"domain":            result.Domain,
		"top_domains":       []gin.H{},
		"last_updated":      result.FetchedAt.Format(time.RFC3339),
		"message":           "From DataForSEO Backlinks index. Rank is DataForSEO Rank, not Moz Domain Authority.",
	}, nil)
}

// DomainExplorer looks up any domain via DataForSEO Labs + Backlinks.
// Does not run on page load unless refresh=1 or cache is cold after an explicit lookup.
func (h *SEOHandler) DomainExplorer(c *gin.Context) {
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

	domain := c.Query("domain")
	if domain == "" {
		domain = project.URL
	}
	refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")
	wantFetch := refresh || c.Query("fetch") == "1" || strings.EqualFold(c.Query("fetch"), "true")

	if !h.dataForSEOReady() {
		utils.Success(c, gin.H{
			"available":      false,
			"configured":     false,
			"source":         "dataforseo",
			"domain":         services.NormalizeDomain(domain),
			"message":        "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD.",
			"organic_ok":     false,
			"backlinks_ok":   false,
			"competitors_ok": false,
			"keywords_ok":    false,
			"competitors":    []gin.H{},
			"keywords":       []gin.H{},
		}, nil)
		return
	}

	if !wantFetch {
		if cached, ok := h.cachedDomainExplorer(domain); ok {
			utils.Success(c, domainExplorerJSON(cached, true), nil)
			return
		}
		utils.Success(c, gin.H{
			"available":      false,
			"configured":     true,
			"domain":         services.NormalizeDomain(domain),
			"source":         "dataforseo",
			"message":        "Click Look up to query DataForSEO. This uses vendor credits; results cache 24 hours.",
			"organic_ok":     false,
			"backlinks_ok":   false,
			"competitors_ok": false,
			"keywords_ok":    false,
			"competitors":    []gin.H{},
			"keywords":       []gin.H{},
		}, nil)
		return
	}

	result, cached, err := h.loadDomainExplorer(c.Request.Context(), domain, refresh)
	if err != nil {
		if errors.Is(err, services.ErrDataForSEONotConfigured) {
			utils.Success(c, gin.H{
				"available":  false,
				"configured": false,
				"message":    "DataForSEO is not connected.",
			}, nil)
			return
		}
		utils.Success(c, gin.H{
			"available":      false,
			"configured":     true,
			"domain":         services.NormalizeDomain(domain),
			"source":         "dataforseo",
			"message":        err.Error(),
			"organic_ok":     false,
			"backlinks_ok":   false,
			"competitors_ok": false,
			"keywords_ok":    false,
			"competitors":    []gin.H{},
			"keywords":       []gin.H{},
		}, nil)
		return
	}
	result.Cached = cached
	utils.Success(c, domainExplorerJSON(result, cached), nil)
}

func domainExplorerJSON(result services.DomainExplorerResult, cached bool) gin.H {
	return gin.H{
		"available":         result.HasAnyData(),
		"configured":        true,
		"source":            result.Source,
		"domain":            result.Domain,
		"location_code":     result.LocationCode,
		"location_name":     result.LocationName,
		"language_code":     result.LanguageCode,
		"organic":           result.Organic,
		"organic_ok":        result.OrganicOK,
		"organic_error":     result.OrganicError,
		"backlinks":         result.Backlinks,
		"backlinks_ok":      result.BacklinksOK,
		"backlinks_error":   result.BacklinksError,
		"competitors":       result.Competitors,
		"competitors_ok":    result.CompetitorsOK,
		"competitors_error": result.CompetitorsError,
		"keywords":          result.Keywords,
		"keywords_ok":       result.KeywordsOK,
		"keywords_error":    result.KeywordsError,
		"cached":            cached,
		"fetched_at":        result.FetchedAt.Format(time.RFC3339),
		"message":           "DataForSEO Labs estimates for Google United States. Not Google Search Console and not a sitewide crawl.",
	}
}

func (h *SEOHandler) cachedDomainExplorer(domain string) (services.DomainExplorerResult, bool) {
	host := services.NormalizeDomain(domain)
	if host == "" || h.seoRepo == nil {
		return services.DomainExplorerResult{}, false
	}
	snap, err := h.seoRepo.FindVendorDomainSnapshot(host, services.DataForSEOLocation, services.DataForSEOLanguage)
	if err != nil || snap == nil || snap.Payload == "" {
		return services.DomainExplorerResult{}, false
	}
	var cached services.DomainExplorerResult
	if json.Unmarshal([]byte(snap.Payload), &cached) != nil || !cached.HasAnyData() {
		return services.DomainExplorerResult{}, false
	}
	cached.Cached = true
	return cached, true
}

func (h *SEOHandler) loadDomainExplorer(ctx context.Context, domain string, refresh bool) (services.DomainExplorerResult, bool, error) {
	host := services.NormalizeDomain(domain)
	if host == "" {
		return services.DomainExplorerResult{}, false, fmt.Errorf("invalid domain")
	}
	if h.seoRepo != nil && !refresh {
		snap, err := h.seoRepo.FindVendorDomainSnapshot(host, services.DataForSEOLocation, services.DataForSEOLanguage)
		if err == nil && snap != nil && time.Since(snap.FetchedAt) < vendorSnapshotTTL && snap.Payload != "" {
			var cached services.DomainExplorerResult
			if json.Unmarshal([]byte(snap.Payload), &cached) == nil && cached.HasAnyData() {
				cached.Cached = true
				return cached, true, nil
			}
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[seo] vendor snapshot read: %v", err)
		}
	}

	result, err := h.dataForSEOSvc.DomainExplorer(ctx, host)
	if err != nil {
		return result, false, err
	}
	if h.seoRepo != nil && result.HasAnyData() {
		raw, _ := json.Marshal(result)
		_ = h.seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
			Domain:       result.Domain,
			LocationCode: result.LocationCode,
			LanguageCode: result.LanguageCode,
			Source:       result.Source,
			Payload:      string(raw),
			FetchedAt:    result.FetchedAt,
		})
	}
	return result, false, nil
}

// AIVisibility returns DataForSEO LLM Mentions for a domain (Google AI
// Overviews + ChatGPT). Cache-only unless fetch=1 or refresh=1. Does not
// write vendor counts into dashboard metrics.
func (h *SEOHandler) AIVisibility(c *gin.Context) {
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

	domain := c.Query("domain")
	if domain == "" {
		domain = project.URL
	}
	refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")
	wantFetch := refresh || c.Query("fetch") == "1" || strings.EqualFold(c.Query("fetch"), "true")

	if !h.dataForSEOReady() {
		utils.Success(c, gin.H{
			"available":    false,
			"configured":   false,
			"source":       "dataforseo",
			"domain":       services.NormalizeDomain(domain),
			"message":      "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD.",
			"mentions_ok":  false,
			"citations_ok": false,
			"mentions":     services.LLMMetrics{Sources: []services.LLMSourceDomain{}},
			"citations":    services.LLMMetrics{Sources: []services.LLMSourceDomain{}},
		}, nil)
		return
	}

	if !wantFetch {
		if cached, ok := h.cachedAIVisibility(domain); ok {
			utils.Success(c, aiVisibilityJSON(cached, true), nil)
			return
		}
		utils.Success(c, gin.H{
			"available":    false,
			"configured":   true,
			"domain":       services.NormalizeDomain(domain),
			"source":       "dataforseo",
			"message":      "Click Look up to query DataForSEO LLM Mentions. This uses vendor credits; results cache 24 hours.",
			"mentions_ok":  false,
			"citations_ok": false,
			"mentions":     services.LLMMetrics{Sources: []services.LLMSourceDomain{}},
			"citations":    services.LLMMetrics{Sources: []services.LLMSourceDomain{}},
		}, nil)
		return
	}

	result, cached, err := h.loadAIVisibility(c.Request.Context(), domain, refresh)
	if err != nil {
		if errors.Is(err, services.ErrDataForSEONotConfigured) {
			utils.Success(c, gin.H{
				"available":  false,
				"configured": false,
				"message":    "DataForSEO is not connected.",
			}, nil)
			return
		}
		utils.Success(c, gin.H{
			"available":    false,
			"configured":   true,
			"domain":       services.NormalizeDomain(domain),
			"source":       "dataforseo",
			"message":      err.Error(),
			"mentions_ok":  false,
			"citations_ok": false,
			"mentions":     services.LLMMetrics{Sources: []services.LLMSourceDomain{}},
			"citations":    services.LLMMetrics{Sources: []services.LLMSourceDomain{}},
		}, nil)
		return
	}
	result.Cached = cached
	utils.Success(c, aiVisibilityJSON(result, cached), nil)
}

func aiVisibilityJSON(result services.AIVisibilityResult, cached bool) gin.H {
	return gin.H{
		"available":       result.HasAnyData(),
		"configured":      true,
		"source":          result.Source,
		"domain":          result.Domain,
		"location_code":   result.LocationCode,
		"location_name":   result.LocationName,
		"language_code":   result.LanguageCode,
		"mentions":        result.Mentions,
		"mentions_ok":     result.MentionsOK,
		"mentions_error":  result.MentionsError,
		"citations":       result.Citations,
		"citations_ok":    result.CitationsOK,
		"citations_error": result.CitationsError,
		"cached":          cached,
		"fetched_at":      result.FetchedAt.Format(time.RFC3339),
		"message":         "DataForSEO LLM Mentions for Google AI Overviews and ChatGPT (United States, English). Citations are sources the models actually used. Not Search Console.",
	}
}

func (h *SEOHandler) cachedAIVisibility(domain string) (services.AIVisibilityResult, bool) {
	host := services.NormalizeDomain(domain)
	if host == "" || h.seoRepo == nil {
		return services.AIVisibilityResult{}, false
	}
	snap, err := h.seoRepo.FindVendorAIVisibilitySnapshot(host, services.DataForSEOLocation, services.DataForSEOLanguage)
	if err != nil || snap == nil || snap.Payload == "" {
		return services.AIVisibilityResult{}, false
	}
	var cached services.AIVisibilityResult
	if json.Unmarshal([]byte(snap.Payload), &cached) != nil || !cached.HasAnyData() {
		return services.AIVisibilityResult{}, false
	}
	cached.Cached = true
	return cached, true
}

func (h *SEOHandler) loadAIVisibility(ctx context.Context, domain string, refresh bool) (services.AIVisibilityResult, bool, error) {
	host := services.NormalizeDomain(domain)
	if host == "" {
		return services.AIVisibilityResult{}, false, fmt.Errorf("invalid domain")
	}
	if h.seoRepo != nil && !refresh {
		snap, err := h.seoRepo.FindVendorAIVisibilitySnapshot(host, services.DataForSEOLocation, services.DataForSEOLanguage)
		if err == nil && snap != nil && time.Since(snap.FetchedAt) < vendorSnapshotTTL && snap.Payload != "" {
			var cached services.AIVisibilityResult
			if json.Unmarshal([]byte(snap.Payload), &cached) == nil && cached.HasAnyData() {
				cached.Cached = true
				return cached, true, nil
			}
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[seo] ai visibility snapshot read: %v", err)
		}
	}

	result, err := h.dataForSEOSvc.AIVisibility(ctx, host)
	if err != nil {
		return result, false, err
	}
	if h.seoRepo != nil && result.HasAnyData() {
		raw, _ := json.Marshal(result)
		_ = h.seoRepo.SaveVendorAIVisibilitySnapshot(&models.VendorAIVisibilitySnapshot{
			Domain:       result.Domain,
			LocationCode: result.LocationCode,
			LanguageCode: result.LanguageCode,
			Source:       result.Source,
			Payload:      string(raw),
			FetchedAt:    result.FetchedAt,
		})
	}
	return result, false, nil
}
