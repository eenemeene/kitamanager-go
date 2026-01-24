package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
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

	// CORS
	CORSAllowOrigins     []string
	CORSAllowCredentials bool

	// Logging
	LogLevel  string
	LogFormat string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	corsOrigins := getEnv("CORS_ALLOW_ORIGINS", "http://localhost:5173,http://localhost:8080")
	origins := strings.Split(corsOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return &Config{
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

		// CORS
		CORSAllowOrigins:     origins,
		CORSAllowCredentials: getEnv("CORS_ALLOW_CREDENTIALS", "true") == "true",

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
