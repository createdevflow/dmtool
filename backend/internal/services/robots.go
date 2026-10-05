package services

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	robotsCategory = "robots"
	robotsMaxBytes = 512 * 1024
)

// RobotsReport is the structured robots.txt outcome from an audit.
type RobotsReport struct {
	URL              string    `json:"url"`
	Available        bool      `json:"available"`
	Accessible       bool      `json:"accessible"`
	StatusCode       int       `json:"status_code,omitempty"`
	FetchError       string    `json:"fetch_error,omitempty"`
	SitemapDeclared  bool      `json:"sitemap_declared"`
	Sitemaps         []string  `json:"sitemaps,omitempty"`
	BlocksAll        bool      `json:"blocks_all"`
	BlockedImportant []string  `json:"blocked_important,omitempty"`
	SyntaxWarnings   []string  `json:"syntax_warnings,omitempty"`
	CheckedAt        time.Time `json:"checked_at"`
	Status           string    `json:"status"`
}

type robotsGroup struct {
	agents    []string
	disallows []string
	allows    []string
}

type parsedRobots struct {
	sitemaps       []string
	groups         []robotsGroup
	syntaxWarnings []string
	looksLikeHTML  bool
	empty          bool
}

var importantRobotsPaths = []struct {
	path string
	name string
}{
	{"/", "entire site"},
	{"/sitemap.xml", "XML sitemap"},
	{"/index.html", "homepage"},
	{"/css/", "CSS"},
	{"/js/", "JavaScript"},
	{"/static/", "static assets"},
	{"/assets/", "assets"},
	{"/_next/", "Next.js assets"},
}

// InspectRobots fetches robots.txt and returns a report plus audit checks.
func InspectRobots(client *http.Client, robotsURL string, checkedAt time.Time) (RobotsReport, []AuditCheck) {
	if client == nil {
		client = http.DefaultClient
	}
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}

	report := RobotsReport{URL: robotsURL, CheckedAt: checkedAt.UTC()}
	req, err := http.NewRequest(http.MethodGet, robotsURL, nil)
	if err != nil {
		report.FetchError = err.Error()
		report.Status = CheckFail
		return report, robotsFetchFailChecks(robotsURL, checkedAt, "Could not build request: "+err.Error())
	}
	req.Header.Set("User-Agent", "DMTool-SEOCrawler/2.0 (+https://dmtool.app)")
	req.Header.Set("Accept", "text/plain,*/*")

	resp, err := client.Do(req)
	if err != nil {
		report.FetchError = err.Error()
		report.Status = CheckFail
		return report, robotsFetchFailChecks(robotsURL, checkedAt, "Could not fetch robots.txt: "+err.Error())
	}
	defer resp.Body.Close()
	report.StatusCode = resp.StatusCode

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, robotsMaxBytes+1))
	if err != nil {
		report.FetchError = err.Error()
		report.Status = CheckFail
		return report, robotsFetchFailChecks(robotsURL, checkedAt, "Could not read robots.txt: "+err.Error())
	}
	truncated := len(bodyBytes) > robotsMaxBytes
	if truncated {
		bodyBytes = bodyBytes[:robotsMaxBytes]
	}
	body := string(bodyBytes)

	if resp.StatusCode == http.StatusNotFound {
		report.Available = false
		report.Accessible = true
		checks := []AuditCheck{
			robotsCheck("robots.txt availability", CheckWarning, "low",
				fmt.Sprintf("No robots.txt at %s (HTTP 404). Search engines treat this as allow-all. Last checked %s.",
					robotsURL, formatRobotsChecked(checkedAt)),
				"Add a robots.txt at the site root to declare a sitemap and any paths that should stay out of the index."),
			robotsCheck("Sitemap declaration", CheckWarning, "medium",
				"robots.txt is missing, so it cannot declare a sitemap.",
				"Create robots.txt with a Sitemap: line pointing at your XML sitemap."),
		}
		report.Status = worstRobotsStatus(checks)
		return report, checks
	}

	if resp.StatusCode != http.StatusOK {
		report.Available = false
		report.Accessible = false
		report.Status = CheckFail
		return report, []AuditCheck{
			robotsCheck("robots.txt availability", CheckFail, "high",
				fmt.Sprintf("robots.txt returned HTTP %d at %s. Last checked %s.",
					resp.StatusCode, robotsURL, formatRobotsChecked(checkedAt)),
				"Serve robots.txt as HTTP 200 text/plain at the site root. 5xx/403 blocks crawlers from reading rules."),
			robotsCheck("robots.txt accessibility", CheckFail, "high",
				fmt.Sprintf("File is not readable (HTTP %d).", resp.StatusCode),
				"Fix server permissions so Googlebot can GET /robots.txt without auth."),
		}
	}

	report.Available = true
	parsed := parseRobotsTxt(body)
	if truncated {
		parsed.syntaxWarnings = append(parsed.syntaxWarnings, "File exceeds 512 KiB; only the first 512 KiB was parsed.")
	}

	if parsed.looksLikeHTML {
		report.Accessible = false
		report.Status = CheckFail
		return report, []AuditCheck{
			robotsCheck("robots.txt availability", CheckPass, "medium",
				fmt.Sprintf("HTTP 200 at %s. Last checked %s.", robotsURL, formatRobotsChecked(checkedAt)), ""),
			robotsCheck("robots.txt accessibility", CheckFail, "high",
				"Response looks like HTML, not a robots.txt file (often a custom 404 page).",
				"Serve a plain-text robots.txt. Do not redirect /robots.txt to the homepage."),
		}
	}

	report.Accessible = true
	report.Sitemaps = parsed.sitemaps
	report.SitemapDeclared = len(parsed.sitemaps) > 0
	disallows, allows := selectRobotsRules(parsed.groups)
	report.BlocksAll = pathBlockedByRobots(disallows, allows, "/")
	report.BlockedImportant = blockedImportantPaths(disallows, allows)
	report.SyntaxWarnings = parsed.syntaxWarnings

	checks := make([]AuditCheck, 0, 6)
	checks = append(checks, robotsCheck("robots.txt availability", CheckPass, "medium",
		fmt.Sprintf("robots.txt found at %s. Last checked %s.", robotsURL, formatRobotsChecked(checkedAt)), ""))
	if parsed.empty {
		checks = append(checks, robotsCheck("robots.txt accessibility", CheckWarning, "low",
			"robots.txt is empty. Crawlers treat this as allow-all.",
			"Add User-agent and Sitemap directives so crawlers know how to treat the site."))
	} else {
		checks = append(checks, robotsCheck("robots.txt accessibility", CheckPass, "medium",
			"robots.txt is readable as plain text.", ""))
	}

	if len(parsed.syntaxWarnings) > 0 {
		checks = append(checks, robotsCheck("robots.txt errors", CheckWarning, "low",
			strings.Join(parsed.syntaxWarnings, " "),
			"Use User-agent, Disallow, Allow, and Sitemap directives. Other lines are ignored."))
	} else if !parsed.empty {
		checks = append(checks, robotsCheck("robots.txt errors", CheckPass, "low",
			"No syntax issues detected in User-agent / Disallow / Allow / Sitemap lines.", ""))
	}

	if report.BlocksAll {
		checks = append(checks, robotsCheck("Important URLs blocked", CheckFail, "high",
			"Disallow: / blocks the entire site from Googlebot (User-agent * or Googlebot).",
			"Remove Disallow: / unless this environment must stay unindexed."))
	} else if len(report.BlockedImportant) > 0 {
		checks = append(checks, robotsCheck("Important URLs blocked", CheckWarning, "medium",
			"These paths are disallowed: "+strings.Join(report.BlockedImportant, ", ")+". Blocking CSS/JS can hurt rendering and ranking.",
			"Allow CSS, JS, and the sitemap. Only Disallow private or duplicate URL spaces."))
	} else {
		checks = append(checks, robotsCheck("Important URLs blocked", CheckPass, "high",
			"Homepage, sitemap, and common asset paths are not blocked for Googlebot.", ""))
	}

	if report.SitemapDeclared {
		checks = append(checks, robotsCheck("Sitemap declaration", CheckPass, "medium",
			"Sitemap: "+strings.Join(report.Sitemaps, ", "), ""))
	} else {
		checks = append(checks, robotsCheck("Sitemap declaration", CheckWarning, "medium",
			"No Sitemap: directive in robots.txt.",
			"Add a Sitemap: line in robots.txt pointing at your XML sitemap."))
	}

	report.Status = worstRobotsStatus(checks)
	return report, checks
}

func robotsFetchFailChecks(robotsURL string, checkedAt time.Time, detail string) []AuditCheck {
	return []AuditCheck{
		robotsCheck("robots.txt availability", CheckFail, "high",
			detail+" Last checked "+formatRobotsChecked(checkedAt)+". URL: "+robotsURL,
			"Ensure the host is reachable and /robots.txt is served publicly."),
		robotsCheck("robots.txt accessibility", CheckFail, "high",
			"File could not be downloaded.",
			"Fix DNS, TLS, or firewall so the crawler can GET robots.txt."),
	}
}

func robotsCheck(label, status, severity, detail, rec string) AuditCheck {
	return AuditCheck{
		Category:       robotsCategory,
		Label:          label,
		Status:         status,
		Severity:       severity,
		Detail:         detail,
		Recommendation: rec,
	}
}

func formatRobotsChecked(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

func worstRobotsStatus(checks []AuditCheck) string {
	status := CheckPass
	for _, c := range checks {
		if c.Status == CheckFail {
			return CheckFail
		}
		if c.Status == CheckWarning {
			status = CheckWarning
		}
	}
	return status
}

func parseRobotsTxt(body string) parsedRobots {
	out := parsedRobots{}
	trim := strings.TrimSpace(body)
	if trim == "" {
		out.empty = true
		return out
	}
	lower := strings.ToLower(trim)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") {
		out.looksLikeHTML = true
		return out
	}

	var current *robotsGroup
	startGroup := func() {
		out.groups = append(out.groups, robotsGroup{})
		current = &out.groups[len(out.groups)-1]
	}

	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
			if line == "" {
				continue
			}
		}
		key, val, ok := splitRobotsField(line)
		if !ok {
			out.syntaxWarnings = append(out.syntaxWarnings, "Ignored line (not a field:value pair): "+truncate(line, 80))
			continue
		}
		switch key {
		case "user-agent":
			if current == nil || len(current.disallows)+len(current.allows) > 0 {
				startGroup()
			}
			current.agents = append(current.agents, strings.ToLower(strings.TrimSpace(val)))
		case "disallow":
			if current == nil {
				startGroup()
				current.agents = []string{"*"}
			}
			current.disallows = append(current.disallows, val)
		case "allow":
			if current == nil {
				startGroup()
				current.agents = []string{"*"}
			}
			current.allows = append(current.allows, val)
		case "sitemap":
			if strings.TrimSpace(val) != "" {
				out.sitemaps = append(out.sitemaps, strings.TrimSpace(val))
			}
		case "crawl-delay", "host":
			// Valid for some crawlers; Google ignores. Not an error.
		default:
			out.syntaxWarnings = append(out.syntaxWarnings, "Unknown directive "+key+".")
		}
	}
	if len(out.syntaxWarnings) > 8 {
		out.syntaxWarnings = append(out.syntaxWarnings[:8], fmt.Sprintf("…and %d more unknown lines.", len(out.syntaxWarnings)-8))
	}
	return out
}

func splitRobotsField(line string) (key, val string, ok bool) {
	i := strings.Index(line, ":")
	if i <= 0 {
		return "", "", false
	}
	key = strings.ToLower(strings.TrimSpace(line[:i]))
	val = strings.TrimSpace(line[i+1:])
	return key, val, key != ""
}

func selectRobotsRules(groups []robotsGroup) (disallows, allows []string) {
	var googlebot, star []robotsGroup
	for _, g := range groups {
		if robotsHasAgent(g, "googlebot") {
			googlebot = append(googlebot, g)
		}
		if robotsHasAgent(g, "*") {
			star = append(star, g)
		}
	}
	use := star
	if len(googlebot) > 0 {
		use = googlebot
	}
	for _, g := range use {
		disallows = append(disallows, g.disallows...)
		allows = append(allows, g.allows...)
	}
	return
}

func robotsHasAgent(g robotsGroup, name string) bool {
	for _, a := range g.agents {
		if a == name {
			return true
		}
	}
	return false
}

func pathBlockedByRobots(disallows, allows []string, path string) bool {
	dPat, dOK := longestRobotsMatch(disallows, path)
	if !dOK {
		return false
	}
	aPat, aOK := longestRobotsMatch(allows, path)
	if aOK && len(aPat) >= len(dPat) {
		return false
	}
	return true
}

func longestRobotsMatch(patterns []string, path string) (string, bool) {
	best := ""
	ok := false
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if robotsPatternMatches(p, path) && len(p) >= len(best) {
			best = p
			ok = true
		}
	}
	return best, ok
}

func robotsPatternMatches(pattern, path string) bool {
	end := strings.HasSuffix(pattern, "$")
	p := strings.TrimSuffix(pattern, "$")
	if strings.Contains(p, "*") {
		return robotsWildcard(p, path, end)
	}
	if end {
		return path == p
	}
	return strings.HasPrefix(path, p)
}

func robotsWildcard(pattern, path string, end bool) bool {
	parts := strings.Split(pattern, "*")
	rest := path
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(rest, part)
		if idx < 0 {
			return false
		}
		if i == 0 && !strings.HasPrefix(rest, part) {
			return false
		}
		rest = rest[idx+len(part):]
	}
	if end {
		return rest == ""
	}
	return true
}

func blockedImportantPaths(disallows, allows []string) []string {
	if pathBlockedByRobots(disallows, allows, "/") {
		return []string{"entire site (/)"}
	}
	var out []string
	seen := map[string]bool{}
	for _, item := range importantRobotsPaths {
		if item.path == "/" {
			continue
		}
		if pathBlockedByRobots(disallows, allows, item.path) && !seen[item.name] {
			seen[item.name] = true
			out = append(out, item.name+" ("+item.path+")")
		}
	}
	for _, d := range disallows {
		dl := strings.ToLower(strings.TrimSpace(d))
		if strings.Contains(dl, ".css") && !seen["CSS"] {
			seen["CSS"] = true
			out = append(out, "CSS ("+d+")")
		}
		if strings.Contains(dl, ".js") && !seen["JavaScript"] {
			seen["JavaScript"] = true
			out = append(out, "JavaScript ("+d+")")
		}
	}
	return out
}
