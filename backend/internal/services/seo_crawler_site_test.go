package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCrawlSiteFollowsSameHostLinksUpToCap(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Home page title</title>
<meta name="description" content="A long enough description for the homepage check to pass nicely.">
<meta name="viewport" content="width=device-width">
</head><body><h1>Home</h1><a href="/about">About</a><a href="/extra">Extra</a></body></html>`))
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>About page title</title>
<meta name="description" content="About description that is also long enough to pass.">
<meta name="viewport" content="width=device-width">
</head><body><h1>About</h1></body></html>`))
	})
	mux.HandleFunc("/extra", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Extra page title</title></head><body><h1>Extra</h1></body></html>`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	crawler := NewSEOCrawlerService("")
	res, err := crawler.CrawlSite(srv.URL+"/", SiteCrawlOptions{
		PageLimit: 2,
		RunCap:    2,
		Source:    "website",
		Scope:     "domain",
		UserAgent: "desktop",
		Deadline:  time.Now().Add(10 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pages) != 2 {
		t.Fatalf("pages=%d want 2 (cap), urls=%v", len(res.Pages), pageURLs(res))
	}
	if !strings.Contains(res.StoppedReason, "cap") && res.StoppedReason != "page cap" {
		t.Fatalf("stopped=%q want page cap", res.StoppedReason)
	}
}

func pageURLs(res *SiteCrawlResult) []string {
	out := make([]string, 0, len(res.Pages))
	for _, p := range res.Pages {
		out = append(out, p.URL)
	}
	return out
}
