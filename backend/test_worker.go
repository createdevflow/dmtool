package main

import (
	"log"
	"os"

	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/workers"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	cfg := config.LoadConfig()

	db, err := gorm.Open(sqlite.Open("dmtool.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	taskRepo := repository.NewTaskRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	oauthRepo := repository.NewOAuthRepository(db)
	metaService := services.NewMetaService()

	encKey := []byte(os.Getenv("ENCRYPTION_KEY"))
	if len(encKey) != 32 {
		encKey = []byte("01234567890123456789012345678901")
	}

	// I will just copy the runCalendarPublisher logic here to avoid importing an unexported function
	workers.StartCalendarPublisher(taskRepo, projectRepo, oauthRepo, metaService, encKey, cfg)
}
