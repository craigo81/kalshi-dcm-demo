// Package config provides configuration management for the DCM demo.
// Supports multiple exchange backends (Kalshi, Crypto.com) per modular design.
package config

import (
	"os"
	"strconv"
	"time"
)

// Exchange represents supported trading venues
type Exchange string

const (
	ExchangeKalshi   Exchange = "kalshi"
	ExchangeCryptoCom Exchange = "crypto_com"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	Port            string
	Environment     string // development, staging, production
	TLSEnabled      bool
	TLSCertFile     string
	TLSKeyFile      string

	// Active exchange configuration
	ActiveExchange  Exchange

	// Kalshi API settings
	KalshiBaseURL       string
	KalshiAPIKey        string // For authenticated endpoints (demo: empty)
	KalshiAPISecret     string
	KalshiRateLimit     int           // Requests per second
	KalshiTimeout       time.Duration
	KalshiRetryAttempts int
	KalshiRetryDelay    time.Duration

	// Crypto.com API settings (for future transition)
	// CP 2: Compliance - Modular design for exchange switching
	CryptoComBaseURL    string
	CryptoComAPIKey     string
	CryptoComAPISecret  string
	CryptoComRateLimit  int
	CryptoComTimeout    time.Duration

	// Persistence settings
	// CP 18: Recordkeeping - 5-year retention simulation
	DataDir             string
	EnablePersistence   bool
	AuditRetentionDays  int

	// WebSocket settings
	WSPingInterval      time.Duration
	WSPongTimeout       time.Duration
	WSMaxMessageSize    int64

	// Compliance settings
	// CP 5: Position Limits
	DefaultPositionLimit float64
	MaxPositionLimit     float64
	// CP 11: Financial Integrity
	MinCollateralRatio   float64 // 1.0 = 100%
	// CP 4: Market Disruption Prevention
	RateLimitPerUser     int // Orders per minute
	AnomalyThreshold     float64

	// CORS
	AllowedOrigins []string
}

// Load creates configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		// Server
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		TLSEnabled:  getEnvBool("TLS_ENABLED", false),
		TLSCertFile: getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  getEnv("TLS_KEY_FILE", ""),

		// Exchange selection
		ActiveExchange: Exchange(getEnv("ACTIVE_EXCHANGE", "kalshi")),

		// Kalshi
		KalshiBaseURL:       getEnv("KALSHI_BASE_URL", "https://api.elections.kalshi.com/trade-api/v2"),
		KalshiAPIKey:        getEnv("KALSHI_API_KEY", ""),
		KalshiAPISecret:     getEnv("KALSHI_API_SECRET", ""),
		KalshiRateLimit:     getEnvInt("KALSHI_RATE_LIMIT", 10),
		KalshiTimeout:       getEnvDuration("KALSHI_TIMEOUT", 30*time.Second),
		KalshiRetryAttempts: getEnvInt("KALSHI_RETRY_ATTEMPTS", 3),
		KalshiRetryDelay:    getEnvDuration("KALSHI_RETRY_DELAY", 1*time.Second),

		// Crypto.com (UAT placeholder)
		CryptoComBaseURL:   getEnv("CRYPTOCOM_BASE_URL", "https://uat-api.3702.3ona.co/v1/derivatives"),
		CryptoComAPIKey:    getEnv("CRYPTOCOM_API_KEY", ""),
		CryptoComAPISecret: getEnv("CRYPTOCOM_API_SECRET", ""),
		CryptoComRateLimit: getEnvInt("CRYPTOCOM_RATE_LIMIT", 10),
		CryptoComTimeout:   getEnvDuration("CRYPTOCOM_TIMEOUT", 30*time.Second),

		// Persistence
		DataDir:            getEnv("DATA_DIR", "./data"),
		EnablePersistence:  getEnvBool("ENABLE_PERSISTENCE", true),
		AuditRetentionDays: getEnvInt("AUDIT_RETENTION_DAYS", 1825), // 5 years

		// WebSocket
		WSPingInterval:   getEnvDuration("WS_PING_INTERVAL", 30*time.Second),
		WSPongTimeout:    getEnvDuration("WS_PONG_TIMEOUT", 60*time.Second),
		WSMaxMessageSize: int64(getEnvInt("WS_MAX_MESSAGE_SIZE", 512*1024)),

		// Compliance
		DefaultPositionLimit: getEnvFloat("DEFAULT_POSITION_LIMIT", 25000.0),
		MaxPositionLimit:     getEnvFloat("MAX_POSITION_LIMIT", 250000.0),
		MinCollateralRatio:   getEnvFloat("MIN_COLLATERAL_RATIO", 1.0),
		RateLimitPerUser:     getEnvInt("RATE_LIMIT_PER_USER", 60),
		AnomalyThreshold:     getEnvFloat("ANOMALY_THRESHOLD", 0.1),

		// CORS
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001", // Surveillance app
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
		},
	}
}

// GetExchangeURL returns the base URL for the active exchange
func (c *Config) GetExchangeURL() string {
	switch c.ActiveExchange {
	case ExchangeCryptoCom:
		return c.CryptoComBaseURL
	default:
		return c.KalshiBaseURL
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
