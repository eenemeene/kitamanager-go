package service

// Edge cases around TOTP step reuse on the *step-up* path (delete / regenerate),
// as opposed to the login path already covered in auth_mfa_test.go.
//
// Why these exist: the website screenshot script enrolled TOTP and then
// immediately deleted it with the same code. The delete was rejected — correctly,
// because ActivateFactor had already consumed that time step via
// AcceptTOTPStep's compare-and-set — but the script ignored the response and left
// the seeded admin permanently enrolled. The behaviour was right and the caller
// was wrong, so these tests pin the behaviour at the service API level (not just
// tryTOTPCode) and, importantly, prove the escape hatches still work: a code from
// the *next* step, and backup codes.
//
// None of these sleep. The TOTP skew window is ±1 step (totpSkewSteps), so a code
// generated for now+totpPeriod is accepted at the current wall clock while
// carrying a strictly greater step — which is exactly what a user's authenticator
// shows them 30 seconds later.

import (
	"context"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// enrolActiveTOTP enrols and activates a TOTP factor, returning the factor ID,
// the base32 secret, the code used for activation, and the backup codes minted
// alongside the first primary factor.
func enrolActiveTOTP(t *testing.T, svc *FactorService, userID uint, email string) (uint, string, string, []string) {
	t.Helper()
	ctx := context.Background()

	enroll, err := svc.EnrollTOTP(ctx, userID, nil, "pw", "", email)
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	payload := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	activationCode, err := totp.GenerateCode(payload.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generate activation code: %v", err)
	}
	resp, err := svc.ActivateFactor(ctx, userID, enroll.ID, &models.FactorActivateRequest{Code: activationCode})
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if resp.BackupCodes == nil {
		t.Fatal("expected backup codes on first primary activation")
	}
	return enroll.ID, payload.Secret, activationCode, resp.BackupCodes.Codes
}

// nextStepCode returns the code the user's authenticator will display one step
// from now. It is accepted today because of the ±1 skew window, but its step is
// strictly greater than the step activation consumed.
func nextStepCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now().UTC().Add(time.Duration(totpPeriod)*time.Second))
	if err != nil {
		t.Fatalf("generate next-step code: %v", err)
	}
	return code
}

// The exact sequence the screenshot script performed. The activation code must
// not be reusable to delete the factor it just activated.
func TestFactorService_DeleteFactor_RejectsActivationCodeReuse(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	factorID, _, activationCode, _ := enrolActiveTOTP(t, svc, user.ID, user.Email)

	err := svc.DeleteFactor(context.Background(), user.ID, factorID, "pw", activationCode)
	if err == nil {
		t.Fatal("delete accepted the code that activation already consumed — replay protection regressed")
	}

	// The factor must still be there; a rejected delete may not half-apply.
	factors, lerr := svc.ListForUser(context.Background(), user.ID)
	if lerr != nil {
		t.Fatalf("list: %v", lerr)
	}
	var found bool
	for _, f := range factors {
		if f.ID == factorID {
			found = true
		}
	}
	if !found {
		t.Error("factor disappeared even though the delete was rejected")
	}
}

// The escape hatch: the code for the following step is accepted, because its
// step is strictly greater than the consumed one. This is what the script should
// have done (wait for the next step) and it needs no 30s sleep to verify.
func TestFactorService_DeleteFactor_AcceptsNextStepCode(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	factorID, secret, activationCode, _ := enrolActiveTOTP(t, svc, user.ID, user.Email)

	fresh := nextStepCode(t, secret)
	if fresh == activationCode {
		t.Skip("next-step code collided with the activation code; skew boundary, rerun")
	}

	if err := svc.DeleteFactor(context.Background(), user.ID, factorID, "pw", fresh); err != nil {
		t.Fatalf("delete with next-step code failed: %v", err)
	}

	factors, err := svc.ListForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, f := range factors {
		if f.ID == factorID {
			t.Error("factor still present after a successful delete")
		}
	}
}

// A code from the step *before* the consumed one is inside the skew window but
// is not strictly greater, so it must be rejected too. Guards against a
// >= comparison creeping into AcceptTOTPStep.
func TestFactorService_DeleteFactor_RejectsPreviousStepCode(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	factorID, secret, activationCode, _ := enrolActiveTOTP(t, svc, user.ID, user.Email)

	previous, err := totp.GenerateCode(secret, time.Now().UTC().Add(-time.Duration(totpPeriod)*time.Second))
	if err != nil {
		t.Fatalf("generate previous-step code: %v", err)
	}
	if previous == activationCode {
		t.Skip("previous-step code collided with the activation code; skew boundary, rerun")
	}

	if err := svc.DeleteFactor(context.Background(), user.ID, factorID, "pw", previous); err == nil {
		t.Error("delete accepted a code from an already-passed step")
	}
}

// Recovery path 1: with the TOTP step exhausted, a backup code must still let the
// user remove the factor. This is the route for "my authenticator is gone".
func TestFactorService_DeleteFactor_BackupCodeWorksAfterTOTPStepConsumed(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	factorID, _, activationCode, backupCodes := enrolActiveTOTP(t, svc, user.ID, user.Email)

	// Confirm the TOTP route really is blocked in this window first.
	if err := svc.DeleteFactor(context.Background(), user.ID, factorID, "pw", activationCode); err == nil {
		t.Fatal("precondition failed: reused TOTP code was accepted")
	}

	// Recovery must be unaffected by the TOTP step counter.
	if err := svc.DeleteFactor(context.Background(), user.ID, factorID, "pw", backupCodes[0]); err != nil {
		t.Fatalf("backup code rejected for delete after TOTP step consumed: %v", err)
	}
}

// Recovery path 2: a backup code is single-use on the step-up path too — the same
// code must not remove a second factor.
func TestFactorService_DeleteFactor_BackupCodeIsSingleUse(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	firstID, _, _, backupCodes := enrolActiveTOTP(t, svc, user.ID, user.Email)

	// A second TOTP factor, activated with a next-step code so it does not
	// collide with the first activation's consumed step.
	enroll2, err := svc.EnrollTOTP(context.Background(), user.ID, nil, "pw", backupCodes[0], user.Email)
	if err != nil {
		t.Fatalf("second enrol: %v", err)
	}
	payload2 := enroll2.Enrollment.(models.TOTPEnrollmentPayload)
	code2, err := totp.GenerateCode(payload2.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := svc.ActivateFactor(context.Background(), user.ID, enroll2.ID, &models.FactorActivateRequest{Code: code2}); err != nil {
		t.Fatalf("second activate: %v", err)
	}

	// backupCodes[0] was consumed by the enrolment step-up above; reusing it
	// to delete must fail.
	if err := svc.DeleteFactor(context.Background(), user.ID, firstID, "pw", backupCodes[0]); err == nil {
		t.Error("a spent backup code was accepted a second time on the step-up path")
	}

	// A different, unspent code still works.
	if err := svc.DeleteFactor(context.Background(), user.ID, firstID, "pw", backupCodes[1]); err != nil {
		t.Errorf("unspent backup code rejected: %v", err)
	}
}

// Deleting the last primary factor must leave no orphaned recovery codes behind:
// an account with backup codes but no primary would still be prompted for MFA at
// login with nothing to satisfy it.
func TestFactorService_DeleteFactor_NoOrphanedBackupCodesAfterLastPrimary(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := newFactorService(t, db)
	user := createUserWithPassword(t, db, "U", "u@example.com", "pw")

	factorID, secret, _, _ := enrolActiveTOTP(t, svc, user.ID, user.Email)

	if err := svc.DeleteFactor(context.Background(), user.ID, factorID, "pw", nextStepCode(t, secret)); err != nil {
		t.Fatalf("delete last primary: %v", err)
	}

	factors, err := svc.ListForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(factors) != 0 {
		for _, f := range factors {
			t.Errorf("factor %d (%s) survived deletion of the last primary", f.ID, f.Type)
		}
	}

	// And the account must be back to password-only.
	hasPrimary, err := svc.HasActivePrimaryFactor(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("HasActivePrimaryFactor: %v", err)
	}
	if hasPrimary {
		t.Error("HasActivePrimaryFactor still true after deleting the last primary")
	}
}
