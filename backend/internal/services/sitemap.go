package services

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	sitemapCategory         = "sitemap"
	sitemapMaxBytes         = 2 * 1024 * 1024
	sitemapMaxFiles         = 5
	sitemapMaxIndexChildren = 3
	sitemapKindURLSet       = "urlset"
	sitemapKindIndex        = "sitemapindex"
	sitemapKindHTML         = "html"
	sitemapKindEmpty        = "empty"
	sitemapKindError        = "error"
)

// SitemapFile is one fetched sitemap or index document.
type SitemapFile struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	Kind       string `json:"kind"`
	URLCount   int    `json:"url_count,omitempty"`
	ChildCount int    `json:"child_count,omitempty"`
	Truncated  bool   `json:"truncated,omitempty"`
	FetchError string `json:"fetch_error,omitempty"`
	FromRobots bool   `json:"from_robots,omitempty"`
	Parent     string `json:"parent,omitempty"`
}

// SitemapReport is the structured sitemap XML outcome from an audit.
// URLCount is how many <loc> entries we counted in fetched urlset XML —
// not a sitewide crawl and not a complete index of the site.
type SitemapReport struct {
	FallbackURL        string        `json:"fallback_url,omitempty"`
	DeclaredFromRobots bool          `json:"declared_from_robots"`
	Declared           []string      `json:"declared,omitempty"`
	Files              []SitemapFile `json:"files,omitempty"`
	URLCount           int           `json:"url_count"`
	IndexChildren      int           `json:"index_children,omitempty"`
	Sample             []string      `json:"sample,omitempty"`
	Truncated          bool          `json:"truncated,omitempty"`
	Status             string        `json:"status"`
	CheckedAt          time.Time     `json:"checked_at"`
}

type parsedSitemap struct {
	kind      string
	locs      []string
	truncated bool
	html      bool
	empty     bool
	invalid   bool
}

// InspectSitemap fetches robots-declared sitemap URLs (or origin /sitemap.xml)
// and parses urlset / sitemapindex XML. It does not crawl the listed pages.
func InspectSitemap(client *http.Client, declared []string, origin *url.URL, checkedAt time.Time) (SitemapReport, []AuditCheck) {
	if client == nil {
		client = http.DefaultClient
	}
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	report := SitemapReport{CheckedAt: checkedAt.UTC()}
	targets, fromRobots, fallback := sitemapTargets(declared, origin)
	report.DeclaredFromRobots = fromRobots
	report.Declared = uniqueHTTPURLs(declared)
	report.FallbackURL = fallback

	if len(targets) == 0 {
		checks := []AuditCheck{
			sitemapCheck("Sitemap fetch", CheckWarning, "medium",
				"No sitemap URL to fetch (invalid origin).",
				"Publish an XML sitemap and declare it with a Sitemap: line in robots.txt."),
		}
		report.Status = CheckWarning
		return report, checks
	}

	seen := map[string]bool{}
	fetched := 0
	for _, raw := range targets {
		if fetched >= sitemapMaxFiles {
			report.Truncated = true
			break
		}
		if seen[raw] {
			continue
		}
		seen[raw] = true
		file, childLocs, samples := fetchSitemapFile(client, raw, fromRobots, "")
		fetched++
		report.Files = append(report.Files, file)
		addSitemapSamples(&report, samples)
		if file.Kind == sitemapKindURLSet {
			report.URLCount += file.URLCount
		}
		if file.Truncated {
			report.Truncated = true
		}
		if file.Kind != sitemapKindIndex {
			continue
		}
		report.IndexChildren += file.ChildCount
		childrenGot := 0
		for _, child := range childLocs {
			if fetched >= sitemapMaxFiles || childrenGot >= sitemapMaxIndexChildren {
				report.Truncated = true
				break
			}
			child = resolveSitemapURL(raw, child)
			if child == "" || seen[child] {
				continue
			}
			seen[child] = true
			cfile, _, csamples := fetchSitemapFile(client, child, false, raw)
			fetched++
			childrenGot++
			report.Files = append(report.Files, cfile)
			addSitemapSamples(&report, csamples)
			if cfile.Kind == sitemapKindURLSet {
				report.URLCount += cfile.URLCount
			}
			if cfile.Kind == sitemapKindIndex {
				report.IndexChildren += cfile.ChildCount
			}
			if cfile.Truncated {
				report.Truncated = true
			}
		}
	}

	checks := sitemapChecks(report, fromRobots, checkedAt)
	report.Status = worstRobotsStatus(checks)
	return report, checks
}

func sitemapTargets(declared []string, origin *url.URL) (targets []string, fromRobots bool, fallback string) {
	clean := uniqueHTTPURLs(declared)
	if len(clean) > 0 {
		if len(clean) > sitemapMaxFiles {
			clean = clean[:sitemapMaxFiles]
		}
		return clean, true, ""
	}
	if origin == nil || origin.Host == "" {
		return nil, false, ""
	}
	scheme := origin.Scheme
	if scheme == "" {
		scheme = "https"
	}
	fallback = scheme + "://" + origin.Host + "/sitemap.xml"
	return []string{fallback}, false, fallback
}

func uniqueHTTPURLs(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, raw := range in {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || u.Host == "" {
			continue
		}
		if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
			continue
		}
		key := u.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

func resolveSitemapURL(base, loc string) string {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return ""
	}
	ref, err := url.Parse(loc)
	if err != nil {
		return ""
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		if strings.EqualFold(ref.Scheme, "http") || strings.EqualFold(ref.Scheme, "https") {
			return ref.String()
		}
		return ""
	}
	resolved := baseURL.ResolveReference(ref)
	if !strings.EqualFold(resolved.Scheme, "http") && !strings.EqualFold(resolved.Scheme, "https") {
		return ""
	}
	return resolved.String()
}

func fetchSitemapFile(client *http.Client, rawURL string, fromRobots bool, parent string) (SitemapFile, []string, []string) {
	file := SitemapFile{URL: rawURL, FromRobots: fromRobots, Parent: parent, Kind: sitemapKindError}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		file.FetchError = err.Error()
		return file, nil, nil
	}
	req.Header.Set("User-Agent", "DMTool-SEOCrawler/2.0 (+https://dmtool.app)")
	req.Header.Set("Accept", "application/xml,text/xml,application/gzip,*/*")

	resp, err := client.Do(req)
	if err != nil {
		file.FetchError = err.Error()
		return file, nil, nil
	}
	defer resp.Body.Close()
	file.StatusCode = resp.StatusCode

	body, truncated, err := readSitemapBody(resp.Body)
	if err != nil {
		file.FetchError = err.Error()
		return file, nil, nil
	}
	file.Truncated = truncated

	if resp.StatusCode != http.StatusOK {
		file.Kind = sitemapKindError
		file.FetchError = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return file, nil, nil
	}

	parsed := parseSitemapXML(body)
	file.Truncated = file.Truncated || parsed.truncated
	switch {
	case parsed.html:
		file.Kind = sitemapKindHTML
	case parsed.invalid:
		file.Kind = sitemapKindError
		file.FetchError = "invalid XML"
	case parsed.empty && parsed.kind == "":
		file.Kind = sitemapKindEmpty
	case parsed.kind == sitemapKindIndex:
		file.Kind = sitemapKindIndex
		file.ChildCount = len(parsed.locs)
		return file, parsed.locs, nil
	case parsed.kind == sitemapKindURLSet:
		file.Kind = sitemapKindURLSet
		file.URLCount = len(parsed.locs)
		return file, nil, sampleLocs(parsed.locs, 5)
	default:
		if parsed.empty {
			file.Kind = sitemapKindEmpty
		} else {
			file.Kind = sitemapKindError
			file.FetchError = "not a urlset or sitemapindex"
		}
	}
	return file, nil, nil
}

func readSitemapBody(r io.Reader) ([]byte, bool, error) {
	raw, err := io.ReadAll(io.LimitReader(r, sitemapMaxBytes+1))
	if err != nil {
		return nil, false, err
	}
	truncated := len(raw) > sitemapMaxBytes
	if truncated {
		raw = raw[:sitemapMaxBytes]
	}
	if !isGzip(raw) {
		return raw, truncated, nil
	}
	gr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, truncated, fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()
	out, err := io.ReadAll(io.LimitReader(gr, sitemapMaxBytes+1))
	if err != nil {
		return nil, truncated, err
	}
	if len(out) > sitemapMaxBytes {
		return out[:sitemapMaxBytes], true, nil
	}
	return out, truncated, nil
}

func isGzip(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b
}

func parseSitemapXML(body []byte) parsedSitemap {
	out := parsedSitemap{}
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 {
		out.empty = true
		return out
	}
	lower := strings.ToLower(string(trim[:min(len(trim), 256)]))
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") {
		out.html = true
		return out
	}

	dec := xml.NewDecoder(bytes.NewReader(trim))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	var (
		inLoc bool
		loc   strings.Builder
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			out.invalid = true
			return out
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "urlset":
				if out.kind == "" {
					out.kind = sitemapKindURLSet
				}
			case "sitemapindex":
				if out.kind == "" {
					out.kind = sitemapKindIndex
				}
			case "loc":
				inLoc = true
				loc.Reset()
			}
		case xml.EndElement:
			if t.Name.Local == "loc" && inLoc {
				inLoc = false
				val := strings.TrimSpace(loc.String())
				if val != "" {
					out.locs = append(out.locs, val)
				}
			}
		case xml.CharData:
			if inLoc {
				loc.Write(t)
			}
		}
	}
	if out.kind == "" && len(out.locs) == 0 {
		out.empty = true
		if looksLikeXML(trim) {
			out.invalid = true
			out.empty = false
		}
	}
	if out.kind != "" && len(out.locs) == 0 {
		out.empty = true
	}
	return out
}

func looksLikeXML(b []byte) bool {
	s := strings.TrimSpace(string(b))
	return strings.HasPrefix(s, "<?xml") || strings.HasPrefix(s, "<")
}

func addSitemapSamples(report *SitemapReport, samples []string) {
	for _, s := range samples {
		if len(report.Sample) >= 5 {
			return
		}
		report.Sample = append(report.Sample, s)
	}
}

func sampleLocs(locs []string, n int) []string {
	if len(locs) <= n {
		return append([]string(nil), locs...)
	}
	return append([]string(nil), locs[:n]...)
}

func sitemapChecks(report SitemapReport, fromRobots bool, checkedAt time.Time) []AuditCheck {
	checked := formatRobotsChecked(checkedAt)
	readable := 0
	htmlFiles, invalid, notFoundDeclared, fetchFails, emptyFiles := 0, 0, 0, 0, 0
	for _, f := range report.Files {
		switch f.Kind {
		case sitemapKindURLSet, sitemapKindIndex:
			readable++
		case sitemapKindHTML:
			htmlFiles++
		case sitemapKindEmpty:
			emptyFiles++
		case sitemapKindError:
			switch {
			case f.StatusCode == http.StatusNotFound && f.FromRobots:
				notFoundDeclared++
			case f.FetchError == "invalid XML":
				invalid++
			case f.StatusCode == http.StatusNotFound:
				// fallback /sitemap.xml missing — warning, not a fetch failure
			default:
				fetchFails++
			}
		}
	}

	checks := make([]AuditCheck, 0, 3)

	switch {
	case readable > 0:
		detail := fmt.Sprintf("Fetched %d sitemap file(s). Last checked %s.", readable, checked)
		if report.DeclaredFromRobots {
			detail = "Using Sitemap: URL(s) from robots.txt. " + detail
		} else if report.FallbackURL != "" {
			detail = "No Sitemap: in robots.txt; fetched " + report.FallbackURL + ". " + detail
		}
		if notFoundDeclared > 0 {
			checks = append(checks, sitemapCheck("Sitemap fetch", CheckWarning, "medium",
				detail+" At least one declared sitemap URL returned 404.",
				"Fix or remove broken Sitemap: lines in robots.txt."))
		} else {
			checks = append(checks, sitemapCheck("Sitemap fetch", CheckPass, "medium", detail, ""))
		}
	case notFoundDeclared > 0 && fromRobots:
		checks = append(checks, sitemapCheck("Sitemap fetch", CheckFail, "high",
			fmt.Sprintf("robots.txt declares a sitemap that returned HTTP 404. Last checked %s.", checked),
			"Publish the XML file at the declared URL or update the Sitemap: line."))
	case emptyFiles > 0:
		checks = append(checks, sitemapCheck("Sitemap fetch", CheckWarning, "medium",
			fmt.Sprintf("Sitemap file was empty. Last checked %s.", checked),
			"Publish urlset or sitemapindex XML at this URL."))
	case fetchFails > 0 || htmlFiles > 0 || invalid > 0:
		checks = append(checks, sitemapCheck("Sitemap fetch", CheckFail, "high",
			fmt.Sprintf("Could not read a usable sitemap. Last checked %s.", checked),
			"Serve a public XML sitemap (urlset or sitemapindex) over HTTP 200."))
	default:
		msg := "No sitemap.xml at the site origin."
		if report.FallbackURL != "" {
			msg = "No sitemap.xml at " + report.FallbackURL + "."
		}
		checks = append(checks, sitemapCheck("Sitemap fetch", CheckWarning, "medium",
			msg+" robots.txt did not declare one. Last checked "+checked+".",
			"Generate an XML sitemap and add a Sitemap: line in robots.txt. Submit it in Search Console."))
	}

	switch {
	case htmlFiles > 0 && readable == 0:
		checks = append(checks, sitemapCheck("Sitemap XML", CheckFail, "high",
			"Response looks like HTML, not sitemap XML (often a custom 404 page).",
			"Serve application/xml with a <urlset> or <sitemapindex> root. Do not redirect sitemap.xml to the homepage."))
	case invalid > 0 && readable == 0:
		checks = append(checks, sitemapCheck("Sitemap XML", CheckFail, "high",
			"Sitemap body is not valid urlset/sitemapindex XML.",
			"Fix the XML so it matches the sitemaps.org schema."))
	case emptyFiles > 0 && readable == 0:
		checks = append(checks, sitemapCheck("Sitemap XML", CheckWarning, "low",
			"Sitemap file had no <urlset> or <sitemapindex> content.",
			"Emit standard sitemap XML with at least one <loc>."))
	case readable > 0:
		kindNote := "Parsed as XML sitemap."
		indexes, urlsets := 0, 0
		for _, f := range report.Files {
			if f.Kind == sitemapKindIndex {
				indexes++
			}
			if f.Kind == sitemapKindURLSet {
				urlsets++
			}
		}
		switch {
		case indexes > 0 && urlsets > 0:
			kindNote = fmt.Sprintf("Parsed sitemap index (%d child sitemap URL(s) listed) plus %d urlset file(s).", report.IndexChildren, urlsets)
		case indexes > 0:
			kindNote = fmt.Sprintf("Parsed sitemap index listing %d child sitemap URL(s).", report.IndexChildren)
		default:
			kindNote = "Parsed urlset XML."
		}
		if report.Truncated {
			kindNote += " Fetch was capped (max 5 files / 2 MiB / 3 index children) — counts are from what was downloaded, not the whole site."
		}
		checks = append(checks, sitemapCheck("Sitemap XML", CheckPass, "medium", kindNote, ""))
	}

	switch {
	case report.URLCount > 0:
		detail := fmt.Sprintf("Counted %d <loc> URL(s) in fetched sitemap XML. This is not a sitewide crawl.", report.URLCount)
		if report.Truncated {
			detail += " Additional files or bytes were not downloaded."
		}
		if len(report.Sample) > 0 {
			detail += " Sample: " + strings.Join(report.Sample, ", ") + "."
		}
		checks = append(checks, sitemapCheck("Sitemap URLs", CheckPass, "medium", detail, ""))
	case readable > 0:
		checks = append(checks, sitemapCheck("Sitemap URLs", CheckWarning, "medium",
			"Sitemap XML was readable but listed 0 <loc> URLs.",
			"Add <url><loc>…</loc></url> entries for pages you want crawled. An empty sitemap does not inventory the site."))
	}

	return checks
}

func sitemapCheck(label, status, severity, detail, rec string) AuditCheck {
	return AuditCheck{
		Category:       sitemapCategory,
		Label:          label,
		Status:         status,
		Severity:       severity,
		Detail:         detail,
		Recommendation: rec,
	}
}
