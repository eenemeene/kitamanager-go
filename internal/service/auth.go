package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Token expiration settings
const (
	AccessTokenExpiry  = 1 * time.Hour      // Access tokens expire in 1 hour
	RefreshTokenExpiry = 7 * 24 * time.Hour // Refresh tokens expire in 7 days
	lockoutThreshold   = 5
	lockoutWindow      = 15 * time.Minute
	// passwordChangeLockoutThreshold and passwordChangeLockoutWindow govern
	// the /me/password brute-force defense. A stolen access token must not
	// allow running the current-password check at the generic mutation rate
	// limit, so we apply a stricter per-user counter.
	passwordChangeLockoutThreshold = 5
	passwordChangeLockoutWindow    = 15 * time.Minute
)

// AuthResult contains the tokens and metadata returned by auth operations.
type AuthResult struct {
	AccessToken  string
	RefreshToken string
	CSRFToken    string
	ExpiresIn    int64
}

// AuthService handles authentication business logic.
type AuthService struct {
	userStore    store.UserStorer
	tokenStore   store.TokenStorer
	jwtSecret    string
	auditService *AuditService
	// dummyPasswordHash is a bcrypt hash used to equalize Login timing when the
	// supplied email does not correspond to any user. Without it, bcrypt is
	// skipped on the not-found path and response latency acts as an account
	// enumeration oracle.
	dummyPasswordHash []byte
}

// NewAuthService creates a new auth service.
func NewAuthService(userStore store.UserStorer, tokenStore store.TokenStorer, jwtSecret string, auditService *AuditService) *AuthService {
	// Cost matches bcrypt.DefaultCost used for real password hashes so that
	// CompareHashAndPassword takes the same time on miss as on hit.
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)
	if err != nil {
		// bcrypt.GenerateFromPassword only fails on bad cost parameters, which
		// can't happen with DefaultCost. Panicking here is appropriate because
		// the service cannot start without a usable dummy hash.
		panic(fmt.Sprintf("failed to generate dummy password hash: %v", err))
	}
	return &AuthService{
		userStore:         userStore,
		tokenStore:        tokenStore,
		jwtSecret:         jwtSecret,
		auditService:      auditService,
		dummyPasswordHash: dummyHash,
	}
}

// canonicalEmail normalizes an email for equality-sensitive operations
// (lockout counters, audit rows) so that case and whitespace variants map to
// the same bucket. Without this, an attacker can reset the lockout by rotating
// "Victim@x.com" / "victim@x.com" / "VICTIM@x.com".
func canonicalEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Login authenticates a user with email and password.
func (s *AuthService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*AuthResult, error) {
	// Canonical form is used for the lockout counter and the audit row so that
	// "Victim@x.com" / "victim@x.com" / "VICTIM@x.com" all share one bucket and
	// an attacker cannot reset the lockout by rotating case. Backwards
	// compatibility: the raw (trimmed) email is still passed to FindByEmail so
	// users whose stored email has mixed case remain findable.
	rawEmail := strings.TrimSpace(email)
	canonical := canonicalEmail(email)

	failedCount, err := s.auditService.CountRecentFailedLogins(ctx, canonical, lockoutWindow)
	if err == nil && failedCount >= lockoutThreshold {
		s.auditService.LogLoginFailed(canonical, ipAddress, userAgent, "account locked - too many failed attempts")
		return nil, apperror.TooManyRequests("too many failed login attempts, please try again later")
	}

	user, err := s.userStore.FindByEmail(ctx, rawEmail)
	if err != nil {
		// Equalize timing against the not-found branch so response latency
		// does not enumerate valid accounts.
		_ = bcrypt.CompareHashAndPassword(s.dummyPasswordHash, []byte(password))
		s.auditService.LogLoginFailed(canonical, ipAddress, userAgent, "user not found")
		return nil, apperror.Unauthorized("invalid credentials")
	}

	if !user.Active {
		_ = bcrypt.CompareHashAndPassword(s.dummyPasswordHash, []byte(password))
		s.auditService.LogLoginFailed(canonical, ipAddress, userAgent, "user inactive")
		return nil, apperror.Unauthorized("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.auditService.LogLoginFailed(canonical, ipAddress, userAgent, "invalid password")
		return nil, apperror.Unauthorized("invalid credentials")
	}

	// Update last login timestamp
	_ = s.userStore.UpdateLastLogin(ctx, user.ID)

	// Log successful login
	s.auditService.LogLogin(user.ID, user.Email, ipAddress, userAgent)

	return s.issueTokens(user.ID, user.Email)
}

// Refresh exchanges a valid refresh token for new tokens. The caller must also
// pass the current access-token string (from the access_token cookie, or empty
// if none was sent); it is added to the revocation list so an access token
// stolen before the rotation does not outlive the refresh. Without this,
// `AccessTokenExpiry` minutes of valid-token window persist after the
// legitimate user refreshes. See M7.
func (s *AuthService) Refresh(ctx context.Context, refreshTokenStr, oldAccessTokenStr string) (*AuthResult, error) {
	claims, err := s.parseAndValidateRefreshToken(refreshTokenStr)
	if err != nil {
		return nil, err
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return nil, apperror.Unauthorized("invalid user ID in token")
	}
	userID := uint(userIDFloat)

	iatFloat, ok := claims["iat"].(float64)
	if !ok || iatFloat <= 0 {
		return nil, apperror.Unauthorized("invalid refresh token")
	}
	tokenIssuedAt := time.Unix(int64(iatFloat), 0).UTC()

	// Check for token replay
	if s.tokenStore != nil {
		oldHash := middleware.HashToken(refreshTokenStr)
		revoked, err := s.tokenStore.IsRevoked(ctx, oldHash)
		if err != nil {
			return nil, apperror.InternalWrap(err, "failed to check token revocation")
		}
		if revoked {
			if err := s.tokenStore.RevokeAllForUser(ctx, userID); err != nil {
				slog.Error("Failed to revoke all tokens after replay detection", "user_id", userID, "error", err)
			}
			return nil, apperror.Unauthorized("token reuse detected, all sessions have been revoked")
		}

		userRevoked, err := s.tokenStore.IsUserRevokedSince(ctx, userID, tokenIssuedAt)
		if err != nil {
			return nil, apperror.InternalWrap(err, "failed to check user token revocation")
		}
		if userRevoked {
			return nil, apperror.Unauthorized("all sessions have been revoked, please login again")
		}
	}

	// Verify user still exists and is active
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil || !user.Active {
		return nil, apperror.Unauthorized("invalid refresh token")
	}

	// Revoke the old refresh token, and the old access token if the client
	// sent one. Both must be unusable after rotation — otherwise an attacker
	// who exfiltrated an access token would still be authorized for up to
	// AccessTokenExpiry after the victim refreshed.
	if s.tokenStore != nil {
		expFloat, _ := claims["exp"].(float64)
		oldExpiresAt := time.Unix(int64(expFloat), 0)
		oldHash := middleware.HashToken(refreshTokenStr)
		if err := s.tokenStore.RevokeToken(ctx, oldHash, user.ID, oldExpiresAt); err != nil {
			slog.Error("Failed to revoke old refresh token", "user_id", user.ID, "error", err)
		}
		if oldAccessTokenStr != "" {
			s.revokeTokenString(ctx, oldAccessTokenStr)
		}
	}

	return s.issueTokens(user.ID, user.Email)
}

// Logout revokes the provided tokens.
func (s *AuthService) Logout(ctx context.Context, accessTokenStr, refreshTokenStr string) {
	if s.tokenStore == nil {
		return
	}
	if accessTokenStr != "" {
		s.revokeTokenString(ctx, accessTokenStr)
	}
	if refreshTokenStr != "" {
		s.revokeTokenString(ctx, refreshTokenStr)
	}
}

// ChangePassword verifies the current password, sets a new one, and re-issues
// tokens. Protected against brute force:
//   - Every failure is audit-logged (action: password_change_failed).
//   - Before checking the current password, a lockout counter of
//     password_change_failed events for this user is consulted; once
//     `passwordChangeLockoutThreshold` failures land within
//     `passwordChangeLockoutWindow`, further attempts return 429 without
//     touching bcrypt.
//   - On user-not-found and post-lockout paths, a dummy bcrypt compare runs
//     so response timing does not leak which users exist.
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword, ipAddress string) (*AuthResult, error) {
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyPasswordHash, []byte(currentPassword))
		return nil, classifyStoreError(err, "user")
	}

	failedCount, countErr := s.auditService.CountRecentFailedPasswordChanges(ctx, userID, passwordChangeLockoutWindow)
	if countErr == nil && failedCount >= passwordChangeLockoutThreshold {
		_ = bcrypt.CompareHashAndPassword(s.dummyPasswordHash, []byte(currentPassword))
		s.auditService.LogPasswordChangeFailed(userID, user.Email, ipAddress, "account locked - too many failed password-change attempts")
		return nil, apperror.TooManyRequests("too many failed password-change attempts, please try again later")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		s.auditService.LogPasswordChangeFailed(userID, user.Email, ipAddress, "invalid current password")
		return nil, apperror.Unauthorized("current password is incorrect")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.Internal("failed to hash password")
	}

	user.Password = string(hashedPassword)
	if err := s.userStore.Update(ctx, user); err != nil {
		return nil, apperror.Internal("failed to update password")
	}

	// Revoke all existing tokens
	if s.tokenStore != nil {
		if err := s.tokenStore.RevokeAllForUser(ctx, userID); err != nil {
			slog.Error("Failed to revoke all tokens after password change", "user_id", userID, "error", err)
		}
		// JWT `iat` is second-precision but the sentinel's CreatedAt is
		// sub-second. Wait past the next whole-second boundary so the fresh
		// tokens we are about to issue have an iat strictly greater than the
		// sentinel cutoff — otherwise the caller's own new session would be
		// revoked in the same tick (#134). Bounded by <1s so this is cheap.
		waitForNextSecondTick()
	}

	s.auditService.LogPasswordChange(userID, user.Email, ipAddress)

	return s.issueTokens(user.ID, user.Email)
}

// GetCurrentUser returns the user for the given ID.
func (s *AuthService) GetCurrentUser(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return nil, classifyStoreError(err, "user")
	}
	return user, nil
}

// issueTokens generates a new set of access, refresh, and CSRF tokens.
func (s *AuthService) issueTokens(userID uint, email string) (*AuthResult, error) {
	accessToken, err := s.generateAccessToken(userID, email)
	if err != nil {
		return nil, apperror.Internal("failed to generate access token")
	}

	refreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, apperror.Internal("failed to generate refresh token")
	}

	csrfToken := middleware.ComputeCSRFToken(accessToken, s.jwtSecret)

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
		ExpiresIn:    int64(AccessTokenExpiry.Seconds()),
	}, nil
}

// generateAccessToken creates a short-lived JWT for API access.
func (s *AuthService) generateAccessToken(userID uint, email string) (string, error) {
	now := time.Now().UTC()
	jti, err := generateJTI()
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":     fmt.Sprintf("%d", userID),
		"user_id": userID,
		"email":   email,
		"type":    "access",
		"jti":     jti,
		"exp":     now.Add(AccessTokenExpiry).Unix(),
		"nbf":     now.Unix(),
		"iat":     now.Unix(),
	})
	return token.SignedString([]byte(s.jwtSecret))
}

// generateRefreshToken creates a long-lived JWT for token refresh.
func (s *AuthService) generateRefreshToken(userID uint) (string, error) {
	now := time.Now().UTC()
	jti, err := generateJTI()
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":     fmt.Sprintf("%d", userID),
		"user_id": userID,
		"type":    "refresh",
		"jti":     jti,
		"exp":     now.Add(RefreshTokenExpiry).Unix(),
		"nbf":     now.Unix(),
		"iat":     now.Unix(),
	})
	return token.SignedString([]byte(s.jwtSecret))
}

// generateJTI generates a unique JWT ID (jti claim).
func generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// waitForNextSecondTick blocks until the current wall-clock second has passed.
// Used to decouple second-precision JWT `iat` claims from the sub-second
// precision of sentinel row CreatedAt timestamps (#134).
var waitForNextSecondTick = func() {
	now := time.Now()
	time.Sleep(now.Truncate(time.Second).Add(time.Second).Sub(now))
}

// parseAndValidateRefreshToken parses a JWT string and validates it is a refresh token.
func (s *AuthService) parseAndValidateRefreshToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, apperror.Unauthorized("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, apperror.Unauthorized("invalid token claims")
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return nil, apperror.Unauthorized("invalid token type")
	}

	return claims, nil
}

// revokeTokenString parses a JWT to extract user_id and exp, then revokes it.
func (s *AuthService) revokeTokenString(ctx context.Context, tokenStr string) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return
	}
	userIDFloat, _ := claims["user_id"].(float64)
	expFloat, _ := claims["exp"].(float64)
	expiresAt := time.Unix(int64(expFloat), 0)
	tokenHash := middleware.HashToken(tokenStr)
	if err := s.tokenStore.RevokeToken(ctx, tokenHash, uint(userIDFloat), expiresAt); err != nil {
		slog.Error("Failed to revoke token during logout", "user_id", uint(userIDFloat), "error", err)
	}
}
