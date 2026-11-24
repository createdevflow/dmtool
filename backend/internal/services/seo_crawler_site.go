package services

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// SiteCrawlOptions controls a bounded Site Audit crawl.
type SiteCrawlOptions struct {
	PageLimit     int
	RunCap        int
	Source        string // website | sitemap
	Scope         string // domain | subdomain | subfolder
	SubfolderPath string
	UserAgent     string
	Deadline      time.Time
}

// SitePageResult is one fetched HTML page in a Site Audit crawl.
type SitePageResult struct {
	URL           string
	StatusCode    int
	Title         string
	Description   string
	Score         int
	Checks        []AuditCheck
	LoadTimeMs    int64
	Indexable     bool
	Outlinks      int
	InternalLinks []string
}

// SiteCrawlResult is a bounded multi-page HTML crawl. PSI is not called.
type SiteCrawlResult struct {
	SeedURL       string
	CrawledAt     time.Time
	Pages         []SitePageResult
	SiteChecks    []AuditCheck
	Robots        RobotsReport
	HTTPS         HTTPSReport
	Sitemap       SitemapReport
	Indexability  IndexabilityReport
	International InternationalReport
	Markup        MarkupReport
	StoppedReason string
}

func (c *seoCrawler) CrawlSite(targetURL string, opts SiteCrawlOptions) (*SiteCrawlResult, error) {
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if opts.RunCap <= 0 {
		opts.RunCap = 8
	}
	if opts.PageLimit <= 0 {
		opts.PageLimit = opts.RunCap
	}
	capN := opts.PageLimit
	if capN > opts.RunCap {
		capN = opts.RunCap
	}
	if capN < 1 {
		capN = 1
	}
	ua := siteAuditUA(opts.UserAgent)
	deadline := opts.Deadline
	if deadline.IsZero() {
		deadline = time.Now().Add(22 * time.Second)
	}

	result := &SiteCrawlResult{
		SeedURL:    targetURL,
		CrawledAt:  time.Now().UTC(),
		Pages:      []SitePageResult{},
		SiteChecks: []AuditCheck{},
	}

	client := c.siteClient()
	seedPage, seedDoc, seedResp, seedFinal, seedBodyOK := c.fetchSitePage(client, targetURL, ua, deadline)
	if !seedBodyOK && seedPage.StatusCode == 0 {
		result.StoppedReason = "seed fetch failed"
		result.Pages = append(result.Pages, seedPage)
		return result, nil
	}
	result.Pages = append(result.Pages, seedPage)

	if seedResp != nil && seedDoc != nil && seedFinal != nil {
		httpsReport, httpsChecks := InspectHTTPS(client, seedFinal, seedDoc, seedResp.Header, result.CrawledAt)
		result.HTTPS = httpsReport
		result.SiteChecks = append(result.SiteChecks, httpsChecks...)

		idxReport, idxChecks := InspectIndexability(seedFinal, seedDoc, seedResp.Header, result.CrawledAt)
		result.Indexability = idxReport
		seedPage.Indexable = idxReport.Indexable
		result.SiteChecks = append(result.SiteChecks, idxChecks...)

		intlReport, intlChecks := InspectInternational(seedFinal, seedDoc, result.CrawledAt)
		result.International = intlReport
		result.SiteChecks = append(result.SiteChecks, intlChecks...)

		markupReport, markupChecks := InspectMarkup(seedDoc, result.CrawledAt)
		result.Markup = markupReport
		result.SiteChecks = append(result.SiteChecks, markupChecks...)

		robotsURL := fmt.Sprintf("%s://%s/robots.txt", seedFinal.Scheme, seedFinal.Host)
		robotsReport, robotsChecks := InspectRobots(client, robotsURL, result.CrawledAt)
		result.Robots = robotsReport
		result.SiteChecks = append(result.SiteChecks, robotsChecks...)

		sitemapReport, sitemapChecks := InspectSitemap(client, robotsReport.Sitemaps, seedFinal, result.CrawledAt)
		result.Sitemap = sitemapReport
		result.SiteChecks = append(result.SiteChecks, sitemapChecks...)

		result.Pages[0] = seedPage
	}

	queue := []string{}
	seen := map[string]bool{normalizeCrawlURL(targetURL): true}
	if seedFinal != nil {
		seen[normalizeCrawlURL(seedFinal.String())] = true
	}
	addLinks := func(links []string) {
		for _, href := range links {
			n := normalizeCrawlURL(href)
			if n == "" || seen[n] {
				continue
			}
			if !inCrawlScope(parsed, href, opts.Scope, opts.SubfolderPath) {
				continue
			}
			seen[n] = true
			queue = append(queue, href)
		}
	}
	addLinks(seedPage.InternalLinks)
	if strings.EqualFold(opts.Source, "sitemap") {
		addLinks(result.Sitemap.Sample)
	}

	for len(result.Pages) < capN && len(queue) > 0 {
		if time.Now().After(deadline) {
			result.StoppedReason = "time budget"
			break
		}
		next := queue[0]
		queue = queue[1:]
		page, _, _, _, ok := c.fetchSitePage(client, next, ua, deadline)
		if !ok && page.StatusCode == 0 {
			continue
		}
		result.Pages = append(result.Pages, page)
		addLinks(page.InternalLinks)
	}
	if result.StoppedReason == "" {
		if len(result.Pages) >= capN && (len(queue) > 0 || capN < opts.PageLimit) {
			result.StoppedReason = "page cap"
		} else {
			result.StoppedReason = "complete"
		}
	}
	return result, nil
}

func (c *seoCrawler) siteClient() *http.Client {
	if c != nil && c.client != nil {
		return &http.Client{
			Timeout:       8 * time.Second,
			CheckRedirect: c.client.CheckRedirect,
		}
	}
	return &http.Client{Timeout: 8 * time.Second}
}

func (c *seoCrawler) fetchSitePage(client *http.Client, targetURL, ua string, deadline time.Time) (SitePageResult, *html.Node, *http.Response, *url.URL, bool) {
	page := SitePageResult{URL: targetURL, InternalLinks: []string{}}
	if time.Now().After(deadline) {
		return page, nil, nil, nil, false
	}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return page, nil, nil, nil, false
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	start := time.Now()
	resp, err := client.Do(req)
	page.LoadTimeMs = time.Since(start).Milliseconds()
	if err != nil {
		page.Checks = append(page.Checks, AuditCheck{
			Category: "accessibility", Label: "Page Reachability",
			Status: CheckFail, Severity: "high",
			Detail:         "Failed to fetch the page: " + err.Error(),
			Recommendation: "Ensure the URL is publicly accessible and returns a 200 OK response.",
		})
		page.Score = 0
		return page, nil, nil, nil, false
	}
	defer resp.Body.Close()
	page.StatusCode = resp.StatusCode
	finalURL := targetURL
	var finalParsed *url.URL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
		finalParsed = resp.Request.URL
		page.URL = normalizeCrawlURL(finalURL)
	} else if u, err := url.Parse(targetURL); err == nil {
		finalParsed = u
		page.URL = normalizeCrawlURL(targetURL)
	}

	if resp.StatusCode == 200 {
		page.Checks = append(page.Checks, AuditCheck{
			Category: "accessibility", Label: "HTTP Status",
			Status: CheckPass, Severity: "high",
			Detail: fmt.Sprintf("Page returned HTTP %d OK.", resp.StatusCode),
		})
	} else {
		page.Checks = append(page.Checks, AuditCheck{
			Category: "accessibility", Label: "HTTP Status",
			Status: CheckFail, Severity: "high",
			Detail:         fmt.Sprintf("Page returned HTTP %d.", resp.StatusCode),
			Recommendation: "Ensure the URL returns a 200 OK status code for search engines to index it.",
		})
	}

	if page.LoadTimeMs < 1500 {
		page.Checks = append(page.Checks, AuditCheck{
			Category: "performance", Label: "Page Load Speed",
			Status: CheckPass, Severity: "medium",
			Detail: fmt.Sprintf("Page loaded in %dms — excellent.", page.LoadTimeMs),
		})
	} else if page.LoadTimeMs < 3000 {
		page.Checks = append(page.Checks, AuditCheck{
			Category: "performance", Label: "Page Load Speed",
			Status: CheckWarning, Severity: "medium",
			Detail:         fmt.Sprintf("Page loaded in %dms — acceptable but improvable.", page.LoadTimeMs),
			Recommendation: "Aim for under 1.5s. Compress images, enable caching, use a CDN.",
		})
	} else {
		page.Checks = append(page.Checks, AuditCheck{
			Category: "performance", Label: "Page Load Speed",
			Status: CheckFail, Severity: "high",
			Detail:         fmt.Sprintf("Page is slow: %dms load time.", page.LoadTimeMs),
			Recommendation: "Critical: page loads over 3 seconds.",
		})
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		page.Score = scoreChecks(page.Checks)
		return page, nil, resp, finalParsed, true
	}
	doc, err := html.Parse(strings.NewReader(string(bodyBytes)))
	if err != nil {
		page.Score = scoreChecks(page.Checks)
		return page, nil, resp, finalParsed, true
	}

	title := extractTitle(doc)
	page.Title = title
	metaDesc := extractMetaContent(doc, "description")
	page.Description = metaDesc
	h1s := extractH1s(doc)
	pageChecksFromOnPage(&page, title, metaDesc, h1s, extractMetaProperty(doc, "og:title"), extractMetaProperty(doc, "og:description"), extractMetaProperty(doc, "og:image"), extractMetaContent(doc, "viewport"), countImagesMissingAlt(doc))

	idxReport, _ := InspectIndexability(finalParsed, doc, resp.Header, time.Now().UTC())
	page.Indexable = idxReport.Indexable

	links := extractInternalAnchors(doc, finalParsed)
	page.InternalLinks = links
	page.Outlinks = len(links)
	page.Score = scoreChecks(page.Checks)
	return page, doc, resp, finalParsed, true
}

func pageChecksFromOnPage(page *SitePageResult, title, metaDesc string, h1s []string, ogTitle, ogDesc, ogImage, viewport string, imagesMissingAlt int) {
	titleLen := len(title)
	switch {
	case title == "":
		page.Checks = append(page.Checks, AuditCheck{Category: "meta", Label: "Title Tag", Status: CheckFail, Severity: "high", Detail: "No <title> tag found.", Recommendation: "Add a unique, descriptive title tag between 10–60 characters."})
	case titleLen < 10:
		page.Checks = append(page.Checks, AuditCheck{Category: "meta", Label: "Title Tag", Status: CheckWarning, Severity: "medium", Detail: fmt.Sprintf("Title tag is too short (%d chars).", titleLen)})
	case titleLen > 60:
		page.Checks = append(page.Checks, AuditCheck{Category: "meta", Label: "Title Tag", Status: CheckWarning, Severity: "medium", Detail: fmt.Sprintf("Title tag is too long (%d chars).", titleLen)})
	default:
		page.Checks = append(page.Checks, AuditCheck{Category: "meta", Label: "Title Tag", Status: CheckPass, Severity: "high", Detail: fmt.Sprintf("Title tag present (%d chars): %q", titleLen, title)})
	}

	descLen := len(metaDesc)
	switch {
	case metaDesc == "":
		page.Checks = append(page.Checks, AuditCheck{Category: "meta", Label: "Meta Description", Status: CheckFail, Severity: "high", Detail: "No meta description found."})
	case descLen < 50:
		page.Checks = append(page.Checks, AuditCheck{Category: "meta", Label: "Meta Description", Status: CheckWarning, Severity: "medium", Detail: fmt.Sprintf("Meta description is too short (%d chars).", descLen)})
	case descLen > 160:
		page.Checks = append(page.Checks, AuditCheck{Category: "meta", Label: "Meta Description", Status: CheckWarning, Severity: "low", Detail: fmt.Sprintf("Meta description is too long (%d chars).", descLen)})
	default:
		page.Checks = append(page.Checks, AuditCheck{Category: "meta", Label: "Meta Description", Status: CheckPass, Severity: "high", Detail: fmt.Sprintf("Meta description present (%d chars).", descLen)})
	}

	switch len(h1s) {
	case 0:
		page.Checks = append(page.Checks, AuditCheck{Category: "content", Label: "H1 Heading", Status: CheckFail, Severity: "high", Detail: "No <h1> tag found on the page."})
	case 1:
		page.Checks = append(page.Checks, AuditCheck{Category: "content", Label: "H1 Heading", Status: CheckPass, Severity: "high", Detail: fmt.Sprintf("One H1 found: %q", truncate(h1s[0], 80))})
	default:
		page.Checks = append(page.Checks, AuditCheck{Category: "content", Label: "H1 Heading", Status: CheckWarning, Severity: "medium", Detail: fmt.Sprintf("Multiple H1 tags found (%d).", len(h1s))})
	}

	if ogTitle != "" && ogDesc != "" && ogImage != "" {
		page.Checks = append(page.Checks, AuditCheck{Category: "social", Label: "Open Graph Tags", Status: CheckPass, Severity: "medium", Detail: "og:title, og:description, and og:image are all present."})
	} else if ogTitle != "" || ogDesc != "" {
		page.Checks = append(page.Checks, AuditCheck{Category: "social", Label: "Open Graph Tags", Status: CheckWarning, Severity: "low", Detail: "Some Open Graph tags are missing."})
	} else {
		page.Checks = append(page.Checks, AuditCheck{Category: "social", Label: "Open Graph Tags", Status: CheckFail, Severity: "medium", Detail: "No Open Graph tags found."})
	}

	if viewport != "" {
		page.Checks = append(page.Checks, AuditCheck{Category: "mobile", Label: "Mobile Viewport", Status: CheckPass, Severity: "high", Detail: "Viewport meta tag present."})
	} else {
		page.Checks = append(page.Checks, AuditCheck{Category: "mobile", Label: "Mobile Viewport", Status: CheckFail, Severity: "high", Detail: "No viewport meta tag found."})
	}

	if imagesMissingAlt == 0 {
		page.Checks = append(page.Checks, AuditCheck{Category: "content", Label: "Image Alt Text", Status: CheckPass, Severity: "medium", Detail: "All sampled images have alt attributes."})
	} else {
		page.Checks = append(page.Checks, AuditCheck{Category: "content", Label: "Image Alt Text", Status: CheckWarning, Severity: "medium", Detail: fmt.Sprintf("%d image(s) are missing alt text.", imagesMissingAlt)})
	}
}

func scoreChecks(checks []AuditCheck) int {
	if len(checks) == 0 {
		return 0
	}
	passed := 0
	for _, ch := range checks {
		if ch.Status == CheckPass {
			passed++
		}
	}
	return (passed * 100) / len(checks)
}

func siteAuditUA(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "mobile":
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1 DMToolSiteAudit/1.0"
	case "googlebot":
		return "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	case "openai-search":
		return "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.2; +https://openai.com/gptbot)"
	default:
		return "Mozilla/5.0 (compatible; DMToolSiteAudit/1.0; +https://dmtool.app)"
	}
}

func extractInternalAnchors(n *html.Node, base *url.URL) []string {
	if base == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, a := range node.Attr {
				if a.Key != "href" {
					continue
				}
				href := strings.TrimSpace(a.Val)
				if href == "" || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "tel:") {
					continue
				}
				ref, err := base.Parse(href)
				if err != nil || ref.Host == "" {
					continue
				}
				if !sameSiteHost(base.Hostname(), ref.Hostname()) {
					continue
				}
				ref.Fragment = ""
				nurl := normalizeCrawlURL(ref.String())
				if nurl == "" || seen[nurl] {
					continue
				}
				seen[nurl] = true
				out = append(out, nurl)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func normalizeCrawlURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return u.Scheme + "://" + strings.ToLower(u.Host) + u.Path
}

func sameSiteHost(a, b string) bool {
	a = strings.ToLower(strings.TrimPrefix(a, "www."))
	b = strings.ToLower(strings.TrimPrefix(b, "www."))
	return a == b
}

func inCrawlScope(seed *url.URL, candidate, scope, subfolder string) bool {
	u, err := url.Parse(candidate)
	if err != nil || seed == nil {
		return false
	}
	switch strings.ToLower(scope) {
	case "subdomain":
		return strings.EqualFold(u.Hostname(), seed.Hostname())
	case "subfolder":
		if !strings.EqualFold(u.Hostname(), seed.Hostname()) && !sameSiteHost(u.Hostname(), seed.Hostname()) {
			return false
		}
		prefix := subfolder
		if prefix == "" {
			prefix = seed.Path
		}
		if prefix == "" {
			prefix = "/"
		}
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}
		return strings.HasPrefix(u.Path, prefix)
	default:
		return sameSiteHost(u.Hostname(), seed.Hostname())
	}
}
