package services

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestInspectInternational_LangAndHreflang(t *testing.T) {
	doc := mustHTML(t, `<html lang="en-GB"><head>
<link rel="alternate" hreflang="en" href="https://example.com/">
<link rel="alternate" hreflang="fr" href="https://example.com/fr/">
</head><body></body></html>`)
	page, _ := url.Parse("https://example.com/")
	report, checks := InspectInternational(page, doc, time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	if report.HTMLLang != "en-GB" {
		t.Fatalf("lang=%s", report.HTMLLang)
	}
	if len(report.Hreflang) != 2 {
		t.Fatalf("hreflang=%+v", report.Hreflang)
	}
	if report.Status != CheckPass {
		t.Fatalf("status=%s checks=%+v", report.Status, checks)
	}
	if checkLabels(checks)["HTML lang"] != CheckPass || checkLabels(checks)["Hreflang"] != CheckPass {
		t.Fatalf("labels=%v", checkLabels(checks))
	}
}

func TestInspectInternational_MissingLangWarns(t *testing.T) {
	doc := mustHTML(t, `<html><head></head><body></body></html>`)
	page, _ := url.Parse("https://example.com/")
	report, checks := InspectInternational(page, doc, time.Now())
	if report.HTMLLang != "" {
		t.Fatalf("lang=%s", report.HTMLLang)
	}
	if checkLabels(checks)["HTML lang"] != CheckWarning {
		t.Fatalf("checks=%+v", checks)
	}
	if !strings.Contains(labelsDetail(checks, "HTML lang"), "Homepage only") {
		t.Fatalf("expected homepage-only: %+v", checks)
	}
}

func TestLocalePathSegment(t *testing.T) {
	u, _ := url.Parse("https://example.com/en-us/about")
	if got := localePathSegment(u); got != "en-us" {
		t.Fatalf("got %s", got)
	}
	root, _ := url.Parse("https://example.com/")
	if got := localePathSegment(root); got != "" {
		t.Fatalf("root=%s", got)
	}
}
