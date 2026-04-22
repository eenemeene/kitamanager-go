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
	if result.SessionToken == "" {
		t.Error("expected non-empty session token")
	}
	if result.CSRFToken == "" {
		t.Error("expected non-empty CSRF token")
	}
	if result.ExpiresIn != int64(SessionLifetime.Seconds()) {
		t.Errorf("ExpiresIn = %d, want %d", result.ExpiresIn, int64(SessionLifetime.Seconds()))
	}

	// A session row must exist on the server keyed by sha256 of the returned
	// raw token.
	sess := store.NewSessionStore(db)
	lookup, err := sess.Lookup(ctx, store.HashSessionToken(result.SessionToken))
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
	if result.SessionToken == "" {
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

	svc.Logout(ctx, loginResult.SessionToken)

	if _, err := sess.Lookup(ctx, store.HashSessionToken(loginResult.SessionToken)); err != store.ErrNotFound {
		t.Errorf("expected session to be deleted after logout, got %v", err)
	}
}

func TestAuthService_Logout_EmptyTokenIsNoOp(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	// Should not panic or error.
	svc.Logout(ctx, "")
}

func TestAuthService_Logout_UnknownTokenIsNoOp(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	// Should not panic or error.
	svc.Logout(ctx, "never-issued-this")
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

	lookup, err := sess.Lookup(ctx, store.HashSessionToken(result.SessionToken))
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	want := time.Now().UTC().Add(SessionLifetime)
	delta := lookup.ExpiresAt.Sub(want)
	if delta < -5*time.Second || delta > 5*time.Second {
		t.Errorf("session ExpiresAt drift too large: got %v want ~%v (delta %v)", lookup.ExpiresAt, want, delta)
	}
}
