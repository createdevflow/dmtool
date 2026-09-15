package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services/entitlements"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func honestyDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	database := db.Init(filepath.Join(t.TempDir(), "honesty.db"), true)
	if database == nil {
		t.Fatal("db.Init returned nil")
	}
	t.Cleanup(func() {
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return database
}

func honestyUserProject(t *testing.T, database *gorm.DB) (*models.User, *models.Project) {
	t.Helper()
	u := &models.User{
		Name:         "Honesty",
		Email:        "honesty-" + t.Name() + "@example.com",
		PasswordHash: "x",
		Role:         "owner",
	}
	if err := database.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	p := &models.Project{
		UserID: u.ID,
		Name:   "Honesty Project",
		URL:    "https://example.com",
		Goal:   models.GoalSocial,
		Status: "active",
	}
	if err := database.Create(p).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	return u, p
}

func withUser(userID uint, h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		h(c)
	}
}

func decodeData(t *testing.T, body []byte) json.RawMessage {
	t.Helper()
	var wrap struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("unmarshal envelope: %v body=%s", err, body)
	}
	if !wrap.Success {
		t.Fatalf("success=false body=%s", body)
	}
	return wrap.Data
}

func TestHonesty_BacklinksUnavailableZeros(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSEOHandler(
		repository.NewProjectRepository(database),
		repository.NewSEORepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil,
	)

	r := gin.New()
	r.GET("/seo/backlinks", withUser(u.ID, h.Backlinks))

	req := httptest.NewRequest(http.MethodGet, "/seo/backlinks?project_id="+itoa(p.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var data struct {
		Available        bool `json:"available"`
		TotalBacklinks   int  `json:"total_backlinks"`
		ReferringDomains int  `json:"referring_domains"`
		DoFollow         int  `json:"do_follow"`
		NoFollow         int  `json:"no_follow"`
		DomainAuthority  int  `json:"domain_authority"`
		IsEstimated      bool `json:"is_estimated"`
	}
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if data.Available {
		t.Error("available want false")
	}
	if data.IsEstimated {
		t.Error("is_estimated want false")
	}
	if data.TotalBacklinks != 0 || data.ReferringDomains != 0 || data.DoFollow != 0 || data.NoFollow != 0 || data.DomainAuthority != 0 {
		t.Errorf("expected all zeros, got %+v", data)
	}
}

func TestHonesty_RelatedProfilesEmpty(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSocialHandler(
		repository.NewProjectRepository(database),
		repository.NewMetricRepository(database),
		repository.NewOAuthRepository(database),
		nil, nil, nil, nil, "", "", nil,
	)

	r := gin.New()
	r.GET("/social/related", withUser(u.ID, h.RelatedProfiles))

	req := httptest.NewRequest(http.MethodGet, "/social/related?project_id="+itoa(p.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var profiles []RelatedProfilesData
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &profiles); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("related profiles = %d, want empty", len(profiles))
	}
}

func TestHonesty_OnboardInsertsNoSeedMetrics(t *testing.T) {
	database := honestyDB(t)
	u := &models.User{
		Name:         "Onboard",
		Email:        "onboard-" + t.Name() + "@example.com",
		PasswordHash: "x",
		Role:         "owner",
	}
	if err := database.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	projectRepo := repository.NewProjectRepository(database)
	metricRepo := repository.NewMetricRepository(database)
	h := NewProjectHandler(
		projectRepo,
		database,
		nil,
		metricRepo,
		repository.NewOAuthRepository(database),
		repository.NewSEORepository(database),
		nil, nil, nil,
		entitlements.New(database, repository.NewSubscriptionRepository(database), repository.NewPlanRepository(database), projectRepo),
	)

	r := gin.New()
	r.POST("/onboard", withUser(u.ID, h.Onboard))

	body, _ := json.Marshal(map[string]any{
		"name": "Empty onboard",
		"goal": "social",
	})
	req := httptest.NewRequest(http.MethodPost, "/onboard", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var wrap struct {
		Data struct {
			Project models.Project `json:"project"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pid := wrap.Data.Project.ID
	if pid == 0 {
		t.Fatal("onboard returned project id 0")
	}
	var wrapMode struct {
		Data struct {
			DashboardMode string `json:"dashboard_mode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrapMode); err != nil {
		t.Fatalf("unmarshal mode: %v", err)
	}
	if wrapMode.Data.DashboardMode != models.DashboardModeSocial {
		t.Errorf("onboard dashboard_mode=%q want social", wrapMode.Data.DashboardMode)
	}
	var uRow models.User
	if err := database.First(&uRow, u.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if uRow.DashboardMode != models.DashboardModeSocial {
		t.Errorf("user.dashboard_mode=%q want social", uRow.DashboardMode)
	}

	time.Sleep(200 * time.Millisecond)

	var seeded []models.Metric
	if err := database.Where("project_id = ? AND source = ?", pid, models.MetricSourceSeed).Find(&seeded).Error; err != nil {
		t.Fatalf("query metrics: %v", err)
	}
	if len(seeded) != 0 {
		t.Errorf("onboard wrote %d source=seed metrics, want 0", len(seeded))
	}
	var all []models.Metric
	if err := database.Where("project_id = ?", pid).Find(&all).Error; err != nil {
		t.Fatalf("query all metrics: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("onboard wrote %d metric rows, want 0", len(all))
	}
}

func TestHonesty_RankTrackingIgnoresPositionZero(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	oauthRepo := repository.NewOAuthRepository(database)
	h := NewSEOHandler(repository.NewProjectRepository(database), seoRepo, oauthRepo, nil, nil, nil, nil)

	if err := seoRepo.UpsertKeywords([]models.KeywordResult{
		{ProjectID: p.ID, Seed: "x", Keyword: "autocomplete zero", Volume: 0, KD: 0, Position: 0},
		{ProjectID: p.ID, Seed: "x", Keyword: "gsc ranked", Volume: 10, KD: 0, Position: 2},
	}); err != nil {
		t.Fatalf("upsert keywords: %v", err)
	}

	r := gin.New()
	r.GET("/seo/rank-tracking", withUser(u.ID, h.RankTracking))

	get := func() (int, map[string]any) {
		req := httptest.NewRequest(http.MethodGet, "/seo/rank-tracking?project_id="+itoa(p.ID), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var data map[string]any
		if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return w.Code, data
	}

	_, noGSC := get()
	if noGSC["gsc_connected"] != false {
		t.Errorf("gsc_connected=%v want false", noGSC["gsc_connected"])
	}
	if total, _ := noGSC["total"].(float64); total != 0 {
		t.Errorf("without GSC total=%v want 0", noGSC["total"])
	}
	buckets, _ := noGSC["buckets"].(map[string]any)
	if buckets["top3"].(float64) != 0 {
		t.Errorf("without GSC top3=%v want 0 (position 0 must not invent top-3)", buckets["top3"])
	}

	if err := database.Create(&models.OAuthCredential{
		UserID:         u.ID,
		ProjectID:      p.ID,
		Provider:       "google",
		AccessTokenEnc: "x",
	}).Error; err != nil {
		t.Fatalf("create google oauth: %v", err)
	}

	_, withGSC := get()
	if withGSC["gsc_connected"] != true {
		t.Errorf("gsc_connected=%v want true", withGSC["gsc_connected"])
	}
	if total, _ := withGSC["total"].(float64); total != 1 {
		t.Errorf("with GSC total=%v want 1 (only position>0)", withGSC["total"])
	}
	gb, _ := withGSC["buckets"].(map[string]any)
	if gb["top3"].(float64) != 1 {
		t.Errorf("with GSC top3=%v want 1", gb["top3"])
	}
	kws, _ := withGSC["keywords"].([]any)
	for _, raw := range kws {
		kw := raw.(map[string]any)
		if kw["position"].(float64) <= 0 {
			t.Errorf("ranked list included position<=0: %+v", kw)
		}
	}
}

func TestHonesty_SyncNot404(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewSyncHandler(
		repository.NewProjectRepository(database),
		repository.NewMetricRepository(database),
		repository.NewOAuthRepository(database),
		repository.NewSEORepository(database),
		nil, nil, nil, nil, nil,
		[]byte("01234567890123456789012345678901"),
		"", "",
	)

	r := gin.New()
	r.POST("/projects/:id/sync", withUser(u.ID, h.SyncProject))

	req := httptest.NewRequest(http.MethodPost, "/projects/"+itoa(p.ID)+"/sync", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusNotFound {
		t.Fatalf("sync 404; body=%s", w.Body.String())
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"traffic", "rankings", "social"} {
		block, _ := data[key].(map[string]any)
		if block["status"] != "skipped" {
			t.Errorf("%s status=%v want skipped", key, block["status"])
		}
	}
}

func TestHonesty_ReplaceOpenIssuesDoesNotDuplicate(t *testing.T) {
	database := honestyDB(t)
	_, p := honestyUserProject(t, database)
	repo := repository.NewSEORepository(database)
	batch := []models.SEOIssue{
		{ProjectID: p.ID, URL: p.URL, Severity: "high", Category: "meta", Detail: "Meta Description: missing"},
		{ProjectID: p.ID, URL: p.URL, Severity: "medium", Category: "technical", Detail: "Canonical Tag: missing"},
	}
	if err := repo.ReplaceOpenIssues(p.ID, batch); err != nil {
		t.Fatalf("first replace: %v", err)
	}
	if err := repo.ReplaceOpenIssues(p.ID, batch); err != nil {
		t.Fatalf("second replace: %v", err)
	}
	open, err := repo.FindOpenIssues(p.ID, "")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(open) != 2 {
		t.Fatalf("open issues=%d want 2 (re-audit must not append)", len(open))
	}
}

func TestHonesty_ContentGenerateProjectZero404(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	h := NewContentHandler(repository.NewProjectRepository(database), repository.NewInsightRepository(database), nil, &config.Config{})

	r := gin.New()
	r.POST("/content/generate", withUser(u.ID, h.Generate))

	zero, _ := json.Marshal(map[string]any{"project_id": 0, "topic": "hello", "platform": "blog"})
	req0 := httptest.NewRequest(http.MethodPost, "/content/generate", bytes.NewReader(zero))
	req0.Header.Set("Content-Type", "application/json")
	w0 := httptest.NewRecorder()
	r.ServeHTTP(w0, req0)
	if w0.Code != http.StatusNotFound {
		t.Fatalf("project_id=0 status=%d want 404 body=%s", w0.Code, w0.Body.String())
	}

	okBody, _ := json.Marshal(map[string]any{"project_id": p.ID, "topic": "hello", "platform": "blog"})
	req := httptest.NewRequest(http.MethodPost, "/content/generate", bytes.NewReader(okBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("real project status=%d body=%s", w.Code, w.Body.String())
	}
}

func itoa(n uint) string {
	return jsonNumber(n)
}

func jsonNumber(n uint) string {
	b, _ := json.Marshal(n)
	return string(b)
}
