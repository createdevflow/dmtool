// main.go — DMTool v2.0 entry point.
// Initialises config, DB, background workers, and the Gin HTTP server.
package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"
	"backend/internal/workers"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

func main() {
	// ── 1. Load configuration ───────────────────────────────────────────────
	cfg := config.Load()

	// ── 2. Parse RS256 key pair ─────────────────────────────────────────────
	privateKey, publicKey := loadRSAKeys(cfg)

	// ── 3. Initialise database ──────────────────────────────────────────────
	isDev := cfg.AppEnv == "development"
	database := db.Init(cfg.DatabaseURL, isDev)

	// ── 4. Build repositories (dependency injection) ────────────────────────
	userRepo := repository.NewUserRepository(database)
	projectRepo := repository.NewProjectRepository(database)
	tokenRepo := repository.NewRefreshTokenRepository(database)
	metricRepo := repository.NewMetricRepository(database)
	oauthRepo := repository.NewOAuthRepository(database)
	seoRepo := repository.NewSEORepository(database)
	insightRepo := repository.NewInsightRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	encKey := utils.EncryptionKeyFromString(cfg.EncryptionKey)

	dataForSEOService := services.NewDataForSEOService(cfg.DataForSEOLogin, cfg.DataForSEOPassword)
	rapidAPIService := services.NewRapidAPIService(cfg.RapidAPIKey)
	
	// Deprecated OAuth services (kept for legacy interfaces but with dummy impl if needed, 
	// or we can remove them if we refactor all routes)
	// We'll replace them below.
	crawlerService := services.NewSEOCrawlerService()
	keywordService := services.NewKeywordService()
	socialScraperService := services.NewSocialScraperService()

	var openaiService services.OpenAIService
	if cfg.OpenAIAPIKey != "" {
		openaiService = services.NewOpenAIService(cfg.OpenAIAPIKey)
	} else {
		log.Println("[main] No OPENAI_API_KEY — AI insights will use rule-based engine")
	}

	// ── 5. Start background workers ─────────────────────────────────────────
	go workers.StartMetricsSyncer(database, projectRepo, metricRepo, oauthRepo, gscService, metaService, encKey, cfg)
	go workers.StartHealthScorer(database, projectRepo, seoRepo)
	go workers.StartInsightGenerator(database, projectRepo, metricRepo, insightRepo, openaiService, cfg)

	// ── 6. Build Gin router ─────────────────────────────────────────────────
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Middleware stack
	r.Use(middleware.Recover())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.Logger())
	r.Use(middleware.RateLimitAPI())

	// ── 7. Health check (public) ─────────────────────────────────────────────
	startTime := time.Now()
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		httpStatus := http.StatusOK
		if err := db.Ping(database); err != nil {
			dbStatus = "error"
			httpStatus = http.StatusServiceUnavailable
		}
		c.JSON(httpStatus, gin.H{
			"status":  dbStatus,
			"db":      dbStatus,
			"version": cfg.Version,
			"uptime":  time.Since(startTime).Round(time.Second).String(),
		})
	})

	// Public SEO audit (no auth required — for landing page demo)
	seoHandlerPublic := handlers.NewSEOHandler(projectRepo, seoRepo, oauthRepo, gscService, crawlerService, keywordService, encKey)
	r.GET("/api/public/seo-audit", seoHandlerPublic.PublicAudit)

	// ── 8. Auth routes (public, with strict rate limit) ──────────────────────
	authGroup := r.Group("/api/auth")
	authGroup.Use(middleware.RateLimitAuth())
	registerAuthRoutes(authGroup, database, userRepo, tokenRepo, projectRepo, privateKey, publicKey, encKey, cfg)

	// ── 9. Protected API routes ──────────────────────────────────────────────
	api := r.Group("/api")
	api.Use(middleware.JWTAuth(publicKey))

	registerProjectRoutes(api, projectRepo, database, encKey, metricRepo, oauthRepo, seoRepo, dataForSEOService, rapidAPIService, crawlerService)
	registerDashboardRoutes(api, projectRepo, metricRepo, insightRepo, seoRepo, taskRepo)
	registerSEORoutes(api, projectRepo, seoRepo, oauthRepo, dataForSEOService, crawlerService, keywordService, encKey)
	registerSocialRoutes(api, projectRepo, oauthRepo, metricRepo, rapidAPIService, encKey)
	registerContentRoutes(api, projectRepo, insightRepo, openaiService, cfg)
	registerTaskRoutes(api, insightRepo)
	registerSystemRoutes(api, projectRepo, taskRepo)
	registerSyncRoutes(api, projectRepo, metricRepo, oauthRepo, seoRepo, insightRepo, dataForSEOService, rapidAPIService, crawlerService, encKey)

	// ── 10. Start server ─────────────────────────────────────────────────────
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🚀 DMTool v%s starting on %s [%s mode]", cfg.Version, addr, cfg.AppEnv)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// loadRSAKeys parses PEM-encoded RSA keys from config.
func loadRSAKeys(cfg *config.Config) (*rsa.PrivateKey, *rsa.PublicKey) {
	if cfg.JWTPrivateKeyPEM == "" || cfg.JWTPublicKeyPEM == "" {
		if cfg.AppEnv == "development" {
			log.Println("[jwt] Using stable dev key pair (consistent across restarts)")
			cfg.JWTPrivateKeyPEM = utils.DevPrivateKey
			cfg.JWTPublicKeyPEM = utils.DevPublicKey
		} else {
			log.Fatal("[jwt] JWT_PRIVATE_KEY and JWT_PUBLIC_KEY are required in production")
		}
	}

	privBlock, _ := pem.Decode([]byte(cfg.JWTPrivateKeyPEM))
	if privBlock == nil {
		log.Fatal("[jwt] Failed to decode JWT_PRIVATE_KEY PEM block")
	}

	var privateKey *rsa.PrivateKey
	if key, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes); err == nil {
		privateKey = key
	} else if key, err2 := x509.ParsePKCS8PrivateKey(privBlock.Bytes); err2 == nil {
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			log.Fatal("[jwt] Private key is not RSA")
		}
	} else {
		log.Fatalf("[jwt] Failed to parse private key: %v", err)
	}

	pubBlock, _ := pem.Decode([]byte(cfg.JWTPublicKeyPEM))
	if pubBlock == nil {
		log.Fatal("[jwt] Failed to decode JWT_PUBLIC_KEY PEM block")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		log.Fatalf("[jwt] Failed to parse public key: %v", err)
	}
	publicKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		log.Fatal("[jwt] Public key is not RSA")
	}

	return privateKey, publicKey
}

// ── Route registration ──────────────────────────────────────────────────────

func registerAuthRoutes(g *gin.RouterGroup, _ *gorm.DB,
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	projectRepo repository.ProjectRepository,
	privKey *rsa.PrivateKey, pubKey *rsa.PublicKey,
	encKey []byte, cfg *config.Config) {

	h := handlers.NewAuthHandler(userRepo, tokenRepo, projectRepo, privKey, pubKey, encKey, cfg)

	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
	g.POST("/refresh", h.Refresh)
	g.POST("/logout", h.Logout)
	g.GET("/me", middleware.JWTAuth(pubKey), h.Me)
}

func registerProjectRoutes(g *gin.RouterGroup, projectRepo repository.ProjectRepository, database *gorm.DB, encKey []byte,
	metricRepo repository.MetricRepository, oauthRepo repository.OAuthRepository, seoRepo repository.SEORepository,
	gsc services.GSCService, meta services.MetaService, crawler services.SEOCrawlerService, scraper services.SocialScraperService) {
	h := handlers.NewProjectHandler(projectRepo, database, encKey, metricRepo, oauthRepo, seoRepo, gsc, meta, crawler, scraper)

	g.GET("/projects", h.List)
	g.POST("/projects", h.Create)
	g.POST("/onboard", h.Onboard)

	projectScoped := g.Group("/projects/:id")
	projectScoped.Use(middleware.ProjectGuard(projectRepo))
	{
		projectScoped.GET("", func(c *gin.Context) {
			userID := c.MustGet("user_id").(uint)
			id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
			p, _ := projectRepo.FindByIDAndUser(uint(id), userID)
			utils.Success(c, p, nil)
		})
		projectScoped.PATCH("", h.Update)
		projectScoped.DELETE("", h.Delete)
	}
}


func registerDashboardRoutes(g *gin.RouterGroup,
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	insightRepo repository.InsightRepository,
	seoRepo repository.SEORepository,
	taskRepo repository.TaskRepository) {

	h := handlers.NewDashboardHandler(projectRepo, metricRepo, insightRepo, seoRepo, taskRepo)

	g.GET("/dashboard/snapshot", h.Snapshot)
	g.GET("/dashboard/metrics", h.Metrics)
	g.GET("/dashboard/traffic", h.Traffic)
	g.GET("/dashboard/insights", h.Insights)
	g.GET("/dashboard/tasks", h.Tasks)
	g.POST("/dashboard/tasks", h.CreateTask)
	g.GET("/competitors", h.Competitors)
	g.GET("/alerts", h.Alerts)
}

func registerSEORoutes(
	g *gin.RouterGroup,
	projectRepo repository.ProjectRepository,
	seoRepo repository.SEORepository,
	oauthRepo repository.OAuthRepository,
	dataForSEOSvc services.DataForSEOService,
	crawler services.SEOCrawlerService,
	keywords services.KeywordService,
	encKey []byte,
) {
	h := handlers.NewSEOHandler(projectRepo, seoRepo, oauthRepo, dataForSEOSvc, crawler, keywords, encKey)

	g.POST("/seo/audit/run", h.AuditRun)
	g.GET("/seo/audit", h.GetAuditStatus)
	g.GET("/seo/issues", h.Issues)
	g.GET("/seo/keywords", h.Keywords)
	g.POST("/seo/keywords", h.KeywordsPost)
	g.GET("/seo/report", h.Report)
	g.PUT("/seo/issues/:id", h.ResolveIssue)
	g.GET("/seo/rank-tracking", h.RankTracking)
	g.GET("/seo/backlinks", h.Backlinks)
}

func registerSocialRoutes(
	g *gin.RouterGroup,
	projectRepo repository.ProjectRepository,
	oauthRepo repository.OAuthRepository,
	metricRepo repository.MetricRepository,
	rapidAPISvc services.RapidAPIService,
	encKey []byte,
) {
	h := handlers.NewSocialHandler(projectRepo, metricRepo, oauthRepo, rapidAPISvc, encKey)

	g.GET("/social/insights", h.SocialInsights)
	g.POST("/social/insights/refresh", h.RefreshSocial)
	g.GET("/social/history", h.SocialHistory)
	g.GET("/social/profile", h.PublicProfile)
}

func registerContentRoutes(g *gin.RouterGroup, projectRepo repository.ProjectRepository, insightRepo repository.InsightRepository, openai services.OpenAIService, cfg *config.Config) {
	h := handlers.NewContentHandler(projectRepo, insightRepo, openai, cfg)

	g.POST("/content/generate", h.Generate)
}

	g.POST("/content/generate", h.Generate)
}

func registerTaskRoutes(g *gin.RouterGroup, insightRepo repository.InsightRepository) {
	h := handlers.NewTaskHandler(insightRepo)

	g.PATCH("/tasks/:id/toggle", h.Toggle)
}

func registerSystemRoutes(g *gin.RouterGroup, projectRepo repository.ProjectRepository, taskRepo repository.TaskRepository) {
	h := handlers.NewSystemHandler(taskRepo, projectRepo)

	g.GET("/system/automations", h.GetAutomations)
	g.POST("/system/automations", h.CreateAutomation)
	g.PATCH("/system/automations/:id/toggle", h.ToggleAutomation)
	
	g.GET("/system/calendar", h.GetCalendar)
	g.POST("/system/calendar/event", h.CreateCalendarEvent)
}

func registerSyncRoutes(
func registerSyncRoutes(
	g *gin.RouterGroup,
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	seoRepo repository.SEORepository,
	insightRepo repository.InsightRepository,
	dataForSEOSvc services.DataForSEOService,
	rapidAPISvc services.RapidAPIService,
	crawler services.SEOCrawlerService,
	encKey []byte,
) {
	h := handlers.NewSyncHandler(projectRepo, metricRepo, oauthRepo, seoRepo, insightRepo, dataForSEOSvc, rapidAPISvc, crawler, encKey)
	g.POST("/projects/:id/sync", h.SyncProject)
}
