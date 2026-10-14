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

func TestDomainCompareUnconfigured(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/domain-compare", withUser(u.ID, h.DomainCompare))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-compare?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
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

func TestDomainCompareCacheMissDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/domain-compare", withUser(u.ID, h.DomainCompare))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-compare?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
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

func TestDomainCompareEmptyCompetitorDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/domain-compare", withUser(u.ID, h.DomainCompare))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-compare?project_id="+itoa(p.ID)+"&fetch=1", nil)
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

func TestDomainCompareSameDomainDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/domain-compare", withUser(u.ID, h.DomainCompare))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-compare?project_id="+itoa(p.ID)+"&competitor=https://www.example.com&fetch=1", nil)
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

func TestDomainCompareServesCacheWithoutVendorCall(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.DomainCompareResult{
		Target:       "example.com",
		Competitor:   "rival.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		TargetSide: services.DomainCompareSide{
			Domain:    "example.com",
			OrganicOK: true,
			Organic:   services.OrganicOverview{Count: 40, ETV: 1234.5},
		},
		FetchedAt: time.Now().UTC(),
	})
	if err := seoRepo.SaveVendorDomainCompareSnapshot(&models.VendorDomainCompareSnapshot{
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
	r.GET("/seo/domain-compare", withUser(u.ID, h.DomainCompare))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-compare?project_id="+itoa(p.ID)+"&competitor=rival.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data struct {
		Available  bool `json:"available"`
		Cached     bool `json:"cached"`
		TargetSide struct {
			Organic struct {
				Count int `json:"count"`
			} `json:"organic"`
		} `json:"target_side"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || !data.Cached || data.TargetSide.Organic.Count != 40 {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("vendor calls=%d want 0 on fresh cache", fake.calls)
	}
}

func TestDomainCompareFetchCallsVendor(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{
		compareResult: services.DomainCompareResult{
			Target:     "example.com",
			Competitor: "rival.com",
			Source:     "dataforseo",
			TargetSide: services.DomainCompareSide{
				Domain:    "example.com",
				OrganicOK: true,
				Organic:   services.OrganicOverview{Count: 11},
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
	r.GET("/seo/domain-compare", withUser(u.ID, h.DomainCompare))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-compare?project_id="+itoa(p.ID)+"&competitor=rival.com&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data struct {
		Available  bool `json:"available"`
		Cached     bool `json:"cached"`
		TargetSide struct {
			Organic struct {
				Count int `json:"count"`
			} `json:"organic"`
		} `json:"target_side"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if !data.Available || data.Cached || data.TargetSide.Organic.Count != 11 {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 1 {
		t.Fatalf("vendor calls=%d want 1", fake.calls)
	}
}
