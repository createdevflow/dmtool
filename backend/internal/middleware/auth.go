package middleware

import (
	"crypto/rsa"
	"net/http"
	"strings"

	"backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// JWTAuth validates RS256 access tokens on every protected route.
// It extracts UserID, Email, and Role from the claims and stores them
// in the Gin context for downstream handlers to consume.
func JWTAuth(publicKey *rsa.PublicKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Authorization header is required"},
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid Authorization header format"},
			})
			c.Abort()
			return
		}

		claims, err := utils.ParseAccessToken(publicKey, parts[1])
		if err != nil {
			tokenPrefix := parts[1]
			if len(tokenPrefix) > 10 {
				tokenPrefix = tokenPrefix[:10] + "..."
			}
			log.Warn().
				Err(err).
				Str("token_prefix", tokenPrefix).
				Msg("JWT validation failed")

			
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"},
			})
			c.Abort()
			return
		}


		// Store claims in context — handlers read from here, never from request body
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}
