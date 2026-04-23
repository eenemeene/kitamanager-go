package handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	cryptopkg "github.com/eenemeene/kitamanager-go/internal/crypto"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// userWithMFA creates a user with a hashed password AND an activated
// TOTP factor. Returns (user, factorID, base32Secret).
func userWithMFA(t *testing.T, db *gorm.DB, email, password string) (*models.User, uint, string) {
	t.Helper()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := createTestUser(t, db, email, email, string(hashedPassword))

	// Enrol + activate TOTP directly via FactorService.
	audit := createAuditService(db)
	keyBytes, _ := hex.DecodeString("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	aead, err := cryptopkg.NewAEAD(keyBytes)
	if err != nil {
		t.Fatalf("aead: %v", err)
	}
	factorSvc := service.NewFactorService(
		store.NewFactorStore(db),
		store.NewUserStore(db),
		aead,
		"KitaManager (test)",
		audit,
	)
	ctx := context.Background()
	enroll, err := factorSvc.EnrollTOTP(ctx, user.ID, nil, password, user.Email)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	payload := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(payload.Secret, time.Now().UTC())
	if _, err := factorSvc.ActivateFactor(ctx, user.ID, enroll.ID, code); err != nil {
		t.Fatalf("activate: %v", err)
	}
	return user, enroll.ID, payload.Secret
}

// --- Login ---

// Non-MFA user: response has status=authenticated and an expires_in.
func TestAuthHandler_Login_NonMFA_StatusField(t *testing.T) {
	db := setupTestDB(t)
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("pw-123456"), bcrypt.DefaultCost)
	createTestUser(t, db, "U", "u@example.com", string(hashedPassword))

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)

	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "u@example.com",
		Password: "pw-123456",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp models.LoginResponse
	parseResponse(t, w, &resp)
	if resp.Status != models.LoginStatusAuthenticated {
		t.Errorf("status = %q, want authenticated", resp.Status)
	}
	if resp.ExpiresIn <= 0 {
		t.Errorf("expires_in = %d, want >0", resp.ExpiresIn)
	}
	// Session cookie set.
	if len(w.Result().Cookies()) == 0 {
		t.Error("expected session cookie")
	}
}

// MFA user: response is mfa_required, no session cookie is set.
func TestAuthHandler_Login_MFA_ReturnsPendingAndNoCookie(t *testing.T) {
	db := setupTestDB(t)
	_, _, _ = userWithMFA(t, db, "u@example.com", "pw-123456")

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)

	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "u@example.com",
		Password: "pw-123456",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp models.LoginMFARequiredResponse
	parseResponse(t, w, &resp)
	if resp.Status != models.LoginStatusMFARequired {
		t.Errorf("status = %q, want mfa_required", resp.Status)
	}
	if resp.PendingToken == "" {
		t.Error("pending_token missing")
	}
	if resp.ExpiresAt == "" {
		t.Error("expires_at missing")
	}
	if len(resp.Factors) == 0 {
		t.Error("factors empty")
	}
	// No session cookie.
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			t.Errorf("session cookie was set on MFA response: %q", c.Value)
		}
	}
}

// MFA factor descriptors in the response never expose forbidden
// fields (last_used_at, created_at, backup_codes_remaining) — we
// assert the JSON keys strictly.
func TestAuthHandler_Login_MFA_NoPostLoginMetadataLeak(t *testing.T) {
	db := setupTestDB(t)
	_, _, _ = userWithMFA(t, db, "u@example.com", "pw-123456")

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)
	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "u@example.com",
		Password: "pw-123456",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	// Decode the factors as map[string]any so we can see all keys.
	var raw struct {
		Factors []map[string]any `json:"factors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	allowed := map[string]bool{"id": true, "type": true, "label": true}
	for _, f := range raw.Factors {
		for k := range f {
			if !allowed[k] {
				t.Errorf("forbidden field leaked on login MFA descriptor: %q", k)
			}
		}
	}
}

// --- MFAVerify ---

func TestAuthHandler_MFAVerify_Success(t *testing.T) {
	db := setupTestDB(t)
	_, factorID, secret := userWithMFA(t, db, "u@example.com", "pw-123456")

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/auth/mfa/verify", handler.MFAVerify)

	// Step 1: password.
	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "u@example.com",
		Password: "pw-123456",
	})
	var step1 models.LoginMFARequiredResponse
	parseResponse(t, w, &step1)
	if step1.PendingToken == "" {
		t.Fatal("no pending_token")
	}

	// Step 2: code.
	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	w = performRequest(r, "POST", "/auth/mfa/verify", models.MFAVerifyRequest{
		PendingToken: step1.PendingToken,
		FactorID:     factorID,
		Code:         code,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", w.Code, w.Body.String())
	}
	var resp models.LoginResponse
	parseResponse(t, w, &resp)
	if resp.Status != models.LoginStatusAuthenticated {
		t.Errorf("status = %q, want authenticated", resp.Status)
	}
	// Session cookie present.
	hasSess := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			hasSess = true
		}
	}
	if !hasSess {
		t.Error("session cookie not set after verify")
	}
}

func TestAuthHandler_MFAVerify_BadRequest(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/auth/mfa/verify", handler.MFAVerify)
	// Missing required fields.
	w := performRequest(r, "POST", "/auth/mfa/verify", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_MFAVerify_UnknownPending_401(t *testing.T) {
	db := setupTestDB(t)
	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/auth/mfa/verify", handler.MFAVerify)
	w := performRequest(r, "POST", "/auth/mfa/verify", models.MFAVerifyRequest{
		PendingToken: "no-such-token",
		FactorID:     1,
		Code:         "123456",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_MFAVerify_WrongCode_401(t *testing.T) {
	db := setupTestDB(t)
	_, factorID, _ := userWithMFA(t, db, "u@example.com", "pw-123456")

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/auth/mfa/verify", handler.MFAVerify)

	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "u@example.com",
		Password: "pw-123456",
	})
	var step1 models.LoginMFARequiredResponse
	parseResponse(t, w, &step1)

	w = performRequest(r, "POST", "/auth/mfa/verify", models.MFAVerifyRequest{
		PendingToken: step1.PendingToken,
		FactorID:     factorID,
		Code:         "000000",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_MFAVerify_FiveWrongCodes_429(t *testing.T) {
	db := setupTestDB(t)
	_, factorID, _ := userWithMFA(t, db, "u@example.com", "pw-123456")

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/auth/mfa/verify", handler.MFAVerify)

	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "u@example.com",
		Password: "pw-123456",
	})
	var step1 models.LoginMFARequiredResponse
	parseResponse(t, w, &step1)

	// First (limit-1) wrong codes are 401.
	for i := 1; i < service.MFAChallengeFailureLimit; i++ {
		w = performRequest(r, "POST", "/auth/mfa/verify", models.MFAVerifyRequest{
			PendingToken: step1.PendingToken,
			FactorID:     factorID,
			Code:         "000000",
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i, w.Code)
		}
	}
	// Nth: 429.
	w = performRequest(r, "POST", "/auth/mfa/verify", models.MFAVerifyRequest{
		PendingToken: step1.PendingToken,
		FactorID:     factorID,
		Code:         "000000",
	})
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("limit attempt: expected 429, got %d", w.Code)
	}
}

// A regular session cookie sent as pending_token must be rejected.
// This is the cross-material-type isolation guarantee.
func TestAuthHandler_MFAVerify_RegularSessionAsPending_401(t *testing.T) {
	db := setupTestDB(t)
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("pw-123456"), bcrypt.DefaultCost)
	createTestUser(t, db, "U", "u@example.com", string(hashedPassword))

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/auth/mfa/verify", handler.MFAVerify)

	// Non-MFA user login → real session cookie issued.
	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "u@example.com",
		Password: "pw-123456",
	})
	var sessionCookie string
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c.Value
		}
	}
	if sessionCookie == "" {
		t.Fatal("no session cookie after non-MFA login")
	}

	// Submit the regular session cookie value as pending_token.
	w = performRequest(r, "POST", "/auth/mfa/verify", models.MFAVerifyRequest{
		PendingToken: sessionCookie,
		FactorID:     1,
		Code:         "123456",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("regular session as pending: expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

// Cross-user factor id: Bob's factor against Alice's pending → 401.
func TestAuthHandler_MFAVerify_CrossUserFactorID(t *testing.T) {
	db := setupTestDB(t)
	_, _, _ = userWithMFA(t, db, "alice@example.com", "pw-alice01")
	_, bobFactorID, bobSecret := userWithMFA(t, db, "bob@example.com", "pw-bob00001")

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)
	r.POST("/auth/mfa/verify", handler.MFAVerify)

	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "alice@example.com",
		Password: "pw-alice01",
	})
	var step1 models.LoginMFARequiredResponse
	parseResponse(t, w, &step1)

	code, _ := totp.GenerateCode(bobSecret, time.Now().UTC())
	w = performRequest(r, "POST", "/auth/mfa/verify", models.MFAVerifyRequest{
		PendingToken: step1.PendingToken,
		FactorID:     bobFactorID,
		Code:         code,
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("cross-user factor id: expected 401, got %d", w.Code)
	}
}

// expires_at on the response is a parseable, non-past timestamp.
func TestAuthHandler_Login_MFA_ExpiresAtParses(t *testing.T) {
	db := setupTestDB(t)
	_, _, _ = userWithMFA(t, db, "u@example.com", "pw-123456")

	handler := createAuthHandler(db)
	r := gin.New()
	r.POST("/login", handler.Login)
	w := performRequest(r, "POST", "/login", models.LoginRequest{
		Email:    "u@example.com",
		Password: "pw-123456",
	})
	var step1 models.LoginMFARequiredResponse
	parseResponse(t, w, &step1)

	parsed, err := time.Parse(http.TimeFormat, step1.ExpiresAt)
	if err != nil {
		t.Fatalf("expires_at unparseable: %v (%q)", err, step1.ExpiresAt)
	}
	if parsed.Before(time.Now().UTC()) {
		t.Errorf("expires_at is in the past: %v", parsed)
	}
}

// Route-level guard: silence a stray unused import while keeping the
// shape of the test file discoverable.
var _ = fmt.Sprintf
