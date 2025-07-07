package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ridhomain/proto-trading-service/internal/config"
	"github.com/ridhomain/proto-trading-service/internal/database"
	"github.com/ridhomain/proto-trading-service/internal/handlers"
	"github.com/ridhomain/proto-trading-service/internal/middleware"
	"github.com/ridhomain/proto-trading-service/internal/services"
	"github.com/ridhomain/proto-trading-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	if err := logger.Init(cfg.Logger.Environment, cfg.Logger.Level); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Log startup info
	logger.Info("Starting service",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.Logger.Environment),
	)

	// Initialize database
	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Initialize services
	marketService := services.NewMarketService(db)
	userService := services.NewUserService(db)

	// Initialize handlers
	handler := handlers.NewHandler(marketService, userService)

	// Setup Gin
	gin.SetMode(cfg.Server.Mode)
	router := setupRouter(handler, cfg)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server started",
			zap.String("port", cfg.Server.Port),
			zap.String("mode", cfg.Server.Mode),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func setupRouter(h *handlers.Handler, cfg *config.Config) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS())

	// Public endpoints (no auth required)
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)

	// Auth endpoints
	auth := r.Group("/auth")
	{
		auth.GET("/me", middleware.AuthRequired(cfg.App.KratosPublicURL), h.GetCurrentUser)
		auth.POST("/logout", h.Logout)
	}

	// API v1 routes (protected)
	v1 := r.Group("/api/v1")
	v1.Use(middleware.AuthRequired(cfg.App.KratosPublicURL))
	{
		// Market data endpoints
		market := v1.Group("/market-data")
		{
			market.GET("", h.GetMarketData)
			market.POST("", h.CreateMarketData)
			market.GET("/:symbol", h.GetMarketDataBySymbol)
			market.POST("/yahoo/:symbol", h.FetchYahooData)
			market.DELETE("/:symbol", middleware.RoleRequired("admin"), h.DeleteMarketData)
			market.POST("/bulk", h.BulkCreateMarketData)
		}

		// Upload endpoints
		upload := v1.Group("/upload")
		{
			upload.POST("/csv", h.UploadCSV)
		}
	}

	return r
}
