// Package workers contains the three background goroutines that keep data
// fresh without blocking HTTP request handling.
package workers

import (
	"context"
	"log"
	"time"

	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// StartMetricsSyncer runs every 6 hours and pulls GSC + Meta metrics for all
// projects that have connected OAuth credentials.
func StartMetricsSyncer(
	db *gorm.DB,
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	gsc services.GSCService,
	meta services.MetaService,
	encKey []byte,
	cfg *config.Config,
) {
	log.Println("[worker:MetricsSyncer] started — interval: 6h")
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// Run immediately on startup
	runMetricsSyncer(db, projectRepo, metricRepo, oauthRepo, gsc, meta, encKey, cfg)

	for range ticker.C {
		runMetricsSyncer(db, projectRepo, metricRepo, oauthRepo, gsc, meta, encKey, cfg)
	}
}

func runMetricsSyncer(
	_ *gorm.DB,
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	oauthRepo repository.OAuthRepository,
	gsc services.GSCService,
	meta services.MetaService,
	encKey []byte,
	_ *config.Config,
) {
	log.Println("[worker:MetricsSyncer] sync cycle started")

	// 1. Get all projects
	projects, err := projectRepo.FindAll()
	if err != nil {
		log.Printf("[worker:MetricsSyncer] failed to fetch projects: %v", err)
		return
	}

	for _, project := range projects {
		// 2. Check for Google Search Console credentials
		googleCred, err := oauthRepo.FindByUserAndProvider(project.UserID, "google")
		if err == nil && googleCred != nil {
			log.Printf("[worker:MetricsSyncer] syncing GSC for project %d (%s)", project.ID, project.URL)
			
			accessToken, err1 := utils.Decrypt(googleCred.AccessTokenEnc, encKey)
			refreshToken, err2 := utils.Decrypt(googleCred.RefreshTokenEnc, encKey)
			
			if err1 == nil && err2 == nil {
				token := &oauth2.Token{
					AccessToken:  accessToken,
					RefreshToken: refreshToken,
					Expiry:       googleCred.ExpiresAt,
				}
				metrics, err := gsc.FetchMetrics(context.Background(), project.URL, token)
				if err == nil {
					for _, m := range metrics {
						m.ProjectID = project.ID
						metricRepo.UpsertMetric(&m)
					}
				} else {
					log.Printf("[worker:MetricsSyncer] GSC sync error: %v", err)
				}
			}
		}

		// 3. Check for Meta (Instagram) credentials
		metaCred, err := oauthRepo.FindByUserAndProvider(project.UserID, "meta")
		if err == nil && metaCred != nil && project.IGHandle != "" {
			log.Printf("[worker:MetricsSyncer] syncing Meta for project %d", project.ID)
			accessToken, err := utils.Decrypt(metaCred.AccessTokenEnc, encKey)
			if err == nil {
				// For a real app, we'd need to find the specific IG User ID associated with the project's handle
				// For this Phase 3 implementation, we'll try to find all accounts and pick one or use a mock discovery
				accounts, err := meta.GetIGUserAccounts(accessToken)
				if err == nil && len(accounts) > 0 {
					var targetID string
					if id, ok := accounts[project.IGHandle]; ok {
						targetID = id
					} else {
						// Fallback: pick first available
						for _, id := range accounts {
							targetID = id
							break
						}
					}

					if targetID != "" {
						sm, err := meta.FetchInstagramMetrics(project.ID, targetID, accessToken)
						if err == nil {
							metricRepo.CreateSocialMetric(sm)
						}
					}
				}
			}
		}
	}

	log.Println("[worker:MetricsSyncer] sync cycle completed")
}

