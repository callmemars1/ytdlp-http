package middlewares

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"github.com/callmemars1/ytdlp-http/internal/configurations"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthMiddleware struct {
	config *configurations.AuthConfig
	logger *zap.Logger
}

func NewAuthMiddleware(config *configurations.AuthConfig, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
		logger: logger,
	}
}

func (a *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.config.Enabled {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			a.logger.Warn("Missing Authorization header", zap.String("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			a.logger.Warn("Invalid Authorization header format", 
				zap.String("ip", c.ClientIP()), 
				zap.String("header", authHeader))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized", 
				"message": "Authorization header must be in format: Bearer <token>",
			})
			c.Abort()
			return
		}

		providedKey := parts[1]
		if !a.validateAPIKey(providedKey) {
			a.logger.Warn("Invalid API key provided", 
				zap.String("ip", c.ClientIP()),
				zap.String("provided_key_hash", a.hashKey(providedKey)))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid API key",
			})
			c.Abort()
			return
		}

		a.logger.Debug("API key validated successfully", zap.String("ip", c.ClientIP()))
		c.Next()
	}
}

func (a *AuthMiddleware) validateAPIKey(providedKey string) bool {
	providedHash := a.hashKey(providedKey)
	expectedHash := a.config.APIKey
	
	return providedHash == expectedHash
}

func (a *AuthMiddleware) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}