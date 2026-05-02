package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
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

func TestAuthHandler_Login_SetsSessionAndCSRFCookies(t *testing.T) {
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

	cookies := w.Result().Cookies()
	cookieByName := make(map[string]*http.Cookie)
	for _, cookie := range cookies {
		cookieByName[cookie.Name] = cookie
	}
	sess, hasSession := cookieByName["session"]
	csrf, hasCSRF := cookieByName["csrf_token"]
	if !hasSession {
		t.Fatal("expected session cookie to be set")
	}
	if !hasCSRF {
		t.Fatal("expected csrf_token cookie to be set")
	}
	if !sess.HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}
	if csrf.HttpOnly {
		t.Error("csrf_token cookie must NOT be HttpOnly (frontend reads it)")
	}
	if sess.Value == "" {
		t.Error("session cookie value is empty")
	}

	// JSON body must not contain any session/token fields.
	bodyStr := w.Body.String()
	if containsStr(bodyStr, `"session"`) || containsStr(bodyStr, `"token"`) {
		t.Errorf("JSON body leaks token-like fields: %s", bodyStr)
	}
}

// containsStr is a tiny helper to avoid pulling in a test-only dependency.
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
		// Verify HttpOnly flag on session cookie
		if cookie.Name == "session" && !cookie.HttpOnly {
			t.Error("session cookie should be HttpOnly")
		}
		// Verify session path is "/"
		if cookie.Name == "session" && cookie.Path != "/" {
			t.Errorf("session cookie path should be '/', got '%s'", cookie.Path)
		}
		// CSRF token should NOT be HttpOnly
		if cookie.Name == "csrf_token" && cookie.HttpOnly {
			t.Error("csrf_token cookie should NOT be HttpOnly")
		}
		// Both cookies must use SameSite=Strict
		if cookie.Name == "session" || cookie.Name == "csrf_token" {
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("%s cookie should have SameSite=Strict, got %v", cookie.Name, cookie.SameSite)
			}
		}
	}

	if !cookieNames["session"] {
		t.Error("expected session cookie to be set")
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
		if cookie.Name == "session" || cookie.Name == "csrf_token" {
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
	if !cookieNames["session"] {
		t.Error("expected session cookie after password change")
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

// routerWithUserAndSession wires a minimal router that behaves like the
// real auth middleware chain for the purpose of testing /me/sessions:
// it sets ctxkeys.UserID and optionally ctxkeys.SessionIDHash before
// delegating to the handler.
func routerWithUserAndSession(userID uint, sessionHash string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, userID)
		c.Set(ctxkeys.UserEmail, "test@example.com")
		if sessionHash != "" {
			c.Set(ctxkeys.SessionIDHash, sessionHash)
		}
		c.Next()
	})
	return r
}

func TestAuthHandler_ListSessions_Empty(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db, "User", "u@example.com", "pw")
	handler := createAuthHandler(db)

	r := routerWithUserAndSession(user.ID, "")
	r.GET("/me/sessions", handler.ListSessions)

	w := performRequest(r, "GET", "/me/sessions", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var result models.UserSessionsResponse
	parseResponse(t, w, &result)
	if len(result.Sessions) != 0 {
		t.Errorf("expected empty list, got %d", len(result.Sessions))
	}
}

func TestAuthHandler_ListSessions_OnlyCallersSessions(t *testing.T) {
	// Critical security invariant: /me/sessions must NEVER include
	// sessions belonging to other users.
	db := setupTestDB(t)
	alice := createTestUser(t, db, "Alice", "alice@example.com", "pw")
	bob := createTestUser(t, db, "Bob", "bob@example.com", "pw")
	sessStore := store.NewSessionStore(db)
	ctx := context.Background()

	// Seed: two sessions for Alice, one for Bob.
	aliceHashes := make([]string, 0, 2)
	for range 2 {
		_, h, _ := store.GenerateSessionToken()
		if err := sessStore.Create(ctx, &models.Session{
			ID: h, UserID: alice.ID, CreatedAt: time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}); err != nil {
			t.Fatalf("seed alice: %v", err)
		}
		aliceHashes = append(aliceHashes, h)
	}
	_, bobH, _ := store.GenerateSessionToken()
	if err := sessStore.Create(ctx, &models.Session{
		ID: bobH, UserID: bob.ID, CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed bob: %v", err)
	}

	handler := createAuthHandler(db)

	// Alice requests /me/sessions — should see only her two, never Bob's.
	r := routerWithUserAndSession(alice.ID, aliceHashes[0])
	r.GET("/me/sessions", handler.ListSessions)
	w := performRequest(r, "GET", "/me/sessions", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var result models.UserSessionsResponse
	parseResponse(t, w, &result)
	if len(result.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(result.Sessions))
	}
	seen := make(map[string]bool)
	for _, s := range result.Sessions {
		if s.ID == bobH {
			t.Errorf("bob's session leaked into alice's list")
		}
		seen[s.ID] = true
	}
	// Exactly one row should be flagged as current.
	var currentCount int
	for _, s := range result.Sessions {
		if s.Current {
			currentCount++
			if s.ID != aliceHashes[0] {
				t.Errorf("wrong session flagged current: %q", s.ID)
			}
		}
	}
	if currentCount != 1 {
		t.Errorf("expected exactly one current=true row, got %d", currentCount)
	}
}

func TestAuthHandler_ListSessions_NotAuthenticated(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	// No middleware -> no UserID in context.
	r := gin.New()
	r.GET("/me/sessions", handler.ListSessions)

	w := performRequest(r, "GET", "/me/sessions", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_RevokeSession_Success(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db, "User", "u@example.com", "pw")
	sessStore := store.NewSessionStore(db)
	ctx := context.Background()

	_, h, _ := store.GenerateSessionToken()
	if err := sessStore.Create(ctx, &models.Session{
		ID: h, UserID: user.ID, CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	handler := createAuthHandler(db)
	r := routerWithUserAndSession(user.ID, "different-current-hash")
	r.DELETE("/me/sessions/:sessionId", handler.RevokeSession)

	w := performRequest(r, "DELETE", fmt.Sprintf("/me/sessions/%s", h), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204, body=%s", w.Code, w.Body.String())
	}
	if _, err := sessStore.Lookup(ctx, h); err != store.ErrNotFound {
		t.Errorf("session should be gone, got %v", err)
	}
}

func TestAuthHandler_RevokeSession_UnknownID_404(t *testing.T) {
	db := setupTestDB(t)
	user := createTestUser(t, db, "User", "u@example.com", "pw")
	handler := createAuthHandler(db)

	r := routerWithUserAndSession(user.ID, "")
	r.DELETE("/me/sessions/:sessionId", handler.RevokeSession)

	// Pass a fully-formed sha256-looking id but one that does not exist.
	fakeID := store.HashSessionToken("never-issued")
	w := performRequest(r, "DELETE", fmt.Sprintf("/me/sessions/%s", fakeID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAuthHandler_RevokeSession_CrossUser_404_NotLeaking(t *testing.T) {
	// Bob issues DELETE /me/sessions/<alice's id>. Must be 404 —
	// indistinguishable from an unknown id — and Alice's session must
	// survive.
	db := setupTestDB(t)
	alice := createTestUser(t, db, "Alice", "alice@example.com", "pw")
	bob := createTestUser(t, db, "Bob", "bob@example.com", "pw")
	sessStore := store.NewSessionStore(db)
	ctx := context.Background()

	_, aliceH, _ := store.GenerateSessionToken()
	if err := sessStore.Create(ctx, &models.Session{
		ID: aliceH, UserID: alice.ID, CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	handler := createAuthHandler(db)
	// Router acts as Bob.
	r := routerWithUserAndSession(bob.ID, "")
	r.DELETE("/me/sessions/:sessionId", handler.RevokeSession)

	w := performRequest(r, "DELETE", fmt.Sprintf("/me/sessions/%s", aliceH), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (not leaking session existence), got %d: %s", w.Code, w.Body.String())
	}
	if _, err := sessStore.Lookup(ctx, aliceH); err != nil {
		t.Errorf("alice's session was wrongly deleted: %v", err)
	}
}

func TestAuthHandler_RevokeSession_SelfRevoke(t *testing.T) {
	// The user revokes their own current session. Handler still returns
	// 204 — the browser's next request will 401 via middleware, which
	// is the mechanism that clears cookies. Nothing special happens
	// server-side in this endpoint.
	db := setupTestDB(t)
	user := createTestUser(t, db, "User", "u@example.com", "pw")
	sessStore := store.NewSessionStore(db)
	ctx := context.Background()

	_, h, _ := store.GenerateSessionToken()
	if err := sessStore.Create(ctx, &models.Session{
		ID: h, UserID: user.ID, CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	handler := createAuthHandler(db)
	// Router thinks the current session IS `h`.
	r := routerWithUserAndSession(user.ID, h)
	r.DELETE("/me/sessions/:sessionId", handler.RevokeSession)

	w := performRequest(r, "DELETE", fmt.Sprintf("/me/sessions/%s", h), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := sessStore.Lookup(ctx, h); err != store.ErrNotFound {
		t.Errorf("self-revoked session should be gone, got %v", err)
	}
}

func TestAuthHandler_RevokeSession_NotAuthenticated(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)

	r := gin.New()
	r.DELETE("/me/sessions/:sessionId", handler.RevokeSession)

	w := performRequest(r, "DELETE", "/me/sessions/anything", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
