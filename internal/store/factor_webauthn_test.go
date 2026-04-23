package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// seedWebAuthnFactor creates a parent Factor row of type webauthn
// plus a matching subtable credential. Returns the factor id. The
// tests below address both rows via the same id.
func seedWebAuthnFactor(t *testing.T, s *FactorStore, userID uint, credentialID, publicKey []byte, activated bool) uint {
	t.Helper()
	ctx := context.Background()
	f := &models.Factor{
		UserID:    userID,
		Type:      models.FactorTypeWebAuthn,
		CreatedAt: time.Now().UTC(),
	}
	if activated {
		now := time.Now().UTC()
		f.EnabledAt = &now
	}
	if err := s.CreateFactor(ctx, f); err != nil {
		t.Fatalf("create factor: %v", err)
	}
	cred := &models.FactorWebAuthnCredential{
		FactorID:     f.ID,
		CredentialID: credentialID,
		PublicKey:    publicKey,
		SignCount:    0,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.CreateWebAuthnCredential(ctx, cred); err != nil {
		t.Fatalf("create webauthn credential: %v", err)
	}
	return f.ID
}

func TestFactorStore_WebAuthn_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	credID := []byte("cred-id-1")
	pubKey := []byte("cose-public-key")
	fid := seedWebAuthnFactor(t, s, user.ID, credID, pubKey, true)

	// Lookup by factor id.
	got, err := s.FindWebAuthnCredential(context.Background(), fid)
	if err != nil {
		t.Fatalf("find by factor id: %v", err)
	}
	if string(got.CredentialID) != string(credID) {
		t.Errorf("credential_id mismatch")
	}
	if string(got.PublicKey) != string(pubKey) {
		t.Errorf("public_key mismatch")
	}

	// Lookup by credential id.
	got2, err := s.FindWebAuthnCredentialByID(context.Background(), credID)
	if err != nil {
		t.Fatalf("find by credential id: %v", err)
	}
	if got2.FactorID != fid {
		t.Errorf("wrong factor id resolved: %d want %d", got2.FactorID, fid)
	}
}

func TestFactorStore_WebAuthn_FindMissing(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	_, err := s.FindWebAuthnCredential(context.Background(), 99999)
	if err != ErrNotFound {
		t.Errorf("missing factor: got %v, want ErrNotFound", err)
	}
	_, err = s.FindWebAuthnCredentialByID(context.Background(), []byte("no-such-credential"))
	if err != ErrNotFound {
		t.Errorf("missing credential id: got %v, want ErrNotFound", err)
	}
}

// Duplicate credential_id rejected by the unique index. Two users
// cannot bind the same hardware key — WebAuthn L3 §7.1 step 22.
func TestFactorStore_WebAuthn_DuplicateCredentialIDRejected(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	alice := createTestUser(t, db, "Alice", "alice@example.com")
	bob := createTestUser(t, db, "Bob", "bob@example.com")

	credID := []byte("shared-credential-id")
	pubKey := []byte("key")
	_ = seedWebAuthnFactor(t, s, alice.ID, credID, pubKey, true)

	// Bob tries to claim the same credential_id.
	ctx := context.Background()
	f2 := &models.Factor{UserID: bob.ID, Type: models.FactorTypeWebAuthn, CreatedAt: time.Now().UTC()}
	if err := s.CreateFactor(ctx, f2); err != nil {
		t.Fatalf("create bob's factor: %v", err)
	}
	err := s.CreateWebAuthnCredential(ctx, &models.FactorWebAuthnCredential{
		FactorID:     f2.ID,
		CredentialID: credID, // same bytes as Alice's — collision
		PublicKey:    pubKey,
		CreatedAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected duplicate credential_id to be rejected")
	}
}

func TestFactorStore_WebAuthn_UpdateSignCount(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	fid := seedWebAuthnFactor(t, s, user.ID, []byte("c"), []byte("k"), true)

	ctx := context.Background()
	if err := s.UpdateWebAuthnSignCount(ctx, fid, 5, true); err != nil {
		t.Fatalf("bump: %v", err)
	}
	got, _ := s.FindWebAuthnCredential(ctx, fid)
	if got.SignCount != 5 {
		t.Errorf("sign_count = %d, want 5", got.SignCount)
	}
	if !got.BackupState {
		t.Error("backup_state should be true")
	}

	// Subsequent update overwrites (soft-failure / reset to zero for
	// synced-passkey authenticators is also represented here).
	if err := s.UpdateWebAuthnSignCount(ctx, fid, 0, false); err != nil {
		t.Fatalf("reset: %v", err)
	}
	got, _ = s.FindWebAuthnCredential(ctx, fid)
	if got.SignCount != 0 || got.BackupState {
		t.Errorf("expected sign_count=0 backup=false, got %d %v", got.SignCount, got.BackupState)
	}
}

// Two goroutines bumping sign count concurrently — the last writer
// wins, but both succeed without deadlock. (Unlike AcceptTOTPStep,
// WebAuthn sign_count updates are idempotent full-row writes; there
// is no compare-and-set protection at this layer because the spec
// allows non-strict counter semantics.)
func TestFactorStore_WebAuthn_UpdateSignCount_Concurrent(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	fid := seedWebAuthnFactor(t, s, user.ID, []byte("c"), []byte("k"), true)

	ctx := context.Background()
	const n = 5
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = s.UpdateWebAuthnSignCount(ctx, fid, int64(idx+1), true)
		}(i)
	}
	wg.Wait()
	for i, e := range errs {
		if e != nil {
			t.Errorf("goroutine %d: %v", i, e)
		}
	}
	// Final sign_count is one of 1..n (last writer wins), but the row
	// is not corrupted.
	got, err := s.FindWebAuthnCredential(ctx, fid)
	if err != nil {
		t.Fatalf("read final: %v", err)
	}
	if got.SignCount < 1 || got.SignCount > n {
		t.Errorf("sign_count = %d, want in [1,%d]", got.SignCount, n)
	}
}

func TestFactorStore_WebAuthn_RegistrationChallenge(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	ctx := context.Background()

	// Pending factor (no subtable row yet).
	f := &models.Factor{UserID: user.ID, Type: models.FactorTypeWebAuthn, CreatedAt: time.Now().UTC()}
	if err := s.CreateFactor(ctx, f); err != nil {
		t.Fatalf("create: %v", err)
	}

	challenge := []byte("32-random-bytes-of-challenge-nice")
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if err := s.SetRegistrationChallenge(ctx, f.ID, challenge, expiresAt); err != nil {
		t.Fatalf("set: %v", err)
	}
	var row models.Factor
	if err := db.Where("id = ?", f.ID).First(&row).Error; err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(row.RegistrationChallenge) != string(challenge) {
		t.Error("challenge not persisted")
	}
	// Use Unix-second equality to paper over Postgres returning the
	// stored timestamptz in the server's local timezone. The instant
	// is the same; only the formatter differs.
	if row.RegistrationChallengeExpiresAt == nil || row.RegistrationChallengeExpiresAt.Unix() != expiresAt.Unix() {
		t.Errorf("expires_at = %v, want %v", row.RegistrationChallengeExpiresAt, expiresAt)
	}

	// Clear.
	if err := s.ClearRegistrationChallenge(ctx, f.ID); err != nil {
		t.Fatalf("clear: %v", err)
	}
	// Fresh struct so the fields from the previous First don't ghost
	// into the re-read.
	var cleared models.Factor
	if err := db.Where("id = ?", f.ID).First(&cleared).Error; err != nil {
		t.Fatalf("read post-clear: %v", err)
	}
	if cleared.RegistrationChallenge != nil {
		t.Errorf("challenge not cleared: %v", cleared.RegistrationChallenge)
	}
	if cleared.RegistrationChallengeExpiresAt != nil {
		t.Errorf("expires_at not cleared: %v", cleared.RegistrationChallengeExpiresAt)
	}
	// Idempotent: clearing again is a no-op.
	if err := s.ClearRegistrationChallenge(ctx, f.ID); err != nil {
		t.Fatalf("second clear: %v", err)
	}
}

func TestFactorStore_WebAuthn_CascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	s := NewFactorStore(db)
	user := createTestUser(t, db, "U", "u@example.com")
	fid := seedWebAuthnFactor(t, s, user.ID, []byte("c"), []byte("k"), true)

	// Delete the parent — subtable row should go with it.
	if _, err := s.DeleteFactor(context.Background(), fid, user.ID); err != nil {
		t.Fatalf("delete factor: %v", err)
	}
	_, err := s.FindWebAuthnCredential(context.Background(), fid)
	if err != ErrNotFound {
		t.Errorf("cascade delete didn't remove credential: got %v", err)
	}
}
