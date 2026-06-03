package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"backend/internal/services"
	"backend/internal/models"
)

func main() {
	token := os.Getenv("META_PAGE_ACCESS_TOKEN")
	if token == "" {
		log.Fatal("META_PAGE_ACCESS_TOKEN not set")
	}

	metaSvc := services.NewMetaService()

	// Task 104 data (from the validation test logs)
	task := &models.Task{
		ContentType: "story",
		AssetURL:    "https://storage.googleapis.com/gtv-videos-bucket/sample/ForBiggerMeltdowns.mp4",
		AssetMime:   "video/mp4",
		AssetName:   "ForBiggerMeltdowns.mp4",
		Caption:     "TEST VIDEO STORY - FB",
	}

	pageToken := token
	resp, err := http.Get(fmt.Sprintf("https://graph.facebook.com/v19.0/me/accounts?access_token=%s", token))
	if err == nil {
		defer resp.Body.Close()
		var data struct {
			Data []struct {
				ID          string `json:"id"`
				AccessToken string `json:"access_token"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&data)
		if len(data.Data) > 0 {
			pageToken = data.Data[0].AccessToken
		}
	}

	fmt.Println("--- Publishing Facebook Video Story ---")
	id, err := metaSvc.PublishFacebookContent("1097587700111919", pageToken, task)
	if err != nil {
		fmt.Println("FB Error:", err)
	} else {
		fmt.Println("FB Success! ID:", id)
	}
}
