package services

import (
	"testing"
	"time"

	"google.golang.org/api/searchconsole/v1"
)

func TestBreakdownFromAPIRows(t *testing.T) {
	fetched := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	rows := []*searchconsole.ApiDataRow{
		{Keys: []string{"https://example.com/blog"}, Clicks: 12, Impressions: 100, Ctr: 0.12, Position: 4.2},
		{Keys: []string{""}, Clicks: 9},
		nil,
		{Keys: []string{"https://example.com/"}, Clicks: 40, Impressions: 200, Ctr: 0.2, Position: 2},
	}
	got := breakdownFromAPIRows("page", "2026-09-03", "2026-10-03", rows, fetched)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2 (empty key skipped)", len(got))
	}
	if got[0].Key != "https://example.com/blog" || got[0].Clicks != 12 {
		t.Fatalf("first=%+v", got[0])
	}
	if got[1].Clicks != 40 || got[1].CTR != 0.2 {
		t.Fatalf("second=%+v", got[1])
	}
	if got[0].StartDate != "2026-09-03" || got[0].Dimension != "page" {
		t.Fatalf("meta=%+v", got[0])
	}
}

func TestGSCBreakdownSetFetchedBatches(t *testing.T) {
	set := GSCBreakdownSet{GotPages: true, GotDevices: true}
	b := set.FetchedBatches()
	if len(b) != 2 {
		t.Fatalf("batches=%d want 2", len(b))
	}
	empty := GSCBreakdownSet{}
	if n := len(empty.FetchedBatches()); n != 0 {
		t.Fatalf("empty batches=%d", n)
	}
}
