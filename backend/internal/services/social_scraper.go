// Package services provides SocialScraperService — fetches public Instagram
// profile data without requiring OAuth. Works for public profiles only.
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// PublicProfileData holds scraped social profile info.
type PublicProfileData struct {
	Platform    string `json:"platform"`
	Username    string `json:"username"`
	FullName    string `json:"full_name"`
	Bio         string `json:"bio"`
	Followers   int64  `json:"followers"`
	Following   int64  `json:"following"`
	PostCount   int64  `json:"post_count"`
	ProfilePic  string `json:"profile_pic_url"`
	IsVerified  bool   `json:"is_verified"`
	IsPrivate   bool   `json:"is_private"`
	IsSimulated bool   `json:"is_simulated"` // true = could not fetch, data estimated
	FetchedAt   time.Time `json:"fetched_at"`
}

// SocialScraperService fetches public social profile data.
type SocialScraperService interface {
	FetchInstagramPublic(username string) (*PublicProfileData, error)
}

type socialScraper struct {
	client *http.Client
}

// NewSocialScraperService returns a new social scraper.
func NewSocialScraperService() SocialScraperService {
	return &socialScraper{
		client: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

// FetchInstagramPublic fetches public Instagram profile data via Instagram's
// unofficial JSON API endpoint. Falls back to HTML scraping if JSON fails.
func (s *socialScraper) FetchInstagramPublic(username string) (*PublicProfileData, error) {
	username = strings.TrimPrefix(username, "@")
	username = strings.TrimSpace(username)

	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	profile := &PublicProfileData{
		Platform:  "Instagram",
		Username:  username,
		FetchedAt: time.Now(),
	}

	// Method 1: Try Instagram's public JSON endpoint
	if data, err := s.fetchInstagramJSON(username); err == nil {
		return data, nil
	}

	// Method 2: Try scraping the HTML profile page
	if data, err := s.scrapeInstagramHTML(username); err == nil {
		return data, nil
	}

	// Method 3: Fallback — use deterministic hash-based estimation (clearly marked simulated)
	h := hashUsername(username)
	profile.Followers = int64(1000 + (h % 50000))
	profile.Following = int64(200 + (h % 2000))
	profile.PostCount = int64(10 + (h % 500))
	profile.IsSimulated = true
	profile.FullName = username

	return profile, nil
}

// fetchInstagramJSON tries Instagram's public graph JSON endpoint.
func (s *socialScraper) fetchInstagramJSON(username string) (*PublicProfileData, error) {
	apiURL := fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	// Required headers to bypass bot detection
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-IG-App-ID", "936619743392459")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://www.instagram.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("instagram returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			User struct {
				Biography       string `json:"biography"`
				FollowedBy      struct{ Count int64 `json:"count"` } `json:"edge_followed_by"`
				Follow          struct{ Count int64 `json:"count"` } `json:"edge_follow"`
				Media           struct{ Count int64 `json:"count"` } `json:"edge_owner_to_timeline_media"`
				FullName        string `json:"full_name"`
				IsPrivate       bool   `json:"is_private"`
				IsVerified      bool   `json:"is_verified"`
				ProfilePicURL   string `json:"profile_pic_url_hd"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	u := result.Data.User
	if u.FullName == "" && u.FollowedBy.Count == 0 {
		return nil, fmt.Errorf("empty user data")
	}

	return &PublicProfileData{
		Platform:    "Instagram",
		Username:    username,
		FullName:    u.FullName,
		Bio:         u.Biography,
		Followers:   u.FollowedBy.Count,
		Following:   u.Follow.Count,
		PostCount:   u.Media.Count,
		ProfilePic:  u.ProfilePicURL,
		IsVerified:  u.IsVerified,
		IsPrivate:   u.IsPrivate,
		IsSimulated: false,
		FetchedAt:   time.Now(),
	}, nil
}

// scrapeInstagramHTML parses embedded JSON from Instagram HTML page.
func (s *socialScraper) scrapeInstagramHTML(username string) (*PublicProfileData, error) {
	pageURL := fmt.Sprintf("https://www.instagram.com/%s/", username)

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DMTool/2.0)")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return nil, err
	}

	bodyStr := string(body)

	// Extract follower count from page source using regex
	followerRe := regexp.MustCompile(`"edge_followed_by":\{"count":(\d+)\}`)
	followingRe := regexp.MustCompile(`"edge_follow":\{"count":(\d+)\}`)
	nameRe := regexp.MustCompile(`"full_name":"([^"]+)"`)

	var followers, following int64
	var fullName string

	if m := followerRe.FindStringSubmatch(bodyStr); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &followers)
	}
	if m := followingRe.FindStringSubmatch(bodyStr); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &following)
	}
	if m := nameRe.FindStringSubmatch(bodyStr); len(m) > 1 {
		fullName = m[1]
	}

	if followers == 0 && fullName == "" {
		return nil, fmt.Errorf("could not extract data from HTML")
	}

	return &PublicProfileData{
		Platform:    "Instagram",
		Username:    username,
		FullName:    fullName,
		Followers:   followers,
		Following:   following,
		IsSimulated: false,
		FetchedAt:   time.Now(),
	}, nil
}

// hashUsername returns a deterministic uint64 from a username string.
func hashUsername(s string) uint64 {
	var h uint64 = 5381
	for _, c := range s {
		h = ((h << 5) + h) + uint64(c)
	}
	return h
}
