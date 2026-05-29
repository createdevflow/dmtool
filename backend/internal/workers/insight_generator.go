// Package workers contains the InsightGenerator background worker.
// It runs every 12 hours and generates actionable insights from real metric trends.
// When an OpenAI key is configured, it uses GPT. Otherwise it uses rule-based logic
// derived from actual metric deltas — NOT random or hardcoded data.
package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"

	"gorm.io/gorm"
)

// StartInsightGenerator runs every 12 hours and generates fresh AI insights
// for all active projects using the last 30 days of metrics.
func StartInsightGenerator(
	db *gorm.DB,
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	insightRepo repository.InsightRepository,
	openai services.OpenAIService,
	cfg *config.Config,
) {
	log.Println("[worker:InsightGenerator] started — interval: 12h")
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	// Run immediately on startup so dashboard has insights from first launch
	runInsightGenerator(projectRepo, metricRepo, insightRepo, openai)

	for range ticker.C {
		runInsightGenerator(projectRepo, metricRepo, insightRepo, openai)
	}
}

func runInsightGenerator(
	projectRepo repository.ProjectRepository,
	metricRepo repository.MetricRepository,
	insightRepo repository.InsightRepository,
	openai services.OpenAIService,
) {
	log.Println("[worker:InsightGenerator] starting insight generation cycle...")

	projects, err := projectRepo.FindAll()
	if err != nil {
		log.Printf("[worker:InsightGenerator] failed to fetch projects: %v", err)
		return
	}

	now := time.Now()
	t0 := now.Format("2006-01-02")
	t30 := now.AddDate(0, 0, -30).Format("2006-01-02")
	t60 := now.AddDate(0, 0, -60).Format("2006-01-02")

	for _, project := range projects {
		// Fetch current and previous period metrics
		currentMetrics, _ := metricRepo.FindMetricsByProjectAndRange(project.ID, t30, t0)
		previousMetrics, _ := metricRepo.FindMetricsByProjectAndRange(project.ID, t60, t30)
		socialMetrics, _ := metricRepo.FindLatestSocialMetrics(project.ID)

		var insights []models.Insight

		if openai != nil {
			// Use OpenAI for insight generation
			generated, err := openai.GenerateInsights(context.Background(), &project, currentMetrics)
			if err == nil && len(generated) > 0 {
				insights = generated
			} else {
				log.Printf("[worker:InsightGenerator] OpenAI failed for project %d, falling back to rule-based: %v", project.ID, err)
				insights = generateRuleBasedInsights(project, currentMetrics, previousMetrics, socialMetrics)
			}
		} else {
			// Rule-based insights from real metric analysis
			insights = generateRuleBasedInsights(project, currentMetrics, previousMetrics, socialMetrics)
		}

		if len(insights) > 0 {
			if err := insightRepo.ReplaceInsights(project.ID, insights); err != nil {
				log.Printf("[worker:InsightGenerator] failed to save insights for project %d: %v", project.ID, err)
			} else {
				log.Printf("[worker:InsightGenerator] saved %d insights for project %d (%s)", len(insights), project.ID, project.Name)
			}
		}
	}

	log.Println("[worker:InsightGenerator] cycle complete")
}

// trendSignals holds computed analytics for a project period.
type trendSignals struct {
	currClicks      int64
	prevClicks      int64
	currImpressions int64
	prevImpressions int64
	currCTR         float64
	prevCTR         float64
	clickGrowth     float64 // % change
	impressionGrowth float64
	ctrDelta        float64
	socialFollowers int64
	socialReach     int64
	socialEngRate   float64
	hasGSCData      bool
	hasSocialData   bool
	dataPoints      int
}

func computeSignals(current, previous []models.Metric, social []models.SocialMetric) trendSignals {
	s := trendSignals{}

	for _, m := range current {
		s.currClicks += m.Clicks
		s.currImpressions += m.Impressions
		if m.Source == "gsc" {
			s.hasGSCData = true
		}
	}
	for _, m := range previous {
		s.prevClicks += m.Clicks
		s.prevImpressions += m.Impressions
	}

	s.dataPoints = len(current)

	if s.currImpressions > 0 {
		s.currCTR = float64(s.currClicks) / float64(s.currImpressions) * 100
	}
	if s.prevImpressions > 0 {
		s.prevCTR = float64(s.prevClicks) / float64(s.prevImpressions) * 100
	}

	if s.prevClicks > 0 {
		s.clickGrowth = float64(s.currClicks-s.prevClicks) / float64(s.prevClicks) * 100
	}
	if s.prevImpressions > 0 {
		s.impressionGrowth = float64(s.currImpressions-s.prevImpressions) / float64(s.prevImpressions) * 100
	}
	s.ctrDelta = s.currCTR - s.prevCTR

	for _, sm := range social {
		s.socialFollowers += sm.Followers
		s.socialReach += sm.Reach
		s.socialEngRate += sm.Engagement
		if !sm.IsSimulated {
			s.hasSocialData = true
		}
	}
	if len(social) > 0 {
		s.socialEngRate = s.socialEngRate / float64(len(social))
	}

	return s
}

// generateRuleBasedInsights produces real insights by analysing actual metric trends.
// Every insight is derived from computed data — nothing is hardcoded.
func generateRuleBasedInsights(project models.Project, current, previous []models.Metric, social []models.SocialMetric) []models.Insight {
	sig := computeSignals(current, previous, social)

	var insights []models.Insight
	priority := 1

	addInsight := func(iType, title, body string) {
		insights = append(insights, models.Insight{
			ProjectID: project.ID,
			Type:      iType,
			Title:     title,
			Body:      body,
			Priority:  priority,
			CreatedAt: time.Now(),
		})
		priority++
	}

	// ── Insight 1: Click performance vs. impressions (CTR signal) ──────────
	if sig.currImpressions > 0 {
		ctr := sig.currCTR
		if ctr < 2.0 && sig.currImpressions > 500 {
			addInsight(models.InsightTypeCritical,
				"Low Click-Through Rate Detected",
				fmt.Sprintf("Your site received %s impressions in the last 30 days but only a %.1f%% CTR (%.0f clicks). "+
					"This means search engines are showing your pages but users aren't clicking. "+
					"Rewrite your title tags and meta descriptions to be more compelling — add numbers, action words, and your primary keyword.",
					formatNum(sig.currImpressions), ctr, float64(sig.currClicks)),
			)
		} else if ctr >= 2.0 && ctr < 5.0 {
			addInsight(models.InsightTypeOpportunity,
				"Good Impressions — Boost CTR to Convert More Traffic",
				fmt.Sprintf("Your %.1f%% CTR across %s impressions is on the right track. "+
					"To push above 5%%, test A/B variations of your title tags using power words like 'Ultimate', 'Complete', or 'Step-by-Step'. "+
					"Also, add structured data (FAQ, HowTo schema) to earn rich snippets.",
					ctr, formatNum(sig.currImpressions)),
			)
		} else if ctr >= 5.0 {
			addInsight(models.InsightTypeInfo,
				"Excellent Click-Through Rate",
				fmt.Sprintf("Your %.1f%% CTR is above the industry average of 3–5%%. "+
					"You're crafting titles and descriptions that resonate with users. "+
					"Maintain this by continuing to A/B test and monitoring CTR per query in Google Search Console.",
					ctr),
			)
		}
	}

	// ── Insight 2: Traffic trend ─────────────────────────────────────────────
	if sig.prevClicks > 0 && sig.dataPoints >= 7 {
		if sig.clickGrowth >= 20 {
			addInsight(models.InsightTypeOpportunity,
				fmt.Sprintf("Organic Traffic Up %.0f%% — Scale What's Working", sig.clickGrowth),
				fmt.Sprintf("Organic clicks grew from %s to %s (+%.0f%%) this period. "+
					"Identify which pages drove this growth in Google Search Console and create 3–5 supporting articles on those same topics to capture more long-tail traffic.",
					formatNum(sig.prevClicks), formatNum(sig.currClicks), sig.clickGrowth),
			)
		} else if sig.clickGrowth <= -15 {
			addInsight(models.InsightTypeCritical,
				fmt.Sprintf("Organic Traffic Down %.0f%% — Immediate Action Required", math.Abs(sig.clickGrowth)),
				fmt.Sprintf("Clicks dropped from %s to %s (%.0f%%). "+
					"Check Google Search Console for manual actions or algorithm updates. "+
					"Review your top-performing pages for recent content changes. "+
					"Ensure robots.txt isn't accidentally blocking pages.",
					formatNum(sig.prevClicks), formatNum(sig.currClicks), sig.clickGrowth),
			)
		} else if sig.clickGrowth > 0 && sig.clickGrowth < 20 {
			addInsight(models.InsightTypeInfo,
				fmt.Sprintf("Steady Growth: +%.0f%% Traffic This Period", sig.clickGrowth),
				fmt.Sprintf("Organic traffic grew steadily by %.0f%% (%s → %s clicks). "+
					"To accelerate beyond steady growth, focus on building topical authority: "+
					"publish a cluster of 5+ articles around your main keyword themes.",
					sig.clickGrowth, formatNum(sig.prevClicks), formatNum(sig.currClicks)),
			)
		}
	} else if sig.dataPoints < 3 {
		// New project or no data yet
		addInsight(models.InsightTypeInfo,
			"Connect Google Search Console for Real Traffic Data",
			fmt.Sprintf("Your project '%s' is set up but no search performance data has been synced yet. "+
				"Go to Integrations → Connect Google Search Console to start tracking real organic traffic, "+
				"keyword rankings, and CTR data automatically.", project.Name),
		)
	}

	// ── Insight 3: Impressions trend ─────────────────────────────────────────
	if sig.currImpressions > 0 && sig.prevImpressions > 0 {
		if sig.impressionGrowth >= 30 && sig.clickGrowth < sig.impressionGrowth/2 {
			addInsight(models.InsightTypeOpportunity,
				"Impressions Surging — Clicks Not Keeping Up",
				fmt.Sprintf("Impressions increased %.0f%% but clicks grew only %.0f%%. "+
					"Your content is ranking for more queries but not converting to clicks. "+
					"Audit your top-impression pages: are they ranking on page 1 (positions 1-10)? "+
					"Positions 4-10 drive most impressions but low clicks — push them to top 3 with targeted content improvements.",
					sig.impressionGrowth, sig.clickGrowth),
			)
		}
	}

	// ── Insight 4: Social engagement ─────────────────────────────────────────
	if sig.hasSocialData && sig.socialFollowers > 0 {
		engRate := sig.socialEngRate
		if engRate < 1.0 {
			addInsight(models.InsightTypeCritical,
				"Social Engagement Rate Below 1% — Audience Not Converting",
				fmt.Sprintf("Your engagement rate is %.1f%% across %s followers. "+
					"Below 1%% means your content isn't resonating. "+
					"Switch to more interactive content: polls, Q&A stories, behind-the-scenes posts. "+
					"Post at peak times (typically 11am-1pm and 7pm-9pm local time) and use 5-7 relevant hashtags.",
					engRate, formatNum(sig.socialFollowers)),
			)
		} else if engRate >= 3.5 {
			addInsight(models.InsightTypeOpportunity,
				fmt.Sprintf("Strong %.1f%% Engagement — Leverage for Growth", engRate),
				fmt.Sprintf("A %.1f%% engagement rate with %s followers is above the 3%% benchmark — your audience is active. "+
					"Capitalise by cross-promoting your top posts to your email list and running a giveaway or collab to convert engaged followers into website traffic.",
					engRate, formatNum(sig.socialFollowers)),
			)
		} else {
			addInsight(models.InsightTypeInfo,
				fmt.Sprintf("Steady %.1f%% Engagement", engRate),
				fmt.Sprintf("Your engagement rate of %.1f%% across %s followers is average. "+
					"Focus on creating more Save-able and Share-able content to boost it above the 3.5%% benchmark.",
					engRate, formatNum(sig.socialFollowers)),
			)
		}
	} else if project.IGHandle != "" && !sig.hasSocialData {
		addInsight(models.InsightTypeInfo,
			"Connect Meta API for Real Instagram Analytics",
			fmt.Sprintf("Instagram handle @%s is linked, but real metrics require a Meta API connection. "+
				"Go to Integrations → Connect Meta to fetch live follower count, reach, engagement rate, and top posts.",
				project.IGHandle),
		)
	}

	// ── Insight 5: SEO Health Score ──────────────────────────────────────────
	if project.HealthScore > 0 {
		if project.HealthScore < 50 {
			addInsight(models.InsightTypeCritical,
				fmt.Sprintf("SEO Health Score Critical: %d/100", project.HealthScore),
				fmt.Sprintf("Your site scored %d/100 on the SEO audit — multiple high-severity issues detected. "+
					"Go to SEO → Site Explorer and run a fresh audit to see the detailed issue list. "+
					"Prioritise: fixing missing meta tags, adding H1 headings, and enabling HTTPS.",
					project.HealthScore),
			)
		} else if project.HealthScore < 75 {
			addInsight(models.InsightTypeOpportunity,
				fmt.Sprintf("SEO Score at %d — 25 Points Away from Excellent", project.HealthScore),
				fmt.Sprintf("Your SEO health score is %d/100. The remaining issues are likely medium-severity: "+
					"missing Open Graph tags, canonical URL gaps, or structured data. "+
					"Check the SEO → Audit Report for a full list sorted by impact.",
					project.HealthScore),
			)
		} else {
			addInsight(models.InsightTypeInfo,
				fmt.Sprintf("Strong SEO Foundation: %d/100", project.HealthScore),
				fmt.Sprintf("Your site is technically well-optimised at %d/100. "+
					"Focus your energy on content: publish at least 2 new articles per week targeting long-tail keywords. "+
					"Run the audit monthly to catch any regressions.",
					project.HealthScore),
			)
		}
	} else {
		addInsight(models.InsightTypeOpportunity,
			"Run Your First SEO Audit",
			fmt.Sprintf("No SEO audit has been run for %s yet. "+
				"Go to SEO → Site Explorer and run an audit to get your baseline health score, "+
				"identify technical issues, and get a prioritised fix list.", project.URL),
		)
	}

	// Limit to 5 most important insights
	if len(insights) > 5 {
		insights = insights[:5]
	}

	return insights
}

// formatNum formats a large number as human-readable (1.2k, 3.4M, etc.).
func formatNum(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// openAIInsightResponse is the expected JSON shape from GPT.
type openAIInsightResponse struct {
	Insights []struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Priority string `json:"priority"` // high | medium | low
	} `json:"insights"`
}

// ParseOpenAIInsights parses a raw JSON string from GPT into Insight records.
func ParseOpenAIInsights(projectID uint, raw string) ([]models.Insight, error) {
	// Try to extract JSON from the response (GPT sometimes wraps in markdown)
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "{"); idx > 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx > 0 && idx < len(raw)-1 {
		raw = raw[:idx+1]
	}

	var parsed openAIInsightResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	var insights []models.Insight
	for i, item := range parsed.Insights {
		iType := models.InsightTypeInfo
		switch strings.ToLower(item.Priority) {
		case "high":
			iType = models.InsightTypeCritical
		case "medium":
			iType = models.InsightTypeOpportunity
		}
		insights = append(insights, models.Insight{
			ProjectID: projectID,
			Type:      iType,
			Title:     item.Title,
			Body:      item.Content,
			Priority:  i + 1,
			CreatedAt: time.Now(),
		})
	}

	return insights, nil
}
