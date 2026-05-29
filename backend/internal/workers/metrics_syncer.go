// Package workers contains the three background goroutines that keep data
// fresh without blocking HTTP request handling.
package workers

import (
	"context"
	"log"
	"strings"
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
	cfg *config.Config,
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

		// 3. Check for Meta (Instagram/Facebook) credentials
		accessToken := cfg.MetaPageAccessToken
		if metaCred, err := oauthRepo.FindByUserAndProvider(project.UserID, "meta"); err == nil && metaCred != nil {
			if token, decErr := utils.Decrypt(metaCred.AccessTokenEnc, encKey); decErr == nil && token != "" {
				accessToken = token
			}
		}
		if accessToken != "" && (project.IGHandle != "" || project.FBHandle != "") {
			log.Printf("[worker:MetricsSyncer] syncing Meta for project %d", project.ID)
			if project.IGHandle != "" {
				accounts, err := meta.GetIGUserAccounts(accessToken)
				if err == nil && len(accounts) > 0 {
					var targetID string
					var targetToken string
					normIG := strings.ToLower(strings.TrimSpace(project.IGHandle))
					for _, acc := range accounts {
						if acc.Username == normIG || acc.Name == normIG {
							targetID = acc.ID
							targetToken = acc.AccessToken
							break
						}
					}
					if targetID == "" {
						targetID = accounts[0].ID
						targetToken = accounts[0].AccessToken
					}

					if targetID != "" {
						if targetToken == "" {
							targetToken = accessToken
						}
						sm, err := meta.FetchInstagramMetrics(project.ID, targetID, targetToken)
						if err == nil {
							metricRepo.CreateSocialMetric(sm)
						}
					}
				}
			}

			if project.FBHandle != "" {
				accounts, err := meta.GetFacebookPageAccounts(accessToken)
				if err == nil && len(accounts) > 0 {
					var targetID string
					var targetToken string
					normFB := strings.ToLower(strings.TrimSpace(project.FBHandle))
					for _, acc := range accounts {
						if acc.Username == normFB || acc.Name == normFB {
							targetID = acc.ID
							targetToken = acc.AccessToken
							break
						}
					}
					if targetID == "" {
						targetID = accounts[0].ID
						targetToken = accounts[0].AccessToken
					}

					if targetID != "" {
						if targetToken == "" {
							targetToken = accessToken
						}
						sm, err := meta.FetchFacebookPageMetrics(project.ID, targetID, targetToken)
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

