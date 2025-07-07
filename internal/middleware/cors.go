package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ridhomain/proto-trading-service/pkg/logger"
	"go.uber.org/zap"
)

// CORS returns a middleware that configures CORS for production
func CORS() gin.HandlerFunc {
	// Get allowed origins from environment or use defaults
	originsEnv := os.Getenv("CORS_ORIGINS")
	var allowedOrigins []string

	if originsEnv != "" {
		allowedOrigins = strings.Split(originsEnv, ",")
		// Trim whitespace
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	} else {
		// Default origins for development
		allowedOrigins = []string{
			"http://localhost:8000", // Frontend
			"http://127.0.0.1:8000", // Frontend alternative
			"http://localhost:4455", // Kratos UI
			"http://127.0.0.1:4455", // Kratos UI alternative
			"http://localhost:8080", // API (for testing)
		}
	}

	logger.Info("CORS configuration",
		zap.Strings("allowed_origins", allowedOrigins),
	)

	config := cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
			"HEAD",
		},
		AllowHeaders: []string{
			// Standard headers
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-HTTP-Method-Override",

			// Authentication headers
			"Cookie",
			"X-Session-Token",
			"X-CSRF-Token",

			// Custom headers
			"X-Request-ID",
			"X-User-ID",
			"X-API-Key",

			// Content negotiation
			"Accept-Language",
			"Accept-Encoding",
			"Cache-Control",
			"Pragma",
		},
		ExposeHeaders: []string{
			// Response headers that frontend can access
			"Content-Length",
			"Content-Type",
			"Content-Disposition",
			"Set-Cookie",
			"X-Session-Token",
			"X-Request-ID",
			"X-User-ID",
			"X-Session-ID",
			"Location",
			"X-Total-Count", // For pagination
			"X-Rate-Limit",  // For rate limiting info
		},
		AllowCredentials: true, // Essential for cookie-based auth
		MaxAge:           12 * time.Hour,

		// Allow all origins in development, be more strict in production
		AllowWildcard: false,

		// Custom function to check origins dynamically
		AllowOriginFunc: func(origin string) bool {
			// In development, be more permissive
			if gin.Mode() == gin.DebugMode {
				// Allow localhost on any port
				if strings.HasPrefix(origin, "http://localhost:") ||
					strings.HasPrefix(origin, "http://127.0.0.1:") {
					return true
				}
			}

			// Always check against explicitly allowed origins
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}

			logger.Warn("CORS: Origin not allowed",
				zap.String("origin", origin),
				zap.Strings("allowed_origins", allowedOrigins),
			)
			return false
		},
	}

	// Create the CORS middleware
	corsMiddleware := cors.New(config)

	// Wrap with logging for debugging
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		method := c.Request.Method

		// Log CORS requests for debugging
		if origin != "" && method == "OPTIONS" {
			logger.Debug("CORS preflight request",
				zap.String("origin", origin),
				zap.String("method", c.Request.Header.Get("Access-Control-Request-Method")),
				zap.String("headers", c.Request.Header.Get("Access-Control-Request-Headers")),
			)
		}

		// Apply CORS middleware
		corsMiddleware(c)

		// Log CORS response headers for debugging
		if origin != "" {
			logger.Debug("CORS response",
				zap.String("origin", origin),
				zap.String("allow_origin", c.Writer.Header().Get("Access-Control-Allow-Origin")),
				zap.String("allow_credentials", c.Writer.Header().Get("Access-Control-Allow-Credentials")),
				zap.String("expose_headers", c.Writer.Header().Get("Access-Control-Expose-Headers")),
			)
		}
	}
}

// CORSPreflightHandler handles complex CORS preflight requests
func CORSPreflightHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			origin := c.Request.Header.Get("Origin")

			// Log preflight details
			logger.Debug("Handling CORS preflight",
				zap.String("origin", origin),
				zap.String("requested_method", c.Request.Header.Get("Access-Control-Request-Method")),
				zap.String("requested_headers", c.Request.Header.Get("Access-Control-Request-Headers")),
			)

			c.Header("Access-Control-Max-Age", "86400") // 24 hours
			c.Status(204)
			c.Abort()
			return
		}
		c.Next()
	}
}

// SecurityHeaders adds security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Only add HSTS in production with HTTPS
		if gin.Mode() == gin.ReleaseMode && c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Content Security Policy (adjust based on your needs)
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' http://localhost:* ws://localhost:*; " +
			"frame-ancestors 'none'"
		c.Header("Content-Security-Policy", csp)

		c.Next()
	}
}
