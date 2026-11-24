package handlers

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/positiontracking"
	"backend/internal/services"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ptSaveRequest struct {
	ProjectID      uint               `json:"project_id" binding:"required"`
	Domain         string             `json:"domain"`
	LocationName   string             `json:"location_name"`
	LocationCode   int                `json:"location_code"`
	LanguageCode   string             `json:"language_code"`
	Device         string             `json:"device"`
	Engine         string             `json:"engine"`
	Keywords       []models.PTKeyword `json:"keywords"`
	KeywordText    string             `json:"keyword_text"`
	Competitors    []string           `json:"competitors"`
	CompetitorText string             `json:"competitor_text"`
	Targets        []models.PTTarget  `json:"targets"`
}

// GetPositionTracking returns the campaign, plan limits, and Semrush-style reports.
func (h *SEOHandler) GetPositionTracking(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}
	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)
	project, err := h.projectRepo.FindByIDAndUser(pid, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	limits := h.ptLimits(userID, ptIsAdmin(c))
	campaign, _ := h.seoRepo.FindPositionTrackingCampaign(pid)
	gsc := h.ptGSC(userID, pid)
	snap := positiontracking.Snapshot{Keywords: []positiontracking.KeywordRow{}}
	var view gin.H
	domain := services.NormalizeDomain(project.URL)
	comps := []string{}
	targets := []models.PTTarget{}
	if campaign != nil {
		view = ptCampaignJSON(campaign)
		domain = campaign.Domain
		comps = decodeStringSlice(campaign.CompetitorsJSON)
		targets = decodeTargets(campaign.TargetsJSON)
		if campaign.SnapshotJSON != "" {
			_ = json.Unmarshal([]byte(campaign.SnapshotJSON), &snap)
		}
	}
	if len(snap.Keywords) == 0 && gsc.connected {
		snap = gscFallbackSnapshot(decodeKeywords(campaignKeywordsJSON(campaign)), gsc.byKeyword)
	}

	utils.Success(c, gin.H{
		"limits":   limits,
		"campaign": view,
		"reports":  positiontracking.BuildReports(limits, snap, domain, comps, targets),
		"gsc": gin.H{
			"connected":        gsc.connected,
			"used_as_fallback": snap.Source == "gsc",
			"keyword_count":    len(gsc.byKeyword),
		},
		"serp": gin.H{
			"configured":    h.dataForSEOReady(),
			"available":     snap.Source == "dataforseo_serp",
			"message":       ptSERPMessage(h.dataForSEOReady(), snap.Source),
			"refresh_limit": positiontracking.RefreshKeywordLimit,
		},
		"honesty": ptHonesty(),
	}, nil)
}

// SavePositionTracking creates or updates the campaign within plan caps.
func (h *SEOHandler) SavePositionTracking(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	var req ptSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "project_id is required", "VALIDATION_ERROR")
		return
	}
	project, err := h.projectRepo.FindByIDAndUser(req.ProjectID, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	limits := h.ptLimits(userID, ptIsAdmin(c))
	existing, findErr := h.seoRepo.FindPositionTrackingCampaign(req.ProjectID)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		utils.InternalError(c, "Could not load campaign")
		return
	}
	if existing == nil {
		n, _ := h.seoRepo.CountPositionTrackingWebsites(userID)
		if n >= limits.Websites {
			utils.BadRequest(c, "Website cap reached for this plan ("+strconv.Itoa(limits.Websites)+"). Upgrade to track more sites.", "PLAN_LIMIT")
			return
		}
	}

	kws := req.Keywords
	if len(kws) == 0 && req.KeywordText != "" {
		kws = parseKeywordText(req.KeywordText)
	}
	comps := req.Competitors
	if len(comps) == 0 && req.CompetitorText != "" {
		comps = splitLines(req.CompetitorText)
	}
	kws, comps, targets := positiontracking.ApplyPlan(limits, kws, comps, req.Targets)

	if len(kws) == 0 {
		utils.BadRequest(c, "Add at least one keyword", "VALIDATION_ERROR")
		return
	}
	units := positiontracking.KeywordUnits(len(kws), len(targets), limits.Multitargeting)
	if units > limits.Keywords {
		utils.BadRequest(c, "Keyword cap exceeded for this plan ("+strconv.Itoa(limits.Keywords)+"). Extra locations and devices count toward the cap.", "PLAN_LIMIT")
		return
	}

	accountComps := ptAccountCompetitors(h, userID, req.ProjectID, comps)
	if len(accountComps) > limits.CompetitorsTotal {
		utils.BadRequest(c, "Competitor cap exceeded for this plan ("+strconv.Itoa(limits.CompetitorsTotal)+").", "PLAN_LIMIT")
		return
	}

	domain := services.NormalizeDomain(req.Domain)
	if domain == "" {
		domain = services.NormalizeDomain(project.URL)
	}
	if domain == "" {
		utils.BadRequest(c, "A domain is required", "VALIDATION_ERROR")
		return
	}

	engine := strings.ToLower(strings.TrimSpace(req.Engine))
	if engine == "" {
		engine = services.SERPEngineGoogle
	}
	if _, err := ptEngineOK(engine); err != nil {
		utils.BadRequest(c, err.Error(), "VALIDATION_ERROR")
		return
	}
	device := strings.ToLower(strings.TrimSpace(req.Device))
	if device != "mobile" {
		device = "desktop"
	}
	locName := strings.TrimSpace(req.LocationName)
	if locName == "" {
		locName = services.DataForSEOLocationName
	}
	locCode := req.LocationCode
	if locCode <= 0 {
		locCode = services.DataForSEOLocation
	}
	lang := strings.TrimSpace(req.LanguageCode)
	if lang == "" {
		lang = services.DataForSEOLanguage
	}

	row := &models.PositionTrackingCampaign{
		ProjectID:    req.ProjectID,
		Domain:       domain,
		LocationName: locName,
		LocationCode: locCode,
		LanguageCode: lang,
		Device:       device,
		Engine:       engine,
	}
	if existing != nil {
		row.ID = existing.ID
		row.CreatedAt = existing.CreatedAt
		row.SnapshotJSON = existing.SnapshotJSON
		row.SnapshotSource = existing.SnapshotSource
		row.SnapshotAt = existing.SnapshotAt
		row.SnapshotError = existing.SnapshotError
	}
	row.KeywordsJSON = mustJSON(kws)
	row.CompetitorsJSON = mustJSON(comps)
	row.TargetsJSON = mustJSON(targets)
	if err := h.seoRepo.SavePositionTrackingCampaign(row); err != nil {
		utils.InternalError(c, "Could not save campaign")
		return
	}

	saved, _ := h.seoRepo.FindPositionTrackingCampaign(req.ProjectID)
	utils.Success(c, gin.H{
		"limits":   limits,
		"campaign": ptCampaignJSON(saved),
		"message":  "Campaign saved. Click Update rankings to query Google or Bing via DataForSEO (max 10 keywords per refresh).",
	}, nil)
}

// RefreshPositionTracking looks up live SERP ranks. Cache-only unless fetch=1 or refresh=1.
func (h *SEOHandler) RefreshPositionTracking(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		var body struct {
			ProjectID uint `json:"project_id"`
		}
		_ = c.ShouldBindJSON(&body)
		if body.ProjectID == 0 {
			utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
			return
		}
		projectIDStr = strconv.FormatUint(uint64(body.ProjectID), 10)
	}
	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	pid := uint(projectID)
	project, err := h.projectRepo.FindByIDAndUser(pid, userID)
	if err != nil || project == nil {
		utils.NotFound(c, "Project not found")
		return
	}

	campaign, err := h.seoRepo.FindPositionTrackingCampaign(pid)
	if err != nil || campaign == nil {
		utils.BadRequest(c, "Save a Position Tracking campaign first", "NO_CAMPAIGN")
		return
	}

	refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")
	wantFetch := refresh || c.Query("fetch") == "1" || strings.EqualFold(c.Query("fetch"), "true")

	if !wantFetch {
		h.GetPositionTracking(c)
		return
	}
	if !h.dataForSEOReady() {
		campaign.SnapshotError = "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD."
		_ = h.seoRepo.SavePositionTrackingCampaign(campaign)
		utils.Success(c, gin.H{
			"limits":   h.ptLimits(userID, ptIsAdmin(c)),
			"campaign": ptCampaignJSON(campaign),
			"serp": gin.H{
				"configured": false,
				"available":  false,
				"message":    campaign.SnapshotError,
			},
			"honesty": ptHonesty(),
		}, nil)
		return
	}

	kws := decodeKeywords(campaign.KeywordsJSON)
	names := make([]string, 0, len(kws))
	for _, k := range kws {
		names = append(names, k.Keyword)
	}
	if len(names) > positiontracking.RefreshKeywordLimit {
		names = names[:positiontracking.RefreshKeywordLimit]
	}
	comps := decodeStringSlice(campaign.CompetitorsJSON)
	domains := append([]string{campaign.Domain}, comps...)

	results, serr := h.dataForSEOSvc.OrganicSERP(
		c.Request.Context(),
		campaign.Engine,
		campaign.Device,
		campaign.LocationCode,
		campaign.LanguageCode,
		names,
		domains,
	)
	now := time.Now().UTC()
	if serr != nil {
		campaign.SnapshotError = serr.Error()
		campaign.SnapshotAt = &now
		_ = h.seoRepo.SavePositionTrackingCampaign(campaign)
		utils.Success(c, gin.H{
			"limits":   h.ptLimits(userID, ptIsAdmin(c)),
			"campaign": ptCampaignJSON(campaign),
			"serp": gin.H{
				"configured": true,
				"available":  false,
				"message":    serr.Error(),
			},
			"honesty": ptHonesty(),
		}, nil)
		return
	}

	tagBy := map[string][]string{}
	for _, k := range kws {
		tagBy[k.Keyword] = k.Tags
	}
	gsc := h.ptGSC(userID, pid)
	rows := make([]positiontracking.KeywordRow, 0, len(results))
	byResult := map[string]services.SERPKeywordResult{}
	for _, r := range results {
		byResult[strings.ToLower(r.Keyword)] = r
	}
	for _, name := range names {
		r, ok := byResult[name]
		if !ok {
			r = services.SERPKeywordResult{Keyword: name, Competitors: []services.SERPRank{}, Cannibalizing: []services.SERPRank{}, SERPFeatures: []string{}}
		}
		row := positiontracking.KeywordRow{
			Keyword:       name,
			Tags:          tagBy[name],
			SERPFeatures:  r.SERPFeatures,
			Competitors:   ptRanks(r.Competitors),
			Cannibalizing: ptRanks(r.Cannibalizing),
			Discovered:    ptDiscovered(r.Items, campaign.Domain),
			Source:        "dataforseo_serp",
			Error:         r.Error,
			OK:            r.OK,
		}
		if r.Target != nil {
			row.Position = r.Target.Position
			row.URL = r.Target.URL
			row.Title = r.Target.Title
		}
		if r.FeaturedSnippet != nil {
			sn := ptRank(*r.FeaturedSnippet)
			row.FeaturedSnippet = &sn
		}
		if g, ok := gsc.byKeyword[name]; ok {
			row.GSCPosition = g.Position
			row.GSCImpressions = g.Volume
			row.GSCClicks = g.Clicks
		}
		if row.SERPFeatures == nil {
			row.SERPFeatures = []string{}
		}
		rows = append(rows, row)
	}

	snap := positiontracking.Snapshot{
		Source:       "dataforseo_serp",
		Engine:       campaign.Engine,
		Device:       campaign.Device,
		LocationName: campaign.LocationName,
		LocationCode: campaign.LocationCode,
		LanguageCode: campaign.LanguageCode,
		FetchedAt:    &now,
		Keywords:     rows,
	}
	campaign.SnapshotJSON = mustJSON(snap)
	campaign.SnapshotSource = "dataforseo_serp"
	campaign.SnapshotAt = &now
	campaign.SnapshotError = ""
	if err := h.seoRepo.SavePositionTrackingCampaign(campaign); err != nil {
		utils.InternalError(c, "Could not save snapshot")
		return
	}

	limits := h.ptLimits(userID, ptIsAdmin(c))
	utils.Success(c, gin.H{
		"limits":   limits,
		"campaign": ptCampaignJSON(campaign),
		"reports":  positiontracking.BuildReports(limits, snap, campaign.Domain, comps, decodeTargets(campaign.TargetsJSON)),
		"serp": gin.H{
			"configured":    true,
			"available":     true,
			"refreshed":     len(names),
			"tracked":       len(kws),
			"refresh_limit": positiontracking.RefreshKeywordLimit,
			"message":       ptRefreshNote(len(kws), len(names)),
		},
		"honesty": ptHonesty(),
	}, nil)
}

func (h *SEOHandler) ptLimits(userID uint, isAdmin bool) positiontracking.Limits {
	plan := models.PlanCodeFree
	if h.seoRepo != nil {
		if code, err := h.seoRepo.FindPlanCodeByUser(userID); err == nil && code != "" {
			plan = code
		}
	}
	return positiontracking.ForPlan(plan, isAdmin)
}

func ptIsAdmin(c *gin.Context) bool {
	role, _ := c.Get("user_role")
	s, _ := role.(string)
	return s == models.RoleAdmin
}

type ptGSCState struct {
	connected bool
	byKeyword map[string]models.KeywordResult
}

func (h *SEOHandler) ptGSC(userID, projectID uint) ptGSCState {
	out := ptGSCState{byKeyword: map[string]models.KeywordResult{}}
	if h.oauthRepo == nil {
		return out
	}
	cred, err := h.oauthRepo.FindByUserAndProvider(userID, "google")
	if err != nil || cred == nil {
		return out
	}
	out.connected = true
	if h.seoRepo == nil {
		return out
	}
	stored, _, _ := h.seoRepo.FindKeywords(projectID, "")
	for _, kw := range stored {
		if kw.Position <= 0 {
			continue
		}
		out.byKeyword[strings.ToLower(strings.TrimSpace(kw.Keyword))] = kw
	}
	return out
}

func gscFallbackSnapshot(campaignKWs []models.PTKeyword, gsc map[string]models.KeywordResult) positiontracking.Snapshot {
	rows := []positiontracking.KeywordRow{}
	seen := map[string]bool{}
	for _, kw := range campaignKWs {
		g, ok := gsc[kw.Keyword]
		row := positiontracking.KeywordRow{
			Keyword: kw.Keyword, Tags: kw.Tags, Source: "gsc",
			SERPFeatures: []string{}, Competitors: []positiontracking.Rank{}, Cannibalizing: []positiontracking.Rank{}, Discovered: []positiontracking.Rank{},
		}
		if ok {
			row.Position = int(g.Position)
			row.GSCPosition = g.Position
			row.GSCImpressions = g.Volume
			row.GSCClicks = g.Clicks
			row.OK = true
		}
		rows = append(rows, row)
		seen[kw.Keyword] = true
	}
	if len(campaignKWs) == 0 {
		for _, g := range gsc {
			k := strings.ToLower(strings.TrimSpace(g.Keyword))
			if seen[k] {
				continue
			}
			rows = append(rows, positiontracking.KeywordRow{
				Keyword: k, Position: int(g.Position), GSCPosition: g.Position,
				GSCImpressions: g.Volume, GSCClicks: g.Clicks, Source: "gsc", OK: true,
				SERPFeatures: []string{}, Competitors: []positiontracking.Rank{}, Cannibalizing: []positiontracking.Rank{}, Discovered: []positiontracking.Rank{},
			})
		}
	}
	src := "gsc"
	if len(rows) == 0 {
		src = ""
	}
	return positiontracking.Snapshot{Source: src, Keywords: rows}
}

func campaignKeywordsJSON(c *models.PositionTrackingCampaign) string {
	if c == nil {
		return ""
	}
	return c.KeywordsJSON
}

func ptCampaignJSON(c *models.PositionTrackingCampaign) gin.H {
	if c == nil {
		return nil
	}
	return gin.H{
		"id":              c.ID,
		"project_id":      c.ProjectID,
		"domain":          c.Domain,
		"location_name":   c.LocationName,
		"location_code":   c.LocationCode,
		"language_code":   c.LanguageCode,
		"device":          c.Device,
		"engine":          c.Engine,
		"keywords":        decodeKeywords(c.KeywordsJSON),
		"competitors":     decodeStringSlice(c.CompetitorsJSON),
		"targets":         decodeTargets(c.TargetsJSON),
		"snapshot_source": c.SnapshotSource,
		"snapshot_at":     c.SnapshotAt,
		"snapshot_error":  c.SnapshotError,
	}
}

func ptHonesty() gin.H {
	return gin.H{
		"etv":        "Estimated traffic is not computed. Semrush ETV is proprietary.",
		"visibility": "Visibility is the average of max(0, 100 − position) over ranked keywords in the current snapshot.",
		"ai_search":  "ChatGPT, Baidu, and Google AI Mode prompt tracking are not connected.",
		"history":    "Only the latest lookup is stored. Daily archive is not kept yet.",
	}
}

func ptSERPMessage(configured bool, source string) string {
	if !configured {
		return "DataForSEO is not connected. Set DATAFORSEO_LOGIN and DATAFORSEO_PASSWORD to query live Google or Bing rankings."
	}
	if source == "dataforseo_serp" {
		return "Live SERP snapshot from DataForSEO. Cached until you click Update rankings."
	}
	if source == "gsc" {
		return "Showing Google Search Console positions (not a daily SERP crawl). Click Update rankings to query Google or Bing."
	}
	return "Click Update rankings to query DataForSEO (uses vendor credits; max 10 keywords per refresh)."
}

func ptRefreshNote(tracked, refreshed int) string {
	if tracked > refreshed {
		return "Refreshed " + strconv.Itoa(refreshed) + " of " + strconv.Itoa(tracked) + " keywords. Render times out at 30s, so each refresh is capped at 10."
	}
	return "Refreshed " + strconv.Itoa(refreshed) + " keywords from DataForSEO live SERP."
}

func ptEngineOK(engine string) (string, error) {
	switch engine {
	case services.SERPEngineGoogle, services.SERPEngineBing:
		return engine, nil
	default:
		return "", errors.New("engine " + engine + " is not connected (google and bing use DataForSEO)")
	}
}

func ptRanks(in []services.SERPRank) []positiontracking.Rank {
	out := make([]positiontracking.Rank, 0, len(in))
	for _, r := range in {
		out = append(out, ptRank(r))
	}
	return out
}

func ptRank(r services.SERPRank) positiontracking.Rank {
	return positiontracking.Rank{Domain: r.Domain, Position: r.Position, URL: r.URL, Title: r.Title, Type: r.Type}
}

func ptDiscovered(items []services.SERPItem, target string) []positiontracking.Rank {
	target = strings.ToLower(strings.TrimSpace(target))
	seen := map[string]bool{}
	out := []positiontracking.Rank{}
	for _, it := range items {
		if it.Type != "organic" && it.Type != "featured_snippet" {
			continue
		}
		d := strings.ToLower(it.Domain)
		if d == "" || d == target || seen[d] {
			continue
		}
		seen[d] = true
		pos := it.RankGroup
		if pos <= 0 {
			pos = it.RankAbsolute
		}
		out = append(out, positiontracking.Rank{Domain: d, Position: pos, URL: it.URL, Title: it.Title, Type: it.Type})
		if len(out) >= 20 {
			break
		}
	}
	return out
}

func ptAccountCompetitors(h *SEOHandler, userID, skipProject uint, next []string) []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(list []string) {
		for _, d := range list {
			if d == "" || seen[d] {
				continue
			}
			seen[d] = true
			out = append(out, d)
		}
	}
	add(next)
	rows, err := h.seoRepo.ListPositionTrackingCampaignsByUser(userID)
	if err != nil {
		return out
	}
	for _, row := range rows {
		if row.ProjectID == skipProject {
			continue
		}
		add(decodeStringSlice(row.CompetitorsJSON))
	}
	return out
}

func parseKeywordText(s string) []models.PTKeyword {
	out := []models.PTKeyword{}
	for _, line := range splitLines(s) {
		kw := line
		var tags []string
		if i := strings.Index(line, "|"); i >= 0 {
			kw = strings.TrimSpace(line[:i])
			for _, t := range strings.Split(line[i+1:], ",") {
				if x := strings.TrimSpace(t); x != "" {
					tags = append(tags, x)
				}
			}
		}
		if kw == "" {
			continue
		}
		out = append(out, models.PTKeyword{Keyword: kw, Tags: tags})
	}
	return out
}

func splitLines(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' })
	out := []string{}
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func decodeKeywords(raw string) []models.PTKeyword {
	if raw == "" {
		return []models.PTKeyword{}
	}
	var out []models.PTKeyword
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []models.PTKeyword{}
	}
	return out
}

func decodeStringSlice(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

func decodeTargets(raw string) []models.PTTarget {
	if raw == "" {
		return []models.PTTarget{}
	}
	var out []models.PTTarget
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []models.PTTarget{}
	}
	return out
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
