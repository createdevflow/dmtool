// Package positiontracking holds Semrush-aligned Position Tracking plan limits.
package positiontracking

import "backend/internal/models"

// Limits mirrors Semrush Position Tracking caps by plan.
// Mapped onto DMTool plans:
//
//	free        → Semrush Free (no SEO Toolkit)
//	pro_monthly → Semrush SEO / Pro
//	pro_yearly  → Semrush Pro+ / Guru
//	admin role  → Semrush Advanced / Business
type Limits struct {
	PlanLabel              string `json:"plan_label"`
	PlanCode               string `json:"plan_code"`
	Keywords               int    `json:"keywords"`
	Websites               int    `json:"websites"`
	CompetitorsTotal       int    `json:"competitors_total"`
	CompetitorsPerCampaign int    `json:"competitors_per_campaign"`
	Multitargeting         bool   `json:"multitargeting"`
	MaxTargets             int    `json:"max_targets"`
	Tags                   bool   `json:"tags"`
	Cannibalization        bool   `json:"cannibalization"`
	DevicesLocations       bool   `json:"devices_locations"`
	ShareOfVoice           bool   `json:"share_of_voice"`
	Historical             bool   `json:"historical"`
	AISearch               bool   `json:"ai_search"`
	ScheduledExport        bool   `json:"scheduled_export"`
	MobileData             bool   `json:"mobile_data"`
}

// ForPlan returns Semrush-equivalent Position Tracking limits.
func ForPlan(planCode string, isAdmin bool) Limits {
	if isAdmin {
		return advancedLimits(planCode)
	}
	switch planCode {
	case models.PlanCodeProYearly:
		return Limits{
			PlanLabel:              "Pro+",
			PlanCode:               planCode,
			Keywords:               1500,
			Websites:               15,
			CompetitorsTotal:       3000,
			CompetitorsPerCampaign: 20,
			Multitargeting:         true,
			MaxTargets:             10,
			Tags:                   true,
			Cannibalization:        true,
			DevicesLocations:       true,
			ShareOfVoice:           false,
			Historical:             true,
			AISearch:               true,
			ScheduledExport:        false,
			MobileData:             true,
		}
	case models.PlanCodeProMonthly:
		return Limits{
			PlanLabel:              "SEO",
			PlanCode:               planCode,
			Keywords:               500,
			Websites:               5,
			CompetitorsTotal:       100,
			CompetitorsPerCampaign: 20,
			Multitargeting:         false,
			MaxTargets:             1,
			Tags:                   false,
			Cannibalization:        false,
			DevicesLocations:       false,
			ShareOfVoice:           false,
			Historical:             false,
			AISearch:               false,
			ScheduledExport:        false,
			MobileData:             true,
		}
	default:
		return Limits{
			PlanLabel:              "Free",
			PlanCode:               models.PlanCodeFree,
			Keywords:               10,
			Websites:               1,
			CompetitorsTotal:       20,
			CompetitorsPerCampaign: 20,
			Multitargeting:         false,
			MaxTargets:             1,
			Tags:                   false,
			Cannibalization:        false,
			DevicesLocations:       false,
			ShareOfVoice:           false,
			Historical:             false,
			AISearch:               false,
			ScheduledExport:        false,
			MobileData:             true,
		}
	}
}

func advancedLimits(planCode string) Limits {
	if planCode == "" {
		planCode = models.PlanCodeFree
	}
	return Limits{
		PlanLabel:              "Advanced",
		PlanCode:               planCode,
		Keywords:               5000,
		Websites:               40,
		CompetitorsTotal:       4000000,
		CompetitorsPerCampaign: 20,
		Multitargeting:         true,
		MaxTargets:             5000,
		Tags:                   true,
		Cannibalization:        true,
		DevicesLocations:       true,
		ShareOfVoice:           true,
		Historical:             true,
		AISearch:               true,
		ScheduledExport:        true,
		MobileData:             true,
	}
}
