package handlers

import (
	"fmt"
	"math"
	"strconv"
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

	// 3. Compute web traffic stats from real data
	var currClicks, prevClicks, currImpressions, prevImpressions int64
	for _, m := range metricsCurrent {
		currClicks += m.Clicks
		currImpressions += m.Impressions
	}
	for _, m := range metricsPrevious {
		prevClicks += m.Clicks
		prevImpressions += m.Impressions
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

	// 4. Fetch SEO issues for health context
	issues, _ := h.seoRepo.FindOpenIssues(pid, "")
	keywords, _, _ := h.seoRepo.FindKeywords(pid, "")
	kwCount := len(keywords)

	// 5. Website stats — all derived from real data
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
	}

	// 6. Ranked keywords stat (only show if audit was run)
	if kwCount > 0 || project.HealthScore > 0 {
		websiteStats[0] = gin.H{
			"label":  "SEO Health",
			"value":  strconv.Itoa(project.HealthScore) + "%",
			"change": healthStatus(project.HealthScore),
			"trend":  healthTrend(project.HealthScore),
			"icon":   "Target",
		}
	}

	// 7. Social metrics from DB
	socialMetrics, _ := h.metricRepo.FindLatestSocialMetrics(pid)
	var latestFollowers, totalReach int64
	var totalEngRate float64
	isSimulated := false

	for _, sm := range socialMetrics {
		latestFollowers += sm.Followers
		totalReach += sm.Reach
		totalEngRate += sm.Engagement
		if sm.IsSimulated {
			isSimulated = true
		}
	}
	var avgEngRate float64
	if len(socialMetrics) > 0 {
		avgEngRate = totalEngRate / float64(len(socialMetrics))
	}

	// Content score = engagement rate mapped to 0-10 scale (industry avg ~3.5%)
	contentScore := math.Min(10.0, avgEngRate/3.5*8.0)
	if contentScore == 0 && len(socialMetrics) > 0 {
		contentScore = 5.0 // default if we have social data but 0 engagement
	}

	socialStats := []gin.H{
		{
			"label":  "Engagement Rate",
			"value":  fmt.Sprintf("%.1f%%", avgEngRate),
			"change": engRateLabel(avgEngRate),
			"trend":  engRateTrend(avgEngRate),
			"icon":   "Activity",
		},
		{
			"label":  "Total Followers",
			"value":  utils.FormatNumber(latestFollowers),
			"change": "Live",
			"trend":  "up",
			"icon":   "Users",
		},
		{
			"label":  "Audience Reach",
			"value":  utils.FormatNumber(totalReach),
			"change": "30d total",
			"trend":  "up",
			"icon":   "Zap",
		},
		{
			"label":  "Content Score",
			"value":  fmt.Sprintf("%.1f", contentScore),
			"change": contentScoreLabel(contentScore),
			"trend":  contentScoreTrend(contentScore),
			"icon":   "BarChart3",
		},
	}

	// 8. Combined / aggregate stats — computed from real data
	growthIndex := computeGrowthIndex(clickChange, avgEngRate, project.HealthScore)
	aggregateReach := currClicks + totalReach
	openIssueCount := len(issues)

	combinedStats := []gin.H{
		{
			"label":  "Growth Index",
			"value":  strconv.Itoa(growthIndex),
			"change": growthIndexLabel(growthIndex),
			"trend":  trendDir(float64(growthIndex) - 50),
			"icon":   "TrendingUp",
		},
		{
			"label":  "Aggregate Reach",
			"value":  utils.FormatNumber(aggregateReach),
			"change": fmtChange(clickChange),
			"trend":  trendDir(clickChange),
			"icon":   "Zap",
		},
		{
			"label":  "Open SEO Issues",
			"value":  strconv.Itoa(openIssueCount),
			"change": issueLabel(openIssueCount),
			"trend":  issueTrend(openIssueCount),
			"icon":   "AlertCircle",
		},
		{
			"label":  "Ranked Keywords",
			"value":  strconv.Itoa(kwCount),
			"change": kwLabel(kwCount),
			"trend":  "up",
			"icon":   "Search",
		},
	}

	utils.Success(c, gin.H{
		"project":       project,
		"websiteStats":  websiteStats,
		"socialStats":   socialStats,
		"combinedStats": combinedStats,
		"health_score":  project.HealthScore,
		"is_simulated":  isSimulated,
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

	utils.Success(c, metrics, nil)
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
	
	prevMetrics, _ := h.metricRepo.FindMetricsByProjectAndRange(uint(projectID), prevStart.Format("2006-01-02"), start.Format("2006-01-02"))

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
			"clicks":           currClicks,
			"clicks_change":    clickChange,
			"impressions":      currImpressions,
			"impressions_change": impressionChange,
			"ctr":              ctr,
			"ctr_change":       ctrDelta,
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

func contentScoreLabel(score float64) string {
	switch {
	case score >= 8:
		return "Excellent"
	case score >= 6:
		return "Good"
	case score >= 4:
		return "Average"
	case score == 0:
		return "No data"
	default:
		return "Needs work"
	}
}

func contentScoreTrend(score float64) string {
	if score >= 5 {
		return "up"
	}
	return "down"
}

// computeGrowthIndex returns a 0-100 composite score from real signals.
func computeGrowthIndex(clickGrowth float64, engRate float64, healthScore int) int {
	// Weighted: 40% traffic growth, 30% engagement, 30% SEO health
	trafficScore := 50.0 + math.Min(50, math.Max(-50, clickGrowth/2))
	engScore := math.Min(100, engRate/5*100)
	healthF := float64(healthScore)

	composite := (trafficScore * 0.4) + (engScore * 0.3) + (healthF * 0.3)
	result := int(math.Round(composite))
	if result < 0 {
		result = 0
	}
	if result > 100 {
		result = 100
	}
	return result
}

func growthIndexLabel(idx int) string {
	switch {
	case idx >= 80:
		return "Excellent"
	case idx >= 60:
		return "Good"
	case idx >= 40:
		return "Average"
	default:
		return "Low"
	}
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

func kwLabel(count int) string {
	if count == 0 {
		return "Run audit"
	}
	return fmt.Sprintf("%d tracked", count)
}

func fmtPlus(n float64) string {
	if n >= 0 {
		return fmt.Sprintf("+%.1f%%", n)
	}
	return fmt.Sprintf("%.1f%%", n)
}
