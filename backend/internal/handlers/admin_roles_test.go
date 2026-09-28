package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/models"

	"github.com/gin-gonic/gin"
)

func TestCreateRole_OKAndReservedAndRequiresAdminAccess(t *testing.T) {
	h, _, _ := newRBACHandler(t)
	r := gin.New()
	r.POST("/admin/roles", h.CreateRole)

	post := func(body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/admin/roles", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	w := post(map[string]any{
		"code":        "support",
		"name":        "Support",
		"description": "User support",
		"permissions": []string{models.PermAdminAccess, models.PermUsersRead},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d body=%s", w.Code, w.Body.String())
	}

	w = post(map[string]any{
		"code":        "admin",
		"name":        "Nope",
		"permissions": []string{models.PermAdminAccess},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("reserved: got %d", w.Code)
	}

	w = post(map[string]any{
		"code":        "analyst",
		"name":        "Analyst",
		"permissions": []string{models.PermStatsOverviewRead},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing admin.access: got %d body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateAndDeleteRole_SystemProtectedAndInUse(t *testing.T) {
	h, sa, _ := newRBACHandler(t)
	support := models.Role{Code: "support", Name: "Support", IsSystem: false}
	if err := h.db.Create(&support).Error; err != nil {
		t.Fatal(err)
	}
	var access models.Permission
	if err := h.db.Where("code = ?", models.PermAdminAccess).First(&access).Error; err != nil {
		t.Fatal(err)
	}
	if err := h.db.Create(&models.RolePermission{RoleID: support.ID, PermissionID: access.ID}).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.PATCH("/admin/roles/:id", h.UpdateRole)
	r.DELETE("/admin/roles/:id", h.DeleteRole)

	var super models.Role
	if err := h.db.Where("code = ?", models.RoleCodeSuperAdmin).First(&super).Error; err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"name":        "Hacked",
		"permissions": []string{models.PermAdminAccess},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/roles/%d", super.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("patch super: got %d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/roles/%d", super.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete super: got %d", w.Code)
	}

	if err := h.db.Model(sa).Update("role", "support").Error; err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/roles/%d", support.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete in use: got %d body=%s", w.Code, w.Body.String())
	}

	if err := h.db.Model(sa).Update("role", models.RoleAdmin).Error; err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/roles/%d", support.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete unused: got %d body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateRole_ReplacesPermissions(t *testing.T) {
	h, _, _ := newRBACHandler(t)
	role := models.Role{Code: "ops", Name: "Ops"}
	if err := h.db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	var access models.Permission
	if err := h.db.Where("code = ?", models.PermAdminAccess).First(&access).Error; err != nil {
		t.Fatal(err)
	}
	if err := h.db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: access.ID}).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.PATCH("/admin/roles/:id", h.UpdateRole)
	body, _ := json.Marshal(map[string]any{
		"name":        "Operations",
		"description": "ops desk",
		"permissions": []string{models.PermAdminAccess, models.PermUsersRead, models.PermAuditRead},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/roles/%d", role.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update: got %d body=%s", w.Code, w.Body.String())
	}
	var join int64
	h.db.Model(&models.RolePermission{}).Where("role_id = ?", role.ID).Count(&join)
	if join != 3 {
		t.Fatalf("join count=%d want 3", join)
	}
}

func TestListRoles_IncludesSuperAdminAndUserCount(t *testing.T) {
	h, _, _ := newRBACHandler(t)
	r := gin.New()
	r.GET("/admin/roles", h.ListRoles)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/roles", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list: got %d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Data []adminRoleDTO `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data) < 1 {
		t.Fatal("expected Super Admin in list")
	}
	found := false
	for _, row := range env.Data {
		if row.Code == models.RoleCodeSuperAdmin {
			found = true
			if !row.IsSystem {
				t.Error("Super Admin is_system=false")
			}
			if row.UserCount < 1 {
				t.Errorf("Super Admin user_count=%d", row.UserCount)
			}
			if len(row.Permissions) < 19 {
				t.Errorf("Super Admin perms=%d", len(row.Permissions))
			}
		}
	}
	if !found {
		t.Fatal("Super Admin missing from list")
	}
}
