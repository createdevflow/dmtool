package services

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const internationalCategory = "international"

// HreflangLink is one rel=alternate hreflang on this page.
type HreflangLink struct {
	Lang string `json:"lang"`
	Href string `json:"href"`
}

// InternationalReport is homepage-only language targeting detection.
// It does not read Google Search Console country targeting.
type InternationalReport struct {
	URL        string         `json:"url"`
	HTMLLang   string         `json:"html_lang,omitempty"`
	Hreflang   []HreflangLink `json:"hreflang,omitempty"`
	URLPattern string         `json:"url_pattern,omitempty"` // locale segment or empty
	Status     string         `json:"status"`
	CheckedAt  time.Time      `json:"checked_at"`
}

// InspectInternational reads html lang, hreflang links, and a locale URL segment on this page only.
func InspectInternational(pageURL *url.URL, doc *html.Node, checkedAt time.Time) (InternationalReport, []AuditCheck) {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	report := InternationalReport{CheckedAt: checkedAt.UTC()}
	if pageURL != nil {
		report.URL = pageURL.String()
		report.URLPattern = localePathSegment(pageURL)
	}
	checked := formatRobotsChecked(checkedAt)
	report.HTMLLang = extractHTMLLang(doc)
	report.Hreflang = extractHreflang(doc)

	checks := make([]AuditCheck, 0, 3)
	if report.HTMLLang == "" {
		checks = append(checks, internationalCheck("HTML lang", CheckWarning, "low",
			fmt.Sprintf("No lang attribute on the html element. Last checked %s. Homepage only — not a sitewide hreflang crawl.", checked),
			"Add <html lang=\"en\"> (or the page language) so browsers and crawlers know the language."))
	} else {
		checks = append(checks, internationalCheck("HTML lang", CheckPass, "low",
			fmt.Sprintf("html lang is %s. Last checked %s.", report.HTMLLang, checked), ""))
	}

	if len(report.Hreflang) == 0 {
		checks = append(checks, internationalCheck("Hreflang", CheckWarning, "low",
			fmt.Sprintf("No hreflang alternate links on this page. Fine for a single-language site. Last checked %s.", checked),
			"Add rel=alternate hreflang only if this site serves multiple languages or countries."))
	} else {
		parts := make([]string, 0, len(report.Hreflang))
		for _, h := range report.Hreflang {
			parts = append(parts, h.Lang)
		}
		checks = append(checks, internationalCheck("Hreflang", CheckPass, "low",
			fmt.Sprintf("Hreflang on this page: %s. Last checked %s. Not Google's country targeting.", strings.Join(parts, ", "), checked), ""))
	}

	if report.URLPattern != "" {
		checks = append(checks, internationalCheck("URL language pattern", CheckPass, "low",
			fmt.Sprintf("Path looks locale-prefixed (%s). Detection only — not Search Console targeting. Last checked %s.", report.URLPattern, checked), ""))
	} else {
		checks = append(checks, internationalCheck("URL language pattern", CheckPass, "low",
			fmt.Sprintf("No locale prefix in this URL path. Last checked %s.", checked), ""))
	}

	report.Status = worstRobotsStatus(checks)
	return report, checks
}

func internationalCheck(label, status, severity, detail, rec string) AuditCheck {
	return AuditCheck{
		Category:       internationalCategory,
		Label:          label,
		Status:         status,
		Severity:       severity,
		Detail:         detail,
		Recommendation: rec,
	}
}

func extractHTMLLang(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.ElementNode && n.Data == "html" {
		for _, a := range n.Attr {
			if strings.EqualFold(a.Key, "lang") {
				return strings.TrimSpace(a.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := extractHTMLLang(c); v != "" {
			return v
		}
	}
	return ""
}

func extractHreflang(n *html.Node) []HreflangLink {
	var out []HreflangLink
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "link" {
			var rel, href, hl string
			for _, a := range node.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(strings.TrimSpace(a.Val))
				case "href":
					href = strings.TrimSpace(a.Val)
				case "hreflang":
					hl = strings.TrimSpace(a.Val)
				}
			}
			if strings.Contains(rel, "alternate") && hl != "" {
				out = append(out, HreflangLink{Lang: hl, Href: href})
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Lang == out[j].Lang {
			return out[i].Href < out[j].Href
		}
		return out[i].Lang < out[j].Lang
	})
	return out
}

func localePathSegment(u *url.URL) string {
	if u == nil {
		return ""
	}
	p := strings.Trim(u.EscapedPath(), "/")
	if p == "" {
		return ""
	}
	seg := strings.Split(p, "/")[0]
	seg = strings.ToLower(seg)
	if isLocaleSegment(seg) {
		return seg
	}
	return ""
}

func isLocaleSegment(seg string) bool {
	if len(seg) == 2 && isAlpha(seg) {
		return true
	}
	if len(seg) == 5 && seg[2] == '-' && isAlpha(seg[:2]) && isAlpha(seg[3:]) {
		return true
	}
	return false
}

func isAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}
