package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/config"

	"github.com/gin-gonic/gin"
)

func TestSetRefreshCookie_DevLaxProdNoneSecure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dev := &AuthHandler{cfg: &config.Config{AppEnv: "development"}}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	dev.setRefreshCookie(c, "tok", 3600)
	got := rec.Header().Get("Set-Cookie")
	if !strings.Contains(strings.ToLower(got), "httponly") {
		t.Errorf("dev cookie missing HttpOnly: %s", got)
	}
	if strings.Contains(strings.ToLower(got), "samesite=none") {
		t.Errorf("dev cookie should not be SameSite=None: %s", got)
	}
	if strings.Contains(got, "Secure") {
		t.Errorf("dev cookie should not be Secure: %s", got)
	}

	prod := &AuthHandler{cfg: &config.Config{AppEnv: "production"}}
	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	prod.setRefreshCookie(c, "tok", 3600)
	got = rec.Header().Get("Set-Cookie")
	if !strings.Contains(strings.ToLower(got), "samesite=none") {
		t.Errorf("prod cookie want SameSite=None, got %s", got)
	}
	if !strings.Contains(got, "Secure") {
		t.Errorf("prod cookie want Secure, got %s", got)
	}
	_ = http.StatusOK
}
