package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	cwvCategory         = "cwv"
	lighthouseCategory  = "lighthouse"
	cwvMaxBody          = 4 * 1024 * 1024
	pagespeedDefaultURL = "https://www.googleapis.com/pagespeedonline/v5/runPagespeed"
)

// pagespeedRunURL is the PSI endpoint. Tests replace it with an httptest server.
var pagespeedRunURL = pagespeedDefaultURL

// CWVMetric is one LCP/INP/CLS reading.
type CWVMetric struct {
	Name     string  `json:"name"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
	Display  string  `json:"display"`
	Category string  `json:"category,omitempty"` // FAST | AVERAGE | SLOW
	Rating   string  `json:"rating,omitempty"`   // good | needs_improvement | poor
	Source   string  `json:"source"`             // field_url | field_origin | lab
}

// CWVReport is PageSpeed Insights field (CrUX) + lab (Lighthouse) for one URL.
// Field data is Core Web Vitals. Lab is not CWV and is labeled separately.
type CWVReport struct {
	URL            string           `json:"url"`
	Strategy       string           `json:"strategy"`
	FieldScope     string           `json:"field_scope,omitempty"` // url | origin
	FieldAvailable bool             `json:"field_available"`
	OverallLabel   string           `json:"overall_label"`
	LCP            *CWVMetric       `json:"lcp,omitempty"`
	INP            *CWVMetric       `json:"inp,omitempty"`
	CLS            *CWVMetric       `json:"cls,omitempty"`
	LabPerformance *int             `json:"lab_performance,omitempty"`
	LabLCP         string           `json:"lab_lcp,omitempty"`
	LabFCP         string           `json:"lab_fcp,omitempty"`
	LabTTFB        string           `json:"lab_ttfb,omitempty"`
	LabCLS         string           `json:"lab_cls,omitempty"`
	Opportunities  []PSIOpportunity `json:"opportunities,omitempty"`

	DesktopPerformance *int   `json:"desktop_performance,omitempty"`
	DesktopLCP         string `json:"desktop_lcp,omitempty"`
	DesktopFCP         string `json:"desktop_fcp,omitempty"`
	DesktopTTFB        string `json:"desktop_ttfb,omitempty"`
	DesktopCLS         string `json:"desktop_cls,omitempty"`
	DesktopError       string `json:"desktop_error,omitempty"`

	Error     string    `json:"error,omitempty"`
	Score     int       `json:"score"`
	Status    string    `json:"status"`
	CheckedAt time.Time `json:"checked_at"`
}

// PSIOpportunity is one Lighthouse lab opportunity with measured savings.
// Simulated — not Core Web Vitals and not a sitewide crawl.
type PSIOpportunity struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Display   string `json:"display,omitempty"`
	SavingsMs int    `json:"savings_ms,omitempty"`
}

type psiResponse struct {
	LoadingExperience       psiExperience `json:"loadingExperience"`
	OriginLoadingExperience psiExperience `json:"originLoadingExperience"`
	LighthouseResult        psiLighthouse `json:"lighthouseResult"`
	Error                   *psiAPIError  `json:"error"`
}

type psiAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type psiExperience struct {
	ID              string               `json:"id"`
	OverallCategory string               `json:"overall_category"`
	Metrics         map[string]psiMetric `json:"metrics"`
}

type psiMetric struct {
	Percentile float64 `json:"percentile"`
	Category   string  `json:"category"`
}

type psiLighthouse struct {
	Categories map[string]struct {
		Score *float64 `json:"score"`
	} `json:"categories"`
	Audits map[string]psiAudit `json:"audits"`
}

type psiAudit struct {
	ID           string           `json:"id"`
	Title        string           `json:"title"`
	Score        *float64         `json:"score"`
	NumericValue *float64         `json:"numericValue"`
	DisplayValue string           `json:"displayValue"`
	Details      *psiAuditDetails `json:"details"`
}

type psiAuditDetails struct {
	Type                string  `json:"type"`
	OverallSavingsMs    float64 `json:"overallSavingsMs"`
	OverallSavingsBytes float64 `json:"overallSavingsBytes"`
}

// InspectCWV calls PageSpeed Insights for mobile field CrUX + lab Lighthouse,
// and a parallel desktop lab run. Desktop CrUX is not mixed into CWV.
func InspectCWV(client *http.Client, pageURL, apiKey string, checkedAt time.Time) (CWVReport, []AuditCheck) {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	report := CWVReport{
		URL:       pageURL,
		Strategy:  "mobile",
		CheckedAt: checkedAt.UTC(),
		Status:    CheckWarning,
	}
	checked := formatRobotsChecked(checkedAt)
	if strings.TrimSpace(pageURL) == "" {
		report.OverallLabel = "Unavailable"
		report.Error = "No URL to measure."
		return report, []AuditCheck{cwvCheck("Core Web Vitals (CrUX)", CheckWarning, "medium",
			"No URL to measure. Last checked "+checked+".", "")}
	}

	var (
		mobileBody, desktopBody []byte
		mobileCode, desktopCode int
		mobileErr, desktopErr   error
	)
	var fetchWG sync.WaitGroup
	fetchWG.Add(2)
	go func() {
		defer fetchWG.Done()
		mobileBody, mobileCode, mobileErr = fetchPagespeed(client, pageURL, apiKey, "mobile")
	}()
	go func() {
		defer fetchWG.Done()
		desktopBody, desktopCode, desktopErr = fetchPagespeed(client, pageURL, apiKey, "desktop")
	}()
	fetchWG.Wait()

	if mobileErr != nil {
		report.OverallLabel = "Unavailable"
		report.Error = mobileErr.Error()
		applyDesktopLab(&report, desktopBody, desktopCode, desktopErr)
		checks := []AuditCheck{cwvCheck("Core Web Vitals (CrUX)", CheckWarning, "medium",
			fmt.Sprintf("PageSpeed Insights could not measure this URL: %s. Last checked %s.", mobileErr.Error(), checked),
			"Set PAGESPEED_API_KEY (Google Cloud PageSpeed Insights API) if anonymous quota is exhausted.")}
		return report, append(checks, lighthouseExtraChecks(report, checked)...)
	}

	psi, perr := decodePSI(mobileBody, mobileCode)
	if perr != nil {
		report.OverallLabel = "Unavailable"
		report.Error = perr.Error()
		applyDesktopLab(&report, desktopBody, desktopCode, desktopErr)
		checks := []AuditCheck{cwvCheck("Core Web Vitals (CrUX)", CheckWarning, "medium",
			fmt.Sprintf("PageSpeed Insights error: %s. Last checked %s.", perr.Error(), checked),
			"Enable the PageSpeed Insights API and set PAGESPEED_API_KEY.")}
		return report, append(checks, lighthouseExtraChecks(report, checked)...)
	}

	urlExp, originExp := psi.LoadingExperience, psi.OriginLoadingExperience
	scope, exp := pickCrUX(urlExp, originExp)
	if scope != "" {
		report.FieldScope = scope
		report.FieldAvailable = true
		src := "field_url"
		if scope == "origin" {
			src = "field_origin"
		}
		report.LCP = cruxMetric(exp.Metrics, "LARGEST_CONTENTFUL_PAINT_MS", "LCP", "ms", false, src)
		report.INP = cruxMetric(exp.Metrics, "INTERACTION_TO_NEXT_PAINT", "INP", "ms", false, src)
		report.CLS = cruxMetric(exp.Metrics, "CUMULATIVE_LAYOUT_SHIFT_SCORE", "CLS", "score", true, src)
	}

	lab := labFromLighthouse(psi.LighthouseResult)
	report.LabPerformance = lab.Performance
	report.LabLCP = lab.LCP
	report.LabFCP = lab.FCP
	report.LabTTFB = lab.TTFB
	report.LabCLS = lab.CLS
	report.Opportunities = lab.Opportunities

	applyDesktopLab(&report, desktopBody, desktopCode, desktopErr)

	report.Status, report.OverallLabel = cwvOverall(report)
	report.Score = cwvScore(report)
	return report, cwvChecksFromReport(report, checked)
}

func decodePSI(body []byte, statusCode int) (psiResponse, error) {
	var psi psiResponse
	if err := json.Unmarshal(body, &psi); err != nil {
		return psi, fmt.Errorf("invalid PageSpeed JSON (HTTP %d)", statusCode)
	}
	if psi.Error != nil && psi.Error.Message != "" {
		return psi, fmt.Errorf("%s", psi.Error.Message)
	}
	if statusCode >= 400 {
		return psi, fmt.Errorf("HTTP %d", statusCode)
	}
	return psi, nil
}

func applyDesktopLab(report *CWVReport, body []byte, statusCode int, fetchErr error) {
	if fetchErr != nil {
		report.DesktopError = fetchErr.Error()
		return
	}
	if len(body) == 0 {
		return
	}
	psi, err := decodePSI(body, statusCode)
	if err != nil {
		report.DesktopError = err.Error()
		return
	}
	lab := labFromLighthouse(psi.LighthouseResult)
	report.DesktopPerformance = lab.Performance
	report.DesktopLCP = lab.LCP
	report.DesktopFCP = lab.FCP
	report.DesktopTTFB = lab.TTFB
	report.DesktopCLS = lab.CLS
}

func fetchPagespeed(client *http.Client, pageURL, apiKey, strategy string) ([]byte, int, error) {
	if client == nil {
		client = &http.Client{Timeout: 55 * time.Second}
	}
	if strategy == "" {
		strategy = "mobile"
	}
	u, err := url.Parse(pagespeedRunURL)
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	q.Set("url", pageURL)
	q.Set("strategy", strategy)
	q.Set("category", "PERFORMANCE")
	if strings.TrimSpace(apiKey) != "" {
		q.Set("key", apiKey)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "DMTool-SEOCrawler/2.0 (+https://dmtool.app)")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, cwvMaxBody+1))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(body) > cwvMaxBody {
		return nil, resp.StatusCode, fmt.Errorf("PageSpeed response too large")
	}
	return body, resp.StatusCode, nil
}

func pickCrUX(urlExp, originExp psiExperience) (scope string, exp psiExperience) {
	if cruxHasVitals(urlExp) {
		return "url", urlExp
	}
	if cruxHasVitals(originExp) {
		return "origin", originExp
	}
	return "", psiExperience{}
}

func cruxHasVitals(exp psiExperience) bool {
	if len(exp.Metrics) == 0 {
		return false
	}
	_, lcp := exp.Metrics["LARGEST_CONTENTFUL_PAINT_MS"]
	_, inp := exp.Metrics["INTERACTION_TO_NEXT_PAINT"]
	_, cls := exp.Metrics["CUMULATIVE_LAYOUT_SHIFT_SCORE"]
	return lcp || inp || cls
}

func cruxMetric(metrics map[string]psiMetric, key, name, unit string, isCLS bool, source string) *CWVMetric {
	if metrics == nil {
		return nil
	}
	m, ok := metrics[key]
	if !ok {
		return nil
	}
	if m.Category == "" && m.Percentile == 0 {
		return nil
	}
	val := m.Percentile
	if isCLS {
		val = cruxCLS(val)
	}
	out := &CWVMetric{
		Name:     name,
		Value:    val,
		Unit:     unit,
		Category: strings.ToUpper(m.Category),
		Rating:   ratingFromCategory(m.Category),
		Source:   source,
	}
	if isCLS {
		out.Display = formatCLS(val)
	} else {
		out.Display = formatMs(val)
	}
	if out.Rating == "" {
		out.Rating = ratingFromThreshold(name, val, isCLS)
	}
	return out
}

func cruxCLS(percentile float64) float64 {
	// CrUX CLS percentile is typically 100× the CLS score (20 → 0.20).
	if percentile > 1 {
		return percentile / 100
	}
	return percentile
}

func ratingFromCategory(cat string) string {
	switch strings.ToUpper(strings.TrimSpace(cat)) {
	case "FAST":
		return "good"
	case "AVERAGE":
		return "needs_improvement"
	case "SLOW":
		return "poor"
	default:
		return ""
	}
}

func ratingFromThreshold(name string, val float64, isCLS bool) string {
	if isCLS {
		switch {
		case val <= 0.1:
			return "good"
		case val <= 0.25:
			return "needs_improvement"
		default:
			return "poor"
		}
	}
	// LCP and INP in milliseconds.
	if strings.EqualFold(name, "LCP") {
		switch {
		case val <= 2500:
			return "good"
		case val <= 4000:
			return "needs_improvement"
		default:
			return "poor"
		}
	}
	// INP
	switch {
	case val <= 200:
		return "good"
	case val <= 500:
		return "needs_improvement"
	default:
		return "poor"
	}
}

type labSnapshot struct {
	Performance   *int
	LCP           string
	FCP           string
	CLS           string
	TTFB          string
	Opportunities []PSIOpportunity
}

func labFromLighthouse(lh psiLighthouse) labSnapshot {
	var out labSnapshot
	if cat, ok := lh.Categories["performance"]; ok && cat.Score != nil {
		n := int((*cat.Score)*100 + 0.5)
		if n < 0 {
			n = 0
		}
		if n > 100 {
			n = 100
		}
		out.Performance = &n
	}
	out.LCP = lighthouseAuditDisplay(lh, "largest-contentful-paint", false)
	out.FCP = lighthouseAuditDisplay(lh, "first-contentful-paint", false)
	out.TTFB = lighthouseAuditDisplay(lh, "server-response-time", false)
	if out.TTFB == "" {
		out.TTFB = lighthouseAuditDisplay(lh, "time-to-first-byte", false)
	}
	out.CLS = lighthouseAuditDisplay(lh, "cumulative-layout-shift", true)
	out.Opportunities = lighthouseOpportunities(lh, 8)
	return out
}

func lighthouseAuditDisplay(lh psiLighthouse, id string, cls bool) string {
	a, ok := lh.Audits[id]
	if !ok {
		return ""
	}
	if cls {
		if a.NumericValue != nil {
			return formatCLS(*a.NumericValue)
		}
		return a.DisplayValue
	}
	if a.DisplayValue != "" {
		return a.DisplayValue
	}
	if a.NumericValue != nil {
		return formatMs(*a.NumericValue)
	}
	return ""
}

func lighthouseOpportunities(lh psiLighthouse, limit int) []PSIOpportunity {
	if limit <= 0 {
		return nil
	}
	out := make([]PSIOpportunity, 0)
	for id, a := range lh.Audits {
		if a.Details == nil || !strings.EqualFold(a.Details.Type, "opportunity") {
			continue
		}
		savings := int(a.Details.OverallSavingsMs + 0.5)
		if savings <= 0 && strings.TrimSpace(a.DisplayValue) == "" {
			continue
		}
		title := strings.TrimSpace(a.Title)
		if title == "" {
			title = id
		}
		aid := a.ID
		if aid == "" {
			aid = id
		}
		out = append(out, PSIOpportunity{
			ID:        aid,
			Title:     title,
			Display:   strings.TrimSpace(a.DisplayValue),
			SavingsMs: savings,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SavingsMs == out[j].SavingsMs {
			return out[i].Title < out[j].Title
		}
		return out[i].SavingsMs > out[j].SavingsMs
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func cwvOverall(r CWVReport) (status, label string) {
	if !r.FieldAvailable {
		if r.LabPerformance != nil {
			return CheckWarning, "Lab only"
		}
		return CheckWarning, "Unavailable"
	}
	worst := "good"
	for _, m := range []*CWVMetric{r.LCP, r.INP, r.CLS} {
		if m == nil {
			continue
		}
		switch m.Rating {
		case "poor":
			worst = "poor"
		case "needs_improvement":
			if worst != "poor" {
				worst = "needs_improvement"
			}
		}
	}
	switch worst {
	case "good":
		return CheckPass, "Good"
	case "needs_improvement":
		return CheckWarning, "Needs work"
	default:
		return CheckFail, "Poor"
	}
}

func cwvScore(r CWVReport) int {
	if !r.FieldAvailable {
		return 0
	}
	n := 0
	for _, m := range []*CWVMetric{r.LCP, r.INP, r.CLS} {
		if m == nil {
			continue
		}
		switch m.Rating {
		case "good":
			n += 34
		case "needs_improvement":
			n += 20
		case "poor":
			n += 5
		}
	}
	if n > 100 {
		n = 100
	}
	return n
}

func cwvChecksFromReport(r CWVReport, checked string) []AuditCheck {
	scopeNote := "URL-level Chrome UX Report (mobile, ~28 days)."
	if r.FieldScope == "origin" {
		scopeNote = "Not enough URL-level CrUX data; using origin-level Chrome UX Report (mobile, ~28 days)."
	}
	checks := make([]AuditCheck, 0, 6)

	if !r.FieldAvailable {
		checks = append(checks, cwvCheck("Core Web Vitals (CrUX)", CheckWarning, "medium",
			fmt.Sprintf("Not enough Chrome UX Report field data for this URL or its origin. Last checked %s.", checked),
			"CWV is real-user field data. Lab Lighthouse is not a CWV substitute."))
	} else {
		detail := fmt.Sprintf("%s %s Last checked %s.", r.OverallLabel, scopeNote, checked)
		sev := "high"
		if r.Status == CheckWarning {
			sev = "medium"
		}
		if r.Status == CheckPass {
			sev = "high"
		}
		checks = append(checks, cwvCheck("Core Web Vitals (CrUX)", r.Status, sev, detail, cwvRec(r)))
		checks = append(checks, metricCheck("LCP", r.LCP, checked))
		if r.INP != nil {
			checks = append(checks, metricCheck("INP", r.INP, checked))
		} else {
			checks = append(checks, cwvCheck("INP", CheckWarning, "medium",
				"INP is not in this CrUX record. Last checked "+checked+".",
				"INP requires field data; Lighthouse does not measure INP in the lab."))
		}
		checks = append(checks, metricCheck("CLS", r.CLS, checked))
	}

	checks = append(checks, lighthouseExtraChecks(r, checked)...)
	return checks
}

func lighthouseExtraChecks(r CWVReport, checked string) []AuditCheck {
	checks := make([]AuditCheck, 0, 6)
	if r.LabPerformance != nil {
		n := *r.LabPerformance
		labStatus, sev := CheckPass, "low"
		switch {
		case n < 50:
			labStatus, sev = CheckFail, "medium"
		case n < 90:
			labStatus, sev = CheckWarning, "low"
		}
		detail := fmt.Sprintf("Lighthouse mobile performance %d/100.", n)
		if r.LabLCP != "" {
			detail += " Lab LCP " + r.LabLCP + "."
		}
		if r.LabFCP != "" {
			detail += " Lab FCP " + r.LabFCP + "."
		}
		if r.LabTTFB != "" {
			detail += " Lab TTFB " + r.LabTTFB + "."
		}
		if r.LabCLS != "" {
			detail += " Lab CLS " + r.LabCLS + "."
		}
		detail += " Lab is a simulated test, not Core Web Vitals. Last checked " + checked + "."
		checks = append(checks, lighthouseCheck("Lighthouse performance", labStatus, sev, detail, ""))
	}

	if r.LabFCP != "" {
		checks = append(checks, lighthouseCheck("Lab FCP", CheckPass, "low",
			fmt.Sprintf("Lighthouse mobile FCP %s. Simulated lab metric, not a Core Web Vital. Last checked %s.", r.LabFCP, checked),
			""))
	}
	if r.LabTTFB != "" {
		checks = append(checks, lighthouseCheck("Lab TTFB", CheckPass, "low",
			fmt.Sprintf("Lighthouse mobile TTFB (server response) %s. Simulated lab metric, not a Core Web Vital. Last checked %s.", r.LabTTFB, checked),
			""))
	}

	if len(r.Opportunities) > 0 {
		parts := make([]string, 0, len(r.Opportunities))
		for _, o := range r.Opportunities {
			line := o.Title
			if o.Display != "" {
				line += " (" + o.Display + ")"
			} else if o.SavingsMs > 0 {
				line += fmt.Sprintf(" (~%dms)", o.SavingsMs)
			}
			parts = append(parts, line)
		}
		checks = append(checks, lighthouseCheck("Lighthouse opportunities", CheckWarning, "low",
			"Lab-only savings on this URL (not CWV, not sitewide): "+strings.Join(parts, "; ")+". Last checked "+checked+".",
			"These are Lighthouse estimates from one simulated mobile run."))
	}

	if r.DesktopError != "" && r.DesktopPerformance == nil {
		checks = append(checks, lighthouseCheck("Lighthouse desktop", CheckWarning, "low",
			fmt.Sprintf("Desktop PageSpeed Insights unavailable: %s. Last checked %s.", r.DesktopError, checked),
			"Mobile lab still applies. Desktop is a separate simulated run, not CWV."))
	} else if r.DesktopPerformance != nil {
		n := *r.DesktopPerformance
		st, sev := CheckPass, "low"
		switch {
		case n < 50:
			st, sev = CheckFail, "medium"
		case n < 90:
			st, sev = CheckWarning, "low"
		}
		detail := fmt.Sprintf("Lighthouse desktop performance %d/100.", n)
		if r.DesktopLCP != "" {
			detail += " Lab LCP " + r.DesktopLCP + "."
		}
		if r.DesktopFCP != "" {
			detail += " Lab FCP " + r.DesktopFCP + "."
		}
		if r.DesktopTTFB != "" {
			detail += " Lab TTFB " + r.DesktopTTFB + "."
		}
		detail += " Desktop lab is simulated, not Core Web Vitals. Last checked " + checked + "."
		checks = append(checks, lighthouseCheck("Lighthouse desktop", st, sev, detail, ""))
	}

	if len(checks) == 0 {
		msg := "No Lighthouse lab result for this URL."
		if r.Error != "" {
			msg = "PageSpeed lab unavailable: " + r.Error + "."
		}
		checks = append(checks, lighthouseCheck("Lighthouse performance", CheckWarning, "low",
			msg+" Lab is simulated, not Core Web Vitals. Last checked "+checked+".", ""))
	}
	return checks
}

func lighthouseCheck(label, status, severity, detail, rec string) AuditCheck {
	return AuditCheck{
		Category:       lighthouseCategory,
		Label:          label,
		Status:         status,
		Severity:       severity,
		Detail:         detail,
		Recommendation: rec,
	}
}

func metricCheck(label string, m *CWVMetric, checked string) AuditCheck {
	if m == nil {
		return cwvCheck(label, CheckWarning, "medium",
			label+" is missing from CrUX. Last checked "+checked+".", "")
	}
	st := CheckPass
	sev := "high"
	switch m.Rating {
	case "needs_improvement":
		st, sev = CheckWarning, "medium"
	case "poor":
		st, sev = CheckFail, "high"
	}
	src := "this URL"
	if m.Source == "field_origin" {
		src = "the origin (not this specific URL)"
	}
	return cwvCheck(label, st, sev,
		fmt.Sprintf("%s %s (%s) for %s. Last checked %s.", label, m.Display, ratingLabel(m.Rating), src, checked),
		"")
}

func ratingLabel(r string) string {
	switch r {
	case "good":
		return "good"
	case "needs_improvement":
		return "needs improvement"
	case "poor":
		return "poor"
	default:
		return r
	}
}

func cwvRec(r CWVReport) string {
	if r.Status == CheckPass {
		return ""
	}
	return "Improve the failing field metric(s) on real mobile traffic. Lab-only changes are not a CWV pass."
}

func cwvCheck(label, status, severity, detail, rec string) AuditCheck {
	return AuditCheck{
		Category:       cwvCategory,
		Label:          label,
		Status:         status,
		Severity:       severity,
		Detail:         detail,
		Recommendation: rec,
	}
}

func formatMs(ms float64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.1fs", ms/1000)
	}
	return fmt.Sprintf("%.0fms", ms)
}

func formatCLS(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
