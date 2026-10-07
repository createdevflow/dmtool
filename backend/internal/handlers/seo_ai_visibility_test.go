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

func TestAIVisibilityUnconfigured(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/ai-visibility", withUser(u.ID, h.AIVisibility))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility?project_id="+itoa(p.ID)+"&domain=rival.com", nil)
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

func TestAIVisibilityCacheMissDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/ai-visibility", withUser(u.ID, h.AIVisibility))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility?project_id="+itoa(p.ID)+"&domain=rival.com", nil)
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

func TestAIVisibilityServesCacheWithoutVendorCall(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.AIVisibilityResult{
		Domain:       "example.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		CitationsOK:  true,
		Citations: services.LLMMetrics{
			Mentions:       15,
			AISearchVolume: 490,
			Google:         services.LLMPlatformMetrics{Mentions: 12, AISearchVolume: 400},
			ChatGPT:        services.LLMPlatformMetrics{Mentions: 3, AISearchVolume: 90},
		},
		FetchedAt: time.Now().UTC(),
	})
	if err := seoRepo.SaveVendorAIVisibilitySnapshot(&models.VendorAIVisibilitySnapshot{
		Domain:       "example.com",
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
	r.GET("/seo/ai-visibility", withUser(u.ID, h.AIVisibility))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility?project_id="+itoa(p.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data struct {
		Available bool `json:"available"`
		Cached    bool `json:"cached"`
		Citations struct {
			Mentions int `json:"mentions"`
		} `json:"citations"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || !data.Cached || data.Citations.Mentions != 15 {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("vendor calls=%d want 0 on fresh cache", fake.calls)
	}
}

func TestAIVisibilityFetchCallsVendor(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{
		aiResult: services.AIVisibilityResult{
			Domain:      "example.com",
			Source:      "dataforseo",
			CitationsOK: true,
			Citations:   services.LLMMetrics{Mentions: 9},
			FetchedAt:   time.Now().UTC(),
		},
	}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/ai-visibility", withUser(u.ID, h.AIVisibility))
	req := httptest.NewRequest(http.MethodGet, "/seo/ai-visibility?project_id="+itoa(p.ID)+"&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available bool `json:"available"`
		Cached    bool `json:"cached"`
		Citations struct {
			Mentions int `json:"mentions"`
		} `json:"citations"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || data.Cached || data.Citations.Mentions != 9 {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 1 {
		t.Fatalf("vendor calls=%d want 1", fake.calls)
	}
}
