package store

import (
	"context"
	"errors"
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

// RevokeAllForUser revokes every token issued to the user up to the moment of
// the call. It works by writing a sentinel row whose CreatedAt acts as a
// cutoff: the middleware refuses any token whose `iat` claim lies before this
// cutoff. Tokens issued *after* the sentinel was written remain valid — this
// is what makes self-initiated password change work (the caller receives a
// fresh token in the same request) while still locking out other holders of
// older tokens. See #134.
func (s *TokenStore) RevokeAllForUser(ctx context.Context, userID uint) error {
	// Drop any older rows for this user: the cutoff they encoded is now
	// superseded by the new sentinel, and the individual-token rows they
	// protected are likewise covered by the cutoff.
	if err := DBFromContext(ctx, s.db).Where("user_id = ?", userID).Delete(&models.RevokedToken{}).Error; err != nil {
		return err
	}

	sentinel := &models.RevokedToken{
		UserID:    userID,
		TokenHash: revokeAllSentinel(userID),
		ExpiresAt: time.Now().UTC().Add(refreshTokenMaxExpiry),
		// Set CreatedAt explicitly: correctness of revocation now depends on
		// this value, so do not rely on GORM's auto-populate hook.
		CreatedAt: time.Now().UTC(),
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

// IsUserRevokedSince reports whether a token with the given `iat` (issued-at)
// has been covered by a prior RevokeAllForUser for this user. Returns true
// iff a sentinel row exists whose CreatedAt is strictly after tokenIssuedAt.
//
// Because JWT `iat` is second-precision while the sentinel's CreatedAt has
// sub-second precision, callers that need to issue fresh tokens immediately
// after RevokeAllForUser must wait past the next whole-second boundary (see
// AuthService.ChangePassword), otherwise the new tokens' second-truncated iat
// would tie with the sentinel's CreatedAt and be treated as revoked. See #134.
func (s *TokenStore) IsUserRevokedSince(ctx context.Context, userID uint, tokenIssuedAt time.Time) (bool, error) {
	var sentinel models.RevokedToken
	err := DBFromContext(ctx, s.db).
		Where("token_hash = ?", revokeAllSentinel(userID)).
		First(&sentinel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return tokenIssuedAt.Before(sentinel.CreatedAt), nil
}

// CleanupExpired removes expired revocation records.
func (s *TokenStore) CleanupExpired(ctx context.Context) error {
	return DBFromContext(ctx, s.db).
		Where("expires_at < ?", time.Now().UTC()).
		Delete(&models.RevokedToken{}).Error
}
