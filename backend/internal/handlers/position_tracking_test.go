package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

func TestPositionTrackingEmptyDoesNotInventRanks(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, nil, nil, nil)
	r := gin.New()
	r.GET("/seo/position-tracking", withUser(u.ID, h.GetPositionTracking))

	req := httptest.NewRequest(http.MethodGet, "/seo/position-tracking?project_id="+itoa(p.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	if data["campaign"] != nil {
		t.Fatalf("campaign=%v want null", data["campaign"])
	}
	limits, _ := data["limits"].(map[string]any)
	if limits["plan_label"] != "Free" || limits["keywords"].(float64) != 10 {
		t.Fatalf("limits %+v", limits)
	}
	reports, _ := data["reports"].(map[string]any)
	ov, _ := reports["overview"].(map[string]any)
	if ov["visibility"] != nil || ov["etv"] != nil || ov["ranked"].(float64) != 0 {
		t.Fatalf("overview invented scores %+v", ov)
	}
	tags, _ := reports["tags"].(map[string]any)
	if tags["locked"] != true {
		t.Fatalf("free tags should be locked %+v", tags)
	}
	ai, _ := reports["ai_search"].(map[string]any)
	if ai["locked"] != true {
		t.Fatalf("free AI search should be locked %+v", ai)
	}
}

func TestPositionTrackingFreeKeywordCap(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, nil, nil, nil)
	r := gin.New()
	r.POST("/seo/position-tracking", withUser(u.ID, h.SavePositionTracking))

	kws := make([]map[string]string, 0, 11)
	for i := 0; i < 11; i++ {
		kws = append(kws, map[string]string{"keyword": "kw" + itoa(uint(i+1))})
	}
	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "domain": "example.com", "keywords": kws})
	req := httptest.NewRequest(http.MethodPost, "/seo/position-tracking", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s want 400", w.Code, w.Body.String())
	}
}

func TestPositionTrackingSaveAndGSCFallback(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	oauthRepo := repository.NewOAuthRepository(database)
	h := NewSEOHandler(repository.NewProjectRepository(database), seoRepo, oauthRepo, nil, nil, nil, nil)
	r := gin.New()
	r.POST("/seo/position-tracking", withUser(u.ID, h.SavePositionTracking))
	r.GET("/seo/position-tracking", withUser(u.ID, h.GetPositionTracking))

	body, _ := json.Marshal(map[string]any{
		"project_id":      p.ID,
		"domain":          "example.com",
		"keyword_text":    "coffee|bev\ntea",
		"competitor_text": "rival.com",
		"engine":          "google",
	})
	req := httptest.NewRequest(http.MethodPost, "/seo/position-tracking", bytes.NewReader(body))
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
	kws, _ := camp["keywords"].([]any)
	if len(kws) != 2 {
		t.Fatalf("keywords %+v", kws)
	}
	first := kws[0].(map[string]any)
	if tags, _ := first["tags"].([]any); len(tags) != 0 {
		t.Fatalf("free plan must strip tags %+v", first)
	}

	if err := seoRepo.UpsertKeywords([]models.KeywordResult{
		{ProjectID: p.ID, Seed: "x", Keyword: "coffee", Volume: 12, Clicks: 2, Position: 4},
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.OAuthCredential{UserID: u.ID, ProjectID: p.ID, Provider: "google", AccessTokenEnc: "x"}).Error; err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/seo/position-tracking?project_id="+itoa(p.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", w.Code, w.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	gsc, _ := data["gsc"].(map[string]any)
	if gsc["connected"] != true || gsc["used_as_fallback"] != true {
		t.Fatalf("gsc %+v", gsc)
	}
	reports, _ := data["reports"].(map[string]any)
	rankings, _ := reports["rankings"].([]any)
	found := false
	for _, raw := range rankings {
		row := raw.(map[string]any)
		if row["keyword"] == "coffee" {
			found = true
			if row["source"] != "gsc" {
				t.Fatalf("coffee source=%v want gsc", row["source"])
			}
			if row["position"].(float64) != 4 {
				t.Fatalf("coffee pos=%v", row["position"])
			}
		}
	}
	if !found {
		t.Fatalf("coffee missing from rankings %+v", rankings)
	}
}

func TestPositionTrackingRefreshRequiresFetch(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	fake := &configuredDFS{serpResults: []services.SERPKeywordResult{{
		Keyword: "coffee", OK: true,
		Target:      &services.SERPRank{Domain: "example.com", Position: 3, URL: "https://example.com/coffee"},
		Competitors: []services.SERPRank{}, Cannibalizing: []services.SERPRank{}, SERPFeatures: []string{},
	}}}
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), fake, nil, nil, nil)
	r := gin.New()
	r.POST("/seo/position-tracking", withUser(u.ID, h.SavePositionTracking))
	r.POST("/seo/position-tracking/refresh", withUser(u.ID, h.RefreshPositionTracking))
	r.GET("/seo/position-tracking", withUser(u.ID, h.GetPositionTracking))

	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "domain": "example.com", "keyword_text": "coffee"})
	req := httptest.NewRequest(http.MethodPost, "/seo/position-tracking", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("save %d %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/seo/position-tracking/refresh?project_id="+itoa(p.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh without fetch %d %s", w.Code, w.Body.String())
	}
	if fake.serpCalls != 0 {
		t.Fatalf("cache-only refresh must not call DataForSEO, calls=%d", fake.serpCalls)
	}

	req = httptest.NewRequest(http.MethodPost, "/seo/position-tracking/refresh?project_id="+itoa(p.ID)+"&fetch=1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh fetch %d %s", w.Code, w.Body.String())
	}
	if fake.serpCalls != 1 {
		t.Fatalf("fetch=1 calls=%d want 1", fake.serpCalls)
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	reports, _ := data["reports"].(map[string]any)
	ov, _ := reports["overview"].(map[string]any)
	if ov["visibility"] == nil {
		t.Fatalf("visibility should be computed from real SERP %+v", ov)
	}
	if ov["etv"] != nil {
		t.Fatalf("etv must stay null %+v", ov)
	}
	rankings, _ := reports["rankings"].([]any)
	if len(rankings) != 1 {
		t.Fatalf("rankings %+v", rankings)
	}
	row := rankings[0].(map[string]any)
	if row["position"].(float64) != 3 || row["source"] != "dataforseo_serp" {
		t.Fatalf("row %+v", row)
	}

	req = httptest.NewRequest(http.MethodPost, "/seo/position-tracking/refresh?project_id="+itoa(p.ID)+"&fetch=1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if fake.serpCalls != 2 {
		t.Fatalf("explicit fetch should query again, calls=%d", fake.serpCalls)
	}
}

func TestPositionTrackingRejectsUnknownEngine(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, nil, nil, nil)
	r := gin.New()
	r.POST("/seo/position-tracking", withUser(u.ID, h.SavePositionTracking))
	body, _ := json.Marshal(map[string]any{"project_id": p.ID, "keyword_text": "coffee", "engine": "baidu"})
	req := httptest.NewRequest(http.MethodPost, "/seo/position-tracking", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not connected") {
		t.Fatalf("body=%s", w.Body.String())
	}
}

func TestPositionTrackingAdminUsesAdvancedLimits(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(repository.NewProjectRepository(database), repository.NewSEORepository(database), repository.NewOAuthRepository(database), nil, nil, nil, nil)
	r := gin.New()
	r.GET("/seo/position-tracking", func(c *gin.Context) {
		c.Set("user_id", u.ID)
		c.Set("user_role", models.RoleAdmin)
		h.GetPositionTracking(c)
	})
	req := httptest.NewRequest(http.MethodGet, "/seo/position-tracking?project_id="+itoa(p.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatal(err)
	}
	limits, _ := data["limits"].(map[string]any)
	if limits["plan_label"] != "Advanced" || limits["keywords"].(float64) != 5000 {
		t.Fatalf("admin limits %+v", limits)
	}
	reports, _ := data["reports"].(map[string]any)
	sov, _ := reports["share_of_voice"].(map[string]any)
	if sov["locked"] != false {
		t.Fatalf("advanced SOV should unlock %+v", sov)
	}
}
