package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const serpBatchLimit = 10

// SERPEngineGoogle and SERPEngineBing are the DataForSEO engines we can query.
const (
	SERPEngineGoogle = "google"
	SERPEngineBing   = "bing"
)

// SERPKeywordResult is one keyword's live SERP, parsed for our domain and competitors.
type SERPKeywordResult struct {
	Keyword         string     `json:"keyword"`
	OK              bool       `json:"ok"`
	Error           string     `json:"error,omitempty"`
	Items           []SERPItem `json:"items"`
	Target          *SERPRank  `json:"target,omitempty"`
	Competitors     []SERPRank `json:"competitors"`
	Cannibalizing   []SERPRank `json:"cannibalizing"`
	SERPFeatures    []string   `json:"serp_features"`
	FeaturedSnippet *SERPRank  `json:"featured_snippet,omitempty"`
}

// SERPItem is a raw organic/feature row.
type SERPItem struct {
	Type         string `json:"type"`
	RankGroup    int    `json:"rank_group"`
	RankAbsolute int    `json:"rank_absolute"`
	Domain       string `json:"domain"`
	URL          string `json:"url"`
	Title        string `json:"title"`
}

// SERPRank is a domain's position on one SERP.
type SERPRank struct {
	Domain   string `json:"domain"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Type     string `json:"type"`
}

// OrganicSERP looks up live Google or Bing organic results for keywords.
// depth is capped at 100. Unknown engines return an error without inventing ranks.
func (s *dataForSEO) OrganicSERP(ctx context.Context, engine, device string, locationCode int, languageCode string, keywords, domains []string) ([]SERPKeywordResult, error) {
	if !s.Configured() {
		return nil, ErrDataForSEONotConfigured
	}
	path, err := serpPath(engine)
	if err != nil {
		return nil, err
	}
	dev := strings.ToLower(strings.TrimSpace(device))
	if dev == "tablet" {
		dev = "mobile"
	}
	if dev != "mobile" {
		dev = "desktop"
	}
	if locationCode <= 0 {
		locationCode = DataForSEOLocation
	}
	if languageCode == "" {
		languageCode = DataForSEOLanguage
	}
	cleaned := uniqueKeywordList(keywords, 500)
	if len(cleaned) == 0 {
		return []SERPKeywordResult{}, nil
	}
	if len(cleaned) > serpBatchLimit {
		cleaned = cleaned[:serpBatchLimit]
	}

	tasks := make([]map[string]any, 0, len(cleaned))
	for _, kw := range cleaned {
		tasks = append(tasks, map[string]any{
			"keyword":         kw,
			"location_code":   locationCode,
			"language_code":   languageCode,
			"device":          dev,
			"os":              "windows",
			"depth":           100,
			"max_crawl_pages": 1,
		})
	}
	rawTasks, err := s.postLiveBatch(ctx, path, tasks)
	if err != nil {
		return nil, err
	}

	want := make([]string, 0, len(domains))
	for _, d := range domains {
		if host := NormalizeDomain(d); host != "" {
			want = append(want, host)
		}
	}

	out := make([]SERPKeywordResult, 0, len(cleaned))
	for i, kw := range cleaned {
		row := SERPKeywordResult{Keyword: kw, Competitors: []SERPRank{}, Cannibalizing: []SERPRank{}, SERPFeatures: []string{}, Items: []SERPItem{}}
		if i >= len(rawTasks) {
			row.Error = "missing task result"
			out = append(out, row)
			continue
		}
		items, features, ferr := parseSERPItems(rawTasks[i])
		if ferr != nil {
			row.Error = ferr.Error()
			out = append(out, row)
			continue
		}
		row.OK = true
		row.Items = items
		row.SERPFeatures = features
		row.Target, row.Competitors, row.Cannibalizing, row.FeaturedSnippet = splitSERPRanks(items, want)
		out = append(out, row)
	}
	return out, nil
}

func serpPath(engine string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "", SERPEngineGoogle:
		return "/v3/serp/google/organic/live/regular", nil
	case SERPEngineBing:
		return "/v3/serp/bing/organic/live/regular", nil
	default:
		return "", fmt.Errorf("position tracking engine %q is not connected (google and bing use DataForSEO)", engine)
	}
}

func (s *dataForSEO) postLiveBatch(ctx context.Context, path string, tasks []map[string]any) ([]json.RawMessage, error) {
	if len(tasks) == 0 {
		return []json.RawMessage{}, nil
	}
	body, err := json.Marshal(tasks)
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
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
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
	out := make([]json.RawMessage, 0, len(env.Tasks))
	for _, t := range env.Tasks {
		if t.StatusCode != 20000 {
			out = append(out, json.RawMessage("[]"))
			continue
		}
		if len(t.Result) == 0 || string(t.Result) == "null" {
			out = append(out, json.RawMessage("[]"))
			continue
		}
		out = append(out, t.Result)
	}
	return out, nil
}

type serpLiveResult struct {
	Items []struct {
		Type         string `json:"type"`
		RankGroup    int    `json:"rank_group"`
		RankAbsolute int    `json:"rank_absolute"`
		Domain       string `json:"domain"`
		URL          string `json:"url"`
		Title        string `json:"title"`
	} `json:"items"`
	ItemTypes []string `json:"item_types"`
}

func parseSERPItems(raw json.RawMessage) ([]SERPItem, []string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []SERPItem{}, []string{}, nil
	}
	var rows []serpLiveResult
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return []SERPItem{}, []string{}, nil
	}
	features := rows[0].ItemTypes
	if features == nil {
		features = []string{}
	}
	items := make([]SERPItem, 0, len(rows[0].Items))
	for _, it := range rows[0].Items {
		items = append(items, SERPItem{
			Type:         it.Type,
			RankGroup:    it.RankGroup,
			RankAbsolute: it.RankAbsolute,
			Domain:       NormalizeDomain(it.Domain),
			URL:          it.URL,
			Title:        it.Title,
		})
	}
	return items, features, nil
}

func splitSERPRanks(items []SERPItem, domains []string) (target *SERPRank, competitors []SERPRank, cannibal []SERPRank, snippet *SERPRank) {
	competitors = []SERPRank{}
	cannibal = []SERPRank{}
	if len(domains) == 0 {
		return nil, competitors, cannibal, nil
	}
	primary := domains[0]
	seenPrimary := 0
	compSet := map[string]bool{}
	for _, d := range domains[1:] {
		compSet[d] = true
	}
	for _, it := range items {
		if it.Domain == "" {
			continue
		}
		pos := it.RankGroup
		if pos <= 0 {
			pos = it.RankAbsolute
		}
		rank := SERPRank{Domain: it.Domain, Position: pos, URL: it.URL, Title: it.Title, Type: it.Type}
		if it.Type == "featured_snippet" && (it.Domain == primary || snippet == nil && it.Domain == primary) {
			cp := rank
			snippet = &cp
		}
		if it.Type != "organic" && it.Type != "featured_snippet" {
			continue
		}
		if it.Domain == primary {
			seenPrimary++
			if seenPrimary == 1 {
				cp := rank
				target = &cp
			} else {
				cannibal = append(cannibal, rank)
			}
			continue
		}
		if compSet[it.Domain] {
			if _, dup := findRank(competitors, it.Domain); !dup {
				competitors = append(competitors, rank)
			}
		}
	}
	return target, competitors, cannibal, snippet
}

func findRank(rows []SERPRank, domain string) (SERPRank, bool) {
	for _, r := range rows {
		if r.Domain == domain {
			return r, true
		}
	}
	return SERPRank{}, false
}
