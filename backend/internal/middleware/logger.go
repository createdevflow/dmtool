package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/google/uuid"
)

// Logger returns a Gin middleware that writes structured JSON request logs
// using zerolog.  Each log line includes: timestamp, requestId, userId,
// method, path, status, latencyMs, clientIP.
func Logger() gin.HandlerFunc {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		userID, _ := c.Get("user_id")

		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		event.
			Str("requestId", requestID).
			Interface("userId", userID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", status).
			Int64("latencyMs", latency.Milliseconds()).
			Str("clientIP", c.ClientIP()).
			Msg("request")
	}
}
