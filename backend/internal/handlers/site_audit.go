package handlers

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/services"
	"backend/internal/siteaudit"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type saSaveRequest struct {
	ProjectID     uint   `json:"project_id" binding:"required"`
	Domain        string `json:"domain"`
	Scope         string `json:"scope"`
	PageLimit     int    `json:"page_limit"`
	CrawlSource   string `json:"crawl_source"`
	Schedule      string `json:"schedule"`
	UserAgent     string `json:"user_agent"`
	JSRendering   bool   `json:"js_rendering"`
	SubfolderPath string `json:"subfolder_path"`
}

// ListSiteAudit returns the Semrush-style campaign list (sortable, paginated).
func (h *SEOHandler) ListSiteAudit(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	limits := h.saLimits(userID, ptIsAdmin(c))
	rows, err := h.seoRepo.ListSiteAuditCampaignsByUser(userID)
	if err != nil {
		utils.InternalError(c, "Could not list campaigns")
		return
	}
	projects, _ := h.projectRepo.FindAllByUser(userID)
	names := map[uint]string{}
	for _, p := range projects {
		names[p.ID] = p.Name
	}

	list := make([]siteaudit.ListRow, 0, len(rows))
	for _, row := range rows {
		snap := decodeSASnapshot(row.SnapshotJSON)
		var prev *siteaudit.Snapshot
		if row.PreviousJSON != "" {
			p := decodeSASnapshot(row.PreviousJSON)
			prev = &p
		}
		name := names[row.ProjectID]
		if name == "" {
			name = row.Domain
		}
		list = append(list, siteaudit.ListRowFrom(row.ProjectID, name, row.Domain, row.PageLimit, snap, prev))
	}

	sortKey := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort", "last_update")))
	dir := strings.ToLower(strings.TrimSpace(c.DefaultQuery("dir", "desc")))
	sortSAList(list, sortKey, dir != "asc")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	total := len(list)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	utils.Success(c, gin.H{
		"limits":     limits,
		"campaigns":  list[start:end],
		"honesty":    siteaudit.DefaultHonesty(limits, 0),
		"js_message": saJSMessage(limits),
	}, utils.NewMeta(utils.PaginationParams{Page: page, PageSize: pageSize, Offset: start}, int64(total)))
}

// GetSiteAudit returns one campaign and its Semrush-style reports.
func (h *SEOHandler) GetSiteAudit(c *gin.Context) {
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
	limits := h.saLimits(userID, ptIsAdmin(c))
	campaign, _ := h.seoRepo.FindSiteAuditCampaign(pid)
	var view gin.H
	snap := siteaudit.Snapshot{Thematic: siteaudit.Thematic{}, Pages: []siteaudit.PageRow{}, Issues: []siteaudit.IssueRow{}}
	var prev *siteaudit.Snapshot
	if campaign != nil {
		view = saCampaignJSON(campaign)
		snap = decodeSASnapshot(campaign.SnapshotJSON)
		if campaign.PreviousJSON != "" {
			p := decodeSASnapshot(campaign.PreviousJSON)
			prev = &p
		}
	}
	utils.Success(c, gin.H{
		"limits":     limits,
		"campaign":   view,
		"reports":    siteaudit.BuildReports(limits, snap, prev),
		"js_message": saJSMessage(limits),
		"honesty":    siteaudit.DefaultHonesty(limits, snap.PagesCrawled),
	}, nil)
}

// SaveSiteAudit creates or updates a Site Audit campaign within plan caps.
func (h *SEOHandler) SaveSiteAudit(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var req saSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "project_id is required", "VALIDATION_ERROR")
		return
	}
	project, err := h.projectRepo.FindByIDAndUser(req.ProjectID, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}
	limits := h.saLimits(userID, ptIsAdmin(c))
	existing, findErr := h.seoRepo.FindSiteAuditCampaign(req.ProjectID)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		utils.InternalError(c, "Could not load campaign")
		return
	}
	if existing == nil {
		n, _ := h.seoRepo.CountSiteAuditWebsites(userID)
		if n >= limits.Websites {
			utils.BadRequest(c, "Website cap reached for this plan ("+strconv.Itoa(limits.Websites)+"). Upgrade to audit more sites.", "PLAN_LIMIT")
			return
		}
	}

	domain := services.NormalizeDomain(req.Domain)
	if domain == "" {
		domain = services.NormalizeDomain(project.URL)
	}
	if domain == "" {
		utils.BadRequest(c, "A domain is required", "VALIDATION_ERROR")
		return
	}

	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope != "subdomain" && scope != "subfolder" {
		scope = "domain"
	}
	source := strings.ToLower(strings.TrimSpace(req.CrawlSource))
	if source != "sitemap" {
		source = "website"
	}
	schedule := strings.ToLower(strings.TrimSpace(req.Schedule))
	if schedule != "daily" && schedule != "weekly" {
		schedule = "manual"
	}
	if !limits.Schedule {
		schedule = "manual"
	}
	ua := strings.ToLower(strings.TrimSpace(req.UserAgent))
	switch ua {
	case "mobile", "googlebot", "openai-search":
	default:
		ua = "desktop"
	}
	js := req.JSRendering && limits.JSRendering
	pageLimit := siteaudit.ClampPageLimit(req.PageLimit, limits)

	row := &models.SiteAuditCampaign{
		ProjectID:     req.ProjectID,
		Domain:        domain,
		Scope:         scope,
		PageLimit:     pageLimit,
		CrawlSource:   source,
		Schedule:      schedule,
		UserAgent:     ua,
		JSRendering:   js,
		SubfolderPath: strings.TrimSpace(req.SubfolderPath),
	}
	if existing != nil {
		row.ID = existing.ID
		row.CreatedAt = existing.CreatedAt
		row.SnapshotJSON = existing.SnapshotJSON
		row.PreviousJSON = existing.PreviousJSON
		row.SnapshotAt = existing.SnapshotAt
		row.SnapshotError = existing.SnapshotError
		row.PagesCrawled = existing.PagesCrawled
		row.LastStoppedReason = existing.LastStoppedReason
	}
	if err := h.seoRepo.SaveSiteAuditCampaign(row); err != nil {
		utils.InternalError(c, "Could not save campaign")
		return
	}
	saved, _ := h.seoRepo.FindSiteAuditCampaign(req.ProjectID)
	utils.Success(c, gin.H{
		"limits":   limits,
		"campaign": saCampaignJSON(saved),
		"honesty":  siteaudit.DefaultHonesty(limits, 0),
	}, nil)
}

// RunSiteAudit crawls HTML pages. Cache-only unless fetch=1 or refresh=1.
func (h *SEOHandler) RunSiteAudit(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		var body struct {
			ProjectID uint `json:"project_id"`
		}
		_ = c.ShouldBindJSON(&body)
		if body.ProjectID == 0 {
			utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
			return
		}
		projectIDStr = strconv.FormatUint(uint64(body.ProjectID), 10)
	}
	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)
	project, err := h.projectRepo.FindByIDAndUser(pid, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}
	campaign, err := h.seoRepo.FindSiteAuditCampaign(pid)
	if err != nil || campaign == nil {
		utils.BadRequest(c, "Save a Site Audit campaign first", "NO_CAMPAIGN")
		return
	}

	refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")
	wantFetch := refresh || c.Query("fetch") == "1" || strings.EqualFold(c.Query("fetch"), "true")
	if !wantFetch {
		h.GetSiteAudit(c)
		return
	}
	if h.crawlerService == nil {
		campaign.SnapshotError = "Crawler is not configured."
		_ = h.seoRepo.SaveSiteAuditCampaign(campaign)
		utils.InternalError(c, campaign.SnapshotError)
		return
	}

	seed := campaign.Domain
	if campaign.Scope == "subfolder" && campaign.SubfolderPath != "" {
		seed = "https://" + campaign.Domain + campaign.SubfolderPath
	}
	limits := h.saLimits(userID, ptIsAdmin(c))
	res, crawlErr := h.crawlerService.CrawlSite(seed, services.SiteCrawlOptions{
		PageLimit:     campaign.PageLimit,
		RunCap:        siteaudit.RunPageCap,
		Source:        campaign.CrawlSource,
		Scope:         campaign.Scope,
		SubfolderPath: campaign.SubfolderPath,
		UserAgent:     campaign.UserAgent,
		Deadline:      time.Now().Add(22 * time.Second),
	})
	now := time.Now().UTC()
	if crawlErr != nil {
		campaign.SnapshotError = crawlErr.Error()
		campaign.SnapshotAt = &now
		_ = h.seoRepo.SaveSiteAuditCampaign(campaign)
		utils.Success(c, gin.H{
			"limits":   limits,
			"campaign": saCampaignJSON(campaign),
			"error":    campaign.SnapshotError,
			"honesty":  siteaudit.DefaultHonesty(limits, 0),
		}, nil)
		return
	}

	if campaign.SnapshotJSON != "" {
		campaign.PreviousJSON = campaign.SnapshotJSON
	}
	snap := siteaudit.FromCrawl(res, campaign.PageLimit, campaign.CrawlSource, limits.PagesPerAudit)
	campaign.SnapshotJSON = mustJSON(snap)
	campaign.SnapshotAt = snap.CrawledAt
	campaign.SnapshotError = ""
	campaign.PagesCrawled = snap.PagesCrawled
	campaign.LastStoppedReason = snap.StoppedReason
	if err := h.seoRepo.SaveSiteAuditCampaign(campaign); err != nil {
		utils.InternalError(c, "Could not save crawl")
		return
	}

	issues := issuesFromSiteAudit(project.ID, snap.Issues)
	if err := h.seoRepo.ReplaceOpenIssues(project.ID, issues); err != nil {
		// keep crawl even if issue replace fails
	}
	if snap.SiteHealth != nil {
		project.HealthScore = *snap.SiteHealth
		if *snap.SiteHealth >= 75 {
			project.Health = "healthy"
		} else if *snap.SiteHealth >= 50 {
			project.Health = "issues"
		} else {
			project.Health = "critical"
		}
		_ = h.projectRepo.Update(project)
	}

	var prev *siteaudit.Snapshot
	if campaign.PreviousJSON != "" {
		p := decodeSASnapshot(campaign.PreviousJSON)
		prev = &p
	}
	utils.Success(c, gin.H{
		"limits":     limits,
		"campaign":   saCampaignJSON(campaign),
		"reports":    siteaudit.BuildReports(limits, snap, prev),
		"js_message": saJSMessage(limits),
		"honesty":    snap.Honesty,
	}, nil)
}

func (h *SEOHandler) saLimits(userID uint, isAdmin bool) siteaudit.Limits {
	plan := models.PlanCodeFree
	if h.seoRepo != nil {
		if code, err := h.seoRepo.FindPlanCodeByUser(userID); err == nil && code != "" {
			plan = code
		}
	}
	return siteaudit.ForPlan(plan, isAdmin)
}

func saJSMessage(limits siteaudit.Limits) string {
	if limits.JSRendering {
		return "JS rendering can be stored on this plan. The crawler still reads HTML only — JavaScript is not executed."
	}
	return "JS rendering is included on Pro+ and above. This crawl reads HTML only."
}

func saCampaignJSON(c *models.SiteAuditCampaign) gin.H {
	if c == nil {
		return nil
	}
	out := gin.H{
		"id":                  c.ID,
		"project_id":          c.ProjectID,
		"domain":              c.Domain,
		"scope":               c.Scope,
		"page_limit":          c.PageLimit,
		"crawl_source":        c.CrawlSource,
		"schedule":            c.Schedule,
		"user_agent":          c.UserAgent,
		"js_rendering":        c.JSRendering,
		"subfolder_path":      c.SubfolderPath,
		"pages_crawled":       c.PagesCrawled,
		"last_stopped_reason": c.LastStoppedReason,
		"snapshot_error":      c.SnapshotError,
	}
	if c.SnapshotAt != nil {
		out["snapshot_at"] = c.SnapshotAt.UTC().Format(time.RFC3339)
	}
	return out
}

func decodeSASnapshot(raw string) siteaudit.Snapshot {
	out := siteaudit.Snapshot{
		Pages:  []siteaudit.PageRow{},
		Issues: []siteaudit.IssueRow{},
	}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	if out.Pages == nil {
		out.Pages = []siteaudit.PageRow{}
	}
	if out.Issues == nil {
		out.Issues = []siteaudit.IssueRow{}
	}
	return out
}

func issuesFromSiteAudit(projectID uint, issues []siteaudit.IssueRow) []models.SEOIssue {
	out := make([]models.SEOIssue, 0, len(issues))
	for _, iss := range issues {
		sev := models.SeverityLow
		switch iss.Severity {
		case "error":
			sev = models.SeverityHigh
		case "warning":
			sev = models.SeverityMed
		}
		out = append(out, models.SEOIssue{
			ProjectID: projectID,
			URL:       iss.URL,
			Severity:  sev,
			Category:  iss.Category,
			Detail:    iss.Label + ": " + iss.Detail,
		})
	}
	return out
}

func sortSAList(list []siteaudit.ListRow, key string, desc bool) {
	less := func(a, b siteaudit.ListRow) bool {
		cmp := 0
		switch key {
		case "project":
			cmp = strings.Compare(strings.ToLower(a.ProjectName+" "+a.Domain), strings.ToLower(b.ProjectName+" "+b.Domain))
		case "pages_crawled":
			cmp = cmpNilInt(a.PagesCrawled, b.PagesCrawled)
		case "site_health":
			cmp = cmpCell(a.SiteHealth, b.SiteHealth)
		case "errors":
			cmp = cmpCell(a.Errors, b.Errors)
		case "warnings":
			cmp = cmpCell(a.Warnings, b.Warnings)
		case "crawlability":
			cmp = cmpCell(a.Crawlability, b.Crawlability)
		case "https":
			cmp = cmpCell(a.HTTPS, b.HTTPS)
		case "international":
			cmp = cmpCell(a.International, b.International)
		case "performance":
			cmp = cmpCell(a.Performance, b.Performance)
		case "linking":
			cmp = cmpCell(a.Linking, b.Linking)
		case "markups":
			cmp = cmpCell(a.Markups, b.Markups)
		default:
			cmp = cmpTime(a.LastUpdate, b.LastUpdate)
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	}
	sort.SliceStable(list, func(i, j int) bool { return less(list[i], list[j]) })
}

func cmpNilInt(a, b *int) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	return *a - *b
}

func cmpCell(a, b siteaudit.ScoreCell) int {
	return cmpNilInt(a.Value, b.Value)
}

func cmpTime(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if a.Equal(*b) {
		return 0
	}
	if a.After(*b) {
		return 1
	}
	return -1
}
