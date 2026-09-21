package middleware

import (
	"net/http"

	"backend/internal/models"

	"github.com/gin-gonic/gin"
)

// RequireRole returns a Gin middleware that aborts with 403 unless the
// authenticated user's role (set on the context by JWTAuth as
// "user_role") matches one of the allowed roles.
//
// Use this AFTER JWTAuth in the middleware chain. Applied to the
// /api/admin/* route group to gate the admin panel.
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"},
			})
			return
		}
		roleStr, _ := role.(string)
		if _, ok := allowedSet[roleStr]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "Admin access required"},
			})
			return
		}
		// Defense in depth: an admin who started impersonating another
		// user carries an impersonation token. Reject those on admin
		// routes so the admin cannot extend their session into the
		// other user's admin context (which they don't have anyway,
		// but this prevents any future code from accidentally trusting
		// the impersonation role).
		if isImp, _ := c.Get("is_impersonation"); isImp != nil {
			if b, ok := isImp.(bool); ok && b {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"success": false,
					"error":   gin.H{"code": "FORBIDDEN", "message": "Impersonation tokens cannot access admin routes"},
				})
				return
			}
		}
		c.Next()
	}
}

// MustRoleAdmin is sugar for RequireRole(models.RoleAdmin).
func MustRoleAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}

// AllowStopImpersonation gates POST /api/admin/users/:id/stop-impersonation.
// That route cannot sit behind RequireRole("admin"): while impersonating,
// the client sends the impersonation JWT (target's role, IsImpersonation=true),
// which RequireRole rejects. Allow either:
//
//   - a real admin token (not impersonating), so CLI/verifiers keep working
//   - a valid impersonation token (IsImpersonation + ImpersonatorID set)
func AllowStopImpersonation() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isImp, _ := c.Get("is_impersonation"); isImp != nil {
			if b, ok := isImp.(bool); ok && b {
				impID, _ := c.Get("impersonator_id")
				id, _ := impID.(uint)
				if id == 0 {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"success": false,
						"error":   gin.H{"code": "FORBIDDEN", "message": "Invalid impersonation token"},
					})
					return
				}
				c.Next()
				return
			}
		}
		role, _ := c.Get("user_role")
		roleStr, _ := role.(string)
		if roleStr == models.RoleAdmin {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   gin.H{"code": "FORBIDDEN", "message": "Admin access required"},
		})
	}
}
