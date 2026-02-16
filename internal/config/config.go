package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Validation errors
var (
	ErrInvalidJWTSecret  = errors.New("JWT_SECRET must be set and not use default value in production")
	ErrInvalidServerPort = errors.New("SERVER_PORT must be a valid port number (1-65535)")
	ErrInvalidDBPort     = errors.New("DB_PORT must be a valid port number (1-65535)")
	ErrInvalidLogLevel   = errors.New("LOG_LEVEL must be one of: debug, info, warn, error")
	ErrInvalidLogFormat  = errors.New("LOG_FORMAT must be one of: json, text")
	ErrInvalidCORSOrigin = errors.New("CORS_ALLOW_ORIGINS contains invalid URL")
	ErrMissingDBConfig   = errors.New("database configuration incomplete: DB_HOST, DB_USER, DB_PASSWORD, and DB_NAME are required")
	ErrWeakAdminPassword = errors.New("SEED_ADMIN_PASSWORD must be at least 8 characters when SEED_ADMIN_EMAIL is set")
	ErrInvalidAdminEmail = errors.New("SEED_ADMIN_EMAIL must be a valid email address")
)

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Server
	ServerPort string
	JWTSecret  string

	// RBAC
	RBACModelPath string

	// Seeding
	SeedAdminEmail    string
	SeedAdminPassword string
	SeedAdminName     string

	// Government Funding Seeding
	GovernmentFundingSeedPath  string
	GovernmentFundingSeedState string

	// Test Data Seeding
	SeedTestData bool

	// CORS
	CORSAllowOrigins     []string
	CORSAllowCredentials bool

	// Logging
	LogLevel  string
	LogFormat string

	// Environment
	Environment string // "development", "staging", "production"

	// Rate Limiting
	LoginRateLimitPerMinute int // 0 = disabled, default = 5
	APIRateLimitPerMinute   int // 0 = disabled, default = 60

	// Database Connection Pool
	DBMaxIdleConns   int // default = 10
	DBMaxOpenConns   int // default = 100
	DBConnMaxLifeMin int // connection max lifetime in minutes, default = 60
	DBConnMaxIdleMin int // idle connection max lifetime in minutes, default = 10

	// Database SSL
	DBSSLMode string // default = "disable", options: disable, require, verify-ca, verify-full

	// Trusted Proxies
	TrustedProxies []string // TRUSTED_PROXIES env var, comma-separated

	// Security
	SecureCookies bool // SECURE_COOKIES env var, default = true in production/staging
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == ""
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	var errs []error

	// JWT Secret validation - critical in production
	if c.JWTSecret == "default-secret-key" || c.JWTSecret == "" {
		if c.IsProduction() {
			errs = append(errs, ErrInvalidJWTSecret)
		}
	}
	if len(c.JWTSecret) < 32 && c.IsProduction() {
		errs = append(errs, fmt.Errorf("JWT_SECRET should be at least 32 characters for security"))
	}

	// Port validation
	if !isValidPort(c.ServerPort) {
		errs = append(errs, ErrInvalidServerPort)
	}
	if !isValidPort(c.DBPort) {
		errs = append(errs, ErrInvalidDBPort)
	}

	// Database config validation
	if c.DBHost == "" || c.DBUser == "" || c.DBPassword == "" || c.DBName == "" {
		errs = append(errs, ErrMissingDBConfig)
	}

	// Log level validation
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		errs = append(errs, ErrInvalidLogLevel)
	}

	// Log format validation
	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[strings.ToLower(c.LogFormat)] {
		errs = append(errs, ErrInvalidLogFormat)
	}

	// CORS origins validation
	for _, origin := range c.CORSAllowOrigins {
		if origin == "*" {
			if c.CORSAllowCredentials && c.IsProduction() {
				errs = append(errs, fmt.Errorf("CORS: wildcard origin with credentials is not allowed in production"))
			}
			continue
		}
		if _, err := url.ParseRequestURI(origin); err != nil {
			errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidCORSOrigin, origin))
		}
	}

	// Database SSL mode validation
	validSSLModes := map[string]bool{"disable": true, "require": true, "verify-ca": true, "verify-full": true}
	if !validSSLModes[c.DBSSLMode] {
		errs = append(errs, fmt.Errorf("DB_SSLMODE must be one of: disable, require, verify-ca, verify-full"))
	}
	if c.DBSSLMode == "disable" && c.IsProduction() {
		slog.Warn("Database SSL is disabled in production — consider using 'require' or 'verify-full'")
	}

	// Admin seeding validation
	if c.SeedAdminEmail != "" {
		if !strings.Contains(c.SeedAdminEmail, "@") {
			errs = append(errs, ErrInvalidAdminEmail)
		}
		if len(c.SeedAdminPassword) < 8 {
			errs = append(errs, ErrWeakAdminPassword)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %w", errors.Join(errs...))
	}

	return nil
}

// isValidPort checks if a string is a valid port number
func isValidPort(port string) bool {
	p, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return p >= 1 && p <= 65535
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	corsOrigins := getEnv("CORS_ALLOW_ORIGINS", "http://localhost:5173,http://localhost:8080")
	origins := strings.Split(corsOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	var trustedProxies []string
	if tp := getEnv("TRUSTED_PROXIES", ""); tp != "" {
		for _, p := range strings.Split(tp, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				trustedProxies = append(trustedProxies, p)
			}
		}
	}

	cfg := &Config{
		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "kitamanager"),
		DBPassword: getEnv("DB_PASSWORD", "kitamanager"),
		DBName:     getEnv("DB_NAME", "kitamanager"),

		// Server
		ServerPort: getEnv("SERVER_PORT", "8080"),
		JWTSecret:  getEnv("JWT_SECRET", "default-secret-key"),

		// RBAC
		RBACModelPath: getEnv("RBAC_MODEL_PATH", "configs/rbac_model.conf"),

		// Seeding
		SeedAdminEmail:    getEnv("SEED_ADMIN_EMAIL", ""),
		SeedAdminPassword: getEnv("SEED_ADMIN_PASSWORD", ""),
		SeedAdminName:     getEnv("SEED_ADMIN_NAME", "admin"),

		// Government Funding Seeding
		GovernmentFundingSeedPath:  getEnv("GOVERNMENT_FUNDING_SEED_PATH", ""),
		GovernmentFundingSeedState: getEnv("GOVERNMENT_FUNDING_SEED_STATE", "berlin"),

		// Test Data Seeding
		SeedTestData: getEnv("SEED_TEST_DATA", "false") == "true",

		// CORS
		CORSAllowOrigins:     origins,
		CORSAllowCredentials: getEnv("CORS_ALLOW_CREDENTIALS", "true") == "true",

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),

		// Rate Limiting
		LoginRateLimitPerMinute: getEnvInt("LOGIN_RATE_LIMIT_PER_MINUTE", 5),
		APIRateLimitPerMinute:   getEnvInt("API_RATE_LIMIT_PER_MINUTE", 60),

		// Database Connection Pool
		DBMaxIdleConns:   getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxOpenConns:   getEnvInt("DB_MAX_OPEN_CONNS", 100),
		DBConnMaxLifeMin: getEnvInt("DB_CONN_MAX_LIFE_MIN", 60),
		DBConnMaxIdleMin: getEnvInt("DB_CONN_MAX_IDLE_MIN", 10),

		// Database SSL
		DBSSLMode: getEnv("DB_SSLMODE", "disable"),

		// Trusted Proxies
		TrustedProxies: trustedProxies,
	}

	// SecureCookies: explicit env var wins, otherwise default to non-development
	if v := os.Getenv("SECURE_COOKIES"); v != "" {
		cfg.SecureCookies = v == "true"
	} else {
		cfg.SecureCookies = !cfg.IsDevelopment()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
