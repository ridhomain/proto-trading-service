package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that configures CORS
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8000", "http://localhost:8080", "http://127.0.0.1:4455"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Session-Token", "Cookie"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Set-Cookie"},
		AllowCredentials: true, // Critical for cookie-based auth
		MaxAge:           12 * time.Hour,
	})
}
