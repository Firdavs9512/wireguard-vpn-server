package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"wireguard-vpn-client-creater/pkg/config"
)

// AuthMiddleware - API uchun token-based autentifikatsiya middleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Token headerdan olish
		token := c.GetHeader("Authorization")

		// Token tekshirish
		if token != "Bearer "+config.Config.API.Token {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Noto'g'ri token"})
			c.Abort()
			return
		}

		// Keyingi middleware ga o'tish
		c.Next()
	}
}

// CORSMiddleware - CORS uchun middleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
