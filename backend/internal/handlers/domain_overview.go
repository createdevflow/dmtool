package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend/internal/domainoverview"
	"backend/internal/models"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// DomainOverview is the Semrush-style Domain Overview report.
// GET is cache-only unless fetch=1 or refresh=1. Vendor estimates never
// write into dashboard metrics. Daily Search caps follow Semrush.
func (h *SEOHandler) DomainOverview(c *gin.Context) {
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

	limits := h.doLimits(userID, ptIsAdmin(c))
	used := 0
	if h.seoRepo != nil {
		used, _ = h.seoRepo.CountDomainOverviewRequests(userID, startOfUTCDay())
	}
	remaining := limits.RequestsPerDay - used
	if remaining < 0 {
		remaining = 0
	}

	domain := c.Query("domain")
	if domain == "" {
		domain = project.URL
	}
	host := services.NormalizeDomain(domain)
	refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")
	wantFetch := refresh || c.Query("fetch") == "1" || strings.EqualFold(c.Query("fetch"), "true")

	emptyPayload := func(configured bool, message string) {
		utils.Success(c, gin.H{
			"limits":     limits,
			"used_today": used,
			"remaining":  remaining,
			"available":  false,
			"configured": configured,
			"domain":     host,
			"source":     "dataforseo",
			"cached":     false,
			"message":    message,
			"reports":    domainoverview.BuildReports(limits, services.DomainExplorerResult{Domain: host}, false),
		}, nil)
	}

	if !h.dataForSEOReady() {
		emptyPayload(false, "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD.")
		return
	}

	if !wantFetch {
		if cached, ok := h.cachedDomainExplorer(host); ok {
			utils.Success(c, doJSON(limits, used, remaining, cached, true, "Cached DataForSEO Labs snapshot. Click Search to use a daily request."), nil)
			return
		}
		emptyPayload(true, "Click Search to query DataForSEO. This uses vendor credits and one daily request; results cache 24 hours.")
		return
	}

	if used >= limits.RequestsPerDay {
		utils.Forbidden(c, fmt.Sprintf("Daily Domain Overview limit reached (%d searches on %s).", limits.RequestsPerDay, limits.PlanLabel))
		return
	}
	if h.seoRepo != nil {
		_ = h.seoRepo.InsertDomainOverviewRequest(userID, host)
		used++
		remaining = limits.RequestsPerDay - used
		if remaining < 0 {
			remaining = 0
		}
	}

	result, cached, err := h.loadDomainExplorer(c.Request.Context(), host, refresh)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not connected") {
			emptyPayload(false, "DataForSEO is not connected.")
			return
		}
		utils.Success(c, gin.H{
			"limits":     limits,
			"used_today": used,
			"remaining":  remaining,
			"available":  false,
			"configured": true,
			"domain":     host,
			"source":     "dataforseo",
			"cached":     false,
			"message":    msg,
			"reports":    domainoverview.BuildReports(limits, services.DomainExplorerResult{Domain: host}, false),
		}, nil)
		return
	}
	result.Cached = cached
	msg := "DataForSEO Labs estimates for Google United States. Not Google Search Console and not a sitewide crawl."
	if cached {
		msg = "Served from 24h cache. Daily request still counted because you searched."
	}
	utils.Success(c, doJSON(limits, used, remaining, result, cached, msg), nil)
}

func doJSON(limits domainoverview.Limits, used, remaining int, result services.DomainExplorerResult, cached bool, message string) gin.H {
	return gin.H{
		"limits":         limits,
		"used_today":     used,
		"remaining":      remaining,
		"available":      result.HasAnyData(),
		"configured":     true,
		"source":         result.Source,
		"domain":         result.Domain,
		"location_code":  result.LocationCode,
		"location_name":  result.LocationName,
		"language_code":  result.LanguageCode,
		"cached":         cached,
		"fetched_at":     result.FetchedAt.Format(time.RFC3339),
		"message":        message,
		"organic_ok":     result.OrganicOK,
		"backlinks_ok":   result.BacklinksOK,
		"competitors_ok": result.CompetitorsOK,
		"keywords_ok":    result.KeywordsOK,
		"reports":        domainoverview.BuildReports(limits, result, result.HasAnyData()),
	}
}

func (h *SEOHandler) doLimits(userID uint, isAdmin bool) domainoverview.Limits {
	plan := models.PlanCodeFree
	if h.seoRepo != nil {
		if code, err := h.seoRepo.FindPlanCodeByUser(userID); err == nil && code != "" {
			plan = code
		}
	}
	return domainoverview.ForPlan(plan, isAdmin)
}

func startOfUTCDay() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
