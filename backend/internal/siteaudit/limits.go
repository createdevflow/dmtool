// Package siteaudit holds Semrush-aligned Site Audit plan limits and reports.
package siteaudit

import "backend/internal/models"

// RunPageCap is the hard per-Start page ceiling. Render's HTTP timeout is 30s;
// PSI is skipped on this path so HTML fetches can finish. Plan page_limit is
// stored and shown; the run stops at this cap and says so.
const RunPageCap = 8

// Limits mirrors Semrush Site Audit caps by plan.
// Mapped onto DMTool plans:
//
//	free        → Semrush Free (no SEO Toolkit)
//	pro_monthly → Semrush SEO / Pro
//	pro_yearly  → Semrush Pro+ / Guru
//	admin role  → Semrush Advanced / Business
type Limits struct {
	PlanLabel        string `json:"plan_label"`
	PlanCode         string `json:"plan_code"`
	PagesPerAudit    int    `json:"pages_per_audit"`
	PagesPerMonth    int    `json:"pages_per_month"`
	Websites         int    `json:"websites"`
	ConcurrentAudits int    `json:"concurrent_audits"`
	JSRendering      bool   `json:"js_rendering"`
	Schedule         bool   `json:"schedule"`
	CompareCrawls    bool   `json:"compare_crawls"`
	AISearchHealth   bool   `json:"ai_search_health"`
	RunPageCap       int    `json:"run_page_cap"`
}

// ForPlan returns Semrush-equivalent Site Audit limits.
func ForPlan(planCode string, isAdmin bool) Limits {
	if isAdmin {
		return advancedLimits(planCode)
	}
	switch planCode {
	case models.PlanCodeProYearly:
		return Limits{
			PlanLabel:        "Pro+",
			PlanCode:         planCode,
			PagesPerAudit:    20000,
			PagesPerMonth:    300000,
			Websites:         15,
			ConcurrentAudits: 2,
			JSRendering:      true,
			Schedule:         true,
			CompareCrawls:    true,
			AISearchHealth:   false,
			RunPageCap:       RunPageCap,
		}
	case models.PlanCodeProMonthly:
		return Limits{
			PlanLabel:        "SEO",
			PlanCode:         planCode,
			PagesPerAudit:    20000,
			PagesPerMonth:    100000,
			Websites:         5,
			ConcurrentAudits: 2,
			JSRendering:      false,
			Schedule:         true,
			CompareCrawls:    true,
			AISearchHealth:   false,
			RunPageCap:       RunPageCap,
		}
	default:
		return Limits{
			PlanLabel:        "Free",
			PlanCode:         models.PlanCodeFree,
			PagesPerAudit:    100,
			PagesPerMonth:    100,
			Websites:         1,
			ConcurrentAudits: 1,
			JSRendering:      false,
			Schedule:         false,
			CompareCrawls:    false,
			AISearchHealth:   false,
			RunPageCap:       RunPageCap,
		}
	}
}

func advancedLimits(planCode string) Limits {
	if planCode == "" {
		planCode = models.PlanCodeFree
	}
	return Limits{
		PlanLabel:        "Advanced",
		PlanCode:         planCode,
		PagesPerAudit:    100000,
		PagesPerMonth:    1000000,
		Websites:         40,
		ConcurrentAudits: 5,
		JSRendering:      true,
		Schedule:         true,
		CompareCrawls:    true,
		AISearchHealth:   false,
		RunPageCap:       RunPageCap,
	}
}

// ClampPageLimit keeps the requested crawl size inside the plan cap.
func ClampPageLimit(limit int, limits Limits) int {
	if limit <= 0 {
		limit = limits.PagesPerAudit
		if limit > RunPageCap {
			limit = RunPageCap
		}
		if limit <= 0 {
			return 1
		}
		return limit
	}
	if limit > limits.PagesPerAudit {
		limit = limits.PagesPerAudit
	}
	if limit < 1 {
		return 1
	}
	return limit
}
