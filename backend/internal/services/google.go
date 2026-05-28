package services

import (
	"context"
	"fmt"
	"time"

	"backend/internal/models"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/searchconsole/v1"
)

// GSCService handles interactions with Google Search Console API.
type GSCService interface {
	FetchMetrics(ctx context.Context, siteURL string, token *oauth2.Token) ([]models.Metric, error)
	// OAuthConfig exposes the underlying config so other services can use it for
	// keyword queries with the same credentials.
	OAuthConfig() *oauth2.Config
}

type gscService struct {
	oauthConfig *oauth2.Config
}

func NewGSCService(oauthConfig *oauth2.Config) GSCService {
	return &gscService{oauthConfig: oauthConfig}
}

func (s *gscService) OAuthConfig() *oauth2.Config {
	return s.oauthConfig
}

func (s *gscService) FetchMetrics(ctx context.Context, siteURL string, token *oauth2.Token) ([]models.Metric, error) {
	client := s.oauthConfig.Client(ctx, token)
	svc, err := searchconsole.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create GSC service: %w", err)
	}

	// Fetch last 30 days (GSC has a 2-3 day lag, so we start from 4 days ago)
	endDate := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -33).Format("2006-01-02")

	req := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: []string{"date"},
		RowLimit:   30,
	}

	resp, err := svc.Searchanalytics.Query(siteURL, req).Do()
	if err != nil {
		return nil, fmt.Errorf("GSC query failed: %w", err)
	}

	var results []models.Metric
	for _, row := range resp.Rows {
		if len(row.Keys) == 0 {
			continue
		}
		date, _ := time.Parse("2006-01-02", row.Keys[0])

		results = append(results, models.Metric{
			Date:        date.Format("2006-01-02"),
			Clicks:      int64(row.Clicks),
			Impressions: int64(row.Impressions),
			Source:      "gsc",
		})
	}

	return results, nil
}
