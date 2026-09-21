package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"backend/internal/db"
	"backend/internal/models"

	"github.com/gin-gonic/gin"
)

func TestLoadStaffPermissions_AdminOKOwnerForbiddenImpersonationForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := db.Init(filepath.Join(t.TempDir(), "perm.db"), true)
	t.Cleanup(func() {
		sql, _ := database.DB()
		_ = sql.Close()
	})
	if err := database.Create(&models.User{Name: "A", Email: "a@x", PasswordHash: "x", Role: models.RoleAdmin}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.User{Name: "O", Email: "o@x", PasswordHash: "x", Role: models.RoleOwner}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.SuperAdminCatalogReady(database); err != nil {
		t.Fatalf("hard gate: %v", err)
	}

	hit := func(role string, imp bool) int {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/x", nil)
		// inject via a wrapping engine
		rr := gin.New()
		rr.Use(func(c *gin.Context) {
			var u models.User
			database.Where("role = ?", role).First(&u)
			c.Set("auth_user", &u)
			c.Set("is_impersonation", imp)
			c.Set("impersonator_id", uint(1))
			c.Next()
		})
		rr.Use(LoadStaffPermissions(database))
		rr.GET("/x", RequirePermission(models.PermAdminAccess), func(c *gin.Context) { c.Status(http.StatusOK) })
		rr.ServeHTTP(w, req)
		return w.Code
	}

	if code := hit(models.RoleAdmin, false); code != 200 {
		t.Fatalf("admin: got %d", code)
	}
	if code := hit(models.RoleOwner, false); code != 403 {
		t.Fatalf("owner: got %d", code)
	}
	if code := hit(models.RoleAdmin, true); code != 403 {
		t.Fatalf("impersonation: got %d", code)
	}
}
