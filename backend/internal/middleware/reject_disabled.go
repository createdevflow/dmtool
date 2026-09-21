package middleware

import (
	"net/http"
	"strings"

	"backend/internal/models"

	"github.com/gin-gonic/gin"
)

// UserByID is the slice of UserRepository RejectDisabled needs.
type UserByID interface {
	FindByID(id uint) (*models.User, error)
}

// RejectDisabled runs after JWTAuth on authenticated /api routes.
//
// JWT user_id is the token subject: the customer when impersonating,
// the account holder otherwise. A naive "disable user_id" check would
// 401 Stop (target JWT) and would miss a suspended admin whose id is
// only on impersonator_id.
//
// Rules:
//   - POST .../stop-impersonation: skip (admin must be able to leave
//     even if the target is disabled).
//   - Normal token: 401 if that user is disabled.
//   - Impersonation token: 401 if the target or the impersonator is disabled.
func RejectDisabled(users UserByID) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isStopImpersonationRequest(c) {
			c.Next()
			return
		}

		rawID, ok := c.Get("user_id")
		uid, _ := rawID.(uint)
		if !ok || uid == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"},
			})
			return
		}

		user, err := users.FindByID(uid)
		if err != nil || user == nil {
			abortDisabled(c)
			return
		}
		if user.DisabledAt != nil {
			abortDisabled(c)
			return
		}
		c.Set("auth_user", user)

		isImp, _ := c.Get("is_impersonation")
		impersonating, _ := isImp.(bool)
		if !impersonating {
			c.Next()
			return
		}

		rawImp, _ := c.Get("impersonator_id")
		impID, _ := rawImp.(uint)
		if impID == 0 {
			abortDisabled(c)
			return
		}
		admin, err := users.FindByID(impID)
		if err != nil || admin == nil || admin.DisabledAt != nil {
			abortDisabled(c)
			return
		}
		c.Set("impersonator_user", admin)
		c.Next()
	}
}

func isStopImpersonationRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	path := c.Request.URL.Path
	return strings.Contains(path, "/admin/users/") && strings.HasSuffix(path, "/stop-impersonation")
}

func abortDisabled(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"},
	})
}
