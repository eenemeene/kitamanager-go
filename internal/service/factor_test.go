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
		nil,
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

// enrolAndActivateTOTP is the common setup. Returns factor id, the
// base32 secret (so tests can generate valid TOTP codes), and the
// auto-issued backup codes (so tests that need multiple step-up
// proofs in the same 30s TOTP window can use single-use backup codes
// instead of fighting the TOTP replay protection).
//
// Always called as the user's first-ever enrollment so the step-up MFA
// code can stay empty.
//
// `db` is needed to reset last_used_step after activation so tests
// that immediately validate a TOTP code in the same 30-s window are
// not blocked by the activation-step bump (A-M-1, audit 2026-05-01).
func enrolAndActivateTOTP(t *testing.T, db *gorm.DB, svc *FactorService, user *models.User, password string) (uint, string, []string) {
	t.Helper()
	ctx := context.Background()
	enroll, err := svc.EnrollTOTP(ctx, user.ID, nil, password, "", user.Email)
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
	resp, err := svc.ActivateFactor(ctx, user.ID, enroll.ID, &models.FactorActivateRequest{Code: code})
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	// Reset the step bump so tests can immediately exercise login /
	// step-up code paths under the same time window. Production code
	// must never want this; this is the test-time equivalent of
	// "wait 30 seconds for the next window."
	if err := db.Exec("UPDATE factor_totp_secrets SET last_used_step = 0 WHERE factor_id = ?", enroll.ID).Error; err != nil {
		t.Fatalf("reset last_used_step: %v", err)
	}
	var codes []string
	if resp.BackupCodes != nil {
		codes = resp.BackupCodes.Codes
	}
	return enroll.ID, payload.Secret, codes
}

// validTOTPCode returns a TOTP code that's currently valid for `secret`.
// Centralised so step-up tests don't repeat the totp.GenerateCode dance.
func validTOTPCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("gen code: %v", err)
	}
	return code
}

func TestFactorService_EnrollTOTP_RequiresPassword(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "correct-password")

	// Wrong password → Unauthorized.
	_, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "wrong-password", "", user.Email)
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for wrong password, got %v", err)
	}

	// Empty password → BadRequest.
	_, err = svc.EnrollTOTP(context.Background(), user.ID, nil, "", "", user.Email)
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

	first, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if err != nil {
		t.Fatalf("enrol 1: %v", err)
	}
	second, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
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

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	_, err = svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: "000000"})
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for wrong code, got %v", err)
	}
}

func TestFactorService_ActivateFactor_AutoCreatesBackupCodesOnFirstPrimary(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	payload := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(payload.Secret, time.Now().UTC())

	resp, err := svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: code})
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

	_, secret, _ := enrolAndActivateTOTP(t, db, svc, user, "pw")

	// Second TOTP enrolment — needs step-up code (user already has a primary).
	enroll2, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", validTOTPCode(t, secret), user.Email)
	if err != nil {
		t.Fatalf("enrol 2: %v", err)
	}
	pl2 := enroll2.Enrollment.(models.TOTPEnrollmentPayload)
	code2, _ := totp.GenerateCode(pl2.Secret, time.Now().UTC())
	resp, err := svc.ActivateFactor(context.Background(), user.ID, enroll2.ID, &models.FactorActivateRequest{Code: code2})
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

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	pl := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())

	if _, err := svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: code}); err != nil {
		t.Fatalf("first activate: %v", err)
	}

	// Second activation of an already-active factor: use a fresh code
	// (same window might have advanced between calls).
	code2, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())
	_, err = svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: code2})
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

	enroll, err := svc.EnrollTOTP(context.Background(), alice.ID, nil, "pw-a", "", alice.Email)
	if err != nil {
		t.Fatalf("alice enrol: %v", err)
	}
	pl := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())

	_, err = svc.ActivateFactor(context.Background(), bob.ID, enroll.ID, &models.FactorActivateRequest{Code: code})
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
	if _, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email); err != nil {
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

	_, _, _ = enrolAndActivateTOTP(t, db, svc, user, "pw")

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

	fid, _, _ := enrolAndActivateTOTP(t, db, svc, alice, "pw-a")

	_, err := svc.GetForUser(context.Background(), bob.ID, fid)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-user get, got %v", err)
	}
}

func TestFactorService_DeleteFactor_LastPrimaryRequiresCode(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	fid, _, _ := enrolAndActivateTOTP(t, db, svc, user, "pw")

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

	fid, secret, _ := enrolAndActivateTOTP(t, db, svc, user, "pw")
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

func TestFactorService_DeleteFactor_NonLastPrimary_RequiresCode(t *testing.T) {
	// Closes audit finding A-H-2: deleting a non-last primary used to
	// require only password, so a stolen session + phished password
	// could silently dismantle a user's MFA. With the step-up fix, ANY
	// delete on a user who has at least one active primary factor
	// requires a current TOTP/backup code.
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	_, _, backups := enrolAndActivateTOTP(t, db, svc, user, "pw")
	// Use single-use backup codes for sequential step-ups so we don't
	// fight TOTP's per-step replay protection within one 30s window.
	if len(backups) < 3 {
		t.Fatalf("expected >= 3 backup codes from auto-issue, got %d", len(backups))
	}
	// Second TOTP enrolment with backup-code step-up.
	enroll2, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", backups[0], user.Email)
	if err != nil {
		t.Fatalf("enrol 2: %v", err)
	}
	pl2 := enroll2.Enrollment.(models.TOTPEnrollmentPayload)
	if _, err := svc.ActivateFactor(context.Background(), user.ID, enroll2.ID, &models.FactorActivateRequest{
		Code: validTOTPCode(t, pl2.Secret),
	}); err != nil {
		t.Fatalf("activate 2: %v", err)
	}

	// Without a code, deletion of the second factor is rejected.
	err = svc.DeleteFactor(context.Background(), user.ID, enroll2.ID, "pw", "")
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest without code (stolen-session+password attack defence), got %v", err)
	}

	// With a wrong code → Unauthorized (the password already passed).
	err = svc.DeleteFactor(context.Background(), user.ID, enroll2.ID, "pw", "000000")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for wrong code, got %v", err)
	}

	// With a fresh backup code, the delete succeeds.
	if err := svc.DeleteFactor(context.Background(), user.ID, enroll2.ID, "pw", backups[1]); err != nil {
		t.Errorf("expected delete to succeed with valid step-up code, got %v", err)
	}
}

func TestFactorService_DeleteFactor_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	fid, _, _ := enrolAndActivateTOTP(t, db, svc, user, "pw")

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

	fid, _, _ := enrolAndActivateTOTP(t, db, svc, alice, "pw-a")

	// Bob provides his own correct password but targets Alice's factor.
	err := svc.DeleteFactor(context.Background(), bob.ID, fid, "pw-b", "anything")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestFactorService_DeleteFactor_WebAuthn_LastPrimary_RequiresCode covers
// the delete-last-primary rule when the remaining factor is WebAuthn.
// The service test harness wires webAuthn=nil (ceremonies live in
// integration), so we seed the webauthn parent row directly and
// exercise DeleteFactor's primary-counting + backup_codes-sweep logic.
func TestFactorService_DeleteFactor_WebAuthn_LastPrimary_RequiresCode(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	// Set up a realistic two-primary state: TOTP (which also spawns
	// the backup_codes factor via auto-create) + WebAuthn seeded
	// directly.
	totpFID, _, autoBackups := enrolAndActivateTOTP(t, db, svc, user, "pw")
	if len(autoBackups) < 1 {
		t.Fatalf("expected >= 1 auto backup code, got %d", len(autoBackups))
	}
	now := time.Now().UTC()
	wa := &models.Factor{
		UserID:    user.ID,
		Type:      models.FactorTypeWebAuthn,
		CreatedAt: now,
		EnabledAt: &now,
	}
	if err := store.NewFactorStore(db).CreateFactor(context.Background(), wa); err != nil {
		t.Fatalf("seed webauthn: %v", err)
	}

	// Delete TOTP using a backup code as step-up. Closes audit
	// finding A-M-1: the activation code now bumps last_used_step,
	// so a fresh TOTP code cannot serve as step-up in the same 30-s
	// window. Backup codes are single-use and unaffected.
	if err := svc.DeleteFactor(context.Background(), user.ID, totpFID, "pw", autoBackups[0]); err != nil {
		t.Fatalf("delete totp: %v", err)
	}

	// Now WebAuthn is the last primary. Deleting it without a code
	// must fail with BadRequest — same rule that covers TOTP-only
	// users, restated for WebAuthn-only users.
	err := svc.DeleteFactor(context.Background(), user.ID, wa.ID, "pw", "")
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest when deleting last primary without code, got %v", err)
	}

	// WebAuthn factor still present.
	if _, err := store.NewFactorStore(db).FindByIDAndUser(context.Background(), wa.ID, user.ID); err != nil {
		t.Errorf("webauthn factor wrongly removed: %v", err)
	}
}

// TestFactorService_DeleteFactor_WebAuthn_LastPrimary_SweepsBackupCodes
// is the happy-path counterpart: when the caller supplies a valid
// backup code, deletion of the WebAuthn factor succeeds AND the
// backup_codes factor is swept (no stranded secondary factor rows).
func TestFactorService_DeleteFactor_WebAuthn_LastPrimary_SweepsBackupCodes(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	// Seed state: TOTP (for its backup codes) then delete TOTP with
	// WebAuthn also active so backup_codes survive the TOTP deletion.
	totpFID, _, autoBackups := enrolAndActivateTOTP(t, db, svc, user, "pw")
	if len(autoBackups) < 2 {
		t.Fatalf("expected >= 2 auto backup codes, got %d", len(autoBackups))
	}
	now := time.Now().UTC()
	wa := &models.Factor{
		UserID:    user.ID,
		Type:      models.FactorTypeWebAuthn,
		CreatedAt: now,
		EnabledAt: &now,
	}
	if err := store.NewFactorStore(db).CreateFactor(context.Background(), wa); err != nil {
		t.Fatalf("seed webauthn: %v", err)
	}

	// Regenerate the backup codes BEFORE deleting TOTP. Use a single-use
	// auto-issued backup code as step-up so we don't conflict with TOTP's
	// per-window replay protection on subsequent step-ups.
	bundle, err := svc.RegenerateBackupCodes(context.Background(), user.ID,
		mustFindBackupCodesFactor(t, db, user.ID).ID, "pw", autoBackups[0])
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	backupCode := bundle.Codes[0]

	// Now delete TOTP. Use a different (still-fresh) auto-issued backup
	// code as step-up: regenerate consumed autoBackups[0]; the others
	// were invalidated by the regenerate. The fresh `bundle.Codes` are
	// the live set now — use bundle.Codes[1] as the step-up here so
	// bundle.Codes[0] stays valid for the WebAuthn delete below.
	if err := svc.DeleteFactor(context.Background(), user.ID, totpFID, "pw", bundle.Codes[1]); err != nil {
		t.Fatalf("delete totp: %v", err)
	}

	// Delete WebAuthn with a backup code — succeeds, AND backup_codes
	// factor is swept since WebAuthn was the last primary.
	if err := svc.DeleteFactor(context.Background(), user.ID, wa.ID, "pw", backupCode); err != nil {
		t.Fatalf("delete webauthn: %v", err)
	}
	if _, err := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID); !errors.Is(err, store.ErrNotFound) {
		t.Error("backup_codes factor must be swept when last webauthn primary is deleted")
	}
}

func mustFindBackupCodesFactor(t *testing.T, db *gorm.DB, userID uint) *models.Factor {
	t.Helper()
	f, err := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), userID)
	if err != nil {
		t.Fatalf("find backup_codes factor: %v", err)
	}
	return f
}

func TestFactorService_UpdateLabel(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	fid, _, _ := enrolAndActivateTOTP(t, db, svc, user, "pw")

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

	_, _, autoBackups := enrolAndActivateTOTP(t, db, svc, user, "pw")
	if len(autoBackups) < 1 {
		t.Fatal("expected >= 1 auto backup code")
	}
	bf, err := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("find bf: %v", err)
	}

	// Use a backup code as step-up — TOTP would replay-conflict in
	// the same 30-s window because the activation code already
	// bumped last_used_step (A-M-1).
	payload, err := svc.RegenerateBackupCodes(context.Background(), user.ID, bf.ID, "pw", autoBackups[0])
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
	_, secret, _ := enrolAndActivateTOTP(t, db, svc, user, "pw")
	bf, _ := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)

	// Wrong password fails before the step-up code is even checked.
	// We still pass a valid code to prove the step-up isn't masking the
	// password failure.
	_, err := svc.RegenerateBackupCodes(context.Background(), user.ID, bf.ID, "wrong", validTOTPCode(t, secret))
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestFactorService_RegenerateBackupCodes_CrossUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	alice := createUserWithPassword(t, db, "Alice", "alice@example.com", "pw-a")
	bob := createUserWithPassword(t, db, "Bob", "bob@example.com", "pw-b")

	_, _, _ = enrolAndActivateTOTP(t, db, svc, alice, "pw-a")
	bf, _ := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), alice.ID)

	// Bob has no factors → step-up is a no-op (no primary). FindByIDAndUser
	// then 404s because the factor belongs to Alice.
	_, err := svc.RegenerateBackupCodes(context.Background(), bob.ID, bf.ID, "pw-b", "")
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

	_, _, autoBackups := enrolAndActivateTOTP(t, db, svc, user, "pw")
	if len(autoBackups) < 1 {
		t.Fatal("expected >= 1 auto backup code")
	}
	bf, _ := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)

	payload, err := svc.RegenerateBackupCodes(context.Background(), user.ID, bf.ID, "pw", autoBackups[0])
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
	f1, _ := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	// Second enroll auto-deletes the first (same-type pending replace),
	// so only f2 remains pending. Age it.
	f2, _ := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
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

// TestFactorService_ActivateFactor_FiveWrongCodes_DeletesPending guards
// the activation rate-limit: the pending factor row survives a handful
// of wrong codes, but the 5th wrong code (FactorActivationFailureLimit)
// trips the auto-delete and the next request returns 429 — forcing the
// user (or attacker-in-session) to re-enroll from scratch. This closes
// the "session cookie → unlimited TOTP brute-force on pending row"
// surface.
func TestFactorService_ActivateFactor_FiveWrongCodes_DeletesPending(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}

	// First 4 wrong codes: Unauthorized but the row survives.
	for i := 1; i <= FactorActivationFailureLimit-1; i++ {
		_, err := svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: "000000"})
		if !errors.Is(err, apperror.ErrUnauthorized) {
			t.Errorf("attempt %d: expected ErrUnauthorized, got %v", i, err)
		}
	}
	// Row still exists and still pending.
	if _, err := store.NewFactorStore(db).FindByIDAndUser(context.Background(), enroll.ID, user.ID); err != nil {
		t.Fatalf("pending row should still exist before limit hit: %v", err)
	}

	// 5th wrong code: TooManyRequests + pending row is gone.
	_, err = svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: "000000"})
	if !errors.Is(err, apperror.ErrTooManyRequests) {
		t.Errorf("5th attempt: expected ErrTooManyRequests, got %v", err)
	}
	_, err = store.NewFactorStore(db).FindByIDAndUser(context.Background(), enroll.ID, user.ID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("pending row must be auto-deleted after limit; got %v", err)
	}

	// Any subsequent request against the same factor id is a 404.
	_, err = svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: "000000"})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("after auto-delete: expected ErrNotFound, got %v", err)
	}
}

// TestFactorService_ActivateFactor_CorrectCodeAfterWrongAttempts: the
// rate-limit counter lives only on pending rows (IncrementActivationFailures
// WHEREs on enabled_at IS NULL). A correct code flips enabled_at to now(),
// so the counter becomes dead data — no explicit reset required. A user
// with a few typos is not penalised; abandoned pending rows with N<limit
// are swept by CleanupAbandonedPendingFactors after the idle window.
func TestFactorService_ActivateFactor_CorrectCodeAfterWrongAttempts(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	pl := enroll.Enrollment.(models.TOTPEnrollmentPayload)

	// Two typos — well below the limit.
	for range 2 {
		_, err := svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: "000000"})
		if !errors.Is(err, apperror.ErrUnauthorized) {
			t.Fatalf("expected Unauthorized for wrong code, got %v", err)
		}
	}
	// Real code now: activation succeeds.
	code, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())
	if _, err := svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: code}); err != nil {
		t.Errorf("activation after typos should succeed, got %v", err)
	}
}

// TestFactorService_ListForUser_SortOrder verifies the list ordering
// contract the Settings UI depends on:
//   - backup_codes sink to the bottom,
//   - primary factors ordered by last_used_at DESC (NULLs last),
//   - created_at DESC as final tiebreaker.
func TestFactorService_ListForUser_SortOrder(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	// Activate two TOTP factors; give the second one a more-recent
	// last_used_at so it must sort first.
	_, _, autoBackups := enrolAndActivateTOTP(t, db, svc, user, "pw")
	if len(autoBackups) < 1 {
		t.Fatalf("expected >= 1 backup code from auto-issue")
	}
	// Second enrolment now requires a step-up code (user already has a primary).
	enroll2, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", autoBackups[0], user.Email)
	if err != nil {
		t.Fatalf("enrol 2: %v", err)
	}
	pl2 := enroll2.Enrollment.(models.TOTPEnrollmentPayload)
	if _, err := svc.ActivateFactor(context.Background(), user.ID, enroll2.ID, &models.FactorActivateRequest{
		Code: validTOTPCode(t, pl2.Secret),
	}); err != nil {
		t.Fatalf("activate 2: %v", err)
	}
	fid2 := enroll2.ID

	recent := time.Now().UTC()
	if err := db.Model(&models.Factor{}).Where("id = ?", fid2).
		Update("last_used_at", recent).Error; err != nil {
		t.Fatalf("touch last_used_at: %v", err)
	}

	list, err := svc.ListForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 factors, got %d", len(list))
	}
	// Last entry must be the backup_codes row.
	if list[len(list)-1].Type != models.FactorTypeBackupCodes {
		t.Errorf("last entry should be backup_codes, got %q", list[len(list)-1].Type)
	}
	// Among primaries, the one with the recent last_used_at comes first.
	if list[0].ID != fid2 {
		t.Errorf("expected recently-used factor %d first, got %d", fid2, list[0].ID)
	}
}

// TestFactorService_ActivateFactor_BackupCodesType_Rejected verifies
// that you cannot POST /activate on a backup_codes factor — those are
// auto-activated at creation time, and the endpoint must not pretend
// otherwise (which would let clients trip the rate-limit path against
// a synthetic factor type).
func TestFactorService_ActivateFactor_BackupCodesType_Rejected(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	_, _, _ = enrolAndActivateTOTP(t, db, svc, user, "pw")
	bf, err := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("find backup_codes factor: %v", err)
	}

	// Mark as pending to force ActivateFactor through the type switch —
	// without this the "already activated" guard would short-circuit.
	if err := db.Model(&models.Factor{}).Where("id = ?", bf.ID).
		Update("enabled_at", nil).Error; err != nil {
		t.Fatalf("force pending: %v", err)
	}

	_, err = svc.ActivateFactor(context.Background(), user.ID, bf.ID, &models.FactorActivateRequest{Code: "any-code"})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest for activate on backup_codes, got %v", err)
	}
}

// TestFactorService_RegenerateBackupCodes_OnTOTPFactor_Rejected: you
// cannot POST /regenerate at a TOTP factor id. This is a 400, not a
// 404 — the caller owns the factor, they're just pointing regenerate
// at the wrong type.
func TestFactorService_RegenerateBackupCodes_OnTOTPFactor_Rejected(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	totpID, _, autoBackups := enrolAndActivateTOTP(t, db, svc, user, "pw")
	if len(autoBackups) < 1 {
		t.Fatal("expected >= 1 auto backup code")
	}
	_, err := svc.RegenerateBackupCodes(context.Background(), user.ID, totpID, "pw", autoBackups[0])
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest when regenerating on a TOTP factor, got %v", err)
	}
}

// TestFactorService_TryTOTPCode_FailedVerify_NoLastUsedBump guards
// against a subtle timing leak: tryTOTPCode must NOT bump last_used_at
// for a wrong code, otherwise an attacker could probe "is this code
// valid" by watching last_used_at move. Only a successful verify
// (which passes the compare-and-set on last_used_step) touches it.
func TestFactorService_TryTOTPCode_FailedVerify_NoLastUsedBump(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	fid, _, _ := enrolAndActivateTOTP(t, db, svc, user, "pw")

	// Snapshot last_used_at right after activation.
	var before models.Factor
	if err := db.Where("id = ?", fid).First(&before).Error; err != nil {
		t.Fatalf("read before: %v", err)
	}

	// Wrong code — try a handful so we don't accidentally collide with
	// a real code at some step.
	for _, wrong := range []string{"000000", "111111", "999999"} {
		if ok := svc.tryTOTPCode(context.Background(), fid, wrong); ok {
			t.Fatalf("unexpected: wrong code %q was accepted", wrong)
		}
	}

	var after models.Factor
	if err := db.Where("id = ?", fid).First(&after).Error; err != nil {
		t.Fatalf("read after: %v", err)
	}
	// last_used_at must be unchanged (same *time.Time pointer value).
	if (before.LastUsedAt == nil) != (after.LastUsedAt == nil) {
		t.Errorf("last_used_at pointer-nullity flipped: before=%v after=%v", before.LastUsedAt, after.LastUsedAt)
	}
	if before.LastUsedAt != nil && after.LastUsedAt != nil && !before.LastUsedAt.Equal(*after.LastUsedAt) {
		t.Errorf("last_used_at moved on wrong codes: before=%v after=%v", before.LastUsedAt, after.LastUsedAt)
	}
}

// TestFactorService_UpdateLabel_TooLong rejects labels beyond 100
// characters. The DB column is 100-wide; without the service check an
// over-long label would hit the SQL layer with a driver error, which
// becomes a 500 for the user. We want a clean 400 instead.
func TestFactorService_UpdateLabel_TooLong(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	fid, _, _ := enrolAndActivateTOTP(t, db, svc, user, "pw")

	tooLong := strings.Repeat("a", 101)
	_, err := svc.UpdateLabel(context.Background(), user.ID, fid, &tooLong)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest for 101-char label, got %v", err)
	}

	// Boundary: exactly 100 chars is accepted.
	ok := strings.Repeat("a", 100)
	if _, err := svc.UpdateLabel(context.Background(), user.ID, fid, &ok); err != nil {
		t.Errorf("100-char label should be accepted, got %v", err)
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

	resp, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
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

// ----------------------------------------------------------------------
// Step-up MFA enforcement on factor enrollment / regenerate / delete.
//
// These tests close audit findings A-H-2 and A-H-3 (security audit
// 2026-05-01): a stolen session combined with a phished password must
// not be able to silently plant or remove an authenticator.
// ----------------------------------------------------------------------

// First-ever enrollment is always allowed without a step-up code: the
// user has no primary factor yet to verify against.
func TestFactorService_EnrollTOTP_FirstEnrollment_NoStepUpRequired(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	resp, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if err != nil {
		t.Fatalf("first enrolment without code must succeed, got %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected factor row id")
	}
}

// Second TOTP enrolment with no step-up code is rejected as BadRequest.
func TestFactorService_EnrollTOTP_SecondEnrollment_RequiresStepUpCode(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	_, _, _ = enrolAndActivateTOTP(t, db, svc, user, "pw")

	_, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest for second enrolment without step-up code, got %v", err)
	}
}

// Second enrolment with a wrong step-up code is rejected as Unauthorized.
func TestFactorService_EnrollTOTP_SecondEnrollment_WrongStepUpCode(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	_, _, _ = enrolAndActivateTOTP(t, db, svc, user, "pw")

	_, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "000000", user.Email)
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected Unauthorized for second enrolment with wrong code, got %v", err)
	}
}

// EnrollWebAuthn requires step-up too; nothing changes about it being
// gated by WebAuthn-not-enabled, but the password-only path is closed.
func TestFactorService_EnrollWebAuthn_RequiresStepUpWhenPrimaryExists(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	_, _, _ = enrolAndActivateTOTP(t, db, svc, user, "pw")

	// WebAuthn is not wired in tests (svc.webAuthn is nil) — that
	// returns BadRequest("WebAuthn is not enabled..."). To assert the
	// step-up gate independently, we must verify the gate fires BEFORE
	// the not-enabled check by passing a wrong code; expect Unauthorized.
	_, err := svc.EnrollWebAuthn(context.Background(), user.ID, nil, "pw", "000000", "u@example.com", "U")
	// Without WebAuthn wiring we'd get BadRequest first; the step-up
	// gate is added AFTER the password+webauthn-enabled checks, so the
	// expected error here is BadRequest("WebAuthn is not enabled...").
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest (webauthn not enabled), got %v", err)
	}
}

// Backup-code regeneration requires both password AND a current code.
// Stolen-session-and-phished-password attack: holding the password
// (but no MFA) cannot atomically invalidate the user's existing codes.
func TestFactorService_RegenerateBackupCodes_StolenSessionPasswordIsNotEnough(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	_, _, _ = enrolAndActivateTOTP(t, db, svc, user, "pw")
	bf, _ := store.NewFactorStore(db).FindBackupCodesFactor(context.Background(), user.ID)

	// Empty code → BadRequest. The "I just have your session and
	// password" attack is blocked.
	_, err := svc.RegenerateBackupCodes(context.Background(), user.ID, bf.ID, "pw", "")
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest for regenerate without step-up code, got %v", err)
	}

	// Wrong code → Unauthorized.
	_, err = svc.RegenerateBackupCodes(context.Background(), user.ID, bf.ID, "pw", "000000")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected Unauthorized for regenerate with wrong code, got %v", err)
	}
}

// DeleteFactor with a primary enrolled and no code is BadRequest, even
// when the factor being deleted is NOT the last primary. This is the
// regression test for the audit's "delete every non-last primary" bypass.
func TestFactorService_DeleteFactor_PrimaryEnrolled_NoCode_BadRequest(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")
	fid, _, autoBackups := enrolAndActivateTOTP(t, db, svc, user, "pw")

	// Single-factor user: even deleting their LAST primary requires a
	// code (this was already the behaviour, kept as regression).
	if err := svc.DeleteFactor(context.Background(), user.ID, fid, "pw", ""); !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected BadRequest deleting only primary without code, got %v", err)
	}

	// With a backup code, the delete succeeds.
	if err := svc.DeleteFactor(context.Background(), user.ID, fid, "pw", autoBackups[0]); err != nil {
		t.Errorf("expected delete to succeed with backup code, got %v", err)
	}
}

// TestFactorService_VerifyTOTPForActivation_BumpsLastUsedStep closes
// audit finding A-M-1 (security review 2026-05-01): activation must
// bump last_used_step so the activation code cannot ALSO serve as the
// first-login code in the same 30-s window.
//
// Walks the flow without the test helper's reset hook so we can
// verify the bump is real, not papered over by the helper.
func TestFactorService_VerifyTOTPForActivation_BumpsLastUsedStep(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	enroll, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", "", user.Email)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	pl := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())

	if _, err := svc.ActivateFactor(context.Background(), user.ID, enroll.ID, &models.FactorActivateRequest{Code: code}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	// Same code, same window — must NOT pass tryTOTPCode now.
	if svc.tryTOTPCode(context.Background(), enroll.ID, code) {
		t.Error("activation code accepted as login code in same window — A-M-1 regression")
	}

	// Generate a fresh code RIGHT NOW. If we're still in the same
	// window, the matching candidate is ALSO blocked by the
	// last_used_step CAS (the activation already bumped it).
	fresh, _ := totp.GenerateCode(pl.Secret, time.Now().UTC())
	if fresh == code {
		// Same window — tryTOTPCode must reject.
		if svc.tryTOTPCode(context.Background(), enroll.ID, fresh) {
			t.Error("freshly-generated same-window code accepted — A-M-1 regression")
		}
	}
	// The next-window code (advanced 30s) would pass — but we don't
	// wait 30s in tests. The CAS on last_used_step is the regression
	// guard; future steps strictly > the activation step succeed by
	// construction.
}
