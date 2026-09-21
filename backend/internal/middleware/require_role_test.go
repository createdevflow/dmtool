package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/models"

	"github.com/gin-gonic/gin"
)

func TestAllowStopImpersonation_AcceptsImpersonationToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("user_role", models.RoleOwner)
		c.Set("is_impersonation", true)
		c.Set("impersonator_id", uint(1))
		c.Next()
	})
	r.POST("/stop", AllowStopImpersonation(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("impersonation JWT: got %d, want 200", w.Code)
	}
}

func TestAllowStopImpersonation_AcceptsAdminToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", models.RoleAdmin)
		c.Set("is_impersonation", false)
		c.Next()
	})
	r.POST("/stop", AllowStopImpersonation(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin JWT: got %d, want 200", w.Code)
	}
}

func TestAllowStopImpersonation_RejectsOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("user_role", models.RoleOwner)
		c.Set("is_impersonation", false)
		c.Next()
	})
	r.POST("/stop", AllowStopImpersonation(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/stop", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("owner JWT: got %d, want 403", w.Code)
	}
}

func TestRequireRole_StillRejectsImpersonationOnAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("user_role", models.RoleAdmin) // even if minted as admin
		c.Set("is_impersonation", true)
		c.Set("impersonator_id", uint(1))
		c.Next()
	})
	r.GET("/admin/stats", RequireRole(models.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/stats", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("impersonation on admin stats: got %d, want 403", w.Code)
	}
}
