package models

import "time"

// Permission is a developer-maintained staff action. Rows are seeded
// from PermissionCatalog; there is no admin API to create codes.
// Super Admin / role_permissions (later) attach these to roles.
type Permission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Code        string `gorm:"uniqueIndex;not null;size:64" json:"code"`
	Name        string `gorm:"not null;size:120" json:"name"`
	Description string `gorm:"size:500" json:"description"`
	Category    string `gorm:"not null;size:40;index" json:"category"`
}

// Permission codes. Stable slugs — renaming is a migration, not a UI edit.
const (
	PermAdminAccess         = "admin.access"
	PermUsersRead           = "users.read"
	PermUsersUpdate         = "users.update"
	PermUsersExport         = "users.export"
	PermUsersActivityRead   = "users.activity.read"
	PermUsersPasswordReset  = "users.password.reset"
	PermUsersSuspend        = "users.suspend"
	PermUsersImpersonate    = "users.impersonate"
	PermPlansRead           = "plans.read"
	PermPlansWrite          = "plans.write"
	PermUsersRoleAssign     = "users.role.assign"
	PermStatsOverviewRead   = "stats.overview.read"
	PermStatsRevenueRead    = "stats.revenue.read"
	PermProjectsRead        = "projects.read"
	PermPlatformSEORead     = "platform.seo.read"
	PermPlatformSocialRead  = "platform.social.read"
	PermPlatformHealthRead  = "platform.health.read"
	PermIntegrationsRead    = "integrations.read"
	PermAuditRead           = "audit.read"
)

// PermissionCatalog is the single list SeedPermissions inserts.
// Stop impersonation is not here — AllowStopImpersonation is a one-off.
func PermissionCatalog() []Permission {
	return []Permission{
		{Code: PermAdminAccess, Name: "Open admin panel", Description: "May enter /admin. Used by the shell guard so it does not depend on stats/revenue.", Category: "platform"},
		{Code: PermUsersRead, Name: "View users", Description: "List and view user profiles.", Category: "users"},
		{Code: PermUsersUpdate, Name: "Edit users", Description: "Change name and email. Role changes need users.role.assign.", Category: "users"},
		{Code: PermUsersRoleAssign, Name: "Assign roles", Description: "Change a user's role. Required in addition to users.update when role actually changes.", Category: "users"},
		{Code: PermUsersExport, Name: "Export users", Description: "Download the user CSV.", Category: "users"},
		{Code: PermUsersActivityRead, Name: "View user activity", Description: "See a user's activity timeline.", Category: "users"},
		{Code: PermUsersPasswordReset, Name: "Reset passwords", Description: "Issue a temporary password.", Category: "users"},
		{Code: PermUsersSuspend, Name: "Suspend users", Description: "Suspend and unsuspend non-admin accounts.", Category: "users"},
		{Code: PermUsersImpersonate, Name: "Impersonate users", Description: "Start an impersonation session. Stop is not this permission.", Category: "users"},
		{Code: PermPlansRead, Name: "View plans", Description: "List billing plans.", Category: "plans"},
		{Code: PermPlansWrite, Name: "Edit plans", Description: "Create and update plans.", Category: "plans"},
		{Code: PermStatsOverviewRead, Name: "View platform overview", Description: "User/project counts. Does not include MRR/ARR or plan money.", Category: "finance"},
		{Code: PermStatsRevenueRead, Name: "View revenue", Description: "MRR, ARR, and plan-derived money fields.", Category: "finance"},
		{Code: PermProjectsRead, Name: "View all projects", Description: "List and inspect any user's projects.", Category: "projects"},
		{Code: PermPlatformSEORead, Name: "View SEO health", Description: "Platform-wide SEO aggregates.", Category: "platform"},
		{Code: PermPlatformSocialRead, Name: "View social health", Description: "Platform-wide social aggregates.", Category: "platform"},
		{Code: PermPlatformHealthRead, Name: "View system health", Description: "API/DB/OAuth health snapshot.", Category: "platform"},
		{Code: PermIntegrationsRead, Name: "View integrations", Description: "OAuth connection counts by provider.", Category: "platform"},
		{Code: PermAuditRead, Name: "View audit log", Description: "Read admin audit entries.", Category: "audit"},
	}
}
