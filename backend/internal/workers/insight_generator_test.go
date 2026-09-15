package workers

import (
	"strings"
	"testing"

	"backend/internal/models"
)

func TestGscOnlyMetricsDropsSeed(t *testing.T) {
	in := []models.Metric{
		{Source: models.MetricSourceSeed, Clicks: 1000},
		{Source: models.MetricSourceGSC, Clicks: 3},
		{Source: "", Clicks: 50},
	}
	out := gscOnlyMetrics(in)
	if len(out) != 1 || out[0].Clicks != 3 {
		t.Fatalf("got %+v, want single gsc row with 3 clicks", out)
	}
}

func TestRuleBasedInsightsIgnoreSeedTraffic(t *testing.T) {
	project := models.Project{ID: 1, Name: "Citc", URL: "https://citc.live/", HealthScore: 46}
	seed := []models.Metric{
		{Source: models.MetricSourceSeed, Clicks: 66000, Impressions: 1000000},
	}
	prev := []models.Metric{
		{Source: models.MetricSourceSeed, Clicks: 4700, Impressions: 100000},
	}
	insights := generateRuleBasedInsights(project, gscOnlyMetrics(seed), gscOnlyMetrics(prev), nil)
	for _, in := range insights {
		if strings.Contains(in.Title, "Organic Traffic Up") || strings.Contains(in.Body, "1314") || strings.Contains(in.Body, "66.0k") {
			t.Fatalf("seed-derived insight leaked: %+v", in)
		}
	}
	foundConnect := false
	for _, in := range insights {
		if strings.Contains(in.Title, "Connect Google Search Console") {
			foundConnect = true
		}
	}
	if !foundConnect {
		t.Fatal("expected honest Connect GSC insight when there is no real traffic")
	}
}
