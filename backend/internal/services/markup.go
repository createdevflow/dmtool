package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const markupCategory = "markup"

// MarkupReport is JSON-LD on this page only — not a sitewide schema inventory.
type MarkupReport struct {
	Present     bool      `json:"present"`
	Types       []string  `json:"types,omitempty"`
	ScriptCount int       `json:"script_count"`
	ParseErrors int       `json:"parse_errors"`
	Status      string    `json:"status"`
	CheckedAt   time.Time `json:"checked_at"`
}

// InspectMarkup parses application/ld+json on this document.
func InspectMarkup(doc *html.Node, checkedAt time.Time) (MarkupReport, []AuditCheck) {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	report := MarkupReport{CheckedAt: checkedAt.UTC()}
	checked := formatRobotsChecked(checkedAt)

	scripts := collectJSONLD(doc)
	report.ScriptCount = len(scripts)
	typeSet := map[string]struct{}{}
	for _, raw := range scripts {
		if err := collectSchemaTypes([]byte(raw), typeSet); err != nil {
			report.ParseErrors++
		}
	}
	for t := range typeSet {
		report.Types = append(report.Types, t)
	}
	sort.Strings(report.Types)
	if len(report.Types) > 12 {
		report.Types = report.Types[:12]
	}
	report.Present = len(report.Types) > 0 || (report.ScriptCount > 0 && report.ParseErrors == 0)

	checks := make([]AuditCheck, 0, 2)
	switch {
	case report.ScriptCount == 0:
		checks = append(checks, markupCheck("JSON-LD", CheckWarning, "low",
			fmt.Sprintf("No JSON-LD structured data on this page. Last checked %s. Homepage only — not a sitewide schema crawl.", checked),
			"Add Schema.org JSON-LD if this page should be eligible for rich results."))
	case report.ParseErrors > 0 && len(report.Types) == 0:
		checks = append(checks, markupCheck("JSON-LD", CheckFail, "medium",
			fmt.Sprintf("%d JSON-LD script(s) on this page failed to parse. Last checked %s.", report.ParseErrors, checked),
			"Fix invalid JSON in application/ld+json scripts."))
	case report.ParseErrors > 0:
		checks = append(checks, markupCheck("JSON-LD", CheckWarning, "medium",
			fmt.Sprintf("Found types %s, but %d script(s) failed to parse. Last checked %s.", strings.Join(report.Types, ", "), report.ParseErrors, checked),
			"Fix invalid JSON-LD blocks on this page."))
	default:
		detail := "JSON-LD present on this page."
		if len(report.Types) > 0 {
			detail = "JSON-LD types on this page: " + strings.Join(report.Types, ", ") + "."
		}
		checks = append(checks, markupCheck("JSON-LD", CheckPass, "low",
			detail+" Last checked "+checked+". Not a rich-results validator.", ""))
	}

	report.Status = worstRobotsStatus(checks)
	return report, checks
}

func markupCheck(label, status, severity, detail, rec string) AuditCheck {
	return AuditCheck{
		Category:       markupCategory,
		Label:          label,
		Status:         status,
		Severity:       severity,
		Detail:         detail,
		Recommendation: rec,
	}
}

func collectJSONLD(n *html.Node) []string {
	var out []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "script" {
			typ := ""
			for _, a := range node.Attr {
				if strings.EqualFold(a.Key, "type") {
					typ = strings.ToLower(strings.TrimSpace(a.Val))
				}
			}
			if typ == "application/ld+json" {
				raw := strings.TrimSpace(extractText(node))
				if raw != "" {
					out = append(out, raw)
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func collectSchemaTypes(raw []byte, into map[string]struct{}) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	walkSchemaValue(v, into)
	return nil
}

func walkSchemaValue(v any, into map[string]struct{}) {
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			walkSchemaValue(item, into)
		}
	case map[string]any:
		if typ, ok := t["@type"]; ok {
			addSchemaType(typ, into)
		}
		if graph, ok := t["@graph"]; ok {
			walkSchemaValue(graph, into)
		}
	}
}

func addSchemaType(v any, into map[string]struct{}) {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s != "" {
			into[s] = struct{}{}
		}
	case []any:
		for _, item := range t {
			addSchemaType(item, into)
		}
	}
}
