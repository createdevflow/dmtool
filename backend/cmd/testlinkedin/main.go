package main

import (
	"fmt"
	"os"

	"backend/internal/services"
	"backend/internal/models"
)

func main() {
	token := os.Getenv("LINKEDIN_ACCESS_TOKEN")
	if token == "" {
		fmt.Println("No LINKEDIN_ACCESS_TOKEN provided. Cannot test.")
		return
	}

	li := services.NewLinkedinService()
	sm, err := li.FetchLinkedInMetrics(1, "test", token)
	if err != nil {
		fmt.Printf("FetchLinkedInMetrics error: %v\n", err)
	} else {
		fmt.Printf("Metrics: %+v\n", sm)
	}

	// Just a mock publish
	taskId, err := li.PublishLinkedInContent("test", token, &models.Task{
		Title:   "Test LinkedIn Post",
		Caption: "This is a test post from dmtool local testing.",
	})
	if err != nil {
		fmt.Printf("PublishLinkedInContent error: %v\n", err)
	} else {
		fmt.Printf("Publish success, ID: %s\n", taskId)
	}
}
