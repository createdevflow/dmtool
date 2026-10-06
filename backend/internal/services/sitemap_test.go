package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseSitemapXML_URLSet(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/</loc></url>
  <url><loc>https://example.com/a</loc></url>
  <url><loc>  https://example.com/b  </loc></url>
</urlset>`)
	p := parseSitemapXML(body)
	if p.kind != sitemapKindURLSet || p.invalid || p.html {
		t.Fatalf("parsed=%+v", p)
	}
	if len(p.locs) != 3 || p.locs[2] != "https://example.com/b" {
		t.Fatalf("locs=%v", p.locs)
	}
}

func TestParseSitemapXML_Index(t *testing.T) {
	body := []byte(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://example.com/s1.xml</loc></sitemap>
  <sitemap><loc>https://example.com/s2.xml</loc></sitemap>
</sitemapindex>`)
	p := parseSitemapXML(body)
	if p.kind != sitemapKindIndex || len(p.locs) != 2 {
		t.Fatalf("parsed=%+v", p)
	}
}

func TestParseSitemapXML_HTMLAndEmpty(t *testing.T) {
	if p := parseSitemapXML([]byte("<!doctype html><html><body>nope</body></html>")); !p.html {
		t.Fatal("expected html")
	}
	if p := parseSitemapXML([]byte("   ")); !p.empty {
		t.Fatal("expected empty")
	}
	p := parseSitemapXML([]byte("<urlset xmlns='http://www.sitemaps.org/schemas/sitemap/0.9'></urlset>"))
	if p.kind != sitemapKindURLSet || !p.empty || len(p.locs) != 0 {
		t.Fatalf("empty urlset=%+v", p)
	}
	if p := parseSitemapXML([]byte("<not-a-sitemap>x</not-a-sitemap>")); !p.invalid {
		t.Fatalf("unknown root should be invalid, got %+v", p)
	}
}

func TestInspectSitemap_FallbackURLSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/pricing</loc></url>
  <url><loc>https://example.com/blog</loc></url>
</urlset>`))
	}))
	t.Cleanup(srv.Close)

	origin, _ := url.Parse(srv.URL)
	checked := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	report, checks := InspectSitemap(srv.Client(), nil, origin, checked)
	if report.DeclaredFromRobots {
		t.Fatal("fallback should not be robots-declared")
	}
	if report.URLCount != 2 {
		t.Fatalf("url_count=%d want 2 report=%+v", report.URLCount, report)
	}
	if report.Status != CheckPass {
		t.Fatalf("status=%s checks=%+v", report.Status, checks)
	}
	labels := checkLabels(checks)
	if labels["Sitemap fetch"] != CheckPass || labels["Sitemap XML"] != CheckPass || labels["Sitemap URLs"] != CheckPass {
		t.Fatalf("labels=%v", labels)
	}
	if !strings.Contains(labelsDetail(checks, "Sitemap URLs"), "not a sitewide crawl") {
		t.Fatalf("honesty missing: %+v", checks)
	}
}

func TestInspectSitemap_DeclaredOverFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/news.xml":
			_, _ = w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/n1</loc></url>
</urlset>`))
		case "/sitemap.xml":
			t.Errorf("should not fetch fallback when robots declared a sitemap")
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	origin, _ := url.Parse(srv.URL)
	report, _ := InspectSitemap(srv.Client(), []string{srv.URL + "/news.xml"}, origin, time.Now())
	if !report.DeclaredFromRobots || report.URLCount != 1 {
		t.Fatalf("report=%+v", report)
	}
}

func TestInspectSitemap_IndexFollowsChildren(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap.xml":
			_, _ = w.Write([]byte(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>/a.xml</loc></sitemap>
  <sitemap><loc>/b.xml</loc></sitemap>
</sitemapindex>`))
		case "/a.xml":
			_, _ = w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/a1</loc></url>
  <url><loc>https://example.com/a2</loc></url>
</urlset>`))
		case "/b.xml":
			_, _ = w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/b1</loc></url>
</urlset>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	origin, _ := url.Parse(srv.URL)
	report, checks := InspectSitemap(srv.Client(), nil, origin, time.Now())
	if report.URLCount != 3 {
		t.Fatalf("url_count=%d want 3 files=%+v", report.URLCount, report.Files)
	}
	if report.IndexChildren != 2 {
		t.Fatalf("index_children=%d", report.IndexChildren)
	}
	if report.Status != CheckPass {
		t.Fatalf("status=%s checks=%+v", report.Status, checks)
	}
}

func TestInspectSitemap_404FallbackIsWarning(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	origin, _ := url.Parse(srv.URL)
	report, checks := InspectSitemap(srv.Client(), nil, origin, time.Now())
	if report.URLCount != 0 {
		t.Fatalf("url_count=%d", report.URLCount)
	}
	if report.Status != CheckWarning {
		t.Fatalf("status=%s want warning checks=%+v", report.Status, checks)
	}
	if checkLabels(checks)["Sitemap fetch"] != CheckWarning {
		t.Fatalf("fetch should warn on missing fallback, got %+v", checks)
	}
}

func TestInspectSitemap_Declared404IsFail(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	origin, _ := url.Parse(srv.URL)
	report, checks := InspectSitemap(srv.Client(), []string{srv.URL + "/missing.xml"}, origin, time.Now())
	if !report.DeclaredFromRobots {
		t.Fatal("expected declared")
	}
	if report.Status != CheckFail {
		t.Fatalf("status=%s want fail checks=%+v", report.Status, checks)
	}
}

func TestInspectSitemap_HTMLIsFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><html><body>homepage</body></html>"))
	}))
	t.Cleanup(srv.Close)
	origin, _ := url.Parse(srv.URL)
	report, checks := InspectSitemap(srv.Client(), nil, origin, time.Now())
	if report.Status != CheckFail {
		t.Fatalf("status=%s checks=%+v", report.Status, checks)
	}
	if checkLabels(checks)["Sitemap XML"] != CheckFail {
		t.Fatalf("xml should fail on HTML, %+v", checks)
	}
}

func TestInspectSitemap_EmptyURLSetWarnsURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`))
	}))
	t.Cleanup(srv.Close)
	origin, _ := url.Parse(srv.URL)
	report, checks := InspectSitemap(srv.Client(), nil, origin, time.Now())
	if report.URLCount != 0 {
		t.Fatalf("url_count=%d", report.URLCount)
	}
	labels := checkLabels(checks)
	if labels["Sitemap XML"] != CheckPass {
		t.Fatalf("empty urlset is still valid XML, got %v", labels)
	}
	if labels["Sitemap URLs"] != CheckWarning {
		t.Fatalf("0 loc should warn, got %v", labels)
	}
}

func checkLabels(checks []AuditCheck) map[string]string {
	out := map[string]string{}
	for _, c := range checks {
		out[c.Label] = c.Status
	}
	return out
}

func labelsDetail(checks []AuditCheck, label string) string {
	for _, c := range checks {
		if c.Label == label {
			return c.Detail
		}
	}
	return ""
}
