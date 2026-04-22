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

// Create inserts a new session row. `idHash` must be the sha256 hex of the
// raw cookie value — use GenerateSessionToken.
func (s *SessionStore) Create(ctx context.Context, sess *models.Session) error {
	return DBFromContext(ctx, s.db).Create(sess).Error
}

// Lookup returns the session + user row joined by user_id, scoped to
// non-expired rows only. ErrNotFound is returned when the session is absent
// or expired.
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
		Where("sessions.id = ? AND sessions.expires_at > ?", idHash, time.Now().UTC()).
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
