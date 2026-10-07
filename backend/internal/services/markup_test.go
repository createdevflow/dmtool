package services

import (
	"strings"
	"testing"
	"time"
)

func TestInspectMarkup_TypesFromJSONLD(t *testing.T) {
	doc := mustHTML(t, `<html><head>
<script type="application/ld+json">{"@context":"https://schema.org","@type":"Organization","name":"Example"}</script>
<script type="application/ld+json">{"@graph":[{"@type":"WebSite"},{"@type":["WebPage","ItemPage"]}]}</script>
</head><body></body></html>`)
	report, checks := InspectMarkup(doc, time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	if !report.Present || report.ParseErrors != 0 {
		t.Fatalf("report=%+v", report)
	}
	joined := strings.Join(report.Types, ",")
	if !strings.Contains(joined, "Organization") || !strings.Contains(joined, "WebSite") || !strings.Contains(joined, "WebPage") {
		t.Fatalf("types=%v", report.Types)
	}
	if checkLabels(checks)["JSON-LD"] != CheckPass {
		t.Fatalf("checks=%+v", checks)
	}
}

func TestInspectMarkup_InvalidJSON(t *testing.T) {
	doc := mustHTML(t, `<html><head>
<script type="application/ld+json">{not json</script>
</head></html>`)
	report, checks := InspectMarkup(doc, time.Now())
	if report.ParseErrors != 1 {
		t.Fatalf("errors=%d", report.ParseErrors)
	}
	if checkLabels(checks)["JSON-LD"] != CheckFail {
		t.Fatalf("checks=%+v", checks)
	}
}

func TestInspectMarkup_Missing(t *testing.T) {
	doc := mustHTML(t, `<html><head></head><body></body></html>`)
	_, checks := InspectMarkup(doc, time.Now())
	if checkLabels(checks)["JSON-LD"] != CheckWarning {
		t.Fatalf("checks=%+v", checks)
	}
	if !strings.Contains(labelsDetail(checks, "JSON-LD"), "Homepage only") {
		t.Fatalf("expected homepage-only: %+v", checks)
	}
}
