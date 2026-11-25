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

func TestDomainOverviewCacheMissDoesNotFetch(t *testing.T) {
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
	r.GET("/seo/domain-overview", withUser(u.ID, h.DomainOverview))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-overview?project_id="+itoa(p.ID)+"&domain=rival.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if data["available"] != false || data["configured"] != true {
		t.Fatalf("got %+v", data)
	}
	if fake.calls != 0 {
		t.Fatalf("GET without fetch must not call vendor, calls=%d", fake.calls)
	}
	reports, _ := data["reports"].(map[string]any)
	ov, _ := reports["overview"].(map[string]any)
	auth, _ := ov["authority_score"].(map[string]any)
	if auth["status"] != "not_implemented" || auth["value"] != nil {
		t.Fatalf("authority invented %+v", auth)
	}
	ai, _ := ov["ai_visibility"].(map[string]any)
	if ai["status"] != "not_implemented" {
		t.Fatalf("ai %+v", ai)
	}
	limits, _ := data["limits"].(map[string]any)
	if limits["plan_label"] != "Free" || limits["requests_per_day"].(float64) != 10 {
		t.Fatalf("limits %+v", limits)
	}
}

func TestDomainOverviewFetchUsesCacheAndCountsRequest(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	payload, _ := json.Marshal(services.DomainExplorerResult{
		Domain:       "example.com",
		Source:       "dataforseo",
		LocationCode: services.DataForSEOLocation,
		LanguageCode: services.DataForSEOLanguage,
		OrganicOK:    true,
		Organic:      services.OrganicOverview{ETV: 50, Count: 3, PaidCount: 1, PaidETV: 2},
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
	fake := &configuredDFS{}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		seoRepo,
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/domain-overview", withUser(u.ID, h.DomainOverview))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-overview?project_id="+itoa(p.ID)+"&domain=example.com&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if fake.calls != 0 {
		t.Fatalf("warm cache must not call vendor, calls=%d", fake.calls)
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if data["used_today"].(float64) != 1 {
		t.Fatalf("used_today=%v", data["used_today"])
	}
	reports, _ := data["reports"].(map[string]any)
	ov, _ := reports["overview"].(map[string]any)
	traffic, _ := ov["organic_traffic"].(map[string]any)
	if traffic["status"] != "ok" || traffic["value"].(float64) != 50 {
		t.Fatalf("traffic %+v", traffic)
	}
	if ov["authority_score"].(map[string]any)["status"] != "not_implemented" {
		t.Fatalf("authority %+v", ov["authority_score"])
	}
}

func TestDomainOverviewDailyCap(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	for i := 0; i < 10; i++ {
		if err := seoRepo.InsertDomainOverviewRequest(u.ID, "example.com"); err != nil {
			t.Fatal(err)
		}
	}
	fake := &configuredDFS{result: services.DomainExplorerResult{Domain: "example.com", OrganicOK: true, Organic: services.OrganicOverview{ETV: 1}}}
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		seoRepo,
		repository.NewOAuthRepository(database),
		fake, nil, nil, nil,
	)
	r := gin.New()
	r.GET("/seo/domain-overview", withUser(u.ID, h.DomainOverview))
	req := httptest.NewRequest(http.MethodGet, "/seo/domain-overview?project_id="+itoa(p.ID)+"&domain=example.com&fetch=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
	if fake.calls != 0 {
		t.Fatalf("over cap must not call vendor, calls=%d", fake.calls)
	}
}
