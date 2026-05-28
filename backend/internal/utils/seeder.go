// Package utils provides the auto-seeder that generates 30 days of realistic
// marketing data when a project is created without API credentials.
// Seeded data uses Source="seed" so it can be identified and replaced later.
package utils

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

// SeedProject inserts 30 days of realistic metrics, social metrics, insights,
// tasks, SEO issues, and keyword results for the given project.
// It is idempotent — calling it twice on the same project is safe (it skips
// dates that already have seeded data).
func SeedProject(db *gorm.DB, projectID uint) error {
	now := time.Now()

	var project models.Project
	db.First(&project, projectID)

	// --- Metrics (daily clicks, impressions, reach, engagement) ---
	if project.URL != "" {
	for i := 29; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")

		var count int64
		db.Model(&models.Metric{}).
			Where("project_id = ? AND date = ? AND source = ?", projectID, date, models.MetricSourceSeed).
			Count(&count)
		if count > 0 {
			continue // already seeded this date
		}

		// Create a realistic growth curve with some noise
		dayIndex := float64(29 - i)
		growth := 1.0 + (dayIndex/29.0)*0.4 // up to 40% growth over 30 days
		noise := 0.85 + rand.Float64()*0.30  // ±15% noise

		clicks := int64(math.Round(float64(rand.Intn(200)+100) * growth * noise))
		impressions := clicks * int64(rand.Intn(8)+8)
		reach := int64(math.Round(float64(rand.Intn(1500)+500) * growth * noise))
		engagement := int64(math.Round(float64(rand.Intn(200)+50) * growth * noise))

		db.Create(&models.Metric{
			ProjectID:   projectID,
			Date:        date,
			Clicks:      clicks,
			Impressions: impressions,
			Reach:       reach,
			Engagement:  engagement,
			Source:      models.MetricSourceSeed,
		})
		}
	}

	// --- Social Metrics ---

	platforms := []string{}
	if project.IGHandle != "" {
		platforms = append(platforms, "instagram")
	}
	if project.FBHandle != "" {
		platforms = append(platforms, "facebook")
	}
	if project.TwitterHandle != "" {
		platforms = append(platforms, "twitter")
	}
	if project.LinkedinHandle != "" {
		platforms = append(platforms, "linkedin")
	}

	for _, platform := range platforms {
		var count int64
		db.Model(&models.SocialMetric{}).
			Where("project_id = ? AND platform = ?", projectID, platform).
			Count(&count)
		if count >= 30 {
			continue
		}

		baseFollowers := int64(rand.Intn(5000) + 1000)
		for i := 29; i >= 0; i-- {
			recorded := now.AddDate(0, 0, -i)
			daysSinceStart := 29 - i
			followers := baseFollowers + int64(float64(daysSinceStart)*float64(rand.Intn(20)+5))
			reach := int64(float64(followers) * (0.05 + rand.Float64()*0.15))
			engagementRate := 2.0 + rand.Float64()*5.0
			engagementCount := int64(float64(reach) * (engagementRate / 100.0))

			status := "stable"
			if rand.Intn(10) > 6 {
				status = "growing"
			}

			db.Create(&models.SocialMetric{
				ProjectID:       projectID,
				Platform:        platform,
				Followers:       followers,
				Reach:           reach,
				Engagement:      math.Round(engagementRate*100) / 100,
				EngagementCount: engagementCount,
				Status:          status,
				IsSimulated:     true,
				RecordedAt:      recorded,
			})
		}
	}

	// --- Insights ---
	var insightCount int64
	db.Model(&models.Insight{}).Where("project_id = ?", projectID).Count(&insightCount)
	if insightCount == 0 {
		seedInsights := []models.Insight{
			{
				ProjectID: projectID,
				Type:      models.InsightTypeCritical,
				Title:     "Missing Meta Description on Homepage",
				Body:      "Your homepage is missing a meta description, which reduces click-through rates from search results by up to 30%.",
				Priority:  1,
			},
			{
				ProjectID: projectID,
				Type:      models.InsightTypeOpportunity,
				Title:     "High Impression Volume on Branded Keywords",
				Body:      "Your branded keywords are generating 8x more impressions than last month. Consider increasing ad spend on these terms.",
				Priority:  2,
			},
			{
				ProjectID: projectID,
				Type:      models.InsightTypeOpportunity,
				Title:     "Instagram Engagement Rate Above Industry Average",
				Body:      "Your Instagram engagement rate of 4.2% exceeds the industry average of 1.9%. Posting 3–4 times per week sustains this momentum.",
				Priority:  3,
			},
		}
		db.Create(&seedInsights)
	}

	// --- Tasks ---
	var taskCount int64
	db.Model(&models.Task{}).Where("project_id = ?", projectID).Count(&taskCount)
	if taskCount == 0 {
		due1 := now.AddDate(0, 0, 3)
		due2 := now.AddDate(0, 0, 7)
		seedTasks := []models.Task{
			{ProjectID: projectID, Title: "Add meta description to homepage", Source: models.TaskSourceAI, DueDate: &due1},
			{ProjectID: projectID, Title: "Publish 2 Instagram posts this week", Source: models.TaskSourceAI, DueDate: &due2},
			{ProjectID: projectID, Title: "Review and fix H1 tags on top 5 pages", Source: models.TaskSourceAI},
			{ProjectID: projectID, Title: "Submit XML sitemap to Google Search Console", Source: models.TaskSourceManual},
			{ProjectID: projectID, Title: "Connect Google Search Console for live data", Source: models.TaskSourceManual},
			{ProjectID: projectID, Title: "Weekly Progress Report [If: Every Friday → Then: Generate PDF Summary]", Source: "automation"},
		}
		db.Create(&seedTasks)
	}

	// --- SEO Issues ---
	if project.URL != "" {
		var issueCount int64
		db.Model(&models.SEOIssue{}).Where("project_id = ?", projectID).Count(&issueCount)
		if issueCount == 0 {
			seedIssues := []models.SEOIssue{
				{ProjectID: projectID, URL: project.URL + "/", Severity: models.SeverityHigh, Category: "meta", Detail: "Missing meta description"},
				{ProjectID: projectID, URL: project.URL + "/about", Severity: models.SeverityMed, Category: "heading", Detail: "Multiple H1 tags found (3)"},
				{ProjectID: projectID, URL: project.URL + "/blog", Severity: models.SeverityMed, Category: "performance", Detail: "Page load time 4.2s (target: <2.5s)"},
				{ProjectID: projectID, URL: project.URL + "/contact", Severity: models.SeverityLow, Category: "meta", Detail: "OG image tag missing"},
				{ProjectID: projectID, URL: project.URL + "/products", Severity: models.SeverityHigh, Category: "canonical", Detail: "Duplicate content — canonical tag missing"},
			}
			db.Create(&seedIssues)
		}
	}

	// --- Keywords ---
	if project.URL != "" {
		var kwCount int64
		db.Model(&models.KeywordResult{}).Where("project_id = ?", projectID).Count(&kwCount)
		if kwCount == 0 {
		seedTerms := []struct {
			keyword  string
			volume   int
			kd       int
			position float64
		}{
			{"digital marketing tools", 12400, 52, 14.3},
			{"seo analysis software", 8200, 67, 22.1},
			{"social media analytics", 18500, 44, 8.7},
			{"content marketing platform", 6300, 38, 31.0},
			{"keyword research tool", 22000, 71, 41.5},
			{"website traffic checker", 9800, 49, 18.2},
			{"meta ads performance", 5100, 35, 12.9},
			{"google search console guide", 14200, 29, 6.1},
		}
		for _, t := range seedTerms {
			db.Create(&models.KeywordResult{
				ProjectID: projectID,
				Seed:      "digital marketing",
				Keyword:   t.keyword,
				Volume:    t.volume,
				KD:        t.kd,
				Position:  t.position,
			})
		}
		}
	}

	return nil
}

// GenerateRefreshToken creates a cryptographically random opaque 256-bit token.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// HashString returns a stable uint64 hash of a string for generating stable mock data.
func HashString(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}
