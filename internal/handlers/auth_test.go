package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestAuthHandler_Login_Success(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)

	body := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	w := performRequest(r, "POST", "/login", body)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.LoginResponse
	parseResponse(t, w, &result)

	if result.ExpiresIn <= 0 {
		t.Error("expected expires_in to be positive")
	}
}

func TestAuthHandler_Login_InvalidEmail(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)

	body := models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	w := performRequest(r, "POST", "/login", body)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Login_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)

	body := models.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	w := performRequest(r, "POST", "/login", body)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Login_InactiveUser(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	// Deactivate user
	user.Active = false
	db.Save(user)

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)

	body := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	w := performRequest(r, "POST", "/login", body)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Login_BadRequest(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)

	// Missing required fields
	body := map[string]any{
		"email": "test@example.com",
		// missing password
	}

	w := performRequest(r, "POST", "/login", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuthHandler_Login_UpdatesLastLogin(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	// Verify last_login is nil initially
	if user.LastLogin != nil {
		t.Error("expected last_login to be nil initially")
	}

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)

	body := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	w := performRequest(r, "POST", "/login", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify last_login was updated
	updatedUser, err := userStore.FindByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	if updatedUser.LastLogin == nil {
		t.Error("expected last_login to be set after login")
	}
}

func TestAuthHandler_Login_TokensOnlyInCookies(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)

	body := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	w := performRequest(r, "POST", "/login", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.LoginResponse
	parseResponse(t, w, &result)

	if result.ExpiresIn <= 0 {
		t.Error("expected expires_in to be positive")
	}

	// Verify tokens are in cookies, not in JSON body
	cookies := w.Result().Cookies()
	cookieNames := make(map[string]bool)
	for _, cookie := range cookies {
		cookieNames[cookie.Name] = true
	}
	if !cookieNames["access_token"] {
		t.Error("expected access_token cookie to be set")
	}
	if !cookieNames["refresh_token"] {
		t.Error("expected refresh_token cookie to be set")
	}

	// Verify JSON body does NOT contain token fields
	bodyStr := w.Body.String()
	if contains(bodyStr, `"token"`) {
		t.Error("JSON body should not contain token field")
	}
	if contains(bodyStr, `"refresh_token"`) {
		t.Error("JSON body should not contain refresh_token field")
	}
}

// contains checks if s contains substr (simple helper for test assertions).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/refresh", handler.Refresh)

	// First login to get cookies
	loginBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginResp := performRequest(r, "POST", "/login", loginBody)

	// Use cookies from login for refresh request
	w := performRequestWithCookies(r, "POST", "/refresh", nil, loginResp.Result().Cookies())

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.RefreshResponse
	parseResponse(t, w, &result)

	if result.ExpiresIn <= 0 {
		t.Error("expected expires_in to be positive")
	}

	// Verify new tokens are set in cookies
	cookies := w.Result().Cookies()
	cookieNames := make(map[string]bool)
	for _, cookie := range cookies {
		cookieNames[cookie.Name] = true
	}
	if !cookieNames["access_token"] {
		t.Error("expected new access_token cookie to be set")
	}
	if !cookieNames["refresh_token"] {
		t.Error("expected new refresh_token cookie to be set")
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/refresh", handler.Refresh)

	// Set an invalid refresh_token cookie
	cookies := []*http.Cookie{
		{Name: "refresh_token", Value: "invalid-token"},
	}
	w := performRequestWithCookies(r, "POST", "/refresh", nil, cookies)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Refresh_WithAccessToken(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/refresh", handler.Refresh)

	// Login to get tokens
	loginBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginResp := performRequest(r, "POST", "/login", loginBody)

	// Extract the access_token cookie value and use it as refresh_token cookie
	var accessTokenValue string
	for _, cookie := range loginResp.Result().Cookies() {
		if cookie.Name == "access_token" {
			accessTokenValue = cookie.Value
			break
		}
	}

	// Try to use ACCESS token as refresh token cookie (should fail)
	cookies := []*http.Cookie{
		{Name: "refresh_token", Value: accessTokenValue},
	}
	w := performRequestWithCookies(r, "POST", "/refresh", nil, cookies)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d when using access token for refresh, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Refresh_InactiveUser(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/refresh", handler.Refresh)

	// Login to get cookies
	loginBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginResp := performRequest(r, "POST", "/login", loginBody)

	// Deactivate the user
	user.Active = false
	db.Save(user)

	// Try to refresh (should fail because user is now inactive)
	w := performRequestWithCookies(r, "POST", "/refresh", nil, loginResp.Result().Cookies())

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for inactive user, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Refresh_MissingToken(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/refresh", handler.Refresh)

	// No cookies set at all
	w := performRequest(r, "POST", "/refresh", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Login_SetsCookies(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/login", handler.Login)

	body := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	w := performRequest(r, "POST", "/login", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Check that cookies are set with correct attributes
	cookies := w.Result().Cookies()
	cookieNames := make(map[string]bool)
	for _, cookie := range cookies {
		cookieNames[cookie.Name] = true
		// Verify HttpOnly flag on access_token
		if cookie.Name == "access_token" && !cookie.HttpOnly {
			t.Error("access_token cookie should be HttpOnly")
		}
		// Verify access_token path is "/"
		if cookie.Name == "access_token" && cookie.Path != "/" {
			t.Errorf("access_token cookie path should be '/', got '%s'", cookie.Path)
		}
		// Verify refresh_token is scoped to refresh endpoint
		if cookie.Name == "refresh_token" && cookie.Path != "/api/v1/refresh" {
			t.Errorf("refresh_token cookie path should be '/api/v1/refresh', got '%s'", cookie.Path)
		}
		// CSRF token should NOT be HttpOnly
		if cookie.Name == "csrf_token" && cookie.HttpOnly {
			t.Error("csrf_token cookie should NOT be HttpOnly")
		}
		// All auth cookies must use SameSite=Strict
		if cookie.Name == "access_token" || cookie.Name == "refresh_token" || cookie.Name == "csrf_token" {
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("%s cookie should have SameSite=Strict, got %v", cookie.Name, cookie.SameSite)
			}
		}
	}

	if !cookieNames["access_token"] {
		t.Error("expected access_token cookie to be set")
	}
	if !cookieNames["refresh_token"] {
		t.Error("expected refresh_token cookie to be set")
	}
	if !cookieNames["csrf_token"] {
		t.Error("expected csrf_token cookie to be set")
	}
}

func TestAuthHandler_Logout_ClearsCookies(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	r := gin.New()
	r.POST("/logout", handler.Logout)

	w := performRequest(r, "POST", "/logout", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check that cookies are cleared (MaxAge <= 0)
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "access_token" || cookie.Name == "refresh_token" || cookie.Name == "csrf_token" {
			if cookie.MaxAge > 0 {
				t.Errorf("%s cookie should have MaxAge <= 0 to clear it, got %d", cookie.Name, cookie.MaxAge)
			}
		}
	}

	// Check response message
	var result models.MessageResponse
	parseResponse(t, w, &result)

	if result.Message == "" {
		t.Error("expected message in logout response")
	}
}

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := setupTestRouterWithUser(user.ID)
	r.PUT("/me/password", handler.ChangePassword)

	body := models.UserPasswordChangeRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword456",
	}

	w := performRequest(r, "PUT", "/me/password", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.LoginResponse
	parseResponse(t, w, &result)

	if result.ExpiresIn <= 0 {
		t.Error("expected expires_in to be positive")
	}

	// Verify new auth cookies are set
	cookies := w.Result().Cookies()
	cookieNames := make(map[string]bool)
	for _, cookie := range cookies {
		cookieNames[cookie.Name] = true
	}
	if !cookieNames["access_token"] {
		t.Error("expected access_token cookie after password change")
	}
	if !cookieNames["refresh_token"] {
		t.Error("expected refresh_token cookie after password change")
	}
	if !cookieNames["csrf_token"] {
		t.Error("expected csrf_token cookie after password change")
	}
}

func TestAuthHandler_ChangePassword_WrongCurrentPassword(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := setupTestRouterWithUser(user.ID)
	r.PUT("/me/password", handler.ChangePassword)

	body := models.UserPasswordChangeRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword456",
	}

	w := performRequest(r, "PUT", "/me/password", body)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestAuthHandler_ChangePassword_NotAuthenticated(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	// No auth middleware — getUserID returns 0
	r := gin.New()
	r.PUT("/me/password", handler.ChangePassword)

	body := models.UserPasswordChangeRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword456",
	}

	w := performRequest(r, "PUT", "/me/password", body)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestAuthHandler_ChangePassword_BadRequest_MissingFields(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := setupTestRouterWithUser(user.ID)
	r.PUT("/me/password", handler.ChangePassword)

	// Missing current_password
	w := performRequest(r, "PUT", "/me/password", map[string]string{
		"new_password": "newpassword456",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing current_password: expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Missing new_password
	w = performRequest(r, "PUT", "/me/password", map[string]string{
		"current_password": "password123",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing new_password: expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuthHandler_ChangePassword_NewPasswordTooShort(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := setupTestRouterWithUser(user.ID)
	r.PUT("/me/password", handler.ChangePassword)

	body := models.UserPasswordChangeRequest{
		CurrentPassword: "password123",
		NewPassword:     "short",
	}

	w := performRequest(r, "PUT", "/me/password", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for short password, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestAuthHandler_ChangePassword_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	// Set userID=99999 which doesn't exist
	r := setupTestRouterWithUser(99999)
	r.PUT("/me/password", handler.ChangePassword)

	body := models.UserPasswordChangeRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword456",
	}

	w := performRequest(r, "PUT", "/me/password", body)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestAuthHandler_ChangePassword_VerifyOldPasswordInvalidated(t *testing.T) {
	db := setupTestDB(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := createAuthHandler(db)

	r := setupTestRouterWithUser(user.ID)
	r.PUT("/me/password", handler.ChangePassword)
	r.POST("/login", handler.Login)

	// Change password
	body := models.UserPasswordChangeRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword456",
	}
	w := performRequest(r, "PUT", "/me/password", body)
	if w.Code != http.StatusOK {
		t.Fatalf("password change failed: %d: %s", w.Code, w.Body.String())
	}

	// Old password should no longer work for login
	loginBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	w = performRequest(r, "POST", "/login", loginBody)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected old password login to fail with %d, got %d", http.StatusUnauthorized, w.Code)
	}

	// New password should work
	loginBody.Password = "newpassword456"
	w = performRequest(r, "POST", "/login", loginBody)
	if w.Code != http.StatusOK {
		t.Errorf("expected new password login to succeed, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Me(t *testing.T) {
	db := setupTestDB(t)

	// Create user with ID matching setupTestRouter's userID (1)
	createTestUser(t, db, "Test User", "test@example.com", "password")

	handler := createAuthHandler(db)

	r := setupTestRouter()
	r.GET("/me", handler.Me)

	w := performRequest(r, "GET", "/me", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.UserResponse
	parseResponse(t, w, &result)

	if result.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", result.Name)
	}
	if result.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", result.Email)
	}
}

func TestAuthHandler_Me_UserNotFound(t *testing.T) {
	db := setupTestDB(t)

	// Don't create any user - setupTestRouter sets userID=1 which won't exist
	handler := createAuthHandler(db)

	r := setupTestRouter()
	r.GET("/me", handler.Me)

	w := performRequest(r, "GET", "/me", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestAuthHandler_Me_NotAuthenticated(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	// Use gin.New() without auth middleware - no userID in context
	r := gin.New()
	r.GET("/me", handler.Me)

	w := performRequest(r, "GET", "/me", nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}
