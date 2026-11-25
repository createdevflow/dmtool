// Package domainoverview holds Semrush-aligned Domain Overview plan limits and reports.
package domainoverview

import "backend/internal/models"

// Limits mirrors Semrush Domain Overview caps by plan.
// Mapped onto DMTool plans:
//
//	free        → Semrush Free (no SEO Toolkit) — 10 requests/day
//	pro_monthly → Semrush SEO / Pro — 3,000/day
//	pro_yearly  → Semrush Pro+ / Guru — 5,000/day + historical/growth/countries
//	admin role  → Semrush Advanced / Business — 10,000/day
type Limits struct {
	PlanLabel         string `json:"plan_label"`
	PlanCode          string `json:"plan_code"`
	RequestsPerDay    int    `json:"requests_per_day"`
	Historical        bool   `json:"historical"`
	Growth            bool   `json:"growth"`
	CompareCountries  bool   `json:"compare_countries"`
	ExportCSV         bool   `json:"export_csv"`
	AIVisibility      bool   `json:"ai_visibility"`
	SubfolderURLScope bool   `json:"subfolder_url_scope"`
}

// ForPlan returns Semrush-equivalent Domain Overview limits.
func ForPlan(planCode string, isAdmin bool) Limits {
	if isAdmin {
		return advancedLimits(planCode)
	}
	switch planCode {
	case models.PlanCodeProYearly:
		return Limits{
			PlanLabel:         "Pro+",
			PlanCode:          planCode,
			RequestsPerDay:    5000,
			Historical:        true,
			Growth:            true,
			CompareCountries:  true,
			ExportCSV:         true,
			AIVisibility:      false,
			SubfolderURLScope: false,
		}
	case models.PlanCodeProMonthly:
		return Limits{
			PlanLabel:         "SEO",
			PlanCode:          planCode,
			RequestsPerDay:    3000,
			Historical:        false,
			Growth:            false,
			CompareCountries:  false,
			ExportCSV:         true,
			AIVisibility:      false,
			SubfolderURLScope: false,
		}
	default:
		return Limits{
			PlanLabel:         "Free",
			PlanCode:          models.PlanCodeFree,
			RequestsPerDay:    10,
			Historical:        false,
			Growth:            false,
			CompareCountries:  false,
			ExportCSV:         true,
			AIVisibility:      false,
			SubfolderURLScope: false,
		}
	}
}

func advancedLimits(planCode string) Limits {
	if planCode == "" {
		planCode = models.PlanCodeFree
	}
	return Limits{
		PlanLabel:         "Advanced",
		PlanCode:          planCode,
		RequestsPerDay:    10000,
		Historical:        true,
		Growth:            true,
		CompareCountries:  true,
		ExportCSV:         true,
		AIVisibility:      false,
		SubfolderURLScope: false,
	}
}
