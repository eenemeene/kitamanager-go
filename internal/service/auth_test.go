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

func TestAuthService_Login_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	result, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if result.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if result.CSRFToken == "" {
		t.Error("expected non-empty CSRF token")
	}
	if result.ExpiresIn != int64(AccessTokenExpiry.Seconds()) {
		t.Errorf("ExpiresIn = %d, want %d", result.ExpiresIn, int64(AccessTokenExpiry.Seconds()))
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

	// Next attempt should be locked out even with correct password
	_, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrTooManyRequests) {
		t.Errorf("expected ErrTooManyRequests, got %v", err)
	}
}

func TestAuthService_Refresh_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	loginResult, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	refreshResult, err := svc.Refresh(ctx, loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if refreshResult.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if refreshResult.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	// New tokens should differ from the old ones
	if refreshResult.AccessToken == loginResult.AccessToken {
		t.Error("expected different access token after refresh")
	}
}

func TestAuthService_Refresh_ReplayDetection(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	loginResult, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// First refresh should succeed (and revoke the old token)
	_, err = svc.Refresh(ctx, loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}

	// Second refresh with the same token should fail (replay detection)
	_, err = svc.Refresh(ctx, loginResult.RefreshToken)
	if err == nil {
		t.Fatal("expected error on token replay, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Refresh_InactiveUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	loginResult, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Deactivate the user after login
	db.Model(user).Update("active", false)

	_, err = svc.Refresh(ctx, loginResult.RefreshToken)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	_, err := svc.Refresh(ctx, "not-a-valid-jwt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Logout(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	loginResult, err := svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Logout should not panic or error
	svc.Logout(ctx, loginResult.AccessToken, loginResult.RefreshToken)

	// Refresh with the revoked token should fail
	_, err = svc.Refresh(ctx, loginResult.RefreshToken)
	if err == nil {
		t.Fatal("expected error after logout, got nil")
	}
}

func TestAuthService_ChangePassword_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	result, err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456", "127.0.0.1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected non-empty access token after password change")
	}

	// Should be able to login with new password
	_, err = svc.Login(ctx, "test@example.com", "newpassword456", "127.0.0.1", "test-agent")
	if err != nil {
		t.Errorf("login with new password failed: %v", err)
	}

	// Old password should no longer work
	_, err = svc.Login(ctx, "test@example.com", "password123", "127.0.0.1", "test-agent")
	if err == nil {
		t.Error("expected login with old password to fail")
	}
}

func TestAuthService_ChangePassword_WrongCurrentPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "test@example.com", "password123")

	_, err := svc.ChangePassword(ctx, user.ID, "wrongpassword", "newpassword456", "127.0.0.1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
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

func TestAuthService_Login_LockoutWindowExpiry(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "lockout-expiry@example.com", "password123")

	// Insert failed login audit entries with timestamps OUTSIDE the lockout window
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

	// Login with correct credentials should succeed because failures are outside the 15-min window
	result, err := svc.Login(ctx, "lockout-expiry@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected login to succeed after lockout window expiry, got %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestAuthService_Refresh_WithAccessTokenInsteadOfRefresh(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	createTestUserWithHashedPassword(t, db, "Test User", "access-refresh@example.com", "password123")

	loginResult, err := svc.Login(ctx, "access-refresh@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Attempt to refresh using the access token instead of the refresh token
	_, err = svc.Refresh(ctx, loginResult.AccessToken)
	if err == nil {
		t.Fatal("expected error when using access token for refresh, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_ChangePassword_TokensRevokedAfterChange(t *testing.T) {
	db := setupTestDB(t)
	svc := createAuthService(db)
	ctx := context.Background()

	user := createTestUserWithHashedPassword(t, db, "Test User", "change-pw@example.com", "password123")

	// Login to get tokens
	loginResult, err := svc.Login(ctx, "change-pw@example.com", "password123", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Change password successfully
	_, err = svc.ChangePassword(ctx, user.ID, "password123", "newpassword456", "127.0.0.1")
	if err != nil {
		t.Fatalf("change password failed: %v", err)
	}

	// Attempt to refresh with the OLD refresh token should fail
	_, err = svc.Refresh(ctx, loginResult.RefreshToken)
	if err == nil {
		t.Fatal("expected error when refreshing with old token after password change, got nil")
	}
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
