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

	handler := NewAuthHandler(userStore, "test-jwt-secret")

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
	handler := NewAuthHandler(userStore, "test-jwt-secret")

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

	handler := NewAuthHandler(userStore, "test-jwt-secret")

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

	handler := NewAuthHandler(userStore, "test-jwt-secret")

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
	handler := NewAuthHandler(userStore, "test-jwt-secret")

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

	handler := NewAuthHandler(userStore, "test-jwt-secret")

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
