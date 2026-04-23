package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// seedPending inserts a pending_mfa row directly. Returns the raw
// token (never persisted) and the id-hash stored in the row.
func seedPending(t *testing.T, s *SessionStore, userID uint, lifetime time.Duration) (raw, idHash string) {
	t.Helper()
	raw, idHash, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	now := time.Now().UTC()
	pv := now
	row := &models.Session{
		ID:                 idHash,
		UserID:             userID,
		Kind:               models.SessionKindPendingMFA,
		CreatedAt:          now,
		ExpiresAt:          now.Add(lifetime),
		PasswordVerifiedAt: &pv,
		CreatedIP:          "10.0.0.1",
		CreatedUserAgent:   "test-ua",
	}
	if err := s.Create(context.Background(), row); err != nil {
		t.Fatalf("create pending: %v", err)
	}
	return raw, idHash
}

func TestSessionStore_LookupPendingMFA_ReturnsRow(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	_, idHash := seedPending(t, s, user.ID, time.Minute)

	got, err := s.LookupPendingMFA(context.Background(), idHash)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got.UserID != user.ID {
		t.Errorf("user id mismatch: got %d, want %d", got.UserID, user.ID)
	}
	if got.UserEmail != user.Email {
		t.Errorf("email mismatch: got %q, want %q", got.UserEmail, user.Email)
	}
	if got.MFAChallengeFailures != 0 {
		t.Errorf("expected fresh counter = 0, got %d", got.MFAChallengeFailures)
	}
}

// Missing row → ErrNotFound, not an internal error.
func TestSessionStore_LookupPendingMFA_Missing(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	_, err := s.LookupPendingMFA(context.Background(), "no-such-row")
	if err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// Expired row → ErrNotFound. Scoping on expires_at > now() is what
// makes the lookup safe to use as a gate.
func TestSessionStore_LookupPendingMFA_Expired(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	_, idHash := seedPending(t, s, user.ID, -time.Minute)
	_, err := s.LookupPendingMFA(context.Background(), idHash)
	if err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// Kind isolation: a REGULAR session id-hash must not be looked up
// as pending_mfa. This is the security guarantee of the two-step
// architecture — if it broke, a stolen session cookie could be used
// as a pending token.
func TestSessionStore_LookupPendingMFA_KindIsolation(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	// Seed a regular session.
	_, regularHash, _ := GenerateSessionToken()
	now := time.Now().UTC()
	if err := s.Create(context.Background(), &models.Session{
		ID:        regularHash,
		UserID:    user.ID,
		Kind:      models.SessionKindRegular,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// LookupPendingMFA must NOT return it.
	if _, err := s.LookupPendingMFA(context.Background(), regularHash); err != ErrNotFound {
		t.Errorf("regular session id found via pending lookup: %v", err)
	}

	// And vice versa: a pending row must NOT be returned by Lookup()
	// (which powers RequireAuth).
	_, pendingHash := seedPending(t, s, user.ID, time.Minute)
	if _, err := s.Lookup(context.Background(), pendingHash); err != ErrNotFound {
		t.Errorf("pending row exposed to Lookup: %v", err)
	}
}

func TestSessionStore_BumpMFAChallengeFailures_Increments(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	_, idHash := seedPending(t, s, user.ID, time.Minute)

	for i := 1; i <= 3; i++ {
		got, err := s.BumpMFAChallengeFailures(context.Background(), idHash)
		if err != nil {
			t.Fatalf("bump %d: %v", i, err)
		}
		if got != i {
			t.Errorf("bump %d: returned %d, want %d", i, got, i)
		}
	}
}

// Two goroutines bumping concurrently: both must succeed AND return
// distinct post-increment values — the counter must not lose updates.
func TestSessionStore_BumpMFAChallengeFailures_ConcurrentNoLost(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	_, idHash := seedPending(t, s, user.ID, time.Minute)

	const n = 5
	var wg sync.WaitGroup
	returned := make([]int, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			returned[idx], errs[idx] = s.BumpMFAChallengeFailures(context.Background(), idHash)
		}(i)
	}
	wg.Wait()

	// All should succeed.
	for i, e := range errs {
		if e != nil {
			t.Errorf("goroutine %d: %v", i, e)
		}
	}
	// Distinct and cover 1..n.
	seen := map[int]bool{}
	for _, v := range returned {
		seen[v] = true
	}
	for i := 1; i <= n; i++ {
		if !seen[i] {
			t.Errorf("bump value %d missing — update was lost (returned=%v)", i, returned)
		}
	}
}

// Bump on a missing row returns ErrNotFound — the caller translates
// to 401.
func TestSessionStore_BumpMFAChallengeFailures_MissingRow(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	_, err := s.BumpMFAChallengeFailures(context.Background(), "no-such-hash")
	if err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// Bump on a REGULAR session is a no-op (row stays unaffected, caller
// gets ErrNotFound). Defence in depth against accidentally mutating
// regular sessions.
func TestSessionStore_BumpMFAChallengeFailures_RegularSession_NoOp(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	_, regularHash, _ := GenerateSessionToken()
	now := time.Now().UTC()
	if err := s.Create(context.Background(), &models.Session{
		ID: regularHash, UserID: user.ID, Kind: models.SessionKindRegular,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := s.BumpMFAChallengeFailures(context.Background(), regularHash)
	if err != ErrNotFound {
		t.Errorf("bump on regular session: got %v, want ErrNotFound", err)
	}

	// Regular row unaffected.
	var fails int
	_ = db.Model(&models.Session{}).Select("mfa_challenge_failures").
		Where("id = ?", regularHash).Scan(&fails)
	if fails != 0 {
		t.Errorf("regular session counter was mutated: %d", fails)
	}
}

// Delete pending is idempotent and scoped.
func TestSessionStore_DeletePendingMFA_IdempotentAndScoped(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	_, regularHash, _ := GenerateSessionToken()
	now := time.Now().UTC()
	if err := s.Create(context.Background(), &models.Session{
		ID: regularHash, UserID: user.ID, Kind: models.SessionKindRegular,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed regular: %v", err)
	}
	_, pendingHash := seedPending(t, s, user.ID, time.Minute)

	// Delete pending — regular must survive.
	if err := s.DeletePendingMFA(context.Background(), pendingHash); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var regN, pendN int64
	_ = db.Model(&models.Session{}).Where("id = ?", regularHash).Count(&regN).Error
	_ = db.Model(&models.Session{}).Where("id = ?", pendingHash).Count(&pendN).Error
	if regN != 1 {
		t.Error("regular session deleted by pending-delete")
	}
	if pendN != 0 {
		t.Error("pending row survived pending-delete")
	}

	// Re-delete is a no-op (not an error).
	if err := s.DeletePendingMFA(context.Background(), pendingHash); err != nil {
		t.Errorf("idempotent delete: %v", err)
	}
	// DeletePendingMFA scoped to kind — passing a regular id must NOT
	// delete it.
	if err := s.DeletePendingMFA(context.Background(), regularHash); err != nil {
		t.Errorf("delete on regular-id as pending: %v", err)
	}
	_ = db.Model(&models.Session{}).Where("id = ?", regularHash).Count(&regN).Error
	if regN != 1 {
		t.Error("regular session deleted via DeletePendingMFA (kind scope broken)")
	}
}

// ListForUser must exclude pending rows — the Settings page's "where
// am I signed in" should only show fully-authenticated sessions.
func TestSessionStore_ListForUser_ExcludesPending(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	_, regularHash, _ := GenerateSessionToken()
	now := time.Now().UTC()
	if err := s.Create(context.Background(), &models.Session{
		ID: regularHash, UserID: user.ID, Kind: models.SessionKindRegular,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed regular: %v", err)
	}
	_, _ = seedPending(t, s, user.ID, time.Minute)

	rows, err := s.ListForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ID != regularHash {
		t.Error("listed a non-regular row")
	}
}

// DeleteAllForUser wipes BOTH kinds — password change should never
// leave a user mid-flow-pending alive.
func TestSessionStore_DeleteAllForUser_IncludesPending(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	_, _, _ = GenerateSessionToken() // unused
	_, regularHash, _ := GenerateSessionToken()
	now := time.Now().UTC()
	if err := s.Create(context.Background(), &models.Session{
		ID: regularHash, UserID: user.ID, Kind: models.SessionKindRegular,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed regular: %v", err)
	}
	_, _ = seedPending(t, s, user.ID, time.Minute)

	if err := s.DeleteAllForUser(context.Background(), user.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var n int64
	_ = db.Model(&models.Session{}).Where("user_id = ?", user.ID).Count(&n).Error
	if n != 0 {
		t.Errorf("DeleteAllForUser left %d rows", n)
	}
}

// CleanupExpired removes pending rows too (both kinds share the
// expires_at column).
func TestSessionStore_CleanupExpired_IncludesPending(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	_, _ = seedPending(t, s, user.ID, -time.Minute) // expired
	_, _ = seedPending(t, s, user.ID, time.Minute)  // still alive

	if err := s.CleanupExpired(context.Background()); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	var n int64
	_ = db.Model(&models.Session{}).
		Where("user_id = ? AND kind = ?", user.ID, models.SessionKindPendingMFA).
		Count(&n).Error
	if n != 1 {
		t.Errorf("expected 1 pending after cleanup, got %d", n)
	}
}

// FK cascade: deleting a user removes their pending_mfa rows too.
func TestSessionStore_UserDelete_CascadesPending(t *testing.T) {
	db := setupTestDB(t)
	s := NewSessionStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	_, _ = seedPending(t, s, user.ID, time.Minute)

	if err := db.Delete(&models.User{}, user.ID).Error; err != nil {
		t.Fatalf("delete user: %v", err)
	}
	var n int64
	_ = db.Model(&models.Session{}).Where("user_id = ?", user.ID).Count(&n).Error
	if n != 0 {
		t.Errorf("user delete didn't cascade to pending rows: %d", n)
	}
}
