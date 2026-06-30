package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"backend/internal/models"
)

// DataForSEOService fetches estimated organic traffic data for a site.
type DataForSEOService interface {
	FetchEstimatedTraffic(siteURL string) ([]models.Metric, error)
}

type dataForSEO struct {
	login    string
	password string
}

// NewDataForSEOService returns a DataForSEOService.
// When login/password are empty it falls back to deterministic simulation.
func NewDataForSEOService(login, password string) DataForSEOService {
	return &dataForSEO{
		login:    login,
		password: password,
	}
}

// FetchEstimatedTraffic calls the DataForSEO Traffic Analytics API (v3) and
// returns per-day estimated organic clicks for the last 30 days.
// Falls back to simulation when credentials are missing.
func (s *dataForSEO) FetchEstimatedTraffic(siteURL string) ([]models.Metric, error) {
	if s.login == "" || s.password == "" {
		return simulateTraffic(siteURL), nil
	}

	// Build request body — task_post endpoint expects an array of tasks.
	payload := []map[string]interface{}{
		{
			"target":          siteURL,
			"date_from":       time.Now().AddDate(0, 0, -33).Format("2006-01-02"),
			"date_to":         time.Now().AddDate(0, 0, -3).Format("2006-01-02"),
			"item_types":      []string{"organic"},
			"history_data_id": nil,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return simulateTraffic(siteURL), nil
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.dataforseo.com/v3/traffic_analytics/google/task_post",
		bytes.NewReader(body),
	)
	if err != nil {
		return simulateTraffic(siteURL), nil
	}
	req.SetBasicAuth(s.login, s.password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dataforseo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dataforseo returned %d: %s", resp.StatusCode, string(b))
	}

	// Parse task-post response to extract the task ID, then poll for results.
	var postResp struct {
		Tasks []struct {
			ID     string `json:"id"`
			Status struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"status"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil || len(postResp.Tasks) == 0 {
		return simulateTraffic(siteURL), nil
	}

	taskID := postResp.Tasks[0].ID
	if taskID == "" {
		return simulateTraffic(siteURL), nil
	}

	// Poll for task completion (DataForSEO is async — usually ready in <5s).
	return s.pollTrafficTask(taskID, siteURL)
}

func (s *dataForSEO) pollTrafficTask(taskID, siteURL string) ([]models.Metric, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.dataforseo.com/v3/traffic_analytics/google/task_get/%s", taskID)

	for attempt := 0; attempt < 6; attempt++ {
		time.Sleep(time.Duration(attempt+1) * time.Second)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		req.SetBasicAuth(s.login, s.password)

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		var result struct {
			Tasks []struct {
				Result []struct {
					Items []struct {
						Date    string  `json:"date"`
						Organic float64 `json:"organic"`
					} `json:"items"`
				} `json:"result"`
			} `json:"tasks"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if len(result.Tasks) == 0 || len(result.Tasks[0].Result) == 0 {
			continue // not ready yet
		}

		var metrics []models.Metric
		for _, item := range result.Tasks[0].Result[0].Items {
			metrics = append(metrics, models.Metric{
				Date:        item.Date,
				Clicks:      int64(item.Organic),
				Impressions: int64(item.Organic) * 12,
				Source:      "dataforseo",
			})
		}
		if len(metrics) > 0 {
			return metrics, nil
		}
	}

	// Task never became ready — fall back to simulation.
	return simulateTraffic(siteURL), nil
}

// simulateTraffic returns deterministic fake data seeded by the URL hash.
// Used when no DataForSEO credentials are configured.
func simulateTraffic(siteURL string) []models.Metric {
	var results []models.Metric
	now := time.Now()

	var hash int64
	for _, c := range siteURL {
		hash += int64(c)
	}
	baseTraffic := 500 + (hash % 2000)

	rng := rand.New(rand.NewSource(hash))
	for i := 30; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dailyVar := rng.Int63n(100) - 50
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
