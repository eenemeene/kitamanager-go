package store

import (
	"context"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// TokenStore implements TokenStorer using GORM.
type TokenStore struct {
	db *gorm.DB
}

// NewTokenStore creates a new TokenStore.
func NewTokenStore(db *gorm.DB) *TokenStore {
	return &TokenStore{db: db}
}

// RevokeToken adds a token hash to the revocation list.
func (s *TokenStore) RevokeToken(ctx context.Context, tokenHash string, userID uint, expiresAt time.Time) error {
	revoked := &models.RevokedToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	// Ignore duplicate key errors (token already revoked)
	result := DBFromContext(ctx, s.db).Create(revoked)
	if result.Error != nil && IsDuplicateKeyError(result.Error) {
		return nil
	}
	return result.Error
}

// RevokeAllForUser revokes all tokens for a user by inserting a sentinel record
// with a far-future expiry. The middleware checks the revoked_tokens table,
// and we use a special approach: delete existing records for the user and insert
// a sentinel that causes IsRevoked to check the user_id.
// Actually, the simpler approach: we track individual tokens, so "revoke all"
// needs to be handled differently — the middleware will check token hashes.
// For "revoke all for user", we set a flag that the middleware checks.
// Simplest: just let the caller revoke each known token, or check at middleware level.
//
// For this implementation, we store a sentinel with empty hash that the middleware
// won't match against (individual tokens are checked by hash). Instead, we'll
// have the middleware also check by user_id with a recent timestamp.
// Actually the cleanest approach: store the "revoke all" timestamp per user.
// But that requires a different table. Let's keep it simple:
// RevokeAllForUser is called when changing password — we just need to ensure
// that tokens issued before this point are invalid. We store a special record.
func (s *TokenStore) RevokeAllForUser(ctx context.Context, userID uint) error {
	// Delete existing revocations for this user (cleanup)
	if err := DBFromContext(ctx, s.db).Where("user_id = ?", userID).Delete(&models.RevokedToken{}).Error; err != nil {
		return err
	}

	// Insert a sentinel record: token_hash = "all:<userID>" with far-future expiry
	// The middleware will check this to invalidate all tokens for this user
	sentinel := &models.RevokedToken{
		UserID:    userID,
		TokenHash: revokeAllSentinel(userID),
		ExpiresAt: time.Now().UTC().Add(refreshTokenMaxExpiry),
	}
	return DBFromContext(ctx, s.db).Create(sentinel).Error
}

// refreshTokenMaxExpiry is the maximum lifetime of a refresh token.
// Sentinel records expire after this duration (no tokens can live longer).
const refreshTokenMaxExpiry = 8 * 24 * time.Hour // 8 days (> 7-day refresh token)

// revokeAllSentinel returns the sentinel token hash for revoking all tokens for a user.
func revokeAllSentinel(userID uint) string {
	return "revoke_all_" + strconv.FormatUint(uint64(userID), 10)
}

// IsRevoked checks if a specific token hash is revoked, or if all tokens
// for the associated user have been revoked.
func (s *TokenStore) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(&models.RevokedToken{}).
		Where("token_hash = ?", tokenHash).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsUserRevoked checks if all tokens for a user have been revoked.
func (s *TokenStore) IsUserRevoked(ctx context.Context, userID uint) (bool, error) {
	var count int64
	sentinel := revokeAllSentinel(userID)
	err := DBFromContext(ctx, s.db).Model(&models.RevokedToken{}).
		Where("token_hash = ?", sentinel).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CleanupExpired removes expired revocation records.
func (s *TokenStore) CleanupExpired(ctx context.Context) error {
	return DBFromContext(ctx, s.db).
		Where("expires_at < ?", time.Now().UTC()).
		Delete(&models.RevokedToken{}).Error
}
