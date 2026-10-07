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
	if fake.overviewCalls != 0 {
		t.Fatalf("GET must not call Labs, calls=%d", fake.overviewCalls)
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
			SearchVolume int     `json:"search_volume"`
			CPC          float64 `json:"cpc"`
			KD           int     `json:"kd"`
			Impressions  int     `json:"impressions"`
			LabsEnriched bool    `json:"labs_enriched"`
		} `json:"keywords"`
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
	if row.Impressions != 12 {
		t.Fatalf("impressions=%d must stay GSC, not Labs volume", row.Impressions)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/seo/keywords", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if fake.overviewCalls != 1 {
		t.Fatalf("second POST within cache must not recall Labs, calls=%d", fake.overviewCalls)
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
