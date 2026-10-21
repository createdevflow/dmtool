package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestAlerts_GSCMissingIsTheOnlyEmptyProjectAlert(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	alerts := getAlerts(t, database, u.ID, p.ID)
	if alerts == nil {
		t.Fatal("alerts must be a slice, not nil")
	}
	if len(alerts) != 1 {
		t.Fatalf("alerts=%+v want only gsc_missing", alerts)
	}
	if alerts[0].ID != "gsc_missing" || alerts[0].IsRead || alerts[0].Href != "/integrations" {
		t.Fatalf("got %+v", alerts[0])
	}
}

func TestAlerts_EmptyWhenGSCConnectedAndNoHighIssues(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	if err := database.Create(&models.Metric{
		ProjectID: p.ID, Date: time.Now().Format("2006-01-02"),
		Clicks: 2, Impressions: 20, Source: models.MetricSourceGSC,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.SEOIssue{
		ProjectID: p.ID, URL: p.URL, Severity: models.SeverityMed, Category: "meta", Detail: "short title",
	}).Error; err != nil {
		t.Fatal(err)
	}
	alerts := getAlerts(t, database, u.ID, p.ID)
	if len(alerts) != 0 {
		t.Fatalf("medium issue must not invent a high alert: %+v", alerts)
	}
}

func TestAlerts_HighSeverityIssuesAndStaleVendorCache(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	if err := database.Create(&models.Metric{
		ProjectID: p.ID, Date: time.Now().Format("2006-01-02"),
		Clicks: 2, Impressions: 20, Source: models.MetricSourceGSC,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.SEOIssue{
		ProjectID: p.ID, URL: p.URL, Severity: models.SeverityHigh, Category: "robots", Detail: "blocked",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.SEOIssue{
		ProjectID: p.ID, URL: p.URL, Severity: models.SeverityHigh, Category: "https", Detail: "mixed content",
	}).Error; err != nil {
		t.Fatal(err)
	}
	seoRepo := repository.NewSEORepository(database)
	fetched := time.Now().UTC().Add(-48 * time.Hour)
	payload, _ := json.Marshal(services.DomainExplorerResult{
		Domain: "example.com", Source: "dataforseo", OrganicOK: true,
		Organic: services.OrganicOverview{ETV: 80, Count: 9}, FetchedAt: fetched,
	})
	if err := seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
		Domain: "example.com", LocationCode: services.DataForSEOLocation, LanguageCode: services.DataForSEOLanguage,
		Source: "dataforseo", Payload: string(payload), FetchedAt: fetched,
	}); err != nil {
		t.Fatal(err)
	}

	alerts := getAlerts(t, database, u.ID, p.ID)
	if findAlert(alerts, "gsc_missing") != nil {
		t.Fatal("GSC connected must not keep gsc_missing")
	}
	high := findAlert(alerts, "seo_high")
	if high == nil || high.Count != 2 || high.Href != "/seo/site-explorer" {
		t.Fatalf("seo_high=%+v", high)
	}
	stale := findAlert(alerts, "vendor_stale_domain")
	if stale == nil || stale.IsRead || stale.Severity != "medium" {
		t.Fatalf("vendor_stale_domain=%+v", stale)
	}
	if findAlert(alerts, "vendor_stale_ai") != nil {
		t.Fatal("missing AI cache must not be a stale alert")
	}
}

func TestAlerts_FreshVendorCacheIsNotAnAlert(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	if err := database.Create(&models.Metric{
		ProjectID: p.ID, Date: time.Now().Format("2006-01-02"),
		Clicks: 1, Impressions: 10, Source: models.MetricSourceGSC,
	}).Error; err != nil {
		t.Fatal(err)
	}
	seoRepo := repository.NewSEORepository(database)
	now := time.Now().UTC()
	payload, _ := json.Marshal(services.DomainExplorerResult{
		Domain: "example.com", Source: "dataforseo", OrganicOK: true,
		Organic: services.OrganicOverview{ETV: 12, Count: 2}, FetchedAt: now,
	})
	if err := seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
		Domain: "example.com", LocationCode: services.DataForSEOLocation, LanguageCode: services.DataForSEOLanguage,
		Source: "dataforseo", Payload: string(payload), FetchedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	alerts := getAlerts(t, database, u.ID, p.ID)
	if len(alerts) != 0 {
		t.Fatalf("fresh cache must not alert: %+v", alerts)
	}
}

type alertJSON struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Href     string `json:"href"`
	Severity string `json:"severity"`
	IsRead   bool   `json:"is_read"`
	Count    int    `json:"count"`
}

func getAlerts(t *testing.T, database *gorm.DB, userID, projectID uint) []alertJSON {
	t.Helper()
	h := NewDashboardHandler(
		repository.NewProjectRepository(database),
		repository.NewMetricRepository(database),
		repository.NewInsightRepository(database),
		repository.NewSEORepository(database),
		repository.NewTaskRepository(database),
	)
	r := gin.New()
	r.GET("/alerts", withUser(userID, h.Alerts))
	req := httptest.NewRequest(http.MethodGet, "/alerts?project_id="+strconv.FormatUint(uint64(projectID), 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var alerts []alertJSON
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &alerts); err != nil {
		t.Fatalf("unmarshal alerts: %v body=%s", err, w.Body.String())
	}
	return alerts
}

func findAlert(alerts []alertJSON, id string) *alertJSON {
	for i := range alerts {
		if alerts[i].ID == id {
			return &alerts[i]
		}
	}
	return nil
}
