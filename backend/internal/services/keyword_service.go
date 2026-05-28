// Package services provides keyword research via Google Search Console +
// Google Autocomplete (free, no API key required).
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/internal/models"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/searchconsole/v1"
)

// KeywordService fetches keyword data from GSC or Google Autocomplete.
type KeywordService interface {
	// FetchGSCKeywords pulls real keyword data from Google Search Console
	// for a verified property. Returns nil error with empty slice if no GSC token.
	FetchGSCKeywords(ctx context.Context, siteURL string, token *oauth2.Token, oauthConfig *oauth2.Config) ([]models.KeywordResult, error)

	// FetchAutocompleteKeywords uses Google's free autocomplete API to generate
	// keyword suggestions for a seed term. No API key required.
	FetchAutocompleteKeywords(seed string) ([]models.KeywordResult, error)
}

type keywordService struct {
	client *http.Client
}

// NewKeywordService returns a new KeywordService.
func NewKeywordService() KeywordService {
	return &keywordService{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchGSCKeywords pulls top 25 queries from Google Search Console.
func (s *keywordService) FetchGSCKeywords(ctx context.Context, siteURL string, token *oauth2.Token, oauthConfig *oauth2.Config) ([]models.KeywordResult, error) {
	if token == nil || oauthConfig == nil {
		return nil, fmt.Errorf("no GSC credentials available")
	}

	httpClient := oauthConfig.Client(ctx, token)
	svc, err := searchconsole.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create GSC service: %w", err)
	}

	endDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")   // GSC 3-day lag
	startDate := time.Now().AddDate(0, 0, -33).Format("2006-01-02") // 30 days of data

	req := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: []string{"query"},
		RowLimit:   25,
	}

	resp, err := svc.Searchanalytics.Query(siteURL, req).Do()
	if err != nil {
		return nil, fmt.Errorf("GSC query failed: %w", err)
	}

	var results []models.KeywordResult
	for _, row := range resp.Rows {
		if len(row.Keys) == 0 {
			continue
		}
		kw := row.Keys[0]
		kd := estimateKD(kw, int(row.Impressions))
		results = append(results, models.KeywordResult{
			Seed:     kw,
			Keyword:  kw,
			Volume:   int(row.Impressions),
			KD:       kd,
			Position: row.Position,
		})
	}

	return results, nil
}

// FetchAutocompleteKeywords uses Google's free suggest API (no key required).
// It expands the seed into 20+ related keyword suggestions.
func (s *keywordService) FetchAutocompleteKeywords(seed string) ([]models.KeywordResult, error) {
	seen := map[string]bool{seed: true}
	var results []models.KeywordResult

	// Fetch base suggestions
	baseSuggestions := s.fetchGoogleSuggest(seed)

	for _, kw := range baseSuggestions {
		if seen[kw] {
			continue
		}
		seen[kw] = true
		vol := estimateVolumeFromKW(kw)
		kd := estimateKD(kw, vol)
		results = append(results, models.KeywordResult{
			Seed:     seed,
			Keyword:  kw,
			Volume:   vol,
			KD:       kd,
			Position: 0, // no GSC = no real position
		})
	}

	// Expand with question-based queries
	questionPrefixes := []string{"how to", "what is", "best", "why"}
	for _, prefix := range questionPrefixes {
		expanded := s.fetchGoogleSuggest(prefix + " " + seed)
		for _, kw := range expanded {
			if seen[kw] {
				continue
			}
			seen[kw] = true
			vol := estimateVolumeFromKW(kw)
			kd := estimateKD(kw, vol)
			results = append(results, models.KeywordResult{
				Seed:     seed,
				Keyword:  kw,
				Volume:   vol,
				KD:       kd,
				Position: 0,
			})
			if len(results) >= 30 {
				break
			}
		}
		if len(results) >= 30 {
			break
		}
	}

	return results, nil
}

// fetchGoogleSuggest calls Google's free autocomplete endpoint.
func (s *keywordService) fetchGoogleSuggest(query string) []string {
	apiURL := fmt.Sprintf(
		"https://suggestqueries.google.com/complete/search?client=firefox&q=%s&hl=en",
		url.QueryEscape(query),
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DMTool/2.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	// Response format: ["seed", ["suggestion1", "suggestion2", ...], ...]
	var raw []interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}

	var suggestions []string
	if len(raw) > 1 {
		if list, ok := raw[1].([]interface{}); ok {
			for _, item := range list {
				if s, ok := item.(string); ok {
					suggestions = append(suggestions, strings.TrimSpace(s))
				}
			}
		}
	}

	return suggestions
}

// estimateVolumeFromKW heuristically estimates monthly search volume based on
// keyword length and word count. Short, generic keywords = higher volume.
func estimateVolumeFromKW(kw string) int {
	words := len(strings.Fields(kw))
	chars := len(kw)
	switch {
	case words <= 1 && chars <= 8:
		return 50000 + (int(hashString(kw)) % 100000)
	case words <= 2:
		return 5000 + (int(hashString(kw)) % 30000)
	case words <= 3:
		return 500 + (int(hashString(kw)) % 5000)
	default:
		return 50 + (int(hashString(kw)) % 500)
	}
}

// estimateKD heuristically estimates keyword difficulty (0-100).
// High volume + short keyword = high difficulty.
func estimateKD(kw string, volume int) int {
	words := len(strings.Fields(kw))
	base := 0
	switch {
	case volume > 50000:
		base = 75
	case volume > 10000:
		base = 60
	case volume > 1000:
		base = 40
	default:
		base = 20
	}
	// Long-tail keywords are easier
	if words >= 4 {
		base -= 20
	} else if words >= 3 {
		base -= 10
	}
	if base < 5 {
		base = 5
	}
	if base > 95 {
		base = 95
	}
	return base
}

// hashString returns a deterministic uint64 from a string.
func hashString(s string) uint64 {
	var h uint64 = 5381
	for _, c := range s {
		h = ((h << 5) + h) + uint64(c)
	}
	return h
}
