package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ridhomain/proto-trading-service/internal/middleware"
	"go.uber.org/zap"
)

// AuthStatus returns authentication status (public endpoint)
func (h *Handler) AuthStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if userID == "" {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"login_url":     "http://localhost:4455/login",
			"kratos_ui":     "http://localhost:4455",
		})
		return
	}

	email := middleware.GetUserEmail(c)
	role := middleware.GetUserRole(c)
	sessionID := middleware.GetSessionID(c)

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user": gin.H{
			"id":    userID,
			"email": email,
			"role":  role,
		},
		"session_id": sessionID,
		"logout_url": "http://localhost:4433/self-service/logout/browser",
	})
}

// GetCurrentUser returns the current authenticated user
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetUserEmail(c)
	role := middleware.GetUserRole(c)
	sessionID := middleware.GetSessionID(c)

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
		"user": gin.H{
			"id":    userID,
			"email": email,
			"role":  role,
		},
		"session_id":    sessionID,
		"preferences":   prefs,
		"authenticated": true,
	})
}

// GetLoginURL returns the Kratos login URL
func (h *Handler) GetLoginURL(c *gin.Context) {
	returnTo := c.Query("return_to")
	if returnTo == "" {
		returnTo = "http://localhost:8000/dashboard"
	}

	c.JSON(http.StatusOK, gin.H{
		"login_url":  "http://localhost:4455/login",
		"kratos_api": "http://localhost:4433/self-service/login/browser",
		"return_to":  returnTo,
	})
}

// Logout endpoint (provides logout information)
func (h *Handler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	sessionID := middleware.GetSessionID(c)

	h.logger.Info("User logout",
		zap.String("user_id", userID),
		zap.String("session_id", sessionID),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":    "To complete logout, visit the logout URL",
		"logout_url": "http://localhost:4433/self-service/logout/browser",
		"redirect":   "http://localhost:8000/",
	})
}

// GetUserPreferences returns user preferences
func (h *Handler) GetUserPreferences(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	prefs, err := h.userService.GetPreferences(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user preferences",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get preferences",
		})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// UpdateUserPreferences updates user preferences
func (h *Handler) UpdateUserPreferences(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Validate allowed fields
	allowedFields := map[string]bool{
		"default_source":   true,
		"selected_symbols": true,
		"watchlist":        true,
	}

	for field := range updates {
		if !allowedFields[field] {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid field",
				Message: "Field '" + field + "' is not allowed",
			})
			return
		}
	}

	err := h.userService.UpdatePreferences(ctx, userID, updates)
	if err != nil {
		h.logger.Error("Failed to update user preferences",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to update preferences",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preferences updated successfully",
	})
}

// AddToWatchlist adds a symbol to user's watchlist
func (h *Handler) AddToWatchlist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	symbol := c.Param("symbol")
	ctx := c.Request.Context()

	if symbol == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Symbol is required",
		})
		return
	}

	err := h.userService.AddToWatchlist(ctx, userID, symbol)
	if err != nil {
		h.logger.Error("Failed to add to watchlist",
			zap.String("user_id", userID),
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to add to watchlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Symbol added to watchlist",
		"symbol":  symbol,
	})
}

// RemoveFromWatchlist removes a symbol from user's watchlist
func (h *Handler) RemoveFromWatchlist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	symbol := c.Param("symbol")
	ctx := c.Request.Context()

	if symbol == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Symbol is required",
		})
		return
	}

	err := h.userService.RemoveFromWatchlist(ctx, userID, symbol)
	if err != nil {
		h.logger.Error("Failed to remove from watchlist",
			zap.String("user_id", userID),
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to remove from watchlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Symbol removed from watchlist",
		"symbol":  symbol,
	})
}
