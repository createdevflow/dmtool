package siteaudit

import (
	"testing"

	"backend/internal/models"
)

func TestForPlanMatchesSemrushCaps(t *testing.T) {
	free := ForPlan(models.PlanCodeFree, false)
	if free.PlanLabel != "Free" || free.PagesPerAudit != 100 || free.Websites != 1 || free.JSRendering || free.Schedule || free.CompareCrawls || free.AISearchHealth {
		t.Fatalf("free limits %+v", free)
	}
	if free.RunPageCap != RunPageCap || RunPageCap != 8 {
		t.Fatalf("run cap %d", free.RunPageCap)
	}
	seo := ForPlan(models.PlanCodeProMonthly, false)
	if seo.PlanLabel != "SEO" || seo.PagesPerAudit != 20000 || seo.PagesPerMonth != 100000 || seo.Websites != 5 || seo.JSRendering || !seo.Schedule {
		t.Fatalf("seo limits %+v", seo)
	}
	guru := ForPlan(models.PlanCodeProYearly, false)
	if guru.PlanLabel != "Pro+" || guru.PagesPerAudit != 20000 || guru.PagesPerMonth != 300000 || guru.Websites != 15 || !guru.JSRendering || !guru.CompareCrawls {
		t.Fatalf("guru limits %+v", guru)
	}
	adv := ForPlan(models.PlanCodeFree, true)
	if adv.PlanLabel != "Advanced" || adv.PagesPerAudit != 100000 || adv.PagesPerMonth != 1000000 || adv.Websites != 40 || !adv.JSRendering {
		t.Fatalf("admin/advanced limits %+v", adv)
	}
}

func TestClampPageLimit(t *testing.T) {
	free := ForPlan(models.PlanCodeFree, false)
	if ClampPageLimit(0, free) != RunPageCap {
		t.Fatalf("default 0 should use run cap, got %d", ClampPageLimit(0, free))
	}
	if ClampPageLimit(500, free) != 100 {
		t.Fatalf("free must clamp to 100, got %d", ClampPageLimit(500, free))
	}
	if ClampPageLimit(5, free) != 5 {
		t.Fatalf("under cap stays, got %d", ClampPageLimit(5, free))
	}
}
