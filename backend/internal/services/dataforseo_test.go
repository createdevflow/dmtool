package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	cases := map[string]string{
		"https://www.Example.com/path": "example.com",
		"example.com":                  "example.com",
		"http://sub.example.co.uk":     "sub.example.co.uk",
		"":                             "",
	}
	for in, want := range cases {
		if got := NormalizeDomain(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestDomainExplorerUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	if svc.Configured() {
		t.Fatal("empty creds should not be configured")
	}
	_, err := svc.DomainExplorer(context.Background(), "example.com")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestDomainExplorerParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); !ok {
			t.Errorf("missing basic auth on %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var tasks []map[string]any
		_ = json.Unmarshal(body, &tasks)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "domain_rank_overview"):
			_, _ = w.Write([]byte(`{"status_code":20000,"status_message":"Ok.","tasks":[{"status_code":20000,"status_message":"Ok.","result":[{"target":"example.com","location_code":2840,"language_code":"en","metrics":{"organic":{"pos_1":2,"pos_2_3":5,"pos_4_10":10,"etv":1234.5,"count":40},"paid":{"etv":10}}}]}]}`))
		case strings.Contains(r.URL.Path, "competitors_domain"):
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"domain":"example.com","intersections":9,"avg_position":4.2,"full_domain_metrics":{"organic":{"etv":1,"count":1}}},{"domain":"rival.com","intersections":12,"avg_position":6.1,"full_domain_metrics":{"organic":{"etv":800.2,"count":20}}}]}]}]}`))
		case strings.Contains(r.URL.Path, "ranked_keywords"):
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"keyword_data":{"keyword":"buy widgets","keyword_info":{"search_volume":2200,"cpc":1.5},"keyword_properties":{"keyword_difficulty":44}},"ranked_serp_element":{"serp_item":{"rank_group":3,"type":"organic"}}}]}]}]}`))
		case strings.Contains(r.URL.Path, "backlinks/summary"):
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"backlinks":99,"referring_domains":12,"rank":371,"backlinks_spam_score":8}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc := &dataForSEO{
		login:    "user",
		password: "pass",
		client:   srv.Client(),
		baseURL:  srv.URL,
	}
	got, err := svc.DomainExplorer(context.Background(), "https://www.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Domain != "example.com" {
		t.Fatalf("domain=%s", got.Domain)
	}
	if !got.OrganicOK || got.Organic.Count != 40 || got.Organic.ETV != 1234.5 {
		t.Fatalf("organic=%+v ok=%v err=%s", got.Organic, got.OrganicOK, got.OrganicError)
	}
	if !got.BacklinksOK || got.Backlinks.Backlinks != 99 || got.Backlinks.ReferringDomains != 12 {
		t.Fatalf("backlinks=%+v", got.Backlinks)
	}
	if !got.CompetitorsOK || len(got.Competitors) != 1 || got.Competitors[0].Domain != "rival.com" {
		t.Fatalf("competitors=%+v (target domain must be skipped)", got.Competitors)
	}
	if !got.KeywordsOK || len(got.Keywords) != 1 || got.Keywords[0].Volume != 2200 || got.Keywords[0].KD != 44 {
		t.Fatalf("keywords=%+v", got.Keywords)
	}
	if got.Keywords[0].Intent != "transactional" {
		t.Fatalf("intent=%s", got.Keywords[0].Intent)
	}
}

func TestAIVisibilityUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.AIVisibility(context.Background(), "example.com")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestAIVisibilityParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "llm_mentions/target_metrics") {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"sources"`) {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"aggregated_metrics":{"platform":[{"key":"google","mentions":12,"ai_search_volume":400},{"key":"chat_gpt","mentions":3,"ai_search_volume":90}],"sources_domain":[{"key":"example.com","mentions":12,"ai_search_volume":400},{"key":"rival.com","mentions":7,"ai_search_volume":120}],"total":{"mentions":15,"ai_search_volume":490}}}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"aggregated_metrics":{"platform":[{"key":"google","mentions":40,"ai_search_volume":800},{"key":"chat_gpt","mentions":10,"ai_search_volume":200}],"sources_domain":[{"key":"other.com","mentions":5,"ai_search_volume":50}],"total":{"mentions":50,"ai_search_volume":1000}}}]}]}`))
	}))
	t.Cleanup(srv.Close)

	svc := &dataForSEO{
		login:    "user",
		password: "pass",
		client:   srv.Client(),
		baseURL:  srv.URL,
	}
	got, err := svc.AIVisibility(context.Background(), "https://www.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Domain != "example.com" {
		t.Fatalf("domain=%s", got.Domain)
	}
	if !got.MentionsOK || got.Mentions.Mentions != 50 || got.Mentions.Google.Mentions != 40 || got.Mentions.ChatGPT.Mentions != 10 {
		t.Fatalf("mentions=%+v ok=%v err=%s", got.Mentions, got.MentionsOK, got.MentionsError)
	}
	if !got.CitationsOK || got.Citations.Mentions != 15 || got.Citations.Google.Mentions != 12 || got.Citations.ChatGPT.Mentions != 3 {
		t.Fatalf("citations=%+v ok=%v err=%s", got.Citations, got.CitationsOK, got.CitationsError)
	}
	if len(got.Citations.Sources) != 1 || got.Citations.Sources[0].Domain != "rival.com" {
		t.Fatalf("citation sources=%+v (target domain must be skipped)", got.Citations.Sources)
	}
}

func TestAIVisibilitySectionErrorDoesNotInvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"sources"`) {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"aggregated_metrics":{"total":{"mentions":4,"ai_search_volume":10},"platform":[{"key":"google","mentions":4,"ai_search_volume":10}]}}]}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.AIVisibility(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !got.MentionsOK || got.Mentions.Mentions != 4 {
		t.Fatalf("mentions=%+v", got.Mentions)
	}
	if got.CitationsOK || got.Citations.Mentions != 0 {
		t.Fatalf("citations should be empty on error, got %+v ok=%v", got.Citations, got.CitationsOK)
	}
	if got.CitationsError == "" {
		t.Fatal("expected citations_error")
	}
}

func TestPostLiveTaskError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	_, err := svc.postLive(context.Background(), "/v3/x", map[string]any{"target": "example.com"})
	if err == nil || !strings.Contains(err.Error(), "Auth failed") {
		t.Fatalf("err=%v", err)
	}
}
