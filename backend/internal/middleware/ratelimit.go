package middleware

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ipBucket tracks request counts in a sliding window per IP.
type ipBucket struct {
	count    int
	windowStart time.Time
	mu       sync.Mutex
}

// rateLimiter holds per-IP state.
type rateLimiter struct {
	buckets  sync.Map
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window}
}

func (rl *rateLimiter) allow(ip string) bool {
	val, _ := rl.buckets.LoadOrStore(ip, &ipBucket{windowStart: time.Now()})
	bucket := val.(*ipBucket)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	if time.Since(bucket.windowStart) > rl.window {
		bucket.count = 0
		bucket.windowStart = time.Now()
	}
	bucket.count++
	return bucket.count <= rl.limit
}

// authLimiter: 5 requests per IP per 15 minutes for auth endpoints
var authLimiter = newRateLimiter(5, 15*time.Minute)

// apiLimiter: 200 requests per IP per minute for all other endpoints
var apiLimiter = newRateLimiter(200, time.Minute)

// RateLimitAuth applies a strict 5/15min per-IP rate limit for auth endpoints.
func RateLimitAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if os.Getenv("APP_ENV") == "development" {
			c.Next()
			return
		}
		if !authLimiter.allow(c.ClientIP()) {
			c.Header("Retry-After", "900")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many auth attempts. Try again in 15 minutes.",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitAPI applies a 200/min per-IP rate limit for general API endpoints.
func RateLimitAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		if os.Getenv("APP_ENV") == "development" {
			c.Next()
			return
		}
		if !apiLimiter.allow(c.ClientIP()) {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Rate limit exceeded. Please slow down.",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
