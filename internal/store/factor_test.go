package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// seedTOTPFactor inserts a TOTP factor + encrypted-secret subrow in one
// go, returning the factor id.
func seedTOTPFactor(t *testing.T, s *FactorStore, userID uint, activated bool) uint {
	t.Helper()
	ctx := context.Background()
	f := &models.Factor{
		UserID:    userID,
		Type:      models.FactorTypeTOTP,
		CreatedAt: time.Now().UTC(),
	}
	if activated {
		now := time.Now().UTC()
		f.EnabledAt = &now
	}
	if err := s.CreateFactor(ctx, f); err != nil {
		t.Fatalf("create factor: %v", err)
	}
	sec := make([]byte, 32)
	_, _ = rand.Read(sec)
	nonce := make([]byte, 12)
	_, _ = rand.Read(nonce)
	if err := s.CreateTOTPSecret(ctx, &models.FactorTOTPSecret{
		FactorID: f.ID, SecretCiphertext: sec, SecretNonce: nonce,
	}); err != nil {
		t.Fatalf("create totp secret: %v", err)
	}
	return f.ID
}

func hashCode(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func TestFactorStore_FindByIDAndUser_OwnerOnly(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	alice := createTestUser(t, db, "Alice", "a@example.com")
	bob := createTestUser(t, db, "Bob", "b@example.com")

	fid := seedTOTPFactor(t, s, alice.ID, true)

	// Alice can read her factor.
	if _, err := s.FindByIDAndUser(context.Background(), fid, alice.ID); err != nil {
		t.Errorf("alice finding her factor: %v", err)
	}
	// Bob cannot. Must be ErrNotFound — not any other distinguishable error.
	if _, err := s.FindByIDAndUser(context.Background(), fid, bob.ID); err != ErrNotFound {
		t.Errorf("bob finding alice's factor: got %v, want ErrNotFound", err)
	}
}

func TestFactorStore_ActivateFactor_Atomic(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	fid := seedTOTPFactor(t, s, user.ID, false) // pending

	// First activate returns true.
	ok, err := s.ActivateFactor(context.Background(), fid, user.ID)
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if !ok {
		t.Fatal("first activate should succeed")
	}
	// Second activate returns false — compare-and-set caught it.
	ok2, err := s.ActivateFactor(context.Background(), fid, user.ID)
	if err != nil {
		t.Fatalf("re-activate: %v", err)
	}
	if ok2 {
		t.Fatal("second activate must not succeed — factor already active")
	}
}

func TestFactorStore_ActivateFactor_CrossUser_IsNoOp(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	alice := createTestUser(t, db, "Alice", "a@example.com")
	bob := createTestUser(t, db, "Bob", "b@example.com")
	fid := seedTOTPFactor(t, s, alice.ID, false)

	// Bob cannot activate Alice's pending factor.
	ok, err := s.ActivateFactor(context.Background(), fid, bob.ID)
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if ok {
		t.Fatal("bob must not be able to activate alice's factor")
	}
}

func TestFactorStore_AcceptTOTPStep_ReplayRejected(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	fid := seedTOTPFactor(t, s, user.ID, true)

	ctx := context.Background()

	// First acceptance at step 100 — succeeds.
	ok, err := s.AcceptTOTPStep(ctx, fid, 100)
	if err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if !ok {
		t.Fatal("first accept should succeed")
	}

	// Replay: same step rejected.
	ok, err = s.AcceptTOTPStep(ctx, fid, 100)
	if err != nil {
		t.Fatalf("replay check: %v", err)
	}
	if ok {
		t.Fatal("replay of same step must be rejected")
	}

	// Earlier step rejected (catches library-tolerance replays).
	ok, err = s.AcceptTOTPStep(ctx, fid, 99)
	if err != nil {
		t.Fatalf("earlier-step check: %v", err)
	}
	if ok {
		t.Fatal("earlier step must be rejected (would allow tolerance replay)")
	}

	// Later step accepted — clock advanced.
	ok, err = s.AcceptTOTPStep(ctx, fid, 101)
	if err != nil {
		t.Fatalf("later step: %v", err)
	}
	if !ok {
		t.Fatal("later step should be accepted")
	}
}

func TestFactorStore_AcceptTOTPStep_ConcurrentRacesOneWinner(t *testing.T) {
	// Two goroutines trying to bump the same step. Exactly one should
	// see RowsAffected=1. The other sees 0 because the compare-and-set
	// WHERE clause fails on the second attempt.
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	fid := seedTOTPFactor(t, s, user.ID, true)

	ctx := context.Background()
	var wg sync.WaitGroup
	results := make([]bool, 2)
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = s.AcceptTOTPStep(ctx, fid, 50)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}
	winners := 0
	for _, ok := range results {
		if ok {
			winners++
		}
	}
	if winners != 1 {
		t.Errorf("exactly one concurrent accept should win, got %d winners", winners)
	}
}

func TestFactorStore_ConsumeBackupCode_SingleUse(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	ctx := context.Background()
	bf := &models.Factor{UserID: user.ID, Type: models.FactorTypeBackupCodes, CreatedAt: time.Now().UTC()}
	now := time.Now().UTC()
	bf.EnabledAt = &now
	if err := s.CreateFactor(ctx, bf); err != nil {
		t.Fatalf("create backup factor: %v", err)
	}
	rawCode := "hk7m93px2fnr"
	codes := []models.FactorBackupCode{
		{FactorID: bf.ID, CodeHash: hashCode(rawCode), CreatedAt: time.Now().UTC()},
	}
	if err := s.InsertBackupCodes(ctx, codes); err != nil {
		t.Fatalf("insert codes: %v", err)
	}

	// First use: succeeds.
	ok, err := s.ConsumeBackupCode(ctx, bf.ID, hashCode(rawCode))
	if err != nil {
		t.Fatalf("consume 1: %v", err)
	}
	if !ok {
		t.Fatal("first consume should succeed")
	}
	// Second use of the same code: rejected.
	ok, err = s.ConsumeBackupCode(ctx, bf.ID, hashCode(rawCode))
	if err != nil {
		t.Fatalf("consume 2: %v", err)
	}
	if ok {
		t.Fatal("second consume of same code must be rejected")
	}
}

func TestFactorStore_ConsumeBackupCode_ConcurrentRacesOneWinner(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	ctx := context.Background()
	bf := &models.Factor{UserID: user.ID, Type: models.FactorTypeBackupCodes, CreatedAt: time.Now().UTC()}
	now := time.Now().UTC()
	bf.EnabledAt = &now
	if err := s.CreateFactor(ctx, bf); err != nil {
		t.Fatalf("create backup factor: %v", err)
	}
	rawCode := "race-code-xyz1"
	if err := s.InsertBackupCodes(ctx, []models.FactorBackupCode{
		{FactorID: bf.ID, CodeHash: hashCode(rawCode), CreatedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var wg sync.WaitGroup
	results := make([]bool, 4)
	errs := make([]error, 4)
	for i := range 4 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = s.ConsumeBackupCode(ctx, bf.ID, hashCode(rawCode))
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}
	winners := 0
	for _, ok := range results {
		if ok {
			winners++
		}
	}
	if winners != 1 {
		t.Errorf("exactly one concurrent consume should win, got %d", winners)
	}
}

func TestFactorStore_FindBackupCodesFactor_SingletonConstraint(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	ctx := context.Background()

	// First insert succeeds.
	bf1 := &models.Factor{UserID: user.ID, Type: models.FactorTypeBackupCodes, CreatedAt: time.Now().UTC()}
	if err := s.CreateFactor(ctx, bf1); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Second insert violates the partial unique index idx_factors_user_singleton_backup.
	bf2 := &models.Factor{UserID: user.ID, Type: models.FactorTypeBackupCodes, CreatedAt: time.Now().UTC()}
	err := s.CreateFactor(ctx, bf2)
	if err == nil {
		t.Fatal("expected unique-violation on second backup_codes factor, got nil")
	}
	// Don't assert on exact error string — it's Postgres-specific —
	// just assert that it fails.
}

func TestFactorStore_ReplaceBackupCodes_Atomic(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	ctx := context.Background()

	bf := &models.Factor{UserID: user.ID, Type: models.FactorTypeBackupCodes, CreatedAt: time.Now().UTC()}
	if err := s.CreateFactor(ctx, bf); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Seed 3 old codes.
	if err := s.InsertBackupCodes(ctx, []models.FactorBackupCode{
		{FactorID: bf.ID, CodeHash: hashCode("old-1"), CreatedAt: time.Now().UTC()},
		{FactorID: bf.ID, CodeHash: hashCode("old-2"), CreatedAt: time.Now().UTC()},
		{FactorID: bf.ID, CodeHash: hashCode("old-3"), CreatedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Replace with a fresh set of 2.
	fresh := []models.FactorBackupCode{
		{FactorID: bf.ID, CodeHash: hashCode("new-1"), CreatedAt: time.Now().UTC()},
		{FactorID: bf.ID, CodeHash: hashCode("new-2"), CreatedAt: time.Now().UTC()},
	}
	if err := s.ReplaceBackupCodes(ctx, bf.ID, fresh); err != nil {
		t.Fatalf("replace: %v", err)
	}

	rows, err := s.ListBackupCodes(ctx, bf.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows after replace, got %d", len(rows))
	}
	// None of the old hashes should remain.
	for _, r := range rows {
		if r.CodeHash == hashCode("old-1") || r.CodeHash == hashCode("old-2") || r.CodeHash == hashCode("old-3") {
			t.Error("old code hash survived replace")
		}
	}
}

func TestFactorStore_CleanupAbandonedPending(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")

	ctx := context.Background()
	// Three factors: one old pending, one fresh pending, one active.
	oldPending := &models.Factor{
		UserID:    user.ID,
		Type:      models.FactorTypeTOTP,
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	freshPending := &models.Factor{
		UserID:    user.ID,
		Type:      models.FactorTypeTOTP,
		CreatedAt: time.Now().UTC(),
	}
	activeNow := time.Now().UTC()
	oldActivated := &models.Factor{
		UserID:    user.ID,
		Type:      models.FactorTypeTOTP,
		EnabledAt: &activeNow,
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	for _, f := range []*models.Factor{oldPending, freshPending, oldActivated} {
		if err := s.CreateFactor(ctx, f); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	n, err := s.CleanupAbandonedPending(ctx, time.Hour)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if n != 1 {
		t.Errorf("cleanup removed %d rows, want 1 (only oldPending)", n)
	}

	// oldPending is gone; the other two survive.
	if _, err := s.FindByIDAndUser(ctx, oldPending.ID, user.ID); err != ErrNotFound {
		t.Error("oldPending should have been cleaned up")
	}
	if _, err := s.FindByIDAndUser(ctx, freshPending.ID, user.ID); err != nil {
		t.Error("freshPending should survive")
	}
	if _, err := s.FindByIDAndUser(ctx, oldActivated.ID, user.ID); err != nil {
		t.Error("oldActivated should survive")
	}
}

func TestFactorStore_DeleteFactor_CrossUser_IsNoOp(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	alice := createTestUser(t, db, "Alice", "a@example.com")
	bob := createTestUser(t, db, "Bob", "b@example.com")
	fid := seedTOTPFactor(t, s, alice.ID, true)

	// Bob cannot delete Alice's factor.
	rows, err := s.DeleteFactor(context.Background(), fid, bob.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if rows != 0 {
		t.Errorf("cross-user delete affected %d rows; want 0", rows)
	}
	// Alice's factor still there.
	if _, err := s.FindByIDAndUser(context.Background(), fid, alice.ID); err != nil {
		t.Errorf("alice's factor wrongly deleted: %v", err)
	}
}
