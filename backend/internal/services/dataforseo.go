package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/internal/models"
)

const (
	dataForSEOBaseURL      = "https://api.dataforseo.com"
	DataForSEOLocation     = 2840 // United States
	DataForSEOLocationName = "United States"
	DataForSEOLanguage     = "en"
	rankedKeywordLimit     = 15
	competitorLimit        = 8
	keywordOverviewLimit   = 30
	backlinkGapLimit       = 15
	referringDomainsLimit  = 15
)

// ErrDataForSEONotConfigured is returned when login/password are empty.
var ErrDataForSEONotConfigured = errors.New("dataforseo is not connected")

type DataForSEOService interface {
	Configured() bool
	FetchEstimatedTraffic(siteURL string) ([]models.Metric, error)
	DomainExplorer(ctx context.Context, domain string) (DomainExplorerResult, error)
	AIVisibility(ctx context.Context, domain string) (AIVisibilityResult, error)
	KeywordOverview(ctx context.Context, keywords []string) (map[string]KeywordLabsMetrics, error)
	RelatedKeywords(ctx context.Context, seed string) (RelatedKeywordsResult, error)
	KeywordGap(ctx context.Context, target, competitor string) (KeywordGapResult, error)
	DomainCompare(ctx context.Context, target, competitor string) (DomainCompareResult, error)
	ContentGap(ctx context.Context, target, competitor string) (ContentGapResult, error)
	CitationGap(ctx context.Context, target, competitor string) (CitationGapResult, error)
	BacklinkGap(ctx context.Context, target, competitor string) (BacklinkGapResult, error)
	BacklinkExplorer(ctx context.Context, domain string) (BacklinkExplorerResult, error)
	AIVisibilityGap(ctx context.Context, target, competitor string) (AIVisibilityGapResult, error)
}

type dataForSEO struct {
	login    string
	password string
	client   *http.Client
	baseURL  string
}

func NewDataForSEOService(login, password string) DataForSEOService {
	return &dataForSEO{
		login:    strings.TrimSpace(login),
		password: strings.TrimSpace(password),
		client:   &http.Client{Timeout: 90 * time.Second},
		baseURL:  dataForSEOBaseURL,
	}
}

func (s *dataForSEO) Configured() bool {
	return s != nil && s.login != "" && s.password != ""
}

func (s *dataForSEO) FetchEstimatedTraffic(siteURL string) ([]models.Metric, error) {
	return nil, fmt.Errorf("dataforseo traffic is not written to dashboard metrics")
}

// DomainExplorer pulls Labs overview, organic competitors, ranked keywords, and
// backlink summary for any domain. Sections fail independently — zeros are not
// invented when a call errors.
func (s *dataForSEO) DomainExplorer(ctx context.Context, domain string) (DomainExplorerResult, error) {
	out := DomainExplorerResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(domain)
	if host == "" {
		return out, fmt.Errorf("invalid domain")
	}
	out.Domain = host

	if err := s.fillOrganicOverview(ctx, host, &out); err != nil {
		out.OrganicError = err.Error()
	} else {
		out.OrganicOK = true
	}
	if err := s.fillCompetitors(ctx, host, &out); err != nil {
		out.CompetitorsError = err.Error()
	} else {
		out.CompetitorsOK = true
	}
	if err := s.fillRankedKeywords(ctx, host, &out); err != nil {
		out.KeywordsError = err.Error()
	} else {
		out.KeywordsOK = true
	}
	if err := s.fillBacklinks(ctx, host, &out); err != nil {
		out.BacklinksError = err.Error()
	} else {
		out.BacklinksOK = true
	}
	return out, nil
}

// AIVisibility pulls LLM Mentions target metrics for a domain (Google AI
// Overviews + ChatGPT, United States / English). Mentions (any) and
// citations (sources) fail independently — zeros are not invented on error.
// Results are not written to dashboard metrics.
func (s *dataForSEO) AIVisibility(ctx context.Context, domain string) (AIVisibilityResult, error) {
	out := AIVisibilityResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(domain)
	if host == "" {
		return out, fmt.Errorf("invalid domain")
	}
	out.Domain = host

	if err := s.fillLLMTargetMetrics(ctx, host, "any", &out.Mentions); err != nil {
		out.MentionsError = err.Error()
	} else {
		out.MentionsOK = true
	}
	if err := s.fillLLMTargetMetrics(ctx, host, "sources", &out.Citations); err != nil {
		out.CitationsError = err.Error()
	} else {
		out.CitationsOK = true
	}
	return out, nil
}

// KeywordOverview returns Labs search volume, CPC, KD, competition, SERP
// features, and monthly search trend for the given keywords (Google
// United States). Missing keywords are omitted — zeros are not invented.
// Results are not written to dashboard metrics.
func (s *dataForSEO) KeywordOverview(ctx context.Context, keywords []string) (map[string]KeywordLabsMetrics, error) {
	out := map[string]KeywordLabsMetrics{}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	cleaned := uniqueKeywordList(keywords, keywordOverviewLimit)
	if len(cleaned) == 0 {
		return out, nil
	}
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/keyword_overview/live", map[string]any{
		"keywords":          cleaned,
		"location_code":     DataForSEOLocation,
		"language_code":     DataForSEOLanguage,
		"include_serp_info": true,
	})
	if err != nil {
		return out, err
	}
	var wrap []struct {
		Items []keywordOverviewRow `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return out, err
	}
	if len(wrap) == 0 {
		return out, nil
	}
	for _, item := range wrap[0].Items {
		kw := strings.ToLower(strings.TrimSpace(item.Keyword))
		if kw == "" {
			continue
		}
		kd := item.KeywordProperties.KeywordDifficulty
		if kd == 0 {
			kd = item.KeywordInfo.KeywordDifficulty
		}
		out[kw] = KeywordLabsMetrics{
			SearchVolume:     item.KeywordInfo.SearchVolume,
			CPC:              item.KeywordInfo.CPC,
			KD:               kd,
			Competition:      item.KeywordInfo.Competition,
			CompetitionLevel: strings.ToUpper(strings.TrimSpace(item.KeywordInfo.CompetitionLevel)),
			LabsIntent:       strings.ToLower(strings.TrimSpace(item.SearchIntentInfo.MainIntent)),
			SerpFeatures:     cleanSerpFeatures(item.SerpInfo.SerpItemTypes),
			MonthlySearches:  copyMonthlySearches(item.KeywordInfo.MonthlySearches),
			TrendMonthly:     item.KeywordInfo.SearchVolumeTrend.Monthly,
		}
	}
	return out, nil
}

// RelatedKeywords returns Labs "searches related to" the seed (Google US).
// Empty related is valid. Results are not written to dashboard metrics.
func (s *dataForSEO) RelatedKeywords(ctx context.Context, seed string) (RelatedKeywordsResult, error) {
	out := RelatedKeywordsResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
		Related:      []RelatedKeyword{},
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	cleaned := uniqueKeywordList([]string{seed}, 1)
	if len(cleaned) == 0 {
		return out, fmt.Errorf("invalid keyword")
	}
	out.Seed = cleaned[0]
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/related_keywords/live", map[string]any{
		"keyword":           out.Seed,
		"location_code":     DataForSEOLocation,
		"language_code":     DataForSEOLanguage,
		"depth":             1,
		"limit":             rankedKeywordLimit,
		"include_serp_info": false,
	})
	if err != nil {
		out.RelatedError = err.Error()
		return out, nil
	}
	var wrap []struct {
		Items []struct {
			KeywordData relatedKeywordData `json:"keyword_data"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		out.RelatedError = err.Error()
		return out, nil
	}
	out.RelatedOK = true
	if len(wrap) == 0 {
		return out, nil
	}
	for _, item := range wrap[0].Items {
		kw := strings.ToLower(strings.TrimSpace(item.KeywordData.Keyword))
		if kw == "" || kw == out.Seed {
			continue
		}
		kd := item.KeywordData.KeywordProperties.KeywordDifficulty
		if kd == 0 {
			kd = item.KeywordData.KeywordInfo.KeywordDifficulty
		}
		out.Related = append(out.Related, RelatedKeyword{
			Keyword:          kw,
			SearchVolume:     item.KeywordData.KeywordInfo.SearchVolume,
			CPC:              item.KeywordData.KeywordInfo.CPC,
			KD:               kd,
			CompetitionLevel: strings.ToUpper(strings.TrimSpace(item.KeywordData.KeywordInfo.CompetitionLevel)),
		})
	}
	return out, nil
}

func cleanSerpFeatures(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, v := range raw {
		s := strings.ToLower(strings.TrimSpace(v))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func copyMonthlySearches(raw []monthlySearchRow) []MonthlySearch {
	out := make([]MonthlySearch, 0, len(raw))
	for _, row := range raw {
		if row.Year == 0 || row.Month < 1 || row.Month > 12 {
			continue
		}
		out = append(out, MonthlySearch{
			Year:         row.Year,
			Month:        row.Month,
			SearchVolume: row.SearchVolume,
		})
	}
	return out
}

// KeywordGap compares two domains via Labs domain_intersection (Google US).
// Missing = competitor ranks and target does not. Overlap = both rank.
// Sections fail independently. Results are not written to dashboard metrics.
func (s *dataForSEO) KeywordGap(ctx context.Context, target, competitor string) (KeywordGapResult, error) {
	out := KeywordGapResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
		Missing:      []GapKeyword{},
		Overlap:      []GapKeyword{},
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(target)
	rival := NormalizeDomain(competitor)
	if host == "" || rival == "" {
		return out, fmt.Errorf("invalid domain")
	}
	if host == rival {
		return out, fmt.Errorf("competitor must be a different domain")
	}
	out.Target = host
	out.Competitor = rival

	if err := s.fillKeywordGapMissing(ctx, host, rival, &out); err != nil {
		out.MissingError = err.Error()
	} else {
		out.MissingOK = true
	}
	if err := s.fillKeywordGapOverlap(ctx, host, rival, &out); err != nil {
		out.OverlapError = err.Error()
	} else {
		out.OverlapOK = true
	}
	return out, nil
}

func (s *dataForSEO) fillKeywordGapMissing(ctx context.Context, host, rival string, out *KeywordGapResult) error {
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/domain_intersection/live", map[string]any{
		"target1":       rival,
		"target2":       host,
		"location_code": DataForSEOLocation,
		"language_code": DataForSEOLanguage,
		"intersections": false,
		"item_types":    []string{"organic"},
		"limit":         rankedKeywordLimit,
		"order_by":      []string{"keyword_data.keyword_info.search_volume,desc"},
	})
	if err != nil {
		return err
	}
	rows, err := parseGapItems(raw)
	if err != nil {
		return err
	}
	out.Missing = make([]GapKeyword, 0, len(rows))
	for _, row := range rows {
		if row.Keyword == "" {
			continue
		}
		out.Missing = append(out.Missing, GapKeyword{
			Keyword:            row.Keyword,
			Volume:             row.Volume,
			CPC:                row.CPC,
			KD:                 row.KD,
			TargetPosition:     0,
			CompetitorPosition: row.FirstRank,
			Intent:             GuessSearchIntent(row.Keyword),
		})
	}
	return nil
}

func (s *dataForSEO) fillKeywordGapOverlap(ctx context.Context, host, rival string, out *KeywordGapResult) error {
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/domain_intersection/live", map[string]any{
		"target1":       host,
		"target2":       rival,
		"location_code": DataForSEOLocation,
		"language_code": DataForSEOLanguage,
		"intersections": true,
		"item_types":    []string{"organic"},
		"limit":         rankedKeywordLimit,
		"order_by":      []string{"keyword_data.keyword_info.search_volume,desc"},
	})
	if err != nil {
		return err
	}
	rows, err := parseGapItems(raw)
	if err != nil {
		return err
	}
	out.Overlap = make([]GapKeyword, 0, len(rows))
	for _, row := range rows {
		if row.Keyword == "" {
			continue
		}
		out.Overlap = append(out.Overlap, GapKeyword{
			Keyword:            row.Keyword,
			Volume:             row.Volume,
			CPC:                row.CPC,
			KD:                 row.KD,
			TargetPosition:     row.FirstRank,
			CompetitorPosition: row.SecondRank,
			Intent:             GuessSearchIntent(row.Keyword),
		})
	}
	return nil
}

// DomainCompare pulls Labs rank overview and backlink summary for two
// domains. Sections fail independently. Results are not written to dashboard metrics.
func (s *dataForSEO) DomainCompare(ctx context.Context, target, competitor string) (DomainCompareResult, error) {
	out := DomainCompareResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(target)
	rival := NormalizeDomain(competitor)
	if host == "" || rival == "" {
		return out, fmt.Errorf("invalid domain")
	}
	if host == rival {
		return out, fmt.Errorf("competitor must be a different domain")
	}
	out.Target = host
	out.Competitor = rival
	out.TargetSide = s.compareSide(ctx, host)
	out.CompetitorSide = s.compareSide(ctx, rival)
	return out, nil
}

func (s *dataForSEO) compareSide(ctx context.Context, host string) DomainCompareSide {
	side := DomainCompareSide{Domain: host}
	tmp := DomainExplorerResult{}
	if err := s.fillOrganicOverview(ctx, host, &tmp); err != nil {
		side.OrganicError = err.Error()
	} else {
		side.OrganicOK = true
		side.Organic = tmp.Organic
	}
	if err := s.fillBacklinks(ctx, host, &tmp); err != nil {
		side.BacklinksError = err.Error()
	} else {
		side.BacklinksOK = true
		side.Backlinks = tmp.Backlinks
	}
	return side
}

// ContentGap pulls competitor ranking pages (relevant_pages) and keywords
// their pages rank for that ours do not (page_intersection + exclude).
// Sections fail independently. Results are not written to dashboard metrics.
func (s *dataForSEO) ContentGap(ctx context.Context, target, competitor string) (ContentGapResult, error) {
	out := ContentGapResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
		Missing:      []ContentGapKeyword{},
		Pages:        []CompetitorPage{},
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(target)
	rival := NormalizeDomain(competitor)
	if host == "" || rival == "" {
		return out, fmt.Errorf("invalid domain")
	}
	if host == rival {
		return out, fmt.Errorf("competitor must be a different domain")
	}
	out.Target = host
	out.Competitor = rival

	if err := s.fillContentGapMissing(ctx, host, rival, &out); err != nil {
		out.MissingError = err.Error()
	} else {
		out.MissingOK = true
	}
	if err := s.fillContentGapPages(ctx, rival, &out); err != nil {
		out.PagesError = err.Error()
	} else {
		out.PagesOK = true
	}
	return out, nil
}

func domainWildcardURL(host string) string {
	return "https://" + host + "/*"
}

func (s *dataForSEO) fillContentGapMissing(ctx context.Context, host, rival string, out *ContentGapResult) error {
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/page_intersection/live", map[string]any{
		"pages":         map[string]string{"1": domainWildcardURL(rival)},
		"exclude_pages": []string{domainWildcardURL(host)},
		"location_code": DataForSEOLocation,
		"language_code": DataForSEOLanguage,
		"item_types":    []string{"organic"},
		"limit":         rankedKeywordLimit,
		"order_by":      []string{"keyword_data.keyword_info.search_volume,desc"},
	})
	if err != nil {
		return err
	}
	var wrap []struct {
		Items []pageIntersectionRow `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return err
	}
	out.Missing = []ContentGapKeyword{}
	if len(wrap) == 0 {
		return nil
	}
	for _, item := range wrap[0].Items {
		kw := strings.TrimSpace(item.KeywordData.Keyword)
		if kw == "" {
			continue
		}
		kd := item.KeywordData.KeywordProperties.KeywordDifficulty
		if kd == 0 {
			kd = item.KeywordData.KeywordInfo.KeywordDifficulty
		}
		hit := item.IntersectionResult["1"]
		out.Missing = append(out.Missing, ContentGapKeyword{
			Keyword:            kw,
			Volume:             item.KeywordData.KeywordInfo.SearchVolume,
			CPC:                item.KeywordData.KeywordInfo.CPC,
			KD:                 kd,
			CompetitorURL:      strings.TrimSpace(hit.URL),
			CompetitorPosition: hit.RankGroup,
			Intent:             GuessSearchIntent(kw),
		})
	}
	return nil
}

func (s *dataForSEO) fillContentGapPages(ctx context.Context, rival string, out *ContentGapResult) error {
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/relevant_pages/live", map[string]any{
		"target":        rival,
		"location_code": DataForSEOLocation,
		"language_code": DataForSEOLanguage,
		"limit":         rankedKeywordLimit,
		"order_by":      []string{"metrics.organic.etv,desc"},
	})
	if err != nil {
		return err
	}
	var wrap []struct {
		Items []relevantPageRow `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return err
	}
	out.Pages = []CompetitorPage{}
	if len(wrap) == 0 {
		return nil
	}
	for _, item := range wrap[0].Items {
		url := strings.TrimSpace(item.PageAddress)
		if url == "" {
			continue
		}
		out.Pages = append(out.Pages, CompetitorPage{
			URL:      url,
			ETV:      item.Metrics.Organic.ETV,
			Keywords: item.Metrics.Organic.Count,
			Pos1:     item.Metrics.Organic.Pos1,
			Pos2To3:  item.Metrics.Organic.Pos2To3,
			Pos4To10: item.Metrics.Organic.Pos4To10,
		})
	}
	return nil
}

// CitationGap compares LLM citation counts for two domains (Google AI
// Overviews + ChatGPT, US/en). Sections fail independently. Unique sources
// are co-cited with the competitor and not with the target when both
// lookups succeed. Results are not written to dashboard metrics.
func (s *dataForSEO) CitationGap(ctx context.Context, target, competitor string) (CitationGapResult, error) {
	out := CitationGapResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
		TargetCitations: LLMMetrics{
			Sources: []LLMSourceDomain{},
		},
		CompetitorCitations: LLMMetrics{
			Sources: []LLMSourceDomain{},
		},
		UniqueSources: []LLMSourceDomain{},
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(target)
	rival := NormalizeDomain(competitor)
	if host == "" || rival == "" {
		return out, fmt.Errorf("invalid domain")
	}
	if host == rival {
		return out, fmt.Errorf("competitor must be a different domain")
	}
	out.Target = host
	out.Competitor = rival

	if err := s.fillLLMTargetMetrics(ctx, host, "sources", &out.TargetCitations); err != nil {
		out.TargetError = err.Error()
	} else {
		out.TargetOK = true
	}
	if err := s.fillLLMTargetMetrics(ctx, rival, "sources", &out.CompetitorCitations); err != nil {
		out.CompetitorError = err.Error()
	} else {
		out.CompetitorOK = true
	}
	if out.TargetOK && out.CompetitorOK {
		ours := map[string]bool{host: true, rival: true}
		for _, src := range out.TargetCitations.Sources {
			if src.Domain != "" {
				ours[src.Domain] = true
			}
		}
		for _, src := range out.CompetitorCitations.Sources {
			if src.Domain == "" || ours[src.Domain] {
				continue
			}
			out.UniqueSources = append(out.UniqueSources, src)
		}
	}
	return out, nil
}

// AIVisibilityGap compares LLM mention counts (search_scope any) for two
// domains. Distinct from CitationGap, which uses sources. Sections fail
// independently. Unique sources are co-mentioned with the competitor and
// not with the target when both lookups succeed. No composite score is
// computed. Results are not written to dashboard metrics.
func (s *dataForSEO) AIVisibilityGap(ctx context.Context, target, competitor string) (AIVisibilityGapResult, error) {
	out := AIVisibilityGapResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
		TargetMentions: LLMMetrics{
			Sources: []LLMSourceDomain{},
		},
		CompetitorMentions: LLMMetrics{
			Sources: []LLMSourceDomain{},
		},
		UniqueSources: []LLMSourceDomain{},
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(target)
	rival := NormalizeDomain(competitor)
	if host == "" || rival == "" {
		return out, fmt.Errorf("invalid domain")
	}
	if host == rival {
		return out, fmt.Errorf("competitor must be a different domain")
	}
	out.Target = host
	out.Competitor = rival

	if err := s.fillLLMTargetMetrics(ctx, host, "any", &out.TargetMentions); err != nil {
		out.TargetError = err.Error()
	} else {
		out.TargetOK = true
	}
	if err := s.fillLLMTargetMetrics(ctx, rival, "any", &out.CompetitorMentions); err != nil {
		out.CompetitorError = err.Error()
	} else {
		out.CompetitorOK = true
	}
	if out.TargetOK && out.CompetitorOK {
		ours := map[string]bool{host: true, rival: true}
		for _, src := range out.TargetMentions.Sources {
			if src.Domain != "" {
				ours[src.Domain] = true
			}
		}
		for _, src := range out.CompetitorMentions.Sources {
			if src.Domain == "" || ours[src.Domain] {
				continue
			}
			out.UniqueSources = append(out.UniqueSources, src)
		}
	}
	return out, nil
}

// BacklinkGap lists referring domains that link to the competitor and
// not to the target, via DataForSEO backlinks domain intersection.
// Rank is DataForSEO Rank, not Moz DA. Results are not written to
// dashboard metrics and are not an outreach list.
func (s *dataForSEO) BacklinkGap(ctx context.Context, target, competitor string) (BacklinkGapResult, error) {
	out := BacklinkGapResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
		Referring:    []ReferringDomain{},
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(target)
	rival := NormalizeDomain(competitor)
	if host == "" || rival == "" {
		return out, fmt.Errorf("invalid domain")
	}
	if host == rival {
		return out, fmt.Errorf("competitor must be a different domain")
	}
	out.Target = host
	out.Competitor = rival

	raw, err := s.postLive(ctx, "/v3/backlinks/domain_intersection/live", map[string]any{
		"targets":                    map[string]string{"1": rival},
		"exclude_targets":            []string{host},
		"limit":                      backlinkGapLimit,
		"order_by":                   []string{"1.rank,desc"},
		"include_subdomains":         false,
		"exclude_internal_backlinks": true,
		"backlinks_status_type":      "live",
	})
	if err != nil {
		out.IntersectionError = err.Error()
		return out, nil
	}
	var wrap []struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			DomainIntersection map[string]backlinkIntersectionHit `json:"domain_intersection"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		out.IntersectionError = err.Error()
		return out, nil
	}
	out.IntersectionOK = true
	if len(wrap) == 0 {
		return out, nil
	}
	out.TotalCount = wrap[0].TotalCount
	for _, item := range wrap[0].Items {
		hit := item.DomainIntersection["1"]
		domain := strings.TrimSpace(hit.Target)
		if domain == "" {
			continue
		}
		norm := NormalizeDomain(domain)
		if norm == host || norm == rival {
			continue
		}
		out.Referring = append(out.Referring, ReferringDomain{
			Domain:         domain,
			Rank:           hit.Rank,
			Backlinks:      hit.Backlinks,
			ReferringPages: hit.ReferringPages,
			SpamScore:      hit.BacklinksSpamScore,
			FirstSeen:      strings.TrimSpace(hit.FirstSeen),
		})
	}
	return out, nil
}

// BacklinkExplorer returns a backlink audit (summary) and a sample of
// referring domains for a target. Rank is DataForSEO Rank, not Moz DA.
// Results are not written to dashboard metrics and are not an outreach list.
func (s *dataForSEO) BacklinkExplorer(ctx context.Context, domain string) (BacklinkExplorerResult, error) {
	out := BacklinkExplorerResult{
		Source:       "dataforseo",
		LocationCode: DataForSEOLocation,
		LocationName: DataForSEOLocationName,
		LanguageCode: DataForSEOLanguage,
		FetchedAt:    time.Now().UTC(),
		Referring:    []ReferringDomain{},
		Audit: BacklinkAudit{
			LinkTypes:      map[string]int{},
			LinkAttributes: map[string]int{},
		},
	}
	if !s.Configured() {
		return out, ErrDataForSEONotConfigured
	}
	host := NormalizeDomain(domain)
	if host == "" {
		return out, fmt.Errorf("invalid domain")
	}
	out.Domain = host

	if err := s.fillBacklinkAudit(ctx, host, &out); err != nil {
		out.SummaryError = err.Error()
	} else {
		out.SummaryOK = true
	}
	if err := s.fillReferringDomains(ctx, host, &out); err != nil {
		out.ReferringError = err.Error()
	} else {
		out.ReferringOK = true
	}
	return out, nil
}

func (s *dataForSEO) fillBacklinkAudit(ctx context.Context, host string, out *BacklinkExplorerResult) error {
	raw, err := s.postLive(ctx, "/v3/backlinks/summary/live", map[string]any{
		"target":                     host,
		"include_subdomains":         true,
		"exclude_internal_backlinks": true,
		"backlinks_status_type":      "live",
		"internal_list_limit":        10,
	})
	if err != nil {
		return err
	}
	var rows []backlinkSummaryRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("no backlink summary for %s", host)
	}
	row := rows[0]
	out.Audit = BacklinkAudit{
		Backlinks:                row.Backlinks,
		ReferringDomains:         row.ReferringDomains,
		ReferringMainDomains:     row.ReferringMainDomains,
		Rank:                     row.Rank,
		SpamScore:                row.BacklinksSpamScore,
		TargetSpamScore:          row.Info.TargetSpamScore,
		BrokenBacklinks:          row.BrokenBacklinks,
		BrokenPages:              row.BrokenPages,
		ReferringDomainsNofollow: row.ReferringDomainsNofollow,
		ReferringPages:           row.ReferringPages,
		ReferringPagesNofollow:   row.ReferringPagesNofollow,
		ReferringIPs:             row.ReferringIPs,
		ReferringSubnets:         row.ReferringSubnets,
		FirstSeen:                strings.TrimSpace(row.FirstSeen),
		LinkTypes:                copyIntMap(row.ReferringLinksTypes),
		LinkAttributes:           copyIntMap(row.ReferringLinksAttributes),
	}
	return nil
}

func (s *dataForSEO) fillReferringDomains(ctx context.Context, host string, out *BacklinkExplorerResult) error {
	raw, err := s.postLive(ctx, "/v3/backlinks/referring_domains/live", map[string]any{
		"target":                     host,
		"limit":                      referringDomainsLimit,
		"order_by":                   []string{"rank,desc"},
		"include_subdomains":         true,
		"exclude_internal_backlinks": true,
		"backlinks_status_type":      "live",
	})
	if err != nil {
		return err
	}
	var wrap []struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Domain                 string `json:"domain"`
			Rank                   int    `json:"rank"`
			Backlinks              int    `json:"backlinks"`
			ReferringPages         int    `json:"referring_pages"`
			BacklinksSpamScore     int    `json:"backlinks_spam_score"`
			FirstSeen              string `json:"first_seen"`
			BrokenBacklinks        int    `json:"broken_backlinks"`
			ReferringPagesNofollow int    `json:"referring_pages_nofollow"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return err
	}
	out.Referring = []ReferringDomain{}
	if len(wrap) == 0 {
		return nil
	}
	out.TotalCount = wrap[0].TotalCount
	for _, item := range wrap[0].Items {
		domain := strings.TrimSpace(item.Domain)
		if domain == "" {
			continue
		}
		norm := NormalizeDomain(domain)
		if norm == host {
			continue
		}
		out.Referring = append(out.Referring, ReferringDomain{
			Domain:          domain,
			Rank:            item.Rank,
			Backlinks:       item.Backlinks,
			ReferringPages:  item.ReferringPages,
			SpamScore:       item.BacklinksSpamScore,
			FirstSeen:       strings.TrimSpace(item.FirstSeen),
			BrokenBacklinks: item.BrokenBacklinks,
			NofollowPages:   item.ReferringPagesNofollow,
		})
	}
	return nil
}

func copyIntMap(in map[string]int) map[string]int {
	out := map[string]int{}
	for k, v := range in {
		if k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func parseGapItems(raw json.RawMessage) ([]parsedGapRow, error) {
	var wrap []struct {
		Items []domainIntersectionRow `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	if len(wrap) == 0 {
		return []parsedGapRow{}, nil
	}
	out := make([]parsedGapRow, 0, len(wrap[0].Items))
	for _, item := range wrap[0].Items {
		kw := strings.TrimSpace(item.KeywordData.Keyword)
		if kw == "" {
			continue
		}
		kd := item.KeywordData.KeywordProperties.KeywordDifficulty
		if kd == 0 {
			kd = item.KeywordData.KeywordInfo.KeywordDifficulty
		}
		out = append(out, parsedGapRow{
			Keyword:    kw,
			Volume:     item.KeywordData.KeywordInfo.SearchVolume,
			CPC:        item.KeywordData.KeywordInfo.CPC,
			KD:         kd,
			FirstRank:  item.FirstDomainSerpElement.RankGroup,
			SecondRank: item.SecondDomainSerpElement.RankGroup,
		})
	}
	return out, nil
}

func uniqueKeywordList(keywords []string, limit int) []string {
	seen := map[string]bool{}
	out := make([]string, 0, limit)
	for _, raw := range keywords {
		kw := strings.ToLower(strings.TrimSpace(raw))
		if kw == "" || seen[kw] || len(kw) > 80 {
			continue
		}
		seen[kw] = true
		out = append(out, kw)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (s *dataForSEO) fillLLMTargetMetrics(ctx context.Context, host, scope string, dest *LLMMetrics) error {
	raw, err := s.postLive(ctx, "/v3/ai_optimization/llm_mentions/target_metrics/live", map[string]any{
		"target": []map[string]any{{
			"domain":             host,
			"search_scope":       []string{scope},
			"include_subdomains": true,
		}},
		"location_code":       DataForSEOLocation,
		"language_code":       DataForSEOLanguage,
		"internal_list_limit": competitorLimit,
	})
	if err != nil {
		return err
	}
	parsed, err := parseLLMMetrics(raw, host)
	if err != nil {
		return err
	}
	*dest = parsed
	return nil
}

func parseLLMMetrics(raw json.RawMessage, host string) (LLMMetrics, error) {
	var rows []struct {
		AggregatedMetrics struct {
			Platform      []llmMetricGroup `json:"platform"`
			SourcesDomain []llmMetricGroup `json:"sources_domain"`
			Total         struct {
				Mentions       int `json:"mentions"`
				AISearchVolume int `json:"ai_search_volume"`
			} `json:"total"`
		} `json:"aggregated_metrics"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return LLMMetrics{}, err
	}
	if len(rows) == 0 {
		return LLMMetrics{}, fmt.Errorf("no llm mention metrics for %s", host)
	}
	agg := rows[0].AggregatedMetrics
	out := LLMMetrics{
		Mentions:       agg.Total.Mentions,
		AISearchVolume: agg.Total.AISearchVolume,
		Sources:        []LLMSourceDomain{},
	}
	for _, p := range agg.Platform {
		switch strings.ToLower(p.keyString()) {
		case "google":
			out.Google = LLMPlatformMetrics{Mentions: p.Mentions, AISearchVolume: p.AISearchVolume}
		case "chat_gpt", "chatgpt":
			out.ChatGPT = LLMPlatformMetrics{Mentions: p.Mentions, AISearchVolume: p.AISearchVolume}
		}
	}
	for _, s := range agg.SourcesDomain {
		d := NormalizeDomain(s.keyString())
		if d == "" || d == host {
			continue
		}
		out.Sources = append(out.Sources, LLMSourceDomain{
			Domain:         d,
			Mentions:       s.Mentions,
			AISearchVolume: s.AISearchVolume,
		})
	}
	return out, nil
}

func (s *dataForSEO) fillOrganicOverview(ctx context.Context, host string, out *DomainExplorerResult) error {
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/domain_rank_overview/live", map[string]any{
		"target":        host,
		"location_code": DataForSEOLocation,
		"language_code": DataForSEOLanguage,
	})
	if err != nil {
		return err
	}
	var rows []domainRankRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return err
	}
	row, ok := pickDomainRankRow(rows)
	if !ok {
		return fmt.Errorf("no domain rank overview for %s", host)
	}
	out.Organic = OrganicOverview{
		ETV:      row.Metrics.Organic.ETV,
		Count:    row.Metrics.Organic.Count,
		Pos1:     row.Metrics.Organic.Pos1,
		Pos2To3:  row.Metrics.Organic.Pos2To3,
		Pos4To10: row.Metrics.Organic.Pos4To10,
		PaidETV:  row.Metrics.Paid.ETV,
	}
	return nil
}

func (s *dataForSEO) fillCompetitors(ctx context.Context, host string, out *DomainExplorerResult) error {
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/competitors_domain/live", map[string]any{
		"target":              host,
		"location_code":       DataForSEOLocation,
		"language_code":       DataForSEOLanguage,
		"limit":               competitorLimit,
		"exclude_top_domains": true,
		"item_types":          []string{"organic"},
	})
	if err != nil {
		return err
	}
	var wrap []struct {
		Items []competitorRow `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return err
	}
	out.Competitors = []CompetitorDomain{}
	if len(wrap) == 0 {
		return nil
	}
	for _, item := range wrap[0].Items {
		d := strings.ToLower(strings.TrimSpace(item.Domain))
		if d == "" || d == host {
			continue
		}
		out.Competitors = append(out.Competitors, CompetitorDomain{
			Domain:        d,
			Intersections: item.Intersections,
			AvgPosition:   item.AvgPosition,
			OrganicETV:    item.FullDomainMetrics.Organic.ETV,
			OrganicCount:  item.FullDomainMetrics.Organic.Count,
		})
	}
	return nil
}

func (s *dataForSEO) fillRankedKeywords(ctx context.Context, host string, out *DomainExplorerResult) error {
	raw, err := s.postLive(ctx, "/v3/dataforseo_labs/google/ranked_keywords/live", map[string]any{
		"target":        host,
		"location_code": DataForSEOLocation,
		"language_code": DataForSEOLanguage,
		"limit":         rankedKeywordLimit,
		"item_types":    []string{"organic"},
		"order_by":      []string{"keyword_data.keyword_info.search_volume,desc"},
	})
	if err != nil {
		return err
	}
	var wrap []struct {
		Items []rankedKeywordRow `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return err
	}
	out.Keywords = []RankedKeyword{}
	if len(wrap) == 0 {
		return nil
	}
	for _, item := range wrap[0].Items {
		kw := strings.TrimSpace(item.KeywordData.Keyword)
		if kw == "" {
			continue
		}
		out.Keywords = append(out.Keywords, RankedKeyword{
			Keyword:  kw,
			Volume:   item.KeywordData.KeywordInfo.SearchVolume,
			CPC:      item.KeywordData.KeywordInfo.CPC,
			KD:       item.KeywordData.KeywordProperties.KeywordDifficulty,
			Position: item.RankedSerpElement.SerpItem.RankGroup,
			Intent:   GuessSearchIntent(kw),
		})
	}
	return nil
}

func (s *dataForSEO) fillBacklinks(ctx context.Context, host string, out *DomainExplorerResult) error {
	raw, err := s.postLive(ctx, "/v3/backlinks/summary/live", map[string]any{
		"target":             host,
		"include_subdomains": true,
	})
	if err != nil {
		return err
	}
	var rows []backlinkSummaryRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("no backlink summary for %s", host)
	}
	row := rows[0]
	out.Backlinks = BacklinkOverview{
		Backlinks:        row.Backlinks,
		ReferringDomains: row.ReferringDomains,
		Rank:             row.Rank,
		SpamScore:        row.BacklinksSpamScore,
	}
	return nil
}

func (s *dataForSEO) postLive(ctx context.Context, path string, task map[string]any) (json.RawMessage, error) {
	body, err := json.Marshal([]map[string]any{task})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.baseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.login, s.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dataforseo HTTP %d", resp.StatusCode)
	}
	var env dfsEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.StatusCode != 20000 {
		msg := env.StatusMessage
		if msg == "" {
			msg = fmt.Sprintf("status %d", env.StatusCode)
		}
		return nil, fmt.Errorf("dataforseo: %s", msg)
	}
	if len(env.Tasks) == 0 {
		return nil, fmt.Errorf("dataforseo: empty tasks")
	}
	t := env.Tasks[0]
	if t.StatusCode != 20000 {
		msg := t.StatusMessage
		if msg == "" {
			msg = fmt.Sprintf("task status %d", t.StatusCode)
		}
		return nil, fmt.Errorf("dataforseo: %s", msg)
	}
	if len(t.Result) == 0 || string(t.Result) == "null" {
		return json.RawMessage("[]"), nil
	}
	return t.Result, nil
}

// NormalizeDomain returns host without scheme, www, or path.
func NormalizeDomain(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	host := strings.TrimPrefix(u.Hostname(), "www.")
	host = strings.Trim(host, ".")
	if host == "" || strings.Contains(host, " ") {
		return ""
	}
	return host
}

func pickDomainRankRow(rows []domainRankRow) (domainRankRow, bool) {
	if len(rows) == 0 {
		return domainRankRow{}, false
	}
	// Some payloads nest location rows under items.
	if len(rows[0].Items) > 0 {
		for _, item := range rows[0].Items {
			if item.LocationCode == DataForSEOLocation {
				return item, true
			}
		}
		return rows[0].Items[0], true
	}
	for _, row := range rows {
		if row.LocationCode == DataForSEOLocation || row.Metrics.Organic.Count > 0 || row.Metrics.Organic.ETV > 0 {
			return row, true
		}
	}
	return rows[0], rows[0].Target != "" || rows[0].Metrics.Organic.Count > 0
}

type DomainExplorerResult struct {
	Domain           string             `json:"domain"`
	Source           string             `json:"source"`
	LocationCode     int                `json:"location_code"`
	LocationName     string             `json:"location_name"`
	LanguageCode     string             `json:"language_code"`
	Organic          OrganicOverview    `json:"organic"`
	OrganicOK        bool               `json:"organic_ok"`
	OrganicError     string             `json:"organic_error,omitempty"`
	Backlinks        BacklinkOverview   `json:"backlinks"`
	BacklinksOK      bool               `json:"backlinks_ok"`
	BacklinksError   string             `json:"backlinks_error,omitempty"`
	Competitors      []CompetitorDomain `json:"competitors"`
	CompetitorsOK    bool               `json:"competitors_ok"`
	CompetitorsError string             `json:"competitors_error,omitempty"`
	Keywords         []RankedKeyword    `json:"keywords"`
	KeywordsOK       bool               `json:"keywords_ok"`
	KeywordsError    string             `json:"keywords_error,omitempty"`
	FetchedAt        time.Time          `json:"fetched_at"`
	Cached           bool               `json:"cached"`
}

func (r DomainExplorerResult) HasAnyData() bool {
	return r.OrganicOK || r.BacklinksOK || r.CompetitorsOK || r.KeywordsOK
}

type OrganicOverview struct {
	ETV      float64 `json:"etv"`
	Count    int     `json:"count"`
	Pos1     int     `json:"pos_1"`
	Pos2To3  int     `json:"pos_2_3"`
	Pos4To10 int     `json:"pos_4_10"`
	PaidETV  float64 `json:"paid_etv"`
}

type BacklinkOverview struct {
	Backlinks        int `json:"backlinks"`
	ReferringDomains int `json:"referring_domains"`
	Rank             int `json:"rank"`
	SpamScore        int `json:"spam_score"`
}

type CompetitorDomain struct {
	Domain        string  `json:"domain"`
	Intersections int     `json:"intersections"`
	AvgPosition   float64 `json:"avg_position"`
	OrganicETV    float64 `json:"organic_etv"`
	OrganicCount  int     `json:"organic_count"`
}

type RankedKeyword struct {
	Keyword  string  `json:"keyword"`
	Volume   int     `json:"volume"`
	CPC      float64 `json:"cpc"`
	KD       int     `json:"kd"`
	Position int     `json:"position"`
	Intent   string  `json:"intent"`
}

type dfsEnvelope struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Tasks         []struct {
		StatusCode    int             `json:"status_code"`
		StatusMessage string          `json:"status_message"`
		Result        json.RawMessage `json:"result"`
	} `json:"tasks"`
}

type domainRankRow struct {
	Target       string `json:"target"`
	LocationCode int    `json:"location_code"`
	LanguageCode string `json:"language_code"`
	Metrics      struct {
		Organic struct {
			Pos1     int     `json:"pos_1"`
			Pos2To3  int     `json:"pos_2_3"`
			Pos4To10 int     `json:"pos_4_10"`
			ETV      float64 `json:"etv"`
			Count    int     `json:"count"`
		} `json:"organic"`
		Paid struct {
			ETV   float64 `json:"etv"`
			Count int     `json:"count"`
		} `json:"paid"`
	} `json:"metrics"`
	Items []domainRankRow `json:"items"`
}

type competitorRow struct {
	Domain            string  `json:"domain"`
	AvgPosition       float64 `json:"avg_position"`
	Intersections     int     `json:"intersections"`
	FullDomainMetrics struct {
		Organic struct {
			ETV   float64 `json:"etv"`
			Count int     `json:"count"`
		} `json:"organic"`
	} `json:"full_domain_metrics"`
}

type rankedKeywordRow struct {
	KeywordData struct {
		Keyword     string `json:"keyword"`
		KeywordInfo struct {
			SearchVolume int     `json:"search_volume"`
			CPC          float64 `json:"cpc"`
			Competition  float64 `json:"competition"`
		} `json:"keyword_info"`
		KeywordProperties struct {
			KeywordDifficulty int `json:"keyword_difficulty"`
		} `json:"keyword_properties"`
	} `json:"keyword_data"`
	RankedSerpElement struct {
		SerpItem struct {
			RankGroup int    `json:"rank_group"`
			Type      string `json:"type"`
		} `json:"serp_item"`
	} `json:"ranked_serp_element"`
}

type backlinkSummaryRow struct {
	Backlinks                int            `json:"backlinks"`
	ReferringDomains         int            `json:"referring_domains"`
	ReferringMainDomains     int            `json:"referring_main_domains"`
	Rank                     int            `json:"rank"`
	BacklinksSpamScore       int            `json:"backlinks_spam_score"`
	BrokenBacklinks          int            `json:"broken_backlinks"`
	BrokenPages              int            `json:"broken_pages"`
	ReferringDomainsNofollow int            `json:"referring_domains_nofollow"`
	ReferringPages           int            `json:"referring_pages"`
	ReferringPagesNofollow   int            `json:"referring_pages_nofollow"`
	ReferringIPs             int            `json:"referring_ips"`
	ReferringSubnets         int            `json:"referring_subnets"`
	FirstSeen                string         `json:"first_seen"`
	ReferringLinksTypes      map[string]int `json:"referring_links_types"`
	ReferringLinksAttributes map[string]int `json:"referring_links_attributes"`
	Info                     struct {
		TargetSpamScore int `json:"target_spam_score"`
	} `json:"info"`
}

type AIVisibilityResult struct {
	Domain         string     `json:"domain"`
	Source         string     `json:"source"`
	LocationCode   int        `json:"location_code"`
	LocationName   string     `json:"location_name"`
	LanguageCode   string     `json:"language_code"`
	Mentions       LLMMetrics `json:"mentions"`
	MentionsOK     bool       `json:"mentions_ok"`
	MentionsError  string     `json:"mentions_error,omitempty"`
	Citations      LLMMetrics `json:"citations"`
	CitationsOK    bool       `json:"citations_ok"`
	CitationsError string     `json:"citations_error,omitempty"`
	FetchedAt      time.Time  `json:"fetched_at"`
	Cached         bool       `json:"cached"`
}

func (r AIVisibilityResult) HasAnyData() bool {
	return r.MentionsOK || r.CitationsOK
}

type LLMMetrics struct {
	Mentions       int                `json:"mentions"`
	AISearchVolume int                `json:"ai_search_volume"`
	Google         LLMPlatformMetrics `json:"google"`
	ChatGPT        LLMPlatformMetrics `json:"chat_gpt"`
	Sources        []LLMSourceDomain  `json:"sources"`
}

type LLMPlatformMetrics struct {
	Mentions       int `json:"mentions"`
	AISearchVolume int `json:"ai_search_volume"`
}

type LLMSourceDomain struct {
	Domain         string `json:"domain"`
	Mentions       int    `json:"mentions"`
	AISearchVolume int    `json:"ai_search_volume"`
}

type KeywordLabsMetrics struct {
	SearchVolume     int             `json:"search_volume"`
	CPC              float64         `json:"cpc"`
	KD               int             `json:"kd"`
	Competition      float64         `json:"competition"`
	CompetitionLevel string          `json:"competition_level"`
	LabsIntent       string          `json:"labs_intent"`
	SerpFeatures     []string        `json:"serp_features"`
	MonthlySearches  []MonthlySearch `json:"monthly_searches"`
	TrendMonthly     *int            `json:"trend_monthly,omitempty"`
}

type MonthlySearch struct {
	Year         int `json:"year"`
	Month        int `json:"month"`
	SearchVolume int `json:"search_volume"`
}

type RelatedKeyword struct {
	Keyword          string  `json:"keyword"`
	SearchVolume     int     `json:"search_volume"`
	CPC              float64 `json:"cpc"`
	KD               int     `json:"kd"`
	CompetitionLevel string  `json:"competition_level"`
}

type RelatedKeywordsResult struct {
	Seed         string           `json:"seed"`
	Source       string           `json:"source"`
	LocationCode int              `json:"location_code"`
	LocationName string           `json:"location_name"`
	LanguageCode string           `json:"language_code"`
	Related      []RelatedKeyword `json:"related"`
	RelatedOK    bool             `json:"related_ok"`
	RelatedError string           `json:"related_error,omitempty"`
	FetchedAt    time.Time        `json:"fetched_at"`
	Cached       bool             `json:"cached"`
}

func (r RelatedKeywordsResult) HasAnyData() bool {
	return r.RelatedOK
}

type KeywordGapResult struct {
	Target       string       `json:"target"`
	Competitor   string       `json:"competitor"`
	Source       string       `json:"source"`
	LocationCode int          `json:"location_code"`
	LocationName string       `json:"location_name"`
	LanguageCode string       `json:"language_code"`
	Missing      []GapKeyword `json:"missing"`
	MissingOK    bool         `json:"missing_ok"`
	MissingError string       `json:"missing_error,omitempty"`
	Overlap      []GapKeyword `json:"overlap"`
	OverlapOK    bool         `json:"overlap_ok"`
	OverlapError string       `json:"overlap_error,omitempty"`
	FetchedAt    time.Time    `json:"fetched_at"`
	Cached       bool         `json:"cached"`
}

func (r KeywordGapResult) HasAnyData() bool {
	return r.MissingOK || r.OverlapOK
}

type DomainCompareResult struct {
	Target         string            `json:"target"`
	Competitor     string            `json:"competitor"`
	Source         string            `json:"source"`
	LocationCode   int               `json:"location_code"`
	LocationName   string            `json:"location_name"`
	LanguageCode   string            `json:"language_code"`
	TargetSide     DomainCompareSide `json:"target_side"`
	CompetitorSide DomainCompareSide `json:"competitor_side"`
	FetchedAt      time.Time         `json:"fetched_at"`
	Cached         bool              `json:"cached"`
}

func (r DomainCompareResult) HasAnyData() bool {
	return r.TargetSide.HasAnyData() || r.CompetitorSide.HasAnyData()
}

type DomainCompareSide struct {
	Domain         string           `json:"domain"`
	Organic        OrganicOverview  `json:"organic"`
	OrganicOK      bool             `json:"organic_ok"`
	OrganicError   string           `json:"organic_error,omitempty"`
	Backlinks      BacklinkOverview `json:"backlinks"`
	BacklinksOK    bool             `json:"backlinks_ok"`
	BacklinksError string           `json:"backlinks_error,omitempty"`
}

func (s DomainCompareSide) HasAnyData() bool {
	return s.OrganicOK || s.BacklinksOK
}

type ContentGapResult struct {
	Target       string              `json:"target"`
	Competitor   string              `json:"competitor"`
	Source       string              `json:"source"`
	LocationCode int                 `json:"location_code"`
	LocationName string              `json:"location_name"`
	LanguageCode string              `json:"language_code"`
	Missing      []ContentGapKeyword `json:"missing"`
	MissingOK    bool                `json:"missing_ok"`
	MissingError string              `json:"missing_error,omitempty"`
	Pages        []CompetitorPage    `json:"pages"`
	PagesOK      bool                `json:"pages_ok"`
	PagesError   string              `json:"pages_error,omitempty"`
	FetchedAt    time.Time           `json:"fetched_at"`
	Cached       bool                `json:"cached"`
}

func (r ContentGapResult) HasAnyData() bool {
	return r.MissingOK || r.PagesOK
}

type CitationGapResult struct {
	Target              string            `json:"target"`
	Competitor          string            `json:"competitor"`
	Source              string            `json:"source"`
	LocationCode        int               `json:"location_code"`
	LocationName        string            `json:"location_name"`
	LanguageCode        string            `json:"language_code"`
	TargetCitations     LLMMetrics        `json:"target_citations"`
	TargetOK            bool              `json:"target_ok"`
	TargetError         string            `json:"target_error,omitempty"`
	CompetitorCitations LLMMetrics        `json:"competitor_citations"`
	CompetitorOK        bool              `json:"competitor_ok"`
	CompetitorError     string            `json:"competitor_error,omitempty"`
	UniqueSources       []LLMSourceDomain `json:"unique_sources"`
	FetchedAt           time.Time         `json:"fetched_at"`
	Cached              bool              `json:"cached"`
}

func (r CitationGapResult) HasAnyData() bool {
	return r.TargetOK || r.CompetitorOK
}

type AIVisibilityGapResult struct {
	Target             string            `json:"target"`
	Competitor         string            `json:"competitor"`
	Source             string            `json:"source"`
	LocationCode       int               `json:"location_code"`
	LocationName       string            `json:"location_name"`
	LanguageCode       string            `json:"language_code"`
	TargetMentions     LLMMetrics        `json:"target_mentions"`
	TargetOK           bool              `json:"target_ok"`
	TargetError        string            `json:"target_error,omitempty"`
	CompetitorMentions LLMMetrics        `json:"competitor_mentions"`
	CompetitorOK       bool              `json:"competitor_ok"`
	CompetitorError    string            `json:"competitor_error,omitempty"`
	UniqueSources      []LLMSourceDomain `json:"unique_sources"`
	FetchedAt          time.Time         `json:"fetched_at"`
	Cached             bool              `json:"cached"`
}

func (r AIVisibilityGapResult) HasAnyData() bool {
	return r.TargetOK || r.CompetitorOK
}

type BacklinkGapResult struct {
	Target            string            `json:"target"`
	Competitor        string            `json:"competitor"`
	Source            string            `json:"source"`
	LocationCode      int               `json:"location_code"`
	LocationName      string            `json:"location_name"`
	LanguageCode      string            `json:"language_code"`
	Referring         []ReferringDomain `json:"referring"`
	IntersectionOK    bool              `json:"intersection_ok"`
	IntersectionError string            `json:"intersection_error,omitempty"`
	TotalCount        int               `json:"total_count"`
	FetchedAt         time.Time         `json:"fetched_at"`
	Cached            bool              `json:"cached"`
}

func (r BacklinkGapResult) HasAnyData() bool {
	return r.IntersectionOK
}

type BacklinkExplorerResult struct {
	Domain         string            `json:"domain"`
	Source         string            `json:"source"`
	LocationCode   int               `json:"location_code"`
	LocationName   string            `json:"location_name"`
	LanguageCode   string            `json:"language_code"`
	Audit          BacklinkAudit     `json:"audit"`
	SummaryOK      bool              `json:"summary_ok"`
	SummaryError   string            `json:"summary_error,omitempty"`
	Referring      []ReferringDomain `json:"referring"`
	ReferringOK    bool              `json:"referring_ok"`
	ReferringError string            `json:"referring_error,omitempty"`
	TotalCount     int               `json:"total_count"`
	FetchedAt      time.Time         `json:"fetched_at"`
	Cached         bool              `json:"cached"`
}

func (r BacklinkExplorerResult) HasAnyData() bool {
	return r.SummaryOK || r.ReferringOK
}

type BacklinkAudit struct {
	Backlinks                int            `json:"backlinks"`
	ReferringDomains         int            `json:"referring_domains"`
	ReferringMainDomains     int            `json:"referring_main_domains"`
	Rank                     int            `json:"rank"`
	SpamScore                int            `json:"spam_score"`
	TargetSpamScore          int            `json:"target_spam_score"`
	BrokenBacklinks          int            `json:"broken_backlinks"`
	BrokenPages              int            `json:"broken_pages"`
	ReferringDomainsNofollow int            `json:"referring_domains_nofollow"`
	ReferringPages           int            `json:"referring_pages"`
	ReferringPagesNofollow   int            `json:"referring_pages_nofollow"`
	ReferringIPs             int            `json:"referring_ips"`
	ReferringSubnets         int            `json:"referring_subnets"`
	FirstSeen                string         `json:"first_seen"`
	LinkTypes                map[string]int `json:"link_types"`
	LinkAttributes           map[string]int `json:"link_attributes"`
}

type ReferringDomain struct {
	Domain          string `json:"domain"`
	Rank            int    `json:"rank"`
	Backlinks       int    `json:"backlinks"`
	ReferringPages  int    `json:"referring_pages"`
	SpamScore       int    `json:"spam_score"`
	FirstSeen       string `json:"first_seen"`
	BrokenBacklinks int    `json:"broken_backlinks,omitempty"`
	NofollowPages   int    `json:"referring_pages_nofollow,omitempty"`
}

type backlinkIntersectionHit struct {
	Target             string `json:"target"`
	Rank               int    `json:"rank"`
	Backlinks          int    `json:"backlinks"`
	FirstSeen          string `json:"first_seen"`
	BacklinksSpamScore int    `json:"backlinks_spam_score"`
	ReferringPages     int    `json:"referring_pages"`
}

type ContentGapKeyword struct {
	Keyword            string  `json:"keyword"`
	Volume             int     `json:"volume"`
	CPC                float64 `json:"cpc"`
	KD                 int     `json:"kd"`
	CompetitorURL      string  `json:"competitor_url"`
	CompetitorPosition int     `json:"competitor_position"`
	Intent             string  `json:"intent"`
}

type CompetitorPage struct {
	URL      string  `json:"url"`
	ETV      float64 `json:"etv"`
	Keywords int     `json:"keywords"`
	Pos1     int     `json:"pos_1"`
	Pos2To3  int     `json:"pos_2_3"`
	Pos4To10 int     `json:"pos_4_10"`
}

type pageIntersectionRow struct {
	KeywordData struct {
		Keyword     string `json:"keyword"`
		KeywordInfo struct {
			SearchVolume      int     `json:"search_volume"`
			CPC               float64 `json:"cpc"`
			KeywordDifficulty int     `json:"keyword_difficulty"`
		} `json:"keyword_info"`
		KeywordProperties struct {
			KeywordDifficulty int `json:"keyword_difficulty"`
		} `json:"keyword_properties"`
	} `json:"keyword_data"`
	IntersectionResult map[string]struct {
		Type      string `json:"type"`
		RankGroup int    `json:"rank_group"`
		URL       string `json:"url"`
	} `json:"intersection_result"`
}

type relevantPageRow struct {
	PageAddress string `json:"page_address"`
	Metrics     struct {
		Organic struct {
			Pos1     int     `json:"pos_1"`
			Pos2To3  int     `json:"pos_2_3"`
			Pos4To10 int     `json:"pos_4_10"`
			ETV      float64 `json:"etv"`
			Count    int     `json:"count"`
		} `json:"organic"`
	} `json:"metrics"`
}

type GapKeyword struct {
	Keyword            string  `json:"keyword"`
	Volume             int     `json:"volume"`
	CPC                float64 `json:"cpc"`
	KD                 int     `json:"kd"`
	TargetPosition     int     `json:"target_position"`
	CompetitorPosition int     `json:"competitor_position"`
	Intent             string  `json:"intent"`
}

type parsedGapRow struct {
	Keyword    string
	Volume     int
	CPC        float64
	KD         int
	FirstRank  int
	SecondRank int
}

type domainIntersectionRow struct {
	KeywordData struct {
		Keyword     string `json:"keyword"`
		KeywordInfo struct {
			SearchVolume      int     `json:"search_volume"`
			CPC               float64 `json:"cpc"`
			KeywordDifficulty int     `json:"keyword_difficulty"`
		} `json:"keyword_info"`
		KeywordProperties struct {
			KeywordDifficulty int `json:"keyword_difficulty"`
		} `json:"keyword_properties"`
	} `json:"keyword_data"`
	FirstDomainSerpElement struct {
		RankGroup int `json:"rank_group"`
	} `json:"first_domain_serp_element"`
	SecondDomainSerpElement struct {
		RankGroup int `json:"rank_group"`
	} `json:"second_domain_serp_element"`
}

type monthlySearchRow struct {
	Year         int `json:"year"`
	Month        int `json:"month"`
	SearchVolume int `json:"search_volume"`
}

type keywordOverviewRow struct {
	Keyword     string `json:"keyword"`
	KeywordInfo struct {
		SearchVolume      int                `json:"search_volume"`
		CPC               float64            `json:"cpc"`
		KeywordDifficulty int                `json:"keyword_difficulty"`
		Competition       float64            `json:"competition"`
		CompetitionLevel  string             `json:"competition_level"`
		MonthlySearches   []monthlySearchRow `json:"monthly_searches"`
		SearchVolumeTrend struct {
			Monthly *int `json:"monthly"`
		} `json:"search_volume_trend"`
	} `json:"keyword_info"`
	KeywordProperties struct {
		KeywordDifficulty int `json:"keyword_difficulty"`
	} `json:"keyword_properties"`
	SearchIntentInfo struct {
		MainIntent string `json:"main_intent"`
	} `json:"search_intent_info"`
	SerpInfo struct {
		SerpItemTypes []string `json:"serp_item_types"`
	} `json:"serp_info"`
}

type relatedKeywordData struct {
	Keyword     string `json:"keyword"`
	KeywordInfo struct {
		SearchVolume      int     `json:"search_volume"`
		CPC               float64 `json:"cpc"`
		KeywordDifficulty int     `json:"keyword_difficulty"`
		CompetitionLevel  string  `json:"competition_level"`
	} `json:"keyword_info"`
	KeywordProperties struct {
		KeywordDifficulty int `json:"keyword_difficulty"`
	} `json:"keyword_properties"`
}

type llmMetricGroup struct {
	Key            json.RawMessage `json:"key"`
	Mentions       int             `json:"mentions"`
	AISearchVolume int             `json:"ai_search_volume"`
}

func (g llmMetricGroup) keyString() string {
	b := bytes.TrimSpace(g.Key)
	if len(b) == 0 || string(b) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		return s
	}
	var f float64
	if json.Unmarshal(b, &f) == nil {
		return fmt.Sprintf("%.0f", f)
	}
	return strings.Trim(string(b), `"`)
}
