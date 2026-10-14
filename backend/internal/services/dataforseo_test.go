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

func TestKeywordOverviewParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "keyword_overview") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"keyword":"Buy Widgets","keyword_info":{"search_volume":2200,"cpc":1.5,"competition":0.42,"competition_level":"MEDIUM","monthly_searches":[{"year":2026,"month":9,"search_volume":2100},{"year":2026,"month":8,"search_volume":1900}],"search_volume_trend":{"monthly":12}},"keyword_properties":{"keyword_difficulty":44},"search_intent_info":{"main_intent":"commercial"},"serp_info":{"serp_item_types":["organic","featured_snippet","people_also_ask"]}},{"keyword":"cheap widgets","keyword_info":{"search_volume":80,"cpc":0.2,"keyword_difficulty":12}}]}]}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.KeywordOverview(context.Background(), []string{"Buy Widgets", "cheap widgets", "Buy Widgets"})
	if err != nil {
		t.Fatal(err)
	}
	buy := got["buy widgets"]
	if buy.SearchVolume != 2200 || buy.CPC != 1.5 || buy.KD != 44 {
		t.Fatalf("buy=%+v", buy)
	}
	if buy.CompetitionLevel != "MEDIUM" || buy.LabsIntent != "commercial" {
		t.Fatalf("extras=%+v", buy)
	}
	if buy.TrendMonthly == nil || *buy.TrendMonthly != 12 {
		t.Fatalf("trend=%v", buy.TrendMonthly)
	}
	if len(buy.SerpFeatures) != 3 || buy.SerpFeatures[1] != "featured_snippet" {
		t.Fatalf("serp=%v", buy.SerpFeatures)
	}
	if len(buy.MonthlySearches) != 2 || buy.MonthlySearches[0].SearchVolume != 2100 {
		t.Fatalf("monthly=%v", buy.MonthlySearches)
	}
	cheap := got["cheap widgets"]
	if cheap.SearchVolume != 80 || cheap.KD != 12 {
		t.Fatalf("cheap=%+v", cheap)
	}
}

func TestRelatedKeywordsUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.RelatedKeywords(context.Background(), "widgets")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestRelatedKeywordsParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "related_keywords") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[
			{"keyword_data":{"keyword":"widgets","keyword_info":{"search_volume":5000},"keyword_properties":{"keyword_difficulty":40}}},
			{"keyword_data":{"keyword":"best widgets","keyword_info":{"search_volume":880,"cpc":1.1,"competition_level":"LOW"},"keyword_properties":{"keyword_difficulty":21}}}
		]}]}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.RelatedKeywords(context.Background(), "Widgets")
	if err != nil {
		t.Fatal(err)
	}
	if !got.RelatedOK || got.Seed != "widgets" {
		t.Fatalf("got=%+v", got)
	}
	if len(got.Related) != 1 || got.Related[0].Keyword != "best widgets" || got.Related[0].SearchVolume != 880 || got.Related[0].KD != 21 {
		t.Fatalf("related=%+v (seed itself must be skipped)", got.Related)
	}
}

func TestRelatedKeywordsErrorDoesNotInvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.RelatedKeywords(context.Background(), "widgets")
	if err != nil {
		t.Fatal(err)
	}
	if got.RelatedOK || len(got.Related) != 0 {
		t.Fatalf("must not invent related keywords, got %+v", got.Related)
	}
	if got.RelatedError == "" {
		t.Fatal("expected related_error")
	}
}

func TestKeywordOverviewUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.KeywordOverview(context.Background(), []string{"widgets"})
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v", err)
	}
}

func TestKeywordOverviewOmitsMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"keyword":"known","keyword_info":{"search_volume":9,"cpc":0.4}}]}]}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.KeywordOverview(context.Background(), []string{"known", "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["unknown"]; ok {
		t.Fatal("missing keyword must not be invented")
	}
	if got["known"].SearchVolume != 9 {
		t.Fatalf("known=%+v", got["known"])
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

func TestKeywordGapUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.KeywordGap(context.Background(), "example.com", "rival.com")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestKeywordGapSameDomainRejected(t *testing.T) {
	svc := &dataForSEO{login: "u", password: "p"}
	_, err := svc.KeywordGap(context.Background(), "https://www.example.com", "example.com")
	if err == nil || !strings.Contains(err.Error(), "different domain") {
		t.Fatalf("err=%v", err)
	}
}

func TestKeywordGapParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "domain_intersection") {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"intersections":false`) {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"keyword_data":{"keyword":"buy widgets","keyword_info":{"search_volume":2200,"cpc":1.5},"keyword_properties":{"keyword_difficulty":44}},"first_domain_serp_element":{"rank_group":3},"second_domain_serp_element":{"rank_group":0}}]}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"keyword_data":{"keyword":"cheap widgets","keyword_info":{"search_volume":800,"cpc":0.4},"keyword_properties":{"keyword_difficulty":12}},"first_domain_serp_element":{"rank_group":5},"second_domain_serp_element":{"rank_group":8}}]}]}]}`))
	}))
	t.Cleanup(srv.Close)

	svc := &dataForSEO{login: "user", password: "pass", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.KeywordGap(context.Background(), "https://www.example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != "example.com" || got.Competitor != "rival.com" {
		t.Fatalf("hosts=%s/%s", got.Target, got.Competitor)
	}
	if !got.MissingOK || len(got.Missing) != 1 {
		t.Fatalf("missing=%+v ok=%v err=%s", got.Missing, got.MissingOK, got.MissingError)
	}
	miss := got.Missing[0]
	if miss.Keyword != "buy widgets" || miss.Volume != 2200 || miss.KD != 44 || miss.TargetPosition != 0 || miss.CompetitorPosition != 3 {
		t.Fatalf("missing row=%+v", miss)
	}
	if miss.Intent != "transactional" {
		t.Fatalf("missing intent=%s", miss.Intent)
	}
	if !got.OverlapOK || len(got.Overlap) != 1 {
		t.Fatalf("overlap=%+v ok=%v err=%s", got.Overlap, got.OverlapOK, got.OverlapError)
	}
	over := got.Overlap[0]
	if over.Keyword != "cheap widgets" || over.Volume != 800 || over.TargetPosition != 5 || over.CompetitorPosition != 8 {
		t.Fatalf("overlap row=%+v", over)
	}
}

func TestKeywordGapSectionErrorDoesNotInvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"intersections":true`) {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"keyword_data":{"keyword":"widgets","keyword_info":{"search_volume":10}},"first_domain_serp_element":{"rank_group":2}}]}]}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.KeywordGap(context.Background(), "example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if !got.MissingOK || len(got.Missing) != 1 || got.Missing[0].Keyword != "widgets" {
		t.Fatalf("missing=%+v", got.Missing)
	}
	if got.OverlapOK || len(got.Overlap) != 0 {
		t.Fatalf("overlap should be empty on error, got %+v ok=%v", got.Overlap, got.OverlapOK)
	}
	if got.OverlapError == "" {
		t.Fatal("expected overlap_error")
	}
}

func TestDomainCompareUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.DomainCompare(context.Background(), "example.com", "rival.com")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestDomainCompareSameDomainRejected(t *testing.T) {
	svc := &dataForSEO{login: "u", password: "p"}
	_, err := svc.DomainCompare(context.Background(), "https://www.example.com", "example.com")
	if err == nil || !strings.Contains(err.Error(), "different domain") {
		t.Fatalf("err=%v", err)
	}
}

func TestDomainCompareParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		payload := string(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "domain_rank_overview"):
			if strings.Contains(payload, `"target":"rival.com"`) {
				_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"target":"rival.com","location_code":2840,"metrics":{"organic":{"pos_1":1,"pos_2_3":2,"pos_4_10":3,"etv":800.2,"count":20},"paid":{"etv":4}}}]}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"target":"example.com","location_code":2840,"metrics":{"organic":{"pos_1":2,"pos_2_3":5,"pos_4_10":10,"etv":1234.5,"count":40},"paid":{"etv":10}}}]}]}`))
		case strings.Contains(r.URL.Path, "backlinks/summary"):
			if strings.Contains(payload, `"target":"rival.com"`) {
				_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"backlinks":50,"referring_domains":8,"rank":200,"backlinks_spam_score":3}]}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"backlinks":99,"referring_domains":12,"rank":371,"backlinks_spam_score":8}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc := &dataForSEO{login: "user", password: "pass", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.DomainCompare(context.Background(), "https://www.example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != "example.com" || got.Competitor != "rival.com" {
		t.Fatalf("hosts=%s/%s", got.Target, got.Competitor)
	}
	you := got.TargetSide
	if !you.OrganicOK || you.Organic.Count != 40 || you.Organic.ETV != 1234.5 {
		t.Fatalf("target organic=%+v ok=%v err=%s", you.Organic, you.OrganicOK, you.OrganicError)
	}
	if !you.BacklinksOK || you.Backlinks.Backlinks != 99 {
		t.Fatalf("target backlinks=%+v", you.Backlinks)
	}
	them := got.CompetitorSide
	if !them.OrganicOK || them.Organic.Count != 20 || them.Organic.ETV != 800.2 {
		t.Fatalf("competitor organic=%+v", them.Organic)
	}
	if !them.BacklinksOK || them.Backlinks.ReferringDomains != 8 {
		t.Fatalf("competitor backlinks=%+v", them.Backlinks)
	}
}

func TestDomainCompareSectionErrorDoesNotInvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		payload := string(body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "domain_rank_overview") && strings.Contains(payload, `"target":"rival.com"`) {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
			return
		}
		if strings.Contains(r.URL.Path, "domain_rank_overview") {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"target":"example.com","metrics":{"organic":{"count":9,"etv":11}}}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"backlinks":3,"referring_domains":1,"rank":10}]}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.DomainCompare(context.Background(), "example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if !got.TargetSide.OrganicOK || got.TargetSide.Organic.Count != 9 {
		t.Fatalf("target organic=%+v", got.TargetSide.Organic)
	}
	if got.CompetitorSide.OrganicOK || got.CompetitorSide.Organic.Count != 0 {
		t.Fatalf("competitor organic should be empty on error, got %+v ok=%v", got.CompetitorSide.Organic, got.CompetitorSide.OrganicOK)
	}
	if got.CompetitorSide.OrganicError == "" {
		t.Fatal("expected competitor organic_error")
	}
	if !got.CompetitorSide.BacklinksOK || got.CompetitorSide.Backlinks.Backlinks != 3 {
		t.Fatalf("competitor backlinks should still succeed, got %+v", got.CompetitorSide.Backlinks)
	}
}

func TestContentGapUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.ContentGap(context.Background(), "example.com", "rival.com")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestContentGapSameDomainRejected(t *testing.T) {
	svc := &dataForSEO{login: "u", password: "p"}
	_, err := svc.ContentGap(context.Background(), "https://www.example.com", "example.com")
	if err == nil || !strings.Contains(err.Error(), "different domain") {
		t.Fatalf("err=%v", err)
	}
}

func TestContentGapParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "page_intersection"):
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"keyword_data":{"keyword":"buy widgets","keyword_info":{"search_volume":2200,"cpc":1.5},"keyword_properties":{"keyword_difficulty":44}},"intersection_result":{"1":{"type":"organic","rank_group":3,"url":"https://rival.com/widgets"}}}]}]}]}`))
		case strings.Contains(r.URL.Path, "relevant_pages"):
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"page_address":"https://rival.com/widgets","metrics":{"organic":{"pos_1":2,"pos_2_3":4,"pos_4_10":6,"etv":321.5,"count":18}}}]}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc := &dataForSEO{login: "user", password: "pass", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.ContentGap(context.Background(), "https://www.example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != "example.com" || got.Competitor != "rival.com" {
		t.Fatalf("hosts=%s/%s", got.Target, got.Competitor)
	}
	if !got.MissingOK || len(got.Missing) != 1 {
		t.Fatalf("missing=%+v ok=%v err=%s", got.Missing, got.MissingOK, got.MissingError)
	}
	miss := got.Missing[0]
	if miss.Keyword != "buy widgets" || miss.Volume != 2200 || miss.KD != 44 || miss.CompetitorPosition != 3 || miss.CompetitorURL != "https://rival.com/widgets" {
		t.Fatalf("missing row=%+v", miss)
	}
	if miss.Intent != "transactional" {
		t.Fatalf("intent=%s", miss.Intent)
	}
	if !got.PagesOK || len(got.Pages) != 1 || got.Pages[0].Keywords != 18 || got.Pages[0].ETV != 321.5 {
		t.Fatalf("pages=%+v ok=%v err=%s", got.Pages, got.PagesOK, got.PagesError)
	}
}

func TestContentGapSectionErrorDoesNotInvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "relevant_pages") {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[{"keyword_data":{"keyword":"widgets","keyword_info":{"search_volume":10}},"intersection_result":{"1":{"rank_group":2,"url":"https://rival.com/w"}}}]}]}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.ContentGap(context.Background(), "example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if !got.MissingOK || len(got.Missing) != 1 || got.Missing[0].Keyword != "widgets" {
		t.Fatalf("missing=%+v", got.Missing)
	}
	if got.PagesOK || len(got.Pages) != 0 {
		t.Fatalf("pages should be empty on error, got %+v ok=%v", got.Pages, got.PagesOK)
	}
	if got.PagesError == "" {
		t.Fatal("expected pages_error")
	}
}

func TestBacklinkGapUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.BacklinkGap(context.Background(), "example.com", "rival.com")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestBacklinkGapSameDomainRejected(t *testing.T) {
	svc := &dataForSEO{login: "u", password: "p"}
	_, err := svc.BacklinkGap(context.Background(), "https://www.example.com", "example.com")
	if err == nil || !strings.Contains(err.Error(), "different domain") {
		t.Fatalf("err=%v", err)
	}
}

func TestBacklinkGapParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "backlinks/domain_intersection") {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		payload := string(body)
		if !strings.Contains(payload, `"1":"rival.com"`) && !strings.Contains(payload, `"1": "rival.com"`) {
			http.Error(w, "expected competitor as target 1, got "+payload, http.StatusBadRequest)
			return
		}
		if !strings.Contains(payload, `"exclude_targets"`) || !strings.Contains(payload, `"example.com"`) {
			http.Error(w, "expected exclude_targets our host, got "+payload, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"total_count":3,"items_count":3,"items":[
			{"domain_intersection":{"1":{"target":"news.site","rank":120,"backlinks":44,"first_seen":"2022-06-17 19:02:31 +00:00","backlinks_spam_score":5,"referring_pages":12}}},
			{"domain_intersection":{"1":{"target":"www.example.com","rank":9,"backlinks":2,"first_seen":"2021-01-01 00:00:00 +00:00","backlinks_spam_score":0,"referring_pages":1}}},
			{"domain_intersection":{"1":{"target":"rival.com","rank":1,"backlinks":1,"first_seen":"2021-01-01 00:00:00 +00:00","backlinks_spam_score":0,"referring_pages":1}}}
		]}]}]}`))
	}))
	t.Cleanup(srv.Close)

	svc := &dataForSEO{login: "user", password: "pass", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.BacklinkGap(context.Background(), "https://www.example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != "example.com" || got.Competitor != "rival.com" {
		t.Fatalf("hosts=%s/%s", got.Target, got.Competitor)
	}
	if !got.IntersectionOK || got.IntersectionError != "" {
		t.Fatalf("ok=%v err=%s", got.IntersectionOK, got.IntersectionError)
	}
	if got.TotalCount != 3 {
		t.Fatalf("total_count=%d", got.TotalCount)
	}
	if len(got.Referring) != 1 || got.Referring[0].Domain != "news.site" || got.Referring[0].Rank != 120 || got.Referring[0].Backlinks != 44 || got.Referring[0].SpamScore != 5 || got.Referring[0].ReferringPages != 12 {
		t.Fatalf("referring=%+v (host and rival must be skipped)", got.Referring)
	}
}

func TestBacklinkGapSectionErrorDoesNotInvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.BacklinkGap(context.Background(), "example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.IntersectionOK || len(got.Referring) != 0 {
		t.Fatalf("must not invent referring domains on error, got %+v ok=%v", got.Referring, got.IntersectionOK)
	}
	if got.IntersectionError == "" {
		t.Fatal("expected intersection_error")
	}
}

func TestBacklinkExplorerUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.BacklinkExplorer(context.Background(), "example.com")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestBacklinkExplorerInvalidDomain(t *testing.T) {
	svc := &dataForSEO{login: "u", password: "p"}
	_, err := svc.BacklinkExplorer(context.Background(), "not a domain")
	if err == nil || !strings.Contains(err.Error(), "invalid domain") {
		t.Fatalf("err=%v", err)
	}
}

func TestBacklinkExplorerParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "backlinks/summary"):
			body, _ := io.ReadAll(r.Body)
			payload := string(body)
			if !strings.Contains(payload, `"target":"example.com"`) && !strings.Contains(payload, `"target": "example.com"`) {
				http.Error(w, "expected target example.com, got "+payload, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"backlinks":41245,"referring_domains":12372,"referring_main_domains":11438,"rank":371,"backlinks_spam_score":8,"broken_backlinks":209,"broken_pages":265,"referring_domains_nofollow":40,"referring_pages":38786,"referring_pages_nofollow":12,"referring_ips":10401,"referring_subnets":6427,"first_seen":"2020-01-18 11:50:58 +00:00","referring_links_types":{"anchor":36666,"image":210},"referring_links_attributes":{"nofollow":900,"ugc":20},"info":{"target_spam_score":3}}]}]}`))
		case strings.Contains(r.URL.Path, "backlinks/referring_domains"):
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"total_count":3,"items_count":3,"items":[
				{"domain":"news.site","rank":120,"backlinks":44,"first_seen":"2022-06-17 19:02:31 +00:00","backlinks_spam_score":5,"referring_pages":12,"broken_backlinks":1,"referring_pages_nofollow":2},
				{"domain":"www.example.com","rank":9,"backlinks":2,"first_seen":"2021-01-01 00:00:00 +00:00","backlinks_spam_score":0,"referring_pages":1},
				{"domain":"press.site","rank":90,"backlinks":4,"first_seen":"2021-02-02 00:00:00 +00:00","backlinks_spam_score":0,"referring_pages":3}
			]}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc := &dataForSEO{login: "user", password: "pass", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.BacklinkExplorer(context.Background(), "https://www.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Domain != "example.com" {
		t.Fatalf("domain=%s", got.Domain)
	}
	if !got.SummaryOK || got.SummaryError != "" {
		t.Fatalf("summary ok=%v err=%s", got.SummaryOK, got.SummaryError)
	}
	if got.Audit.Backlinks != 41245 || got.Audit.ReferringDomains != 12372 || got.Audit.Rank != 371 || got.Audit.BrokenBacklinks != 209 {
		t.Fatalf("audit=%+v", got.Audit)
	}
	if got.Audit.LinkTypes["anchor"] != 36666 || got.Audit.LinkAttributes["nofollow"] != 900 {
		t.Fatalf("maps types=%v attrs=%v", got.Audit.LinkTypes, got.Audit.LinkAttributes)
	}
	if !got.ReferringOK || got.TotalCount != 3 {
		t.Fatalf("referring ok=%v total=%d err=%s", got.ReferringOK, got.TotalCount, got.ReferringError)
	}
	if len(got.Referring) != 2 || got.Referring[0].Domain != "news.site" || got.Referring[0].Rank != 120 || got.Referring[1].Domain != "press.site" {
		t.Fatalf("referring=%+v (target host must be skipped)", got.Referring)
	}
}

func TestBacklinkExplorerSectionErrorDoesNotInvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.BacklinkExplorer(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.SummaryOK || got.ReferringOK {
		t.Fatalf("must not mark sections ok on error, summary=%v referring=%v", got.SummaryOK, got.ReferringOK)
	}
	if got.Audit.Backlinks != 0 || got.Audit.ReferringDomains != 0 || got.Audit.Rank != 0 {
		t.Fatalf("must not invent audit numbers, got %+v", got.Audit)
	}
	if len(got.Referring) != 0 {
		t.Fatalf("must not invent referring domains, got %+v", got.Referring)
	}
	if got.SummaryError == "" || got.ReferringError == "" {
		t.Fatal("expected section errors")
	}
}

func TestCitationGapUnconfigured(t *testing.T) {
	svc := NewDataForSEOService("", "")
	_, err := svc.CitationGap(context.Background(), "example.com", "rival.com")
	if err != ErrDataForSEONotConfigured {
		t.Fatalf("err=%v want ErrDataForSEONotConfigured", err)
	}
}

func TestCitationGapSameDomainRejected(t *testing.T) {
	svc := &dataForSEO{login: "u", password: "p"}
	_, err := svc.CitationGap(context.Background(), "https://www.example.com", "example.com")
	if err == nil || !strings.Contains(err.Error(), "different domain") {
		t.Fatalf("err=%v", err)
	}
}

func TestCitationGapParsesMockAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "llm_mentions/target_metrics") {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"domain":"rival.com"`) {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"aggregated_metrics":{"platform":[{"key":"google","mentions":20,"ai_search_volume":700},{"key":"chat_gpt","mentions":5,"ai_search_volume":80}],"sources_domain":[{"key":"rival.com","mentions":20,"ai_search_volume":700},{"key":"other.com","mentions":7,"ai_search_volume":120},{"key":"extra.com","mentions":4,"ai_search_volume":50}],"total":{"mentions":25,"ai_search_volume":780}}}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"aggregated_metrics":{"platform":[{"key":"google","mentions":12,"ai_search_volume":400},{"key":"chat_gpt","mentions":3,"ai_search_volume":90}],"sources_domain":[{"key":"example.com","mentions":12,"ai_search_volume":400},{"key":"other.com","mentions":7,"ai_search_volume":120}],"total":{"mentions":15,"ai_search_volume":490}}}]}]}`))
	}))
	t.Cleanup(srv.Close)

	svc := &dataForSEO{login: "user", password: "pass", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.CitationGap(context.Background(), "https://www.example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != "example.com" || got.Competitor != "rival.com" {
		t.Fatalf("hosts=%s/%s", got.Target, got.Competitor)
	}
	if !got.TargetOK || got.TargetCitations.Mentions != 15 || got.TargetCitations.Google.Mentions != 12 {
		t.Fatalf("target=%+v ok=%v err=%s", got.TargetCitations, got.TargetOK, got.TargetError)
	}
	if !got.CompetitorOK || got.CompetitorCitations.Mentions != 25 {
		t.Fatalf("competitor=%+v", got.CompetitorCitations)
	}
	if len(got.UniqueSources) != 1 || got.UniqueSources[0].Domain != "extra.com" {
		t.Fatalf("unique_sources=%+v (shared other.com and both hosts must be skipped)", got.UniqueSources)
	}
}

func TestCitationGapSectionErrorDoesNotInvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), `"domain":"rival.com"`) {
			_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40100,"status_message":"Auth failed."}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"aggregated_metrics":{"total":{"mentions":4,"ai_search_volume":10},"platform":[{"key":"google","mentions":4,"ai_search_volume":10}]}}]}]}`))
	}))
	t.Cleanup(srv.Close)
	svc := &dataForSEO{login: "u", password: "p", client: srv.Client(), baseURL: srv.URL}
	got, err := svc.CitationGap(context.Background(), "example.com", "rival.com")
	if err != nil {
		t.Fatal(err)
	}
	if !got.TargetOK || got.TargetCitations.Mentions != 4 {
		t.Fatalf("target=%+v", got.TargetCitations)
	}
	if got.CompetitorOK || got.CompetitorCitations.Mentions != 0 {
		t.Fatalf("competitor should be empty on error, got %+v ok=%v", got.CompetitorCitations, got.CompetitorOK)
	}
	if got.CompetitorError == "" {
		t.Fatal("expected competitor_error")
	}
	if len(got.UniqueSources) != 0 {
		t.Fatalf("unique sources must not be invented when one side fails, got %+v", got.UniqueSources)
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
