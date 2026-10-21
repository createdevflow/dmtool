package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestKeywordPositionBuckets(t *testing.T) {
	kws := []models.KeywordResult{
		{Keyword: "a", Position: 1},
		{Keyword: "b", Position: 3},
		{Keyword: "c", Position: 10},
		{Keyword: "d", Position: 11},
		{Keyword: "suggest", Position: 0},
		{Keyword: "empty", Position: -1},
	}
	ranked, top3, page1 := keywordPositionBuckets(kws)
	if ranked != 4 {
		t.Fatalf("ranked=%d want 4", ranked)
	}
	if top3 != 2 {
		t.Fatalf("top3=%d want 2", top3)
	}
	if page1 != 3 {
		t.Fatalf("page1=%d want 3", page1)
	}
}

func TestSocialFollowerDelta(t *testing.T) {
	t0 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(24 * time.Hour)
	history := []models.SocialMetric{
		{ID: 2, Platform: "instagram", Followers: 120, RecordedAt: t1},
		{ID: 1, Platform: "instagram", Followers: 100, RecordedAt: t0},
		{ID: 4, Platform: "facebook", Followers: 50, RecordedAt: t1},
		{ID: 3, Platform: "facebook", Followers: 80, RecordedAt: t0},
		{ID: 9, Platform: "instagram", Followers: 999, IsSimulated: true, RecordedAt: t1},
	}
	delta, ok := socialFollowerDelta(history)
	if !ok {
		t.Fatal("expected delta from two snapshots")
	}
	// IG +20, FB -30 → -10
	if delta != -10 {
		t.Fatalf("delta=%d want -10", delta)
	}
}

func TestSocialFollowerDeltaNeedsTwo(t *testing.T) {
	t0 := time.Now()
	delta, ok := socialFollowerDelta([]models.SocialMetric{
		{ID: 1, Platform: "instagram", Followers: 10, RecordedAt: t0},
	})
	if ok || delta != 0 {
		t.Fatalf("ok=%v delta=%d want no delta", ok, delta)
	}
}

func TestSnapshot_OverviewTilesFromStoredData(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	p.HealthScore = 70
	if err := database.Save(p).Error; err != nil {
		t.Fatal(err)
	}
	day := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	if err := database.Create(&models.Metric{
		ProjectID: p.ID, Date: day, Clicks: 12, Impressions: 200, Source: models.MetricSourceGSC,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.KeywordResult{
		ProjectID: p.ID, Seed: "best shoes", Keyword: "best shoes", Position: 2,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.KeywordResult{
		ProjectID: p.ID, Seed: "suggest", Keyword: "suggest only", Position: 0,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.SocialMetric{
		ProjectID: p.ID, Platform: "instagram", Followers: 50, Reach: 9,
		Engagement: 1.5, ProfileVisits: 4, PostsCount: 8, IsSimulated: false,
		RecordedAt: time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}

	h := NewDashboardHandler(
		repository.NewProjectRepository(database),
		repository.NewMetricRepository(database),
		repository.NewInsightRepository(database),
		repository.NewSEORepository(database),
		repository.NewTaskRepository(database),
	)
	r := gin.New()
	r.GET("/dashboard/snapshot", withUser(u.ID, h.Snapshot))
	req := httptest.NewRequest(http.MethodGet, "/dashboard/snapshot?project_id="+strconv.FormatUint(uint64(p.ID), 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		WebsiteStats  []map[string]any `json:"websiteStats"`
		SocialStats   []map[string]any `json:"socialStats"`
		CombinedStats []map[string]any `json:"combinedStats"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &payload); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}

	web := labels(payload.WebsiteStats)
	if !containsAll(web, "SEO Health", "Organic Traffic", "Search Impressions", "Click-Through Rate",
		"Ranked Keywords", "Top 3 Keywords", "Page 1 Keywords", "Top Page", "Top Country", "Mobile Traffic",
		"Open SEO Issues", "robots.txt", "HTTPS", "Core Web Vitals", "PageSpeed Lab", "XML Sitemap", "Indexability", "International SEO", "Structured Data") {
		t.Fatalf("website labels=%v", web)
	}
	if got := valueFor(payload.WebsiteStats, "Top Page"); got != "—" {
		t.Fatalf("Top Page=%q want — (no GSC breakdowns)", got)
	}
	if contains(web, "Growth Index") || contains(web, "Content Score") {
		t.Fatalf("unexpected computed tiles in website: %v", web)
	}
	if got := valueFor(payload.WebsiteStats, "Ranked Keywords"); got != "1" {
		t.Fatalf("Ranked Keywords=%q want 1 (autocomplete row ignored)", got)
	}
	if got := valueFor(payload.WebsiteStats, "Top 3 Keywords"); got != "1" {
		t.Fatalf("Top 3 Keywords=%q want 1", got)
	}

	soc := labels(payload.SocialStats)
	if !containsAll(soc, "Total Followers", "Followers Gained", "Audience Reach", "Engagement Rate", "Profile Visits", "Published Posts", "Link Taps", "Weekly Reach", "Top Audience", "Best Posting Day") {
		t.Fatalf("social labels=%v", soc)
	}
	if contains(soc, "Content Score") {
		t.Fatalf("Content Score should be gone: %v", soc)
	}
	if got := valueFor(payload.SocialStats, "Published Posts"); got != "8" {
		t.Fatalf("Published Posts=%q want 8", got)
	}
	if got := valueFor(payload.SocialStats, "Followers Gained"); got != "—" {
		t.Fatalf("Followers Gained=%q want — (only one snapshot)", got)
	}

	comb := labels(payload.CombinedStats)
	if contains(comb, "Growth Index") || contains(comb, "Aggregate Reach") {
		t.Fatalf("combined still has computed composites: %v", comb)
	}
	if !contains(comb, "robots.txt") {
		t.Fatalf("combined missing robots.txt: %v", comb)
	}
	if !contains(comb, "HTTPS") {
		t.Fatalf("combined missing HTTPS: %v", comb)
	}
	if !contains(comb, "Core Web Vitals") {
		t.Fatalf("combined missing Core Web Vitals: %v", comb)
	}
	if !contains(comb, "XML Sitemap") {
		t.Fatalf("combined missing XML Sitemap: %v", comb)
	}
	if !contains(comb, "PageSpeed Lab") {
		t.Fatalf("combined missing PageSpeed Lab: %v", comb)
	}
	if !contains(comb, "Indexability") {
		t.Fatalf("combined missing Indexability: %v", comb)
	}
	if !contains(comb, "International SEO") {
		t.Fatalf("combined missing International SEO: %v", comb)
	}
	if !contains(comb, "Structured Data") {
		t.Fatalf("combined missing Structured Data: %v", comb)
	}
	if !contains(comb, "Top Page") || !contains(comb, "Mobile Traffic") {
		t.Fatalf("combined missing GSC extras: %v", comb)
	}
}

func TestSnapshot_GSCBreakdownTiles(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	if err := database.Create(&models.Metric{
		ProjectID: p.ID, Date: time.Now().Format("2006-01-02"), Clicks: 5, Impressions: 50, Source: models.MetricSourceGSC,
	}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	rows := []models.GSCBreakdown{
		{ProjectID: p.ID, Dimension: models.GSCDimPage, Key: "https://example.com/pricing", Clicks: 20, Impressions: 100, CTR: 0.2, Position: 3, StartDate: "2026-09-03", EndDate: "2026-10-03", FetchedAt: now},
		{ProjectID: p.ID, Dimension: models.GSCDimPage, Key: "https://example.com/", Clicks: 8, Impressions: 40, FetchedAt: now},
		{ProjectID: p.ID, Dimension: models.GSCDimCountry, Key: "usa", Clicks: 18, Impressions: 80, FetchedAt: now},
		{ProjectID: p.ID, Dimension: models.GSCDimDevice, Key: "MOBILE", Clicks: 15, Impressions: 70, FetchedAt: now},
		{ProjectID: p.ID, Dimension: models.GSCDimDevice, Key: "DESKTOP", Clicks: 5, Impressions: 30, FetchedAt: now},
	}
	if err := database.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	h := NewDashboardHandler(
		repository.NewProjectRepository(database),
		repository.NewMetricRepository(database),
		repository.NewInsightRepository(database),
		repository.NewSEORepository(database),
		repository.NewTaskRepository(database),
	)
	r := gin.New()
	r.GET("/dashboard/snapshot", withUser(u.ID, h.Snapshot))
	req := httptest.NewRequest(http.MethodGet, "/dashboard/snapshot?project_id="+strconv.FormatUint(uint64(p.ID), 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var payload struct {
		WebsiteStats []map[string]any `json:"websiteStats"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &payload); err != nil {
		t.Fatal(err)
	}
	if got := valueFor(payload.WebsiteStats, "Top Page"); got != "/pricing" {
		t.Fatalf("Top Page=%q want /pricing", got)
	}
	if got := changeFor(payload.WebsiteStats, "Top Page"); got != "20 clicks" {
		t.Fatalf("Top Page change=%q want 20 clicks (non-numeric trend)", got)
	}
	if got := valueFor(payload.WebsiteStats, "Top Country"); got != "United States" {
		t.Fatalf("Top Country=%q", got)
	}
	if got := valueFor(payload.WebsiteStats, "Mobile Traffic"); got != "75.0%" {
		t.Fatalf("Mobile Traffic=%q want 75.0%%", got)
	}
	if got := changeFor(payload.WebsiteStats, "Mobile Traffic"); got != "GSC devices" {
		t.Fatalf("Mobile Traffic change=%q", got)
	}
}

func TestRobotsStatusFromIssues(t *testing.T) {
	updated := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	unknown := robotsStatusFromIssues(nil, 0, updated)
	if unknown["status"] != "unknown" {
		t.Fatalf("no audit want unknown, got %+v", unknown)
	}
	pass := robotsStatusFromIssues(nil, 70, updated)
	if pass["label"] != "Pass" {
		t.Fatalf("audited with no robots issues want Pass, got %+v", pass)
	}
	fail := robotsStatusFromIssues([]models.SEOIssue{
		{Category: "robots", Severity: models.SeverityHigh, Detail: "blocked"},
	}, 40, updated)
	if fail["label"] != "Fail" {
		t.Fatalf("want Fail, got %+v", fail)
	}
}

func TestHttpsStatusFromIssues(t *testing.T) {
	updated := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	unknown := httpsStatusFromIssues(nil, 0, updated)
	if unknown["status"] != "unknown" {
		t.Fatalf("no audit want unknown, got %+v", unknown)
	}
	pass := httpsStatusFromIssues(nil, 70, updated)
	if pass["label"] != "Pass" {
		t.Fatalf("audited with no https issues want Pass, got %+v", pass)
	}
	// robots issues must not flip HTTPS
	ignoreRobots := httpsStatusFromIssues([]models.SEOIssue{
		{Category: "robots", Severity: models.SeverityHigh, Detail: "blocked"},
	}, 40, updated)
	if ignoreRobots["label"] != "Pass" {
		t.Fatalf("robots fail should not mark HTTPS fail, got %+v", ignoreRobots)
	}
	fail := httpsStatusFromIssues([]models.SEOIssue{
		{Category: "https", Severity: models.SeverityHigh, Detail: "mixed content"},
	}, 40, updated)
	if fail["label"] != "Fail" {
		t.Fatalf("want Fail, got %+v", fail)
	}
	warn := httpsStatusFromIssues([]models.SEOIssue{
		{Category: "https", Severity: models.SeverityMed, Detail: "no HSTS"},
	}, 80, updated)
	if warn["label"] != "Warning" {
		t.Fatalf("want Warning, got %+v", warn)
	}
}

func TestCwvStatusFromIssues(t *testing.T) {
	updated := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	unknown := cwvStatusFromIssues(nil, 0, updated)
	if unknown["status"] != "unknown" {
		t.Fatalf("no audit want unknown, got %+v", unknown)
	}
	pass := cwvStatusFromIssues(nil, 70, updated)
	if pass["label"] != "Good" {
		t.Fatalf("audited with no cwv issues want Good, got %+v", pass)
	}
	fail := cwvStatusFromIssues([]models.SEOIssue{
		{Category: "cwv", Severity: models.SeverityHigh, Detail: "poor LCP"},
	}, 40, updated)
	if fail["label"] != "Poor" {
		t.Fatalf("want Poor, got %+v", fail)
	}
	needs := cwvStatusFromIssues([]models.SEOIssue{
		{Category: "cwv", Severity: models.SeverityMed, Detail: "no CrUX"},
	}, 80, updated)
	if needs["label"] != "Needs work" {
		t.Fatalf("want Needs work, got %+v", needs)
	}
	ignoreLab := cwvStatusFromIssues([]models.SEOIssue{
		{Category: "lighthouse", Severity: models.SeverityHigh, Detail: "lab 40"},
	}, 70, updated)
	if ignoreLab["label"] != "Good" {
		t.Fatalf("lighthouse must not flip CWV tile, got %+v", ignoreLab)
	}
}

func TestSitemapStatusFromIssues(t *testing.T) {
	updated := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	unknown := sitemapStatusFromIssues(nil, 0, updated)
	if unknown["status"] != "unknown" {
		t.Fatalf("no audit want unknown, got %+v", unknown)
	}
	pass := sitemapStatusFromIssues(nil, 70, updated)
	if pass["label"] != "Pass" {
		t.Fatalf("audited with no sitemap issues want Pass, got %+v", pass)
	}
	fail := sitemapStatusFromIssues([]models.SEOIssue{
		{Category: "sitemap", Severity: models.SeverityHigh, Detail: "declared 404"},
	}, 40, updated)
	if fail["label"] != "Fail" {
		t.Fatalf("want Fail, got %+v", fail)
	}
	ignoreRobots := sitemapStatusFromIssues([]models.SEOIssue{
		{Category: "robots", Severity: models.SeverityHigh, Detail: "blocked"},
	}, 70, updated)
	if ignoreRobots["label"] != "Pass" {
		t.Fatalf("robots fail should not mark sitemap fail, got %+v", ignoreRobots)
	}
}

func TestLighthouseStatusFromIssues(t *testing.T) {
	updated := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	unknown := lighthouseStatusFromIssues(nil, 0, updated)
	if unknown["status"] != "unknown" {
		t.Fatalf("no audit want unknown, got %+v", unknown)
	}
	pass := lighthouseStatusFromIssues(nil, 70, updated)
	if pass["label"] != "Pass" {
		t.Fatalf("audited with no lighthouse issues want Pass, got %+v", pass)
	}
	fail := lighthouseStatusFromIssues([]models.SEOIssue{
		{Category: "lighthouse", Severity: models.SeverityHigh, Detail: "lab 40"},
	}, 40, updated)
	if fail["label"] != "Fail" {
		t.Fatalf("want Fail, got %+v", fail)
	}
	ignoreCWV := lighthouseStatusFromIssues([]models.SEOIssue{
		{Category: "cwv", Severity: models.SeverityHigh, Detail: "poor LCP"},
	}, 70, updated)
	if ignoreCWV["label"] != "Pass" {
		t.Fatalf("cwv fail should not mark PageSpeed Lab fail, got %+v", ignoreCWV)
	}
}

func TestInternationalAndMarkupStatusFromIssues(t *testing.T) {
	updated := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	if internationalStatusFromIssues(nil, 0, updated)["status"] != "unknown" {
		t.Fatal("no audit want unknown international")
	}
	if markupStatusFromIssues(nil, 70, updated)["label"] != "Pass" {
		t.Fatal("audited with no markup issues want Pass")
	}
	fail := internationalStatusFromIssues([]models.SEOIssue{
		{Category: "international", Severity: models.SeverityHigh, Detail: "bad hreflang"},
	}, 40, updated)
	if fail["label"] != "Fail" {
		t.Fatalf("want Fail, got %+v", fail)
	}
	ignore := markupStatusFromIssues([]models.SEOIssue{
		{Category: "cwv", Severity: models.SeverityHigh, Detail: "poor LCP"},
	}, 70, updated)
	if ignore["label"] != "Pass" {
		t.Fatalf("cwv fail should not mark markup fail, got %+v", ignore)
	}
}

func TestIndexabilityStatusFromIssues(t *testing.T) {
	updated := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	unknown := indexabilityStatusFromIssues(nil, 0, updated)
	if unknown["status"] != "unknown" {
		t.Fatalf("no audit want unknown, got %+v", unknown)
	}
	pass := indexabilityStatusFromIssues(nil, 70, updated)
	if pass["label"] != "Pass" {
		t.Fatalf("audited with no indexability issues want Pass, got %+v", pass)
	}
	fail := indexabilityStatusFromIssues([]models.SEOIssue{
		{Category: "indexability", Severity: models.SeverityHigh, Detail: "noindex"},
	}, 40, updated)
	if fail["label"] != "Fail" {
		t.Fatalf("want Fail, got %+v", fail)
	}
	ignoreSitemap := indexabilityStatusFromIssues([]models.SEOIssue{
		{Category: "sitemap", Severity: models.SeverityHigh, Detail: "404"},
	}, 70, updated)
	if ignoreSitemap["label"] != "Pass" {
		t.Fatalf("sitemap fail should not mark indexability fail, got %+v", ignoreSitemap)
	}
}

func TestSnapshot_NoGSCTrafficShowsEmDash(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	payload := getSnapshot(t, database, u.ID, p.ID)

	if payload.TrafficSource != "" {
		t.Fatalf("traffic_source=%q want empty", payload.TrafficSource)
	}
	if got := valueFor(payload.WebsiteStats, "Organic Traffic"); got != "—" {
		t.Fatalf("Organic Traffic=%q want —", got)
	}
	if got := changeFor(payload.WebsiteStats, "Organic Traffic"); got != "Connect GSC" {
		t.Fatalf("Organic Traffic change=%q want Connect GSC", got)
	}
	if got := valueFor(payload.WebsiteStats, "Search Impressions"); got != "—" {
		t.Fatalf("Search Impressions=%q want —", got)
	}
	if got := valueFor(payload.WebsiteStats, "Click-Through Rate"); got != "—" {
		t.Fatalf("CTR=%q want —", got)
	}
	if got := valueFor(payload.CombinedStats, "Organic Traffic"); got != "—" {
		t.Fatalf("combined Organic Traffic=%q want —", got)
	}
	if got := valueFor(payload.WebsiteStats, "SEO Health"); got != "—" {
		t.Fatalf("SEO Health=%q want — when not audited", got)
	}
	if got := changeFor(payload.WebsiteStats, "SEO Health"); got != "Not audited" {
		t.Fatalf("SEO Health change=%q want Not audited", got)
	}
	if got := valueFor(payload.CombinedStats, "SEO Health"); got != "—" {
		t.Fatalf("combined SEO Health=%q want —", got)
	}
}

func TestSnapshot_VendorBriefCacheOnlyDoesNotChangeTraffic(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	if err := database.Create(&models.Metric{
		ProjectID: p.ID, Date: time.Now().AddDate(0, 0, -2).Format("2006-01-02"),
		Clicks: 12, Impressions: 200, Source: models.MetricSourceGSC,
	}).Error; err != nil {
		t.Fatal(err)
	}
	seoRepo := repository.NewSEORepository(database)
	now := time.Now().UTC()
	domainPayload, _ := json.Marshal(services.DomainExplorerResult{
		Domain:       "example.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		OrganicOK:    true,
		Organic:      services.OrganicOverview{ETV: 50, Count: 3},
		BacklinksOK:  true,
		Backlinks:    services.BacklinkOverview{Backlinks: 7, ReferringDomains: 2, Rank: 10},
		FetchedAt:    now,
	})
	if err := seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
		Domain:       "example.com",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		Source:       "dataforseo",
		Payload:      string(domainPayload),
		FetchedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
	aiPayload, _ := json.Marshal(services.AIVisibilityResult{
		Domain:       "example.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		MentionsOK:   true,
		Mentions:     services.LLMMetrics{Mentions: 4},
		CitationsOK:  true,
		Citations:    services.LLMMetrics{Mentions: 15},
		FetchedAt:    now,
	})
	if err := seoRepo.SaveVendorAIVisibilitySnapshot(&models.VendorAIVisibilitySnapshot{
		Domain:       "example.com",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		Source:       "dataforseo",
		Payload:      string(aiPayload),
		FetchedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
	blPayload, _ := json.Marshal(services.BacklinkExplorerResult{
		Domain:       "example.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		SummaryOK:    true,
		Audit:        services.BacklinkAudit{Backlinks: 99, ReferringDomains: 12, Rank: 371},
		FetchedAt:    now,
	})
	if err := seoRepo.SaveVendorBacklinkExplorerSnapshot(&models.VendorBacklinkExplorerSnapshot{
		Domain:       "example.com",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		Source:       "dataforseo",
		Payload:      string(blPayload),
		FetchedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}

	payload := getSnapshot(t, database, u.ID, p.ID)
	if payload.TrafficSource != models.MetricSourceGSC {
		t.Fatalf("traffic_source=%q want %q", payload.TrafficSource, models.MetricSourceGSC)
	}
	if payload.TrafficSource == "dataforseo" {
		t.Fatal("traffic_source must not be dataforseo")
	}
	if got := valueFor(payload.WebsiteStats, "Organic Traffic"); got != "12" {
		t.Fatalf("Organic Traffic=%q want 12 (GSC, not vendor ETV 50)", got)
	}
	web := labels(payload.WebsiteStats)
	if contains(web, "Est. organic traffic") || contains(web, "DataForSEO Rank") {
		t.Fatalf("vendor tiles mixed into websiteStats: %v", web)
	}

	de := payload.VendorBrief.DomainExplorer
	if !de.Available || de.Stale {
		t.Fatalf("domain_explorer=%+v want available fresh", de)
	}
	if de.OrganicETV != 50 || de.OrganicCount != 3 {
		t.Fatalf("domain_explorer organic=%+v", de)
	}
	if de.Backlinks != 7 || de.ReferringDomains != 2 || de.Rank != 10 {
		t.Fatalf("domain_explorer backlinks=%+v", de)
	}
	ai := payload.VendorBrief.AIVisibility
	if !ai.Available || ai.Mentions != 4 || ai.Citations != 15 {
		t.Fatalf("ai_visibility=%+v", ai)
	}
	bl := payload.VendorBrief.BacklinkExplorer
	if !bl.Available || bl.Backlinks != 99 || bl.ReferringDomains != 12 || bl.Rank != 371 {
		t.Fatalf("backlink_explorer=%+v", bl)
	}
	if payload.VendorBrief.Source != "dataforseo" {
		t.Fatalf("vendor_brief.source=%q", payload.VendorBrief.Source)
	}

	step := loopStep(payload.TodayLoop, "lookup_domain")
	if step == nil || !step.Done {
		t.Fatalf("lookup_domain should be done when domain cache exists: %+v", payload.TodayLoop)
	}
}

func TestSnapshot_StaleVendorBriefStillShowsCachedNumbers(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	fetched := time.Now().UTC().Add(-48 * time.Hour)
	domainPayload, _ := json.Marshal(services.DomainExplorerResult{
		Domain:    "example.com",
		Source:    "dataforseo",
		OrganicOK: true,
		Organic:   services.OrganicOverview{ETV: 80, Count: 9},
		FetchedAt: fetched,
	})
	if err := seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
		Domain:       "example.com",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		Source:       "dataforseo",
		Payload:      string(domainPayload),
		FetchedAt:    fetched,
	}); err != nil {
		t.Fatal(err)
	}

	payload := getSnapshot(t, database, u.ID, p.ID)
	if payload.TrafficSource != "" {
		t.Fatalf("traffic_source=%q want empty (no GSC)", payload.TrafficSource)
	}
	if got := valueFor(payload.WebsiteStats, "Organic Traffic"); got != "—" {
		t.Fatalf("Organic Traffic=%q want — with stale vendor cache and no GSC", got)
	}
	de := payload.VendorBrief.DomainExplorer
	if !de.Available {
		t.Fatal("stale cache must still be available")
	}
	if !de.Stale {
		t.Fatal("fetched 48h ago want stale=true")
	}
	if de.OrganicETV != 80 {
		t.Fatalf("stale organic_etv=%v want 80", de.OrganicETV)
	}
}

func TestSnapshot_TodayLoopFromRealState(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)

	empty := getSnapshot(t, database, u.ID, p.ID)
	assertLoop(t, empty.TodayLoop, "connect_gsc", false)
	assertLoop(t, empty.TodayLoop, "run_audit", false)
	assertLoop(t, empty.TodayLoop, "lookup_domain", false)
	assertLoop(t, empty.TodayLoop, "keyword_labs", false)
	assertLoop(t, empty.TodayLoop, "open_issues", false)
	if empty.VendorBrief.DomainExplorer.Available {
		t.Fatal("no cache want domain_explorer.available=false")
	}

	p.HealthScore = 70
	if err := database.Save(p).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.Metric{
		ProjectID: p.ID, Date: time.Now().Format("2006-01-02"),
		Clicks: 3, Impressions: 30, Source: models.MetricSourceGSC,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.SEOIssue{
		ProjectID: p.ID, URL: p.URL, Severity: models.SeverityHigh, Category: "robots", Detail: "blocked",
	}).Error; err != nil {
		t.Fatal(err)
	}

	mid := getSnapshot(t, database, u.ID, p.ID)
	assertLoop(t, mid.TodayLoop, "connect_gsc", true)
	assertLoop(t, mid.TodayLoop, "run_audit", true)
	assertLoop(t, mid.TodayLoop, "open_issues", false)
	if step := loopStep(mid.TodayLoop, "open_issues"); step == nil || step.Detail == "" {
		t.Fatal("open_issues needs a stored-count detail")
	}

	if err := database.Model(&models.SEOIssue{}).Where("project_id = ?", p.ID).Update("resolved_at", time.Now()).Error; err != nil {
		t.Fatal(err)
	}
	clear := getSnapshot(t, database, u.ID, p.ID)
	assertLoop(t, clear.TodayLoop, "open_issues", true)
}

func TestSnapshot_VendorBriefEmptyJSONNotNil(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	payload := getSnapshot(t, database, u.ID, p.ID)
	if payload.TodayLoop == nil {
		t.Fatal("today_loop must be a slice, not nil")
	}
	if len(payload.TodayLoop) != 5 {
		t.Fatalf("today_loop len=%d want 5", len(payload.TodayLoop))
	}
	if payload.VendorBrief.DomainExplorer.Href != "/seo/site-explorer" {
		t.Fatalf("empty domain_explorer href=%q", payload.VendorBrief.DomainExplorer.Href)
	}
	if payload.VendorBrief.AIVisibility.Href != "/seo/ai-visibility" {
		t.Fatalf("empty ai_visibility href=%q", payload.VendorBrief.AIVisibility.Href)
	}
	if payload.VendorBrief.BacklinkExplorer.Href != "/seo/backlinks" {
		t.Fatalf("empty backlink_explorer href=%q", payload.VendorBrief.BacklinkExplorer.Href)
	}
}

type snapshotPayload struct {
	WebsiteStats  []map[string]any `json:"websiteStats"`
	SocialStats   []map[string]any `json:"socialStats"`
	CombinedStats []map[string]any `json:"combinedStats"`
	TrafficSource string           `json:"traffic_source"`
	TodayLoop     []loopStepJSON   `json:"today_loop"`
	VendorBrief   vendorBriefJSON  `json:"vendor_brief"`
}

type loopStepJSON struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Href   string `json:"href"`
	Done   bool   `json:"done"`
	Detail string `json:"detail"`
}

type vendorBriefJSON struct {
	Source           string           `json:"source"`
	Domain           string           `json:"domain"`
	Message          string           `json:"message"`
	DomainExplorer   vendorLookupJSON `json:"domain_explorer"`
	AIVisibility     vendorLookupJSON `json:"ai_visibility"`
	BacklinkExplorer vendorLookupJSON `json:"backlink_explorer"`
}

type vendorLookupJSON struct {
	Available        bool    `json:"available"`
	Stale            bool    `json:"stale"`
	Href             string  `json:"href"`
	OrganicETV       float64 `json:"organic_etv"`
	OrganicCount     int     `json:"organic_count"`
	Backlinks        int     `json:"backlinks"`
	ReferringDomains int     `json:"referring_domains"`
	Rank             int     `json:"rank"`
	Mentions         int     `json:"mentions"`
	Citations        int     `json:"citations"`
}

func getSnapshot(t *testing.T, database *gorm.DB, userID, projectID uint) snapshotPayload {
	t.Helper()
	h := NewDashboardHandler(
		repository.NewProjectRepository(database),
		repository.NewMetricRepository(database),
		repository.NewInsightRepository(database),
		repository.NewSEORepository(database),
		repository.NewTaskRepository(database),
	)
	r := gin.New()
	r.GET("/dashboard/snapshot", withUser(userID, h.Snapshot))
	req := httptest.NewRequest(http.MethodGet, "/dashboard/snapshot?project_id="+strconv.FormatUint(uint64(projectID), 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var payload snapshotPayload
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &payload); err != nil {
		t.Fatalf("unmarshal snapshot: %v body=%s", err, w.Body.String())
	}
	return payload
}

func loopStep(steps []loopStepJSON, id string) *loopStepJSON {
	for i := range steps {
		if steps[i].ID == id {
			return &steps[i]
		}
	}
	return nil
}

func assertLoop(t *testing.T, steps []loopStepJSON, id string, done bool) {
	t.Helper()
	step := loopStep(steps, id)
	if step == nil {
		t.Fatalf("missing loop step %s in %+v", id, steps)
	}
	if step.Href == "" {
		t.Fatalf("step %s missing href", id)
	}
	if step.Done != done {
		t.Fatalf("step %s done=%v want %v", id, step.Done, done)
	}
}

func labels(stats []map[string]any) []string {
	out := make([]string, 0, len(stats))
	for _, s := range stats {
		if v, ok := s["label"].(string); ok {
			out = append(out, v)
		}
	}
	return out
}

func valueFor(stats []map[string]any, label string) string {
	return fieldFor(stats, label, "value")
}

func changeFor(stats []map[string]any, label string) string {
	return fieldFor(stats, label, "change")
}

func fieldFor(stats []map[string]any, label, field string) string {
	for _, s := range stats {
		if s["label"] == label {
			if v, ok := s[field].(string); ok {
				return v
			}
		}
	}
	return ""
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func containsAll(list []string, want ...string) bool {
	for _, w := range want {
		if !contains(list, w) {
			return false
		}
	}
	return true
}
