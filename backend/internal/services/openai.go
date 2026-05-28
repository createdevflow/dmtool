// Package services provides the OpenAIService for content generation and insights.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"backend/internal/models"

	openai "github.com/sashabaranov/go-openai"
)

// ContentVariant is one generated content option returned to the frontend.
type ContentVariant struct {
	ID           int    `json:"id"`
	Content      string `json:"content"`
	PlatformFit  int    `json:"platform_fit"`  // 0-10
	Clarity      int    `json:"clarity"`       // 0-10
	CTAStrength  int    `json:"cta_strength"`  // 0-10
	CharCount    int    `json:"char_count"`
	GeneratedAt  string `json:"generated_at"`
}

// OpenAIService handles interactions with OpenAI API for insights and content.
type OpenAIService interface {
	GenerateInsights(ctx context.Context, project *models.Project, metrics []models.Metric) ([]models.Insight, error)
	GenerateContent(ctx context.Context, topic, platform, tone string) ([]ContentVariant, error)
}

type openAIService struct {
	client *openai.Client
}

func NewOpenAIService(apiKey string) OpenAIService {
	return &openAIService{
		client: openai.NewClient(apiKey),
	}
}

// GenerateInsights calls GPT with project metrics and returns structured insights.
func (s *openAIService) GenerateInsights(ctx context.Context, project *models.Project, metrics []models.Metric) ([]models.Insight, error) {
	if project == nil || len(metrics) == 0 {
		return nil, fmt.Errorf("insufficient data for insights")
	}

	systemPrompt := `You are a senior digital marketing consultant. Analyze the provided project data and metrics.
Return a JSON object with an "insights" array. Each insight must have:
- "title": string (concise, action-oriented, max 60 chars)
- "content": string (specific, actionable recommendation with numbers where possible, 2-3 sentences)  
- "priority": string ("high", "medium", or "low")
Return between 3-5 insights. Respond ONLY with valid JSON.`

	metricsSummary := ""
	for _, m := range metrics {
		metricsSummary += fmt.Sprintf("Date: %s, Clicks: %d, Impressions: %d\n", m.Date, m.Clicks, m.Impressions)
	}

	userPrompt := fmt.Sprintf(
		"Project: %s\nURL: %s\nGoal: %s\nSEO Health: %d/100\n\nLast 30 days metrics:\n%s",
		project.Name, project.URL, project.Goal, project.HealthScore, metricsSummary,
	)

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	})

	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}

	rawContent := resp.Choices[0].Message.Content
	log.Printf("[openai] Insights response: %s", rawContent)

	// Parse the JSON response
	var parsed struct {
		Insights []struct {
			Title    string `json:"title"`
			Content  string `json:"content"`
			Priority string `json:"priority"`
		} `json:"insights"`
	}

	if err := json.Unmarshal([]byte(rawContent), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse GPT response: %w", err)
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
			ProjectID: project.ID,
			Type:      iType,
			Title:     item.Title,
			Body:      item.Content,
			Priority:  i + 1,
			CreatedAt: time.Now(),
		})
	}

	return insights, nil
}

// GenerateContent produces 3 content variants for the given platform and topic.
func (s *openAIService) GenerateContent(ctx context.Context, topic, platform, tone string) ([]ContentVariant, error) {
	platformGuide := map[string]string{
		"instagram":  "Instagram caption (max 2200 chars). Use emojis, 3-5 relevant hashtags at the end.",
		"twitter":    "Twitter/X post (max 280 chars). Be punchy and direct. No hashtags unless critical.",
		"linkedin":   "LinkedIn post (max 3000 chars). Professional tone. Use line breaks. Include a question or CTA.",
		"blog":       "Blog post title and opening paragraph (max 300 chars title + 500 chars intro).",
		"email":      "Email subject line (max 60 chars) + preview text (max 100 chars). Focus on open rate.",
		"facebook":   "Facebook post (max 500 chars). Conversational. Include a question to drive comments.",
	}

	guide, ok := platformGuide[strings.ToLower(platform)]
	if !ok {
		guide = "social media post (max 300 chars)"
	}

	systemPrompt := fmt.Sprintf(`You are an expert social media strategist. Generate 3 distinct content variants.
Format: JSON object with "variants" array. Each variant:
- "content": the actual post text
- "platform_fit": score 0-10 (how well it fits %s)
- "clarity": score 0-10
- "cta_strength": score 0-10

Platform: %s
Tone: %s
Guidelines: %s

Respond ONLY with valid JSON.`, platform, platform, tone, guide)

	userPrompt := fmt.Sprintf("Topic: %s\n\nGenerate 3 unique content variants. Make each variant significantly different in approach (e.g., one storytelling, one data-driven, one question-based).", topic)

	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject},
	})

	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}

	rawContent := resp.Choices[0].Message.Content

	var parsed struct {
		Variants []struct {
			Content     string `json:"content"`
			PlatformFit int    `json:"platform_fit"`
			Clarity     int    `json:"clarity"`
			CTAStrength int    `json:"cta_strength"`
		} `json:"variants"`
	}

	if err := json.Unmarshal([]byte(rawContent), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse GPT content response: %w", err)
	}

	var variants []ContentVariant
	now := time.Now().Format(time.RFC3339)
	for i, v := range parsed.Variants {
		variants = append(variants, ContentVariant{
			ID:          i + 1,
			Content:     v.Content,
			PlatformFit: v.PlatformFit,
			Clarity:     v.Clarity,
			CTAStrength: v.CTAStrength,
			CharCount:   len(v.Content),
			GeneratedAt: now,
		})
	}

	return variants, nil
}
