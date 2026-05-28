// Package middleware provides all Gin middleware for DMTool v2.
// Middleware is applied in the order defined in Section 5.2.
package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders injects OWASP-recommended HTTP security headers on every
// response, as specified in Section 3.2.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:;")
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer-when-downgrade")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}
