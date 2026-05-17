package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Validation errors
var (
	ErrMissingJWTSecret      = errors.New("JWT_SECRET must be set (generate with: openssl rand -hex 32)")
	ErrWeakJWTSecret         = errors.New("JWT_SECRET must be at least 32 characters and must not be the default placeholder value")
	ErrInvalidServerPort     = errors.New("SERVER_PORT must be a valid port number (1-65535)")
	ErrInvalidDBPort         = errors.New("DB_PORT must be a valid port number (1-65535)")
	ErrInvalidLogLevel       = errors.New("LOG_LEVEL must be one of: debug, info, warn, error")
	ErrInvalidLogFormat      = errors.New("LOG_FORMAT must be one of: json, text")
	ErrInvalidCORSOrigin     = errors.New("CORS_ALLOW_ORIGINS contains invalid URL")
	ErrInsecureCORS          = errors.New("CORS_ALLOW_ORIGINS=* combined with CORS_ALLOW_CREDENTIALS=true is not permitted")
	ErrInvalidDBSSLMode      = errors.New("DB_SSLMODE must be one of: disable, require, verify-ca, verify-full")
	ErrMissingDBConfig       = errors.New("database configuration incomplete: DB_HOST, DB_USER, DB_PASSWORD, and DB_NAME are required")
	ErrWeakAdminPassword     = errors.New("SEED_ADMIN_PASSWORD must be at least 8 characters when SEED_ADMIN_EMAIL is set")
	ErrInvalidAdminEmail     = errors.New("SEED_ADMIN_EMAIL must be a valid email address")
	ErrInvalidSMTPPort       = errors.New("SMTP_PORT must be a valid port number (1-65535)")
	ErrInvalidSMTPFrom       = errors.New("SMTP_FROM must be a valid email address")
	ErrMissingTOTPKey        = errors.New("TOTP_ENCRYPTION_KEY must be set (generate with: openssl rand -hex 32)")
	ErrInvalidTOTPKey        = errors.New("TOTP_ENCRYPTION_KEY must be exactly 32 bytes hex-encoded (64 hex chars) and must not be a known placeholder value")
	ErrInvalidWebAuthnOrigin = errors.New("WEBAUTHN_ORIGINS contains an invalid entry (must be scheme+host[+port], no wildcards, no trailing slashes)")
	// ErrProductionRateLimitDisabled fires when SECURE_COOKIES=true
	// (production indicator) and the per-IP login rate limiter is
	// disabled — closes audit finding O-M-9 (security review
	// 2026-05-01). Operators with an external rate limiter (WAF /
	// reverse proxy) must say so explicitly via
	// ALLOW_RATE_LIMIT_DISABLED_IN_PRODUCTION=true.
	ErrProductionRateLimitDisabled = errors.New("LOGIN_RATE_LIMIT_PER_MINUTE=0 is not permitted when SECURE_COOKIES=true; set a positive value, or opt out with ALLOW_RATE_LIMIT_DISABLED_IN_PRODUCTION=true if rate limiting happens upstream")
	// ErrProductionInsecureDB fires when SECURE_COOKIES=true and the
	// Postgres connection is configured with sslmode=disable — closes
	// audit finding O-M-10. Operators on a private network with TLS
	// terminated upstream of pgbouncer must opt in explicitly via
	// ALLOW_DB_SSLMODE_DISABLE_IN_PRODUCTION=true.
	ErrProductionInsecureDB = errors.New("DB_SSLMODE=disable is not permitted when SECURE_COOKIES=true; pick require/verify-ca/verify-full, or opt out with ALLOW_DB_SSLMODE_DISABLE_IN_PRODUCTION=true if TLS is terminated upstream")
	// ErrSeedTestDataInProduction fires when SECURE_COOKIES=true and
	// SEED_TEST_DATA=true. The test seeder plants six fully-privileged
	// users (superadmin/admin/manager) with the publicly-documented
	// password "supersecret"; no legitimate production deployment
	// needs that. Unlike the rate-limit and DB-SSL gates there is no
	// escape hatch — nothing upstream can substitute for "do not seed
	// fixture accounts".
	ErrSeedTestDataInProduction = errors.New("SEED_TEST_DATA=true is not permitted when SECURE_COOKIES=true; the test seeder creates fully-privileged accounts with a publicly-known password")
	// ErrInvalidTrustedProxy fires when TRUSTED_PROXIES contains an
	// entry that is not a valid CIDR or that covers the entire IPv4
	// or IPv6 address space (0.0.0.0/0, ::/0). The latter would
	// effectively trust every client to set X-Forwarded-For, which is
	// the same as having no rate-limit IP keying — defeats O-M-9.
	ErrInvalidTrustedProxy = errors.New("TRUSTED_PROXIES entry is invalid: each value must parse as a CIDR (e.g. 10.0.0.0/8) and must not be 0.0.0.0/0 or ::/0")
)

// knownPlaceholderTOTPKeys mirrors knownPlaceholderJWTSecrets: any string
// that's ever appeared in an example .env file is rejected so a forgotten
// placeholder cannot boot with a predictable key.
var knownPlaceholderTOTPKeys = map[string]bool{
	"0000000000000000000000000000000000000000000000000000000000000000": true,
	"1111111111111111111111111111111111111111111111111111111111111111": true,
	"change-me-in-production": true,
	"dev-only-totp-key-do-not-use-in-production-please-replace-now": true,
}

// knownPlaceholderJWTSecrets are legacy placeholder values that must be rejected.
// Add every placeholder string that has ever shipped in example files here so that
// a forgotten .env cannot boot with a predictable secret.
var knownPlaceholderJWTSecrets = map[string]bool{
	"default-secret-key":                               true,
	"your-super-secret-jwt-key-change-in-production":   true,
	"change-me-in-production":                          true,
	"dev-only-do-not-use-in-production-please-replace": true,
}

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Server
	ServerPort string
	JWTSecret  string

	// CSRFHMACKey is the HMAC key used to derive a session-bound
	// CSRF token from the session cookie value (see
	// middleware.ComputeCSRFToken). Closes audit finding C-M-3
	// (security review 2026-05-01): historically the CSRF derivation
	// shared JWTSecret, even though the codebase signs no JWTs and
	// "JWT secret rotation" carries different operational semantics
	// (logs out every session) than "rotate the CSRF HMAC key"
	// (invalidates CSRF tokens; users get a 403 once until the next
	// page load re-issues a token). Key separation per NIST SP
	// 800-57 §5.2.
	//
	// Falls back to JWTSecret when CSRF_HMAC_KEY is unset, so
	// existing deployments don't break — set CSRF_HMAC_KEY explicitly
	// in new deployments and document a rotation plan if you ever
	// bump JWTSecret without wanting to also invalidate CSRF tokens.
	CSRFHMACKey string

	// RBAC
	RBACModelPath string

	// Seeding
	SeedAdminEmail    string
	SeedAdminPassword string
	SeedAdminName     string
	SeedTestData      bool

	// Government Funding Seeding
	GovernmentFundingSeedPath  string
	GovernmentFundingSeedState string

	// CORS
	CORSAllowOrigins     []string
	CORSAllowCredentials bool

	// Logging
	LogLevel  string
	LogFormat string

	// Rate Limiting. 0 disables the limiter — the operator is responsible
	// for setting sensible values in production.
	LoginRateLimitPerMinute int
	APIRateLimitPerMinute   int

	// Database Connection Pool
	DBMaxIdleConns   int
	DBMaxOpenConns   int
	DBConnMaxLifeMin int
	DBConnMaxIdleMin int

	// Trusted Proxies (comma-separated CIDRs of your reverse proxy).
	// Empty means do not trust any proxy headers — c.ClientIP() then
	// returns the direct peer's address.
	TrustedProxies []string

	// Security
	SecureCookies bool

	// AllowRateLimitDisabledInProduction is the opt-out for
	// ErrProductionRateLimitDisabled — set true ONLY if rate-limiting
	// happens upstream (WAF / reverse proxy / API gateway).
	AllowRateLimitDisabledInProduction bool
	// AllowDBSSLModeDisableInProduction is the opt-out for
	// ErrProductionInsecureDB — set true ONLY if Postgres TLS is
	// terminated upstream (e.g. pgbouncer on the same host) AND the
	// connection between the API and pgbouncer is on a Unix socket /
	// loopback that the operator has independently verified.
	AllowDBSSLModeDisableInProduction bool

	// AuditLogRetentionDays is the rolling window the periodic
	// retention job keeps audit_logs rows for.
	//
	// **This is a policy decision the data controller must make and
	// document under DSGVO Art. 30.** GDPR itself fixes no period;
	// Art. 5(1)(e) requires "no longer than necessary" for the stated
	// purpose. Common landings for security audit logs:
	//   - PCI-DSS:                    1 year minimum (90 days online)
	//   - BSI IT-Grundschutz APP.4.4: 30-90 days minimum
	//   - SOC 2 / ISO 27001:          6-24 months typically
	//   - § 257 HGB / § 147 AO:       6-10 years for financial records
	//
	// Default: 730 days (2 years). Long enough to cover an annual
	// auditor review with margin and to investigate incidents that
	// surface late, short enough that DSGVO storage minimisation is
	// still defensible. Operators with stricter or looser legal
	// obligations should override via AUDIT_LOG_RETENTION_DAYS — set
	// to 0 to disable the periodic purge entirely (only do this if
	// you have an external retention pipeline).
	AuditLogRetentionDays int

	// SMTP
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// TOTP encryption. TOTPEncryptionKey is 64 hex chars (= 32 bytes) used
	// as the AES-256-GCM key for factor_totp_secrets.secret_ciphertext.
	// Rotating this key invalidates all stored TOTP secrets; the operational
	// answer is "affected users re-enroll TOTP." TOTPIssuer is the string
	// shown in the user's authenticator app.
	TOTPEncryptionKey string
	TOTPIssuer        string

	// WebAuthn relying-party identity. All three are required if any
	// WebAuthn factor is to be enrolled; together they nail down the
	// rpId/origin pair the browser binds credentials against. Rotating
	// WebAuthnRPID invalidates every stored credential across the
	// fleet — treat it as identity-forever once set.
	WebAuthnRPID    string
	WebAuthnRPName  string
	WebAuthnOrigins []string // comma-separated in env; split + trimmed at load
}

// Validate checks the configuration and fails closed on every axis.
func (c *Config) Validate() error {
	var errs []error

	// JWT Secret: always required, always strong, never a known placeholder.
	switch {
	case c.JWTSecret == "":
		errs = append(errs, ErrMissingJWTSecret)
	case knownPlaceholderJWTSecrets[c.JWTSecret]:
		errs = append(errs, ErrWeakJWTSecret)
	case len(c.JWTSecret) < 32:
		errs = append(errs, ErrWeakJWTSecret)
	}

	// Ports
	if !isValidPort(c.ServerPort) {
		errs = append(errs, ErrInvalidServerPort)
	}
	if !isValidPort(c.DBPort) {
		errs = append(errs, ErrInvalidDBPort)
	}

	// DB identity
	if c.DBHost == "" || c.DBUser == "" || c.DBPassword == "" || c.DBName == "" {
		errs = append(errs, ErrMissingDBConfig)
	}

	// DB SSL mode — syntactic check first.
	validSSLModes := map[string]bool{"disable": true, "require": true, "verify-ca": true, "verify-full": true}
	if !validSSLModes[c.DBSSLMode] {
		errs = append(errs, ErrInvalidDBSSLMode)
	}

	// Production-readiness gates. SecureCookies=true is the indicator
	// that the operator has wired up HTTPS (cookies must be Secure)
	// and is therefore running this in production. In that mode:
	//   - the per-IP login rate limit MUST be enabled (closes O-M-9),
	//     otherwise brute-force is bounded only by the network round-trip;
	//   - the DB connection MUST be encrypted (closes O-M-10), otherwise
	//     credentials and PII flow over plaintext.
	// Each gate has a documented escape hatch for operators who run
	// the equivalent control upstream (WAF / pgbouncer / VPC).
	if c.SecureCookies {
		if c.LoginRateLimitPerMinute == 0 && !c.AllowRateLimitDisabledInProduction {
			errs = append(errs, ErrProductionRateLimitDisabled)
		}
		if c.DBSSLMode == "disable" && !c.AllowDBSSLModeDisableInProduction {
			errs = append(errs, ErrProductionInsecureDB)
		}
		if c.SeedTestData {
			errs = append(errs, ErrSeedTestDataInProduction)
		}
	}

	// TRUSTED_PROXIES feeds gin's SetTrustedProxies. The default
	// behaviour (empty list) is safe: ClientIP() returns the direct
	// peer and X-Forwarded-* is ignored. A misconfigured value flips
	// this on for everyone — most catastrophically 0.0.0.0/0 (or
	// ::/0), which makes every client a "trusted proxy" allowed to
	// claim any source IP via X-Forwarded-For. That collapses the
	// per-IP login rate limiter to a no-op. Run the check
	// unconditionally (not behind SecureCookies) so a dev-mode
	// misconfiguration fails the same way it would in prod.
	for _, raw := range c.TrustedProxies {
		if _, ipnet, err := net.ParseCIDR(raw); err == nil {
			ones, bits := ipnet.Mask.Size()
			if ones == 0 && bits > 0 {
				// 0.0.0.0/0 or ::/0 — entire address space. Reject
				// outright: anything that wide is indistinguishable
				// from "no protection at all" and is more likely a
				// typo than a deliberate choice.
				errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidTrustedProxy, raw))
			}
			continue
		}
		// Bare IP form (e.g. "10.0.0.1") — Gin's SetTrustedProxies
		// accepts these as /32 or /128 single-host trusts, so we mirror
		// that. A bare IP cannot encode the entire address space, so
		// it skips the wildcard check.
		if net.ParseIP(raw) != nil {
			continue
		}
		errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidTrustedProxy, raw))
	}

	// Log level / format
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		errs = append(errs, ErrInvalidLogLevel)
	}
	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[strings.ToLower(c.LogFormat)] {
		errs = append(errs, ErrInvalidLogFormat)
	}

	// CORS — wildcard + credentials is a browser credential-theft primitive
	// and is rejected unconditionally.
	for _, origin := range c.CORSAllowOrigins {
		if origin == "*" {
			if c.CORSAllowCredentials {
				errs = append(errs, ErrInsecureCORS)
			}
			continue
		}
		if _, err := url.ParseRequestURI(origin); err != nil {
			errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidCORSOrigin, origin))
		}
	}

	// SMTP — optional overall. If host is set, port and From must be valid.
	if c.SMTPHost != "" {
		if c.SMTPPort < 1 || c.SMTPPort > 65535 {
			errs = append(errs, ErrInvalidSMTPPort)
		}
		if !strings.Contains(c.SMTPFrom, "@") {
			errs = append(errs, ErrInvalidSMTPFrom)
		}
	}

	// TOTP encryption key. Same shape discipline as JWT_SECRET: required,
	// known-placeholder values rejected, exact length enforced. Hex
	// decoding is done at startup (cmd/api/main.go) — here we only
	// validate the shape of the input string to fail fast with a clear
	// error.
	switch {
	case c.TOTPEncryptionKey == "":
		errs = append(errs, ErrMissingTOTPKey)
	case knownPlaceholderTOTPKeys[c.TOTPEncryptionKey]:
		errs = append(errs, ErrInvalidTOTPKey)
	case !isValidHex64(c.TOTPEncryptionKey):
		errs = append(errs, ErrInvalidTOTPKey)
	case isUniformHex64(c.TOTPEncryptionKey):
		// Catches the "developer typed `1` 64 times" failure mode that
		// the dev-example file historically ground in. Trips on
		// trivially-uniform keys (all 0x00, all 0x11, ...) that an
		// allowlist of literal placeholder strings wouldn't catch.
		errs = append(errs, ErrInvalidTOTPKey)
	}

	// WebAuthn: require all three fields together, or all three empty.
	// Empty means "the deployment does not support WebAuthn factors"
	// and the service wrapper is simply not wired up. A partially-
	// configured set is a bug, not a supported mode.
	anyWebAuthn := c.WebAuthnRPID != "" || c.WebAuthnRPName != "" || len(c.WebAuthnOrigins) > 0
	allWebAuthn := c.WebAuthnRPID != "" && c.WebAuthnRPName != "" && len(c.WebAuthnOrigins) > 0
	if anyWebAuthn && !allWebAuthn {
		errs = append(errs, errors.New("WEBAUTHN_RP_ID, WEBAUTHN_RP_NAME, and WEBAUTHN_ORIGINS must all be set or all left empty"))
	}
	if allWebAuthn {
		for _, o := range c.WebAuthnOrigins {
			if !isValidWebAuthnOrigin(o) {
				errs = append(errs, ErrInvalidWebAuthnOrigin)
				break
			}
		}
	}

	// Admin seeding
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

func isValidPort(port string) bool {
	p, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return p >= 1 && p <= 65535
}

// isValidWebAuthnOrigin enforces the shape WebAuthn expects for an
// RPOrigin string: scheme://host[:port] with no path, no query, no
// wildcards, no trailing slash. Browsers do an exact-string match
// against clientDataJSON.origin so any deviation breaks a ceremony.
// We accept http:// only for localhost so dev builds work without
// HTTPS but prod can't accidentally deploy with a plaintext origin.
func isValidWebAuthnOrigin(s string) bool {
	if s == "" || strings.Contains(s, "*") || strings.HasSuffix(s, "/") {
		return false
	}
	if rest, ok := strings.CutPrefix(s, "https://"); ok {
		return rest != "" && !strings.Contains(rest, "/")
	}
	if rest, ok := strings.CutPrefix(s, "http://"); ok {
		// Only permit plaintext for localhost-style dev origins.
		if !strings.HasPrefix(rest, "localhost") && !strings.HasPrefix(rest, "127.0.0.1") {
			return false
		}
		return rest != "" && !strings.Contains(rest, "/")
	}
	return false
}

// isValidHex64 returns true if s is exactly 64 hex characters — the
// form of a 32-byte key as emitted by `openssl rand -hex 32`.
func isValidHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// isUniformHex64 returns true when s is a hex string whose decoded
// bytes are all identical (`0x00 0x00 ... 0x00`, `0x11 0x11 ... 0x11`,
// etc.). It rules out the specific "developer typed a single character
// 64 times" failure mode where a placeholder like "1111…1111" ships
// to a real environment. It is intentionally narrower than a generic
// entropy heuristic — patterns like "deadbeefdeadbeef…" (4 distinct
// bytes) are still legal for tests, while genuine random keys from
// `openssl rand -hex 32` have a vanishing chance of tripping it.
func isUniformHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	// Hex is case-insensitive at the byte level — normalise via
	// per-pair comparison so "11111111…" and "1111…" still match the
	// uniformity check regardless of the case the operator typed.
	first := strings.ToLower(s[:2])
	for i := 2; i < 64; i += 2 {
		if strings.ToLower(s[i:i+2]) != first {
			return false
		}
	}
	return true
}

// splitCSV trims + filters a comma-separated env value. Used by
// every multi-valued env (WEBAUTHN_ORIGINS, CORS_ALLOW_ORIGINS
// pattern above, etc) so consumers don't have to implement the same
// split-and-trim dance twice.
func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for v := range strings.SplitSeq(raw, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var origins []string
	if raw := getEnv("CORS_ALLOW_ORIGINS", ""); raw != "" {
		for o := range strings.SplitSeq(raw, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				origins = append(origins, o)
			}
		}
	}

	var trustedProxies []string
	if tp := getEnv("TRUSTED_PROXIES", ""); tp != "" {
		for p := range strings.SplitSeq(tp, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				trustedProxies = append(trustedProxies, p)
			}
		}
	}

	cfg := &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", ""),
		DBSSLMode:  getEnv("DB_SSLMODE", "require"),

		ServerPort: getEnv("SERVER_PORT", "8080"),
		JWTSecret:  getEnv("JWT_SECRET", ""),
		// Falls back to JWTSecret when unset to keep existing
		// deployments working. New installations should set this
		// explicitly to a distinct 32-char-minimum value.
		CSRFHMACKey: getEnv("CSRF_HMAC_KEY", getEnv("JWT_SECRET", "")),

		RBACModelPath: getEnv("RBAC_MODEL_PATH", "configs/rbac_model.conf"),

		SeedAdminEmail:    getEnv("SEED_ADMIN_EMAIL", ""),
		SeedAdminPassword: getEnv("SEED_ADMIN_PASSWORD", ""),
		SeedAdminName:     getEnv("SEED_ADMIN_NAME", "admin"),
		SeedTestData:      getEnv("SEED_TEST_DATA", "false") == "true",

		GovernmentFundingSeedPath:  getEnv("GOVERNMENT_FUNDING_SEED_PATH", ""),
		GovernmentFundingSeedState: getEnv("GOVERNMENT_FUNDING_SEED_STATE", "berlin"),

		CORSAllowOrigins:     origins,
		CORSAllowCredentials: getEnv("CORS_ALLOW_CREDENTIALS", "true") == "true",

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		LoginRateLimitPerMinute: getEnvInt("LOGIN_RATE_LIMIT_PER_MINUTE", 5),
		APIRateLimitPerMinute:   getEnvInt("API_RATE_LIMIT_PER_MINUTE", 60),

		DBMaxIdleConns:   getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxOpenConns:   getEnvInt("DB_MAX_OPEN_CONNS", 100),
		DBConnMaxLifeMin: getEnvInt("DB_CONN_MAX_LIFE_MIN", 60),
		DBConnMaxIdleMin: getEnvInt("DB_CONN_MAX_IDLE_MIN", 10),

		TrustedProxies: trustedProxies,

		SecureCookies: getEnv("SECURE_COOKIES", "true") == "true",

		AllowRateLimitDisabledInProduction: getEnv("ALLOW_RATE_LIMIT_DISABLED_IN_PRODUCTION", "false") == "true",
		AllowDBSSLModeDisableInProduction:  getEnv("ALLOW_DB_SSLMODE_DISABLE_IN_PRODUCTION", "false") == "true",

		AuditLogRetentionDays: getEnvInt("AUDIT_LOG_RETENTION_DAYS", 730),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),

		TOTPEncryptionKey: getEnv("TOTP_ENCRYPTION_KEY", ""),
		TOTPIssuer:        getEnv("TOTP_ISSUER", "KitaManager"),

		WebAuthnRPID:    getEnv("WEBAUTHN_RP_ID", ""),
		WebAuthnRPName:  getEnv("WEBAUTHN_RP_NAME", ""),
		WebAuthnOrigins: splitCSV(getEnv("WEBAUTHN_ORIGINS", "")),
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
