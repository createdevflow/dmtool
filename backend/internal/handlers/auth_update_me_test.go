package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TestAuthUpdateMe_RoundTrip writes dashboard_mode and reads it back.
func TestAuthUpdateMe_RoundTrip(t *testing.T) {
	database, userRepo, _ := newAuthHarness(t)
	authHandler := NewAuthHandler(
		database, userRepo, nil, nil, nil, nil,
		nil, nil,
		[]byte("01234567890123456789012345678901"),
		&config.Config{},
	)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", func(c *gin.Context) {
		c.Set("user_id", uint(11))
		authHandler.Me(c)
	})
	r.PATCH("/me", func(c *gin.Context) {
		c.Set("user_id", uint(11))
		authHandler.UpdateMe(c)
	})

	database.Create(&models.User{
		ID:           11,
		Name:         "Alice",
		Email:        "a@x",
		PasswordHash: "x",
		Role:         "owner",
	})

	body, _ := json.Marshal(map[string]any{"dashboard_mode": "search"})
	req := httptest.NewRequest(http.MethodPatch, "/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/me", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200; body=%s", w2.Code, w2.Body.String())
	}
	var resp struct {
		Data struct {
			DashboardMode string `json:"dashboard_mode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.DashboardMode != "search" {
		t.Errorf("round-trip: got %q want search", resp.Data.DashboardMode)
	}
}

// TestAuthUpdateMe_RejectsBadValue covers the validation path.
func TestAuthUpdateMe_RejectsBadValue(t *testing.T) {
	database, userRepo, _ := newAuthHarness(t)
	authHandler := NewAuthHandler(
		database, userRepo, nil, nil, nil, nil,
		nil, nil,
		[]byte("01234567890123456789012345678901"),
		&config.Config{},
	)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/me", func(c *gin.Context) {
		c.Set("user_id", uint(12))
		authHandler.UpdateMe(c)
	})

	database.Create(&models.User{
		ID:           12,
		Name:         "Bob",
		Email:        "b@x",
		PasswordHash: "x",
		Role:         "owner",
	})

	body, _ := json.Marshal(map[string]any{"dashboard_mode": "bogus"})
	req := httptest.NewRequest(http.MethodPatch, "/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Errorf("expected 4xx on bogus mode; got 200; body=%s", w.Body.String())
	}
}

// TestAuthMe_DefaultDashboardMode confirms a fresh user reads back "combined".
func TestAuthMe_DefaultDashboardMode(t *testing.T) {
	database, userRepo, _ := newAuthHarness(t)
	authHandler := NewAuthHandler(
		database, userRepo, nil, nil, nil, nil,
		nil, nil,
		[]byte("01234567890123456789012345678901"),
		&config.Config{},
	)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", func(c *gin.Context) {
		c.Set("user_id", uint(13))
		authHandler.Me(c)
	})

	database.Create(&models.User{
		ID:           13,
		Name:         "Cara",
		Email:        "c@x",
		PasswordHash: "x",
		Role:         "owner",
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data struct {
			DashboardMode string `json:"dashboard_mode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.DashboardMode != "combined" {
		t.Errorf("default dashboard_mode = %q, want combined", resp.Data.DashboardMode)
	}
}

// newAuthHarness wraps db.Init with the migrations + plan seeds.
func newAuthHarness(t *testing.T) (*gorm.DB, repository.UserRepository, interface{}) {
	t.Helper()
	tmp := t.TempDir()
	dsn := filepath.Join(tmp, "auth_test.db")
	database := db.Init(dsn, true)
	if database == nil {
		t.Fatal("db.Init returned nil")
	}
	t.Cleanup(func() {
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	userRepo := repository.NewUserRepository(database)
	return database, userRepo, nil
}
