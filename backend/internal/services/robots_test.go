package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRobotsTxt_SitemapAndStarDisallow(t *testing.T) {
	p := parseRobotsTxt(`User-agent: *
Disallow: /admin
Sitemap: https://example.com/sitemap.xml
`)
	if len(p.sitemaps) != 1 || p.sitemaps[0] != "https://example.com/sitemap.xml" {
		t.Fatalf("sitemaps=%v", p.sitemaps)
	}
	dis, allow := selectRobotsRules(p.groups)
	if pathBlockedByRobots(dis, allow, "/") {
		t.Fatal("homepage should not be blocked")
	}
	if !pathBlockedByRobots(dis, allow, "/admin/users") {
		t.Fatal("/admin/users should be blocked")
	}
}

func TestParseRobotsTxt_BlocksEntireSite(t *testing.T) {
	p := parseRobotsTxt("User-agent: *\nDisallow: /\n")
	dis, allow := selectRobotsRules(p.groups)
	if !pathBlockedByRobots(dis, allow, "/") {
		t.Fatal("expected BlocksAll")
	}
	got := blockedImportantPaths(dis, allow)
	if len(got) != 1 || !strings.Contains(got[0], "entire site") {
		t.Fatalf("blocked=%v", got)
	}
}

func TestParseRobotsTxt_GooglebotOverridesStar(t *testing.T) {
	p := parseRobotsTxt(`User-agent: *
Disallow: /

User-agent: Googlebot
Disallow: /nogoogle
`)
	dis, allow := selectRobotsRules(p.groups)
	if pathBlockedByRobots(dis, allow, "/") {
		t.Fatal("Googlebot group should not inherit Disallow: / from *")
	}
	if !pathBlockedByRobots(dis, allow, "/nogoogle") {
		t.Fatal("/nogoogle should be blocked for Googlebot")
	}
}

func TestParseRobotsTxt_AllowOverridesShorterDisallow(t *testing.T) {
	p := parseRobotsTxt(`User-agent: *
Disallow: /public
Allow: /public/ok
`)
	dis, allow := selectRobotsRules(p.groups)
	if pathBlockedByRobots(dis, allow, "/public/ok") {
		t.Fatal("longer Allow should win")
	}
	if !pathBlockedByRobots(dis, allow, "/public/no") {
		t.Fatal("/public/no should stay blocked")
	}
}

func TestParseRobotsTxt_HTMLIsDetected(t *testing.T) {
	p := parseRobotsTxt("<!doctype html><html><body>404</body></html>")
	if !p.looksLikeHTML {
		t.Fatal("expected HTML detection")
	}
}

func TestInspectRobots_200WithSitemap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/robots.txt" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("User-agent: *\nDisallow:\nSitemap: https://ex.com/sitemap.xml\n"))
	}))
	t.Cleanup(srv.Close)

	checked := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	report, checks := InspectRobots(srv.Client(), srv.URL+"/robots.txt", checked)
	if !report.Available || !report.Accessible || !report.SitemapDeclared {
		t.Fatalf("report=%+v", report)
	}
	if report.Status != CheckPass && report.Status != CheckWarning {
		t.Fatalf("status=%s checks=%+v", report.Status, checks)
	}
	if !strings.Contains(checks[0].Detail, "2026-10-05 12:00 UTC") {
		t.Fatalf("last checked missing: %s", checks[0].Detail)
	}
	labels := map[string]string{}
	for _, c := range checks {
		labels[c.Label] = c.Status
	}
	if labels["robots.txt availability"] != CheckPass {
		t.Fatalf("availability=%s", labels["robots.txt availability"])
	}
	if labels["Sitemap declaration"] != CheckPass {
		t.Fatalf("sitemap=%s", labels["Sitemap declaration"])
	}
}

func TestInspectRobots_404(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	report, checks := InspectRobots(srv.Client(), srv.URL+"/robots.txt", time.Now())
	if report.Available {
		t.Fatal("404 should not be available")
	}
	if report.Status != CheckWarning {
		t.Fatalf("status=%s want warning", report.Status)
	}
	if len(checks) < 2 {
		t.Fatalf("checks=%d", len(checks))
	}
}

func TestInspectRobots_BlocksAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
	}))
	t.Cleanup(srv.Close)
	report, checks := InspectRobots(srv.Client(), srv.URL+"/robots.txt", time.Now())
	if !report.BlocksAll {
		t.Fatal("expected blocks_all")
	}
	found := false
	for _, c := range checks {
		if c.Label == "Important URLs blocked" && c.Status == CheckFail {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing fail for site block: %+v", checks)
	}
}
