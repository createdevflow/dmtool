package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

func TestAIVisibilityGapUnconfigured(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/ai-visibility-gap", withUser(u.ID, h.AIVisibilityGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data struct {
		Available  bool `json:"available"`
		Configured bool `json:"configured"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if data.Available || data.Configured {
		t.Fatalf("want unconfigured empty, got %+v", data)
	}
}

func TestAIVisibilityGapCacheMissDoesNotFetch(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/ai-visibility-gap", withUser(u.ID, h.AIVisibilityGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available  bool `json:"available"`
		Configured bool `json:"configured"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if data.Available || !data.Configured {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("GET without fetch must not call vendor, calls=%d", fake.calls)
	}
}

func TestAIVisibilityGapEmptyCompetitorDoesNotFetch(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/ai-visibility-gap", withUser(u.ID, h.AIVisibilityGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility-gap?project_id="+itoa(p.ID)+"&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available  bool `json:"available"`
		Configured bool `json:"configured"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if data.Available || !data.Configured {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("empty competitor must not call vendor, calls=%d", fake.calls)
	}
}

func TestAIVisibilityGapSameDomainDoesNotFetch(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/ai-visibility-gap", withUser(u.ID, h.AIVisibilityGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility-gap?project_id="+itoa(p.ID)+"&competitor=https://www.example.com&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available bool `json:"available"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if data.Available {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("same-domain must not call vendor, calls=%d", fake.calls)
	}
}

func TestAIVisibilityGapServesCacheWithoutVendorCall(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.AIVisibilityGapResult{
		Target:       "example.com",
		Competitor:   "rival.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		TargetOK:     true,
		TargetMentions: services.LLMMetrics{
			Mentions: 18,
			Google:   services.LLMPlatformMetrics{Mentions: 14},
		},
		FetchedAt: time.Now().UTC(),
	})
	if err := seoRepo.SaveVendorAIVisibilityGapSnapshot(&models.VendorAIVisibilityGapSnapshot{
		Target:       "example.com",
		Competitor:   "rival.com",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		Source:       "dataforseo",
		Payload:      string(payload),
		FetchedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	fake := &configuredDFS{}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		seoRepo,
		repository.NewOAuthRepository(database),
		fake,
		nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/ai-visibility-gap", withUser(u.ID, h.AIVisibilityGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	raw := decodeData(t, w.Body.Bytes())
	var data struct {
		Available      bool `json:"available"`
		Cached         bool `json:"cached"`
		TargetMentions struct {
			Mentions int `json:"mentions"`
		} `json:"target_mentions"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || !data.Cached || data.TargetMentions.Mentions != 18 {
		t.Fatalf("got %+v", data)
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, ok := envelope["score"]; ok {
		t.Fatal("must not invent an AI visibility score")
	}
	if fake.calls != 0 {
		t.Fatalf("vendor calls=%d want 0 on fresh cache", fake.calls)
	}
}

func TestAIVisibilityGapFetchCallsVendor(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{
		visGapResult: services.AIVisibilityGapResult{
			Target:     "example.com",
			Competitor: "rival.com",
			Source:     "dataforseo",
			TargetOK:   true,
			TargetMentions: services.LLMMetrics{
				Mentions: 11,
			},
			FetchedAt: time.Now().UTC(),
		},
	}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/ai-visibility-gap", withUser(u.ID, h.AIVisibilityGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility-gap?project_id="+itoa(p.ID)+"&competitor=rival.com&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	raw := decodeData(t, w.Body.Bytes())
	var data struct {
		Available      bool `json:"available"`
		Cached         bool `json:"cached"`
		TargetMentions struct {
			Mentions int `json:"mentions"`
		} `json:"target_mentions"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || data.Cached || data.TargetMentions.Mentions != 11 {
		t.Fatalf("got %+v", data)
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, ok := envelope["score"]; ok {
		t.Fatal("must not invent an AI visibility score")
	}
	if fake.calls != 1 {
		t.Fatalf("vendor calls=%d want 1", fake.calls)
	}
}
