package main

import (
	"fmt"
	"time"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"
	"backend/internal/workers"
)

func main() {
	cfg := config.Load()
	database := db.Init("dmtool.db", false)
	taskRepo := repository.NewTaskRepository(database)
	projectRepo := repository.NewProjectRepository(database)
	oauthRepo := repository.NewOAuthRepository(database)
	metaService := services.NewMetaService()
	encKey := utils.EncryptionKeyFromString(cfg.EncryptionKey)

	linkedinService := services.NewLinkedinService()
	fmt.Println("Running publisher manually...")
	go workers.StartCalendarPublisher(taskRepo, projectRepo, oauthRepo, metaService, linkedinService, encKey, cfg)
	time.Sleep(5 * time.Second)
}
