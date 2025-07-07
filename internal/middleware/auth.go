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
	} `json:"identity"`
}

// AuthRequired validates the session with Ory Kratos
func AuthRequired(kratosURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from header or cookie
		sessionToken := extractSessionToken(c)
		if sessionToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "No session token provided",
			})
			c.Abort()
			return
		}

		// Validate session with Kratos
		session, err := validateSession(kratosURL, sessionToken)
		if err != nil {
			logger.Error("Failed to validate session", zap.Error(err), zap.String("token_prefix", sessionToken[:10]+"..."))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid session",
			})
			c.Abort()
			return
		}

		if !session.Active {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Session inactive",
			})
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("user_id", session.Identity.ID)
		c.Set("user_traits", session.Identity.Traits)
		c.Set("session", session)

		c.Next()
	}
}

// RoleRequired checks if user has required role
func RoleRequired(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		traits, exists := c.Get("user_traits")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "No user traits found",
			})
			c.Abort()
			return
		}

		traitsMap := traits.(map[string]interface{})
		role, ok := traitsMap["role"].(string)
		if !ok || role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractSessionToken gets the session token from various sources
// Prioritize cookie for browser-based access
func extractSessionToken(c *gin.Context) string {
	// Try cookie first for better browser experience
	if cookie, err := c.Cookie("ory_kratos_session"); err == nil && cookie != "" {
		return cookie
	}

	// Then try headers for API clients
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	if token := c.GetHeader("X-Session-Token"); token != "" {
		return token
	}

	return ""
}

// validateSession checks the session with Kratos
func validateSession(kratosURL, sessionToken string) (*KratosSession, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Create request
	req, err := http.NewRequest("GET", kratosURL+"/sessions/whoami", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Try all methods of session token delivery
	// 1. As Bearer token
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	// 2. As X-Session-Token
	req.Header.Set("X-Session-Token", sessionToken)
	// 3. As Cookie
	req.AddCookie(&http.Cookie{
		Name:  "ory_kratos_session",
		Value: sessionToken,
	})

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	// Handle different status codes
	switch resp.StatusCode {
	case http.StatusOK:
		var session KratosSession
		if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
			return nil, fmt.Errorf("decode error: %w", err)
		}
		return &session, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("unauthorized: invalid session")
	default:
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
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
		traitsMap := traits.(map[string]interface{})
		if email, ok := traitsMap["email"].(string); ok {
			return email
		}
	}
	return ""
}
