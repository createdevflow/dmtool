package handlers

import (
	"testing"

	"backend/internal/models"
)

func TestGSCContentIdeasSkipsZeroClickWhenClicksUnknown(t *testing.T) {
	kws := []models.KeywordResult{
		{Keyword: "how to rank", Volume: 100, Clicks: 0},
		{Keyword: "buy shoes", Volume: 50, Clicks: 0},
	}
	ideas := gscContentIdeas(kws)
	if len(ideas) != 1 {
		t.Fatalf("len=%d want 1 (no zero_click when all clicks are 0)", len(ideas))
	}
	if ideas[0]["type"] != "question" {
		t.Fatalf("type=%v want question", ideas[0]["type"])
	}
}

func TestGSCContentIdeasZeroClickWhenClicksKnown(t *testing.T) {
	kws := []models.KeywordResult{
		{Keyword: "brand", Volume: 80, Clicks: 5},
		{Keyword: "zero click query", Volume: 40, Clicks: 0},
	}
	ideas := gscContentIdeas(kws)
	if len(ideas) != 1 {
		t.Fatalf("len=%d want 1", len(ideas))
	}
	if ideas[0]["type"] != "zero_click" {
		t.Fatalf("type=%v want zero_click", ideas[0]["type"])
	}
}

func TestIdeasForKeywordSourceAutocompleteEmpty(t *testing.T) {
	kws := []models.KeywordResult{
		{Keyword: "how to rank", Volume: 0, Clicks: 0},
	}
	if got := ideasForKeywordSource(kws, "autocomplete"); len(got) != 0 {
		t.Fatalf("autocomplete ideas=%v want empty", got)
	}
	if got := ideasForKeywordSource(kws, "cache"); len(got) != 0 {
		t.Fatalf("autocomplete cache ideas=%v want empty", got)
	}
}

func TestIdeasForKeywordSourceGSC(t *testing.T) {
	kws := []models.KeywordResult{
		{Keyword: "how to set up gsc", Volume: 20, Clicks: 1, Position: 4},
	}
	if got := ideasForKeywordSource(kws, "gsc"); len(got) != 1 {
		t.Fatalf("gsc ideas=%d want 1", len(got))
	}
	if got := ideasForKeywordSource(kws, "cache"); len(got) != 1 {
		t.Fatalf("gsc cache ideas=%d want 1", len(got))
	}
}

func TestNormalizePipelineStatus(t *testing.T) {
	if got := normalizePipelineStatus(""); got != "idea" {
		t.Fatalf("empty=%s want idea", got)
	}
	if got := normalizePipelineStatus("REVIEW"); got != "review" {
		t.Fatalf("REVIEW=%s want review", got)
	}
	if got := normalizePipelineStatus("nope"); got != "idea" {
		t.Fatalf("invalid=%s want idea", got)
	}
}

func TestKeywordPublicListIntent(t *testing.T) {
	out := keywordPublicList([]models.KeywordResult{
		{Keyword: "buy running shoes", Volume: 12, Clicks: 3, KD: 0, Position: 2},
	})
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0]["intent"] != "transactional" {
		t.Fatalf("intent=%v", out[0]["intent"])
	}
	if out[0]["impressions"] != 12 {
		t.Fatalf("impressions=%v want GSC volume copy", out[0]["impressions"])
	}
	if out[0]["kd"] != 0 {
		t.Fatalf("kd=%v want 0", out[0]["kd"])
	}
}
