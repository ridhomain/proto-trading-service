package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ridhomain/proto-trading-service/pkg/logger"
	"go.uber.org/zap"
)

type KratosSession struct {
	ID       string `json:"id"`
	Active   bool   `json:"active"`
	Identity struct {
		ID     string                 `json:"id"`
		Traits map[string]interface{} `json:"traits"`
		State  string                 `json:"state"`
	} `json:"identity"`
	AuthenticatedAt time.Time `json:"authenticated_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

type AuthConfig struct {
	KratosInternalURL string // For service-to-service calls (http://kratos:4433)
	KratosBrowserURL  string // For browser redirects (http://localhost:4433)
}

var authConfig *AuthConfig

// InitAuthConfig initializes the authentication configuration
func InitAuthConfig(internalURL, browserURL string) {
	authConfig = &AuthConfig{
		KratosInternalURL: internalURL,
		KratosBrowserURL:  browserURL,
	}
}

// AuthRequired validates the session with Ory Kratos
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if authConfig == nil {
			logger.Error("Auth config not initialized")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication service not configured",
			})
			c.Abort()
			return
		}

		// Extract session token
		sessionToken := extractSessionToken(c)
		if sessionToken == "" {
			logger.Warn("No session token provided",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("user_agent", c.Request.UserAgent()),
			)

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "Authentication required",
				"login_url": authConfig.KratosBrowserURL + "/self-service/login/browser",
				"kratos_ui": "http://localhost:4455/login",
			})
			c.Abort()
			return
		}

		// Validate session with Kratos
		session, err := validateSession(sessionToken)
		if err != nil {
			logger.Error("Session validation failed",
				zap.Error(err),
				zap.String("token_hint", maskToken(sessionToken)),
				zap.String("path", c.Request.URL.Path),
			)

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "Invalid or expired session",
				"login_url": authConfig.KratosBrowserURL + "/self-service/login/browser",
				"kratos_ui": "http://localhost:4455/login",
			})
			c.Abort()
			return
		}

		if !session.Active {
			logger.Warn("Inactive session",
				zap.String("session_id", session.ID),
				zap.String("identity_id", session.Identity.ID),
			)

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "Session inactive",
				"login_url": authConfig.KratosBrowserURL + "/self-service/login/browser",
				"kratos_ui": "http://localhost:4455/login",
			})
			c.Abort()
			return
		}

		// Check if session is expired
		if time.Now().After(session.ExpiresAt) {
			logger.Warn("Expired session",
				zap.String("session_id", session.ID),
				zap.Time("expires_at", session.ExpiresAt),
			)

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":     "Session expired",
				"login_url": authConfig.KratosBrowserURL + "/self-service/login/browser",
				"kratos_ui": "http://localhost:4455/login",
			})
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("user_id", session.Identity.ID)
		c.Set("user_traits", session.Identity.Traits)
		c.Set("session", session)
		c.Set("session_id", session.ID)

		// Add user info to response headers for debugging
		c.Header("X-User-ID", session.Identity.ID)
		c.Header("X-Session-ID", session.ID)

		logger.Debug("Authentication successful",
			zap.String("user_id", session.Identity.ID),
			zap.String("session_id", session.ID),
			zap.String("path", c.Request.URL.Path),
		)

		c.Next()
	}
}

// RoleRequired checks if user has required role
func RoleRequired(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		traits, exists := c.Get("user_traits")
		if !exists {
			logger.Error("No user traits found in context")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied - no user context",
			})
			c.Abort()
			return
		}

		traitsMap, ok := traits.(map[string]interface{})
		if !ok {
			logger.Error("Invalid user traits format")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied - invalid user data",
			})
			c.Abort()
			return
		}

		role, hasRole := traitsMap["role"].(string)
		if !hasRole {
			role = "trader" // Default role
		}

		if role != requiredRole {
			userID := GetUserID(c)
			logger.Warn("Insufficient permissions",
				zap.String("user_id", userID),
				zap.String("user_role", role),
				zap.String("required_role", requiredRole),
				zap.String("path", c.Request.URL.Path),
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Insufficient permissions",
				"required_role": requiredRole,
				"user_role":     role,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractSessionToken gets the session token from various sources
func extractSessionToken(c *gin.Context) string {
	// 1. Try cookie first (primary method for browsers)
	if cookie, err := c.Cookie("ory_kratos_session"); err == nil && cookie != "" {
		return cookie
	}

	// 2. Try Authorization header (for API clients)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// Support both "Bearer token" and "Session token" formats
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		if strings.HasPrefix(authHeader, "Session ") {
			return strings.TrimPrefix(authHeader, "Session ")
		}
	}

	// 3. Try X-Session-Token header
	if token := c.GetHeader("X-Session-Token"); token != "" {
		return token
	}

	return ""
}

// validateSession checks the session with Kratos internal API
func validateSession(sessionToken string) (*KratosSession, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Use internal Kratos URL for service-to-service communication
	url := authConfig.KratosInternalURL + "/sessions/whoami"

	// DEBUG: Log the exact URL being used
	logger.Info("Session validation debug",
		zap.String("kratos_url", url),
		zap.String("kratos_internal_url", authConfig.KratosInternalURL),
		zap.String("kratos_browser_url", authConfig.KratosBrowserURL),
		zap.String("token_hint", maskToken(sessionToken)),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set session token in multiple ways to ensure compatibility
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	req.Header.Set("X-Session-Token", sessionToken)
	req.AddCookie(&http.Cookie{
		Name:  "ory_kratos_session",
		Value: sessionToken,
	})

	// Add user agent
	req.Header.Set("User-Agent", "proto-trading-service/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error contacting Kratos: %w", err)
	}
	defer resp.Body.Close()

	// DEBUG: Log response details
	logger.Info("Kratos response debug",
		zap.String("url", url),
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status),
	)

	switch resp.StatusCode {
	case http.StatusOK:
		var session KratosSession
		if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
			return nil, fmt.Errorf("failed to decode session response: %w", err)
		}
		return &session, nil

	case http.StatusUnauthorized:
		return nil, fmt.Errorf("unauthorized: invalid or expired session")

	case http.StatusForbidden:
		return nil, fmt.Errorf("forbidden: session validation failed")

	default:
		return nil, fmt.Errorf("unexpected response from Kratos: %d", resp.StatusCode)
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return ""
}

// GetUserEmail extracts user email from context
func GetUserEmail(c *gin.Context) string {
	if traits, exists := c.Get("user_traits"); exists {
		if traitsMap, ok := traits.(map[string]interface{}); ok {
			if email, ok := traitsMap["email"].(string); ok {
				return email
			}
		}
	}
	return ""
}

// GetUserRole extracts user role from context
func GetUserRole(c *gin.Context) string {
	if traits, exists := c.Get("user_traits"); exists {
		if traitsMap, ok := traits.(map[string]interface{}); ok {
			if role, ok := traitsMap["role"].(string); ok {
				return role
			}
		}
	}
	return "trader" // Default role
}

// GetSessionID extracts session ID from context
func GetSessionID(c *gin.Context) string {
	if sessionID, exists := c.Get("session_id"); exists {
		return sessionID.(string)
	}
	return ""
}

// maskToken masks the session token for logging (security)
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// OptionalAuth middleware that doesn't require authentication but adds user context if available
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if authConfig == nil {
			c.Next()
			return
		}

		sessionToken := extractSessionToken(c)
		if sessionToken == "" {
			c.Next()
			return
		}

		session, err := validateSession(sessionToken)
		if err != nil || !session.Active {
			// Don't fail, just continue without user context
			c.Next()
			return
		}

		// Add user context
		c.Set("user_id", session.Identity.ID)
		c.Set("user_traits", session.Identity.Traits)
		c.Set("session", session)
		c.Set("session_id", session.ID)

		c.Next()
	}
}
