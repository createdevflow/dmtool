package domainoverview

import (
	"sort"
	"strconv"

	"backend/internal/services"
)

// Cell is one metric. Empty stays "empty"; we never invent a score.
type Cell struct {
	Value  *float64 `json:"value,omitempty"`
	Status string   `json:"status"`
	Label  string   `json:"label,omitempty"`
	Hint   string   `json:"hint,omitempty"`
}

// Reports is the Semrush Domain Overview payload. Unimplemented tiles
// stay status=not_implemented. Vendor zeros are shown only when that
// DataForSEO section succeeded.
type Reports struct {
	Overview         OverviewMetrics `json:"overview"`
	TopKeywords      []KeywordRow    `json:"top_keywords"`
	Intent           []IntentRow     `json:"intent"`
	Positions        []Bucket        `json:"positions"`
	Competitors      []CompetitorRow `json:"competitors"`
	CompetitiveMap   []MapPoint      `json:"competitive_map"`
	BacklinkSplit    BacklinkSplit   `json:"backlink_split"`
	LinkTypes        []Bucket        `json:"link_types"`
	Growth           LockedSection   `json:"growth"`
	CompareCountries LockedSection   `json:"compare_countries"`
	Country          LockedSection   `json:"distribution_by_country"`
	Branded          LockedSection   `json:"branded"`
	SERPFeatures     LockedSection   `json:"serp_features"`
	TrafficTrend     LockedSection   `json:"traffic_trend"`
	KeyTopics        LockedSection   `json:"key_topics"`
	PaidKeywords     LockedSection   `json:"paid_keywords"`
	PaidPositions    LockedSection   `json:"paid_positions"`
	PaidCompetitors  LockedSection   `json:"paid_competitors"`
	SampleAds        LockedSection   `json:"sample_ads"`
	TopAnchors       LockedSection   `json:"top_anchors"`
	ReferringList    LockedSection   `json:"referring_domains_list"`
	IndexedPages     LockedSection   `json:"indexed_pages"`
	LocationName     string          `json:"location_name"`
	LanguageCode     string          `json:"language_code"`
	Honesty          string          `json:"honesty"`
}

type OverviewMetrics struct {
	AIVisibility     Cell `json:"ai_visibility"`
	AuthorityScore   Cell `json:"authority_score"`
	OrganicTraffic   Cell `json:"organic_traffic"`
	PaidTraffic      Cell `json:"paid_traffic"`
	ReferringDomains Cell `json:"referring_domains"`
	TrafficShare     Cell `json:"traffic_share"`
	OrganicKeywords  Cell `json:"organic_keywords"`
	PaidKeywords     Cell `json:"paid_keywords"`
	Backlinks        Cell `json:"backlinks"`
	DFSRank          Cell `json:"dfs_rank"`
}

type KeywordRow struct {
	Keyword  string  `json:"keyword"`
	Volume   int     `json:"volume"`
	CPC      float64 `json:"cpc"`
	KD       int     `json:"kd"`
	Position int     `json:"position"`
	Intent   string  `json:"intent"`
}

type IntentRow struct {
	Intent string `json:"intent"`
	Count  int    `json:"count"`
}

type Bucket struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type CompetitorRow struct {
	Domain        string  `json:"domain"`
	Intersections int     `json:"intersections"`
	OrganicETV    float64 `json:"organic_etv"`
	OrganicCount  int     `json:"organic_count"`
}

type MapPoint struct {
	Domain string  `json:"domain"`
	ETV    float64 `json:"etv"`
	Count  int     `json:"count"`
	Self   bool    `json:"self"`
}

type BacklinkSplit struct {
	Follow   int  `json:"follow"`
	Nofollow int  `json:"nofollow"`
	OK       bool `json:"ok"`
}

type LockedSection struct {
	Locked       bool   `json:"locked"`
	Status       string `json:"status"`
	Label        string `json:"label"`
	RequiredPlan string `json:"required_plan,omitempty"`
}

func notImplemented(hint string) Cell {
	return Cell{Status: "not_implemented", Label: "Not implemented", Hint: hint}
}

func emptyCell() Cell {
	return Cell{Status: "empty"}
}

func numCell(v float64, hint string) Cell {
	x := v
	return Cell{Value: &x, Status: "ok", Hint: hint}
}

func locked(on bool, required string) LockedSection {
	if on {
		return LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"}
	}
	return LockedSection{Locked: true, Status: "locked", Label: "Upgrade to unlock", RequiredPlan: required}
}

// BuildReports maps a Domain Explorer snapshot onto Domain Overview widgets.
// available is false when there is no cached/live payload.
func BuildReports(limits Limits, result services.DomainExplorerResult, available bool) Reports {
	out := Reports{
		Overview: OverviewMetrics{
			AIVisibility:   notImplemented("AI Visibility Score is not computed in DMTool."),
			AuthorityScore: notImplemented("Semrush Authority Score is not computed. DataForSEO Rank is shown separately when backlinks succeed."),
		},
		TopKeywords:      []KeywordRow{},
		Intent:           []IntentRow{},
		Positions:        []Bucket{},
		Competitors:      []CompetitorRow{},
		CompetitiveMap:   []MapPoint{},
		LinkTypes:        []Bucket{},
		Growth:           locked(limits.Growth, "Pro+"),
		CompareCountries: locked(limits.CompareCountries, "Pro+"),
		Country:          LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented", RequiredPlan: ""},
		Branded:          LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		SERPFeatures:     LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		TrafficTrend:     locked(limits.Historical, "Pro+"),
		KeyTopics:        LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		PaidKeywords:     LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		PaidPositions:    LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		PaidCompetitors:  LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		SampleAds:        LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		TopAnchors:       LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		ReferringList:    LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		IndexedPages:     LockedSection{Locked: false, Status: "not_implemented", Label: "Not implemented"},
		LocationName:     result.LocationName,
		LanguageCode:     result.LanguageCode,
		Honesty:          DefaultHonesty(limits),
	}
	if out.LocationName == "" {
		out.LocationName = "United States"
	}
	if out.LanguageCode == "" {
		out.LanguageCode = "en"
	}

	out.Overview.OrganicTraffic = emptyCell()
	out.Overview.PaidTraffic = emptyCell()
	out.Overview.OrganicKeywords = emptyCell()
	out.Overview.PaidKeywords = emptyCell()
	out.Overview.ReferringDomains = emptyCell()
	out.Overview.Backlinks = emptyCell()
	out.Overview.TrafficShare = emptyCell()
	out.Overview.DFSRank = emptyCell()

	if !available {
		return out
	}

	if result.OrganicOK {
		out.Overview.OrganicTraffic = numCell(result.Organic.ETV, "Labs ETV / month · Google US · not GSC")
		out.Overview.OrganicKeywords = numCell(float64(result.Organic.Count), "Keywords in Google US top 100")
		out.Overview.PaidTraffic = numCell(result.Organic.PaidETV, "Labs paid ETV / month")
		out.Overview.PaidKeywords = numCell(float64(result.Organic.PaidCount), "Paid keyword count from domain_rank_overview")
		out.Positions = []Bucket{
			{Key: "1", Count: result.Organic.Pos1},
			{Key: "2-3", Count: result.Organic.Pos2To3},
			{Key: "4-10", Count: result.Organic.Pos4To10},
		}
		rest := result.Organic.Count - result.Organic.Pos1 - result.Organic.Pos2To3 - result.Organic.Pos4To10
		if rest > 0 {
			out.Positions = append(out.Positions, Bucket{Key: "11+", Count: rest})
		}
	}

	if result.BacklinksOK {
		out.Overview.Backlinks = numCell(float64(result.Backlinks.Backlinks), "DataForSEO Backlinks index")
		out.Overview.ReferringDomains = numCell(float64(result.Backlinks.ReferringDomains), "Not Moz DA")
		if result.Backlinks.Rank > 0 {
			out.Overview.DFSRank = numCell(float64(result.Backlinks.Rank), "DataForSEO Rank, not Moz DA and not Semrush Authority Score")
		}
		if result.Backlinks.Follow+result.Backlinks.Nofollow > 0 {
			out.BacklinkSplit = BacklinkSplit{
				Follow:   result.Backlinks.Follow,
				Nofollow: result.Backlinks.Nofollow,
				OK:       true,
			}
		}
		for k, v := range result.Backlinks.Types {
			out.LinkTypes = append(out.LinkTypes, Bucket{Key: k, Count: v})
		}
		sort.Slice(out.LinkTypes, func(i, j int) bool { return out.LinkTypes[i].Count > out.LinkTypes[j].Count })
	}

	if result.KeywordsOK {
		for i, k := range result.Keywords {
			if i >= 5 {
				break
			}
			out.TopKeywords = append(out.TopKeywords, KeywordRow{
				Keyword:  k.Keyword,
				Volume:   k.Volume,
				CPC:      k.CPC,
				KD:       k.KD,
				Position: k.Position,
				Intent:   k.Intent,
			})
		}
		intentCounts := map[string]int{}
		order := []string{}
		for _, k := range result.Keywords {
			if k.Intent == "" {
				continue
			}
			if _, ok := intentCounts[k.Intent]; !ok {
				order = append(order, k.Intent)
			}
			intentCounts[k.Intent]++
		}
		for _, name := range order {
			out.Intent = append(out.Intent, IntentRow{Intent: name, Count: intentCounts[name]})
		}
	}

	if result.CompetitorsOK {
		sum := 0.0
		if result.OrganicOK {
			sum = result.Organic.ETV
			out.CompetitiveMap = append(out.CompetitiveMap, MapPoint{
				Domain: result.Domain,
				ETV:    result.Organic.ETV,
				Count:  result.Organic.Count,
				Self:   true,
			})
		}
		for _, c := range result.Competitors {
			out.Competitors = append(out.Competitors, CompetitorRow{
				Domain:        c.Domain,
				Intersections: c.Intersections,
				OrganicETV:    c.OrganicETV,
				OrganicCount:  c.OrganicCount,
			})
			sum += c.OrganicETV
			out.CompetitiveMap = append(out.CompetitiveMap, MapPoint{
				Domain: c.Domain,
				ETV:    c.OrganicETV,
				Count:  c.OrganicCount,
			})
		}
		if result.OrganicOK && sum > 0 {
			out.Overview.TrafficShare = numCell(result.Organic.ETV/sum*100, "Share vs competitors in this lookup, not a market share")
		}
	}

	return out
}

// DefaultHonesty is the always-visible source line.
func DefaultHonesty(limits Limits) string {
	return "Estimated from DataForSEO Labs Google United States. Not Search Console. Authority Score and AI Visibility stay Not implemented. Empty cells stay —. " +
		limits.PlanLabel + " plan: " + strconv.Itoa(limits.RequestsPerDay) + " Domain Overview searches per day."
}
