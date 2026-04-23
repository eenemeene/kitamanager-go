package service

import (
	"context"
	"errors"
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

	// PendingMFALifetime caps how long a pending_mfa row is accepted on
	// /auth/mfa/verify. 5 minutes is the Auth0/Okta industry range's
	// tighter end — plenty for human UX (scan QR + enter 6 digits) and
	// short enough that a stolen pending token has a tight window.
	PendingMFALifetime = 5 * time.Minute

	// MFAChallengeFailureLimit caps wrong codes per pending_mfa row.
	// Hitting it destroys the row and forces the user to restart with
	// the password step. 5 matches the per-user activation limit and
	// stays well below NIST SP 800-63B §5.2.2's cap of 100 failures
	// per account.
	MFAChallengeFailureLimit = 5

	// mfaPerUserLockoutThreshold / mfaPerUserLockoutWindow back the
	// per-pending-row counter: an attacker cycling through freshly-
	// issued pending rows would otherwise blow past the per-row cap.
	// 20 wrong codes in 15 minutes across all pending rows for the
	// same user locks further attempts at the verify endpoint.
	mfaPerUserLockoutThreshold = 20
	mfaPerUserLockoutWindow    = 15 * time.Minute
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

// PendingMFAResult is issued by /login when the user has an active
// primary factor. The `PendingToken` is the raw value to hand back on
// /auth/mfa/verify; the server stores sha256 of it in the pending_mfa
// session row. ExpiresAt is carried through so the client shows a
// clear countdown and doesn't try to reuse an expired handle.
type PendingMFAResult struct {
	PendingToken string
	ExpiresAt    time.Time
	Factors      []models.LoginFactorDescriptor
}

// LoginResult is the union returned by Login. Exactly one field is
// non-nil:
//   - Authenticated populated for users with no primary factor —
//     same shape as pre-MFA Login.
//   - Pending populated for users with an active primary factor; the
//     caller must follow up with VerifyMFALogin.
//
// Handlers should check Pending first (MFA was triggered) and fall
// through to Authenticated otherwise.
type LoginResult struct {
	Authenticated *AuthResult
	Pending       *PendingMFAResult
}

// AuthService handles authentication business logic.
type AuthService struct {
	userStore     store.UserStorer
	sessionStore  store.SessionStorer
	serverSecret  string
	auditService  *AuditService
	factorService *FactorService
	// dummyPasswordHash equalizes Login timing on the not-found path.
	// Without it, bcrypt is skipped when the email is unknown and response
	// latency acts as an account enumeration oracle.
	dummyPasswordHash []byte
}

// NewAuthService creates a new auth service. `serverSecret` is used for CSRF
// HMAC derivation; the existing JWT_SECRET config value is reused because it
// already has a 32-char floor. `factorService` may be nil in tests that do
// not exercise two-step login; Login will then behave as if no user has
// MFA enrolled.
func NewAuthService(userStore store.UserStorer, sessionStore store.SessionStorer, serverSecret string, auditService *AuditService, factorService *FactorService) *AuthService {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("failed to generate dummy password hash: %v", err))
	}
	return &AuthService{
		userStore:         userStore,
		sessionStore:      sessionStore,
		serverSecret:      serverSecret,
		auditService:      auditService,
		factorService:     factorService,
		dummyPasswordHash: dummyHash,
	}
}

// canonicalEmail normalizes an email for storage and lookup so that case
// and whitespace variants map to the same row. Migration 000009 enforces a
// case-insensitive unique index on users.email and rewrote every existing
// row to lowercase; callers that insert or look up an email must normalize
// with this function to keep that invariant.
func canonicalEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Login authenticates a user with email and password. If the user has
// no active primary factor, a regular session is issued immediately
// (LoginResult.Authenticated). If the user does have one, a short-
// lived pending_mfa row is issued instead (LoginResult.Pending); the
// caller must then call VerifyMFALogin with a code to exchange the
// pending token for a real session.
//
// The password-only audit event (`login`) is only emitted on the no-
// MFA path — the MFA path emits `login_mfa_required` at this step and
// the regular `login` at VerifyMFALogin time. This way dashboards that
// query the `login` action always match a fully-authenticated session.
func (s *AuthService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*LoginResult, error) {
	// Single canonical form is used everywhere: lockout counter, audit row,
	// and DB lookup. An attacker cannot reset the lockout by rotating case
	// because all three keys collapse to the same value.
	canonical := canonicalEmail(email)

	failedCount, err := s.auditService.CountRecentFailedLogins(ctx, canonical, lockoutWindow)
	if err == nil && failedCount >= lockoutThreshold {
		s.auditService.LogLoginFailed(canonical, ipAddress, userAgent, "account locked - too many failed attempts")
		return nil, apperror.TooManyRequests("too many failed login attempts, please try again later")
	}

	user, err := s.userStore.FindByEmail(ctx, canonical)
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

	// Check MFA enrolment. A nil factorService (some tests) behaves as
	// "no user has MFA enrolled" — safe because factorService is only
	// nil in paths that aren't exercising MFA.
	if s.factorService != nil {
		hasFactor, err := s.factorService.HasActivePrimaryFactor(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		if hasFactor {
			pending, err := s.issuePendingMFA(ctx, user.ID, ipAddress, userAgent)
			if err != nil {
				return nil, err
			}
			s.auditService.LogLoginMFARequired(user.ID, user.Email, ipAddress, userAgent)
			return &LoginResult{Pending: pending}, nil
		}
	}

	_ = s.userStore.UpdateLastLogin(ctx, user.ID)
	s.auditService.LogLogin(user.ID, user.Email, ipAddress, userAgent)

	authed, err := s.issueSession(ctx, user.ID, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Authenticated: authed}, nil
}

// VerifyMFALogin exchanges a pending_mfa token + code for a real
// session. Step-two of the two-step login.
//
// Flow:
//  1. Look up the pending row by sha256(pendingToken). Missing /
//     expired / wrong kind → 401 (uniform, no enumeration).
//  2. Enforce per-user lockout: too many failed challenges across any
//     pending row in the recent window → 429 and the current pending
//     row is destroyed.
//  3. Verify the code against the claimed factor via FactorService.
//     TOTP: compare-and-set on last_used_step; backup_codes: atomic
//     single-use.
//  4. On wrong code: bump the per-row counter. If it hits
//     MFAChallengeFailureLimit the pending row is destroyed
//     (mfa_challenge_locked audit event) and the response is 429.
//     Otherwise it is 401 (mfa_challenge_failed audit event).
//  5. On success: delete pending row, issue real session, audit
//     mfa_challenge_succeeded + login.
//
// User-active check happens on both sides: even a brief window where
// the user is deactivated between step 1 and step 2 must not allow
// completing the login. We treat a deactivated user as 401 to avoid
// leaking the admin action.
func (s *AuthService) VerifyMFALogin(ctx context.Context, pendingToken string, factorID uint, code, ipAddress, userAgent string) (*AuthResult, error) {
	if pendingToken == "" {
		return nil, apperror.Unauthorized("invalid pending token")
	}
	pendHash := store.HashSessionToken(pendingToken)

	pending, err := s.sessionStore.LookupPendingMFA(ctx, pendHash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.Unauthorized("invalid pending token")
		}
		return nil, apperror.InternalWrap(err, "lookup pending")
	}
	if !pending.UserActive {
		// Soft-delete the row so a subsequent call after the admin
		// un-deactivates the user does not still succeed.
		_ = s.sessionStore.DeletePendingMFA(ctx, pendHash)
		return nil, apperror.Unauthorized("invalid pending token")
	}

	// Per-user lockout gate. Runs BEFORE the factor verify to avoid
	// contributing to bcrypt/TOTP CPU on a known-hostile account.
	perUserFailures, err := s.auditService.CountRecentFailedMFAChallenges(ctx, pending.UserID, mfaPerUserLockoutWindow)
	if err == nil && perUserFailures >= mfaPerUserLockoutThreshold {
		_ = s.sessionStore.DeletePendingMFA(ctx, pendHash)
		s.auditService.LogMFAChallengeLocked(pending.UserID, pending.UserEmail, ipAddress, userAgent)
		return nil, apperror.TooManyRequests("too many failed MFA attempts, please try again later")
	}

	factorType, verifyErr := s.factorService.VerifyCodeForLogin(ctx, pending.UserID, factorID, code)
	if verifyErr != nil {
		// Wrong-factor or unknown-factor cases return NotFound from
		// VerifyCodeForLogin — we surface them as 401 to avoid
		// distinguishing "you picked a factor that isn't yours" from
		// "wrong code." The audit record preserves the detail.
		if errors.Is(verifyErr, apperror.ErrNotFound) {
			s.auditService.LogMFAChallengeFailed(pending.UserID, pending.UserEmail, "", ipAddress, userAgent, "unknown factor")
			// Do NOT bump the per-row counter: a totally invalid
			// factor id is more likely a UI bug than a brute force.
			return nil, apperror.Unauthorized("invalid code")
		}
		if errors.Is(verifyErr, apperror.ErrUnauthorized) {
			newCount, bumpErr := s.sessionStore.BumpMFAChallengeFailures(ctx, pendHash)
			if bumpErr != nil {
				// Pending row vanished mid-request (expired between
				// lookup and bump). Fail closed.
				slog.Error("bump mfa failures", "error", bumpErr)
				return nil, apperror.Unauthorized("invalid code")
			}
			if newCount >= MFAChallengeFailureLimit {
				_ = s.sessionStore.DeletePendingMFA(ctx, pendHash)
				s.auditService.LogMFAChallengeLocked(pending.UserID, pending.UserEmail, ipAddress, userAgent)
				return nil, apperror.TooManyRequests("too many wrong codes, please restart login")
			}
			s.auditService.LogMFAChallengeFailed(pending.UserID, pending.UserEmail, factorType, ipAddress, userAgent, "invalid code")
			return nil, apperror.Unauthorized("invalid code")
		}
		return nil, verifyErr
	}

	// Success — consume pending row, issue real session.
	if err := s.sessionStore.DeletePendingMFA(ctx, pendHash); err != nil {
		slog.Error("delete pending after verify", "error", err)
		// Fall through: the verify succeeded; orphan pending is
		// harmless and will be GC'd.
	}

	_ = s.userStore.UpdateLastLogin(ctx, pending.UserID)
	s.auditService.LogMFAChallengeSucceeded(pending.UserID, pending.UserEmail, factorType, ipAddress, userAgent)
	s.auditService.LogLogin(pending.UserID, pending.UserEmail, ipAddress, userAgent)

	return s.issueSession(ctx, pending.UserID, ipAddress, userAgent)
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

// ListSessions returns every non-expired session belonging to the user.
// `currentIDHash` is the id-hash of the caller's own session (from
// ctxkeys.SessionIDHash) so the response can mark which row is the
// session that served this request. Pass "" when called outside a
// request context.
func (s *AuthService) ListSessions(ctx context.Context, userID uint, currentIDHash string) ([]models.UserSessionResponse, error) {
	rows, err := s.sessionStore.ListForUser(ctx, userID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to list sessions")
	}
	out := make([]models.UserSessionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.UserSessionResponse{
			ID:               r.ID,
			CreatedAt:        r.CreatedAt,
			ExpiresAt:        r.ExpiresAt,
			CreatedIP:        r.CreatedIP,
			CreatedUserAgent: r.CreatedUserAgent,
			Current:          r.ID == currentIDHash,
		})
	}
	return out, nil
}

// RevokeSession deletes a single session belonging to the user. The store
// scopes the DELETE to `user_id = ?` so a user cannot revoke another
// user's session even if they learned the target's id-hash. Returns
// NotFound if no matching row exists, so callers leak neither the
// session's existence nor its owner.
func (s *AuthService) RevokeSession(ctx context.Context, userID uint, idHash string) error {
	rows, err := s.sessionStore.DeleteForUser(ctx, idHash, userID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to revoke session")
	}
	if rows == 0 {
		return apperror.NotFound("session")
	}
	return nil
}

// issuePendingMFA generates a pending_mfa row + raw token pair and
// attaches the current factor descriptors. The raw token is returned
// to the caller in the JSON body; the row carries sha256(raw).
func (s *AuthService) issuePendingMFA(ctx context.Context, userID uint, ipAddress, userAgent string) (*PendingMFAResult, error) {
	raw, hashed, err := store.GenerateSessionToken()
	if err != nil {
		return nil, apperror.Internal("failed to generate pending token")
	}
	now := time.Now().UTC()
	expires := now.Add(PendingMFALifetime)
	pv := now
	row := &models.Session{
		ID:                 hashed,
		UserID:             userID,
		Kind:               models.SessionKindPendingMFA,
		CreatedAt:          now,
		ExpiresAt:          expires,
		PasswordVerifiedAt: &pv,
		CreatedIP:          ipAddress,
		CreatedUserAgent:   userAgent,
	}
	if err := s.sessionStore.Create(ctx, row); err != nil {
		return nil, apperror.Internal("failed to create pending session")
	}

	factors, err := s.factorService.DescriptorsForLogin(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &PendingMFAResult{
		PendingToken: raw,
		ExpiresAt:    expires,
		Factors:      factors,
	}, nil
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
		Kind:             models.SessionKindRegular,
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
