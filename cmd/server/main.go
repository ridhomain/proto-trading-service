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
	logger.Info("Starting Proto Trading Service",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.Logger.Environment),
		zap.String("port", cfg.Server.Port),
		zap.String("gin_mode", cfg.Server.Mode),
	)

	// Initialize authentication configuration
	middleware.InitAuthConfig(cfg.App.KratosPublicURL, cfg.App.KratosBrowserURL)

	// Wait for dependencies to be ready
	if err := waitForDependencies(cfg); err != nil {
		logger.Fatal("Dependencies not ready", zap.Error(err))
	}

	// Initialize database
	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Run migrations (in production, this should be done separately)
	if cfg.Logger.Environment == "development" {
		logger.Info("Running database migrations...")
		if err := runMigrations(db); err != nil {
			logger.Warn("Migration warning", zap.Error(err))
		}
	}

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
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server starting",
			zap.String("port", cfg.Server.Port),
			zap.String("mode", cfg.Server.Mode),
			zap.String("kratos_internal", cfg.App.KratosPublicURL),
			zap.String("kratos_browser", cfg.App.KratosBrowserURL),
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

	logger.Info("Server exited gracefully")
}

func setupRouter(h *handlers.Handler, cfg *config.Config) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.RequestID())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())
	r.Use(middleware.CORSPreflightHandler())

	// Public endpoints (no auth required)
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)

	// Add a public endpoint to check auth status
	r.GET("/auth/status", middleware.OptionalAuth(), h.AuthStatus)

	// Auth endpoints
	auth := r.Group("/auth")
	{
		auth.GET("/me", middleware.AuthRequired(), h.GetCurrentUser)
		auth.POST("/logout", h.Logout)
		auth.GET("/login-url", h.GetLoginURL)
	}

	// API v1 routes (protected)
	v1 := r.Group("/api/v1")
	v1.Use(middleware.AuthRequired())
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

		// User preferences
		prefs := v1.Group("/preferences")
		{
			prefs.GET("", h.GetUserPreferences)
			prefs.PUT("", h.UpdateUserPreferences)
			prefs.POST("/watchlist/:symbol", h.AddToWatchlist)
			prefs.DELETE("/watchlist/:symbol", h.RemoveFromWatchlist)
		}
	}

	return r
}

func waitForDependencies(cfg *config.Config) error {
	logger.Info("Waiting for dependencies...")

	// Wait for database
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		db, err := database.New(&cfg.Database)
		if err == nil {
			if err := db.HealthCheck(context.Background()); err == nil {
				db.Close()
				logger.Info("Database connection established")
				break
			}
			db.Close()
		}

		if i == maxRetries-1 {
			return fmt.Errorf("database not ready after %d attempts", maxRetries)
		}

		logger.Info("Waiting for database...", zap.Int("attempt", i+1))
		time.Sleep(2 * time.Second)
	}

	// Wait for Kratos
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(cfg.App.KratosPublicURL + "/health/ready")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			logger.Info("Kratos service ready")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}

		if i == maxRetries-1 {
			return fmt.Errorf("kratos not ready after %d attempts", maxRetries)
		}

		logger.Info("Waiting for Kratos...", zap.Int("attempt", i+1))
		time.Sleep(2 * time.Second)
	}

	return nil
}

func runMigrations(db *database.DB) error {
	// In production, migrations should be run separately
	// This is just for development convenience
	ctx := context.Background()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS market_data (
			id BIGSERIAL PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL,
			date DATE NOT NULL,
			open DECIMAL(10, 2),
			high DECIMAL(10, 2),
			low DECIMAL(10, 2),
			close DECIMAL(10, 2),
			volume BIGINT,
			source VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(symbol, date, source)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_market_data_symbol_date ON market_data(symbol, date);`,
		`CREATE TABLE IF NOT EXISTS user_preferences (
			user_id VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			default_source VARCHAR(50) DEFAULT 'yahoo',
			selected_symbols TEXT[] DEFAULT '{}',
			watchlist TEXT[] DEFAULT '{}', 
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_user_preferences_email ON user_preferences(email);`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(ctx, migration); err != nil {
			return err
		}
	}

	logger.Info("Database migrations completed")
	return nil
}
