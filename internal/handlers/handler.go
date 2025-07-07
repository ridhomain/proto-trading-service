package handlers

import (
	"github.com/ridhomain/proto-trading-service/internal/services"
	"github.com/ridhomain/proto-trading-service/pkg/logger"

	"go.uber.org/zap"
)

// Handler holds all handler dependencies
type Handler struct {
	marketService *services.MarketService
	userService   *services.UserService
	logger        *zap.Logger
}

// NewHandler creates a new handler with all dependencies
func NewHandler(marketService *services.MarketService, userService *services.UserService) *Handler {
	return &Handler{
		marketService: marketService,
		userService:   userService,
		logger:        logger.With(zap.String("component", "handler")),
	}
}

// Common response types
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
