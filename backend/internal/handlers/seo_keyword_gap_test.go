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

func TestKeywordGapUnconfigured(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/keyword-gap", withUser(u.ID, h.KeywordGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/keyword-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
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

func TestKeywordGapCacheMissDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/keyword-gap", withUser(u.ID, h.KeywordGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/keyword-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
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

func TestKeywordGapEmptyCompetitorDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/keyword-gap", withUser(u.ID, h.KeywordGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/keyword-gap?project_id="+itoa(p.ID)+"&fetch=1", nil)
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

func TestKeywordGapSameDomainDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/keyword-gap", withUser(u.ID, h.KeywordGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/keyword-gap?project_id="+itoa(p.ID)+"&competitor=https://www.example.com&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available bool   `json:"available"`
		Message   string `json:"message"`
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

func TestKeywordGapServesCacheWithoutVendorCall(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.KeywordGapResult{
		Target:       "example.com",
		Competitor:   "rival.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		MissingOK:    true,
		Missing: []services.GapKeyword{
			{Keyword: "buy widgets", Volume: 2200, CompetitorPosition: 3},
		},
		FetchedAt: time.Now().UTC(),
	})
	if err := seoRepo.SaveVendorKeywordGapSnapshot(&models.VendorKeywordGapSnapshot{
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
	r.GET("/seo/keyword-gap", withUser(u.ID, h.KeywordGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/keyword-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data struct {
		Available bool `json:"available"`
		Cached    bool `json:"cached"`
		Missing   []struct {
			Keyword string `json:"keyword"`
			Volume  int    `json:"volume"`
		} `json:"missing"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || !data.Cached || len(data.Missing) != 1 || data.Missing[0].Keyword != "buy widgets" {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("vendor calls=%d want 0 on fresh cache", fake.calls)
	}
}

func TestKeywordGapFetchCallsVendor(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{
		gapResult: services.KeywordGapResult{
			Target:     "example.com",
			Competitor: "rival.com",
			Source:     "dataforseo",
			MissingOK:  true,
			Missing: []services.GapKeyword{
				{Keyword: "gap kw", Volume: 90, CompetitorPosition: 4},
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
	r.GET("/seo/keyword-gap", withUser(u.ID, h.KeywordGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/keyword-gap?project_id="+itoa(p.ID)+"&competitor=rival.com&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available bool `json:"available"`
		Cached    bool `json:"cached"`
		Missing   []struct {
			Keyword string `json:"keyword"`
			Volume  int    `json:"volume"`
		} `json:"missing"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || data.Cached || len(data.Missing) != 1 || data.Missing[0].Volume != 90 {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 1 {
		t.Fatalf("vendor calls=%d want 1", fake.calls)
	}
}
