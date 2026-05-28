package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Recover catches panics in any handler and returns a structured 500 JSON
// response, logging the full stack trace with the request ID for debugging.
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("request_id")
				stack := debug.Stack()

				log.Error().
					Interface("error", err).
					Str("requestId", requestID.(string)).
					Str("stack", string(stack)).
					Msg("panic recovered")

				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "An unexpected error occurred. Our team has been notified.",
					},
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
