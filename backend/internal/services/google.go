package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"backend/internal/models"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/searchconsole/v1"
)

// GSCService handles interactions with Google Search Console API.
type GSCService interface {
	FetchMetrics(ctx context.Context, siteURL string, token *oauth2.Token) ([]models.Metric, error)
	FetchBreakdowns(ctx context.Context, siteURL string, token *oauth2.Token) (GSCBreakdownSet, error)
	// OAuthConfig exposes the underlying config so other services can use it for
	// keyword queries with the same credentials.
	OAuthConfig() *oauth2.Config
}

// GSCBreakdownSet is one GSC Search Analytics window split by dimension.
// GotX is true only when that dimension query succeeded (including zero rows).
type GSCBreakdownSet struct {
	StartDate     string
	EndDate       string
	Pages         []models.GSCBreakdown
	Countries     []models.GSCBreakdown
	Devices       []models.GSCBreakdown
	Appearance    []models.GSCBreakdown
	GotPages      bool
	GotCountries  bool
	GotDevices    bool
	GotAppearance bool
}

// GSCDimBatch is one successfully fetched dimension ready to persist.
type GSCDimBatch struct {
	Dimension string
	Rows      []models.GSCBreakdown
}

func (s GSCBreakdownSet) FetchedBatches() []GSCDimBatch {
	out := make([]GSCDimBatch, 0, 4)
	if s.GotPages {
		out = append(out, GSCDimBatch{Dimension: models.GSCDimPage, Rows: s.Pages})
	}
	if s.GotCountries {
		out = append(out, GSCDimBatch{Dimension: models.GSCDimCountry, Rows: s.Countries})
	}
	if s.GotDevices {
		out = append(out, GSCDimBatch{Dimension: models.GSCDimDevice, Rows: s.Devices})
	}
	if s.GotAppearance {
		out = append(out, GSCDimBatch{Dimension: models.GSCDimSearchAppearance, Rows: s.Appearance})
	}
	return out
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

	startDate, endDate := gscDateWindow()

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

func gscDateWindow() (start, end string) {
	now := time.Now().UTC()
	end = now.AddDate(0, 0, -3).Format("2006-01-02")
	start = now.AddDate(0, 0, -33).Format("2006-01-02")
	return start, end
}

// FetchBreakdowns pulls page, country, device, and searchAppearance rows
// for the same 30-day GSC window as FetchMetrics.
func (s *gscService) FetchBreakdowns(ctx context.Context, siteURL string, token *oauth2.Token) (GSCBreakdownSet, error) {
	set := GSCBreakdownSet{}
	set.StartDate, set.EndDate = gscDateWindow()
	if token == nil || s.oauthConfig == nil {
		return set, fmt.Errorf("no GSC credentials available")
	}
	client := s.oauthConfig.Client(ctx, token)
	svc, err := searchconsole.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return set, fmt.Errorf("failed to create GSC service: %w", err)
	}

	type dimJob struct {
		name  string
		limit int64
		set   func([]models.GSCBreakdown)
		ok    func()
	}
	jobs := []dimJob{
		{models.GSCDimPage, 50, func(r []models.GSCBreakdown) { set.Pages = r }, func() { set.GotPages = true }},
		{models.GSCDimCountry, 50, func(r []models.GSCBreakdown) { set.Countries = r }, func() { set.GotCountries = true }},
		{models.GSCDimDevice, 10, func(r []models.GSCBreakdown) { set.Devices = r }, func() { set.GotDevices = true }},
		{models.GSCDimSearchAppearance, 25, func(r []models.GSCBreakdown) { set.Appearance = r }, func() { set.GotAppearance = true }},
	}

	var firstErr error
	for _, job := range jobs {
		rows, qErr := queryGSCDimension(svc, siteURL, job.name, set.StartDate, set.EndDate, job.limit)
		if qErr != nil {
			if firstErr == nil {
				firstErr = qErr
			}
			continue
		}
		job.set(rows)
		job.ok()
	}
	if !set.GotPages && !set.GotCountries && !set.GotDevices && !set.GotAppearance {
		if firstErr != nil {
			return set, firstErr
		}
	}
	return set, nil
}

func queryGSCDimension(svc *searchconsole.Service, siteURL, dimension, start, end string, limit int64) ([]models.GSCBreakdown, error) {
	req := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  start,
		EndDate:    end,
		Dimensions: []string{dimension},
		RowLimit:   limit,
	}
	resp, err := svc.Searchanalytics.Query(siteURL, req).Do()
	if err != nil {
		return nil, fmt.Errorf("GSC %s query failed: %w", dimension, err)
	}
	return breakdownFromAPIRows(dimension, start, end, resp.Rows, time.Now().UTC()), nil
}

func breakdownFromAPIRows(dimension, start, end string, rows []*searchconsole.ApiDataRow, fetchedAt time.Time) []models.GSCBreakdown {
	out := make([]models.GSCBreakdown, 0, len(rows))
	for _, row := range rows {
		if row == nil || len(row.Keys) == 0 {
			continue
		}
		key := strings.TrimSpace(row.Keys[0])
		if key == "" {
			continue
		}
		out = append(out, models.GSCBreakdown{
			Dimension:   dimension,
			Key:         key,
			Clicks:      int64(row.Clicks),
			Impressions: int64(row.Impressions),
			CTR:         row.Ctr,
			Position:    row.Position,
			StartDate:   start,
			EndDate:     end,
			FetchedAt:   fetchedAt,
		})
	}
	return out
}
