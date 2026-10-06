package services

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func TestParseIndexDirectives(t *testing.T) {
	d := parseIndexDirectives("noindex, nofollow")
	if !d.noindex || !d.nofollow {
		t.Fatalf("got %+v", d)
	}
	none := parseIndexDirectives("none")
	if !none.noindex || !none.nofollow {
		t.Fatalf("none=%+v", none)
	}
	gb := parseIndexDirectives("googlebot: noindex")
	if !gb.noindex {
		t.Fatal("googlebot noindex should count")
	}
	bing := parseIndexDirectives("bingbot: noindex")
	if bing.noindex {
		t.Fatal("bingbot-only noindex should not count as Google noindex")
	}
	ok := parseIndexDirectives("index, follow")
	if ok.noindex {
		t.Fatal("index,follow should not noindex")
	}
}

func TestCanonicalRelation(t *testing.T) {
	page, _ := url.Parse("https://example.com/")
	if got := canonicalRelation(page, ""); got != "missing" {
		t.Fatalf("empty=%s", got)
	}
	if got := canonicalRelation(page, "https://example.com/"); got != "match" {
		t.Fatalf("self=%s", got)
	}
	if got := canonicalRelation(page, "https://example.com"); got != "match" {
		t.Fatalf("host only=%s", got)
	}
	if got := canonicalRelation(page, "https://example.com/other"); got != "other_path" {
		t.Fatalf("path=%s", got)
	}
	if got := canonicalRelation(page, "https://other.com/"); got != "other_host" {
		t.Fatalf("host=%s", got)
	}
	if got := canonicalRelation(page, "mailto:hi@example.com"); got != "invalid" {
		t.Fatalf("mailto=%s", got)
	}
}

func TestInspectIndexability_IndexableHomepage(t *testing.T) {
	doc := mustHTML(t, `<html><head>
<link rel="canonical" href="https://example.com/">
<meta name="robots" content="index, follow">
</head><body></body></html>`)
	page, _ := url.Parse("https://example.com/")
	report, checks := InspectIndexability(page, doc, http.Header{}, time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC))
	if !report.Indexable || report.Noindex {
		t.Fatalf("report=%+v", report)
	}
	if report.Status != CheckPass {
		t.Fatalf("status=%s checks=%+v", report.Status, checks)
	}
	labels := checkLabels(checks)
	if labels["Meta robots"] != CheckPass || labels["X-Robots-Tag"] != CheckPass || labels["Canonical URL"] != CheckPass {
		t.Fatalf("labels=%v", labels)
	}
	if !strings.Contains(labelsDetail(checks, "Meta robots"), "Homepage only") &&
		!strings.Contains(labelsDetail(checks, "Meta robots"), "this page") {
		t.Fatalf("expected homepage-only wording: %+v", checks)
	}
}

func TestInspectIndexability_MetaNoindex(t *testing.T) {
	doc := mustHTML(t, `<html><head><meta name="robots" content="noindex"></head><body></body></html>`)
	page, _ := url.Parse("https://example.com/")
	report, checks := InspectIndexability(page, doc, nil, time.Now())
	if report.Indexable || !report.Noindex {
		t.Fatalf("report=%+v", report)
	}
	if report.Status != CheckFail {
		t.Fatalf("status=%s", report.Status)
	}
	if checkLabels(checks)["Meta robots"] != CheckFail {
		t.Fatalf("checks=%+v", checks)
	}
}

func TestInspectIndexability_HeaderNoindex(t *testing.T) {
	doc := mustHTML(t, `<html><head></head><body></body></html>`)
	page, _ := url.Parse("https://example.com/")
	h := http.Header{}
	h.Set("X-Robots-Tag", "noindex, nofollow")
	report, checks := InspectIndexability(page, doc, h, time.Now())
	if report.Indexable || !report.Noindex || !report.Nofollow {
		t.Fatalf("report=%+v", report)
	}
	if checkLabels(checks)["X-Robots-Tag"] != CheckFail {
		t.Fatalf("checks=%+v", checks)
	}
}

func TestInspectIndexability_NofollowOnlyWarns(t *testing.T) {
	doc := mustHTML(t, `<html><head><meta name="robots" content="nofollow"></head><body></body></html>`)
	page, _ := url.Parse("https://example.com/")
	report, checks := InspectIndexability(page, doc, nil, time.Now())
	if !report.Indexable || report.Noindex || !report.Nofollow {
		t.Fatalf("report=%+v", report)
	}
	if checkLabels(checks)["Meta robots"] != CheckWarning {
		t.Fatalf("checks=%+v", checks)
	}
}

func TestInspectIndexability_GooglebotMeta(t *testing.T) {
	doc := mustHTML(t, `<html><head><meta name="googlebot" content="noindex"></head><body></body></html>`)
	page, _ := url.Parse("https://example.com/")
	report, _ := InspectIndexability(page, doc, nil, time.Now())
	if !report.Noindex {
		t.Fatalf("googlebot meta should noindex: %+v", report)
	}
}

func TestInspectIndexability_CanonicalMissing(t *testing.T) {
	doc := mustHTML(t, `<html><head></head><body></body></html>`)
	page, _ := url.Parse("https://example.com/")
	report, checks := InspectIndexability(page, doc, nil, time.Now())
	if report.CanonicalRel != "missing" {
		t.Fatalf("rel=%s", report.CanonicalRel)
	}
	if checkLabels(checks)["Canonical URL"] != CheckWarning {
		t.Fatalf("checks=%+v", checks)
	}
}

func mustHTML(t *testing.T, s string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}
