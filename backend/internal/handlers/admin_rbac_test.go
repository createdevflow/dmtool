package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/gin-gonic/gin"
)

func newRBACHandler(t *testing.T) (*AdminHandler, *models.User, *models.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	database := db.Init(filepath.Join(t.TempDir(), "rbac.db"), true)
	t.Cleanup(func() {
		sql, _ := database.DB()
		_ = sql.Close()
	})
	sa := models.User{Name: "SA", Email: "sa@x", PasswordHash: "x", Role: models.RoleAdmin}
	owner := models.User{Name: "O", Email: "o@x", PasswordHash: "x", Role: models.RoleOwner}
	if err := database.Create(&sa).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	userRepo := repository.NewUserRepository(database)
	auditRepo := repository.NewAdminAuditLogRepository(database)
	h := NewAdminHandler(database, userRepo, nil, nil, auditRepo, nil, nil, nil)
	return h, &sa, &owner
}

func TestUpdateUser_CannotDemoteLastSuperAdmin(t *testing.T) {
	h, sa, _ := newRBACHandler(t)
	r := gin.New()
	r.PATCH("/admin/users/:id", h.UpdateUser)

	body, _ := json.Marshal(map[string]string{"role": models.RoleOwner})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d", sa.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("last Super Admin demote: got %d body=%s", w.Code, w.Body.String())
	}
	var env map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "LAST_SUPER_ADMIN" {
		t.Fatalf("error code = %v, want LAST_SUPER_ADMIN body=%s", errObj["code"], w.Body.String())
	}
	got, _ := h.userRepo.FindByID(sa.ID)
	if got.Role != models.RoleAdmin {
		t.Fatalf("role mutated to %q", got.Role)
	}
}

func TestUpdateUser_CanDemoteWhenTwoSuperAdmins(t *testing.T) {
	h, sa, _ := newRBACHandler(t)
	second := models.User{Name: "SA2", Email: "sa2@x", PasswordHash: "x", Role: models.RoleAdmin}
	if err := h.db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.PATCH("/admin/users/:id", h.UpdateUser)

	body, _ := json.Marshal(map[string]string{"role": models.RoleOwner})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d", sa.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("demote with two Super Admins: got %d body=%s", w.Code, w.Body.String())
	}
	got, _ := h.userRepo.FindByID(sa.ID)
	if got.Role != models.RoleOwner {
		t.Fatalf("role = %q, want owner", got.Role)
	}
}

func TestSuspendUser_CannotSuspendSuperAdmin(t *testing.T) {
	h, sa, _ := newRBACHandler(t)
	r := gin.New()
	r.POST("/admin/users/:id/suspend", h.SuspendUser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/admin/users/%d/suspend", sa.ID), bytes.NewReader([]byte(`{"reason":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("suspend Super Admin: got %d body=%s", w.Code, w.Body.String())
	}
	var env map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "CANNOT_SUSPEND_ADMIN" {
		t.Fatalf("error code = %v, want CANNOT_SUSPEND_ADMIN", errObj["code"])
	}
	got, _ := h.userRepo.FindByID(sa.ID)
	if got.DisabledAt != nil {
		t.Fatal("Super Admin was suspended")
	}
}

func TestSuspendUser_CanSuspendOwner(t *testing.T) {
	h, _, owner := newRBACHandler(t)
	r := gin.New()
	r.POST("/admin/users/:id/suspend", h.SuspendUser)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/admin/users/%d/suspend", owner.ID), bytes.NewReader([]byte(`{"reason":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("suspend owner: got %d body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateUser_RoleAssignSkippedWhenPermsNotLoaded(t *testing.T) {
	// Legacy RequireRole fallback: StaffPermsLoaded is false, so a Super
	// Admin must still be able to change an owner's role.
	h, _, owner := newRBACHandler(t)
	r := gin.New()
	r.PATCH("/admin/users/:id", h.UpdateUser)

	body, _ := json.Marshal(map[string]string{"role": models.RoleAdmin})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d", owner.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("fallback role assign: got %d body=%s", w.Code, w.Body.String())
	}
}

func TestStats_DoesNotRedactWhenPermsNotLoaded(t *testing.T) {
	harness := newRevenueHarness(t)
	got := callStatsViaHarness(t, harness.h)
	if !got.RevenueAvailable {
		t.Fatal("revenue_available=false under RequireRole fallback — would lock Super Admin stats")
	}
}
