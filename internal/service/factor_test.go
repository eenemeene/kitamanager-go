package service

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	cryptopkg "github.com/eenemeene/kitamanager-go/internal/crypto"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// testFactorAEAD builds a deterministic AEAD for tests using a fixed
// 32-byte key. Tests don't care what the key is, only that the same
// AEAD instance seals and opens.
func testFactorAEAD(t *testing.T) *cryptopkg.AEAD {
	t.Helper()
	key, _ := hex.DecodeString("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	a, err := cryptopkg.NewAEAD(key)
	if err != nil {
		t.Fatalf("aead: %v", err)
	}
	return a
}

func newFactorService(t *testing.T, db *gorm.DB) (*FactorService, *AuditService) {
	t.Helper()
	audit := createAuditService(db)
	svc := NewFactorService(
		store.NewFactorStore(db),
		store.NewUserStore(db),
		testFactorAEAD(t),
		"KitaManager (test)",
		audit,
	)
	return svc, audit
}

// createUserWithPassword creates a user and bcrypts the given plaintext
// into their password column. Returns the user.
func createUserWithPassword(t *testing.T, db *gorm.DB, name, email, password string) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	u := &models.User{
		Name:     name,
		Email:    strings.ToLower(strings.TrimSpace(email)),
		Password: string(hash),
		Active:   true,
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// enrolAndActivateTOTP is the common setup. Returns factor id + the
// base32 secret extracted from the enrollment URI so tests can
// generate valid TOTP codes.
func enrolAndActivateTOTP(t *testing.T, svc *FactorService, user *models.User, password string) (uint, string) {
	t.Helper()
	ctx := context.Background()
	enroll, err := svc.EnrollTOTP(ctx, user.ID, nil, password, user.Email)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	payload, ok := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	if !ok {
		t.Fatalf("enrollment payload wrong type: %T", enroll.Enrollment)
	}
	// Build a valid code for now().
	code, err := totp.GenerateCode(payload.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("gen code: %v", err)
	}
	if _, err := svc.ActivateFactor(ctx, user.ID, enroll.ID, code); err != nil {
		t.Fatalf("activate: %v", err)
	}
	return enroll.ID, payload.Secret
}

func TestFactorService_EnrollTOTP_RequiresPassword(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "correct-password")

	// Wrong password → Unauthorized.
	_, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "wrong-password", user.Email)
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for wrong password, got %v", err)
	}

	// Empty password → BadRequest.
	_, err = svc.EnrollTOTP(context.Background(), user.ID, nil, "", user.Email)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest for empty password, got %v", err)
	}
}

func TestFactorService_EnrollTOTP_ReplacesPendingRow(t *testing.T) {
	// Two enrollments in a row: the first pending factor must be
	// discarded in favour of the second, so the user never accumulates
	// abandoned rows they can see.
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	first, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	if err != nil {
		t.Fatalf("enrol 1: %v", err)
	}
	second, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	if err != nil {
		t.Fatalf("enrol 2: %v", err)
	}
	if first.ID == second.ID {
		t.Error("expected a new factor id on second enrolment")
	}

	// The first should be gone.
	_, err = store.NewFactorStore(db).FindByIDAndUser(context.Background(), first.ID, user.ID)
	if err != store.ErrNotFound {
		t.Errorf("expected first pending factor gone, got %v", err)
	}
}

func TestFactorService_ActivateFactor_WrongCode(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	_, err = svc.ActivateFactor(context.Background(), user.ID, enroll.ID, "000000")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for wrong code, got %v", err)
	}
}

func TestFactorService_ActivateFactor_AutoCreatesBackupCodesOnFirstPrimary(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	payload := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(payload.Secret, time.Now().UTC())

	resp, err := svc.ActivateFactor(context.Background(), user.ID, enroll.ID, code)
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if !resp.Activated {
		t.Error("expected Activated=true")
	}
	if resp.BackupCodes == nil {
		t.Fatal("expected backup codes payload on first primary activation")
	}
	if len(resp.BackupCodes.Codes) != BackupCodeCount {
		t.Errorf("backup codes count = %d, want %d", len(resp.BackupCodes.Codes), BackupCodeCount)
	}
	// No duplicates.
	seen := make(map[string]bool, len(resp.BackupCodes.Codes))
	for _, c := range resp.BackupCodes.Codes {
		if seen[c] {
			t.Errorf("duplicate backup code: %q", c)
		}
		seen[c] = true
	}
}

func TestFactorService_ActivateFactor_SecondPrimaryDoesNotRegenerateBackupCodes(t *testing.T) {
	// Enrolling a second TOTP must NOT rotate backup codes. The user's
	// existing set stays usable.
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	_, _ = enrolAndActivateTOTP(t, svc, user, "pw")

	// Second TOTP enrolment.
	enroll2, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	if err != nil {
		t.Fatalf("enrol 2: %v", err)
	}
	pl2 := enroll2.Enrollment.(models.TOTPEnrollmentPayload)
	code2, _ := totp.GenerateCode(pl2.Secret, time.Now().UTC())
	resp, err := svc.ActivateFactor(context.Background(), user.ID, enroll2.ID, code2)
	if err != nil {
		t.Fatalf("activate 2: %v", err)
	}
	if resp.BackupCodes != nil {
		t.Error("activating a second primary factor must NOT return fresh backup codes")
	}
}

func TestFactorService_ActivateFactor_DoubleActivateIsConflict(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	pl := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())

	if _, err := svc.ActivateFactor(context.Background(), user.ID, enroll.ID, code); err != nil {
		t.Fatalf("first activate: %v", err)
	}

	// Second activation of an already-active factor: use a fresh code
	// (same window might have advanced between calls).
	code2, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())
	_, err = svc.ActivateFactor(context.Background(), user.ID, enroll.ID, code2)
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict on double-activate, got %v", err)
	}
}

func TestFactorService_ActivateFactor_CrossUser_NotFound(t *testing.T) {
	// Critical: Bob must not be able to activate Alice's pending factor.
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	alice := createUserWithPassword(t, db, "Alice", "alice@example.com", "pw-a")
	bob := createUserWithPassword(t, db, "Bob", "bob@example.com", "pw-b")

	enroll, err := svc.EnrollTOTP(context.Background(), alice.ID, nil, "pw-a", alice.Email)
	if err != nil {
		t.Fatalf("alice enrol: %v", err)
	}
	pl := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())

	_, err = svc.ActivateFactor(context.Background(), bob.ID, enroll.ID, code)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-user activate, got %v", err)
	}
}

func TestFactorService_ListForUser_OnlyActivated(t *testing.T) {
	// Pending (unactivated) factors must not appear in the list
	// response. Users see only what's actually protecting them.
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	// Enrol but never activate.
	if _, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email); err != nil {
		t.Fatalf("enrol: %v", err)
	}
	list, err := svc.ListForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("pending factor leaked into list: %+v", list)
	}
}

func TestFactorService_ListForUser_IncludesBackupCodesRemaining(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	_, _ = enrolAndActivateTOTP(t, svc, user, "pw")

	list, err := svc.ListForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var totp, backup *models.FactorResponse
	for i := range list {
		switch list[i].Type {
		case models.FactorTypeTOTP:
			totp = &list[i]
		case models.FactorTypeBackupCodes:
			backup = &list[i]
		}
	}
	if totp == nil {
		t.Fatal("totp factor missing from list")
	}
	if backup == nil {
		t.Fatal("backup_codes factor missing from list")
	}
	if backup.BackupCodesRemaining == nil || *backup.BackupCodesRemaining != BackupCodeCount {
		t.Errorf("BackupCodesRemaining = %v, want %d", backup.BackupCodesRemaining, BackupCodeCount)
	}
	if totp.BackupCodesRemaining != nil {
		t.Error("non-backup factor must not carry BackupCodesRemaining")
	}
}

func TestFactorService_GetForUser_CrossUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	alice := createUserWithPassword(t, db, "Alice", "alice@example.com", "pw-a")
	bob := createUserWithPassword(t, db, "Bob", "bob@example.com", "pw-b")

	fid, _ := enrolAndActivateTOTP(t, svc, alice, "pw-a")

	_, err := svc.GetForUser(context.Background(), bob.ID, fid)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-user get, got %v", err)
	}
}

func TestFactorService_DeleteFactor_LastPrimaryRequiresCode(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	fid, _ := enrolAndActivateTOTP(t, svc, user, "pw")

	// Without a code, deletion is rejected (this would leave the user
	// with password-only, so extra proof is required).
	err := svc.DeleteFactor(context.Background(), user.ID, fid, "pw", "")
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest without code, got %v", err)
	}
}

func TestFactorService_DeleteFactor_SweepsBackupCodesOnLastPrimary(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	fid, secret := enrolAndActivateTOTP(t, svc, user, "pw")
	// Confirm backup_codes factor exists before delete.
	if _, err := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID); err != nil {
		t.Fatalf("precondition: backup_codes factor should exist: %v", err)
	}

	// Wait past the current TOTP step so the code we generate now
	// won't be rejected by replay-prevention (the test harness uses
	// real time).
	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	if err := svc.DeleteFactor(context.Background(), user.ID, fid, "pw", code); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Backup codes factor gone too.
	_, err := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Error("backup_codes factor must be swept when last primary is deleted")
	}
}

func TestFactorService_DeleteFactor_NonPrimary_NoCode_OK(t *testing.T) {
	// User has two primaries. Deleting one of them (not the last) is
	// allowed without a code — still requires password.
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	_, _ = enrolAndActivateTOTP(t, svc, user, "pw")
	fid2, _ := enrolAndActivateTOTP(t, svc, user, "pw")

	if err := svc.DeleteFactor(context.Background(), user.ID, fid2, "pw", ""); err != nil {
		t.Errorf("expected delete to succeed without code (two primaries), got %v", err)
	}
}

func TestFactorService_DeleteFactor_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	fid, _ := enrolAndActivateTOTP(t, svc, user, "pw")

	err := svc.DeleteFactor(context.Background(), user.ID, fid, "wrong-password", "000000")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for wrong password, got %v", err)
	}
}

func TestFactorService_DeleteFactor_CrossUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	alice := createUserWithPassword(t, db, "Alice", "alice@example.com", "pw-a")
	bob := createUserWithPassword(t, db, "Bob", "bob@example.com", "pw-b")

	fid, _ := enrolAndActivateTOTP(t, svc, alice, "pw-a")

	// Bob provides his own correct password but targets Alice's factor.
	err := svc.DeleteFactor(context.Background(), bob.ID, fid, "pw-b", "anything")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFactorService_UpdateLabel(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	fid, _ := enrolAndActivateTOTP(t, svc, user, "pw")

	label := "Old phone"
	resp, err := svc.UpdateLabel(context.Background(), user.ID, fid, &label)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if resp.Label == nil || *resp.Label != label {
		t.Errorf("label = %v, want %q", resp.Label, label)
	}

	// Empty string normalises to nil (clear the label).
	empty := ""
	resp, err = svc.UpdateLabel(context.Background(), user.ID, fid, &empty)
	if err != nil {
		t.Fatalf("update clear: %v", err)
	}
	if resp.Label != nil {
		t.Errorf("expected label cleared, got %v", *resp.Label)
	}
}

func TestFactorService_RegenerateBackupCodes_InvalidatesOld(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	_, _ = enrolAndActivateTOTP(t, svc, user, "pw")
	bf, err := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("find bf: %v", err)
	}

	payload, err := svc.RegenerateBackupCodes(context.Background(), user.ID, bf.ID, "pw")
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if len(payload.Codes) != BackupCodeCount {
		t.Errorf("codes count = %d", len(payload.Codes))
	}

	// Counting unused codes should match BackupCodeCount exactly (old
	// codes, used or not, have all been deleted).
	n, err := store.NewFactorStore(db).CountUnusedBackupCodes(context.Background(), bf.ID)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != BackupCodeCount {
		t.Errorf("unused count after regenerate = %d, want %d", n, BackupCodeCount)
	}
}

func TestFactorService_RegenerateBackupCodes_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	_, _ = enrolAndActivateTOTP(t, svc, user, "pw")
	bf, _ := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)

	_, err := svc.RegenerateBackupCodes(context.Background(), user.ID, bf.ID, "wrong")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestFactorService_RegenerateBackupCodes_CrossUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	alice := createUserWithPassword(t, db, "Alice", "alice@example.com", "pw-a")
	bob := createUserWithPassword(t, db, "Bob", "bob@example.com", "pw-b")

	_, _ = enrolAndActivateTOTP(t, svc, alice, "pw-a")
	bf, _ := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), alice.ID)

	_, err := svc.RegenerateBackupCodes(context.Background(), bob.ID, bf.ID, "pw-b")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-user regenerate, got %v", err)
	}
}

func TestFactorService_BackupCode_CaseAndHyphenInsensitive(t *testing.T) {
	// The verifier accepts hyphenated/uppercased input because that's
	// how we formatted the code for the user. We need the comparison
	// to normalise on input.
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	_, _ = enrolAndActivateTOTP(t, svc, user, "pw")
	bf, _ := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)

	payload, err := svc.RegenerateBackupCodes(context.Background(), user.ID, bf.ID, "pw")
	if err != nil {
		t.Fatalf("regen: %v", err)
	}
	// The user might paste the code with extra spaces / upper-casing.
	mangled := strings.ToUpper(strings.ReplaceAll(payload.Codes[0], "-", " "))
	if !svc.tryBackupCode(context.Background(), bf.ID, mangled) {
		t.Errorf("normalised backup code should verify: raw=%q mangled=%q", payload.Codes[0], mangled)
	}
}

func TestFactorService_CleanupAbandonedPending(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	// Enrol two pending factors; age one by rewriting created_at.
	f1, _ := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	// Second enroll auto-deletes the first (same-type pending replace),
	// so only f2 remains pending. Age it.
	f2, _ := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	_ = f1 // f1 is gone
	twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour)
	if err := db.Model(&models.Factor{}).Where("id = ?", f2.ID).Update("created_at", twoHoursAgo).Error; err != nil {
		t.Fatalf("age: %v", err)
	}

	if err := svc.CleanupAbandonedPendingFactors(context.Background(), time.Hour); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	_, err := store.NewFactorStore(db).FindByIDAndUser(context.Background(), f2.ID, user.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Error("aged pending factor should have been GC'd")
	}
}

// Sanity: enrollment via our service produces URIs that match what
// pquerna/otp.Validate accepts end-to-end. This guards against a
// future change to totp-generation options not also updating the
// validate path.
func TestFactorService_EnrollTOTP_GeneratedURIValidates(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	resp, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", user.Email)
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	payload := resp.Enrollment.(models.TOTPEnrollmentPayload)

	now := time.Now().UTC()
	code, err := totp.GenerateCode(payload.Secret, now)
	if err != nil {
		t.Fatalf("gen code: %v", err)
	}
	ok, _ := totp.ValidateCustom(code, payload.Secret, now, totp.ValidateOpts{
		Period: 30, Skew: 1, Digits: otp.DigitsSix,
	})
	if !ok {
		t.Error("generated code must validate against enrolment secret")
	}
}
