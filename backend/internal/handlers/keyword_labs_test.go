package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

func TestKeywordsGetDoesNotCallLabs(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	if err := seoRepo.UpsertKeywords([]models.KeywordResult{
		{ProjectID: p.ID, Seed: "widgets", Keyword: "buy widgets", Volume: 0, KD: 0},
	}); err != nil {
		t.Fatal(err)
	}
	fake := &configuredDFS{
		labs: map[string]services.KeywordLabsMetrics{
			"buy widgets": {SearchVolume: 2200, CPC: 1.5, KD: 44},
		},
	}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		seoRepo,
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/keywords", withUser(u.ID, h.Keywords))
	req := httptest.NewRequest(http.MethodGet, "/seo/keywords?project_id="+itoa(p.ID)+"&seed=widgets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if fake.overviewCalls != 0 || fake.relatedCalls != 0 {
		t.Fatalf("GET must not call Labs, overview=%d related=%d", fake.overviewCalls, fake.relatedCalls)
	}
	var data struct {
		LabsConfigured bool `json:"labs_configured"`
		Keywords       []struct {
			SearchVolume int `json:"search_volume"`
			KD           int `json:"kd"`
			Impressions  int `json:"impressions"`
		} `json:"keywords"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.LabsConfigured {
		t.Fatal("expected labs_configured")
	}
	if len(data.Keywords) != 1 || data.Keywords[0].SearchVolume != 0 || data.Keywords[0].KD != 0 {
		t.Fatalf("GET must not invent Labs volume, got %+v", data.Keywords)
	}
}

func TestKeywordsPostEnrichesLabs(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	if err := seoRepo.UpsertKeywords([]models.KeywordResult{
		{ProjectID: p.ID, Seed: "widgets", Keyword: "buy widgets", Volume: 12, Clicks: 3, KD: 0},
	}); err != nil {
		t.Fatal(err)
	}
	trend := 12
	fake := &configuredDFS{
		labs: map[string]services.KeywordLabsMetrics{
			"buy widgets": {
				SearchVolume:     2200,
				CPC:              1.5,
				KD:               44,
				CompetitionLevel: "MEDIUM",
				LabsIntent:       "commercial",
				SerpFeatures:     []string{"featured_snippet"},
				TrendMonthly:     &trend,
			},
		},
		relatedResult: services.RelatedKeywordsResult{
			Seed:      "widgets",
			Source:    "dataforseo",
			RelatedOK: true,
			Related: []services.RelatedKeyword{
				{Keyword: "best widgets", SearchVolume: 880, KD: 21},
			},
		},
	}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		seoRepo,
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.POST("/seo/keywords", withUser(u.ID, h.KeywordsPost))
	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "seed": "widgets"})
	req := httptest.NewRequest(http.MethodPost, "/seo/keywords", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if fake.overviewCalls != 1 {
		t.Fatalf("POST Generate should call Labs once, calls=%d", fake.overviewCalls)
	}
	var data struct {
		Keywords []struct {
			SearchVolume     int      `json:"search_volume"`
			CPC              float64  `json:"cpc"`
			KD               int      `json:"kd"`
			Impressions      int      `json:"impressions"`
			LabsEnriched     bool     `json:"labs_enriched"`
			CompetitionLevel string   `json:"competition_level"`
			LabsIntent       string   `json:"labs_intent"`
			SerpFeatures     []string `json:"serp_features"`
			TrendMonthly     *int     `json:"trend_monthly"`
		} `json:"keywords"`
		Related []struct {
			Keyword      string `json:"keyword"`
			SearchVolume int    `json:"search_volume"`
		} `json:"related"`
		RelatedOK bool `json:"related_ok"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Keywords) != 1 {
		t.Fatalf("len=%d", len(data.Keywords))
	}
	row := data.Keywords[0]
	if row.SearchVolume != 2200 || row.CPC != 1.5 || row.KD != 44 || !row.LabsEnriched {
		t.Fatalf("got %+v", row)
	}
	if row.CompetitionLevel != "MEDIUM" || row.LabsIntent != "commercial" || len(row.SerpFeatures) != 1 || row.TrendMonthly == nil || *row.TrendMonthly != 12 {
		t.Fatalf("extras=%+v", row)
	}
	if row.Impressions != 12 {
		t.Fatalf("impressions=%d must stay GSC, not Labs volume", row.Impressions)
	}
	if !data.RelatedOK || len(data.Related) != 1 || data.Related[0].Keyword != "best widgets" {
		t.Fatalf("related=%+v ok=%v", data.Related, data.RelatedOK)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/seo/keywords", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if fake.overviewCalls != 1 {
		t.Fatalf("second POST within cache must not recall Labs, calls=%d", fake.overviewCalls)
	}
	if fake.relatedCalls != 1 {
		t.Fatalf("second POST within cache must not recall related, calls=%d", fake.relatedCalls)
	}

	r.GET("/seo/keywords", withUser(u.ID, h.Keywords))
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/seo/keywords?project_id="+itoa(p.ID)+"&seed=widgets", nil)
	r.ServeHTTP(w3, req3)
	var cached struct {
		Keywords []struct {
			CompetitionLevel string   `json:"competition_level"`
			SerpFeatures     []string `json:"serp_features"`
		} `json:"keywords"`
		Related []struct {
			Keyword string `json:"keyword"`
		} `json:"related"`
	}
	if err := json.Unmarshal(decodeData(t, w3.Body.Bytes()), &cached); err != nil {
		t.Fatal(err)
	}
	if len(cached.Keywords) != 1 || cached.Keywords[0].CompetitionLevel != "MEDIUM" || len(cached.Keywords[0].SerpFeatures) != 1 {
		t.Fatalf("GET extras=%+v", cached.Keywords)
	}
	if len(cached.Related) != 1 || cached.Related[0].Keyword != "best widgets" {
		t.Fatalf("GET related=%+v", cached.Related)
	}
	if fake.overviewCalls != 1 || fake.relatedCalls != 1 {
		t.Fatalf("GET must not call vendor, overview=%d related=%d", fake.overviewCalls, fake.relatedCalls)
	}
}

func TestKeywordsPostUnconfiguredDoesNotCallLabs(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	if err := seoRepo.UpsertKeywords([]models.KeywordResult{
		{ProjectID: p.ID, Seed: "widgets", Keyword: "buy widgets", Volume: 0, KD: 0},
	}); err != nil {
		t.Fatal(err)
	}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		seoRepo,
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)
	r := gin.New()
	r.POST("/seo/keywords", withUser(u.ID, h.KeywordsPost))
	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "seed": "widgets"})
	req := httptest.NewRequest(http.MethodPost, "/seo/keywords", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		LabsConfigured bool `json:"labs_configured"`
		Keywords       []struct {
			SearchVolume int  `json:"search_volume"`
			KD           int  `json:"kd"`
			LabsEnriched bool `json:"labs_enriched"`
		} `json:"keywords"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if data.LabsConfigured {
		t.Fatal("nil vendor must not report configured")
	}
	if data.Keywords[0].SearchVolume != 0 || data.Keywords[0].KD != 0 || data.Keywords[0].LabsEnriched {
		t.Fatalf("unconfigured must stay empty, got %+v", data.Keywords[0])
	}
}
