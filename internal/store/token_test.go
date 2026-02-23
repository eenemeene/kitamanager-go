package store

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestTokenStore_RevokeToken(t *testing.T) {
	db := setupTestDB(t)
	store := NewTokenStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	err := store.RevokeToken(ctx, "abc123hash", user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var count int64
	db.Model(&models.RevokedToken{}).Where("token_hash = ?", "abc123hash").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 revoked token row, got %d", count)
	}
}

func TestTokenStore_RevokeToken_DuplicateIgnored(t *testing.T) {
	db := setupTestDB(t)
	store := NewTokenStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	expiry := time.Now().Add(time.Hour)
	if err := store.RevokeToken(ctx, "dup_hash", user.ID, expiry); err != nil {
		t.Fatalf("first revoke: %v", err)
	}

	// Revoking the same token again should not error.
	if err := store.RevokeToken(ctx, "dup_hash", user.ID, expiry); err != nil {
		t.Fatalf("duplicate revoke should be ignored, got %v", err)
	}

	var count int64
	db.Model(&models.RevokedToken{}).Where("token_hash = ?", "dup_hash").Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 row after duplicate insert, got %d", count)
	}
}

func TestTokenStore_IsRevoked(t *testing.T) {
	db := setupTestDB(t)
	store := NewTokenStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	// Not revoked initially.
	revoked, err := store.IsRevoked(ctx, "some_hash")
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if revoked {
		t.Error("expected token to not be revoked")
	}

	// Revoke it.
	if err := store.RevokeToken(ctx, "some_hash", user.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	revoked, err = store.IsRevoked(ctx, "some_hash")
	if err != nil {
		t.Fatalf("IsRevoked after revoke: %v", err)
	}
	if !revoked {
		t.Error("expected token to be revoked")
	}
}

func TestTokenStore_RevokeAllForUser(t *testing.T) {
	db := setupTestDB(t)
	store := NewTokenStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	// Revoke individual tokens first.
	if err := store.RevokeToken(ctx, "token1", user.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
	if err := store.RevokeToken(ctx, "token2", user.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	// Revoke all.
	if err := store.RevokeAllForUser(ctx, user.ID); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}

	// Individual tokens should be cleaned up, replaced by sentinel.
	var count int64
	db.Model(&models.RevokedToken{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 sentinel row, got %d", count)
	}

	// Sentinel should be detectable via IsUserRevoked.
	revoked, err := store.IsUserRevoked(ctx, user.ID)
	if err != nil {
		t.Fatalf("IsUserRevoked: %v", err)
	}
	if !revoked {
		t.Error("expected user to be revoked after RevokeAllForUser")
	}
}

func TestTokenStore_IsUserRevoked_NotRevoked(t *testing.T) {
	db := setupTestDB(t)
	store := NewTokenStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	revoked, err := store.IsUserRevoked(ctx, user.ID)
	if err != nil {
		t.Fatalf("IsUserRevoked: %v", err)
	}
	if revoked {
		t.Error("expected user NOT to be revoked initially")
	}
}

func TestTokenStore_CleanupExpired(t *testing.T) {
	db := setupTestDB(t)
	store := NewTokenStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	// Insert an already-expired token.
	expired := &models.RevokedToken{
		UserID:    user.ID,
		TokenHash: "expired_hash",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	if err := db.Create(expired).Error; err != nil {
		t.Fatalf("create expired token: %v", err)
	}

	// Insert a still-valid token.
	if err := store.RevokeToken(ctx, "valid_hash", user.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	// Cleanup.
	if err := store.CleanupExpired(ctx); err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}

	// Expired should be gone, valid should remain.
	var count int64
	db.Model(&models.RevokedToken{}).Where("token_hash = ?", "expired_hash").Count(&count)
	if count != 0 {
		t.Errorf("expected expired token to be cleaned up, but found %d rows", count)
	}

	db.Model(&models.RevokedToken{}).Where("token_hash = ?", "valid_hash").Count(&count)
	if count != 1 {
		t.Errorf("expected valid token to remain, but found %d rows", count)
	}
}

func TestTokenStore_RevokeAllForUser_IsolatesUsers(t *testing.T) {
	db := setupTestDB(t)
	store := NewTokenStore(db)
	ctx := context.Background()
	user1 := createTestUser(t, db, "User 1", "user1@example.com")
	user2 := createTestUser(t, db, "User 2", "user2@example.com")

	// Revoke token for user2.
	if err := store.RevokeToken(ctx, "user2_token", user2.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	// Revoke all for user1.
	if err := store.RevokeAllForUser(ctx, user1.ID); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}

	// User2's token should still exist.
	revoked, err := store.IsRevoked(ctx, "user2_token")
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if !revoked {
		t.Error("user2's individual token should still be revoked")
	}

	// User2 should NOT be marked as all-revoked.
	userRevoked, err := store.IsUserRevoked(ctx, user2.ID)
	if err != nil {
		t.Fatalf("IsUserRevoked: %v", err)
	}
	if userRevoked {
		t.Error("user2 should NOT be user-revoked after revoking user1")
	}
}
