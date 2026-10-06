package handlers

import (
	"fmt"
	"net/url"
	"strings"

	"backend/internal/models"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

func topBreakdown(rows []models.GSCBreakdown) *models.GSCBreakdown {
	var best *models.GSCBreakdown
	for i := range rows {
		r := &rows[i]
		if best == nil {
			best = r
			continue
		}
		if r.Clicks > best.Clicks || (r.Clicks == best.Clicks && r.Impressions > best.Impressions) {
			best = r
		}
	}
	return best
}

func mobileClickShare(devices []models.GSCBreakdown) (pct float64, ok bool) {
	var mobileClicks, totalClicks, mobileImp, totalImp int64
	for _, d := range devices {
		totalClicks += d.Clicks
		totalImp += d.Impressions
		if strings.EqualFold(d.Key, "MOBILE") {
			mobileClicks = d.Clicks
			mobileImp = d.Impressions
		}
	}
	if totalClicks > 0 {
		return float64(mobileClicks) / float64(totalClicks) * 100, true
	}
	if totalImp > 0 {
		return float64(mobileImp) / float64(totalImp) * 100, true
	}
	return 0, false
}

func displayPageKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	if path == "/" {
		return u.Host + "/"
	}
	return path
}

func displayCountryKey(code string) string {
	c := strings.ToLower(strings.TrimSpace(code))
	if name, ok := gscCountryNames[c]; ok {
		return name
	}
	if c == "" {
		return code
	}
	return strings.ToUpper(c)
}

func displayDeviceKey(key string) string {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "MOBILE":
		return "Mobile"
	case "DESKTOP":
		return "Desktop"
	case "TABLET":
		return "Tablet"
	default:
		return key
	}
}

func displayAppearanceKey(key string) string {
	s := strings.ReplaceAll(strings.TrimSpace(key), "_", " ")
	if s == "" {
		return key
	}
	return s
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

func gscBreakdownJSON(rows []models.GSCBreakdown, labelFn func(string) string) []gin.H {
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"key":         r.Key,
			"label":       labelFn(r.Key),
			"clicks":      r.Clicks,
			"impressions": r.Impressions,
			"ctr_pct":     r.CTR * 100,
			"position":    r.Position,
		})
	}
	return out
}

func gscTopPageStat(pages []models.GSCBreakdown, hasGSC bool) gin.H {
	if !hasGSC && len(pages) == 0 {
		return dashStat("Top Page", "Connect GSC", "Link")
	}
	top := topBreakdown(pages)
	if top == nil {
		return dashStat("Top Page", "Sync GSC", "Link")
	}
	return gin.H{
		"label":  "Top Page",
		"value":  truncateRunes(displayPageKey(top.Key), 28),
		"change": utils.FormatNumber(top.Clicks) + " clicks",
		"trend":  "flat",
		"icon":   "Link",
	}
}

func gscTopCountryStat(countries []models.GSCBreakdown, hasGSC bool) gin.H {
	if !hasGSC && len(countries) == 0 {
		return dashStat("Top Country", "Connect GSC", "MapPin")
	}
	top := topBreakdown(countries)
	if top == nil {
		return dashStat("Top Country", "Sync GSC", "MapPin")
	}
	return gin.H{
		"label":  "Top Country",
		"value":  displayCountryKey(top.Key),
		"change": utils.FormatNumber(top.Clicks) + " clicks",
		"trend":  "flat",
		"icon":   "MapPin",
	}
}

func gscMobileShareStat(devices []models.GSCBreakdown, hasGSC bool) gin.H {
	if !hasGSC && len(devices) == 0 {
		return dashStat("Mobile Traffic", "Connect GSC", "Smartphone")
	}
	pct, ok := mobileClickShare(devices)
	if !ok {
		return dashStat("Mobile Traffic", "Sync GSC", "Smartphone")
	}
	return gin.H{
		"label":  "Mobile Traffic",
		"value":  fmt.Sprintf("%.1f%%", pct),
		"change": "GSC devices",
		"trend":  "flat",
		"icon":   "Smartphone",
	}
}

// GSC country dimension is ISO 3166-1 alpha-3 (lowercase). Labels only — not metrics.
var gscCountryNames = map[string]string{
	"usa": "United States",
	"gbr": "United Kingdom",
	"ind": "India",
	"can": "Canada",
	"aus": "Australia",
	"deu": "Germany",
	"fra": "France",
	"bra": "Brazil",
	"jpn": "Japan",
	"idn": "Indonesia",
	"mex": "Mexico",
	"ita": "Italy",
	"esp": "Spain",
	"nld": "Netherlands",
	"pol": "Poland",
	"tur": "Turkey",
	"kor": "South Korea",
	"vnm": "Vietnam",
	"phl": "Philippines",
	"tha": "Thailand",
	"sgp": "Singapore",
	"are": "United Arab Emirates",
	"sau": "Saudi Arabia",
	"zaf": "South Africa",
	"nga": "Nigeria",
	"arg": "Argentina",
	"col": "Colombia",
	"chl": "Chile",
	"swe": "Sweden",
	"nor": "Norway",
	"dnk": "Denmark",
	"fin": "Finland",
	"irl": "Ireland",
	"nzl": "New Zealand",
	"mys": "Malaysia",
	"pak": "Pakistan",
	"bgd": "Bangladesh",
	"ukr": "Ukraine",
	"rou": "Romania",
	"cze": "Czechia",
	"prt": "Portugal",
	"bel": "Belgium",
	"che": "Switzerland",
	"aut": "Austria",
	"isr": "Israel",
	"egy": "Egypt",
	"chn": "China",
	"twn": "Taiwan",
	"hkg": "Hong Kong",
	"rus": "Russia",
}
