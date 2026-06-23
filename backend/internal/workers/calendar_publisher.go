// Package workers publishes scheduled Instagram content when its due date arrives.
package workers

import (
	"log"
	"os"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"
)

// StartCalendarPublisher checks for due scheduled tasks and publishes them to
// Instagram using the Meta Graph API.
func StartCalendarPublisher(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	oauthRepo repository.OAuthRepository,
	meta services.MetaService,
	linkedin services.LinkedinService,
	encKey []byte,
	cfg *config.Config,
) {
	log.Println("[worker:CalendarPublisher] started — interval: 1m")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	runCalendarPublisher(taskRepo, projectRepo, oauthRepo, meta, linkedin, encKey, cfg)

	for range ticker.C {
		runCalendarPublisher(taskRepo, projectRepo, oauthRepo, meta, linkedin, encKey, cfg)
	}
}

func runCalendarPublisher(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	oauthRepo repository.OAuthRepository,
	meta services.MetaService,
	linkedin services.LinkedinService,
	encKey []byte,
	cfg *config.Config,
) {
	log.Println("[worker:CalendarPublisher] scanning due tasks...")

	projects, err := projectRepo.FindAll()
	if err != nil {
		log.Printf("[worker:CalendarPublisher] failed to load projects: %v", err)
		return
	}

	projectByID := make(map[uint]struct {
		UserID         uint
		IGHandle       string
		FBHandle       string
		LinkedinHandle string
	})
	for _, project := range projects {
		projectByID[project.ID] = struct {
			UserID         uint
			IGHandle       string
			FBHandle       string
			LinkedinHandle string
		}{UserID: project.UserID, IGHandle: project.IGHandle, FBHandle: project.FBHandle, LinkedinHandle: project.LinkedinHandle}
	}

	tasks, err := taskRepo.FindDueScheduledTasks(time.Now().UTC())
	if err != nil {
		log.Printf("[worker:CalendarPublisher] failed to load scheduled tasks: %v", err)
		return
	}

	for _, task := range tasks {
		project, ok := projectByID[task.ProjectID]
		if !ok {
			continue
		}

		platform := strings.ToLower(strings.TrimSpace(task.Platform))
		if platform != "instagram" && platform != "facebook" && platform != "linkedin" {
			continue
		}

		accessToken := cfg.MetaPageAccessToken
		if metaCred, err := oauthRepo.FindByUserAndProvider(project.UserID, "meta"); err == nil && metaCred != nil {
			if token, decErr := utils.Decrypt(metaCred.AccessTokenEnc, encKey); decErr == nil && token != "" {
				accessToken = token
			}
		}
		if accessToken == "" {
			markPublishFailure(taskRepo, &task, "no meta access token available")
			continue
		}

		var publishedID string
		var pubErr error

		if platform == "instagram" {
			if strings.TrimSpace(project.IGHandle) == "" {
				markPublishFailure(taskRepo, &task, "instagram handle is missing for this project")
				continue
			}
			if strings.TrimSpace(task.AssetURL) == "" {
				markPublishFailure(taskRepo, &task, "missing uploaded media url")
				continue
			}

			accounts, err := meta.GetIGUserAccounts(accessToken)
			if err != nil || len(accounts) == 0 {
				markPublishFailure(taskRepo, &task, "failed to resolve instagram account")
				continue
			}

			targetID, targetToken := resolveIGTargetAccount(accounts, project.IGHandle)
			if targetID == "" {
				markPublishFailure(taskRepo, &task, "instagram account could not be resolved")
				continue
			}
			if targetToken == "" {
				targetToken = accessToken
			}

			publishedID, pubErr = meta.PublishInstagramContent(targetID, targetToken, &task)
		} else if platform == "facebook" {
			if strings.TrimSpace(project.FBHandle) == "" {
				markPublishFailure(taskRepo, &task, "facebook handle is missing for this project")
				continue
			}
			if strings.TrimSpace(task.AssetURL) == "" {
				markPublishFailure(taskRepo, &task, "missing uploaded media url")
				continue
			}

			accounts, err := meta.GetFacebookPageAccounts(accessToken)
			if err != nil || len(accounts) == 0 {
				markPublishFailure(taskRepo, &task, "failed to resolve facebook account")
				continue
			}

			targetID, targetToken := resolveIGTargetAccount(accounts, project.FBHandle)
			if targetID == "" {
				markPublishFailure(taskRepo, &task, "facebook page could not be resolved")
				continue
			}
			if targetToken == "" {
				targetToken = accessToken
			}

			publishedID, pubErr = meta.PublishFacebookContent(targetID, targetToken, &task)
		} else if platform == "linkedin" {
			linkedinAccessToken := os.Getenv("LINKEDIN_ACCESS_TOKEN")
			if linkedinAccessToken == "" {
				markPublishFailure(taskRepo, &task, "linkedin access token is missing")
				continue
			}

			publishedID, pubErr = linkedin.PublishLinkedInContent(project.LinkedinHandle, linkedinAccessToken, &task)
		}

		if pubErr != nil {
			markPublishFailure(taskRepo, &task, pubErr.Error())
			continue
		}

		now := time.Now().UTC()
		task.Completed = true
		task.PublishStatus = "published"
		task.PublishError = ""
		task.PublishedAt = &now
		if err := taskRepo.Update(&task); err != nil {
			log.Printf("[worker:CalendarPublisher] published %d but failed to update task: %v", task.ID, err)
			continue
		}

		log.Printf("[worker:CalendarPublisher] published task %d as media %s on %s", task.ID, publishedID, platform)
	}
}

func markPublishFailure(taskRepo repository.TaskRepository, task *models.Task, message string) {
	task.PublishStatus = "failed"
	task.PublishError = message
	if err := taskRepo.Update(task); err != nil {
		log.Printf("[worker:CalendarPublisher] failed to mark task %d as failed: %v", task.ID, err)
	}
	log.Printf("[worker:CalendarPublisher] task %d failed: %s", task.ID, message)
}

func resolveIGTargetAccount(accounts []services.MetaAccount, handle string) (string, string) {
	if len(accounts) == 0 {
		return "", ""
	}
	norm := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(handle, "@")))
	if norm != "" {
		for _, acc := range accounts {
			if acc.Username == norm || acc.Name == norm {
				return acc.ID, acc.AccessToken
			}
		}
	}
	return accounts[0].ID, accounts[0].AccessToken
}
