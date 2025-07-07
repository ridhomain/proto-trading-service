package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Health check endpoint
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "proto-trading-service",
	})
}

// Ready check endpoint - checks database connection
func (h *Handler) Ready(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.marketService.HealthCheck(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Database not ready",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"database": "connected",
	})
}
