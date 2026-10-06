package services

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const indexabilityCategory = "indexability"

// IndexabilityReport is the structured indexability outcome for one URL.
// It does not crawl other pages or claim a sitewide index.
type IndexabilityReport struct {
	URL          string    `json:"url"`
	Indexable    bool      `json:"indexable"`
	MetaRobots   string    `json:"meta_robots,omitempty"`
	XRobotsTag   string    `json:"x_robots_tag,omitempty"`
	Noindex      bool      `json:"noindex"`
	Nofollow     bool      `json:"nofollow"`
	Canonical    string    `json:"canonical,omitempty"`
	CanonicalRel string    `json:"canonical_rel,omitempty"` // match | other_path | other_host | missing | invalid
	Status       string    `json:"status"`
	CheckedAt    time.Time `json:"checked_at"`
}

type indexDirs struct {
	noindex  bool
	nofollow bool
	raw      []string
}

// InspectIndexability reads meta robots, X-Robots-Tag, and canonical on this page only.
func InspectIndexability(pageURL *url.URL, doc *html.Node, headers http.Header, checkedAt time.Time) (IndexabilityReport, []AuditCheck) {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	report := IndexabilityReport{CheckedAt: checkedAt.UTC(), Indexable: true}
	if pageURL != nil {
		report.URL = pageURL.String()
	}
	checked := formatRobotsChecked(checkedAt)

	meta := collectMetaByNames(doc, "robots", "googlebot")
	metaRaw := joinMetaContents(meta)
	report.MetaRobots = metaRaw
	metaDirs := parseIndexDirectives(metaRaw)

	headerRaw := ""
	if headers != nil {
		headerRaw = strings.Join(headers.Values("X-Robots-Tag"), ", ")
	}
	report.XRobotsTag = strings.TrimSpace(headerRaw)
	headerDirs := parseIndexDirectives(report.XRobotsTag)

	report.Noindex = metaDirs.noindex || headerDirs.noindex
	report.Nofollow = metaDirs.nofollow || headerDirs.nofollow
	report.Indexable = !report.Noindex

	canon := strings.TrimSpace(extractLinkHref(doc, "canonical"))
	report.Canonical = canon
	report.CanonicalRel = canonicalRelation(pageURL, canon)

	checks := make([]AuditCheck, 0, 3)

	switch {
	case metaDirs.noindex:
		checks = append(checks, indexabilityCheck("Meta robots", CheckFail, "high",
			fmt.Sprintf("This page has a noindex directive in meta robots (%s). Last checked %s. Homepage only — not a sitewide indexability crawl.",
				metaRaw, checked),
			"Remove noindex from the homepage meta robots tag if this URL should appear in Google."))
	case metaRaw == "":
		checks = append(checks, indexabilityCheck("Meta robots", CheckPass, "high",
			fmt.Sprintf("No meta robots tag on this page (default is index). Last checked %s.", checked), ""))
	case metaDirs.nofollow:
		checks = append(checks, indexabilityCheck("Meta robots", CheckWarning, "medium",
			fmt.Sprintf("Meta robots is nofollow without noindex (%s). The page may still be indexed. Last checked %s.", metaRaw, checked),
			"Use nofollow only when links on this page should not pass PageRank. It does not hide the page from the index."))
	default:
		checks = append(checks, indexabilityCheck("Meta robots", CheckPass, "high",
			fmt.Sprintf("Meta robots does not noindex this page (%s). Last checked %s.", metaRaw, checked), ""))
	}

	switch {
	case headerDirs.noindex:
		checks = append(checks, indexabilityCheck("X-Robots-Tag", CheckFail, "high",
			fmt.Sprintf("X-Robots-Tag noindexes this page (%s). Last checked %s. Header applies to this response only.",
				report.XRobotsTag, checked),
			"Remove noindex from X-Robots-Tag on the homepage if it should be indexed."))
	case report.XRobotsTag == "":
		checks = append(checks, indexabilityCheck("X-Robots-Tag", CheckPass, "medium",
			fmt.Sprintf("No X-Robots-Tag header on this response. Last checked %s.", checked), ""))
	case headerDirs.nofollow:
		checks = append(checks, indexabilityCheck("X-Robots-Tag", CheckWarning, "medium",
			fmt.Sprintf("X-Robots-Tag is nofollow without noindex (%s). Last checked %s.", report.XRobotsTag, checked),
			"nofollow on the header does not noindex the page."))
	default:
		checks = append(checks, indexabilityCheck("X-Robots-Tag", CheckPass, "medium",
			fmt.Sprintf("X-Robots-Tag does not noindex this page (%s). Last checked %s.", report.XRobotsTag, checked), ""))
	}

	switch report.CanonicalRel {
	case "missing":
		checks = append(checks, indexabilityCheck("Canonical URL", CheckWarning, "medium",
			fmt.Sprintf("No canonical tag on this page. Last checked %s.", checked),
			"Add <link rel=\"canonical\" href=\"...\"> on this URL so Google knows the preferred address."))
	case "invalid":
		checks = append(checks, indexabilityCheck("Canonical URL", CheckWarning, "medium",
			fmt.Sprintf("Canonical href is not a valid URL (%s).", canon),
			"Use an absolute http(s) canonical URL."))
	case "other_host":
		checks = append(checks, indexabilityCheck("Canonical URL", CheckWarning, "medium",
			fmt.Sprintf("Canonical points at a different host: %s. Checked this page only.", canon),
			"Confirm this homepage should consolidate to another domain."))
	case "other_path":
		checks = append(checks, indexabilityCheck("Canonical URL", CheckWarning, "low",
			fmt.Sprintf("Canonical points at a different path: %s. This is not a duplicate-content crawl of the rest of the site.", canon),
			"Use a self-canonical on the homepage unless this URL is intentionally a duplicate."))
	default:
		detail := "Canonical matches this page."
		if canon != "" {
			detail = "Canonical URL: " + canon + "."
		}
		checks = append(checks, indexabilityCheck("Canonical URL", CheckPass, "medium",
			detail+" Last checked "+checked+".", ""))
	}

	report.Status = worstRobotsStatus(checks)
	return report, checks
}

func indexabilityCheck(label, status, severity, detail, rec string) AuditCheck {
	return AuditCheck{
		Category:       indexabilityCategory,
		Label:          label,
		Status:         status,
		Severity:       severity,
		Detail:         detail,
		Recommendation: rec,
	}
}

func collectMetaByNames(n *html.Node, names ...string) []string {
	want := map[string]bool{}
	for _, name := range names {
		want[strings.ToLower(name)] = true
	}
	var out []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "meta" {
			var nm, content string
			for _, a := range node.Attr {
				if strings.EqualFold(a.Key, "name") {
					nm = strings.ToLower(strings.TrimSpace(a.Val))
				}
				if strings.EqualFold(a.Key, "content") {
					content = strings.TrimSpace(a.Val)
				}
			}
			if want[nm] && content != "" {
				out = append(out, nm+": "+content)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func joinMetaContents(items []string) string {
	return strings.Join(items, "; ")
}

func parseIndexDirectives(raw string) indexDirs {
	out := indexDirs{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	normalized := strings.ReplaceAll(raw, ";", ",")
	applyIndexDirectiveString(normalized, &out)
	return out
}

func applyIndexDirectiveString(raw string, out *indexDirs) {
	parts := strings.Split(raw, ",")
	pendingBot := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out.raw = append(out.raw, part)
		lower := strings.ToLower(part)
		if strings.HasPrefix(lower, "unavailable_after") || strings.HasPrefix(lower, "max-") {
			continue
		}
		if i := strings.Index(part, ":"); i >= 0 {
			bot := strings.ToLower(strings.TrimSpace(part[:i]))
			rest := strings.TrimSpace(part[i+1:])
			if relevantIndexBot(bot) {
				pendingBot = bot
				applyOneIndexDir(rest, out)
			} else {
				pendingBot = ""
			}
			continue
		}
		if pendingBot != "" && !relevantIndexBot(pendingBot) {
			continue
		}
		applyOneIndexDir(part, out)
	}
}

func applyOneIndexDir(dir string, out *indexDirs) {
	switch strings.ToLower(strings.TrimSpace(dir)) {
	case "noindex":
		out.noindex = true
	case "nofollow":
		out.nofollow = true
	case "none":
		out.noindex = true
		out.nofollow = true
	}
}

func relevantIndexBot(bot string) bool {
	bot = strings.ToLower(strings.TrimSpace(bot))
	if bot == "" || bot == "*" || bot == "robots" {
		return true
	}
	return strings.HasPrefix(bot, "googlebot")
}

func canonicalRelation(page *url.URL, raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "missing"
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return "invalid"
	}
	if page == nil {
		if ref.Scheme != "" && ref.Host != "" {
			return "match"
		}
		if !strings.EqualFold(ref.Scheme, "http") && !strings.EqualFold(ref.Scheme, "https") && ref.Host == "" {
			return "invalid"
		}
		return "match"
	}
	resolved := page.ResolveReference(ref)
	if resolved.Host == "" || (!strings.EqualFold(resolved.Scheme, "http") && !strings.EqualFold(resolved.Scheme, "https")) {
		return "invalid"
	}
	if !strings.EqualFold(resolved.Hostname(), page.Hostname()) {
		return "other_host"
	}
	if normalizeIndexPath(resolved) != normalizeIndexPath(page) {
		return "other_path"
	}
	return "match"
}

func normalizeIndexPath(u *url.URL) string {
	if u == nil {
		return ""
	}
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		p = "/"
	}
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}
	return strings.ToLower(p)
}
