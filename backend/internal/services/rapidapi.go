package services

import (
	"time"
	"backend/internal/models"
)

type RapidAPIService interface {
	FetchInstagramProfile(username string) (*models.SocialMetric, error)
}

type rapidAPI struct {
	apiKey string
}

func NewRapidAPIService(apiKey string) RapidAPIService {
	return &rapidAPI{apiKey: apiKey}
}

func (s *rapidAPI) FetchInstagramProfile(username string) (*models.SocialMetric, error) {
	if s.apiKey == "" {
		return simulateInstagramProfile(username), nil
	}

	// Real implementation would call rapid API endpoint
	// e.g., https://instagram-data1.p.rapidapi.com/user/info?username=...
	
	return simulateInstagramProfile(username), nil
}

func simulateInstagramProfile(username string) *models.SocialMetric {
	var hash int64
	for _, c := range username {
		hash += int64(c)
	}
	
	followers := int64(5000 + (hash % 100000))
	
	return &models.SocialMetric{
		Platform:        "Instagram",
		Followers:       followers,
		Reach:           int64(float64(followers) * 0.15),
		Engagement:      4.2 + (float64(hash%10) * 0.1),
		EngagementCount: int64(float64(followers) * 0.042),
		Status:          "stable",
		IsSimulated:     true,
		RecordedAt:      time.Now(),
	}
}
