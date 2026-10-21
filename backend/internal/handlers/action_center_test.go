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

func TestTasks_EmptyWorkHasNoInventedRows(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	payload := getActionCenter(t, database, u.ID, p.ID)

	if payload.Tasks == nil {
		t.Fatal("tasks must be a slice, not nil")
	}
	if len(payload.Tasks) != 0 {
		t.Fatalf("tasks=%v want empty", payload.Tasks)
	}
	if len(payload.Work) != 5 {
		t.Fatalf("work len=%d want 5 loop steps", len(payload.Work))
	}
	for _, w := range payload.Work {
		if w.Kind != "loop" {
			t.Fatalf("empty project work kind=%q want loop", w.Kind)
		}
		if w.Done {
			t.Fatalf("empty project step %s should be incomplete", w.ID)
		}
		if isNumericID(w.ID) {
			t.Fatalf("work id %q must not look like a task row", w.ID)
		}
	}
	assertWork(t, payload.Work, "connect_gsc", false, "/integrations")
	assertWork(t, payload.Work, "run_audit", false, "/seo/site-explorer")
}

func TestTasks_WorkFromStoredStateDoesNotCreateTaskRows(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	p.HealthScore = 70
	if err := database.Save(p).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.Metric{
		ProjectID: p.ID, Date: time.Now().Format("2006-01-02"),
		Clicks: 4, Impressions: 40, Source: models.MetricSourceGSC,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.SEOIssue{
		ProjectID: p.ID, URL: p.URL, Severity: models.SeverityHigh, Category: "robots", Detail: "blocked",
	}).Error; err != nil {
		t.Fatal(err)
	}
	seoRepo := repository.NewSEORepository(database)
	now := time.Now().UTC()
	domainPayload, _ := json.Marshal(services.DomainExplorerResult{
		Domain: "example.com", Source: "dataforseo", OrganicOK: true,
		Organic: services.OrganicOverview{ETV: 50, Count: 3}, FetchedAt: now,
	})
	if err := seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
		Domain: "example.com", LocationCode: services.DataForSEOLocation, LanguageCode: services.DataForSEOLanguage,
		Source: "dataforseo", Payload: string(domainPayload), FetchedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.Task{
		ProjectID: p.ID, Title: "Write homepage copy", Source: models.TaskSourceManual,
	}).Error; err != nil {
		t.Fatal(err)
	}

	payload := getActionCenter(t, database, u.ID, p.ID)
	if len(payload.Tasks) != 1 {
		t.Fatalf("tasks=%d want 1 real row", len(payload.Tasks))
	}
	if payload.Tasks[0].Title != "Write homepage copy" {
		t.Fatalf("task title=%q", payload.Tasks[0].Title)
	}
	assertWork(t, payload.Work, "connect_gsc", true, "/integrations")
	assertWork(t, payload.Work, "run_audit", true, "/seo/site-explorer")
	assertWork(t, payload.Work, "lookup_domain", true, "/seo/site-explorer")
	assertWork(t, payload.Work, "open_issues", false, "/seo/site-explorer")

	var issue *actionWorkJSON
	for i := range payload.Work {
		if payload.Work[i].Kind == "issue" {
			issue = &payload.Work[i]
			break
		}
	}
	if issue == nil {
		t.Fatalf("missing issue work item in %+v", payload.Work)
	}
	if issue.Title != "blocked" || issue.Done {
		t.Fatalf("issue=%+v", issue)
	}
	if isNumericID(issue.ID) {
		t.Fatalf("issue id %q must not be a task id", issue.ID)
	}

	var count int64
	if err := database.Model(&models.Task{}).Where("project_id = ?", p.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("task rows=%d want 1 (work must not insert tasks)", count)
	}
}

func TestTasks_VendorCacheDoesNotBecomeATask(t *testing.T) {
	database := honestyDB(t)
	u, p := honestyUserProject(t, database)
	seoRepo := repository.NewSEORepository(database)
	now := time.Now().UTC()
	domainPayload, _ := json.Marshal(services.DomainExplorerResult{
		Domain: "example.com", Source: "dataforseo", OrganicOK: true,
		Organic: services.OrganicOverview{ETV: 999, Count: 40}, FetchedAt: now,
	})
	if err := seoRepo.SaveVendorDomainSnapshot(&models.VendorDomainSnapshot{
		Domain: "example.com", LocationCode: services.DataForSEOLocation, LanguageCode: services.DataForSEOLanguage,
		Source: "dataforseo", Payload: string(domainPayload), FetchedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	payload := getActionCenter(t, database, u.ID, p.ID)
	if len(payload.Tasks) != 0 {
		t.Fatalf("vendor cache must not create tasks: %+v", payload.Tasks)
	}
	assertWork(t, payload.Work, "lookup_domain", true, "/seo/site-explorer")
}

type actionCenterPayload struct {
	Tasks []models.Task    `json:"tasks"`
	Work  []actionWorkJSON `json:"work"`
}

type actionWorkJSON struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Href   string `json:"href"`
	Done   bool   `json:"done"`
	Detail string `json:"detail"`
}

func getActionCenter(t *testing.T, database *gorm.DB, userID, projectID uint) actionCenterPayload {
	t.Helper()
	h := NewDashboardHandler(
		repository.NewProjectRepository(database),
		repository.NewMetricRepository(database),
		repository.NewInsightRepository(database),
		repository.NewSEORepository(database),
		repository.NewTaskRepository(database),
	)
	r := gin.New()
	r.GET("/dashboard/tasks", withUser(userID, h.Tasks))
	req := httptest.NewRequest(http.MethodGet, "/dashboard/tasks?project_id="+strconv.FormatUint(uint64(projectID), 10), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var payload actionCenterPayload
	if err := json.Unmarshal(decodeData(t, w.Body.Bytes()), &payload); err != nil {
		t.Fatalf("unmarshal tasks: %v body=%s", err, w.Body.String())
	}
	return payload
}

func assertWork(t *testing.T, work []actionWorkJSON, id string, done bool, href string) {
	t.Helper()
	for _, w := range work {
		if w.ID != id {
			continue
		}
		if w.Done != done {
			t.Fatalf("work %s done=%v want %v", id, w.Done, done)
		}
		if href != "" && w.Href != href {
			t.Fatalf("work %s href=%q want %q", id, w.Href, href)
		}
		return
	}
	t.Fatalf("missing work %s in %+v", id, work)
}

func isNumericID(id string) bool {
	if id == "" {
		return false
	}
	_, err := strconv.ParseUint(id, 10, 64)
	return err == nil
}
