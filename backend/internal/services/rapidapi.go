package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"backend/internal/models"
)

// RapidAPIService fetches Instagram profile data.
type RapidAPIService interface {
	FetchInstagramProfile(username string) (*models.SocialMetric, error)
}

type rapidAPI struct {
	apiKey string
}

// NewRapidAPIService returns a RapidAPIService.
// When apiKey is empty it returns deterministic simulation.
func NewRapidAPIService(apiKey string) RapidAPIService {
	return &rapidAPI{apiKey: apiKey}
}

// FetchInstagramProfile calls the Instagram Data API on RapidAPI to fetch
// public profile statistics.  Falls back to simulation when apiKey is empty.
func (s *rapidAPI) FetchInstagramProfile(username string) (*models.SocialMetric, error) {
	if s.apiKey == "" {
		return simulateInstagramProfile(username), nil
	}

	// Endpoint: instagram-data1.p.rapidapi.com
	url := fmt.Sprintf("https://instagram-data1.p.rapidapi.com/user/info?username=%s", username)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return simulateInstagramProfile(username), nil
	}
	req.Header.Set("X-RapidAPI-Key", s.apiKey)
	req.Header.Set("X-RapidAPI-Host", "instagram-data1.p.rapidapi.com")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rapidapi request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rapidapi returned %d: %s", resp.StatusCode, string(b))
	}

	// The instagram-data1 API returns a flat JSON object.
	var profile struct {
		FollowerCount  int64  `json:"follower_count"`
		FollowingCount int64  `json:"following_count"`
		MediaCount     int64  `json:"media_count"`
		FullName       string `json:"full_name"`
		Biography      string `json:"biography"`
		ProfilePicURL  string `json:"profile_pic_url"`
		IsPrivate      bool   `json:"is_private"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("rapidapi decode error: %w", err)
	}

	if profile.FollowerCount == 0 {
		// Empty / private profile — fall back to simulation.
		return simulateInstagramProfile(username), nil
	}

	return &models.SocialMetric{
		Platform:          "Instagram",
		DisplayName:       profile.FullName,
		Biography:         profile.Biography,
		ProfilePictureURL: profile.ProfilePicURL,
		Followers:         profile.FollowerCount,
		FollowingCount:    profile.FollowingCount,
		PostsCount:        profile.MediaCount,
		// Reach is estimated as ~15% of followers for public accounts.
		Reach:           int64(float64(profile.FollowerCount) * 0.15),
		Engagement:      3.5,
		EngagementCount: int64(float64(profile.FollowerCount) * 0.035),
		Status:          "stable",
		IsSimulated:     false,
		RecordedAt:      time.Now(),
	}, nil
}

// simulateInstagramProfile returns deterministic fake data seeded by username.
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
