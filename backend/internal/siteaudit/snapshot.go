package siteaudit

import (
	"strconv"
	"time"

	"backend/internal/services"
)

// ScoreCell is one list/overview metric. Empty stays null. Features we do
// not compute use Status "not_implemented".
type ScoreCell struct {
	Value  *int   `json:"value"`
	Delta  *int   `json:"delta,omitempty"`
	Status string `json:"status"` // ok | empty | not_implemented
	Label  string `json:"label,omitempty"`
}

// PageRow is one crawled URL in the Crawled Pages report.
type PageRow struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	Title      string `json:"title"`
	Score      int    `json:"score"`
	Errors     int    `json:"errors"`
	Warnings   int    `json:"warnings"`
	Notices    int    `json:"notices"`
	Inlinks    int    `json:"inlinks"`
	Outlinks   int    `json:"outlinks"`
	Indexable  bool   `json:"indexable"`
	LoadTimeMs int64  `json:"load_time_ms"`
}

// IssueRow is one finding from the last crawl.
type IssueRow struct {
	URL            string `json:"url"`
	Severity       string `json:"severity"` // error | warning | notice
	Category       string `json:"category"`
	Label          string `json:"label"`
	Detail         string `json:"detail"`
	Recommendation string `json:"recommendation"`
}

// Snapshot is the stored last (or previous) crawl. Nil scores mean "—".
type Snapshot struct {
	CrawledAt     *time.Time     `json:"crawled_at,omitempty"`
	SeedURL       string         `json:"seed_url,omitempty"`
	PagesCrawled  int            `json:"pages_crawled"`
	PageLimit     int            `json:"page_limit"`
	RunCap        int            `json:"run_cap"`
	CrawlSource   string         `json:"crawl_source,omitempty"`
	StoppedReason string         `json:"stopped_reason,omitempty"`
	SiteHealth    *int           `json:"site_health"`
	Errors        int            `json:"errors"`
	Warnings      int            `json:"warnings"`
	Notices       int            `json:"notices"`
	Thematic      Thematic       `json:"thematic"`
	Pages         []PageRow      `json:"pages"`
	Issues        []IssueRow     `json:"issues"`
	StatusCodes   map[string]int `json:"status_codes,omitempty"`
	Indexable     int            `json:"indexable"`
	Blocked       int            `json:"blocked"`
	Honesty       string         `json:"honesty"`
}

// Thematic holds the Semrush list widgets.
type Thematic struct {
	Crawlability   ScoreCell `json:"crawlability"`
	HTTPS          ScoreCell `json:"https"`
	International  ScoreCell `json:"international"`
	Performance    ScoreCell `json:"performance"`
	Linking        ScoreCell `json:"linking"`
	Markups        ScoreCell `json:"markups"`
	CoreWebVitals  ScoreCell `json:"core_web_vitals"`
	AISearchHealth ScoreCell `json:"ai_search_health"`
}

// Reports is the campaign-inside payload.
type Reports struct {
	Overview OverviewReport `json:"overview"`
	Issues   []IssueRow     `json:"issues"`
	Pages    []PageRow      `json:"pages"`
	Stats    StatsReport    `json:"statistics"`
	Compare  CompareReport  `json:"compare"`
}

type OverviewReport struct {
	SiteHealth    ScoreCell `json:"site_health"`
	PagesCrawled  *int      `json:"pages_crawled"`
	Errors        ScoreCell `json:"errors"`
	Warnings      ScoreCell `json:"warnings"`
	Notices       ScoreCell `json:"notices"`
	Thematic      Thematic  `json:"thematic"`
	Honesty       string    `json:"honesty"`
	StoppedReason string    `json:"stopped_reason,omitempty"`
}

type StatsReport struct {
	StatusCodes map[string]int `json:"status_codes"`
	Indexable   int            `json:"indexable"`
	Blocked     int            `json:"blocked"`
	Pages       int            `json:"pages"`
	Honesty     string         `json:"honesty"`
}

type CompareReport struct {
	Locked     bool      `json:"locked"`
	PlanNeeded string    `json:"plan_needed,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Current    *Snapshot `json:"current,omitempty"`
	Previous   *Snapshot `json:"previous,omitempty"`
	Honesty    string    `json:"honesty"`
}

func notImplemented(label, reason string) ScoreCell {
	return ScoreCell{Status: "not_implemented", Label: label}
}

func emptyCell() ScoreCell {
	return ScoreCell{Status: "empty"}
}

func valueCell(v int, delta *int) ScoreCell {
	x := v
	return ScoreCell{Value: &x, Delta: delta, Status: "ok"}
}

func deltaInt(cur, prev *int) *int {
	if cur == nil || prev == nil {
		return nil
	}
	d := *cur - *prev
	return &d
}

func deltaCount(cur int, prev *Snapshot, pick func(*Snapshot) int, hasPrev bool) *int {
	if !hasPrev || prev == nil {
		return nil
	}
	d := cur - pick(prev)
	return &d
}

// BuildReports turns the latest snapshot (and optional previous) into UI reports.
func BuildReports(limits Limits, snap Snapshot, prev *Snapshot) Reports {
	hasSnap := snap.CrawledAt != nil || snap.PagesCrawled > 0 || len(snap.Pages) > 0
	hasPrev := prev != nil && (prev.CrawledAt != nil || prev.PagesCrawled > 0)

	var pagesPtr *int
	var health ScoreCell
	var errors, warnings, notices ScoreCell
	thematic := emptyThematic()
	if hasSnap {
		pc := snap.PagesCrawled
		pagesPtr = &pc
		if snap.SiteHealth != nil {
			var d *int
			if hasPrev {
				d = deltaInt(snap.SiteHealth, prev.SiteHealth)
			}
			health = valueCell(*snap.SiteHealth, d)
		} else {
			health = emptyCell()
		}
		errors = valueCell(snap.Errors, deltaCount(snap.Errors, prev, func(s *Snapshot) int { return s.Errors }, hasPrev))
		warnings = valueCell(snap.Warnings, deltaCount(snap.Warnings, prev, func(s *Snapshot) int { return s.Warnings }, hasPrev))
		notices = valueCell(snap.Notices, deltaCount(snap.Notices, prev, func(s *Snapshot) int { return s.Notices }, hasPrev))
		thematic = withDeltas(snap.Thematic, prev, hasPrev)
	}

	compare := CompareReport{
		Honesty: "Compare Crawls uses the last two stored snapshots. Daily history is not archived.",
	}
	if !limits.CompareCrawls {
		compare.Locked = true
		compare.PlanNeeded = "SEO"
		compare.Reason = "Compare Crawls is included on SEO and above."
	} else if hasSnap {
		cur := snap
		compare.Current = &cur
		if hasPrev {
			compare.Previous = prev
		}
	}

	honesty := snap.Honesty
	if honesty == "" {
		honesty = DefaultHonesty(limits, snap.PagesCrawled)
	}

	return Reports{
		Overview: OverviewReport{
			SiteHealth:    health,
			PagesCrawled:  pagesPtr,
			Errors:        errors,
			Warnings:      warnings,
			Notices:       notices,
			Thematic:      thematic,
			Honesty:       honesty,
			StoppedReason: snap.StoppedReason,
		},
		Issues: snap.Issues,
		Pages:  snap.Pages,
		Stats: StatsReport{
			StatusCodes: snap.StatusCodes,
			Indexable:   snap.Indexable,
			Blocked:     snap.Blocked,
			Pages:       snap.PagesCrawled,
			Honesty:     "Counts are from this crawl snapshot only.",
		},
		Compare: compare,
	}
}

func emptyThematic() Thematic {
	return Thematic{
		Crawlability:   emptyCell(),
		HTTPS:          emptyCell(),
		International:  emptyCell(),
		Performance:    emptyCell(),
		Linking:        emptyCell(),
		Markups:        emptyCell(),
		CoreWebVitals:  notImplemented("Not implemented", "Sitewide Core Web Vitals are not computed. PageSpeed is not called on this crawl."),
		AISearchHealth: notImplemented("Not implemented", "AI Search Health is not computed."),
	}
}

func withDeltas(cur Thematic, prev *Snapshot, hasPrev bool) Thematic {
	apply := func(cell ScoreCell, pick func(Thematic) ScoreCell) ScoreCell {
		if cell.Status != "ok" || cell.Value == nil || !hasPrev || prev == nil {
			return cell
		}
		p := pick(prev.Thematic)
		if p.Status != "ok" || p.Value == nil {
			return cell
		}
		cell.Delta = deltaInt(cell.Value, p.Value)
		return cell
	}
	cur.Crawlability = apply(cur.Crawlability, func(t Thematic) ScoreCell { return t.Crawlability })
	cur.HTTPS = apply(cur.HTTPS, func(t Thematic) ScoreCell { return t.HTTPS })
	cur.International = apply(cur.International, func(t Thematic) ScoreCell { return t.International })
	cur.Performance = apply(cur.Performance, func(t Thematic) ScoreCell { return t.Performance })
	cur.Linking = apply(cur.Linking, func(t Thematic) ScoreCell { return t.Linking })
	cur.Markups = apply(cur.Markups, func(t Thematic) ScoreCell { return t.Markups })
	cur.CoreWebVitals = notImplemented("Not implemented", "Sitewide Core Web Vitals are not computed.")
	cur.AISearchHealth = notImplemented("Not implemented", "AI Search Health is not computed.")
	return cur
}

// FromCrawl builds a snapshot from a real multi-page crawl.
func FromCrawl(res *services.SiteCrawlResult, pageLimit int, source string, planPages int) Snapshot {
	if planPages <= 0 {
		planPages = pageLimit
	}
	empty := Snapshot{
		PageLimit:   pageLimit,
		RunCap:      RunPageCap,
		CrawlSource: source,
		Thematic:    emptyThematic(),
		Pages:       []PageRow{},
		Issues:      []IssueRow{},
		StatusCodes: map[string]int{},
		Honesty:     DefaultHonesty(Limits{RunPageCap: RunPageCap, PagesPerAudit: planPages}, 0),
	}
	if res == nil {
		return empty
	}
	now := res.CrawledAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	snap := empty
	snap.CrawledAt = &now
	snap.SeedURL = res.SeedURL
	snap.StoppedReason = res.StoppedReason
	snap.PagesCrawled = len(res.Pages)

	allChecks := append([]services.AuditCheck{}, res.SiteChecks...)
	inlinks := map[string]int{}
	for _, p := range res.Pages {
		for _, href := range p.InternalLinks {
			inlinks[href]++
		}
	}

	statusCodes := map[string]int{}
	indexable, blocked := 0, 0
	pagesWithIn := 0
	var issues []IssueRow
	var pages []PageRow
	for _, p := range res.Pages {
		allChecks = append(allChecks, p.Checks...)
		codeKey := strconv.Itoa(p.StatusCode)
		statusCodes[codeKey]++
		if p.Indexable {
			indexable++
		} else {
			blocked++
		}
		in := inlinks[p.URL]
		if in > 0 {
			pagesWithIn++
		}
		errN, warnN, noteN := 0, 0, 0
		for _, ch := range p.Checks {
			sev := classifyIssue(ch)
			if sev == "" {
				continue
			}
			switch sev {
			case "error":
				errN++
			case "warning":
				warnN++
			default:
				noteN++
			}
			issues = append(issues, IssueRow{
				URL:            p.URL,
				Severity:       sev,
				Category:       ch.Category,
				Label:          ch.Label,
				Detail:         ch.Detail,
				Recommendation: ch.Recommendation,
			})
		}
		pages = append(pages, PageRow{
			URL:        p.URL,
			StatusCode: p.StatusCode,
			Title:      p.Title,
			Score:      p.Score,
			Errors:     errN,
			Warnings:   warnN,
			Notices:    noteN,
			Inlinks:    in,
			Outlinks:   p.Outlinks,
			Indexable:  p.Indexable,
			LoadTimeMs: p.LoadTimeMs,
		})
	}
	for _, ch := range res.SiteChecks {
		sev := classifyIssue(ch)
		if sev == "" {
			continue
		}
		issues = append(issues, IssueRow{
			URL:            res.SeedURL,
			Severity:       sev,
			Category:       ch.Category,
			Label:          ch.Label,
			Detail:         ch.Detail,
			Recommendation: ch.Recommendation,
		})
	}

	errors, warnings, notices := 0, 0, 0
	for _, iss := range issues {
		switch iss.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		default:
			notices++
		}
	}

	var health *int
	if total := len(allChecks); total > 0 {
		passed := 0
		for _, ch := range allChecks {
			if ch.Status == services.CheckPass {
				passed++
			}
		}
		v := (passed * 100) / total
		health = &v
	}

	thematic := emptyThematic()
	thematic.Crawlability = percentCell(allChecks, []string{"accessibility", "robots", "indexability", "sitemap"})
	thematic.HTTPS = percentCell(allChecks, []string{"https"})
	thematic.International = percentCell(allChecks, []string{"international"})
	thematic.Performance = percentCell(allChecks, []string{"performance", "lighthouse"})
	thematic.Markups = percentCell(allChecks, []string{"markup"})
	if snap.PagesCrawled >= 2 {
		v := (pagesWithIn * 100) / snap.PagesCrawled
		thematic.Linking = valueCell(v, nil)
	} else {
		thematic.Linking = emptyCell()
	}

	snap.SiteHealth = health
	snap.Errors = errors
	snap.Warnings = warnings
	snap.Notices = notices
	snap.Thematic = thematic
	snap.Pages = pages
	snap.Issues = issues
	snap.StatusCodes = statusCodes
	snap.Indexable = indexable
	snap.Blocked = blocked
	snap.Honesty = DefaultHonesty(Limits{RunPageCap: RunPageCap, PagesPerAudit: planPages}, snap.PagesCrawled)
	return snap
}

func percentCell(checks []services.AuditCheck, cats []string) ScoreCell {
	want := map[string]bool{}
	for _, c := range cats {
		want[c] = true
	}
	total, pass := 0, 0
	for _, ch := range checks {
		if !want[ch.Category] {
			continue
		}
		total++
		if ch.Status == services.CheckPass {
			pass++
		}
	}
	if total == 0 {
		return emptyCell()
	}
	return valueCell((pass*100)/total, nil)
}

func classifyIssue(ch services.AuditCheck) string {
	if ch.Status == services.CheckPass {
		return ""
	}
	if ch.Status == services.CheckFail && ch.Severity == "high" {
		return "error"
	}
	if ch.Status == services.CheckFail && ch.Severity == "medium" {
		return "warning"
	}
	if ch.Status == services.CheckWarning && (ch.Severity == "high" || ch.Severity == "medium") {
		return "warning"
	}
	return "notice"
}

func DefaultHonesty(limits Limits, crawled int) string {
	return "This run crawled " + strconv.Itoa(crawled) + " HTML pages (cap " +
		strconv.Itoa(RunPageCap) + " per Start because of the 30s host timeout). Plan allows up to " +
		strconv.Itoa(limits.PagesPerAudit) + " pages per audit. JS rendering is not implemented. AI Search Health is not computed. Sitewide Core Web Vitals are not computed. Auto-schedule is stored and not run."
}

// ListRow is one Site Audit campaign-list row.
type ListRow struct {
	ProjectID      uint       `json:"project_id"`
	ProjectName    string     `json:"project_name"`
	Domain         string     `json:"domain"`
	LastUpdate     *time.Time `json:"last_update"`
	PagesCrawled   *int       `json:"pages_crawled"`
	PageLimit      int        `json:"page_limit"`
	SiteHealth     ScoreCell  `json:"site_health"`
	AISearchHealth ScoreCell  `json:"ai_search_health"`
	Errors         ScoreCell  `json:"errors"`
	Warnings       ScoreCell  `json:"warnings"`
	Crawlability   ScoreCell  `json:"crawlability"`
	HTTPS          ScoreCell  `json:"https"`
	International  ScoreCell  `json:"international"`
	Performance    ScoreCell  `json:"performance"`
	Linking        ScoreCell  `json:"linking"`
	Markups        ScoreCell  `json:"markups"`
	CoreWebVitals  ScoreCell  `json:"core_web_vitals"`
}

func ListRowFrom(projectID uint, projectName, domain string, pageLimit int, snap Snapshot, prev *Snapshot) ListRow {
	hasSnap := snap.CrawledAt != nil || snap.PagesCrawled > 0
	hasPrev := prev != nil && (prev.CrawledAt != nil || prev.PagesCrawled > 0)
	row := ListRow{
		ProjectID:      projectID,
		ProjectName:    projectName,
		Domain:         domain,
		PageLimit:      pageLimit,
		AISearchHealth: notImplemented("Not implemented", "AI Search Health is not computed."),
		CoreWebVitals:  notImplemented("Not implemented", "Sitewide Core Web Vitals are not computed."),
		SiteHealth:     emptyCell(),
		Errors:         emptyCell(),
		Warnings:       emptyCell(),
		Crawlability:   emptyCell(),
		HTTPS:          emptyCell(),
		International:  emptyCell(),
		Performance:    emptyCell(),
		Linking:        emptyCell(),
		Markups:        emptyCell(),
	}
	if !hasSnap {
		return row
	}
	row.LastUpdate = snap.CrawledAt
	pc := snap.PagesCrawled
	row.PagesCrawled = &pc
	if snap.SiteHealth != nil {
		row.SiteHealth = valueCell(*snap.SiteHealth, deltaInt(snap.SiteHealth, pickHealth(prev, hasPrev)))
	}
	row.Errors = valueCell(snap.Errors, deltaCount(snap.Errors, prev, func(s *Snapshot) int { return s.Errors }, hasPrev))
	row.Warnings = valueCell(snap.Warnings, deltaCount(snap.Warnings, prev, func(s *Snapshot) int { return s.Warnings }, hasPrev))
	th := withDeltas(snap.Thematic, prev, hasPrev)
	row.Crawlability = th.Crawlability
	row.HTTPS = th.HTTPS
	row.International = th.International
	row.Performance = th.Performance
	row.Linking = th.Linking
	row.Markups = th.Markups
	return row
}

func pickHealth(prev *Snapshot, hasPrev bool) *int {
	if !hasPrev || prev == nil {
		return nil
	}
	return prev.SiteHealth
}
