package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/gin-gonic/gin"
)

func TestHonesty_SyncRouteRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := db.Init(filepath.Join(t.TempDir(), "sync_route.db"), true)
	if database == nil {
		t.Fatal("db.Init returned nil")
	}
	t.Cleanup(func() {
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	u := &models.User{Name: "R", Email: "sync-route@example.com", PasswordHash: "x", Role: "owner"}
	if err := database.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	p := &models.Project{UserID: u.ID, Name: "P", URL: "https://example.com", Goal: models.GoalSocial, Status: "active"}
	if err := database.Create(p).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}

	r := gin.New()
	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		c.Set("user_id", u.ID)
		c.Next()
	})
	registerSyncRoutes(
		api,
		repository.NewProjectRepository(database),
		repository.NewMetricRepository(database),
		repository.NewOAuthRepository(database),
		repository.NewSEORepository(database),
		nil, nil, nil, nil, nil,
		[]byte("01234567890123456789012345678901"),
		"", "",
	)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+strconv.FormatUint(uint64(p.ID), 10)+"/sync", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusNotFound {
		t.Fatalf("POST /api/projects/:id/sync is 404 — registerSyncRoutes not mounted")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
