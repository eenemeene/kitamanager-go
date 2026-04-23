package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// SessionTokenBytes is the length of the raw random bytes behind a session
// token. 32 bytes = 256 bits of entropy, encoded as base64url (~43 chars).
const SessionTokenBytes = 32

// SessionDefaultLifetime is how long a newly issued session remains valid.
// Matches the old refresh-token lifetime so the user-visible auth window
// after login is unchanged.
const SessionDefaultLifetime = 7 * 24 * time.Hour

// SessionLookupResult is what `Lookup` returns: enough user state for the
// middleware to populate request context without needing a second query.
type SessionLookupResult struct {
	UserID     uint
	UserEmail  string
	UserActive bool
	ExpiresAt  time.Time
}

// SessionStore implements SessionStorer using GORM.
type SessionStore struct {
	db *gorm.DB
}

// NewSessionStore creates a new SessionStore.
func NewSessionStore(db *gorm.DB) *SessionStore {
	return &SessionStore{db: db}
}

// GenerateSessionToken returns (raw, hashed). The raw value goes into the
// client cookie, the hashed value is what Create stores.
func GenerateSessionToken() (raw string, hashed string, err error) {
	b := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	hashed = HashSessionToken(raw)
	return raw, hashed, nil
}

// HashSessionToken returns sha256(raw) as hex. The middleware hashes the
// incoming cookie value with this function before looking the row up.
func HashSessionToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// SessionPendingLookupResult is the pending_mfa lookup shape. It
// carries the caller's user id + the row's attempt counter so the
// service layer can enforce the per-row failure limit without a
// second query.
type SessionPendingLookupResult struct {
	UserID               uint
	UserActive           bool
	UserEmail            string
	ExpiresAt            time.Time
	MFAChallengeFailures int
	ChallengeNonce       []byte
}

// Create inserts a new session row. `idHash` must be the sha256 hex of the
// raw cookie value — use GenerateSessionToken.
func (s *SessionStore) Create(ctx context.Context, sess *models.Session) error {
	return DBFromContext(ctx, s.db).Create(sess).Error
}

// Lookup returns the REGULAR session + user row joined by user_id,
// scoped to non-expired rows only. pending_mfa rows are explicitly
// excluded: they must never satisfy RequireAuth — letting a pending
// handle through cookie would collapse the whole two-step flow back
// into a password-only login.
//
// ErrNotFound is returned when the session is absent, expired, or is
// not a regular session.
func (s *SessionStore) Lookup(ctx context.Context, idHash string) (*SessionLookupResult, error) {
	var row struct {
		UserID     uint
		UserEmail  string
		UserActive bool
		ExpiresAt  time.Time
	}
	err := DBFromContext(ctx, s.db).
		Table("sessions").
		Select("sessions.user_id AS user_id, users.email AS user_email, users.active AS user_active, sessions.expires_at AS expires_at").
		Joins("JOIN users ON users.id = sessions.user_id").
		Where("sessions.id = ? AND sessions.kind = ? AND sessions.expires_at > ?",
			idHash, models.SessionKindRegular, time.Now().UTC()).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &SessionLookupResult{
		UserID:     row.UserID,
		UserEmail:  row.UserEmail,
		UserActive: row.UserActive,
		ExpiresAt:  row.ExpiresAt,
	}, nil
}

// LookupPendingMFA is the counterpart to Lookup for the two-step login
// state. Returns the pending row's user binding + attempt counter so
// the verify path can decide (accept / bump / destroy) in one query.
// Scoped strictly to kind='pending_mfa' so a regular session value
// cannot be pointed at /auth/mfa/verify.
func (s *SessionStore) LookupPendingMFA(ctx context.Context, idHash string) (*SessionPendingLookupResult, error) {
	var row struct {
		UserID               uint
		UserEmail            string
		UserActive           bool
		ExpiresAt            time.Time
		MFAChallengeFailures int
		ChallengeNonce       []byte
	}
	err := DBFromContext(ctx, s.db).
		Table("sessions").
		Select("sessions.user_id AS user_id, users.email AS user_email, users.active AS user_active, sessions.expires_at AS expires_at, sessions.mfa_challenge_failures AS mfa_challenge_failures, sessions.challenge_nonce AS challenge_nonce").
		Joins("JOIN users ON users.id = sessions.user_id").
		Where("sessions.id = ? AND sessions.kind = ? AND sessions.expires_at > ?",
			idHash, models.SessionKindPendingMFA, time.Now().UTC()).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &SessionPendingLookupResult{
		UserID:               row.UserID,
		UserEmail:            row.UserEmail,
		UserActive:           row.UserActive,
		ExpiresAt:            row.ExpiresAt,
		MFAChallengeFailures: row.MFAChallengeFailures,
		ChallengeNonce:       row.ChallengeNonce,
	}, nil
}

// BumpMFAChallengeFailures atomically increments mfa_challenge_failures
// on a pending_mfa row and returns the post-increment value. Returns
// ErrNotFound if the row is missing, expired, or not pending_mfa, so
// the caller can translate cleanly without a second query.
//
// Scoped to kind='pending_mfa' so that a regular session ID smuggled
// in here would be a no-op. The post-increment value is read within
// the same UPDATE via RETURNING to avoid the read-increment-read race
// two concurrent verify attempts could otherwise exploit to each
// observe "4 < 5, still OK".
func (s *SessionStore) BumpMFAChallengeFailures(ctx context.Context, idHash string) (int, error) {
	var updated struct {
		MFAChallengeFailures int
	}
	err := DBFromContext(ctx, s.db).
		Raw(
			`UPDATE sessions
			 SET mfa_challenge_failures = mfa_challenge_failures + 1
			 WHERE id = ?
			   AND kind = ?
			   AND expires_at > ?
			 RETURNING mfa_challenge_failures`,
			idHash, models.SessionKindPendingMFA, time.Now().UTC()).
		Scan(&updated).Error
	if err != nil {
		return 0, err
	}
	if updated.MFAChallengeFailures == 0 {
		// RETURNING yielded no row — pending not found / expired / wrong kind.
		return 0, ErrNotFound
	}
	return updated.MFAChallengeFailures, nil
}

// DeletePendingMFA removes a single pending_mfa row by id-hash. Scoped
// on kind so a malformed call cannot accidentally delete a regular
// session. Idempotent; rows missing are silently a no-op.
func (s *SessionStore) DeletePendingMFA(ctx context.Context, idHash string) error {
	return DBFromContext(ctx, s.db).
		Where("id = ? AND kind = ?", idHash, models.SessionKindPendingMFA).
		Delete(&models.Session{}).Error
}

// Delete removes a single session. Returns nil when the row does not exist
// so logout is idempotent.
func (s *SessionStore) Delete(ctx context.Context, idHash string) error {
	return DBFromContext(ctx, s.db).
		Where("id = ?", idHash).
		Delete(&models.Session{}).Error
}

// DeleteAllForUser removes every session for the user. Used on password
// change (admin-initiated reset) and on account deactivation. Atomic single
// statement — no sentinel/race dance.
func (s *SessionStore) DeleteAllForUser(ctx context.Context, userID uint) error {
	return DBFromContext(ctx, s.db).
		Where("user_id = ?", userID).
		Delete(&models.Session{}).Error
}

// DeleteAllForUserExcept removes every session for the user except the one
// whose id-hash is `keepIDHash`. Used by self-service password change so the
// caller keeps their current session, everybody else gets logged out.
func (s *SessionStore) DeleteAllForUserExcept(ctx context.Context, userID uint, keepIDHash string) error {
	return DBFromContext(ctx, s.db).
		Where("user_id = ? AND id <> ?", userID, keepIDHash).
		Delete(&models.Session{}).Error
}

// CleanupExpired removes expired sessions. Called on a ticker.
func (s *SessionStore) CleanupExpired(ctx context.Context) error {
	return DBFromContext(ctx, s.db).
		Where("expires_at < ?", time.Now().UTC()).
		Delete(&models.Session{}).Error
}

// ListForUser returns every non-expired REGULAR session belonging to
// the user, ordered by most recently created first. Used by the
// /me/sessions endpoint so a user can see where they're signed in.
// pending_mfa rows are excluded — they're mid-flow, not "where I'm
// signed in."
func (s *SessionStore) ListForUser(ctx context.Context, userID uint) ([]models.Session, error) {
	var sessions []models.Session
	err := DBFromContext(ctx, s.db).
		Where("user_id = ? AND kind = ? AND expires_at > ?",
			userID, models.SessionKindRegular, time.Now().UTC()).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// DeleteForUser removes a session iff it belongs to the given user.
// Returns the number of rows affected — 0 means the session id did not
// match a row belonging to this user, which the caller should surface as
// 404. Scoping on user_id prevents a user from revoking another user's
// session even if they learned the target's id-hash somehow.
func (s *SessionStore) DeleteForUser(ctx context.Context, idHash string, userID uint) (int64, error) {
	result := DBFromContext(ctx, s.db).
		Where("id = ? AND user_id = ?", idHash, userID).
		Delete(&models.Session{})
	return result.RowsAffected, result.Error
}
