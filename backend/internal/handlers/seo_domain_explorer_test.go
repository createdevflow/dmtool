package handlers

import (
	"context"
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

type configuredDFS struct {
	result        services.DomainExplorerResult
	aiResult      services.AIVisibilityResult
	labs          map[string]services.KeywordLabsMetrics
	err           error
	calls         int
	overviewCalls int
}

func (f *configuredDFS) Configured() bool { return true }
func (f *configuredDFS) FetchEstimatedTraffic(string) ([]models.Metric, error) {
	return nil, nil
}
func (f *configuredDFS) DomainExplorer(_ context.Context, _ string) (services.DomainExplorerResult, error) {
	f.calls++
	return f.result, f.err
}
func (f *configuredDFS) AIVisibility(_ context.Context, _ string) (services.AIVisibilityResult, error) {
	f.calls++
	return f.aiResult, f.err
}
func (f *configuredDFS) KeywordOverview(_ context.Context, _ []string) (map[string]services.KeywordLabsMetrics, error) {
	f.overviewCalls++
	if f.err != nil {
		return nil, f.err
	}
	if f.labs == nil {
		return map[string]services.KeywordLabsMetrics{}, nil
	}
	return f.labs, nil
}

func TestDomainExplorerUnconfigured(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/domain-explorer", withUser(u.ID, h.DomainExplorer))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-explorer?project_id="+itoa(p.ID)+"&domain=rival.com", nil)
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

func TestDomainExplorerCacheMissDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/domain-explorer", withUser(u.ID, h.DomainExplorer))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-explorer?project_id="+itoa(p.ID)+"&domain=rival.com", nil)
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

func TestDomainExplorerServesCacheWithoutVendorCall(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.DomainExplorerResult{
		Domain:       "example.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		OrganicOK:    true,
		Organic:      services.OrganicOverview{ETV: 50, Count: 3},
		BacklinksOK:  true,
		Backlinks:    services.BacklinkOverview{Backlinks: 7, ReferringDomains: 2, Rank: 10},
		FetchedAt:    time.Now().UTC(),
	})
	if err := seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
		Domain:       "example.com",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		Source:       "dataforseo",
		Payload:      string(payload),
		FetchedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	fake := &configuredDFS{result: services.DomainExplorerResult{}}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		seoRepo,
		repository.NewOAuthRepository(database),
		fake,
		nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/domain-explorer", withUser(u.ID, h.DomainExplorer))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-explorer?project_id="+itoa(p.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data struct {
		Available bool `json:"available"`
		Cached    bool `json:"cached"`
		Organic   struct {
			ETV float64 `json:"etv"`
		} `json:"organic"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || !data.Cached || data.Organic.ETV != 50 {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("vendor calls=%d want 0 on fresh cache", fake.calls)
	}
}

func TestBacklinksUsesCachedVendorSnapshot(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.DomainExplorerResult{
		Domain:      "example.com",
		Source:      "dataforseo",
		BacklinksOK: true,
		Backlinks:   services.BacklinkOverview{Backlinks: 7, ReferringDomains: 2, Rank: 10},
		FetchedAt:   time.Now().UTC(),
	})
	if err := seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
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
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/backlinks", withUser(u.ID, h.Backlinks))
	req := httptest.NewRequest(http.MethodGet, "/seo/backlinks?project_id="+itoa(p.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available        bool `json:"available"`
		TotalBacklinks   int  `json:"total_backlinks"`
		ReferringDomains int  `json:"referring_domains"`
		DomainAuthority  int  `json:"domain_authority"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || data.TotalBacklinks != 7 || data.ReferringDomains != 2 {
		t.Fatalf("got %+v", data)
	}
	if data.DomainAuthority != 0 {
		t.Fatal("must not map DataForSEO rank to Moz DA")
	}
	if fake.calls != 0 {
		t.Fatalf("backlinks GET must not call vendor, calls=%d", fake.calls)
	}
}
