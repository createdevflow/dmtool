package middleware

import (
	"net/http"
	"strconv"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProjectGuard verifies that the authenticated user owns the project referenced
// in the request (via ?project_id query param or :id URL param).
// This middleware MUST run after JWTAuth so user_id is already in context.
func ProjectGuard(projectRepo repository.ProjectRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"},
			})
			c.Abort()
			return
		}

		// Accept project_id from query param or :id URL param
		projectIDStr := c.Query("project_id")
		if projectIDStr == "" {
			projectIDStr = c.Param("id")
		}

		if projectIDStr == "" {
			// No project scoping required for this route — pass through
			c.Next()
			return
		}

		projectID, err := strconv.ParseUint(projectIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   gin.H{"code": "VALIDATION_ERROR", "message": "Invalid project_id"},
			})
			c.Abort()
			return
		}

		uid := userID.(uint)
		_, err = projectRepo.FindByIDAndUser(uint(projectID), uid)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   gin.H{"code": "FORBIDDEN", "message": "Access denied to this project"},
			})
			c.Abort()
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Failed to validate project access"},
			})
			c.Abort()
			return
		}

		// Store validated project_id in context for handlers
		c.Set("project_id", uint(projectID))
		c.Next()
	}
}
