package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin_RejectsDisabledUser(t *testing.T) {
	database, userRepo, _ := newAuthHarness(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret12"), 12)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := database.Create(&models.User{
		Name:         "Suspended",
		Email:        "suspended@x.com",
		PasswordHash: string(hash),
		Role:         models.RoleOwner,
		DisabledAt:   &now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	h := NewAuthHandler(database, userRepo, nil, nil, nil, nil, nil, nil, nil, &config.Config{})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", h.Login)

	body, _ := json.Marshal(map[string]string{"email": "suspended@x.com", "password": "secret12"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("login disabled: got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRefresh_RejectsDisabledUser(t *testing.T) {
	database, userRepo, _ := newAuthHarness(t)
	tokenRepo := repository.NewRefreshTokenRepository(database)
	now := time.Now()
	u := &models.User{
		Name:         "Suspended",
		Email:        "suspended-rt@x.com",
		PasswordHash: "x",
		Role:         models.RoleOwner,
		DisabledAt:   &now,
	}
	if err := database.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	raw := "refresh-raw-token"
	rt := &models.RefreshToken{
		UserID:    u.ID,
		TokenHash: utils.HashToken(raw),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := tokenRepo.Create(rt); err != nil {
		t.Fatal(err)
	}

	h := NewAuthHandler(database, userRepo, tokenRepo, nil, nil, nil, nil, nil, nil, &config.Config{AppEnv: "development"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: raw})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("refresh disabled: got %d body=%s", w.Code, w.Body.String())
	}

	got, _ := tokenRepo.FindByRawToken(raw)
	if got != nil {
		t.Errorf("refresh token should be deleted after disabled refresh, still found id=%d", got.ID)
	}
}
