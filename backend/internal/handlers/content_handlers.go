package handlers

import (
	"fmt"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type ContentHandler struct {
	projectRepo repository.ProjectRepository
	insightRepo repository.InsightRepository
	openai      services.OpenAIService
	cfg         *config.Config
}

func NewContentHandler(
	projectRepo repository.ProjectRepository,
	insightRepo repository.InsightRepository,
	openai services.OpenAIService,
	cfg *config.Config,
) *ContentHandler {
	return &ContentHandler{
		projectRepo: projectRepo,
		insightRepo: insightRepo,
		openai:      openai,
		cfg:         cfg,
	}
}

type ContentRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Topic     string `json:"topic" binding:"required"`
	Platform  string `json:"platform" binding:"required"` // instagram | twitter | linkedin | blog | email | facebook
	Tone      string `json:"tone"`                         // professional | casual | persuasive | witty | informative
}

// Generate produces 3 content variants for the given platform and topic.
// If no OpenAI key is configured, returns rule-based template content clearly labeled as template.
func (h *ContentHandler) Generate(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req ContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	// Verify project ownership
	project, err := h.projectRepo.FindByIDAndUser(req.ProjectID, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	tone := req.Tone
	if tone == "" {
		tone = "professional"
	}

	if h.openai != nil {
		// Use real GPT generation
		variants, err := h.openai.GenerateContent(c.Request.Context(), req.Topic, req.Platform, tone)
		if err != nil {
			utils.InternalError(c, "Content generation failed: "+err.Error())
			return
		}

		utils.Success(c, gin.H{
			"variants":    variants,
			"topic":       req.Topic,
			"platform":    req.Platform,
			"tone":        tone,
			"source":      "ai",
			"generated_at": time.Now().Format(time.RFC3339),
		}, nil)
		return
	}

	// No OpenAI key — use template-based generation clearly labeled as templates
	variants := generateTemplateContent(project, req.Topic, req.Platform, tone)

	utils.Success(c, gin.H{
		"variants":    variants,
		"topic":       req.Topic,
		"platform":    req.Platform,
		"tone":        tone,
		"source":      "template",
		"source_note": "These are smart templates based on your project data. Add an OPENAI_API_KEY to enable AI-powered generation.",
		"generated_at": time.Now().Format(time.RFC3339),
	}, nil)
}

// generateTemplateContent creates useful template-based content without requiring OpenAI.
// These are context-aware templates using the project's actual details — not random filler.
func generateTemplateContent(project *models.Project, topic, platform, tone string) []services.ContentVariant {
	now := time.Now().Format(time.RFC3339)
	topicClean := strings.TrimSpace(topic)

	// Extract a clean domain name for branding
	domain := extractBrandName(project.URL)
	if domain == "" {
		domain = project.Name
	}

	variants := []services.ContentVariant{}

	switch strings.ToLower(platform) {
	case "instagram":
		variants = []services.ContentVariant{
			{
				ID:          1,
				Content:     fmt.Sprintf("✨ %s just got even better! We're excited to share our latest updates on %s. Whether you're a long-time follower or just discovering us, this is the moment you've been waiting for.\n\n💡 Swipe to learn more!\n\n#%s #innovation #growth #digitalmarketing", domain, topicClean, strings.ToLower(strings.ReplaceAll(domain, " ", ""))),
				PlatformFit: 8, Clarity: 7, CTAStrength: 7,
				CharCount:   0,
				GeneratedAt: now,
			},
			{
				ID:          2,
				Content:     fmt.Sprintf("Did you know? %s has helped hundreds of businesses achieve their goals. Today, we're talking about %s — and why it matters for YOUR growth. 👇\n\nDouble tap if this resonates!\n\n#marketing #business #success #%s", domain, topicClean, strings.ToLower(strings.ReplaceAll(topicClean, " ", ""))),
				PlatformFit: 8, Clarity: 8, CTAStrength: 6,
				CharCount:   0,
				GeneratedAt: now,
			},
			{
				ID:          3,
				Content:     fmt.Sprintf("⚡ Quick tip on %s:\n\n1️⃣ Start with a clear goal\n2️⃣ Measure what matters\n3️⃣ Optimise and iterate\n\nAt %s, we believe in data-driven decisions. Link in bio for more!\n\n#tips #growthhacks #digitalstrategy", topicClean, domain),
				PlatformFit: 9, Clarity: 9, CTAStrength: 7,
				CharCount:   0,
				GeneratedAt: now,
			},
		}

	case "twitter", "x":
		variants = []services.ContentVariant{
			{
				ID:          1,
				Content:     fmt.Sprintf("Hot take: %s is the most underrated lever for business growth in 2025. Here's why it matters more than ever → [link]", topicClean),
				PlatformFit: 9, Clarity: 9, CTAStrength: 7,
				CharCount:   0,
				GeneratedAt: now,
			},
			{
				ID:          2,
				Content:     fmt.Sprintf("We just discovered something about %s that changed everything for us at %s. Thread 🧵👇", topicClean, domain),
				PlatformFit: 9, Clarity: 8, CTAStrength: 9,
				CharCount:   0,
				GeneratedAt: now,
			},
			{
				ID:          3,
				Content:     fmt.Sprintf("If you're not thinking about %s yet, you're already behind. @%s shares the playbook:", topicClean, strings.ToLower(strings.ReplaceAll(domain, " ", ""))),
				PlatformFit: 8, Clarity: 8, CTAStrength: 8,
				CharCount:   0,
				GeneratedAt: now,
			},
		}

	case "linkedin":
		variants = []services.ContentVariant{
			{
				ID: 1,
				Content: fmt.Sprintf(`I've been thinking a lot about %s lately.

After working with dozens of businesses at %s, I've noticed one pattern: the companies that get this right grow 3x faster than those who don't.

Here's what the best ones do differently:

→ They start with clear metrics
→ They optimise based on data, not gut feel  
→ They iterate quickly and document everything

What's your experience with %s? I'd love to hear in the comments.

#%s #businessgrowth #strategy`, topicClean, domain, topicClean, strings.ReplaceAll(strings.Title(topicClean), " ", "")),
				PlatformFit: 9, Clarity: 8, CTAStrength: 8,
				CharCount:   0,
				GeneratedAt: now,
			},
			{
				ID: 2,
				Content: fmt.Sprintf(`3 things I wish I knew about %s when I started:

1. It's not about perfection — it's about progress
2. Data tells you what, creativity tells you why
3. The best time to start was yesterday. The second best is now.

At %s, we help businesses navigate this journey every day.

What would you add to this list?`, topicClean, domain),
				PlatformFit: 9, Clarity: 9, CTAStrength: 7,
				CharCount:   0,
				GeneratedAt: now,
			},
			{
				ID: 3,
				Content: fmt.Sprintf(`Unpopular opinion: Most businesses approach %s completely wrong.

They focus on vanity metrics instead of the numbers that actually move the needle.

Here's the framework we use at %s:

✅ Define success upfront
✅ Track leading indicators, not just lagging ones
✅ Review weekly, adjust monthly
✅ Share wins AND losses with your team

The difference between good and great is what you measure.

Drop a 🔥 if you agree!`, topicClean, domain),
				PlatformFit: 8, Clarity: 8, CTAStrength: 8,
				CharCount:   0,
				GeneratedAt: now,
			},
		}

	default: // blog, email, facebook, generic
		variants = []services.ContentVariant{
			{
				ID:          1,
				Content:     fmt.Sprintf("Everything you need to know about %s: A complete guide for 2025. At %s, we've broken down the key strategies that actually work.", topicClean, domain),
				PlatformFit: 7, Clarity: 8, CTAStrength: 6,
				CharCount:   0,
				GeneratedAt: now,
			},
			{
				ID:          2,
				Content:     fmt.Sprintf("Struggling with %s? You're not alone. Here's what we've learned after helping hundreds of businesses at %s — and what you should do differently.", topicClean, domain),
				PlatformFit: 7, Clarity: 9, CTAStrength: 8,
				CharCount:   0,
				GeneratedAt: now,
			},
			{
				ID:          3,
				Content:     fmt.Sprintf("The ultimate %s checklist: 10 things every marketer needs to do in 2025. Curated by the team at %s.", topicClean, domain),
				PlatformFit: 7, Clarity: 8, CTAStrength: 7,
				CharCount:   0,
				GeneratedAt: now,
			},
		}
	}

	// Set char counts
	for i := range variants {
		variants[i].CharCount = len(variants[i].Content)
	}

	return variants
}

func extractBrandName(rawURL string) string {
	s := rawURL
	for _, pfx := range []string{"https://", "http://", "www."} {
		if len(s) > len(pfx) && s[:len(pfx)] == pfx {
			s = s[len(pfx):]
		}
	}
	for i, ch := range s {
		if ch == '.' || ch == '/' {
			return s[:i]
		}
	}
	return s
}
