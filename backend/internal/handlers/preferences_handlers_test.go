package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"backend/internal/db"
	"backend/internal/repository"

	"github.com/gin-gonic/gin"
)

// TestPreferences_GetReturnsDefaultWhenMissing verifies phase 3's
// "fall back to 'combined' in memory when no preference row exists"
// behaviour. The repo returns gorm.ErrRecordNotFound; the handler must
// surface that as the default and never let the error reach the client.
func TestPreferences_GetReturnsDefaultWhenMissing(t *testing.T) {
	database, prefRepo, _ := newPrefsHarness(t)

	prefHandler := NewPreferencesHandler(prefRepo)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/users/me/preferences", func(c *gin.Context) {
		c.Set("user_id", uint(42))
		prefHandler.Get(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/me/preferences", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			DashboardMode string `json:"dashboard_mode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Success {
		t.Errorf("success = false, want true")
	}
	if body.Data.DashboardMode != "combined" {
		t.Errorf("dashboard_mode = %q, want combined", body.Data.DashboardMode)
	}
	_ = database
}

// TestPreferences_UpdateModeThenGetRoundTrip writes a mode and reads it
// back through the API.
func TestPreferences_UpdateModeThenGetRoundTrip(t *testing.T) {
	database, prefRepo, _ := newPrefsHarness(t)

	prefHandler := NewPreferencesHandler(prefRepo)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/users/me/preferences", func(c *gin.Context) {
		c.Set("user_id", uint(7))
		prefHandler.Get(c)
	})
	r.PATCH("/api/users/me/preferences", func(c *gin.Context) {
		c.Set("user_id", uint(7))
		prefHandler.UpdateMode(c)
	})

	// Write "search".
	body, _ := json.Marshal(map[string]any{"dashboard_mode": "search"})
	req := httptest.NewRequest(http.MethodPatch, "/api/users/me/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	// Read back.
	req2 := httptest.NewRequest(http.MethodGet, "/api/users/me/preferences", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", w2.Code)
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
	_ = database
}

// TestPreferences_UpdateModeRejectsBadValue covers the validation
// path. We don't enumerate all bad values; one failing sample proves
// the gate works.
func TestPreferences_UpdateModeRejectsBadValue(t *testing.T) {
	database, prefRepo, _ := newPrefsHarness(t)

	prefHandler := NewPreferencesHandler(prefRepo)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/api/users/me/preferences", func(c *gin.Context) {
		c.Set("user_id", uint(8))
		prefHandler.UpdateMode(c)
	})

	body, _ := json.Marshal(map[string]any{"dashboard_mode": "bogus"})
	req := httptest.NewRequest(http.MethodPatch, "/api/users/me/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatalf("expected 4xx on bogus mode; got 200; body=%s", w.Body.String())
	}
	_ = database
}

// newPrefsHarness builds an isolated SQLite DB with migrations + plan
// seeds, returns the gorm DB, the UserPreferenceRepository, and the
// entitlements service (unused in these tests; kept for future use).
func newPrefsHarness(t *testing.T) (any, repository.UserPreferenceRepository, any) {
	t.Helper()
	tmp := t.TempDir()
	dsn := filepath.Join(tmp, "prefs_test.db")
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
	prefRepo := repository.NewUserPreferenceRepository(database)
	return database, prefRepo, nil
}
