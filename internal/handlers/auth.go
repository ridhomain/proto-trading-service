package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ridhomain/proto-trading-service/internal/middleware"
	"go.uber.org/zap"
)

// GetCurrentUser returns the current authenticated user
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetUserEmail(c)

	// Get or create user preferences
	ctx := c.Request.Context()
	prefs, err := h.userService.GetOrCreatePreferences(ctx, userID, email)
	if err != nil {
		h.logger.Error("Failed to get user preferences",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get user preferences",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     userID,
		"email":       email,
		"preferences": prefs,
	})
}

// Logout endpoint (Kratos handles the actual logout)
func (h *Handler) Logout(c *gin.Context) {
	// Clear any server-side session if needed
	c.JSON(http.StatusOK, gin.H{
		"message":    "Please redirect to Kratos logout URL",
		"logout_url": "http://localhost:4433/self-service/logout/browser",
	})
}
