package config

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		want         string
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_CONFIG_UNSET",
			defaultValue: "default_value",
			setEnv:       false,
			want:         "default_value",
		},
		{
			name:         "returns env value when set",
			key:          "TEST_CONFIG_SET",
			defaultValue: "default_value",
			envValue:     "env_value",
			setEnv:       true,
			want:         "env_value",
		},
		{
			name:         "returns default when env is empty string",
			key:          "TEST_CONFIG_EMPTY",
			defaultValue: "default_value",
			envValue:     "",
			setEnv:       true,
			want:         "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Save original env values and restore after test
	envVars := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"SERVER_PORT", "JWT_SECRET", "RBAC_MODEL_PATH",
		"SEED_ADMIN_EMAIL", "SEED_ADMIN_PASSWORD", "SEED_ADMIN_NAME",
		"CORS_ALLOW_ORIGINS", "CORS_ALLOW_CREDENTIALS",
		"LOG_LEVEL", "LOG_FORMAT",
	}
	originalValues := make(map[string]string)
	for _, key := range envVars {
		originalValues[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, value := range originalValues {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("returns default values when env not set", func(t *testing.T) {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.DBHost != "localhost" {
			t.Errorf("DBHost = %q, want %q", cfg.DBHost, "localhost")
		}
		if cfg.DBPort != "5432" {
			t.Errorf("DBPort = %q, want %q", cfg.DBPort, "5432")
		}
		if cfg.DBUser != "kitamanager" {
			t.Errorf("DBUser = %q, want %q", cfg.DBUser, "kitamanager")
		}
		if cfg.DBPassword != "kitamanager" {
			t.Errorf("DBPassword = %q, want %q", cfg.DBPassword, "kitamanager")
		}
		if cfg.DBName != "kitamanager" {
			t.Errorf("DBName = %q, want %q", cfg.DBName, "kitamanager")
		}
		if cfg.ServerPort != "8080" {
			t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "8080")
		}
		if cfg.JWTSecret != "default-secret-key" {
			t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "default-secret-key")
		}
		if cfg.LogLevel != "info" {
			t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
		}
		if cfg.LogFormat != "json" {
			t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "json")
		}
	})

	t.Run("parses CORS origins correctly", func(t *testing.T) {
		os.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:3000, http://example.com , http://test.com")
		defer os.Unsetenv("CORS_ALLOW_ORIGINS")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		expectedOrigins := []string{"http://localhost:3000", "http://example.com", "http://test.com"}
		if len(cfg.CORSAllowOrigins) != len(expectedOrigins) {
			t.Fatalf("CORSAllowOrigins length = %d, want %d", len(cfg.CORSAllowOrigins), len(expectedOrigins))
		}

		for i, origin := range cfg.CORSAllowOrigins {
			if origin != expectedOrigins[i] {
				t.Errorf("CORSAllowOrigins[%d] = %q, want %q", i, origin, expectedOrigins[i])
			}
		}
	})

	t.Run("parses CORS credentials correctly", func(t *testing.T) {
		os.Setenv("CORS_ALLOW_CREDENTIALS", "false")
		defer os.Unsetenv("CORS_ALLOW_CREDENTIALS")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.CORSAllowCredentials != false {
			t.Errorf("CORSAllowCredentials = %v, want false", cfg.CORSAllowCredentials)
		}
	})

	t.Run("uses custom values from env", func(t *testing.T) {
		os.Setenv("DB_HOST", "custom-host")
		os.Setenv("DB_PORT", "5433")
		os.Setenv("SERVER_PORT", "9090")
		os.Setenv("JWT_SECRET", "super-secret")
		defer func() {
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("SERVER_PORT")
			os.Unsetenv("JWT_SECRET")
		}()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.DBHost != "custom-host" {
			t.Errorf("DBHost = %q, want %q", cfg.DBHost, "custom-host")
		}
		if cfg.DBPort != "5433" {
			t.Errorf("DBPort = %q, want %q", cfg.DBPort, "5433")
		}
		if cfg.ServerPort != "9090" {
			t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "9090")
		}
		if cfg.JWTSecret != "super-secret" {
			t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "super-secret")
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("passes validation in development with defaults", func(t *testing.T) {
		cfg := &Config{
			DBHost:           "localhost",
			DBPort:           "5432",
			DBUser:           "user",
			DBPassword:       "pass",
			DBName:           "db",
			ServerPort:       "8080",
			JWTSecret:        "default-secret-key",
			LogLevel:         "info",
			LogFormat:        "json",
			Environment:      "development",
			CORSAllowOrigins: []string{"http://localhost:3000"},
		}

		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("fails validation in production with default JWT secret", func(t *testing.T) {
		cfg := &Config{
			DBHost:           "localhost",
			DBPort:           "5432",
			DBUser:           "user",
			DBPassword:       "pass",
			DBName:           "db",
			ServerPort:       "8080",
			JWTSecret:        "default-secret-key",
			LogLevel:         "info",
			LogFormat:        "json",
			Environment:      "production",
			CORSAllowOrigins: []string{"https://example.com"},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for default JWT secret in production")
		}
	})

	t.Run("fails validation with invalid server port", func(t *testing.T) {
		cfg := &Config{
			DBHost:           "localhost",
			DBPort:           "5432",
			DBUser:           "user",
			DBPassword:       "pass",
			DBName:           "db",
			ServerPort:       "invalid",
			JWTSecret:        "secret",
			LogLevel:         "info",
			LogFormat:        "json",
			Environment:      "development",
			CORSAllowOrigins: []string{"http://localhost:3000"},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for invalid server port")
		}
	})

	t.Run("fails validation with port out of range", func(t *testing.T) {
		cfg := &Config{
			DBHost:           "localhost",
			DBPort:           "5432",
			DBUser:           "user",
			DBPassword:       "pass",
			DBName:           "db",
			ServerPort:       "70000",
			JWTSecret:        "secret",
			LogLevel:         "info",
			LogFormat:        "json",
			Environment:      "development",
			CORSAllowOrigins: []string{"http://localhost:3000"},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for port out of range")
		}
	})

	t.Run("fails validation with invalid log level", func(t *testing.T) {
		cfg := &Config{
			DBHost:           "localhost",
			DBPort:           "5432",
			DBUser:           "user",
			DBPassword:       "pass",
			DBName:           "db",
			ServerPort:       "8080",
			JWTSecret:        "secret",
			LogLevel:         "invalid",
			LogFormat:        "json",
			Environment:      "development",
			CORSAllowOrigins: []string{"http://localhost:3000"},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for invalid log level")
		}
	})

	t.Run("fails validation with invalid log format", func(t *testing.T) {
		cfg := &Config{
			DBHost:           "localhost",
			DBPort:           "5432",
			DBUser:           "user",
			DBPassword:       "pass",
			DBName:           "db",
			ServerPort:       "8080",
			JWTSecret:        "secret",
			LogLevel:         "info",
			LogFormat:        "xml",
			Environment:      "development",
			CORSAllowOrigins: []string{"http://localhost:3000"},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for invalid log format")
		}
	})

	t.Run("fails validation with invalid CORS origin", func(t *testing.T) {
		cfg := &Config{
			DBHost:           "localhost",
			DBPort:           "5432",
			DBUser:           "user",
			DBPassword:       "pass",
			DBName:           "db",
			ServerPort:       "8080",
			JWTSecret:        "secret",
			LogLevel:         "info",
			LogFormat:        "json",
			Environment:      "development",
			CORSAllowOrigins: []string{"not-a-valid-url"},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for invalid CORS origin")
		}
	})

	t.Run("fails validation with missing database config", func(t *testing.T) {
		cfg := &Config{
			DBHost:           "",
			DBPort:           "5432",
			DBUser:           "user",
			DBPassword:       "pass",
			DBName:           "db",
			ServerPort:       "8080",
			JWTSecret:        "secret",
			LogLevel:         "info",
			LogFormat:        "json",
			Environment:      "development",
			CORSAllowOrigins: []string{"http://localhost:3000"},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for missing database config")
		}
	})

	t.Run("fails validation with weak admin password", func(t *testing.T) {
		cfg := &Config{
			DBHost:            "localhost",
			DBPort:            "5432",
			DBUser:            "user",
			DBPassword:        "pass",
			DBName:            "db",
			ServerPort:        "8080",
			JWTSecret:         "secret",
			LogLevel:          "info",
			LogFormat:         "json",
			Environment:       "development",
			CORSAllowOrigins:  []string{"http://localhost:3000"},
			SeedAdminEmail:    "admin@example.com",
			SeedAdminPassword: "short",
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for weak admin password")
		}
	})

	t.Run("fails validation with invalid admin email", func(t *testing.T) {
		cfg := &Config{
			DBHost:            "localhost",
			DBPort:            "5432",
			DBUser:            "user",
			DBPassword:        "pass",
			DBName:            "db",
			ServerPort:        "8080",
			JWTSecret:         "secret",
			LogLevel:          "info",
			LogFormat:         "json",
			Environment:       "development",
			CORSAllowOrigins:  []string{"http://localhost:3000"},
			SeedAdminEmail:    "not-an-email",
			SeedAdminPassword: "longenoughpassword",
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for invalid admin email")
		}
	})

	t.Run("fails validation with wildcard CORS and credentials in production", func(t *testing.T) {
		cfg := &Config{
			DBHost:               "localhost",
			DBPort:               "5432",
			DBUser:               "user",
			DBPassword:           "pass",
			DBName:               "db",
			ServerPort:           "8080",
			JWTSecret:            "a-very-long-and-secure-secret-key-for-production",
			LogLevel:             "info",
			LogFormat:            "json",
			Environment:          "production",
			CORSAllowOrigins:     []string{"*"},
			CORSAllowCredentials: true,
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() error = nil, want error for wildcard CORS with credentials in production")
		}
	})

	t.Run("allows wildcard CORS without credentials in production", func(t *testing.T) {
		cfg := &Config{
			DBHost:               "localhost",
			DBPort:               "5432",
			DBUser:               "user",
			DBPassword:           "pass",
			DBName:               "db",
			ServerPort:           "8080",
			JWTSecret:            "a-very-long-and-secure-secret-key-for-production",
			LogLevel:             "info",
			LogFormat:            "json",
			Environment:          "production",
			CORSAllowOrigins:     []string{"*"},
			CORSAllowCredentials: false,
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v, want nil for wildcard CORS without credentials", err)
		}
	})
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		environment string
		want        bool
	}{
		{"production", true},
		{"development", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			cfg := &Config{Environment: tt.environment}
			if got := cfg.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		environment string
		want        bool
	}{
		{"development", true},
		{"", true},
		{"production", false},
		{"staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.environment, func(t *testing.T) {
			cfg := &Config{Environment: tt.environment}
			if got := cfg.IsDevelopment(); got != tt.want {
				t.Errorf("IsDevelopment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad_TrustedProxies(t *testing.T) {
	t.Run("parses TRUSTED_PROXIES correctly", func(t *testing.T) {
		os.Setenv("TRUSTED_PROXIES", "10.0.0.1, 10.0.0.2 , 192.168.1.0/24")
		defer os.Unsetenv("TRUSTED_PROXIES")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		expected := []string{"10.0.0.1", "10.0.0.2", "192.168.1.0/24"}
		if len(cfg.TrustedProxies) != len(expected) {
			t.Fatalf("TrustedProxies length = %d, want %d", len(cfg.TrustedProxies), len(expected))
		}
		for i, p := range cfg.TrustedProxies {
			if p != expected[i] {
				t.Errorf("TrustedProxies[%d] = %q, want %q", i, p, expected[i])
			}
		}
	})

	t.Run("returns nil when TRUSTED_PROXIES not set", func(t *testing.T) {
		os.Unsetenv("TRUSTED_PROXIES")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.TrustedProxies != nil {
			t.Errorf("TrustedProxies = %v, want nil", cfg.TrustedProxies)
		}
	})
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		port string
		want bool
	}{
		{"8080", true},
		{"1", true},
		{"65535", true},
		{"0", false},
		{"65536", false},
		{"-1", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.port, func(t *testing.T) {
			if got := isValidPort(tt.port); got != tt.want {
				t.Errorf("isValidPort(%q) = %v, want %v", tt.port, got, tt.want)
			}
		})
	}
}
