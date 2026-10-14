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

func TestContentGapUnconfigured(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/content-gap", withUser(u.ID, h.ContentGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/content-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
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

func TestContentGapCacheMissDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/content-gap", withUser(u.ID, h.ContentGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/content-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
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

func TestContentGapEmptyCompetitorDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/content-gap", withUser(u.ID, h.ContentGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/content-gap?project_id="+itoa(p.ID)+"&fetch=1", nil)
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

func TestContentGapSameDomainDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/content-gap", withUser(u.ID, h.ContentGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/content-gap?project_id="+itoa(p.ID)+"&competitor=https://www.example.com&fetch=1", nil)
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

func TestContentGapServesCacheWithoutVendorCall(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.ContentGapResult{
		Target:       "example.com",
		Competitor:   "rival.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		PagesOK:      true,
		Pages: []services.CompetitorPage{
			{URL: "https://rival.com/widgets", ETV: 321.5, Keywords: 18},
		},
		FetchedAt: time.Now().UTC(),
	})
	if err := seoRepo.SaveVendorContentGapSnapshot(&models.VendorContentGapSnapshot{
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
	r.GET("/seo/content-gap", withUser(u.ID, h.ContentGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/content-gap?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data struct {
		Available bool `json:"available"`
		Cached    bool `json:"cached"`
		Pages     []struct {
			URL      string  `json:"url"`
			Keywords int     `json:"keywords"`
			ETV      float64 `json:"etv"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || !data.Cached || len(data.Pages) != 1 || data.Pages[0].Keywords != 18 {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("vendor calls=%d want 0 on fresh cache", fake.calls)
	}
}

func TestContentGapFetchCallsVendor(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{
		contentResult: services.ContentGapResult{
			Target:     "example.com",
			Competitor: "rival.com",
			Source:     "dataforseo",
			MissingOK:  true,
			Missing: []services.ContentGapKeyword{
				{Keyword: "gap page", Volume: 90, CompetitorURL: "https://rival.com/g"},
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
	r.GET("/seo/content-gap", withUser(u.ID, h.ContentGap))
	req := httptest.NewRequest(http.MethodGet, "/seo/content-gap?project_id="+itoa(p.ID)+"&competitor=rival.com&fetch=1", nil)
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
