package store

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestSessionStore_CreateAndLookup(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	raw, hashed, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if raw == "" {
		t.Fatal("raw token is empty")
	}
	if hashed != HashSessionToken(raw) {
		t.Error("hashed output must match HashSessionToken(raw)")
	}

	sess := &models.Session{
		ID:        hashed,
		UserID:    user.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		CreatedIP: "127.0.0.1",
	}
	if err := s.Create(ctx, sess); err != nil {
		t.Fatalf("create: %v", err)
	}

	lookup, err := s.Lookup(ctx, hashed)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if lookup.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", lookup.UserID, user.ID)
	}
	if lookup.UserEmail != "test@example.com" {
		t.Errorf("UserEmail = %q, want test@example.com", lookup.UserEmail)
	}
	if !lookup.UserActive {
		t.Error("UserActive must be true for an active user")
	}
}

func TestSessionStore_Lookup_Expired(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	_, hashed, _ := GenerateSessionToken()
	sess := &models.Session{
		ID:        hashed,
		UserID:    user.ID,
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-time.Hour), // already expired
	}
	if err := s.Create(ctx, sess); err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := s.Lookup(ctx, hashed); err != ErrNotFound {
		t.Errorf("expected ErrNotFound for expired session, got %v", err)
	}
}

func TestSessionStore_Lookup_Missing(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()

	_, err := s.Lookup(ctx, HashSessionToken("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSessionStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "Test User", "test@example.com")

	_, hashed, _ := GenerateSessionToken()
	sess := &models.Session{
		ID:        hashed,
		UserID:    user.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := s.Create(ctx, sess); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := s.Delete(ctx, hashed); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := s.Lookup(ctx, hashed); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSessionStore_Delete_Missing_IsNoOp(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()

	if err := s.Delete(ctx, HashSessionToken("never-existed")); err != nil {
		t.Errorf("delete of missing session must be a no-op, got %v", err)
	}
}

func TestSessionStore_DeleteAllForUser(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	u1 := createTestUser(t, db, "User 1", "u1@example.com")
	u2 := createTestUser(t, db, "User 2", "u2@example.com")

	for _, uid := range []uint{u1.ID, u1.ID, u2.ID} {
		_, hashed, _ := GenerateSessionToken()
		sess := &models.Session{
			ID:        hashed,
			UserID:    uid,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}
		if err := s.Create(ctx, sess); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	if err := s.DeleteAllForUser(ctx, u1.ID); err != nil {
		t.Fatalf("DeleteAllForUser: %v", err)
	}

	var countU1, countU2 int64
	db.Model(&models.Session{}).Where("user_id = ?", u1.ID).Count(&countU1)
	db.Model(&models.Session{}).Where("user_id = ?", u2.ID).Count(&countU2)
	if countU1 != 0 {
		t.Errorf("user1 sessions should be gone, got %d remaining", countU1)
	}
	if countU2 != 1 {
		t.Errorf("user2 sessions must be untouched, got %d (want 1)", countU2)
	}
}

func TestSessionStore_DeleteAllForUserExcept(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "User", "u@example.com")

	_, keep, _ := GenerateSessionToken()
	_, other1, _ := GenerateSessionToken()
	_, other2, _ := GenerateSessionToken()
	for _, h := range []string{keep, other1, other2} {
		if err := s.Create(ctx, &models.Session{
			ID:        h,
			UserID:    user.ID,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	if err := s.DeleteAllForUserExcept(ctx, user.ID, keep); err != nil {
		t.Fatalf("DeleteAllForUserExcept: %v", err)
	}

	// keep survives
	if _, err := s.Lookup(ctx, keep); err != nil {
		t.Errorf("kept session missing: %v", err)
	}
	// others gone
	for _, h := range []string{other1, other2} {
		if _, err := s.Lookup(ctx, h); err != ErrNotFound {
			t.Errorf("other session should be gone, got %v", err)
		}
	}
}

func TestSessionStore_CleanupExpired(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "User", "u@example.com")

	_, expiredHash, _ := GenerateSessionToken()
	_, liveHash, _ := GenerateSessionToken()

	if err := s.Create(ctx, &models.Session{
		ID:        expiredHash,
		UserID:    user.ID,
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create expired: %v", err)
	}
	if err := s.Create(ctx, &models.Session{
		ID:        liveHash,
		UserID:    user.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create live: %v", err)
	}

	if err := s.CleanupExpired(ctx); err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}

	var count int64
	db.Model(&models.Session{}).Where("id = ?", expiredHash).Count(&count)
	if count != 0 {
		t.Errorf("expired session should be cleaned up, %d rows remain", count)
	}
	db.Model(&models.Session{}).Where("id = ?", liveHash).Count(&count)
	if count != 1 {
		t.Errorf("live session must survive cleanup, got %d rows", count)
	}
}

func TestHashSessionToken_Deterministic(t *testing.T) {
	a1 := HashSessionToken("abc")
	a2 := HashSessionToken("abc")
	if a1 != a2 {
		t.Error("HashSessionToken must be deterministic for the same input")
	}
	if a1 == HashSessionToken("abd") {
		t.Error("HashSessionToken must differ for different inputs")
	}
}

func TestGenerateSessionToken_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		raw, _, err := GenerateSessionToken()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if seen[raw] {
			t.Fatalf("duplicate token generated: %q", raw)
		}
		seen[raw] = true
	}
}

func TestSessionStore_ListForUser_Empty(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "User", "u@example.com")

	sessions, err := s.ListForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestSessionStore_ListForUser_OnlyOwnSessions(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	alice := createTestUser(t, db, "Alice", "alice@example.com")
	bob := createTestUser(t, db, "Bob", "bob@example.com")

	// Seed two sessions for Alice, one for Bob.
	for _, u := range []uint{alice.ID, alice.ID, bob.ID} {
		_, h, _ := GenerateSessionToken()
		if err := s.Create(ctx, &models.Session{
			ID: h, UserID: u, CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	aliceRows, err := s.ListForUser(ctx, alice.ID)
	if err != nil {
		t.Fatalf("ListForUser alice: %v", err)
	}
	if len(aliceRows) != 2 {
		t.Errorf("expected 2 rows for alice, got %d", len(aliceRows))
	}
	for _, r := range aliceRows {
		if r.UserID != alice.ID {
			t.Errorf("row leaked with UserID=%d", r.UserID)
		}
	}

	bobRows, _ := s.ListForUser(ctx, bob.ID)
	if len(bobRows) != 1 {
		t.Errorf("expected 1 row for bob, got %d", len(bobRows))
	}
}

func TestSessionStore_ListForUser_ExcludesExpired(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "User", "u@example.com")

	// One live, one expired.
	_, live, _ := GenerateSessionToken()
	_, expired, _ := GenerateSessionToken()
	if err := s.Create(ctx, &models.Session{
		ID: live, UserID: user.ID, CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create live: %v", err)
	}
	if err := s.Create(ctx, &models.Session{
		ID: expired, UserID: user.ID, CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create expired: %v", err)
	}

	rows, err := s.ListForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 live row, got %d", len(rows))
	}
	if rows[0].ID != live {
		t.Errorf("returned wrong row: got %q, want live %q", rows[0].ID, live)
	}
}

func TestSessionStore_ListForUser_OrderedByCreatedAtDesc(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "User", "u@example.com")

	// Three rows created 3h/2h/1h ago respectively. Must come back newest-first.
	now := time.Now().UTC()
	created := []time.Time{
		now.Add(-3 * time.Hour),
		now.Add(-2 * time.Hour),
		now.Add(-1 * time.Hour),
	}
	ids := []string{"", "", ""}
	for i, ts := range created {
		_, h, _ := GenerateSessionToken()
		ids[i] = h
		if err := s.Create(ctx, &models.Session{
			ID: h, UserID: user.ID, CreatedAt: ts,
			ExpiresAt: now.Add(time.Hour),
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	rows, err := s.ListForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// rows[0] must be the newest (ids[2]), rows[2] the oldest (ids[0]).
	if rows[0].ID != ids[2] {
		t.Errorf("newest row first: got %q, want %q", rows[0].ID, ids[2])
	}
	if rows[2].ID != ids[0] {
		t.Errorf("oldest row last: got %q, want %q", rows[2].ID, ids[0])
	}
}

func TestSessionStore_DeleteForUser_Success(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "User", "u@example.com")

	_, h, _ := GenerateSessionToken()
	if err := s.Create(ctx, &models.Session{
		ID: h, UserID: user.ID, CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	rows, err := s.DeleteForUser(ctx, h, user.ID)
	if err != nil {
		t.Fatalf("DeleteForUser: %v", err)
	}
	if rows != 1 {
		t.Errorf("rowsAffected = %d, want 1", rows)
	}
	if _, err := s.Lookup(ctx, h); err != ErrNotFound {
		t.Errorf("session should be gone after delete, got %v", err)
	}
}

func TestSessionStore_DeleteForUser_WrongUser_IsNoOp(t *testing.T) {
	// Critical security invariant: a user cannot revoke another user's
	// session even if they guess or exfiltrate the target's id-hash.
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	alice := createTestUser(t, db, "Alice", "alice@example.com")
	bob := createTestUser(t, db, "Bob", "bob@example.com")

	_, h, _ := GenerateSessionToken()
	if err := s.Create(ctx, &models.Session{
		ID: h, UserID: alice.ID, CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Bob tries to delete Alice's session by its id-hash.
	rows, err := s.DeleteForUser(ctx, h, bob.ID)
	if err != nil {
		t.Fatalf("DeleteForUser: %v", err)
	}
	if rows != 0 {
		t.Errorf("rowsAffected = %d, want 0 (cross-user delete must be a no-op)", rows)
	}
	// Alice's session must still exist.
	if _, err := s.Lookup(ctx, h); err != nil {
		t.Errorf("alice's session was wrongly deleted: %v", err)
	}
}

func TestSessionStore_DeleteForUser_UnknownID_IsNoOp(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	ctx := context.Background()
	user := createTestUser(t, db, "User", "u@example.com")

	rows, err := s.DeleteForUser(ctx, HashSessionToken("never-existed"), user.ID)
	if err != nil {
		t.Errorf("expected no error for unknown id, got %v", err)
	}
	if rows != 0 {
		t.Errorf("rowsAffected = %d, want 0 for unknown id", rows)
	}
}
