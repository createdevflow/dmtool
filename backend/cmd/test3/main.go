package main

import (
	"fmt"
	"strings"
	"time"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/repository"
)

func main() {
	cfg := config.Load()
	database := db.Init("dmtool.db", false)
	taskRepo := repository.NewTaskRepository(database)
	projectRepo := repository.NewProjectRepository(database)

	tasks, _ := taskRepo.FindDueScheduledTasks(time.Now().UTC())
	projects, _ := projectRepo.FindAll()

	projectByID := make(map[uint]struct{ UserID uint; IGHandle string })
	for _, p := range projects {
		projectByID[p.ID] = struct{ UserID uint; IGHandle string }{p.UserID, p.IGHandle}
	}

	for _, task := range tasks {
		if task.ID != 79 { continue }
		fmt.Printf("Task 79 processing...\n")
		
		project, ok := projectByID[task.ProjectID]
		if !ok {
			fmt.Println("Skipped: project not found", task.ProjectID)
			continue
		}

		if strings.ToLower(strings.TrimSpace(task.Platform)) != "instagram" {
			fmt.Println("Skipped: platform not instagram:", task.Platform)
			continue
		}
		if strings.TrimSpace(project.IGHandle) == "" {
			fmt.Println("Skipped: ig handle empty")
			continue
		}
		if strings.TrimSpace(task.AssetURL) == "" {
			fmt.Println("Skipped: asset url empty")
			continue
		}

		accessToken := cfg.MetaPageAccessToken
		fmt.Printf("Access token: %s...\n", accessToken[:10])
		fmt.Println("Made it to Meta call!")
	}
}
