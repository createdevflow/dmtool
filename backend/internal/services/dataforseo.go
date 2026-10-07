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
)

// ErrDataForSEONotConfigured is returned when login/password are empty.
var ErrDataForSEONotConfigured = errors.New("dataforseo is not connected")

type DataForSEOService interface {
	Configured() bool
	FetchEstimatedTraffic(siteURL string) ([]models.Metric, error)
	DomainExplorer(ctx context.Context, domain string) (DomainExplorerResult, error)
	AIVisibility(ctx context.Context, domain string) (AIVisibilityResult, error)
	KeywordOverview(ctx context.Context, keywords []string) (map[string]KeywordLabsMetrics, error)
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

// KeywordOverview returns Labs search volume, CPC, and KD for the given
// keywords (Google United States). Missing keywords are omitted — zeros
// are not invented. Results are not written to dashboard metrics.
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
		"include_serp_info": false,
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
			SearchVolume: item.KeywordInfo.SearchVolume,
			CPC:          item.KeywordInfo.CPC,
			KD:           kd,
		}
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
	Backlinks          int `json:"backlinks"`
	ReferringDomains   int `json:"referring_domains"`
	Rank               int `json:"rank"`
	BacklinksSpamScore int `json:"backlinks_spam_score"`
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
	SearchVolume int     `json:"search_volume"`
	CPC          float64 `json:"cpc"`
	KD           int     `json:"kd"`
}

type keywordOverviewRow struct {
	Keyword     string `json:"keyword"`
	KeywordInfo struct {
		SearchVolume      int     `json:"search_volume"`
		CPC               float64 `json:"cpc"`
		KeywordDifficulty int     `json:"keyword_difficulty"`
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
