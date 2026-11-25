package domainoverview

import (
	"testing"

	"backend/internal/models"
	"backend/internal/services"
)

func TestBuildReportsEmptyDoesNotInventScores(t *testing.T) {
	limits := ForPlan(models.PlanCodeFree, false)
	got := BuildReports(limits, services.DomainExplorerResult{}, false)
	if got.Overview.AIVisibility.Status != "not_implemented" {
		t.Fatalf("ai=%+v", got.Overview.AIVisibility)
	}
	if got.Overview.AuthorityScore.Status != "not_implemented" || got.Overview.AuthorityScore.Value != nil {
		t.Fatalf("authority invented %+v", got.Overview.AuthorityScore)
	}
	if got.Overview.OrganicTraffic.Status != "empty" || got.Overview.OrganicTraffic.Value != nil {
		t.Fatalf("organic %+v", got.Overview.OrganicTraffic)
	}
	if got.Growth.Locked != true {
		t.Fatalf("free growth should be locked %+v", got.Growth)
	}
	if got.CompareCountries.Locked != true {
		t.Fatalf("free countries should be locked %+v", got.CompareCountries)
	}
	if len(got.TopKeywords) != 0 || len(got.Competitors) != 0 {
		t.Fatalf("invented rows keywords=%d competitors=%d", len(got.TopKeywords), len(got.Competitors))
	}
}

func TestBuildReportsUsesVendorPayload(t *testing.T) {
	limits := ForPlan(models.PlanCodeProYearly, false)
	etv := 1234.5
	got := BuildReports(limits, services.DomainExplorerResult{
		Domain:        "example.com",
		LocationName:  "United States",
		OrganicOK:     true,
		Organic:       services.OrganicOverview{ETV: etv, Count: 40, Pos1: 2, Pos2To3: 5, Pos4To10: 10, PaidETV: 9, PaidCount: 3},
		BacklinksOK:   true,
		Backlinks:     services.BacklinkOverview{Backlinks: 99, ReferringDomains: 12, Rank: 371, Follow: 80, Nofollow: 19},
		CompetitorsOK: true,
		Competitors: []services.CompetitorDomain{
			{Domain: "rival.com", Intersections: 12, OrganicETV: 800, OrganicCount: 20},
		},
		KeywordsOK: true,
		Keywords: []services.RankedKeyword{
			{Keyword: "buy widgets", Volume: 2200, CPC: 1.5, KD: 44, Position: 3, Intent: "transactional"},
		},
	}, true)
	if got.Overview.OrganicTraffic.Value == nil || *got.Overview.OrganicTraffic.Value != etv {
		t.Fatalf("traffic %+v", got.Overview.OrganicTraffic)
	}
	if got.Overview.AIVisibility.Status != "not_implemented" {
		t.Fatalf("AI visibility must stay not implemented")
	}
	if got.Overview.AuthorityScore.Status != "not_implemented" {
		t.Fatalf("authority must stay not implemented")
	}
	if got.Overview.DFSRank.Value == nil || *got.Overview.DFSRank.Value != 371 {
		t.Fatalf("dfs rank %+v", got.Overview.DFSRank)
	}
	if got.Growth.Locked {
		t.Fatalf("Pro+ growth should unlock the tab, still not implemented: %+v", got.Growth)
	}
	if got.Growth.Status != "not_implemented" {
		t.Fatalf("growth still has no history %+v", got.Growth)
	}
	if len(got.TopKeywords) != 1 || got.TopKeywords[0].Keyword != "buy widgets" {
		t.Fatalf("keywords %+v", got.TopKeywords)
	}
	if got.Overview.TrafficShare.Value == nil {
		t.Fatalf("traffic share empty")
	}
}

func TestForPlanFreeTenRequests(t *testing.T) {
	if ForPlan(models.PlanCodeFree, false).RequestsPerDay != 10 {
		t.Fatal("free cap")
	}
	if ForPlan(models.PlanCodeProMonthly, false).RequestsPerDay != 3000 {
		t.Fatal("seo cap")
	}
	if ForPlan(models.PlanCodeProYearly, false).RequestsPerDay != 5000 {
		t.Fatal("pro+ cap")
	}
	if ForPlan(models.PlanCodeFree, true).RequestsPerDay != 10000 {
		t.Fatal("admin cap")
	}
}
