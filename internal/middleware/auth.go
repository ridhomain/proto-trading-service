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
			logger.Error("Failed to validate session", zap.Error(err))
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
func extractSessionToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// Try X-Session-Token header
	if token := c.GetHeader("X-Session-Token"); token != "" {
		return token
	}

	// Try cookie
	if cookie, err := c.Cookie("ory_kratos_session"); err == nil {
		return cookie
	}

	return ""
}

// validateSession checks the session with Kratos
func validateSession(kratosURL, sessionToken string) (*KratosSession, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", kratosURL+"/sessions/whoami", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+sessionToken)
	req.Header.Set("X-Session-Token", sessionToken)
	req.AddCookie(&http.Cookie{
		Name:  "ory_kratos_session",
		Value: sessionToken,
	})

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid session: status %d", resp.StatusCode)
	}

	var session KratosSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	return &session, nil
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
