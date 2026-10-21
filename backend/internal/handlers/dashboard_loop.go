package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// gscCountStat is Organic Traffic / Search Impressions. Without GSC the
// tile is "—" — a stored 0 from seed or vendor must never appear here.
func gscCountStat(label string, n int64, change float64, hasGSC bool, icon string) gin.H {
	if !hasGSC {
		return dashStat(label, "Connect GSC", icon)
	}
	return gin.H{
		"label":  label,
		"value":  utils.FormatNumber(n),
		"change": fmtChange(change),
		"trend":  trendDir(change),
		"icon":   icon,
	}
}

func gscRateStat(label string, rate, delta float64, hasGSC bool, icon string) gin.H {
	if !hasGSC {
		return dashStat(label, "Connect GSC", icon)
	}
	return gin.H{
		"label":  label,
		"value":  fmt.Sprintf("%.1f%%", rate),
		"change": fmtCTRDelta(delta),
		"trend":  trendDir(delta),
		"icon":   icon,
	}
}

func todayStep(id, title, href string, done bool, detail string) gin.H {
	return gin.H{
		"id":     id,
		"title":  title,
		"href":   href,
		"done":   done,
		"detail": detail,
	}
}

// buildTodayLoop is the Overview daily checklist from stored state.
// Links never trigger a vendor fetch.
func buildTodayLoop(hasGSC bool, healthScore, openIssues int, domainLookedUp bool) []gin.H {
	issuesDone := healthScore > 0 && openIssues == 0
	issuesDetail := "Run an audit first"
	if healthScore > 0 {
		if openIssues == 0 {
			issuesDetail = "All clear from last homepage audit"
		} else {
			issuesDetail = fmt.Sprintf("%d open from last homepage audit", openIssues)
		}
	}
	return []gin.H{
		todayStep("connect_gsc", "Connect Google Search Console", "/integrations", hasGSC, "Overview traffic comes from GSC only."),
		todayStep("run_audit", "Run homepage audit", "/seo/site-explorer", healthScore > 0, "Homepage crawl and PageSpeed. Not a full-site crawl."),
		todayStep("lookup_domain", "Look up domain", "/seo/site-explorer", domainLookedUp, "DataForSEO Labs. Overview does not auto-fetch."),
		todayStep("keyword_labs", "Look up keywords", "/seo/keywords", false, "DataForSEO Labs extras. Lookup spends vendor credits."),
		todayStep("open_issues", "Review open SEO issues", "/seo/site-explorer", issuesDone, issuesDetail),
	}
}

func emptyVendorLookup(href string) gin.H {
	return gin.H{
		"available": false,
		"stale":     false,
		"href":      href,
	}
}

func vendorStale(fetchedAt time.Time) bool {
	if fetchedAt.IsZero() {
		return false
	}
	return time.Since(fetchedAt) >= vendorSnapshotTTL
}

func (h *DashboardHandler) vendorBrief(projectURL string) gin.H {
	host := services.NormalizeDomain(projectURL)
	brief := gin.H{
		"source":            "dataforseo",
		"domain":            host,
		"message":           "Cached DataForSEO lookups. Not Search Console. Overview never calls the vendor.",
		"domain_explorer":   emptyVendorLookup("/seo/site-explorer"),
		"ai_visibility":     emptyVendorLookup("/seo/ai-visibility"),
		"backlink_explorer": emptyVendorLookup("/seo/backlinks"),
	}
	if host == "" || h.seoRepo == nil {
		return brief
	}
	brief["domain_explorer"] = h.peekDomainExplorer(host)
	brief["ai_visibility"] = h.peekAIVisibility(host)
	brief["backlink_explorer"] = h.peekBacklinkExplorer(host)
	return brief
}

func (h *DashboardHandler) peekDomainExplorer(host string) gin.H {
	out := emptyVendorLookup("/seo/site-explorer")
	snap, err := h.seoRepo.FindVendorDomainSnapshot(host, services.DataForSEOLocation, services.DataForSEOLanguage)
	if err != nil || snap == nil || snap.Payload == "" {
		return out
	}
	var cached services.DomainExplorerResult
	if json.Unmarshal([]byte(snap.Payload), &cached) != nil || !cached.HasAnyData() {
		return out
	}
	out["available"] = true
	out["stale"] = vendorStale(snap.FetchedAt)
	out["source"] = "dataforseo"
	if !snap.FetchedAt.IsZero() {
		out["fetched_at"] = snap.FetchedAt.UTC().Format(time.RFC3339)
	}
	if cached.OrganicOK {
		out["organic_etv"] = cached.Organic.ETV
		out["organic_count"] = cached.Organic.Count
	}
	if cached.BacklinksOK {
		out["backlinks"] = cached.Backlinks.Backlinks
		out["referring_domains"] = cached.Backlinks.ReferringDomains
		out["rank"] = cached.Backlinks.Rank
	}
	return out
}

func (h *DashboardHandler) peekAIVisibility(host string) gin.H {
	out := emptyVendorLookup("/seo/ai-visibility")
	snap, err := h.seoRepo.FindVendorAIVisibilitySnapshot(host, services.DataForSEOLocation, services.DataForSEOLanguage)
	if err != nil || snap == nil || snap.Payload == "" {
		return out
	}
	var cached services.AIVisibilityResult
	if json.Unmarshal([]byte(snap.Payload), &cached) != nil || !cached.HasAnyData() {
		return out
	}
	out["available"] = true
	out["stale"] = vendorStale(snap.FetchedAt)
	out["source"] = "dataforseo"
	if !snap.FetchedAt.IsZero() {
		out["fetched_at"] = snap.FetchedAt.UTC().Format(time.RFC3339)
	}
	if cached.MentionsOK {
		out["mentions"] = cached.Mentions.Mentions
	}
	if cached.CitationsOK {
		out["citations"] = cached.Citations.Mentions
	}
	return out
}

func (h *DashboardHandler) peekBacklinkExplorer(host string) gin.H {
	out := emptyVendorLookup("/seo/backlinks")
	snap, err := h.seoRepo.FindVendorBacklinkExplorerSnapshot(host, services.DataForSEOLocation, services.DataForSEOLanguage)
	if err != nil || snap == nil || snap.Payload == "" {
		return out
	}
	var cached services.BacklinkExplorerResult
	if json.Unmarshal([]byte(snap.Payload), &cached) != nil || !cached.HasAnyData() {
		return out
	}
	out["available"] = true
	out["stale"] = vendorStale(snap.FetchedAt)
	out["source"] = "dataforseo"
	if !snap.FetchedAt.IsZero() {
		out["fetched_at"] = snap.FetchedAt.UTC().Format(time.RFC3339)
	}
	if cached.SummaryOK {
		out["backlinks"] = cached.Audit.Backlinks
		out["referring_domains"] = cached.Audit.ReferringDomains
		out["rank"] = cached.Audit.Rank
	}
	return out
}

func domainLookedUp(brief gin.H) bool {
	raw, ok := brief["domain_explorer"].(gin.H)
	if !ok {
		return false
	}
	available, _ := raw["available"].(bool)
	return available
}

type storedActionState struct {
	hasGSC         bool
	issues         []models.SEOIssue
	domainLookedUp bool
}

func (h *DashboardHandler) storedActionState(project *models.Project) storedActionState {
	st := storedActionState{issues: []models.SEOIssue{}}
	if project == nil || h.seoRepo == nil {
		return st
	}
	pid := project.ID
	now := time.Now()
	t0 := now.Format("2006-01-02")
	t30 := now.AddDate(0, 0, -30).Format("2006-01-02")
	t60 := now.AddDate(0, 0, -60).Format("2006-01-02")

	metricsCurrent, _ := h.metricRepo.FindMetricsByProjectAndRange(pid, t30, t0)
	metricsPrevious, _ := h.metricRepo.FindMetricsByProjectAndRange(pid, t60, t30)
	metricsCurrent = gscOnly(metricsCurrent)
	metricsPrevious = gscOnly(metricsPrevious)

	issues, _ := h.seoRepo.FindOpenIssues(pid, "")
	if issues == nil {
		issues = []models.SEOIssue{}
	}
	keywords, _, _ := h.seoRepo.FindKeywords(pid, "")
	ranked, _, _ := keywordPositionBuckets(keywords)
	pages, _ := h.seoRepo.FindBreakdowns(pid, models.GSCDimPage)
	countries, _ := h.seoRepo.FindBreakdowns(pid, models.GSCDimCountry)
	devices, _ := h.seoRepo.FindBreakdowns(pid, models.GSCDimDevice)
	traffic := len(metricsCurrent) > 0 || len(metricsPrevious) > 0
	st.hasGSC = traffic || ranked > 0 || len(pages)+len(countries)+len(devices) > 0
	st.issues = issues
	st.domainLookedUp = domainLookedUp(h.vendorBrief(project.URL))
	return st
}

// buildActionWork is the Action Center list: Overview loop steps plus
// each open SEO issue. Items are links. They are not Task rows.
func buildActionWork(hasGSC bool, healthScore int, domainLookedUp bool, issues []models.SEOIssue) []gin.H {
	if issues == nil {
		issues = []models.SEOIssue{}
	}
	loop := buildTodayLoop(hasGSC, healthScore, len(issues), domainLookedUp)
	work := make([]gin.H, 0, len(loop)+len(issues))
	for _, step := range loop {
		step["kind"] = "loop"
		work = append(work, step)
	}
	for _, iss := range issues {
		title := strings.TrimSpace(iss.Detail)
		if title == "" {
			title = strings.TrimSpace(iss.Category)
		}
		if title == "" {
			title = "Open SEO issue"
		}
		detail := strings.TrimSpace(iss.Severity)
		if iss.Category != "" {
			if detail == "" {
				detail = iss.Category
			} else {
				detail = detail + " · " + iss.Category
			}
		}
		work = append(work, gin.H{
			"id":       "issue-" + strconv.FormatUint(uint64(iss.ID), 10),
			"kind":     "issue",
			"title":    title,
			"href":     "/seo/site-explorer",
			"done":     false,
			"detail":   detail,
			"severity": iss.Severity,
		})
	}
	return work
}
