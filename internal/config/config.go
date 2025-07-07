package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	App      AppConfig
}

type ServerConfig struct {
	Port         string
	Mode         string // gin mode: debug, release, test
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type LoggerConfig struct {
	Level       string
	Environment string // development or production
}

type AppConfig struct {
	Name             string
	Version          string
	YahooAPIBaseURL  string
	YahooAPITimeout  time.Duration
	DefaultDataLimit int
	MaxDataLimit     int
	CacheTTL         time.Duration
	KratosPublicURL  string
	KratosAdminURL   string
}

// Load reads configuration from file and environment
func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults()

	// Auto read environment variables
	viper.AutomaticEnv()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	config := &Config{
		Server: ServerConfig{
			Port:         viper.GetString("PORT"),
			Mode:         viper.GetString("GIN_MODE"),
			ReadTimeout:  viper.GetDuration("SERVER_READ_TIMEOUT"),
			WriteTimeout: viper.GetDuration("SERVER_WRITE_TIMEOUT"),
		},
		Database: DatabaseConfig{
			URL:             viper.GetString("DATABASE_URL"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: viper.GetDuration("DB_CONN_MAX_LIFETIME"),
			ConnMaxIdleTime: viper.GetDuration("DB_CONN_MAX_IDLE_TIME"),
		},
		Logger: LoggerConfig{
			Level:       viper.GetString("LOG_LEVEL"),
			Environment: viper.GetString("ENVIRONMENT"),
		},
		App: AppConfig{
			Name:             "proto-trading-service",
			Version:          viper.GetString("APP_VERSION"),
			YahooAPIBaseURL:  viper.GetString("YAHOO_API_BASE_URL"),
			YahooAPITimeout:  viper.GetDuration("YAHOO_API_TIMEOUT"),
			DefaultDataLimit: viper.GetInt("DEFAULT_DATA_LIMIT"),
			MaxDataLimit:     viper.GetInt("MAX_DATA_LIMIT"),
			CacheTTL:         viper.GetDuration("CACHE_TTL"),
			KratosPublicURL:  viper.GetString("KRATOS_PUBLIC_URL"),
			KratosAdminURL:   viper.GetString("KRATOS_ADMIN_URL"),
		},
	}

	return config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("GIN_MODE", "debug")
	viper.SetDefault("SERVER_READ_TIMEOUT", 15*time.Second)
	viper.SetDefault("SERVER_WRITE_TIMEOUT", 15*time.Second)

	// Database defaults
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	viper.SetDefault("DB_CONN_MAX_IDLE_TIME", 10*time.Minute)

	// Logger defaults
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ENVIRONMENT", "development")

	// App defaults
	viper.SetDefault("APP_VERSION", "1.0.0")
	viper.SetDefault("YAHOO_API_BASE_URL", "https://query1.finance.yahoo.com/v8/finance")
	viper.SetDefault("YAHOO_API_TIMEOUT", 30*time.Second)
	viper.SetDefault("DEFAULT_DATA_LIMIT", 30)
	viper.SetDefault("MAX_DATA_LIMIT", 1000)
	viper.SetDefault("CACHE_TTL", 5*time.Minute)
	viper.SetDefault("KRATOS_PUBLIC_URL", "http://localhost:4433")
	viper.SetDefault("KRATOS_ADMIN_URL", "http://localhost:4434")
}
