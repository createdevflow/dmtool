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

func TestBacklinkExplorerUnconfigured(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/backlink-explorer", withUser(u.ID, h.BacklinkExplorer))
	req := httptest.NewRequest(http.MethodGet, "/seo/backlink-explorer?project_id="+itoa(p.ID), nil)
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

func TestBacklinkExplorerCacheMissDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/backlink-explorer", withUser(u.ID, h.BacklinkExplorer))
	req := httptest.NewRequest(http.MethodGet, "/seo/backlink-explorer?project_id="+itoa(p.ID), nil)
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

func TestBacklinkExplorerInvalidDomainDoesNotFetch(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	p.URL = "not a domain"
	if err := database.Save(p).Error; err != nil {
		t.Fatal(err)
	}
	fake := &configuredDFS{}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/backlink-explorer", withUser(u.ID, h.BacklinkExplorer))
	req := httptest.NewRequest(http.MethodGet, "/seo/backlink-explorer?project_id="+itoa(p.ID)+"&domain=not%20a%20domain&fetch=1", nil)
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
		t.Fatalf("invalid domain must not call vendor, calls=%d", fake.calls)
	}
}

func TestBacklinkExplorerServesCacheWithoutVendorCall(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.BacklinkExplorerResult{
		Domain:       "example.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		SummaryOK:    true,
		Audit: services.BacklinkAudit{
			Backlinks:        99,
			ReferringDomains: 12,
			Rank:             371,
			LinkTypes:        map[string]int{"anchor": 80},
			LinkAttributes:   map[string]int{},
		},
		ReferringOK: true,
		Referring: []services.ReferringDomain{
			{Domain: "news.example", Rank: 42, Backlinks: 11},
		},
		FetchedAt: time.Now().UTC(),
	})
	if err := seoRepo.SaveVendorBacklinkExplorerSnapshot(&models.VendorBacklinkExplorerSnapshot{
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
	r.GET("/seo/backlink-explorer", withUser(u.ID, h.BacklinkExplorer))
	req := httptest.NewRequest(http.MethodGet, "/seo/backlink-explorer?project_id="+itoa(p.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data struct {
		Available bool `json:"available"`
		Cached    bool `json:"cached"`
		Audit     struct {
			Backlinks        int `json:"backlinks"`
			ReferringDomains int `json:"referring_domains"`
			Rank             int `json:"rank"`
		} `json:"audit"`
		Referring []struct {
			Domain string `json:"domain"`
			Rank   int    `json:"rank"`
		} `json:"referring"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || !data.Cached || data.Audit.Backlinks != 99 || data.Audit.ReferringDomains != 12 || data.Audit.Rank != 371 {
		t.Fatalf("got %+v", data)
	}
	if len(data.Referring) != 1 || data.Referring[0].Domain != "news.example" || data.Referring[0].Rank != 42 {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("vendor calls=%d want 0 on fresh cache", fake.calls)
	}
}

func TestBacklinkExplorerFetchCallsVendor(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{
		backlinkExplorerResult: services.BacklinkExplorerResult{
			Domain:      "example.com",
			Source:      "dataforseo",
			SummaryOK:   true,
			ReferringOK: true,
			Audit: services.BacklinkAudit{
				Backlinks:        50,
				ReferringDomains: 8,
				Rank:             200,
			},
			Referring: []services.ReferringDomain{
				{Domain: "press.site", Rank: 90, Backlinks: 4},
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
	r.GET("/seo/backlink-explorer", withUser(u.ID, h.BacklinkExplorer))
	req := httptest.NewRequest(http.MethodGet, "/seo/backlink-explorer?project_id="+itoa(p.ID)+"&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available bool `json:"available"`
		Cached    bool `json:"cached"`
		Referring []struct {
			Domain string `json:"domain"`
		} `json:"referring"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || data.Cached || len(data.Referring) != 1 || data.Referring[0].Domain != "press.site" {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 1 {
		t.Fatalf("vendor calls=%d want 1", fake.calls)
	}
}
