package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/middleware"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Session configuration.
const (
	// SessionLifetime matches the old refresh-token lifetime so the
	// user-visible auth window after login is unchanged.
	SessionLifetime = store.SessionDefaultLifetime

	lockoutThreshold = 5
	lockoutWindow    = 15 * time.Minute

	// passwordChangeLockout* govern the /me/password brute-force defense. A
	// stolen session must not allow running the current-password check at
	// the generic mutation rate limit, so we apply a stricter per-user
	// counter.
	passwordChangeLockoutThreshold = 5
	passwordChangeLockoutWindow    = 15 * time.Minute
)

// AuthResult contains the session token and metadata returned by auth
// operations. `SessionToken` is the raw value that must be placed in the
// client's `session` cookie (or used as `Authorization: Bearer` for CLI
// clients); the server only ever stores sha256 of this value.
type AuthResult struct {
	SessionToken string
	CSRFToken    string
	ExpiresIn    int64
}

// AuthService handles authentication business logic.
type AuthService struct {
	userStore    store.UserStorer
	sessionStore store.SessionStorer
	serverSecret string
	auditService *AuditService
	// dummyPasswordHash equalizes Login timing on the not-found path.
	// Without it, bcrypt is skipped when the email is unknown and response
	// latency acts as an account enumeration oracle.
	dummyPasswordHash []byte
}

// NewAuthService creates a new auth service. `serverSecret` is used for CSRF
// HMAC derivation; the existing JWT_SECRET config value is reused because it
// already has a 32-char floor.
func NewAuthService(userStore store.UserStorer, sessionStore store.SessionStorer, serverSecret string, auditService *AuditService) *AuthService {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("failed to generate dummy password hash: %v", err))
	}
	return &AuthService{
		userStore:         userStore,
		sessionStore:      sessionStore,
		serverSecret:      serverSecret,
		auditService:      auditService,
		dummyPasswordHash: dummyHash,
	}
}

// canonicalEmail normalizes an email for equality-sensitive operations
// (lockout counters, audit rows) so that case and whitespace variants map to
// the same bucket.
func canonicalEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Login authenticates a user with email and password and issues a new
// session.
func (s *AuthService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*AuthResult, error) {
	// Canonical form is used for the lockout counter and the audit row so
	// an attacker cannot reset the lockout by rotating case. Backwards
	// compatibility: the raw (trimmed) email is still passed to FindByEmail
	// so users whose stored email has mixed case remain findable.
	rawEmail := strings.TrimSpace(email)
	canonical := canonicalEmail(email)

	failedCount, err := s.auditService.CountRecentFailedLogins(ctx, canonical, lockoutWindow)
	if err == nil && failedCount >= lockoutThreshold {
		s.auditService.LogLoginFailed(canonical, ipAddress, userAgent, "account locked - too many failed attempts")
		return nil, apperror.TooManyRequests("too many failed login attempts, please try again later")
	}

	user, err := s.userStore.FindByEmail(ctx, rawEmail)
	if err != nil {
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

	_ = s.userStore.UpdateLastLogin(ctx, user.ID)
	s.auditService.LogLogin(user.ID, user.Email, ipAddress, userAgent)

	return s.issueSession(ctx, user.ID, ipAddress, userAgent)
}

// Logout deletes the caller's session. Idempotent — a missing row is not an
// error, so logging out an already-expired session is a no-op.
func (s *AuthService) Logout(ctx context.Context, sessionToken string) {
	if sessionToken == "" {
		return
	}
	idHash := store.HashSessionToken(sessionToken)
	if err := s.sessionStore.Delete(ctx, idHash); err != nil {
		slog.Error("Failed to delete session during logout", "error", err)
	}
}

// ChangePassword verifies the current password, sets a new one, logs every
// other session out, and returns a freshly rotated session for the caller.
//
// Brute-force protection:
//   - Every failure is audit-logged (action: password_change_failed).
//   - Before checking the current password, a lockout counter of
//     password_change_failed events for this user is consulted; once
//     `passwordChangeLockoutThreshold` failures land within
//     `passwordChangeLockoutWindow`, further attempts return 429 without
//     touching bcrypt.
//   - On user-not-found and post-lockout paths, a dummy bcrypt compare runs
//     so response timing does not leak which users exist.
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword, currentSessionIDHash, ipAddress, userAgent string) (*AuthResult, error) {
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

	// Log out every OTHER session belonging to this user. Keep the caller's
	// current session alive so the password change doesn't log them out
	// mid-request; we rotate it separately to close the tiny window in which
	// a stolen copy of the old session token is still accepted.
	if currentSessionIDHash != "" {
		if err := s.sessionStore.DeleteAllForUserExcept(ctx, userID, currentSessionIDHash); err != nil {
			slog.Error("Failed to revoke other sessions after password change", "user_id", userID, "error", err)
		}
		// Rotate the caller's session: delete the current row, issue a new
		// one. The handler sets the new cookie so the browser keeps working.
		if err := s.sessionStore.Delete(ctx, currentSessionIDHash); err != nil {
			slog.Error("Failed to rotate current session after password change", "user_id", userID, "error", err)
		}
	} else {
		// No caller session (e.g. ChangePassword invoked outside a request
		// context in a test): just nuke everything.
		if err := s.sessionStore.DeleteAllForUser(ctx, userID); err != nil {
			slog.Error("Failed to revoke all sessions after password change", "user_id", userID, "error", err)
		}
	}

	s.auditService.LogPasswordChange(userID, user.Email, ipAddress)

	return s.issueSession(ctx, user.ID, ipAddress, userAgent)
}

// GetCurrentUser returns the user for the given ID.
func (s *AuthService) GetCurrentUser(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return nil, classifyStoreError(err, "user")
	}
	return user, nil
}

// issueSession generates a new session row and the matching CSRF token.
func (s *AuthService) issueSession(ctx context.Context, userID uint, ipAddress, userAgent string) (*AuthResult, error) {
	raw, hashed, err := store.GenerateSessionToken()
	if err != nil {
		return nil, apperror.Internal("failed to generate session token")
	}
	now := time.Now().UTC()
	sess := &models.Session{
		ID:               hashed,
		UserID:           userID,
		CreatedAt:        now,
		ExpiresAt:        now.Add(SessionLifetime),
		CreatedIP:        ipAddress,
		CreatedUserAgent: userAgent,
	}
	if err := s.sessionStore.Create(ctx, sess); err != nil {
		return nil, apperror.Internal("failed to create session")
	}
	return &AuthResult{
		SessionToken: raw,
		CSRFToken:    middleware.ComputeCSRFToken(raw, s.serverSecret),
		ExpiresIn:    int64(SessionLifetime.Seconds()),
	}, nil
}
