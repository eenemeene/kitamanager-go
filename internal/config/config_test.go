package config

import (
	"os"
	"strings"
	"testing"
)

// validTestJWTSecret is a 32+ character secret suitable for tests.
// It must not appear in knownPlaceholderJWTSecrets.
const validTestJWTSecret = "test-secret-this-value-is-at-least-32-chars-long"

// validTestTOTPKey is a 64-hex-char key suitable for tests. Must not
// appear in knownPlaceholderTOTPKeys.
const validTestTOTPKey = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

// baseValidConfig returns a Config that passes Validate() — tests copy it and
// mutate a single field to isolate failure cases.
func baseValidConfig() *Config {
	return &Config{
		DBHost:               "localhost",
		DBPort:               "5432",
		DBUser:               "user",
		DBPassword:           "pass",
		DBName:               "db",
		DBSSLMode:            "disable",
		ServerPort:           "8080",
		JWTSecret:            validTestJWTSecret,
		TOTPEncryptionKey:    validTestTOTPKey,
		TOTPIssuer:           "KitaManager",
		LogLevel:             "info",
		LogFormat:            "json",
		CORSAllowOrigins:     []string{"http://localhost:3000"},
		CORSAllowCredentials: true,
	}
}

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

// envKeysUnderTest is every env var Load() reads. Tests clear these to isolate
// behavior from the host environment.
var envKeysUnderTest = []string{
	"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
	"SERVER_PORT", "JWT_SECRET", "RBAC_MODEL_PATH",
	"SEED_ADMIN_EMAIL", "SEED_ADMIN_PASSWORD", "SEED_ADMIN_NAME", "SEED_TEST_DATA",
	"GOVERNMENT_FUNDING_SEED_PATH", "GOVERNMENT_FUNDING_SEED_STATE",
	"CORS_ALLOW_ORIGINS", "CORS_ALLOW_CREDENTIALS",
	"LOG_LEVEL", "LOG_FORMAT",
	"LOGIN_RATE_LIMIT_PER_MINUTE", "API_RATE_LIMIT_PER_MINUTE",
	"DB_MAX_IDLE_CONNS", "DB_MAX_OPEN_CONNS", "DB_CONN_MAX_LIFE_MIN", "DB_CONN_MAX_IDLE_MIN",
	"TRUSTED_PROXIES", "SECURE_COOKIES",
	"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASSWORD", "SMTP_FROM",
	"TOTP_ENCRYPTION_KEY", "TOTP_ISSUER",
}

func snapshotEnv(t *testing.T) func() {
	t.Helper()
	original := make(map[string]string, len(envKeysUnderTest))
	for _, k := range envKeysUnderTest {
		original[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	return func() {
		for k, v := range original {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
}

func TestLoad_RejectsEmptyJWTSecret(t *testing.T) {
	defer snapshotEnv(t)()
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")

	if _, err := Load(); err == nil {
		t.Fatal("Load() with empty JWT_SECRET expected to fail, got nil")
	}
}

func TestLoad_RejectsMissingDBCreds(t *testing.T) {
	defer snapshotEnv(t)()
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	// DB_USER, DB_PASSWORD, DB_NAME intentionally unset.

	if _, err := Load(); err == nil {
		t.Fatal("Load() with missing DB creds expected to fail, got nil")
	}
}

func TestLoad_SucceedsWithRequiredEnv(t *testing.T) {
	defer snapshotEnv(t)()
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_SSLMODE", "disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if cfg.JWTSecret != validTestJWTSecret {
		t.Errorf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("DBHost default = %q, want localhost", cfg.DBHost)
	}
}

func TestLoad_ParsesCORSOrigins(t *testing.T) {
	defer snapshotEnv(t)()
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_SSLMODE", "disable")
	os.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:3000, http://example.com , http://test.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	want := []string{"http://localhost:3000", "http://example.com", "http://test.com"}
	if len(cfg.CORSAllowOrigins) != len(want) {
		t.Fatalf("got %d origins, want %d", len(cfg.CORSAllowOrigins), len(want))
	}
	for i, o := range cfg.CORSAllowOrigins {
		if o != want[i] {
			t.Errorf("origin[%d] = %q, want %q", i, o, want[i])
		}
	}
}

func TestLoad_EmptyCORSOriginsYieldsNil(t *testing.T) {
	defer snapshotEnv(t)()
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_SSLMODE", "disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if cfg.CORSAllowOrigins != nil {
		t.Errorf("CORSAllowOrigins = %v, want nil", cfg.CORSAllowOrigins)
	}
}

func TestLoad_ParsesCORSCredentials(t *testing.T) {
	defer snapshotEnv(t)()
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_SSLMODE", "disable")
	os.Setenv("CORS_ALLOW_CREDENTIALS", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if cfg.CORSAllowCredentials {
		t.Errorf("CORSAllowCredentials = true, want false")
	}
}

func TestLoad_DBSSLDefaultsToRequire(t *testing.T) {
	defer snapshotEnv(t)()
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")
	// DB_SSLMODE not set — default should be "require".

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if cfg.DBSSLMode != "require" {
		t.Errorf("DBSSLMode default = %q, want require", cfg.DBSSLMode)
	}
}

func TestLoad_SecureCookiesDefaultTrue(t *testing.T) {
	defer snapshotEnv(t)()
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_SSLMODE", "disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if !cfg.SecureCookies {
		t.Errorf("SecureCookies default = false, want true")
	}
}

func TestLoad_TrustedProxies(t *testing.T) {
	t.Run("parses comma-separated list", func(t *testing.T) {
		defer snapshotEnv(t)()
		os.Setenv("JWT_SECRET", validTestJWTSecret)
		os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
		os.Setenv("DB_USER", "u")
		os.Setenv("DB_PASSWORD", "p")
		os.Setenv("DB_NAME", "db")
		os.Setenv("DB_SSLMODE", "disable")
		os.Setenv("TRUSTED_PROXIES", "10.0.0.1, 10.0.0.2 , 192.168.1.0/24")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() = %v", err)
		}
		want := []string{"10.0.0.1", "10.0.0.2", "192.168.1.0/24"}
		if len(cfg.TrustedProxies) != len(want) {
			t.Fatalf("len = %d, want %d", len(cfg.TrustedProxies), len(want))
		}
		for i, p := range cfg.TrustedProxies {
			if p != want[i] {
				t.Errorf("TrustedProxies[%d] = %q, want %q", i, p, want[i])
			}
		}
	})

	t.Run("empty yields nil", func(t *testing.T) {
		defer snapshotEnv(t)()
		os.Setenv("JWT_SECRET", validTestJWTSecret)
		os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
		os.Setenv("DB_USER", "u")
		os.Setenv("DB_PASSWORD", "p")
		os.Setenv("DB_NAME", "db")
		os.Setenv("DB_SSLMODE", "disable")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() = %v", err)
		}
		if cfg.TrustedProxies != nil {
			t.Errorf("TrustedProxies = %v, want nil", cfg.TrustedProxies)
		}
	})
}

func TestConfig_Validate_Passes(t *testing.T) {
	if err := baseValidConfig().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestConfig_Validate_RejectsEmptyJWTSecret(t *testing.T) {
	cfg := baseValidConfig()
	cfg.JWTSecret = ""
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("Validate() = %v, want JWT_SECRET error", err)
	}
}

func TestConfig_Validate_RejectsShortJWTSecret(t *testing.T) {
	cfg := baseValidConfig()
	cfg.JWTSecret = "too-short"
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() = nil, want error for short JWT secret")
	}
}

func TestConfig_Validate_RejectsKnownPlaceholderJWTSecret(t *testing.T) {
	placeholders := []string{
		"default-secret-key",
		"your-super-secret-jwt-key-change-in-production",
		"change-me-in-production",
	}
	for _, p := range placeholders {
		t.Run(p, func(t *testing.T) {
			cfg := baseValidConfig()
			cfg.JWTSecret = p
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() = nil, want error for placeholder %q", p)
			}
		})
	}
}

func TestConfig_Validate_PortAndLogFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"invalid server port", func(c *Config) { c.ServerPort = "invalid" }},
		{"server port out of range", func(c *Config) { c.ServerPort = "70000" }},
		{"invalid db port", func(c *Config) { c.DBPort = "abc" }},
		{"invalid log level", func(c *Config) { c.LogLevel = "verbose" }},
		{"invalid log format", func(c *Config) { c.LogFormat = "xml" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseValidConfig()
			tc.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() = nil, want error")
			}
		})
	}
}

func TestConfig_Validate_RejectsInvalidCORSOrigin(t *testing.T) {
	cfg := baseValidConfig()
	cfg.CORSAllowOrigins = []string{"not-a-valid-url"}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for invalid CORS origin")
	}
}

func TestConfig_Validate_RejectsWildcardCORSWithCredentials(t *testing.T) {
	cfg := baseValidConfig()
	cfg.CORSAllowOrigins = []string{"*"}
	cfg.CORSAllowCredentials = true
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "CORS_ALLOW_ORIGINS=*") {
		t.Errorf("Validate() = %v, want wildcard+credentials error", err)
	}
}

func TestConfig_Validate_AllowsWildcardCORSWithoutCredentials(t *testing.T) {
	cfg := baseValidConfig()
	cfg.CORSAllowOrigins = []string{"*"}
	cfg.CORSAllowCredentials = false
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for wildcard without credentials", err)
	}
}

func TestConfig_Validate_RejectsMissingDBConfig(t *testing.T) {
	cfg := baseValidConfig()
	cfg.DBHost = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for missing DB host")
	}
}

func TestConfig_Validate_RejectsInvalidDBSSLMode(t *testing.T) {
	cfg := baseValidConfig()
	cfg.DBSSLMode = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for invalid DB_SSLMODE")
	}
}

func TestConfig_Validate_AllowsAllValidDBSSLModes(t *testing.T) {
	for _, mode := range []string{"disable", "require", "verify-ca", "verify-full"} {
		t.Run(mode, func(t *testing.T) {
			cfg := baseValidConfig()
			cfg.DBSSLMode = mode
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil for sslmode=%q", err, mode)
			}
		})
	}
}

func TestConfig_Validate_SMTPIsOptional(t *testing.T) {
	cfg := baseValidConfig()
	cfg.SMTPHost = ""
	cfg.SMTPFrom = ""
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil with no SMTP configured", err)
	}
}

func TestConfig_Validate_SMTPRequiresValidFromAndPortWhenHostSet(t *testing.T) {
	t.Run("invalid port", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.SMTPHost = "smtp.example.com"
		cfg.SMTPPort = 0
		cfg.SMTPFrom = "noreply@example.com"
		if err := cfg.Validate(); err == nil {
			t.Error("Validate() = nil, want error for invalid SMTP port")
		}
	})
	t.Run("invalid from", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.SMTPHost = "smtp.example.com"
		cfg.SMTPPort = 587
		cfg.SMTPFrom = "not-an-email"
		if err := cfg.Validate(); err == nil {
			t.Error("Validate() = nil, want error for invalid SMTP from")
		}
	})
}

func TestConfig_Validate_AdminSeeding(t *testing.T) {
	t.Run("invalid email", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.SeedAdminEmail = "not-an-email"
		cfg.SeedAdminPassword = "longenoughpassword"
		if err := cfg.Validate(); err == nil {
			t.Error("Validate() = nil, want error for invalid admin email")
		}
	})
	t.Run("short password", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.SeedAdminEmail = "admin@example.com"
		cfg.SeedAdminPassword = "short"
		if err := cfg.Validate(); err == nil {
			t.Error("Validate() = nil, want error for short admin password")
		}
	})
	t.Run("both valid", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.SeedAdminEmail = "admin@example.com"
		cfg.SeedAdminPassword = "strongenough"
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil", err)
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

// ----------------------------------------------------------------------
// CSRF_HMAC_KEY — separate from JWT_SECRET (closes audit C-M-3).
// ----------------------------------------------------------------------

func TestLoad_CSRFHMACKey_FallsBackToJWTSecret(t *testing.T) {
	// Existing deployments don't set CSRF_HMAC_KEY. The fallback to
	// JWT_SECRET keeps them booting and CSRF-validating without a
	// breaking change.
	defer snapshotEnv(t)()
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_SSLMODE", "disable")
	os.Unsetenv("CSRF_HMAC_KEY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CSRFHMACKey != validTestJWTSecret {
		t.Errorf("CSRFHMACKey = %q, want fallback to JWTSecret", cfg.CSRFHMACKey)
	}
}

func TestLoad_CSRFHMACKey_OverridesWhenSet(t *testing.T) {
	// New deployments set CSRF_HMAC_KEY explicitly to a distinct
	// value, so future JWT_SECRET rotations don't silently invalidate
	// every CSRF token.
	defer snapshotEnv(t)()
	const distinct = "another-32-plus-character-secret-for-csrf-only"
	os.Setenv("JWT_SECRET", validTestJWTSecret)
	os.Setenv("CSRF_HMAC_KEY", distinct)
	os.Setenv("TOTP_ENCRYPTION_KEY", validTestTOTPKey)
	os.Setenv("DB_USER", "u")
	os.Setenv("DB_PASSWORD", "p")
	os.Setenv("DB_NAME", "db")
	os.Setenv("DB_SSLMODE", "disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CSRFHMACKey != distinct {
		t.Errorf("CSRFHMACKey = %q, want %q", cfg.CSRFHMACKey, distinct)
	}
	if cfg.JWTSecret == cfg.CSRFHMACKey {
		t.Error("JWTSecret and CSRFHMACKey must be independent when both env vars are set")
	}
}
