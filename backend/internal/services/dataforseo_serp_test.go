package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrganicSERPUnknownEngine(t *testing.T) {
	svc := NewDataForSEOService("l", "p")
	_, err := svc.OrganicSERP(context.Background(), "baidu", "desktop", 2840, "en", []string{"coffee"}, []string{"example.com"})
	if err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("err=%v", err)
	}
}

func TestOrganicSERPParsesTargetAndCannibal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v3/serp/google/organic/live/regular") {
			t.Errorf("path=%s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "coffee") {
			t.Errorf("body=%s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":20000,"status_message":"Ok.","tasks":[{"status_code":20000,"status_message":"Ok.","result":[{"item_types":["organic","featured_snippet"],"items":[
			{"type":"featured_snippet","rank_group":1,"rank_absolute":1,"domain":"example.com","url":"https://example.com/a","title":"A"},
			{"type":"organic","rank_group":1,"rank_absolute":2,"domain":"example.com","url":"https://example.com/a","title":"A"},
			{"type":"organic","rank_group":2,"rank_absolute":3,"domain":"example.com","url":"https://example.com/b","title":"B"},
			{"type":"organic","rank_group":3,"rank_absolute":4,"domain":"rival.com","url":"https://rival.com/c","title":"C"}
		]}]}]}`))
	}))
	defer srv.Close()
	raw := NewDataForSEOService("l", "p")
	s := raw.(*dataForSEO)
	s.baseURL = srv.URL
	s.client = srv.Client()

	got, err := s.OrganicSERP(context.Background(), "google", "desktop", 2840, "en", []string{"coffee"}, []string{"example.com", "rival.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].OK {
		t.Fatalf("got %+v", got)
	}
	if got[0].Target == nil || got[0].Target.Position != 1 {
		t.Fatalf("target %+v", got[0].Target)
	}
	if len(got[0].Cannibalizing) != 2 {
		t.Fatalf("cannibal %+v", got[0].Cannibalizing)
	}
	if len(got[0].Competitors) != 1 || got[0].Competitors[0].Domain != "rival.com" {
		t.Fatalf("competitors %+v", got[0].Competitors)
	}
}
