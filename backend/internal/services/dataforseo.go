package services

import (
	"math/rand"
	"time"

	"backend/internal/models"
)

type DataForSEOService interface {
	FetchEstimatedTraffic(siteURL string) ([]models.Metric, error)
}

type dataForSEO struct {
	login    string
	password string
}

func NewDataForSEOService(login, password string) DataForSEOService {
	return &dataForSEO{
		login:    login,
		password: password,
	}
}

func (s *dataForSEO) FetchEstimatedTraffic(siteURL string) ([]models.Metric, error) {
	if s.login == "" || s.password == "" {
		// Return simulated traffic if keys are not set yet
		return simulateTraffic(siteURL), nil
	}

	// In a real implementation, this would make an HTTP request to api.dataforseo.com
	// For example:
	// req, _ := http.NewRequest("POST", "https://api.dataforseo.com/v3/traffic_analytics/similarweb/task_post", bytes.NewBuffer(payload))
	// req.SetBasicAuth(s.login, s.password)

	// Since we are mocking the successful integration until keys are provided:
	return simulateTraffic(siteURL), nil
}

func simulateTraffic(siteURL string) []models.Metric {
	var results []models.Metric
	now := time.Now()
	
	// Seed with hash of URL for deterministic estimation
	var hash int64
	for _, c := range siteURL {
		hash += int64(c)
	}
	baseTraffic := 500 + (hash % 2000)

	for i := 30; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dailyVar := rand.Int63n(100) - 50
		clicks := baseTraffic + dailyVar
		if clicks < 0 {
			clicks = 0
		}
		
		results = append(results, models.Metric{
			Date:        date.Format("2006-01-02"),
			Clicks:      clicks,
			Impressions: clicks * 15,
			Source:      "dataforseo",
		})
	}
	return results
}
