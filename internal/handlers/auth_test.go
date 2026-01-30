package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestAuthHandler_Login_Success(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)

	// Create user with hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

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

	if result.Token == "" {
		t.Error("expected token to be set")
	}
}

func TestAuthHandler_Login_InvalidEmail(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

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
	userStore := store.NewUserStore(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

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
	userStore := store.NewUserStore(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	// Deactivate user
	user.Active = false
	db.Save(user)

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

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
	userStore := store.NewUserStore(db)
	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

	r := gin.New()
	r.POST("/login", handler.Login)

	// Missing required fields
	body := map[string]interface{}{
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

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

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
	updatedUser, err := userStore.FindByID(user.ID)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	if updatedUser.LastLogin == nil {
		t.Error("expected last_login to be set after login")
	}
}

func TestAuthHandler_Login_ReturnsRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

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

	if result.Token == "" {
		t.Error("expected access token to be set")
	}
	if result.RefreshToken == "" {
		t.Error("expected refresh token to be set")
	}
	if result.ExpiresIn <= 0 {
		t.Error("expected expires_in to be positive")
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/refresh", handler.Refresh)

	// First login to get tokens
	loginBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginResp := performRequest(r, "POST", "/login", loginBody)

	var loginResult models.LoginResponse
	parseResponse(t, loginResp, &loginResult)

	// Use refresh token to get new tokens
	refreshBody := models.RefreshRequest{
		RefreshToken: loginResult.RefreshToken,
	}
	w := performRequest(r, "POST", "/refresh", refreshBody)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result models.LoginResponse
	parseResponse(t, w, &result)

	if result.Token == "" {
		t.Error("expected new access token to be set")
	}
	if result.RefreshToken == "" {
		t.Error("expected new refresh token to be set")
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

	r := gin.New()
	r.POST("/refresh", handler.Refresh)

	body := models.RefreshRequest{
		RefreshToken: "invalid-token",
	}
	w := performRequest(r, "POST", "/refresh", body)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Refresh_WithAccessToken(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/refresh", handler.Refresh)

	// Login to get tokens
	loginBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginResp := performRequest(r, "POST", "/login", loginBody)

	var loginResult models.LoginResponse
	parseResponse(t, loginResp, &loginResult)

	// Try to use ACCESS token for refresh (should fail)
	refreshBody := models.RefreshRequest{
		RefreshToken: loginResult.Token, // Using access token instead of refresh token
	}
	w := performRequest(r, "POST", "/refresh", refreshBody)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d when using access token for refresh, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Refresh_InactiveUser(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/refresh", handler.Refresh)

	// Login to get tokens
	loginBody := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginResp := performRequest(r, "POST", "/login", loginBody)

	var loginResult models.LoginResponse
	parseResponse(t, loginResp, &loginResult)

	// Deactivate the user
	user.Active = false
	db.Save(user)

	// Try to refresh (should fail because user is now inactive)
	refreshBody := models.RefreshRequest{
		RefreshToken: loginResult.RefreshToken,
	}
	w := performRequest(r, "POST", "/refresh", refreshBody)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for inactive user, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthHandler_Refresh_MissingToken(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)
	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

	r := gin.New()
	r.POST("/refresh", handler.Refresh)

	// Missing refresh_token field
	body := map[string]interface{}{}
	w := performRequest(r, "POST", "/refresh", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAuthHandler_Login_SetsCookies(t *testing.T) {
	db := setupTestDB(t)
	userStore := store.NewUserStore(db)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	createTestUser(t, db, "Test User", "test@example.com", string(hashedPassword))

	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

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

	// Check that cookies are set
	cookies := w.Result().Cookies()
	cookieNames := make(map[string]bool)
	for _, cookie := range cookies {
		cookieNames[cookie.Name] = true
		// Verify HttpOnly flag on access_token
		if cookie.Name == "access_token" && !cookie.HttpOnly {
			t.Error("access_token cookie should be HttpOnly")
		}
		// CSRF token should NOT be HttpOnly
		if cookie.Name == "csrf_token" && cookie.HttpOnly {
			t.Error("csrf_token cookie should NOT be HttpOnly")
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
	userStore := store.NewUserStore(db)
	handler := NewAuthHandler(userStore, "test-jwt-secret", nil)

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
