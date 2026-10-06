package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCruxCLS(t *testing.T) {
	if got := cruxCLS(20); got != 0.20 {
		t.Fatalf("cruxCLS(20)=%v want 0.20", got)
	}
	if got := cruxCLS(0.08); got != 0.08 {
		t.Fatalf("cruxCLS(0.08)=%v want 0.08", got)
	}
}

func TestCwvOverallFieldGood(t *testing.T) {
	st, label := cwvOverall(CWVReport{
		FieldAvailable: true,
		LCP:            &CWVMetric{Rating: "good"},
		INP:            &CWVMetric{Rating: "good"},
		CLS:            &CWVMetric{Rating: "good"},
	})
	if st != CheckPass || label != "Good" {
		t.Fatalf("status=%s label=%s", st, label)
	}
}

func TestCwvOverallLabOnly(t *testing.T) {
	n := 92
	st, label := cwvOverall(CWVReport{LabPerformance: &n})
	if st != CheckWarning || label != "Lab only" {
		t.Fatalf("status=%s label=%s", st, label)
	}
	if s := cwvScore(CWVReport{LabPerformance: &n}); s != 0 {
		t.Fatalf("lab must not count as CWV score, got %d", s)
	}
}

func TestInspectCWV_FieldURL(t *testing.T) {
	payload := psiResponse{
		LoadingExperience: psiExperience{
			OverallCategory: "FAST",
			Metrics: map[string]psiMetric{
				"LARGEST_CONTENTFUL_PAINT_MS":   {Percentile: 1800, Category: "FAST"},
				"INTERACTION_TO_NEXT_PAINT":     {Percentile: 120, Category: "FAST"},
				"CUMULATIVE_LAYOUT_SHIFT_SCORE": {Percentile: 5, Category: "FAST"},
			},
		},
		LighthouseResult: psiLighthouse{
			Categories: map[string]struct {
				Score *float64 `json:"score"`
			}{"performance": {Score: floatPtr(0.91)}},
			Audits: map[string]psiAudit{
				"largest-contentful-paint": {NumericValue: floatPtr(1600), DisplayValue: "1.6 s"},
				"cumulative-layout-shift":  {NumericValue: floatPtr(0.02)},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		strategy := r.URL.Query().Get("strategy")
		if strategy != "mobile" && strategy != "desktop" {
			t.Errorf("strategy=%s", strategy)
		}
		if r.URL.Query().Get("url") == "" {
			t.Error("missing url")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(srv.Close)
	prev := pagespeedRunURL
	pagespeedRunURL = srv.URL
	t.Cleanup(func() { pagespeedRunURL = prev })

	report, checks := InspectCWV(srv.Client(), "https://example.com/", "", time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC))
	if !report.FieldAvailable || report.FieldScope != "url" {
		t.Fatalf("field=%v scope=%s", report.FieldAvailable, report.FieldScope)
	}
	if report.OverallLabel != "Good" || report.Status != CheckPass {
		t.Fatalf("label=%s status=%s", report.OverallLabel, report.Status)
	}
	if report.LCP == nil || report.LCP.Display != "1.8s" {
		t.Fatalf("lcp=%+v", report.LCP)
	}
	if report.CLS == nil || report.CLS.Display != "0.05" {
		t.Fatalf("cls=%+v", report.CLS)
	}
	if report.LabPerformance == nil || *report.LabPerformance != 91 {
		t.Fatalf("lab=%v", report.LabPerformance)
	}
	if report.Score != 100 {
		t.Fatalf("score=%d", report.Score)
	}
	foundField, foundLab := false, false
	for _, c := range checks {
		if c.Label == "Core Web Vitals (CrUX)" && c.Status == CheckPass {
			foundField = true
		}
		if c.Label == "Lighthouse performance" && c.Category == lighthouseCategory {
			foundLab = true
			if !strings.Contains(c.Detail, "not Core Web Vitals") {
				t.Fatalf("lab check must say it is not CWV: %s", c.Detail)
			}
		}
	}
	if !foundField || !foundLab {
		t.Fatalf("checks=%+v", checks)
	}
}

func TestInspectCWV_OriginFallbackAndPoorLCP(t *testing.T) {
	payload := psiResponse{
		OriginLoadingExperience: psiExperience{
			Metrics: map[string]psiMetric{
				"LARGEST_CONTENTFUL_PAINT_MS":   {Percentile: 5200, Category: "SLOW"},
				"INTERACTION_TO_NEXT_PAINT":     {Percentile: 180, Category: "FAST"},
				"CUMULATIVE_LAYOUT_SHIFT_SCORE": {Percentile: 8, Category: "FAST"},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(srv.Close)
	prev := pagespeedRunURL
	pagespeedRunURL = srv.URL
	t.Cleanup(func() { pagespeedRunURL = prev })

	report, checks := InspectCWV(srv.Client(), "https://example.com/rare", "", time.Now())
	if report.FieldScope != "origin" || report.OverallLabel != "Poor" {
		t.Fatalf("scope=%s label=%s", report.FieldScope, report.OverallLabel)
	}
	found := false
	for _, c := range checks {
		if c.Label == "Core Web Vitals (CrUX)" && strings.Contains(c.Detail, "origin-level") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected origin-level note: %+v", checks)
	}
}

func TestInspectCWV_NoFieldLabOnly(t *testing.T) {
	payload := psiResponse{
		LighthouseResult: psiLighthouse{
			Categories: map[string]struct {
				Score *float64 `json:"score"`
			}{"performance": {Score: floatPtr(0.42)}},
			Audits: map[string]psiAudit{
				"largest-contentful-paint": {DisplayValue: "4.2 s"},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(srv.Close)
	prev := pagespeedRunURL
	pagespeedRunURL = srv.URL
	t.Cleanup(func() { pagespeedRunURL = prev })

	report, checks := InspectCWV(srv.Client(), "https://low-traffic.example/", "", time.Now())
	if report.FieldAvailable {
		t.Fatal("expected no field data")
	}
	if report.OverallLabel != "Lab only" {
		t.Fatalf("label=%s", report.OverallLabel)
	}
	if report.Score != 0 {
		t.Fatalf("must not treat lab as CWV score: %d", report.Score)
	}
	found := false
	for _, c := range checks {
		if c.Label == "Core Web Vitals (CrUX)" && c.Status == CheckWarning {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected field-data warning: %+v", checks)
	}
}

func TestInspectCWV_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 429, "message": "Quota exceeded"},
		})
	}))
	t.Cleanup(srv.Close)
	prev := pagespeedRunURL
	pagespeedRunURL = srv.URL
	t.Cleanup(func() { pagespeedRunURL = prev })

	report, checks := InspectCWV(srv.Client(), "https://example.com/", "fake-key", time.Now())
	if report.OverallLabel != "Unavailable" {
		t.Fatalf("label=%s err=%s", report.OverallLabel, report.Error)
	}
	if report.Status != CheckWarning {
		t.Fatalf("quota must be warning not fail, status=%s", report.Status)
	}
	if len(checks) == 0 || checks[0].Status != CheckWarning {
		t.Fatalf("checks=%+v", checks)
	}
}

func TestInspectCWV_LabExtrasAndDesktop(t *testing.T) {
	mobile := psiResponse{
		LoadingExperience: psiExperience{
			OverallCategory: "FAST",
			Metrics: map[string]psiMetric{
				"LARGEST_CONTENTFUL_PAINT_MS":   {Percentile: 1800, Category: "FAST"},
				"INTERACTION_TO_NEXT_PAINT":     {Percentile: 120, Category: "FAST"},
				"CUMULATIVE_LAYOUT_SHIFT_SCORE": {Percentile: 5, Category: "FAST"},
			},
		},
		LighthouseResult: psiLighthouse{
			Categories: map[string]struct {
				Score *float64 `json:"score"`
			}{"performance": {Score: floatPtr(0.91)}},
			Audits: map[string]psiAudit{
				"largest-contentful-paint": {NumericValue: floatPtr(1600), DisplayValue: "1.6 s"},
				"first-contentful-paint":   {NumericValue: floatPtr(1100), DisplayValue: "1.1 s"},
				"server-response-time":     {NumericValue: floatPtr(120), DisplayValue: "120 ms"},
				"cumulative-layout-shift":  {NumericValue: floatPtr(0.02)},
				"unused-javascript": {
					ID: "unused-javascript", Title: "Reduce unused JavaScript",
					DisplayValue: "Est savings of 400 ms",
					Details:      &psiAuditDetails{Type: "opportunity", OverallSavingsMs: 400},
				},
				"render-blocking-resources": {
					ID: "render-blocking-resources", Title: "Eliminate render-blocking resources",
					DisplayValue: "Est savings of 800 ms",
					Details:      &psiAuditDetails{Type: "opportunity", OverallSavingsMs: 800},
				},
				"empty-opportunity": {
					ID: "empty-opportunity", Title: "Invented savings",
					Details: &psiAuditDetails{Type: "opportunity", OverallSavingsMs: 0},
				},
				"not-an-opportunity": {
					ID: "unused-css", Title: "Reduce unused CSS",
					Details: &psiAuditDetails{Type: "table"},
				},
			},
		},
	}
	desktop := psiResponse{
		LoadingExperience: psiExperience{
			OverallCategory: "SLOW",
			Metrics: map[string]psiMetric{
				"LARGEST_CONTENTFUL_PAINT_MS":   {Percentile: 9000, Category: "SLOW"},
				"INTERACTION_TO_NEXT_PAINT":     {Percentile: 800, Category: "SLOW"},
				"CUMULATIVE_LAYOUT_SHIFT_SCORE": {Percentile: 40, Category: "SLOW"},
			},
		},
		LighthouseResult: psiLighthouse{
			Categories: map[string]struct {
				Score *float64 `json:"score"`
			}{"performance": {Score: floatPtr(0.55)}},
			Audits: map[string]psiAudit{
				"largest-contentful-paint": {DisplayValue: "3.4 s"},
				"first-contentful-paint":   {DisplayValue: "1.2 s"},
				"time-to-first-byte":       {DisplayValue: "80 ms"},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("strategy") == "desktop" {
			_ = json.NewEncoder(w).Encode(desktop)
			return
		}
		_ = json.NewEncoder(w).Encode(mobile)
	}))
	t.Cleanup(srv.Close)
	prev := pagespeedRunURL
	pagespeedRunURL = srv.URL
	t.Cleanup(func() { pagespeedRunURL = prev })

	report, checks := InspectCWV(srv.Client(), "https://example.com/", "", time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC))
	if report.OverallLabel != "Good" || report.Score != 100 {
		t.Fatalf("desktop CrUX must not mix into CWV: label=%s score=%d lcp=%+v", report.OverallLabel, report.Score, report.LCP)
	}
	if report.LabFCP != "1.1 s" || report.LabTTFB != "120 ms" {
		t.Fatalf("mobile lab extras fcp=%s ttfb=%s", report.LabFCP, report.LabTTFB)
	}
	if len(report.Opportunities) != 2 {
		t.Fatalf("opportunities=%+v (empty/non-opportunity must not be invented)", report.Opportunities)
	}
	if report.Opportunities[0].Title != "Eliminate render-blocking resources" || report.Opportunities[0].SavingsMs != 800 {
		t.Fatalf("opportunities not sorted by savings: %+v", report.Opportunities)
	}
	if report.DesktopPerformance == nil || *report.DesktopPerformance != 55 {
		t.Fatalf("desktop lab=%v", report.DesktopPerformance)
	}
	if report.DesktopFCP != "1.2 s" || report.DesktopTTFB != "80 ms" {
		t.Fatalf("desktop extras fcp=%s ttfb=%s", report.DesktopFCP, report.DesktopTTFB)
	}

	foundFCP, foundTTFB, foundOpp, foundDesktop := false, false, false, false
	for _, c := range checks {
		if c.Category == cwvCategory && (c.Label == "Lab FCP" || c.Label == "Lab TTFB" || c.Label == "Lighthouse opportunities" || c.Label == "Lighthouse desktop") {
			t.Fatalf("lab check leaked into cwv category: %+v", c)
		}
		switch c.Label {
		case "Lab FCP":
			foundFCP = c.Category == lighthouseCategory && strings.Contains(c.Detail, "1.1 s")
		case "Lab TTFB":
			foundTTFB = c.Category == lighthouseCategory && strings.Contains(c.Detail, "120 ms")
		case "Lighthouse opportunities":
			foundOpp = c.Category == lighthouseCategory && c.Status == CheckWarning &&
				strings.Contains(c.Detail, "Eliminate render-blocking resources") &&
				!strings.Contains(c.Detail, "Invented savings")
		case "Lighthouse desktop":
			foundDesktop = c.Category == lighthouseCategory && strings.Contains(c.Detail, "55/100")
		}
	}
	if !foundFCP || !foundTTFB || !foundOpp || !foundDesktop {
		t.Fatalf("missing lab checks fcp=%v ttfb=%v opp=%v desktop=%v checks=%+v", foundFCP, foundTTFB, foundOpp, foundDesktop, checks)
	}
}

func TestInspectCWV_DesktopErrorKeepsMobileField(t *testing.T) {
	mobile := psiResponse{
		LoadingExperience: psiExperience{
			Metrics: map[string]psiMetric{
				"LARGEST_CONTENTFUL_PAINT_MS":   {Percentile: 1800, Category: "FAST"},
				"INTERACTION_TO_NEXT_PAINT":     {Percentile: 120, Category: "FAST"},
				"CUMULATIVE_LAYOUT_SHIFT_SCORE": {Percentile: 5, Category: "FAST"},
			},
		},
		LighthouseResult: psiLighthouse{
			Categories: map[string]struct {
				Score *float64 `json:"score"`
			}{"performance": {Score: floatPtr(0.88)}},
			Audits: map[string]psiAudit{
				"first-contentful-paint": {DisplayValue: "1.0 s"},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("strategy") == "desktop" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"code":500,"message":"backend error"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mobile)
	}))
	t.Cleanup(srv.Close)
	prev := pagespeedRunURL
	pagespeedRunURL = srv.URL
	t.Cleanup(func() { pagespeedRunURL = prev })

	report, checks := InspectCWV(srv.Client(), "https://example.com/", "", time.Now())
	if !report.FieldAvailable || report.OverallLabel != "Good" || report.Score != 100 {
		t.Fatalf("desktop 500 must not wipe mobile CWV: field=%v label=%s score=%d err=%s",
			report.FieldAvailable, report.OverallLabel, report.Score, report.Error)
	}
	if report.DesktopPerformance != nil {
		t.Fatalf("desktop lab must be empty on error, got %v", *report.DesktopPerformance)
	}
	if report.DesktopError == "" {
		t.Fatal("expected desktop_error")
	}
	foundDesktopWarn := false
	for _, c := range checks {
		if c.Label == "Lighthouse desktop" && c.Category == lighthouseCategory && c.Status == CheckWarning {
			foundDesktopWarn = true
		}
		if c.Category == cwvCategory && c.Label == "Lighthouse desktop" {
			t.Fatalf("desktop lab must not be a cwv check: %+v", c)
		}
	}
	if !foundDesktopWarn {
		t.Fatalf("expected lighthouse desktop warning: %+v", checks)
	}
}

func TestInspectCWV_LabOnlyScoreStaysZero(t *testing.T) {
	n := 99
	if s := cwvScore(CWVReport{
		LabPerformance:     &n,
		LabFCP:             "0.8 s",
		LabTTFB:            "40 ms",
		DesktopPerformance: &n,
		Opportunities:      []PSIOpportunity{{ID: "x", Title: "Unused JS", SavingsMs: 500}},
	}); s != 0 {
		t.Fatalf("lab extras must not count as CWV score, got %d", s)
	}
}

func floatPtr(v float64) *float64 { return &v }
