package siteaudit

import (
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/services"
)

func TestBuildReportsEmptyDoesNotInventScores(t *testing.T) {
	rep := BuildReports(ForPlan(models.PlanCodeFree, false), Snapshot{}, nil)
	if rep.Overview.SiteHealth.Value != nil || rep.Overview.PagesCrawled != nil {
		t.Fatalf("empty snapshot invented health/pages %+v", rep.Overview)
	}
	if rep.Overview.Thematic.AISearchHealth.Status != "not_implemented" {
		t.Fatalf("AI Search Health must be not_implemented %+v", rep.Overview.Thematic.AISearchHealth)
	}
	if rep.Overview.Thematic.CoreWebVitals.Status != "not_implemented" {
		t.Fatalf("CWV must be not_implemented %+v", rep.Overview.Thematic.CoreWebVitals)
	}
	if !rep.Compare.Locked {
		t.Fatal("free plan must lock Compare Crawls")
	}
	if len(rep.Issues) != 0 || len(rep.Pages) != 0 {
		t.Fatalf("empty issues/pages %+v", rep)
	}
}

func TestFromCrawlUsesRealPageCount(t *testing.T) {
	now := time.Now().UTC()
	snap := FromCrawl(&services.SiteCrawlResult{
		SeedURL:   "https://example.com/",
		CrawledAt: now,
		Pages: []services.SitePageResult{
			{
				URL:        "https://example.com/",
				StatusCode: 200,
				Title:      "Home",
				Score:      80,
				Indexable:  true,
				Checks: []services.AuditCheck{
					{Category: "https", Label: "HTTPS", Status: services.CheckPass, Severity: "high"},
					{Category: "https", Label: "HSTS", Status: services.CheckFail, Severity: "high", Detail: "missing"},
					{Category: "meta", Label: "Title", Status: services.CheckWarning, Severity: "medium", Detail: "short"},
				},
				InternalLinks: []string{"https://example.com/about"},
			},
			{
				URL:        "https://example.com/about",
				StatusCode: 200,
				Title:      "About",
				Score:      90,
				Indexable:  true,
				Checks: []services.AuditCheck{
					{Category: "https", Label: "HTTPS", Status: services.CheckPass, Severity: "high"},
				},
			},
		},
		StoppedReason: "complete",
	}, 100, "website", 100)
	if snap.PagesCrawled != 2 {
		t.Fatalf("pages=%d want 2", snap.PagesCrawled)
	}
	if snap.SiteHealth == nil {
		t.Fatal("health should be computed from checks")
	}
	if snap.Thematic.AISearchHealth.Status != "not_implemented" || snap.Thematic.CoreWebVitals.Status != "not_implemented" {
		t.Fatal("AI/CWV must stay not_implemented")
	}
	if snap.Thematic.HTTPS.Value == nil || *snap.Thematic.HTTPS.Value != 66 {
		t.Fatalf("https thematic=%v want 66 (2/3)", snap.Thematic.HTTPS.Value)
	}
	if snap.Thematic.Linking.Value == nil {
		t.Fatal("linking should compute with 2 pages")
	}
	if snap.Errors != 1 || snap.Warnings != 1 {
		t.Fatalf("errors=%d warnings=%d", snap.Errors, snap.Warnings)
	}
}

func TestListRowEmptyLeavesEmDashCells(t *testing.T) {
	row := ListRowFrom(1, "Acme", "example.com", 100, Snapshot{}, nil)
	if row.PagesCrawled != nil || row.LastUpdate != nil || row.SiteHealth.Value != nil || row.Errors.Value != nil {
		t.Fatalf("empty row invented values %+v", row)
	}
	if row.AISearchHealth.Status != "not_implemented" || row.CoreWebVitals.Status != "not_implemented" {
		t.Fatalf("AI/CWV %+v %+v", row.AISearchHealth, row.CoreWebVitals)
	}
}

func TestProPlusUnlocksCompare(t *testing.T) {
	now := time.Now().UTC()
	snap := Snapshot{CrawledAt: &now, PagesCrawled: 1, Pages: []PageRow{{URL: "https://example.com/"}}}
	rep := BuildReports(ForPlan(models.PlanCodeProYearly, false), snap, nil)
	if rep.Compare.Locked {
		t.Fatal("Pro+ should unlock Compare Crawls")
	}
}
