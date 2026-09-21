package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type memUsers struct {
	byID map[uint]*models.User
}

func (m *memUsers) FindByID(id uint) (*models.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

func disabledUser(id uint) *models.User {
	now := time.Now()
	return &models.User{ID: id, Email: "d@x", Role: models.RoleOwner, DisabledAt: &now}
}

func activeUser(id uint, role string) *models.User {
	return &models.User{ID: id, Email: "a@x", Role: role}
}

func mount(users UserByID, method, path string, pretends func(*gin.Context)) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		pretends(c)
		c.Next()
	})
	r.Use(RejectDisabled(users))
	r.Handle(method, path, func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func TestRejectDisabled_NormalTokenDisabled(t *testing.T) {
	users := &memUsers{byID: map[uint]*models.User{2: disabledUser(2)}}
	w := mount(users, http.MethodGet, "/api/projects", func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("is_impersonation", false)
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestRejectDisabled_NormalTokenActive(t *testing.T) {
	users := &memUsers{byID: map[uint]*models.User{2: activeUser(2, models.RoleOwner)}}
	w := mount(users, http.MethodGet, "/api/projects", func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("is_impersonation", false)
	})
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

func TestRejectDisabled_ImpersonationTargetDisabled(t *testing.T) {
	users := &memUsers{byID: map[uint]*models.User{
		1: activeUser(1, models.RoleAdmin),
		2: disabledUser(2),
	}}
	w := mount(users, http.MethodGet, "/api/projects", func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("is_impersonation", true)
		c.Set("impersonator_id", uint(1))
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("target disabled on product route: got %d, want 401", w.Code)
	}
}

func TestRejectDisabled_ImpersonationAdminDisabled(t *testing.T) {
	users := &memUsers{byID: map[uint]*models.User{
		1: disabledUser(1),
		2: activeUser(2, models.RoleOwner),
	}}
	users.byID[1].Role = models.RoleAdmin
	w := mount(users, http.MethodGet, "/api/projects", func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("is_impersonation", true)
		c.Set("impersonator_id", uint(1))
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("impersonator disabled on product route: got %d, want 401", w.Code)
	}
}

func TestRejectDisabled_StopAllowsDisabledTarget(t *testing.T) {
	users := &memUsers{byID: map[uint]*models.User{
		1: activeUser(1, models.RoleAdmin),
		2: disabledUser(2),
	}}
	w := mount(users, http.MethodPost, "/api/admin/users/2/stop-impersonation", func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("is_impersonation", true)
		c.Set("impersonator_id", uint(1))
	})
	if w.Code != http.StatusOK {
		t.Fatalf("stop with disabled target: got %d, want 200", w.Code)
	}
}

func TestRejectDisabled_ImpersonationBothActive(t *testing.T) {
	users := &memUsers{byID: map[uint]*models.User{
		1: activeUser(1, models.RoleAdmin),
		2: activeUser(2, models.RoleOwner),
	}}
	w := mount(users, http.MethodGet, "/api/projects", func(c *gin.Context) {
		c.Set("user_id", uint(2))
		c.Set("is_impersonation", true)
		c.Set("impersonator_id", uint(1))
	})
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}
