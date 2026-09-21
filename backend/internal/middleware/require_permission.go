package middleware

import (
	"net/http"

	"backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const staffPermsKey = "staff_perms"

// LoadStaffPermissions runs on /api/admin after JWTAuth + RejectDisabled.
// Impersonation JWTs are always 403. Staff perms are loaded from
// users.role → roles.code → role_permissions (DB every request).
func LoadStaffPermissions(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isImp, _ := c.Get("is_impersonation"); isImp != nil {
			if b, ok := isImp.(bool); ok && b {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"success": false,
					"error":   gin.H{"code": "FORBIDDEN", "message": "Impersonation tokens cannot access admin routes"},
				})
				return
			}
		}
		u, _ := c.Get("auth_user")
		user, _ := u.(*models.User)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "Admin access required"},
			})
			return
		}
		var codes []string
		if err := database.Table("permissions").
			Select("permissions.code").
			Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
			Joins("JOIN roles ON roles.id = role_permissions.role_id").
			Where("roles.code = ?", user.Role).
			Pluck("code", &codes).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "Admin access required"},
			})
			return
		}
		set := map[string]bool{}
		for _, code := range codes {
			set[code] = true
		}
		c.Set(staffPermsKey, set)
		c.Next()
	}
}

// StaffPermsLoaded is true after LoadStaffPermissions ran (permission
// guard path). False under the legacy RequireRole fallback.
func StaffPermsLoaded(c *gin.Context) bool {
	_, ok := c.Get(staffPermsKey)
	return ok
}

// StaffHas reports whether the request's loaded staff perms include code.
func StaffHas(c *gin.Context, code string) bool {
	v, ok := c.Get(staffPermsKey)
	if !ok {
		return false
	}
	set, ok := v.(map[string]bool)
	if !ok {
		return false
	}
	return set[code]
}

// RequirePermission requires admin.access plus the given code.
func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if code != models.PermAdminAccess && !StaffHas(c, models.PermAdminAccess) {
			forbid(c)
			return
		}
		if !StaffHas(c, code) {
			forbid(c)
			return
		}
		c.Next()
	}
}

// RequireAnyPermission requires admin.access plus at least one of codes.
func RequireAnyPermission(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !StaffHas(c, models.PermAdminAccess) {
			forbid(c)
			return
		}
		for _, code := range codes {
			if StaffHas(c, code) {
				c.Next()
				return
			}
		}
		forbid(c)
	}
}

func forbid(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error":   gin.H{"code": "FORBIDDEN", "message": "Admin access required"},
	})
}
