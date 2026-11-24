package handlers

import (
	"bytes"
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

type stubSiteCrawler struct{}

func (stubSiteCrawler) Crawl(string) (*services.AuditResult, error) {
	return &services.AuditResult{}, nil
}

func (stubSiteCrawler) CrawlSite(target string, opts services.SiteCrawlOptions) (*services.SiteCrawlResult, error) {
	now := time.Now().UTC()
	return &services.SiteCrawlResult{
		SeedURL:   target,
		CrawledAt: now,
		Pages: []services.SitePageResult{{
			URL:        "https://example.com/",
			StatusCode: 200,
			Title:      "Home",
			Score:      80,
			Indexable:  true,
			Checks: []services.AuditCheck{
				{Category: "https", Label: "HTTPS", Status: services.CheckPass, Severity: "high"},
			},
		}},
		StoppedReason: "complete",
	}, nil
}

func TestSiteAuditListEmptyDoesNotInventScores(t *testing.T) {
	database := honestyDB(t)
	u, _ := honestyUserProject(t, database)
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, nil, nil, nil)
	r := gin.New()
	r.GET("/seo/site-audit", withUser(u.ID, h.ListSiteAudit))

	req := httptest.NewRequest(http.MethodGet, "/seo/site-audit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	camps, _ := data["campaigns"].([]any)
	if len(camps) != 0 {
		t.Fatalf("campaigns=%v want empty", camps)
	}
	limits, _ := data["limits"].(map[string]any)
	if limits["plan_label"] != "Free" || limits["pages_per_audit"].(float64) != 100 {
		t.Fatalf("limits %+v", limits)
	}
	if limits["ai_search_health"] != false {
		t.Fatalf("AI Search Health must be off %+v", limits)
	}
}

func TestSiteAuditSaveAndEmptyGet(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, nil, nil, nil)
	r := gin.New()
	r.POST("/seo/site-audit", withUser(u.ID, h.SaveSiteAudit))
	r.GET("/seo/site-audit/campaign", withUser(u.ID, h.GetSiteAudit))

	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "domain": "example.com", "page_limit": 500})
	req := httptest.NewRequest(http.MethodPost, "/seo/site-audit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%s", w.Code, w.Body.String())
	}
	var saved map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &saved); err != nil {
		t.Fatal(err)
	}
	camp, _ := saved["campaign"].(map[string]any)
	if camp["page_limit"].(float64) != 100 {
		t.Fatalf("free page_limit must clamp to 100, got %v", camp["page_limit"])
	}

	req = httptest.NewRequest(http.MethodGet, "/seo/site-audit/campaign?project_id="+itoa(p.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	reports, _ := data["reports"].(map[string]any)
	ov, _ := reports["overview"].(map[string]any)
	sh, _ := ov["site_health"].(map[string]any)
	if sh["value"] != nil {
		t.Fatalf("unaudited site health invented %v", sh)
	}
	th, _ := ov["thematic"].(map[string]any)
	ai, _ := th["ai_search_health"].(map[string]any)
	if ai["status"] != "not_implemented" {
		t.Fatalf("AI Search Health %+v", ai)
	}
}

func TestSiteAuditWebsiteCapOnFree(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	p2 := models.Project{UserID: u.ID, Name: "Two", URL: "https://two.example", Goal: models.GoalSEO}
	if err := database.Create(&p2).Error; err != nil {
		t.Fatal(err)
	}
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, nil, nil, nil)
	r := gin.New()
	r.POST("/seo/site-audit", withUser(u.ID, h.SaveSiteAudit))

	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "domain": "example.com"})
	req := httptest.NewRequest(http.MethodPost, "/seo/site-audit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first save %d %s", w.Code, w.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"project_id": p2.ID, "domain": "two.example"})
	req = httptest.NewRequest(http.MethodPost, "/seo/site-audit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("second save status=%d want 400 body=%s", w.Code, w.Body.String())
	}
}

func TestSiteAuditRunFetchStoresRealPageCount(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, stubSiteCrawler{}, nil, nil)
	r := gin.New()
	r.POST("/seo/site-audit", withUser(u.ID, h.SaveSiteAudit))
	r.POST("/seo/site-audit/run", withUser(u.ID, h.RunSiteAudit))
	r.GET("/seo/site-audit", withUser(u.ID, h.ListSiteAudit))

	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "domain": "example.com"})
	req := httptest.NewRequest(http.MethodPost, "/seo/site-audit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("save %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/seo/site-audit/run?project_id="+itoa(p.ID)+"&fetch=1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("run %d %s", w.Code, w.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	reports, _ := data["reports"].(map[string]any)
	ov, _ := reports["overview"].(map[string]any)
	if ov["pages_crawled"].(float64) != 1 {
		t.Fatalf("pages_crawled=%v want 1 (must not invent 33)", ov["pages_crawled"])
	}
	th, _ := ov["thematic"].(map[string]any)
	ai, _ := th["ai_search_health"].(map[string]any)
	if ai["status"] != "not_implemented" {
		t.Fatalf("must not invent AI Search Health %+v", ai)
	}

	req = httptest.NewRequest(http.MethodGet, "/seo/site-audit", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	camps, _ := data["campaigns"].([]any)
	if len(camps) != 1 {
		t.Fatalf("list len=%d", len(camps))
	}
	row, _ := camps[0].(map[string]any)
	if row["pages_crawled"].(float64) != 1 {
		t.Fatalf("list pages=%v", row["pages_crawled"])
	}
}

func TestSiteAuditRunWithoutFetchDoesNotCrawl(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, stubSiteCrawler{}, nil, nil)
	r := gin.New()
	r.POST("/seo/site-audit", withUser(u.ID, h.SaveSiteAudit))
	r.POST("/seo/site-audit/run", withUser(u.ID, h.RunSiteAudit))

	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "domain": "example.com"})
	req := httptest.NewRequest(http.MethodPost, "/seo/site-audit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodPost, "/seo/site-audit/run?project_id="+itoa(p.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("run %d %s", w.Code, w.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	reports, _ := data["reports"].(map[string]any)
	ov, _ := reports["overview"].(map[string]any)
	if ov["pages_crawled"] != nil {
		t.Fatalf("cache-only run invented pages %v", ov["pages_crawled"])
	}
}
