package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// createTestUserWithHashedPassword creates a user with a bcrypt-hashed password
// so that AuthService.Login can verify the credentials.
func createTestUserWithHashedPassword(t *testing.T, db *gorm.DB, name, email, password string) *models.User {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := &models.User{
		Name:     name,
		Email:    email,
		Password: string(hashed),
		Active:   true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// countSessionsForUser returns how many session rows exist for the given user
// ID right now. Used as the observable side-effect in tests that assert
// invalidation behavior.
func countSessionsForUser(t *testing.T, db *gorm.DB, userID uint) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&models.Session{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	return count
}

func TestAuthService_Login_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	result, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Authenticated.SessionToken == "" {
		t.Error("expected non-empty session token")
	}
	if result.Authenticated.CSRFToken == "" {
		t.Error("expected non-empty CSRF token")
	}
	if result.Authenticated.ExpiresIn != int64(SessionLifetime.Seconds()) {
		t.Errorf("ExpiresIn = %d, want %d", result.Authenticated.ExpiresIn, int64(SessionLifetime.Seconds()))
	}

	// A session row must exist on the server keyed by sha256 of the returned
	// raw token.
	sess := store.NewSessionStore(db)
	lookup, err := sess.Lookup(ctx, store.HashSessionToken(result.Authenticated.SessionToken))
	if err != nil {
		t.Fatalf("lookup session after login: %v", err)
	}
	if lookup.UserEmail != "test@example.com" {
		t.Errorf("looked-up session email = %q", lookup.UserEmail)
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	_, err := svc.Login(ctx, "test@example.com", "wrongpassword", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	_, err := svc.Login(ctx, "nonexistent@example.com", "password", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Inactive User", "inactive@example.com", "password123")
	db.Model(user).Update("active", false)

	_, err := svc.Login(ctx, "inactive@example.com", "password123", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Login_AccountLockout(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	// Insert failed login audit entries directly (bypassing async channel)
	// so they are visible to CountRecentFailedLogins immediately.
	for range lockoutThreshold {
		if err := db.Create(&models.AuditLog{
			UserEmail: "test@example.com",
			Action:    models.AuditActionLoginFailed,
			IPAddress: "127.0.0.1",
			Success:   false,
			Timestamp: time.Now(),
		}).Error; err != nil {
			t.Fatalf("failed to insert audit log: %v", err)
		}
	}

	// Next attempt should be locked out even with correct password.
	_, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrTooManyRequests) {
		t.Errorf("expected ErrTooManyRequests, got %v", err)
	}
}

func TestAuthService_Login_LockoutWindowExpiry(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "lockout-expiry@example.com", "password123")

	outsideWindow := time.Now().Add(-lockoutWindow - time.Minute)
	for range lockoutThreshold {
		if err := db.Create(&models.AuditLog{
			UserEmail: "lockout-expiry@example.com",
			Action:    models.AuditActionLoginFailed,
			IPAddress: "127.0.0.1",
			Success:   false,
			Timestamp: outsideWindow,
		}).Error; err != nil {
			t.Fatalf("failed to insert audit log: %v", err)
		}
	}

	result, err := svc.Login(ctx, "lockout-expiry@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected login to succeed after lockout window expiry, got %v", err)
	}
	if result.Authenticated.SessionToken == "" {
		t.Error("expected non-empty session token")
	}
}

func TestAuthService_Logout_DeletesSession(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	sess := store.NewSessionStore(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")
	loginResult, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := svc.Logout(ctx, store.HashSessionToken(loginResult.Authenticated.SessionToken), 1, "test@example.com", "127.0.0.1"); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := sess.Lookup(ctx, store.HashSessionToken(loginResult.Authenticated.SessionToken)); err != store.ErrNotFound {
		t.Errorf("expected session to be deleted after logout, got %v", err)
	}
}

func TestAuthService_Logout_EmptyTokenIsNoOp(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	// Should not panic or error.
	if err := svc.Logout(ctx, "", 0, "", "127.0.0.1"); err != nil {
		t.Errorf("empty session hash must be a no-op, got %v", err)
	}
}

func TestAuthService_Logout_UnknownTokenIsNoOp(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	// Should not panic or error: deleting a row that is not there is how a
	// double-click on "logout" behaves, and it is not a failure.
	if err := svc.Logout(ctx, "never-issued-this", 0, "", "127.0.0.1"); err != nil {
		t.Errorf("unknown session hash must be a no-op, got %v", err)
	}
}

func TestAuthService_Logout_EmitsAuditRow(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	result, err := svc.Login(ctx, "u@example.com", "pw-123456", "10.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if err := svc.Logout(ctx, store.HashSessionToken(result.Authenticated.SessionToken), user.ID, user.Email, "10.0.0.1"); err != nil {
		t.Fatalf("logout: %v", err)
	}
	svc.auditService.Shutdown()

	var rows []models.AuditLog
	if err := db.Where("action = ? AND user_id = ?",
		models.AuditActionLogout, user.ID).Find(&rows).Error; err != nil {
		t.Fatalf("query audit rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 logout audit row, got %d", len(rows))
	}
	if rows[0].IPAddress != "10.0.0.1" || rows[0].UserEmail != user.Email || !rows[0].Success {
		t.Errorf("unexpected row: %+v", rows[0])
	}
}

// An expired / unknown session must NOT emit a logout audit row —
// otherwise a user double-clicking "logout" after their session
// already expired would spam the audit log.
func TestAuthService_Logout_UnknownTokenDoesNotEmitAudit(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	_ = svc.Logout(ctx, "never-issued-this", 0, "", "10.0.0.1")
	svc.auditService.Shutdown()

	var count int64
	if err := db.Model(&models.AuditLog{}).
		Where("action = ?", models.AuditActionLogout).Count(&count).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 logout audit rows for unknown-token logout, got %d", count)
	}
}

func TestAuthService_ChangePassword_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	result, err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456", "", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.SessionToken == "" {
		t.Error("expected non-empty session token after password change")
	}

	// New password works.
	if _, err := svc.Login(ctx, "test@example.com", "newpassword456", "127.0.0.1", "ua"); err != nil {
		t.Errorf("login with new password failed: %v", err)
	}
	// Old password does not.
	if _, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "ua"); err == nil {
		t.Error("expected login with old password to fail")
	}
}

func TestAuthService_ChangePassword_WrongCurrentPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	_, err := svc.ChangePassword(ctx, user.ID, "wrongpassword", "newpassword456", "", "127.0.0.1", "ua")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// ChangePassword must log every failed attempt so the lockout counter works.
func TestAuthService_ChangePassword_FailureIsAuditLogged(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	_, err := svc.ChangePassword(ctx, user.ID, "wrongpassword", "newpassword456", "", "127.0.0.1", "ua")
	if err == nil {
		t.Fatal("expected error for wrong current password")
	}

	// The audit log is written async. Drain it.
	svc.auditService.Shutdown()

	var rows []models.AuditLog
	if err := db.Where("action = ? AND user_id = ?",
		models.AuditActionPasswordChangeFailed, user.ID).Find(&rows).Error; err != nil {
		t.Fatalf("query audit logs: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 password_change_failed audit row, got %d", len(rows))
	}
	if rows[0].Success {
		t.Error("audit row must be marked unsuccessful")
	}
	if rows[0].IPAddress != "127.0.0.1" {
		t.Errorf("IP address = %q, want 127.0.0.1", rows[0].IPAddress)
	}
}

func TestAuthService_ChangePassword_LocksOutAfterThreshold(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	// Seed the lockout counter directly.
	for range passwordChangeLockoutThreshold {
		if err := db.Create(&models.AuditLog{
			UserID:    &user.ID,
			UserEmail: user.Email,
			Action:    models.AuditActionPasswordChangeFailed,
			IPAddress: "127.0.0.1",
			Success:   false,
			Timestamp: time.Now(),
		}).Error; err != nil {
			t.Fatalf("seed audit row: %v", err)
		}
	}

	// Even the CORRECT current password must be refused once locked out.
	_, err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456", "", "127.0.0.1", "ua")
	if err == nil {
		t.Fatal("expected 429, got nil — lockout failed to fire with correct password")
	}
	if !errors.Is(err, apperror.ErrTooManyRequests) {
		t.Errorf("expected ErrTooManyRequests, got %v", err)
	}
}

func TestAuthService_ChangePassword_LockoutExpiresAfterWindow(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	// Seed failed attempts from BEFORE the lockout window — they must not
	// count against the current window.
	oldEnough := time.Now().Add(-2 * passwordChangeLockoutWindow)
	for range passwordChangeLockoutThreshold {
		if err := db.Create(&models.AuditLog{
			UserID:    &user.ID,
			UserEmail: user.Email,
			Action:    models.AuditActionPasswordChangeFailed,
			Success:   false,
			Timestamp: oldEnough,
		}).Error; err != nil {
			t.Fatalf("seed audit row: %v", err)
		}
	}

	_, err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456", "", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("expected success — stale failures must not lock out, got %v", err)
	}
}

func TestAuthService_ChangePassword_SuccessIsAuditLogged(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	_, err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456", "", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("change password: %v", err)
	}

	svc.auditService.Shutdown()

	var rows []models.AuditLog
	if err := db.Where("action = ? AND user_id = ?",
		models.AuditActionPasswordChange, user.ID).Find(&rows).Error; err != nil {
		t.Fatalf("query audit logs: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 password_change audit row, got %d", len(rows))
	}
	if !rows[0].Success {
		t.Error("success row must be marked Success=true")
	}
}

// When no current-session id-hash is supplied, ChangePassword must remove
// EVERY session the user has — the path for out-of-band invocations (tests,
// admin tooling).
func TestAuthService_ChangePassword_WipesAllSessions_WhenNoCurrentSession(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	// Simulate two live sessions (different devices).
	_, hashA, _ := store.GenerateSessionToken()
	_, hashB, _ := store.GenerateSessionToken()
	for _, h := range []string{hashA, hashB} {
		if err := db.Create(&models.Session{
			ID:        h,
			UserID:    user.ID,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}).Error; err != nil {
			t.Fatalf("seed session: %v", err)
		}
	}

	result, err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456", "", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("change password: %v", err)
	}

	// Only the freshly issued session should exist.
	if n := countSessionsForUser(t, db, user.ID); n != 1 {
		t.Errorf("expected exactly 1 session after change, got %d", n)
	}
	// And that new session must be the one we returned.
	newHash := store.HashSessionToken(result.SessionToken)
	var count int64
	db.Model(&models.Session{}).Where("id = ? AND user_id = ?", newHash, user.ID).Count(&count)
	if count != 1 {
		t.Error("freshly issued session is missing from DB")
	}
}

// With a current-session id-hash supplied (the normal request path), the
// caller's current session is rotated (deleted + replaced) and every OTHER
// session belonging to the user is deleted.
func TestAuthService_ChangePassword_RotatesCurrent_RevokesOthers(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	_, currentHash, _ := store.GenerateSessionToken()
	_, otherHash, _ := store.GenerateSessionToken()
	for _, h := range []string{currentHash, otherHash} {
		if err := db.Create(&models.Session{
			ID:        h,
			UserID:    user.ID,
			CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}).Error; err != nil {
			t.Fatalf("seed session: %v", err)
		}
	}

	result, err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456", currentHash, "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("change password: %v", err)
	}

	// Old current session gone.
	var count int64
	db.Model(&models.Session{}).Where("id = ?", currentHash).Count(&count)
	if count != 0 {
		t.Error("old current session was not rotated")
	}
	// Other session gone.
	db.Model(&models.Session{}).Where("id = ?", otherHash).Count(&count)
	if count != 0 {
		t.Error("other sessions were not revoked")
	}
	// Newly issued session present.
	newHash := store.HashSessionToken(result.SessionToken)
	db.Model(&models.Session{}).Where("id = ?", newHash).Count(&count)
	if count != 1 {
		t.Error("freshly issued session missing from DB")
	}
}

func TestAuthService_GetCurrentUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	found, err := svc.GetCurrentUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("ID = %d, want %d", found.ID, user.ID)
	}
	if found.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", found.Email)
	}
}

func TestAuthService_GetCurrentUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	_, err := svc.GetCurrentUser(ctx, 99999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAuthService_ListSessions_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "User", "u@example.com", "pw-123456")

	rows, err := svc.ListSessions(ctx, user.ID, "")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected empty list, got %d rows", len(rows))
	}
}

func TestAuthService_ListSessions_MarksCurrent(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	createTestUserWithHashedPassword(t, db, "User", "u@example.com", "pw-123456")

	// Two logins — two sessions. The caller's "current" session is the
	// one we pass to ListSessions via its id-hash.
	a, err := svc.Login(ctx, "u@example.com", "pw-123456", "10.0.0.1", "ua-a")
	if err != nil {
		t.Fatalf("login a: %v", err)
	}
	b, err := svc.Login(ctx, "u@example.com", "pw-123456", "10.0.0.2", "ua-b")
	if err != nil {
		t.Fatalf("login b: %v", err)
	}

	// Find the user ID from the second session's lookup.
	look, err := store.NewSessionStore(db).Lookup(ctx, store.HashSessionToken(a.Authenticated.SessionToken))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	userID := look.UserID

	rows, err := svc.ListSessions(ctx, userID, store.HashSessionToken(b.Authenticated.SessionToken))
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(rows))
	}
	var currentCount, nonCurrentCount int
	for _, r := range rows {
		if r.Current {
			currentCount++
			if r.ID != store.HashSessionToken(b.Authenticated.SessionToken) {
				t.Errorf("wrong session flagged as current: %q", r.ID)
			}
		} else {
			nonCurrentCount++
		}
	}
	if currentCount != 1 || nonCurrentCount != 1 {
		t.Errorf("expected exactly one Current row, got current=%d non=%d", currentCount, nonCurrentCount)
	}
}

func TestAuthService_ListSessions_NoCurrentHash(t *testing.T) {
	// When called without a current-session hash (e.g. from an admin
	// tool), no row is flagged as current.
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	createTestUserWithHashedPassword(t, db, "User", "u@example.com", "pw-123456")
	_, err := svc.Login(ctx, "u@example.com", "pw-123456", "10.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	look, _ := store.NewSessionStore(db).Lookup(ctx,
		store.HashSessionToken("dummy")) // will fail, use a different path
	_ = look
	// Get userID via FindByEmail instead.
	userStore := store.NewUserStore(db)
	u, err := userStore.FindByEmail(ctx, "u@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}

	rows, err := svc.ListSessions(ctx, u.ID, "")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 session, got %d", len(rows))
	}
	if rows[0].Current {
		t.Error("no session should be flagged as current when hash is empty")
	}
}

func TestAuthService_RevokeSession_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	createTestUserWithHashedPassword(t, db, "User", "u@example.com", "pw-123456")
	result, err := svc.Login(ctx, "u@example.com", "pw-123456", "10.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	u, _ := store.NewUserStore(db).FindByEmail(ctx, "u@example.com")

	hash := store.HashSessionToken(result.Authenticated.SessionToken)
	if err := svc.RevokeSession(ctx, u.ID, hash, u.Email, "10.0.0.1"); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	// Session must be gone.
	sess := store.NewSessionStore(db)
	if _, err := sess.Lookup(ctx, hash); err != store.ErrNotFound {
		t.Errorf("session should be gone, got %v", err)
	}
}

func TestAuthService_RevokeSession_UnknownID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "User", "u@example.com", "pw-123456")

	err := svc.RevokeSession(ctx, user.ID, store.HashSessionToken("never-issued"), user.Email, "10.0.0.1")
	if err == nil {
		t.Fatal("expected NotFound, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAuthService_RevokeSession_EmitsAuditRow(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")
	result, err := svc.Login(ctx, "u@example.com", "pw-123456", "10.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	u, _ := store.NewUserStore(db).FindByEmail(ctx, "u@example.com")
	hash := store.HashSessionToken(result.Authenticated.SessionToken)

	if err := svc.RevokeSession(ctx, u.ID, hash, u.Email, "10.0.0.1"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	svc.auditService.Shutdown()

	var rows []models.AuditLog
	if err := db.Where("action = ? AND user_id = ?",
		models.AuditActionSessionRevoked, u.ID).Find(&rows).Error; err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 session_revoked row, got %d", len(rows))
	}
	if rows[0].ResourceType != "session" || !rows[0].Success {
		t.Errorf("unexpected row shape: %+v", rows[0])
	}
	// The session id hash belongs in Details, not ResourceID (which is
	// a uint and can't carry a hex string).
	if !stringContains(rows[0].Details, hash) {
		t.Errorf("details should embed session id hash; got %q", rows[0].Details)
	}
}

// A NotFound revoke (wrong id / cross-user) must NOT emit an audit
// row — the handler maps it to 404 and the event is not a security
// signal, just a stale UI.
func TestAuthService_RevokeSession_NotFoundDoesNotEmitAudit(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	user := createTestUserWithHashedPassword(t, db, "U", "u@example.com", "pw-123456")

	err := svc.RevokeSession(ctx, user.ID, store.HashSessionToken("nope"), user.Email, "10.0.0.1")
	if err == nil {
		t.Fatal("expected NotFound")
	}
	svc.auditService.Shutdown()

	var count int64
	if err := db.Model(&models.AuditLog{}).
		Where("action = ?", models.AuditActionSessionRevoked).Count(&count).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 session_revoked rows after NotFound, got %d", count)
	}
}

func stringContains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && findSubstring(haystack, needle) >= 0
}

func findSubstring(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestAuthService_RevokeSession_CrossUser_NotFound(t *testing.T) {
	// Security invariant: Bob cannot revoke Alice's session even with
	// her id-hash. The service returns NotFound — same as an unknown id
	// — so the endpoint does not leak session existence.
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()
	createTestUserWithHashedPassword(t, db, "Alice", "alice@example.com", "pw-123456")
	bob := createTestUserWithHashedPassword(t, db, "Bob", "bob@example.com", "pw-123456")

	aliceSession, err := svc.Login(ctx, "alice@example.com", "pw-123456", "10.0.0.1", "ua")
	if err != nil {
		t.Fatalf("alice login: %v", err)
	}
	aliceHash := store.HashSessionToken(aliceSession.Authenticated.SessionToken)

	err = svc.RevokeSession(ctx, bob.ID, aliceHash, bob.Email, "10.0.0.2")
	if err == nil {
		t.Fatal("bob should not be able to revoke alice's session")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound to avoid leaking session existence, got %v", err)
	}

	// Alice's session must still exist.
	sess := store.NewSessionStore(db)
	if _, err := sess.Lookup(ctx, aliceHash); err != nil {
		t.Errorf("alice's session was wrongly deleted: %v", err)
	}
}

// Sessions issued by Login must have the configured lifetime.
func TestAuthService_Login_SessionLifetime(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	sess := store.NewSessionStore(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")
	result, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	lookup, err := sess.Lookup(ctx, store.HashSessionToken(result.Authenticated.SessionToken))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	want := time.Now().UTC().Add(SessionLifetime)
	delta := lookup.ExpiresAt.Sub(want)
	if delta < -5*time.Second || delta > 5*time.Second {
		t.Errorf("session ExpiresAt drift too large: got %v want ~%v (delta %v)", lookup.ExpiresAt, want, delta)
	}
}
