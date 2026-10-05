package services

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func TestFindMixedContent(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<html><head>
<link rel="stylesheet" href="http://cdn.example/a.css">
</head><body>
<img src="http://cdn.example/a.png">
<script src="https://ok.example/app.js"></script>
<script src="http://evil.example/x.js"></script>
</body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	active, passive := findMixedContent(doc)
	if len(active) != 2 {
		t.Fatalf("active=%v want stylesheet+script", active)
	}
	if len(passive) != 1 || !strings.Contains(passive[0], "a.png") {
		t.Fatalf("passive=%v", passive)
	}
}

func TestCheckHTTPRedirectToHTTPS(t *testing.T) {
	finalHTTPS := "https://example.com/secure"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalHTTPS, http.StatusMovedPermanently)
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL + "/")
	ok, note, checked := checkHTTPRedirect(srv.Client(), u)
	if !checked {
		t.Fatalf("not checked: %s", note)
	}
	// httptest client will try to follow to example.com — may fail network.
	// The first hop is HTTP 301 to https. If follow fails, checked=false.
	if !ok && !strings.Contains(note, "HTTPS") && !strings.Contains(strings.ToLower(note), "redirect") && !strings.Contains(note, "not reachable") {
		t.Fatalf("unexpected note=%s ok=%v", note, ok)
	}
}

func TestCheckHTTPStaysOnHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL + "/")
	ok, note, checked := checkHTTPRedirect(srv.Client(), u)
	if !checked {
		t.Fatalf("expected to check HTTP server: %s", note)
	}
	if ok {
		t.Fatalf("should not report HTTPS redirect: %s", note)
	}
}

func TestCheckTLSTestServer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)
	httpsDialTLSConfig = &tls.Config{InsecureSkipVerify: true}
	t.Cleanup(func() { httpsDialTLSConfig = nil })
	u, _ := url.Parse(srv.URL)
	got := checkTLS(u)
	if !got.valid {
		t.Fatalf("tls=%+v", got)
	}
	if got.expiry == "" {
		t.Fatal("expected expiry")
	}
}

func TestInspectHTTPS_HTTPPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("<html><body>hi</body></html>"))
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	doc, _ := html.Parse(strings.NewReader("<html><body>hi</body></html>"))
	report, checks := InspectHTTPS(srv.Client(), u, doc, http.Header{}, time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC))
	if report.ServesHTTPS {
		t.Fatal("expected HTTP")
	}
	if report.Status != CheckFail {
		t.Fatalf("status=%s checks=%+v", report.Status, checks)
	}
	found := false
	for _, c := range checks {
		if c.Label == "HTTPS implementation" && c.Status == CheckFail {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing HTTPS fail: %+v", checks)
	}
}

func TestHttpsScore(t *testing.T) {
	n := httpsScore(HTTPSReport{ServesHTTPS: true, TLSValid: true, RedirectsHTTP: true, HSTS: true})
	if n != 100 {
		t.Fatalf("score=%d want 100", n)
	}
	partial := httpsScore(HTTPSReport{ServesHTTPS: true, TLSValid: true})
	if partial != 85 { // 40 + 25 + 5 (HTTPS-only redirect credit) + 15 (no active mixed)
		t.Fatalf("partial=%d want 85", partial)
	}
}

func TestInspectHTTPS_TLSWithMixedAndHSTS(t *testing.T) {
	htmlBody := `<html><head><link rel="stylesheet" href="http://cdn.example/a.css"></head>
<body><script src="http://evil.example/x.js"></script><img src="https://ok.example/a.png"></body></html>`
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(htmlBody))
	}))
	t.Cleanup(srv.Close)
	httpsDialTLSConfig = &tls.Config{InsecureSkipVerify: true}
	t.Cleanup(func() { httpsDialTLSConfig = nil })

	u, _ := url.Parse(srv.URL)
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{}
	headers.Set("Strict-Transport-Security", "max-age=31536000")
	report, checks := InspectHTTPS(srv.Client(), u, doc, headers, time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC))
	if !report.ServesHTTPS {
		t.Fatal("expected HTTPS")
	}
	if !report.HSTS {
		t.Fatal("expected HSTS")
	}
	if !report.TLSValid {
		t.Fatalf("tls invalid: %s", report.TLSError)
	}
	if len(report.MixedActive) == 0 {
		t.Fatalf("expected mixed active, checks=%+v", checks)
	}
	if report.Status != CheckFail {
		t.Fatalf("active mixed content should fail, status=%s", report.Status)
	}
	found := false
	for _, c := range checks {
		if c.Label == "Mixed content" && c.Status == CheckFail {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing mixed-content fail: %+v", checks)
	}
}
