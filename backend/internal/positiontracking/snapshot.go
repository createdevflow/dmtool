package positiontracking

import (
	"strings"
	"time"

	"backend/internal/models"
)

// RefreshKeywordLimit is the max DataForSEO SERP tasks per refresh (Render 30s).
const RefreshKeywordLimit = 10

// Rank is one domain's position on a SERP.
type Rank struct {
	Domain   string `json:"domain"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Type     string `json:"type"`
}

// KeywordRow is one tracked keyword in a snapshot.
type KeywordRow struct {
	Keyword         string   `json:"keyword"`
	Tags            []string `json:"tags,omitempty"`
	Position        int      `json:"position"`
	URL             string   `json:"url,omitempty"`
	Title           string   `json:"title,omitempty"`
	SERPFeatures    []string `json:"serp_features"`
	Competitors     []Rank   `json:"competitors"`
	Cannibalizing   []Rank   `json:"cannibalizing"`
	Discovered      []Rank   `json:"discovered"`
	FeaturedSnippet *Rank    `json:"featured_snippet,omitempty"`
	GSCPosition     float64  `json:"gsc_position,omitempty"`
	GSCImpressions  int      `json:"gsc_impressions,omitempty"`
	GSCClicks       int      `json:"gsc_clicks,omitempty"`
	Source          string   `json:"source"`
	Error           string   `json:"error,omitempty"`
	OK              bool     `json:"ok"`
}

// Snapshot is the latest lookup for a campaign. History is not archived.
type Snapshot struct {
	Source       string       `json:"source"`
	Engine       string       `json:"engine"`
	Device       string       `json:"device"`
	LocationName string       `json:"location_name"`
	LocationCode int          `json:"location_code"`
	LanguageCode string       `json:"language_code"`
	FetchedAt    *time.Time   `json:"fetched_at,omitempty"`
	Keywords     []KeywordRow `json:"keywords"`
	Error        string       `json:"error,omitempty"`
}

// Gate describes a plan-locked report.
type Gate struct {
	Locked     bool   `json:"locked"`
	PlanNeeded string `json:"plan_needed,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// PageRow aggregates landing pages from the snapshot.
type PageRow struct {
	URL      string `json:"url"`
	Keywords int    `json:"keywords"`
	BestPos  int    `json:"best_position"`
}

// TagRow aggregates keywords by tag.
type TagRow struct {
	Tag      string `json:"tag"`
	Keywords int    `json:"keywords"`
	Ranked   int    `json:"ranked"`
}

// LandscapeCell is one competitor's rank on one keyword.
type LandscapeCell struct {
	Keyword   string         `json:"keyword"`
	TargetPos int            `json:"target_position"`
	ByDomain  map[string]int `json:"by_domain"`
}

// CannibalRow is a keyword with more than one ranking URL.
type CannibalRow struct {
	Keyword string `json:"keyword"`
	URLs    []Rank `json:"urls"`
}

// SnippetRow is a featured snippet observation.
type SnippetRow struct {
	Keyword  string `json:"keyword"`
	Owner    string `json:"owner"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	IsTarget bool   `json:"is_target"`
}

// Reports is the Semrush-style Position Tracking report pack.
type Reports struct {
	Overview             OverviewReport   `json:"overview"`
	Rankings             []KeywordRow     `json:"rankings"`
	Distribution         map[string]int   `json:"distribution"`
	Pages                []PageRow        `json:"pages"`
	Tags                 TagReport        `json:"tags"`
	DevicesLocations     DevicesReport    `json:"devices_locations"`
	Landscape            []LandscapeCell  `json:"landscape"`
	CompetitorsDiscovery DiscoveryReport  `json:"competitors_discovery"`
	Cannibalization      CannibalReport   `json:"cannibalization"`
	FeaturedSnippets     []SnippetRow     `json:"featured_snippets"`
	AISearch             Gate             `json:"ai_search"`
	ShareOfVoice         SOVReport        `json:"share_of_voice"`
	Historical           HistoricalReport `json:"historical"`
	ScheduledExport      Gate             `json:"scheduled_export"`
}

// OverviewReport is the Overview tab.
type OverviewReport struct {
	Visibility  *float64       `json:"visibility"`
	AvgPosition *float64       `json:"avg_position"`
	ETV         *float64       `json:"etv"`
	Ranked      int            `json:"ranked"`
	Tracked     int            `json:"tracked"`
	Buckets     map[string]int `json:"buckets"`
	Source      string         `json:"source"`
	Honesty     string         `json:"honesty"`
}

// TagReport is the Tags tab (Guru+).
type TagReport struct {
	Gate
	Rows []TagRow `json:"rows"`
}

// DevicesReport is Devices & Locations (Guru+).
type DevicesReport struct {
	Gate
	Targets []models.PTTarget `json:"targets"`
	Note    string            `json:"note,omitempty"`
}

// DiscoveryReport lists domains seen in the current SERP snapshot.
type DiscoveryReport struct {
	Domains []string `json:"domains"`
	Honesty string   `json:"honesty"`
}

// CannibalReport is Cannibalization (Guru+).
type CannibalReport struct {
	Gate
	Rows []CannibalRow `json:"rows"`
}

// SOVReport is Share of Voice (Advanced). Keyword share of this snapshot.
type SOVReport struct {
	Gate
	Shares  map[string]float64 `json:"shares"`
	Honesty string             `json:"honesty,omitempty"`
}

// HistoricalReport is history (Guru+). We store only the latest lookup.
type HistoricalReport struct {
	Gate
	Points  []HistoricalPoint `json:"points"`
	Honesty string            `json:"honesty,omitempty"`
}

// HistoricalPoint is one stored snapshot time.
type HistoricalPoint struct {
	At         *time.Time `json:"at,omitempty"`
	Visibility *float64   `json:"visibility"`
	Ranked     int        `json:"ranked"`
	Source     string     `json:"source"`
}

// KeywordUnits counts keywords × targets the way Semrush bills extra locations.
func KeywordUnits(keywordCount, targetCount int, multitarget bool) int {
	n := targetCount
	if n < 1 || !multitarget {
		n = 1
	}
	return keywordCount * n
}

// ApplyPlan strips features the plan does not include.
func ApplyPlan(limits Limits, keywords []models.PTKeyword, competitors []string, targets []models.PTTarget) ([]models.PTKeyword, []string, []models.PTTarget) {
	outK := make([]models.PTKeyword, 0, len(keywords))
	seen := map[string]bool{}
	for _, kw := range keywords {
		k := strings.ToLower(strings.TrimSpace(kw.Keyword))
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		row := models.PTKeyword{Keyword: k}
		if limits.Tags {
			row.Tags = cleanTags(kw.Tags)
		}
		outK = append(outK, row)
	}
	outC := uniqueDomains(competitors, limits.CompetitorsPerCampaign)
	outT := []models.PTTarget{}
	if limits.Multitargeting {
		for _, t := range targets {
			if len(outT) >= limits.MaxTargets {
				break
			}
			outT = append(outT, normalizeTarget(t))
		}
	}
	return outK, outC, outT
}

func cleanTags(tags []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range tags {
		s := strings.ToLower(strings.TrimSpace(t))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func uniqueDomains(in []string, limit int) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, d := range in {
		s := strings.ToLower(strings.TrimSpace(d))
		s = strings.TrimPrefix(s, "http://")
		s = strings.TrimPrefix(s, "https://")
		s = strings.TrimPrefix(s, "www.")
		if i := strings.IndexAny(s, "/?#"); i >= 0 {
			s = s[:i]
		}
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func normalizeTarget(t models.PTTarget) models.PTTarget {
	dev := strings.ToLower(strings.TrimSpace(t.Device))
	if dev != "mobile" {
		dev = "desktop"
	}
	eng := strings.ToLower(strings.TrimSpace(t.Engine))
	if eng == "" {
		eng = "google"
	}
	lang := strings.TrimSpace(t.LanguageCode)
	if lang == "" {
		lang = "en"
	}
	name := strings.TrimSpace(t.LocationName)
	if name == "" {
		name = "United States"
	}
	code := t.LocationCode
	if code <= 0 {
		code = 2840
	}
	return models.PTTarget{
		LocationName: name,
		LocationCode: code,
		LanguageCode: lang,
		Device:       dev,
		Engine:       eng,
	}
}

// BuildReports computes Semrush-style reports from the current snapshot only.
func BuildReports(limits Limits, snap Snapshot, targetDomain string, trackedCompetitors []string, campaignTargets []models.PTTarget) Reports {
	if snap.Keywords == nil {
		snap.Keywords = []KeywordRow{}
	}
	rankedPos := []int{}
	top3, top10, top30, beyond := 0, 0, 0, 0
	for _, kw := range snap.Keywords {
		if kw.Position <= 0 {
			continue
		}
		rankedPos = append(rankedPos, kw.Position)
		switch {
		case kw.Position <= 3:
			top3++
		case kw.Position <= 10:
			top10++
		case kw.Position <= 30:
			top30++
		default:
			beyond++
		}
	}
	buckets := map[string]int{"top3": top3, "top10": top10, "top30": top30, "beyond": beyond}
	var vis *float64
	var avg *float64
	if len(rankedPos) > 0 {
		sumVis := 0.0
		sumPos := 0.0
		for _, p := range rankedPos {
			s := 100.0 - float64(p)
			if s < 0 {
				s = 0
			}
			sumVis += s
			sumPos += float64(p)
		}
		v := sumVis / float64(len(rankedPos))
		a := sumPos / float64(len(rankedPos))
		vis = &v
		avg = &a
	}

	rep := Reports{
		Overview: OverviewReport{
			Visibility:  vis,
			AvgPosition: avg,
			ETV:         nil,
			Ranked:      len(rankedPos),
			Tracked:     len(snap.Keywords),
			Buckets:     buckets,
			Source:      snap.Source,
			Honesty:     "Visibility is the average of max(0, 100 − position) over ranked keywords in this snapshot. Estimated traffic is not computed.",
		},
		Rankings:     snap.Keywords,
		Distribution: buckets,
		Pages:        buildPages(snap.Keywords),
		Tags: TagReport{
			Gate: gate(limits.Tags, "Pro+"),
			Rows: buildTags(snap.Keywords, limits.Tags),
		},
		DevicesLocations: DevicesReport{
			Gate:    gate(limits.DevicesLocations, "Pro+"),
			Targets: campaignTargets,
			Note:    "Each extra location, device, or engine counts toward the keyword cap. Refresh queries the primary target only (max 10 keywords).",
		},
		Landscape: buildLandscape(snap.Keywords, trackedCompetitors),
		CompetitorsDiscovery: DiscoveryReport{
			Domains: buildDiscovered(snap.Keywords, targetDomain),
			Honesty: "Domains observed in this SERP snapshot. Not a Semrush Competitors Discovery crawl of the whole market.",
		},
		Cannibalization: CannibalReport{
			Gate: gate(limits.Cannibalization, "Pro+"),
			Rows: buildCannibal(snap.Keywords, limits.Cannibalization),
		},
		FeaturedSnippets: buildSnippets(snap.Keywords, targetDomain),
		AISearch: Gate{
			Locked:     !limits.AISearch,
			PlanNeeded: "Pro+",
			Reason:     "ChatGPT, Baidu, and Google AI Mode prompt tracking are not connected. Google AI Overviews and ChatGPT mention counts live under AI Visibility when DataForSEO is queried.",
		},
		ShareOfVoice: SOVReport{
			Gate:    gate(limits.ShareOfVoice, "Advanced"),
			Shares:  buildSOV(snap.Keywords, targetDomain, trackedCompetitors, limits.ShareOfVoice),
			Honesty: "Keyword share of this snapshot among tracked domains. Not Semrush estimated-traffic share of voice.",
		},
		Historical: HistoricalReport{
			Gate: gate(limits.Historical, "Pro+"),
			Points: func() []HistoricalPoint {
				if !limits.Historical || snap.FetchedAt == nil {
					return []HistoricalPoint{}
				}
				return []HistoricalPoint{{At: snap.FetchedAt, Visibility: vis, Ranked: len(rankedPos), Source: snap.Source}}
			}(),
			Honesty: "History starts at campaign creation. Only the latest lookup is stored; daily archive is not kept yet.",
		},
		ScheduledExport: gate(limits.ScheduledExport, "Advanced"),
	}
	if !limits.DevicesLocations {
		rep.DevicesLocations.Targets = []models.PTTarget{}
	}
	return rep
}

func gate(allowed bool, needed string) Gate {
	if allowed {
		return Gate{Locked: false}
	}
	return Gate{Locked: true, PlanNeeded: needed, Reason: "Included on " + needed + " and above."}
}

func buildPages(rows []KeywordRow) []PageRow {
	byURL := map[string]*PageRow{}
	order := []string{}
	for _, kw := range rows {
		if kw.URL == "" || kw.Position <= 0 {
			continue
		}
		p, ok := byURL[kw.URL]
		if !ok {
			p = &PageRow{URL: kw.URL, BestPos: kw.Position}
			byURL[kw.URL] = p
			order = append(order, kw.URL)
		}
		p.Keywords++
		if kw.Position > 0 && (p.BestPos <= 0 || kw.Position < p.BestPos) {
			p.BestPos = kw.Position
		}
	}
	out := make([]PageRow, 0, len(order))
	for _, u := range order {
		out = append(out, *byURL[u])
	}
	return out
}

func buildTags(rows []KeywordRow, allowed bool) []TagRow {
	if !allowed {
		return []TagRow{}
	}
	byTag := map[string]*TagRow{}
	order := []string{}
	for _, kw := range rows {
		for _, t := range kw.Tags {
			r, ok := byTag[t]
			if !ok {
				r = &TagRow{Tag: t}
				byTag[t] = r
				order = append(order, t)
			}
			r.Keywords++
			if kw.Position > 0 {
				r.Ranked++
			}
		}
	}
	out := make([]TagRow, 0, len(order))
	for _, t := range order {
		out = append(out, *byTag[t])
	}
	return out
}

func buildLandscape(rows []KeywordRow, competitors []string) []LandscapeCell {
	out := make([]LandscapeCell, 0, len(rows))
	for _, kw := range rows {
		cell := LandscapeCell{Keyword: kw.Keyword, TargetPos: kw.Position, ByDomain: map[string]int{}}
		for _, c := range competitors {
			cell.ByDomain[c] = 0
		}
		for _, r := range kw.Competitors {
			if r.Position > 0 {
				cell.ByDomain[r.Domain] = r.Position
			}
		}
		out = append(out, cell)
	}
	return out
}

func buildDiscovered(rows []KeywordRow, target string) []string {
	seen := map[string]bool{}
	out := []string{}
	target = strings.ToLower(strings.TrimSpace(target))
	for _, kw := range rows {
		for _, r := range kw.Discovered {
			d := strings.ToLower(r.Domain)
			if d == "" || d == target || seen[d] {
				continue
			}
			seen[d] = true
			out = append(out, d)
			if len(out) >= 50 {
				return out
			}
		}
	}
	return out
}

func buildCannibal(rows []KeywordRow, allowed bool) []CannibalRow {
	if !allowed {
		return []CannibalRow{}
	}
	out := []CannibalRow{}
	for _, kw := range rows {
		if len(kw.Cannibalizing) == 0 {
			continue
		}
		urls := []Rank{}
		if kw.URL != "" {
			urls = append(urls, Rank{URL: kw.URL, Position: kw.Position, Title: kw.Title})
		}
		urls = append(urls, kw.Cannibalizing...)
		out = append(out, CannibalRow{Keyword: kw.Keyword, URLs: urls})
	}
	return out
}

func buildSnippets(rows []KeywordRow, target string) []SnippetRow {
	target = strings.ToLower(strings.TrimSpace(target))
	out := []SnippetRow{}
	for _, kw := range rows {
		if kw.FeaturedSnippet == nil {
			continue
		}
		s := kw.FeaturedSnippet
		out = append(out, SnippetRow{
			Keyword:  kw.Keyword,
			Owner:    s.Domain,
			URL:      s.URL,
			Title:    s.Title,
			IsTarget: strings.ToLower(s.Domain) == target,
		})
	}
	return out
}

func buildSOV(rows []KeywordRow, target string, competitors []string, allowed bool) map[string]float64 {
	shares := map[string]float64{}
	if !allowed {
		return shares
	}
	counts := map[string]int{}
	counts[target] = 0
	for _, c := range competitors {
		counts[c] = 0
	}
	total := 0
	for _, kw := range rows {
		if kw.Position > 0 && target != "" {
			counts[target]++
			total++
		}
		for _, r := range kw.Competitors {
			if r.Position > 0 {
				counts[r.Domain]++
				total++
			}
		}
	}
	if total == 0 {
		return shares
	}
	for d, n := range counts {
		if d == "" {
			continue
		}
		shares[d] = float64(n) / float64(total)
	}
	return shares
}
