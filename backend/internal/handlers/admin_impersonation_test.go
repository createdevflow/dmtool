package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/db"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/gin-gonic/gin"
)

func TestStopImpersonation_ImpersonationJWTWritesAuditAsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := t.TempDir()
	database := db.Init(tmp+"/imp.db", true)
	t.Cleanup(func() {
		sqlDB, _ := database.DB()
		_ = sqlDB.Close()
	})
	auditRepo := repository.NewAdminAuditLogRepository(database)
	h := NewAdminHandler(database, nil, nil, nil, auditRepo, nil, nil, nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("user_role", models.RoleOwner)
		c.Set("is_impersonation", true)
		c.Set("impersonator_id", uint(1))
		c.Next()
	})
	r.POST("/admin/users/:id/stop-impersonation", h.StopImpersonation)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]uint{"target_id": 2})
	req, _ := http.NewRequest(http.MethodPost, "/admin/users/2/stop-impersonation", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("stop with impersonation JWT: got %d body=%s", w.Code, w.Body.String())
	}

	var rows []models.AdminAuditLog
	if err := database.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("audit rows = %d, want 1", len(rows))
	}
	got := rows[0]
	if got.Action != models.AdminAuditActionImpersonateStop {
		t.Errorf("action = %q, want %q", got.Action, models.AdminAuditActionImpersonateStop)
	}
	if got.ActorUserID != 1 {
		t.Errorf("actor = %d, want 1 (impersonator, not target)", got.ActorUserID)
	}
	if got.TargetUserID != 2 {
		t.Errorf("target = %d, want 2", got.TargetUserID)
	}
}

func TestStopImpersonation_AdminJWTStillWritesAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := t.TempDir()
	database := db.Init(tmp+"/imp2.db", true)
	t.Cleanup(func() {
		sqlDB, _ := database.DB()
		_ = sqlDB.Close()
	})
	auditRepo := repository.NewAdminAuditLogRepository(database)
	h := NewAdminHandler(database, nil, nil, nil, auditRepo, nil, nil, nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", models.RoleAdmin)
		c.Set("is_impersonation", false)
		c.Next()
	})
	r.POST("/admin/users/:id/stop-impersonation", h.StopImpersonation)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]uint{"target_id": 2})
	req, _ := http.NewRequest(http.MethodPost, "/admin/users/2/stop-impersonation", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("stop with admin JWT: got %d body=%s", w.Code, w.Body.String())
	}

	var rows []models.AdminAuditLog
	if err := database.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("audit rows = %d, want 1", len(rows))
	}
	if rows[0].ActorUserID != 1 || rows[0].TargetUserID != 2 {
		t.Errorf("actor/target = %d/%d, want 1/2", rows[0].ActorUserID, rows[0].TargetUserID)
	}
}
