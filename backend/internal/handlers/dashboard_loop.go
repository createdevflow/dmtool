package handlers

import (
	"encoding/json"
	"fmt"
	"time"

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
