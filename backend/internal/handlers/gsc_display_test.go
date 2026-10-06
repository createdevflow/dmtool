package handlers

import (
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
)

func TestTopBreakdown(t *testing.T) {
	if topBreakdown(nil) != nil {
		t.Fatal("empty want nil")
	}
	rows := []models.GSCBreakdown{
		{Key: "b", Clicks: 3, Impressions: 100},
		{Key: "a", Clicks: 10, Impressions: 10},
		{Key: "c", Clicks: 10, Impressions: 50},
	}
	top := topBreakdown(rows)
	if top == nil || top.Key != "c" {
		t.Fatalf("tie on clicks should prefer more impressions, got %+v", top)
	}
}

func TestMobileClickShare(t *testing.T) {
	_, ok := mobileClickShare(nil)
	if ok {
		t.Fatal("empty should not be ok")
	}
	pct, ok := mobileClickShare([]models.GSCBreakdown{
		{Key: "MOBILE", Clicks: 3, Impressions: 10},
		{Key: "DESKTOP", Clicks: 1, Impressions: 10},
	})
	if !ok || pct != 75 {
		t.Fatalf("pct=%v ok=%v want 75", pct, ok)
	}
	pct, ok = mobileClickShare([]models.GSCBreakdown{
		{Key: "MOBILE", Clicks: 0, Impressions: 80},
		{Key: "DESKTOP", Clicks: 0, Impressions: 20},
	})
	if !ok || pct != 80 {
		t.Fatalf("impressions fallback pct=%v ok=%v", pct, ok)
	}
}

func TestDisplayKeys(t *testing.T) {
	if got := displayPageKey("https://example.com/blog/post"); got != "/blog/post" {
		t.Fatalf("page=%q", got)
	}
	if got := displayPageKey("https://example.com/"); got != "example.com/" {
		t.Fatalf("home=%q", got)
	}
	if got := displayCountryKey("usa"); got != "United States" {
		t.Fatalf("usa=%q", got)
	}
	if got := displayCountryKey("zzz"); got != "ZZZ" {
		t.Fatalf("unknown=%q", got)
	}
	if got := displayDeviceKey("MOBILE"); got != "Mobile" {
		t.Fatalf("device=%q", got)
	}
	if got := displayAppearanceKey("RICH_RESULT"); got != "RICH RESULT" {
		t.Fatalf("appearance=%q", got)
	}
}

func TestGSCStatEmptyStates(t *testing.T) {
	page := gscTopPageStat(nil, false)
	if page["value"] != "—" || page["change"] != "Connect GSC" {
		t.Fatalf("disconnected page=%v", page)
	}
	page = gscTopPageStat(nil, true)
	if page["value"] != "—" || page["change"] != "Sync GSC" {
		t.Fatalf("connected empty page=%v", page)
	}
	country := gscTopCountryStat(nil, false)
	if country["value"] != "—" || country["change"] != "Connect GSC" {
		t.Fatalf("disconnected country=%v", country)
	}
	mobile := gscMobileShareStat(nil, true)
	if mobile["value"] != "—" || mobile["change"] != "Sync GSC" {
		t.Fatalf("connected empty mobile=%v", mobile)
	}
}

func TestPersistGSCBreakdownsReplaceAndPartialClear(t *testing.T) {
	database := honestyDB(t)
	_, p := honestyUserProject(t, database)
	repo := repository.NewSEORepository(database)

	set := services.GSCBreakdownSet{
		GotPages: true,
		Pages: []models.GSCBreakdown{
			{Key: "https://example.com/a", Clicks: 4, Impressions: 10, CTR: 0.4, Position: 2, StartDate: "2026-09-01", EndDate: "2026-10-01"},
		},
		GotDevices: true,
		Devices: []models.GSCBreakdown{
			{Key: "MOBILE", Clicks: 3, Impressions: 9},
		},
	}
	if n := persistGSCBreakdowns(repo, p.ID, set); n != 2 {
		t.Fatalf("n=%d want 2", n)
	}

	pages, err := repo.FindBreakdowns(p.ID, models.GSCDimPage)
	if err != nil || len(pages) != 1 || pages[0].Key != "https://example.com/a" || pages[0].ProjectID != p.ID {
		t.Fatalf("pages=%+v err=%v", pages, err)
	}

	// Successful empty page fetch clears pages; devices stay (that dim was not in this batch).
	cleared := persistGSCBreakdowns(repo, p.ID, services.GSCBreakdownSet{GotPages: true})
	if cleared != 0 {
		t.Fatalf("cleared n=%d want 0", cleared)
	}
	pages, _ = repo.FindBreakdowns(p.ID, models.GSCDimPage)
	if len(pages) != 0 {
		t.Fatalf("pages after empty replace=%d want 0", len(pages))
	}
	devices, _ := repo.FindBreakdowns(p.ID, models.GSCDimDevice)
	if len(devices) != 1 {
		t.Fatalf("devices should remain after page-only clear, got %d", len(devices))
	}

	if n := persistGSCBreakdowns(nil, p.ID, set); n != 0 {
		t.Fatalf("nil repo n=%d", n)
	}
}
