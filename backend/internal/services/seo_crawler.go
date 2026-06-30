// Package services provides the SEOCrawler — a real Go HTTP crawler that
// fetches a target URL and performs 13 technical SEO checks without any
// third-party paid API.
package services

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// CheckStatus constants map to UI icons.
const (
	CheckPass    = "pass"
	CheckWarning = "warning"
	CheckFail    = "fail"
)

// AuditCheck represents one SEO check result.
type AuditCheck struct {
	Category       string `json:"category"`
	Label          string `json:"label"`
	Status         string `json:"status"` // pass | warning | fail
	Detail         string `json:"detail"`
	Recommendation string `json:"recommendation"`
	Severity       string `json:"severity"` // high | medium | low
}

// AuditResult is the full output of a crawl.
type AuditResult struct {
	URL        string       `json:"url"`
	Score      int          `json:"score"`       // 0-100
	Checks     []AuditCheck `json:"checks"`
	CrawledAt  time.Time    `json:"crawled_at"`
	LoadTimeMs int64        `json:"load_time_ms"`
}

// SEOCrawlerService crawls a URL and returns a structured AuditResult.
type SEOCrawlerService interface {
	Crawl(targetURL string) (*AuditResult, error)
}

type seoCrawler struct {
	client *http.Client
}

// NewSEOCrawlerService returns a new crawler with a 15-second timeout.
func NewSEOCrawlerService() SEOCrawlerService {
	return &seoCrawler{
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (c *seoCrawler) Crawl(targetURL string) (*AuditResult, error) {
	// Normalise URL
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	result := &AuditResult{
		URL:       targetURL,
		CrawledAt: time.Now(),
		Checks:    []AuditCheck{},
	}

	// ── 1. Fetch the page ──────────────────────────────────────────────────
	start := time.Now()
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", "DMTool-SEOCrawler/2.0 (+https://dmtool.app)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.client.Do(req)
	if err != nil {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "accessibility", Label: "Page Reachability",
			Status: CheckFail, Severity: "high",
			Detail:         "Failed to fetch the page: " + err.Error(),
			Recommendation: "Ensure the URL is publicly accessible and returns a 200 OK response.",
		})
		result.Score = 0
		return result, nil
	}
	defer resp.Body.Close()
	result.LoadTimeMs = time.Since(start).Milliseconds()

	// Read body (limit to 2 MB to avoid memory issues)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	bodyStr := string(bodyBytes)

	// ── 2. Parse HTML ──────────────────────────────────────────────────────
	doc, err := html.Parse(strings.NewReader(bodyStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract key elements
	title := extractTitle(doc)
	metaDesc := extractMetaContent(doc, "description")
	h1s := extractH1s(doc)
	ogTitle := extractMetaProperty(doc, "og:title")
	ogDesc := extractMetaProperty(doc, "og:description")
	ogImage := extractMetaProperty(doc, "og:image")
	canonical := extractLinkHref(doc, "canonical")
	viewport := extractMetaContent(doc, "viewport")
	hasStructuredData := strings.Contains(bodyStr, `"@type"`) || strings.Contains(bodyStr, `application/ld+json`)
	imagesMissingAlt := countImagesMissingAlt(doc)

	// ── 3. Run checks ──────────────────────────────────────────────────────

	// Check: HTTPS
	if parsedURL.Scheme == "https" {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "security", Label: "HTTPS",
			Status: CheckPass, Severity: "high",
			Detail:         "Site is served over HTTPS.",
			Recommendation: "",
		})
	} else {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "security", Label: "HTTPS",
			Status: CheckFail, Severity: "high",
			Detail:         "Site is not using HTTPS.",
			Recommendation: "Install an SSL certificate and redirect all HTTP traffic to HTTPS.",
		})
	}

	// Check: HTTP Status Code
	if resp.StatusCode == 200 {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "accessibility", Label: "HTTP Status",
			Status: CheckPass, Severity: "high",
			Detail:         fmt.Sprintf("Page returned HTTP %d OK.", resp.StatusCode),
			Recommendation: "",
		})
	} else {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "accessibility", Label: "HTTP Status",
			Status: CheckFail, Severity: "high",
			Detail:         fmt.Sprintf("Page returned HTTP %d.", resp.StatusCode),
			Recommendation: "Ensure the URL returns a 200 OK status code for search engines to index it.",
		})
	}

	// Check: Page Load Time
	if result.LoadTimeMs < 1500 {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "performance", Label: "Page Load Speed",
			Status: CheckPass, Severity: "medium",
			Detail:         fmt.Sprintf("Page loaded in %dms — excellent.", result.LoadTimeMs),
			Recommendation: "",
		})
	} else if result.LoadTimeMs < 3000 {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "performance", Label: "Page Load Speed",
			Status: CheckWarning, Severity: "medium",
			Detail:         fmt.Sprintf("Page loaded in %dms — acceptable but improvable.", result.LoadTimeMs),
			Recommendation: "Aim for under 1.5s. Compress images, enable caching, use a CDN.",
		})
	} else {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "performance", Label: "Page Load Speed",
			Status: CheckFail, Severity: "high",
			Detail:         fmt.Sprintf("Page is slow: %dms load time.", result.LoadTimeMs),
			Recommendation: "Critical: page loads over 3 seconds. Optimise server response time, compress assets, and enable browser caching.",
		})
	}

	// Check: Title Tag
	titleLen := len(title)
	switch {
	case title == "":
		result.Checks = append(result.Checks, AuditCheck{
			Category: "meta", Label: "Title Tag",
			Status: CheckFail, Severity: "high",
			Detail:         "No <title> tag found.",
			Recommendation: "Add a unique, descriptive title tag between 10–60 characters.",
		})
	case titleLen < 10:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "meta", Label: "Title Tag",
			Status: CheckWarning, Severity: "medium",
			Detail:         fmt.Sprintf("Title tag is too short (%d chars): \"%s\"", titleLen, title),
			Recommendation: "Expand your title to 10–60 characters with primary keyword first.",
		})
	case titleLen > 60:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "meta", Label: "Title Tag",
			Status: CheckWarning, Severity: "medium",
			Detail:         fmt.Sprintf("Title tag is too long (%d chars): \"%s\"", titleLen, truncate(title, 60)),
			Recommendation: "Shorten title to under 60 characters to avoid truncation in SERPs.",
		})
	default:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "meta", Label: "Title Tag",
			Status: CheckPass, Severity: "high",
			Detail:         fmt.Sprintf("Title tag present (%d chars): \"%s\"", titleLen, title),
			Recommendation: "",
		})
	}

	// Check: Meta Description
	descLen := len(metaDesc)
	switch {
	case metaDesc == "":
		result.Checks = append(result.Checks, AuditCheck{
			Category: "meta", Label: "Meta Description",
			Status: CheckFail, Severity: "high",
			Detail:         "No meta description found.",
			Recommendation: "Add a meta description between 50–160 characters summarising the page.",
		})
	case descLen < 50:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "meta", Label: "Meta Description",
			Status: CheckWarning, Severity: "medium",
			Detail:         fmt.Sprintf("Meta description is too short (%d chars).", descLen),
			Recommendation: "Expand the meta description to 50–160 characters with target keywords.",
		})
	case descLen > 160:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "meta", Label: "Meta Description",
			Status: CheckWarning, Severity: "low",
			Detail:         fmt.Sprintf("Meta description is too long (%d chars) — will be truncated.", descLen),
			Recommendation: "Trim the meta description to under 160 characters.",
		})
	default:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "meta", Label: "Meta Description",
			Status: CheckPass, Severity: "high",
			Detail:         fmt.Sprintf("Meta description present (%d chars).", descLen),
			Recommendation: "",
		})
	}

	// Check: H1 Tag
	switch len(h1s) {
	case 0:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "content", Label: "H1 Heading",
			Status: CheckFail, Severity: "high",
			Detail:         "No <h1> tag found on the page.",
			Recommendation: "Add a single H1 tag containing your primary keyword.",
		})
	case 1:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "content", Label: "H1 Heading",
			Status: CheckPass, Severity: "high",
			Detail:         fmt.Sprintf("One H1 found: \"%s\"", truncate(h1s[0], 80)),
			Recommendation: "",
		})
	default:
		result.Checks = append(result.Checks, AuditCheck{
			Category: "content", Label: "H1 Heading",
			Status: CheckWarning, Severity: "medium",
			Detail:         fmt.Sprintf("Multiple H1 tags found (%d). First: \"%s\"", len(h1s), truncate(h1s[0], 60)),
			Recommendation: "Use only one H1 per page. Multiple H1s dilute SEO signal.",
		})
	}

	// Check: Open Graph Tags
	if ogTitle != "" && ogDesc != "" && ogImage != "" {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "social", Label: "Open Graph Tags",
			Status: CheckPass, Severity: "medium",
			Detail:         "og:title, og:description, and og:image are all present.",
			Recommendation: "",
		})
	} else if ogTitle != "" || ogDesc != "" {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "social", Label: "Open Graph Tags",
			Status: CheckWarning, Severity: "low",
			Detail:         "Some Open Graph tags are missing (og:image or og:description).",
			Recommendation: "Add og:title, og:description, and og:image for optimal social sharing previews.",
		})
	} else {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "social", Label: "Open Graph Tags",
			Status: CheckFail, Severity: "medium",
			Detail:         "No Open Graph tags found.",
			Recommendation: "Add og:title, og:description, and og:image meta tags for rich social media previews.",
		})
	}

	// Check: Canonical Tag
	if canonical != "" {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "technical", Label: "Canonical Tag",
			Status: CheckPass, Severity: "medium",
			Detail:         fmt.Sprintf("Canonical URL: %s", canonical),
			Recommendation: "",
		})
	} else {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "technical", Label: "Canonical Tag",
			Status: CheckWarning, Severity: "medium",
			Detail:         "No canonical tag found.",
			Recommendation: "Add <link rel=\"canonical\" href=\"...\"> to prevent duplicate content issues.",
		})
	}

	// Check: Mobile Viewport
	if viewport != "" {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "mobile", Label: "Mobile Viewport",
			Status: CheckPass, Severity: "high",
			Detail:         "Viewport meta tag present.",
			Recommendation: "",
		})
	} else {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "mobile", Label: "Mobile Viewport",
			Status: CheckFail, Severity: "high",
			Detail:         "No viewport meta tag found.",
			Recommendation: "Add <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"> for mobile friendliness.",
		})
	}

	// Check: Structured Data
	if hasStructuredData {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "technical", Label: "Structured Data",
			Status: CheckPass, Severity: "medium",
			Detail:         "JSON-LD structured data found.",
			Recommendation: "",
		})
	} else {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "technical", Label: "Structured Data",
			Status: CheckWarning, Severity: "low",
			Detail:         "No JSON-LD structured data detected.",
			Recommendation: "Add Schema.org structured data (JSON-LD) to improve rich snippets in search results.",
		})
	}

	// Check: Image Alt Text
	if imagesMissingAlt == 0 {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "content", Label: "Image Alt Text",
			Status: CheckPass, Severity: "medium",
			Detail:         "All sampled images have alt attributes.",
			Recommendation: "",
		})
	} else {
		result.Checks = append(result.Checks, AuditCheck{
			Category: "content", Label: "Image Alt Text",
			Status: CheckWarning, Severity: "medium",
			Detail:         fmt.Sprintf("%d image(s) are missing alt text.", imagesMissingAlt),
			Recommendation: "Add descriptive alt text to all images for accessibility and image search ranking.",
		})
	}

	// Fetch robots.txt and sitemap.xml concurrently.
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)
	sitemapURL := fmt.Sprintf("%s://%s/sitemap.xml", parsedURL.Scheme, parsedURL.Host)

	var (
		robotsResp *http.Response
		smResp     *http.Response
		robotsErr  error
		smErr      error
	)
	var fetchWG sync.WaitGroup
	fetchWG.Add(2)
	go func() {
		defer fetchWG.Done()
		robotsResp, robotsErr = c.client.Get(robotsURL)
	}()
	go func() {
		defer fetchWG.Done()
		smResp, smErr = c.client.Get(sitemapURL)
	}()
	fetchWG.Wait()

	// Check: Robots.txt
	if robotsErr == nil && robotsResp != nil {
		defer robotsResp.Body.Close()
		if robotsResp.StatusCode == 200 {
			result.Checks = append(result.Checks, AuditCheck{
				Category: "technical", Label: "robots.txt",
				Status: CheckPass, Severity: "medium",
				Detail:         "robots.txt found at " + robotsURL,
				Recommendation: "",
			})
		} else {
			result.Checks = append(result.Checks, AuditCheck{
				Category: "technical", Label: "robots.txt",
				Status: CheckWarning, Severity: "low",
				Detail:         "robots.txt not found (returned non-200).",
				Recommendation: "Create a robots.txt file to guide search engine crawlers.",
			})
		}
	}

	// Check: Sitemap.xml
	if smErr == nil && smResp != nil {
		defer smResp.Body.Close()
		if smResp.StatusCode == 200 {
			result.Checks = append(result.Checks, AuditCheck{
				Category: "technical", Label: "XML Sitemap",
				Status: CheckPass, Severity: "medium",
				Detail:         "sitemap.xml found at " + sitemapURL,
				Recommendation: "",
			})
		} else {
			result.Checks = append(result.Checks, AuditCheck{
				Category: "technical", Label: "XML Sitemap",
				Status: CheckWarning, Severity: "medium",
				Detail:         "sitemap.xml not found.",
				Recommendation: "Generate and submit an XML sitemap to Google Search Console.",
			})
		}
	}

	// ── 4. Calculate score ─────────────────────────────────────────────────
	total := len(result.Checks)
	passed := 0
	for _, ch := range result.Checks {
		if ch.Status == CheckPass {
			passed++
		}
	}
	if total > 0 {
		result.Score = (passed * 100) / total
	}

	return result, nil
}

// ── HTML parsing helpers ───────────────────────────────────────────────────

func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
		return strings.TrimSpace(n.FirstChild.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := extractTitle(c); v != "" {
			return v
		}
	}
	return ""
}

func extractMetaContent(n *html.Node, name string) string {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var nm, content string
		for _, a := range n.Attr {
			if a.Key == "name" {
				nm = strings.ToLower(a.Val)
			}
			if a.Key == "content" {
				content = a.Val
			}
		}
		if nm == strings.ToLower(name) {
			return content
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := extractMetaContent(c, name); v != "" {
			return v
		}
	}
	return ""
}

func extractMetaProperty(n *html.Node, property string) string {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var prop, content string
		for _, a := range n.Attr {
			if a.Key == "property" {
				prop = strings.ToLower(a.Val)
			}
			if a.Key == "content" {
				content = a.Val
			}
		}
		if prop == strings.ToLower(property) {
			return content
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := extractMetaProperty(c, property); v != "" {
			return v
		}
	}
	return ""
}

func extractH1s(n *html.Node) []string {
	var h1s []string
	if n.Type == html.ElementNode && n.Data == "h1" {
		text := extractText(n)
		if text != "" {
			h1s = append(h1s, text)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		h1s = append(h1s, extractH1s(c)...)
	}
	return h1s
}

func extractLinkHref(n *html.Node, rel string) string {
	if n.Type == html.ElementNode && n.Data == "link" {
		var r, href string
		for _, a := range n.Attr {
			if a.Key == "rel" {
				r = strings.ToLower(a.Val)
			}
			if a.Key == "href" {
				href = a.Val
			}
		}
		if r == rel {
			return href
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := extractLinkHref(c, rel); v != "" {
			return v
		}
	}
	return ""
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return strings.TrimSpace(sb.String())
}

func countImagesMissingAlt(n *html.Node) int {
	count := 0
	if n.Type == html.ElementNode && n.Data == "img" {
		hasAlt := false
		for _, a := range n.Attr {
			if a.Key == "alt" {
				hasAlt = true
				break
			}
		}
		if !hasAlt {
			count++
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		count += countImagesMissingAlt(c)
	}
	return count
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
