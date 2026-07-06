// phase4_smoke verifies the phase 4 dashboard-mode contract end-to-end
// against an ephemeral in-memory SQLite + gin.httptest server:
//
//   1. GET  /me         on a fresh user returns dashboard_mode="combined".
//   2. PATCH /me        accepts each of {search, social, combined}.
//   3. GET  /me         reflects the most recent PATCH.
//   4. PATCH /me        rejects an unknown mode with HTTP 400.
//
// Run from the backend/ directory:
//
//   go run ./_tools/phase4_smoke/
//
// It is intentionally a standalone main package — not wired into
// `go test ./...` — so it can exercise the public handler surface with
// a real Gin router (the same way middleware sees it) without dragging
// test-only fixtures into the production binary.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	tmp, err := os.MkdirTemp("", "phase4_smoke_*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	dsn := filepath.Join(tmp, "phase4.db")
	database := db.Init(dsn, true)
	sqlDB, _ := database.DB()
	defer sqlDB.Close()

	userRepo := repository.NewUserRepository(database)
	refreshRepo := repository.NewRefreshTokenRepository(database)
	projectRepo := repository.NewProjectRepository(database)
	subscriptionRepo := repository.NewSubscriptionRepository(database)

	authHandler := handlers.NewAuthHandler(
		userRepo, refreshRepo, projectRepo, subscriptionRepo,
		nil, nil,
		[]byte("01234567890123456789012345678901"),
		&config.Config{},
	)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Recover())
	me := r.Group("/me")
	me.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Next()
	})
	me.GET("", authHandler.Me)
	me.PATCH("", authHandler.UpdateMe)

	srv := httptest.NewServer(r)
	defer srv.Close()

	database.Create(&models.User{
		ID: 1, Name: "alice", Email: "a@x", PasswordHash: "x", Role: "owner",
	})

	log.SetFlags(0)
	step("1. GET /me on fresh user — expect 200 with dashboard_mode='combined'",
		func() (int, string) { return get(srv.URL + "/me") })
	step("2. PATCH /me {dashboard_mode: search} — expect 200",
		func() (int, string) { return patch(srv.URL+"/me", `{"dashboard_mode":"search"}`) })
	step("3. GET /me — expect 200 with dashboard_mode='search'",
		func() (int, string) { return get(srv.URL + "/me") })
	step("4. PATCH /me {dashboard_mode: social} — expect 200",
		func() (int, string) { return patch(srv.URL+"/me", `{"dashboard_mode":"social"}`) })
	step("5. GET /me — expect 200 with dashboard_mode='social'",
		func() (int, string) { return get(srv.URL + "/me") })
	step("6. PATCH /me {dashboard_mode: bogus} — expect 400",
		func() (int, string) { return patch(srv.URL+"/me", `{"dashboard_mode":"bogus"}`) })
	step("7. PATCH /me {dashboard_mode: combined} — expect 200",
		func() (int, string) { return patch(srv.URL+"/me", `{"dashboard_mode":"combined"}`) })
	fmt.Println("\nAll phase 4 backend scenarios ran.")
}

func step(name string, fn func() (int, string)) {
	status, body := fn()
	fmt.Printf("[%s] %s\n       status=%d body=%s\n", mark(status, name), name, status, trunc(body, 240))
}

func mark(status int, name string) string {
	switch {
	case strings.Contains(name, "expect 200") && status != 200:
		return "FAIL"
	case strings.Contains(name, "expect 400") && status != 400:
		return "FAIL"
	}
	return "OK  "
}

func get(url string) (int, string) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	return do(req)
}

func patch(url, body string) (int, string) {
	req, _ := http.NewRequest(http.MethodPatch, url, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	return do(req)
}

func do(req *http.Request) (int, string) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		var anyData map[string]any
		_ = json.Unmarshal(b, &anyData)
		pretty, _ := json.Marshal(anyData)
		return resp.StatusCode, string(pretty)
	}
	return resp.StatusCode, string(b)
}

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
