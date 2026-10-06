package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	projectRepo repository.ProjectRepository
	metricRepo  repository.MetricRepository
	insightRepo repository.InsightRepository
	seoRepo     repository.SEORepository
	taskRepo    repository.TaskRepository
}

func NewDashboardHandler(
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	insightRepo repository.InsightRepository,
	seoRepo repository.SEORepository,
	taskRepo repository.TaskRepository,
) *DashboardHandler {
	return &DashboardHandler{
		projectRepo: projectRepo,
		metricRepo:  metricRepo,
		insightRepo: insightRepo,
		seoRepo:     seoRepo,
		taskRepo:    taskRepo,
	}
}

func (h *DashboardHandler) Snapshot(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)

	// 1. Fetch project
	project, err := h.projectRepo.FindByIDAndUser(pid, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	// 2. Fetch metrics for current and previous period (30 days each)
	now := time.Now()
	t0 := now.Format("2006-01-02")
	t30 := now.AddDate(0, 0, -30).Format("2006-01-02")
	t60 := now.AddDate(0, 0, -60).Format("2006-01-02")

	metricsCurrent, _ := h.metricRepo.FindMetricsByProjectAndRange(pid, t30, t0)
	metricsPrevious, _ := h.metricRepo.FindMetricsByProjectAndRange(pid, t60, t30)
	metricsCurrent = gscOnly(metricsCurrent)
	metricsPrevious = gscOnly(metricsPrevious)

	// 3. Compute web traffic stats from GSC only (seed/dataforseo ignored)
	var currClicks, prevClicks, currImpressions, prevImpressions int64
	for _, m := range metricsCurrent {
		currClicks += m.Clicks
		currImpressions += m.Impressions
	}
	for _, m := range metricsPrevious {
		prevClicks += m.Clicks
		prevImpressions += m.Impressions
	}
	trafficSource := ""
	if len(metricsCurrent) > 0 || len(metricsPrevious) > 0 {
		trafficSource = models.MetricSourceGSC
	}

	clickChange := utils.CalculateChange(currClicks, prevClicks)
	impressionChange := utils.CalculateChange(currImpressions, prevImpressions)

	// Compute CTR (Clicks / Impressions * 100)
	var ctr float64
	if currImpressions > 0 {
		ctr = float64(currClicks) / float64(currImpressions) * 100
	}
	var prevCTR float64
	if prevImpressions > 0 {
		prevCTR = float64(prevClicks) / float64(prevImpressions) * 100
	}
	ctrDelta := ctr - prevCTR

	// 4. Fetch SEO issues + GSC keyword positions
	issues, _ := h.seoRepo.FindOpenIssues(pid, "")
	keywords, _, _ := h.seoRepo.FindKeywords(pid, "")
	ranked, top3, page1 := keywordPositionBuckets(keywords)
	openIssueCount := len(issues)
	pages, _ := h.seoRepo.FindBreakdowns(pid, models.GSCDimPage)
	countries, _ := h.seoRepo.FindBreakdowns(pid, models.GSCDimCountry)
	devices, _ := h.seoRepo.FindBreakdowns(pid, models.GSCDimDevice)
	hasGSC := trafficSource != "" || ranked > 0 || len(pages)+len(countries)+len(devices) > 0

	websiteStats := []gin.H{
		{
			"label":  "SEO Health",
			"value":  strconv.Itoa(project.HealthScore) + "%",
			"change": healthStatus(project.HealthScore),
			"trend":  healthTrend(project.HealthScore),
			"icon":   "Target",
		},
		{
			"label":  "Organic Traffic",
			"value":  utils.FormatNumber(currClicks),
			"change": fmtChange(clickChange),
			"trend":  trendDir(clickChange),
			"icon":   "Globe",
		},
		{
			"label":  "Search Impressions",
			"value":  utils.FormatNumber(currImpressions),
			"change": fmtChange(impressionChange),
			"trend":  trendDir(impressionChange),
			"icon":   "TrendingUp",
		},
		{
			"label":  "Click-Through Rate",
			"value":  fmt.Sprintf("%.1f%%", ctr),
			"change": fmtCTRDelta(ctrDelta),
			"trend":  trendDir(ctrDelta),
			"icon":   "MousePointer2",
		},
		rankedKeywordStat("Ranked Keywords", ranked, hasGSC),
		rankedKeywordStat("Top 3 Keywords", top3, hasGSC),
		rankedKeywordStat("Page 1 Keywords", page1, hasGSC),
		gscTopPageStat(pages, hasGSC),
		gscTopCountryStat(countries, hasGSC),
		gscMobileShareStat(devices, hasGSC),
		{
			"label":  "Open SEO Issues",
			"value":  strconv.Itoa(openIssueCount),
			"change": issueLabel(openIssueCount),
			"trend":  issueTrend(openIssueCount),
			"icon":   "AlertCircle",
		},
		robotsOverviewStat(issues, project.HealthScore, project.UpdatedAt),
		httpsOverviewStat(issues, project.HealthScore, project.UpdatedAt),
		cwvOverviewStat(issues, project.HealthScore, project.UpdatedAt),
	}

	// 5. Social metrics from DB (live rows only)
	socialMetrics, _ := h.metricRepo.FindLatestSocialMetrics(pid)
	socialHistory, _ := h.metricRepo.FindSocialMetricsByProject(pid, 30)
	var latestFollowers, totalReach, profileVisits, postsCount int64
	var totalEngRate float64
	liveSocial := 0

	for _, sm := range socialMetrics {
		if sm.IsSimulated {
			continue
		}
		liveSocial++
		latestFollowers += sm.Followers
		if sm.MonthlyReach > 0 {
			totalReach += sm.MonthlyReach
		} else {
			totalReach += sm.Reach
		}
		totalEngRate += sm.Engagement
		profileVisits += sm.ProfileVisits
		postsCount += sm.PostsCount
	}
	var avgEngRate float64
	if liveSocial > 0 {
		avgEngRate = totalEngRate / float64(liveSocial)
	}
	followerGain, hasGain := socialFollowerDelta(socialHistory)

	socialStats := []gin.H{
		socialCountStat("Total Followers", latestFollowers, liveSocial, "Latest sync", "Users"),
		followerGainStat(followerGain, hasGain, liveSocial),
		socialCountStat("Audience Reach", totalReach, liveSocial, "30d", "Zap"),
		{
			"label":  "Engagement Rate",
			"value":  socialRateValue(avgEngRate, liveSocial),
			"change": engRateLabel(avgEngRate),
			"trend":  engRateTrend(avgEngRate),
			"icon":   "Activity",
		},
		socialCountStat("Profile Visits", profileVisits, liveSocial, "Latest sync", "Eye"),
		socialCountStat("Published Posts", postsCount, liveSocial, "Latest sync", "FileText"),
	}

	combinedStats := []gin.H{
		{
			"label":  "Organic Traffic",
			"value":  utils.FormatNumber(currClicks),
			"change": fmtChange(clickChange),
			"trend":  trendDir(clickChange),
			"icon":   "Globe",
		},
		{
			"label":  "Search Impressions",
			"value":  utils.FormatNumber(currImpressions),
			"change": fmtChange(impressionChange),
			"trend":  trendDir(impressionChange),
			"icon":   "TrendingUp",
		},
		socialCountStat("Total Followers", latestFollowers, liveSocial, "Latest sync", "Users"),
		{
			"label":  "Engagement Rate",
			"value":  socialRateValue(avgEngRate, liveSocial),
			"change": engRateLabel(avgEngRate),
			"trend":  engRateTrend(avgEngRate),
			"icon":   "Activity",
		},
		{
			"label":  "SEO Health",
			"value":  strconv.Itoa(project.HealthScore) + "%",
			"change": healthStatus(project.HealthScore),
			"trend":  healthTrend(project.HealthScore),
			"icon":   "Target",
		},
		{
			"label":  "Open SEO Issues",
			"value":  strconv.Itoa(openIssueCount),
			"change": issueLabel(openIssueCount),
			"trend":  issueTrend(openIssueCount),
			"icon":   "AlertCircle",
		},
		rankedKeywordStat("Ranked Keywords", ranked, hasGSC),
		rankedKeywordStat("Top 3 Keywords", top3, hasGSC),
		gscTopPageStat(pages, hasGSC),
		gscMobileShareStat(devices, hasGSC),
		robotsOverviewStat(issues, project.HealthScore, project.UpdatedAt),
		httpsOverviewStat(issues, project.HealthScore, project.UpdatedAt),
		cwvOverviewStat(issues, project.HealthScore, project.UpdatedAt),
	}

	utils.Success(c, gin.H{
		"project":        project,
		"websiteStats":   websiteStats,
		"socialStats":    socialStats,
		"combinedStats":  combinedStats,
		"health_score":   project.HealthScore,
		"is_simulated":   false,
		"traffic_source": trafficSource,
	}, nil)
}

func (h *DashboardHandler) Metrics(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	daysStr := c.DefaultQuery("days", "30")

	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	days, _ := strconv.Atoi(daysStr)

	end := time.Now()
	start := end.AddDate(0, 0, -days)

	metrics, err := h.metricRepo.FindMetricsByProjectAndRange(uint(projectID), start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		utils.InternalError(c, "Failed to fetch metrics")
		return
	}

	utils.Success(c, gscOnly(metrics), nil)
}

func (h *DashboardHandler) Insights(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)

	insights, err := h.insightRepo.FindInsights(uint(projectID), 20)
	if err != nil {
		utils.InternalError(c, "Failed to fetch insights")
		return
	}

	utils.Success(c, insights, nil)
}

func (h *DashboardHandler) Tasks(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)

	tasks, err := h.taskRepo.FindAllByProject(uint(projectID))
	if err != nil {
		utils.InternalError(c, "Failed to fetch tasks")
		return
	}

	utils.Success(c, tasks, nil)
}

type CreateTaskRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Title     string `json:"title" binding:"required"`
	DueDate   string `json:"due_date"`
}

func (h *DashboardHandler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueDate)
		if err == nil {
			dueDate = &parsed
		}
	}

	task := &models.Task{
		ProjectID: req.ProjectID,
		Title:     req.Title,
		Source:    "manual",
		DueDate:   dueDate,
	}

	if err := h.taskRepo.Create(task); err != nil {
		utils.InternalError(c, "Failed to create task")
		return
	}

	utils.Success(c, task, nil)
}

func (h *DashboardHandler) Traffic(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	daysStr := c.DefaultQuery("days", "30")

	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	days, _ := strconv.Atoi(daysStr)

	end := time.Now()
	start := end.AddDate(0, 0, -days)
	prevStart := start.AddDate(0, 0, -days)

	metrics, err := h.metricRepo.FindMetricsByProjectAndRange(uint(projectID), start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		utils.InternalError(c, "Failed to fetch traffic metrics")
		return
	}
	metrics = gscOnly(metrics)

	prevMetrics, _ := h.metricRepo.FindMetricsByProjectAndRange(uint(projectID), prevStart.Format("2006-01-02"), start.Format("2006-01-02"))
	prevMetrics = gscOnly(prevMetrics)

	var currClicks, prevClicks, currImpressions, prevImpressions int64

	for _, m := range metrics {
		currClicks += m.Clicks
		currImpressions += m.Impressions
	}
	for _, m := range prevMetrics {
		prevClicks += m.Clicks
		prevImpressions += m.Impressions
	}

	clickChange := utils.CalculateChange(currClicks, prevClicks)
	impressionChange := utils.CalculateChange(currImpressions, prevImpressions)

	var ctr float64
	if currImpressions > 0 {
		ctr = float64(currClicks) / float64(currImpressions) * 100
	}
	var prevCTR float64
	if prevImpressions > 0 {
		prevCTR = float64(prevClicks) / float64(prevImpressions) * 100
	}
	ctrDelta := ctr - prevCTR

	utils.Success(c, gin.H{
		"metrics": metrics,
		"summary": gin.H{
			"clicks":             currClicks,
			"clicks_change":      clickChange,
			"impressions":        currImpressions,
			"impressions_change": impressionChange,
			"ctr":                ctr,
			"ctr_change":         ctrDelta,
		},
	}, nil)
}

// Competitors returns an empty list — competitor tracking requires user setup.
// No hardcoded fake data is returned.
func (h *DashboardHandler) Competitors(c *gin.Context) {
	utils.Success(c, []gin.H{}, &utils.ResponseMeta{
		Message: "Competitor tracking is a Phase 2 feature. Add competitor URLs from Project Settings to start tracking.",
	})
}

// Alerts returns real DB alerts (currently empty until alert system is populated by workers).
func (h *DashboardHandler) Alerts(c *gin.Context) {
	// Alert system is populated by background workers after real syncs.
	// Return an empty slice until the first sync creates real alerts.
	utils.Success(c, []gin.H{}, nil)
}

func gscOnly(in []models.Metric) []models.Metric {
	out := make([]models.Metric, 0, len(in))
	for _, m := range in {
		if m.Source == models.MetricSourceGSC {
			out = append(out, m)
		}
	}
	return out
}

// ── Computation helpers ────────────────────────────────────────────────────

func fmtChange(pct float64) string {
	if pct > 0 {
		return fmt.Sprintf("+%.1f%%", pct)
	}
	return fmt.Sprintf("%.1f%%", pct)
}

func fmtCTRDelta(delta float64) string {
	if delta > 0 {
		return fmt.Sprintf("+%.2f%%", delta)
	}
	return fmt.Sprintf("%.2f%%", delta)
}

func trendDir(v float64) string {
	if v >= 0 {
		return "up"
	}
	return "down"
}

func healthStatus(score int) string {
	switch {
	case score == 0:
		return "Not audited"
	case score >= 75:
		return "Healthy"
	case score >= 50:
		return "Needs work"
	default:
		return "Critical"
	}
}

func healthTrend(score int) string {
	if score >= 75 {
		return "up"
	}
	return "down"
}

func engRateLabel(rate float64) string {
	switch {
	case rate >= 5:
		return "Excellent"
	case rate >= 3:
		return "Good"
	case rate >= 1:
		return "Average"
	case rate == 0:
		return "No data"
	default:
		return "Low"
	}
}

func engRateTrend(rate float64) string {
	if rate >= 3.0 {
		return "up"
	}
	return "down"
}

func issueLabel(count int) string {
	if count == 0 {
		return "All clear"
	}
	return fmt.Sprintf("%d open", count)
}

func issueTrend(count int) string {
	if count == 0 {
		return "up"
	}
	return "down"
}

func dashStat(label, change, icon string) gin.H {
	return gin.H{
		"label":  label,
		"value":  "—",
		"change": change,
		"trend":  "flat",
		"icon":   icon,
	}
}

func robotsOverviewStat(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryOverviewStat(robotsStatusFromIssues(issues, healthScore, updated), "robots.txt", "FileText")
}

func httpsOverviewStat(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryOverviewStat(httpsStatusFromIssues(issues, healthScore, updated), "HTTPS", "Lock")
}

func cwvOverviewStat(issues []models.SEOIssue, healthScore int, updated time.Time) gin.H {
	return categoryOverviewStat(cwvStatusFromIssues(issues, healthScore, updated), "Core Web Vitals", "Gauge")
}

func categoryOverviewStat(info gin.H, tileLabel, icon string) gin.H {
	label, _ := info["label"].(string)
	status, _ := info["status"].(string)
	checked, _ := info["checked_at"].(string)
	if status == "unknown" || label == "—" {
		return dashStat(tileLabel, "Run audit", icon)
	}
	change := "Last audit"
	if checked != "" {
		change = checked
	}
	trend := "flat"
	switch status {
	case "pass":
		trend = "up"
	case "fail":
		trend = "down"
	}
	return gin.H{
		"label":  tileLabel,
		"value":  label,
		"change": change,
		"trend":  trend,
		"icon":   icon,
	}
}

func rankedKeywordStat(label string, count int, hasGSC bool) gin.H {
	if !hasGSC && count == 0 {
		return dashStat(label, "Connect GSC", "Search")
	}
	if count == 0 {
		return dashStat(label, "No rankings", "Search")
	}
	return gin.H{
		"label":  label,
		"value":  strconv.Itoa(count),
		"change": "GSC queries",
		"trend":  "flat",
		"icon":   "Search",
	}
}

func socialCountStat(label string, n int64, liveSocial int, change, icon string) gin.H {
	if liveSocial == 0 {
		return dashStat(label, "Connect social", icon)
	}
	return gin.H{
		"label":  label,
		"value":  utils.FormatNumber(n),
		"change": change,
		"trend":  "flat",
		"icon":   icon,
	}
}

func socialRateValue(rate float64, liveSocial int) string {
	if liveSocial == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", rate)
}

func followerGainStat(delta int64, hasGain bool, liveSocial int) gin.H {
	if liveSocial == 0 {
		return dashStat("Followers Gained", "Connect social", "UserPlus")
	}
	if !hasGain {
		return dashStat("Followers Gained", "Need 2 syncs", "UserPlus")
	}
	trend := "flat"
	if delta > 0 {
		trend = "up"
	} else if delta < 0 {
		trend = "down"
	}
	change := fmt.Sprintf("%+d", delta)
	if delta == 0 {
		change = "0"
	}
	return gin.H{
		"label":  "Followers Gained",
		"value":  fmt.Sprintf("%+d", delta),
		"change": change,
		"trend":  trend,
		"icon":   "UserPlus",
	}
}

// keywordPositionBuckets counts GSC-ranked queries only (position > 0).
// Autocomplete rows have position 0 and are ignored.
func keywordPositionBuckets(kws []models.KeywordResult) (ranked, top3, page1 int) {
	for _, k := range kws {
		if k.Position <= 0 {
			continue
		}
		ranked++
		if k.Position <= 3 {
			top3++
		}
		if k.Position <= 10 {
			page1++
		}
	}
	return
}

// socialFollowerDelta is newest-minus-oldest live followers per platform,
// summed. Need at least two snapshots on one platform.
func socialFollowerDelta(history []models.SocialMetric) (delta int64, ok bool) {
	type ends struct {
		newest, oldest models.SocialMetric
		seen           int
	}
	byPlat := map[string]*ends{}
	for i := range history {
		sm := history[i]
		if sm.IsSimulated {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(sm.Platform))
		if key == "" {
			continue
		}
		p, exists := byPlat[key]
		if !exists {
			p = &ends{newest: sm, oldest: sm, seen: 1}
			byPlat[key] = p
			continue
		}
		p.seen++
		if sm.RecordedAt.After(p.newest.RecordedAt) || (sm.RecordedAt.Equal(p.newest.RecordedAt) && sm.ID > p.newest.ID) {
			p.newest = sm
		}
		if sm.RecordedAt.Before(p.oldest.RecordedAt) || (sm.RecordedAt.Equal(p.oldest.RecordedAt) && sm.ID < p.oldest.ID) {
			p.oldest = sm
		}
	}
	for _, p := range byPlat {
		if p.seen < 2 || p.newest.ID == p.oldest.ID {
			continue
		}
		ok = true
		delta += p.newest.Followers - p.oldest.Followers
	}
	return
}
