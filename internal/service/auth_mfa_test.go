package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// --- helpers ---

// enrolAndActivateTOTPFor returns (factorID, base32Secret). Password
// is the user's plaintext password, which EnrollTOTP re-verifies as
// step-up. The AuthService isn't used (we go direct to FactorService
// for speed) — it's only in the signature so call-sites read
// naturally.
func enrolAndActivateTOTPFor(t *testing.T, db *gorm.DB, _ *AuthService, user *models.User, password string) (uint, string) {
	t.Helper()
	// Pull the AEAD-wired FactorService out by re-constructing via
	// the same testutil path; AuthService keeps it as an unexported
	// field so tests go around it.
	auditService := createAuditService(db)
	factorSvc := NewFactorService(
		store.NewFactorStore(db),
		store.NewUserStore(db),
		testAuthAEAD(),
		"KitaManager (test)",
		nil,
		auditService,
	)
	ctx := context.Background()
	enroll, err := factorSvc.EnrollTOTP(ctx, user.ID, nil, password, user.Email)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	payload := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, err := totp.GenerateCode(payload.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("gen code: %v", err)
	}
	if _, err := factorSvc.ActivateFactor(ctx, user.ID, enroll.ID, code); err != nil {
		t.Fatalf("activate: %v", err)
	}
	return enroll.ID, payload.Secret
}

// seedPendingMFARow bypasses the password step and creates a
// pending_mfa row for `userID` with the given expiry. Returns the raw
// token.
func seedPendingMFARow(t *testing.T, db *gorm.DB, userID uint, lifetime time.Duration) string {
	t.Helper()
	raw, hashed, err := store.GenerateSessionToken()
	if err != nil {
		t.Fatalf("gen token: %v", err)
	}
	now := time.Now().UTC()
	pv := now
	row := &models.Session{
		ID:                 hashed,
		UserID:             userID,
		Kind:               models.SessionKindPendingMFA,
		CreatedAt:          now,
		ExpiresAt:          now.Add(lifetime),
		PasswordVerifiedAt: &pv,
		CreatedIP:          "127.0.0.1",
		CreatedUserAgent:   "test-ua",
	}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	return raw
}

// --- Login (step 1) ---

// No factor → Login returns Authenticated with a real session, same
// shape as pre-MFA behaviour.
func TestAuthService_Login_NoMFA_ReturnsAuthenticated(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")

	result, err := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.Pending != nil {
		t.Error("no factor enrolled — Pending should be nil")
	}
	if result.Authenticated == nil || result.Authenticated.SessionToken == "" {
		t.Error("expected Authenticated with a session token")
	}
}

// Factor enrolled → Login returns Pending, no session cookie, and no
// regular session row is created yet.
func TestAuthService_Login_WithMFA_ReturnsPending(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	_, _ = enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	result, err := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.Authenticated != nil {
		t.Fatal("MFA-enrolled user should not receive Authenticated at step 1")
	}
	if result.Pending == nil {
		t.Fatal("expected Pending result")
	}
	if result.Pending.PendingToken == "" {
		t.Error("pending token empty")
	}
	if time.Until(result.Pending.ExpiresAt) <= 0 {
		t.Error("pending expires_at is in the past")
	}
	if len(result.Pending.Factors) == 0 {
		t.Error("pending factors list is empty")
	}
	// Factor descriptors must not leak created_at/last_used_at. Only
	// id/type/label are JSON-exposed; the struct itself has no other
	// fields — assert on the literal type.
	for _, f := range result.Pending.Factors {
		if f.ID == 0 || f.Type == "" {
			t.Errorf("descriptor missing id/type: %+v", f)
		}
	}

	// No regular session created at this step.
	var n int64
	if err := db.Model(&models.Session{}).
		Where("user_id = ? AND kind = ?", user.ID, models.SessionKindRegular).
		Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("no regular session should exist after step 1; got %d", n)
	}
	// A pending row must exist.
	if err := db.Model(&models.Session{}).
		Where("user_id = ? AND kind = ?", user.ID, models.SessionKindPendingMFA).
		Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("expected exactly one pending_mfa row, got %d", n)
	}
}

// password_verified_at is set and CreatedAt matches — the row's
// forensic trail is complete.
func TestAuthService_Login_WithMFA_PendingRowShape(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	_, _ = enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	before := time.Now().UTC()
	result, err := svc.Login(ctx, "u@example.com", "pw-123456", "203.0.113.42", "curl/8")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	after := time.Now().UTC()

	var row models.Session
	if err := db.Where("id = ?", store.HashSessionToken(result.Pending.PendingToken)).First(&row).Error; err != nil {
		t.Fatalf("fetch pending: %v", err)
	}
	if row.Kind != models.SessionKindPendingMFA {
		t.Errorf("kind = %q, want pending_mfa", row.Kind)
	}
	if row.MFAChallengeFailures != 0 {
		t.Errorf("mfa_challenge_failures = %d, want 0", row.MFAChallengeFailures)
	}
	if row.PasswordVerifiedAt == nil || row.PasswordVerifiedAt.Before(before) || row.PasswordVerifiedAt.After(after) {
		t.Errorf("password_verified_at out of window: %v", row.PasswordVerifiedAt)
	}
	if row.ExpiresAt.Sub(row.CreatedAt) != PendingMFALifetime {
		t.Errorf("expires_at-created_at = %v, want %v", row.ExpiresAt.Sub(row.CreatedAt), PendingMFALifetime)
	}
	if row.CreatedIP != "203.0.113.42" {
		t.Errorf("created_ip = %q, want the request IP", row.CreatedIP)
	}
	if row.CreatedUserAgent != "curl/8" {
		t.Errorf("created_user_agent = %q, want the request UA", row.CreatedUserAgent)
	}
}

// Wrong password never reaches the MFA branch — the old
// 401-for-wrong-creds response shape is unchanged.
func TestAuthService_Login_WrongPassword_WithMFA(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	_, _ = enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	_, err := svc.Login(ctx, "u@example.com", "WRONG-PASSWORD", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("wrong password: expected ErrUnauthorized, got %v", err)
	}

	// No pending row created for a wrong password.
	var n int64
	_ = db.Model(&models.Session{}).
		Where("user_id = ? AND kind = ?", user.ID, models.SessionKindPendingMFA).
		Count(&n).Error
	if n != 0 {
		t.Errorf("pending row created on wrong password: count = %d", n)
	}
}

// Inactive user with MFA — still rejected as 401 before the MFA
// branch runs.
func TestAuthService_Login_InactiveUser_WithMFA(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	_, _ = enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")
	if err := db.Model(&models.User{}).Where("id = ?", user.ID).Update("active", false).Error; err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	_, err := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("inactive user login: expected ErrUnauthorized, got %v", err)
	}
}

// --- VerifyMFALogin (step 2) ---

func TestAuthService_VerifyMFALogin_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, secret := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	step1, err := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	auth, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, code, "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if auth.SessionToken == "" {
		t.Error("expected session token after verify")
	}

	// Pending row consumed, one regular session present.
	var pendings, regulars int64
	_ = db.Model(&models.Session{}).Where("user_id = ? AND kind = ?", user.ID, models.SessionKindPendingMFA).Count(&pendings).Error
	_ = db.Model(&models.Session{}).Where("user_id = ? AND kind = ?", user.ID, models.SessionKindRegular).Count(&regulars).Error
	if pendings != 0 {
		t.Errorf("pending row not consumed: %d", pendings)
	}
	if regulars != 1 {
		t.Errorf("expected 1 regular session, got %d", regulars)
	}
}

// Verify succeeds using a backup code.
func TestAuthService_VerifyMFALogin_BackupCodeSuccess(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	_, _ = enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	// Rotate a fresh set of backup codes so we have the raw values.
	factorStore := store.NewFactorStore(db)
	bf, err := factorStore.FindBackupCodesFactor(ctx, user.ID)
	if err != nil {
		t.Fatalf("find bf: %v", err)
	}
	auditService := createAuditService(db)
	factorSvc := NewFactorService(factorStore, store.NewUserStore(db), testAuthAEAD(), "KitaManager (test)", nil, auditService)
	payload, err := factorSvc.RegenerateBackupCodes(ctx, user.ID, bf.ID, "pw-123456")
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	rawCode := payload.Codes[0]

	step1, err := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	auth, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, bf.ID, rawCode, "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("verify backup: %v", err)
	}
	if auth.SessionToken == "" {
		t.Error("expected session after backup verify")
	}

	// Second use of the same raw code is rejected (single-use).
	step1b, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	_, err = svc.VerifyMFALogin(ctx, step1b.Pending.PendingToken, bf.ID, rawCode, "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("second use of backup code: expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_VerifyMFALogin_UnknownPendingToken(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	_, err := svc.VerifyMFALogin(ctx, "does-not-exist", 1, "123456", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("unknown pending: expected ErrUnauthorized, got %v", err)
	}
}

// Empty pending token — uniform 401, no internal error leak.
func TestAuthService_VerifyMFALogin_EmptyPendingToken(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	_, err := svc.VerifyMFALogin(ctx, "", 1, "123456", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("empty pending: expected ErrUnauthorized, got %v", err)
	}
}

// A REGULAR session cookie smuggled in as pending_token must be
// rejected — otherwise the pending/session separation leaks.
func TestAuthService_VerifyMFALogin_RegularSessionTokenRejected(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")

	// Non-MFA user → regular session issued.
	step1, err := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	regularToken := step1.Authenticated.SessionToken

	// Using it as a pending token must fail.
	_, err = svc.VerifyMFALogin(ctx, regularToken, 1, "000000", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("regular session as pending token: expected ErrUnauthorized, got %v", err)
	}
	// And the original session must still be usable.
	sess, err := store.NewSessionStore(db).Lookup(ctx, store.HashSessionToken(regularToken))
	if err != nil {
		t.Errorf("regular session should still exist: %v", err)
	}
	if sess == nil || sess.UserID != user.ID {
		t.Error("regular session identity mismatch")
	}
}

// Expired pending → 401, same as unknown.
func TestAuthService_VerifyMFALogin_ExpiredPending(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")

	// Seed a pending row with past expiry.
	raw := seedPendingMFARow(t, db, user.ID, -time.Minute)

	_, err := svc.VerifyMFALogin(ctx, raw, 1, "123456", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expired pending: expected ErrUnauthorized, got %v", err)
	}
}

// Cross-user factor_id (Alice's pending token, Bob's factor id) → 401.
func TestAuthService_VerifyMFALogin_CrossUserFactorID(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	alice := createTestUserWithHashedPassword(t, db, "Alice", "alice@example.com", "pw-alice01")
	bob := createTestUserWithHashedPassword(t, db, "Bob", "bob@example.com", "pw-bob00001")
	_, _ = enrolAndActivateTOTPFor(t, db, svc, alice, "pw-alice01")
	bobFactorID, bobSecret := enrolAndActivateTOTPFor(t, db, svc, bob, "pw-bob00001")

	step1, err := svc.Login(ctx, "alice@example.com", "pw-alice01", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("alice login: %v", err)
	}
	code, _ := totp.GenerateCode(bobSecret, time.Now().UTC())
	_, err = svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, bobFactorID, code, "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("cross-user factor id: expected ErrUnauthorized, got %v", err)
	}
	// Alice's pending row should NOT have its failure counter bumped —
	// we decided unknown-factor is not a brute-force signal and so
	// doesn't burn retry budget.
	var failures int
	_ = db.Model(&models.Session{}).Select("mfa_challenge_failures").
		Where("id = ?", store.HashSessionToken(step1.Pending.PendingToken)).
		Scan(&failures)
	if failures != 0 {
		t.Errorf("cross-user shouldn't bump counter, got %d", failures)
	}
}

// Wrong TOTP code bumps the per-row counter but keeps the row alive
// until the limit.
func TestAuthService_VerifyMFALogin_WrongCode_BumpsCounter(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, _ := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	step1, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")

	// 4 wrong codes, row should still exist after.
	for i := 1; i < MFAChallengeFailureLimit; i++ {
		_, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, "000000", "127.0.0.1", "ua")
		if !errors.Is(err, apperror.ErrUnauthorized) {
			t.Errorf("attempt %d: expected ErrUnauthorized, got %v", i, err)
		}
	}
	var row models.Session
	if err := db.Where("id = ?", store.HashSessionToken(step1.Pending.PendingToken)).First(&row).Error; err != nil {
		t.Fatalf("pending should still exist before limit: %v", err)
	}
	if row.MFAChallengeFailures != MFAChallengeFailureLimit-1 {
		t.Errorf("counter = %d, want %d", row.MFAChallengeFailures, MFAChallengeFailureLimit-1)
	}
}

// Nth wrong code destroys the pending row and returns 429.
func TestAuthService_VerifyMFALogin_FiveWrongCodes_Locks(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, _ := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	step1, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	for i := 1; i < MFAChallengeFailureLimit; i++ {
		_, _ = svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, "000000", "127.0.0.1", "ua")
	}
	// Nth: should be 429 and destroy the row.
	_, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, "000000", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrTooManyRequests) {
		t.Errorf("limit attempt: expected ErrTooManyRequests, got %v", err)
	}
	var n int64
	_ = db.Model(&models.Session{}).Where("id = ?", store.HashSessionToken(step1.Pending.PendingToken)).Count(&n).Error
	if n != 0 {
		t.Errorf("row should be destroyed after limit; count = %d", n)
	}

	// Subsequent request returns 401 (no row).
	_, err = svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, "000000", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("post-lock attempt: expected ErrUnauthorized, got %v", err)
	}
}

// After N pending rows consumed by wrong codes, the per-user
// audit-based counter kicks in and the verify endpoint returns 429
// even on a fresh pending row (blocks distributed brute force).
func TestAuthService_VerifyMFALogin_PerUserLockout(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, _ := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	// Cheap way: forge audit events directly. This lets the test run
	// in O(1) audit inserts rather than cycle through 20 pending rows.
	auditStore := store.NewAuditStore(db)
	for range mfaPerUserLockoutThreshold {
		if err := auditStore.Create(ctx, &models.AuditLog{
			UserID:    &user.ID,
			Action:    models.AuditActionMFAChallengeFailed,
			Timestamp: time.Now().UTC(),
			Success:   false,
		}); err != nil {
			t.Fatalf("seed audit: %v", err)
		}
	}

	step1, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	_, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, "000000", "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrTooManyRequests) {
		t.Errorf("per-user lockout: expected ErrTooManyRequests, got %v", err)
	}
	// The pending row we provided should be destroyed so the user
	// restarts cleanly.
	var n int64
	_ = db.Model(&models.Session{}).Where("id = ?", store.HashSessionToken(step1.Pending.PendingToken)).Count(&n).Error
	if n != 0 {
		t.Errorf("pending row should be destroyed on per-user lockout; count = %d", n)
	}
}

// Two concurrent VerifyMFALogin calls with the same valid code and
// the same pending_mfa token: exactly one should succeed, the other
// must get 401 (pending already consumed) — matches the "single-use
// session material" semantics the factor store already guarantees.
func TestAuthService_VerifyMFALogin_ConcurrentSameCode_OneWinner(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, secret := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	step1, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	code, _ := totp.GenerateCode(secret, time.Now().UTC())

	var wg sync.WaitGroup
	results := make([]error, 2)
	tokens := make([]string, 2)
	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			auth, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, code, "127.0.0.1", "ua")
			results[idx] = err
			if auth != nil {
				tokens[idx] = auth.SessionToken
			}
		}(i)
	}
	wg.Wait()

	winners := 0
	for _, r := range results {
		if r == nil {
			winners++
		}
	}
	// The first caller wins: it consumes both the TOTP step and the
	// pending row. The second sees 401 (either invalid code per the
	// AcceptTOTPStep race, or pending_not_found).
	if winners != 1 {
		t.Errorf("expected exactly one winner, got %d (errors=%v)", winners, results)
	}
}

// After success, reusing the same pending token is 401 (row is gone).
func TestAuthService_VerifyMFALogin_ReuseAfterSuccess(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, secret := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")
	_ = user

	step1, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	if _, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, code, "127.0.0.1", "ua"); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	_, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, code, "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("reuse after success: expected ErrUnauthorized, got %v", err)
	}
}

// User deactivated between step 1 and step 2 — verify returns 401
// and deletes the pending row.
func TestAuthService_VerifyMFALogin_UserDeactivatedMidFlow(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, secret := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	step1, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")

	// Admin deactivates.
	if err := db.Model(&models.User{}).Where("id = ?", user.ID).Update("active", false).Error; err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	_, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, code, "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("deactivated user: expected ErrUnauthorized, got %v", err)
	}

	// Pending row gone so re-activation + retry has to restart.
	var n int64
	_ = db.Model(&models.Session{}).Where("id = ?", store.HashSessionToken(step1.Pending.PendingToken)).Count(&n).Error
	if n != 0 {
		t.Errorf("pending row should be destroyed on deactivated user; count = %d", n)
	}
}

// Factor deleted mid-flow — verify returns 401.
func TestAuthService_VerifyMFALogin_FactorDeletedMidFlow(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, secret := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	step1, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")

	// Delete the factor row directly (admin intervention / account
	// cleanup would have the same effect).
	if err := db.Where("id = ?", factorID).Delete(&models.Factor{}).Error; err != nil {
		t.Fatalf("delete factor: %v", err)
	}

	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	_, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, factorID, code, "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("deleted factor: expected ErrUnauthorized, got %v", err)
	}
}

// TOTP replay-prevention: a valid code used once on /verify cannot be
// used again on a different pending row within the same step window.
func TestAuthService_VerifyMFALogin_TOTPReplayAcrossPendings(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	factorID, secret := enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	step1a, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	step1b, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")

	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	if _, err := svc.VerifyMFALogin(ctx, step1a.Pending.PendingToken, factorID, code, "127.0.0.1", "ua"); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	// Same code on step1b: rejected by AcceptTOTPStep replay check.
	_, err := svc.VerifyMFALogin(ctx, step1b.Pending.PendingToken, factorID, code, "127.0.0.1", "ua")
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("replayed TOTP code: expected ErrUnauthorized, got %v", err)
	}
}

// Hyphen/space-normalized backup codes are accepted.
func TestAuthService_VerifyMFALogin_BackupCodeNormalisation(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	_, _ = enrolAndActivateTOTPFor(t, db, svc, user, "pw-123456")

	factorStore := store.NewFactorStore(db)
	bf, _ := factorStore.FindBackupCodesFactor(ctx, user.ID)
	auditService := createAuditService(db)
	factorSvc := NewFactorService(factorStore, store.NewUserStore(db), testAuthAEAD(), "KitaManager (test)", nil, auditService)
	payload, err := factorSvc.RegenerateBackupCodes(ctx, user.ID, bf.ID, "pw-123456")
	if err != nil {
		t.Fatalf("regen: %v", err)
	}
	raw := payload.Codes[0]
	mangled := strings.ToUpper(strings.ReplaceAll(raw, "-", " "))

	step1, _ := svc.Login(ctx, "u@example.com", "pw-123456", "127.0.0.1", "ua")
	if _, err := svc.VerifyMFALogin(ctx, step1.Pending.PendingToken, bf.ID, mangled, "127.0.0.1", "ua"); err != nil {
		t.Errorf("mangled backup code should verify: %v", err)
	}
}

// Login timing must NOT reveal whether a user has MFA. This is a
// smoke test — nothing asserts exact latency, just that both
// branches run the same number of expensive operations (bcrypt +
// one DB read + one DB write). Measured branch mean should be within
// a reasonable factor; we just assert both code paths complete.
func TestAuthService_Login_MFABranchShapeParity(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	userA := createTestUserWithHashedPassword(t, db, "A", "a@example.com", "pw-123456")
	_ = userA
	userB := createTestUserWithHashedPassword(t, db, "B", "b@example.com", "pw-123456")
	_, _ = enrolAndActivateTOTPFor(t, db, svc, userB, "pw-123456")

	// Both must succeed with their respective result shapes.
	resA, err := svc.Login(ctx, "a@example.com", "pw-123456", "127.0.0.1", "ua")
	if err != nil || resA.Authenticated == nil {
		t.Errorf("A login: %v", err)
	}
	resB, err := svc.Login(ctx, "b@example.com", "pw-123456", "127.0.0.1", "ua")
	if err != nil || resB.Pending == nil {
		t.Errorf("B login: %v", err)
	}
}
