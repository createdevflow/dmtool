package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"backend/internal/services"
)

func main() {
	token := os.Getenv("META_PAGE_ACCESS_TOKEN")
	if token == "" {
		log.Fatal("META_PAGE_ACCESS_TOKEN not set")
	}

	metaSvc := services.NewMetaService()
	
	fmt.Println("--- Syncing Instagram ---")
	igMetrics, err := metaSvc.FetchInstagramMetrics(11, "capturedbyaaryan", token)
	if err != nil {
		fmt.Println("IG Error:", err)
	} else {
		b, _ := json.MarshalIndent(igMetrics, "", "  ")
		fmt.Println(string(b))
	}

	fmt.Println("\n--- Syncing Facebook ---")
	fbMetrics, err := metaSvc.FetchFacebookPageMetrics(12, "1097587700111919", token)
	if err != nil {
		fmt.Println("FB Error:", err)
	} else {
		b, _ := json.MarshalIndent(fbMetrics, "", "  ")
		fmt.Println(string(b))
	}
}
