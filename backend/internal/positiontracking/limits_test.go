package positiontracking

import (
	"testing"

	"backend/internal/models"
)

func TestForPlanMatchesSemrushCaps(t *testing.T) {
	free := ForPlan(models.PlanCodeFree, false)
	if free.PlanLabel != "Free" || free.Keywords != 10 || free.Websites != 1 || free.Tags || free.Cannibalization || free.Multitargeting || free.ShareOfVoice {
		t.Fatalf("free limits %+v", free)
	}
	seo := ForPlan(models.PlanCodeProMonthly, false)
	if seo.PlanLabel != "SEO" || seo.Keywords != 500 || seo.Websites != 5 || seo.Tags || seo.Multitargeting {
		t.Fatalf("seo limits %+v", seo)
	}
	guru := ForPlan(models.PlanCodeProYearly, false)
	if guru.PlanLabel != "Pro+" || guru.Keywords != 1500 || guru.Websites != 15 || !guru.Tags || !guru.Cannibalization || !guru.Multitargeting || guru.MaxTargets != 10 || guru.ShareOfVoice {
		t.Fatalf("guru limits %+v", guru)
	}
	adv := ForPlan(models.PlanCodeFree, true)
	if adv.PlanLabel != "Advanced" || adv.Keywords != 5000 || adv.Websites != 40 || !adv.ShareOfVoice || !adv.ScheduledExport {
		t.Fatalf("admin/advanced limits %+v", adv)
	}
}

func TestKeywordUnits(t *testing.T) {
	if KeywordUnits(10, 3, false) != 10 {
		t.Fatal("non-multitarget must count keywords once")
	}
	if KeywordUnits(10, 3, true) != 30 {
		t.Fatal("multitarget counts keywords × targets")
	}
}

func TestApplyPlanStripsTagsAndTargetsOnFree(t *testing.T) {
	kws, comps, targets := ApplyPlan(ForPlan(models.PlanCodeFree, false),
		[]models.PTKeyword{{Keyword: "Coffee", Tags: []string{"bev"}}, {Keyword: "coffee"}, {Keyword: " "}},
		[]string{"rival.com", "https://www.rival.com/x", "other.com"},
		[]models.PTTarget{{Device: "mobile", Engine: "bing"}},
	)
	if len(kws) != 1 || kws[0].Keyword != "coffee" || len(kws[0].Tags) != 0 {
		t.Fatalf("keywords %+v", kws)
	}
	if len(comps) != 2 {
		t.Fatalf("competitors %+v", comps)
	}
	if len(targets) != 0 {
		t.Fatalf("targets %+v want stripped", targets)
	}
}

func TestBuildReportsEmptyHasNilVisibility(t *testing.T) {
	rep := BuildReports(ForPlan(models.PlanCodeFree, false), Snapshot{}, "example.com", nil, nil)
	if rep.Overview.Visibility != nil || rep.Overview.AvgPosition != nil || rep.Overview.ETV != nil {
		t.Fatalf("empty snapshot must not invent scores: %+v", rep.Overview)
	}
	if rep.Overview.Ranked != 0 || len(rep.Rankings) != 0 {
		t.Fatalf("empty rankings %+v", rep)
	}
	if !rep.Tags.Locked || !rep.Cannibalization.Locked || !rep.AISearch.Locked {
		t.Fatal("free plan must lock tags, cannibalization, AI search")
	}
}

func TestBuildReportsVisibilityFromRanks(t *testing.T) {
	rep := BuildReports(ForPlan(models.PlanCodeProYearly, false), Snapshot{
		Source: "dataforseo_serp",
		Keywords: []KeywordRow{
			{Keyword: "a", Position: 1, URL: "https://example.com/a", OK: true, Source: "dataforseo_serp", Cannibalizing: []Rank{{URL: "https://example.com/a2", Position: 8}}},
			{Keyword: "b", Position: 0, OK: true, Source: "dataforseo_serp"},
		},
	}, "example.com", []string{"rival.com"}, nil)
	if rep.Overview.Visibility == nil || *rep.Overview.Visibility != 99 {
		t.Fatalf("visibility=%v want 99", rep.Overview.Visibility)
	}
	if rep.Tags.Locked {
		t.Fatal("Pro+ must unlock tags")
	}
	if len(rep.Cannibalization.Rows) != 1 {
		t.Fatalf("want 1 cannibal row, got %+v", rep.Cannibalization.Rows)
	}
}
