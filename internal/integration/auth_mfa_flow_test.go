//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// seedUserWithTOTP creates a regular user (not superadmin) with a real
// hashed password AND an activated TOTP factor. Returns the userID,
// email, password, factorID, and the base32 TOTP secret the test can
// use to generate codes on demand. Unlike seedSuperadmin, this helper
// goes directly through FactorService for enrol + activate.
func seedUserWithTOTP(t *testing.T, email, password string) (userID uint, factorID uint, secret string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := &models.User{
		Name:     email,
		Email:    email,
		Password: string(hash),
		Active:   true,
	}
	if err := testDB.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	auditStore := store.NewAuditStore(testDB)
	audit := service.NewAuditService(auditStore)
	fstore := store.NewFactorStore(testDB)
	factorSvc := service.NewFactorService(fstore, store.NewUserStore(testDB), testTOTPAEAD(t), "KitaManager (test)", nil, audit)

	ctx := context.Background()
	enroll, err := factorSvc.EnrollTOTP(ctx, u.ID, nil, password, u.Email)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	payload := enroll.Enrollment.(models.TOTPEnrollmentPayload)
	code, _ := totp.GenerateCode(payload.Secret, time.Now().UTC())
	if _, err := factorSvc.ActivateFactor(ctx, u.ID, enroll.ID, code); err != nil {
		t.Fatalf("activate: %v", err)
	}
	return u.ID, enroll.ID, payload.Secret
}

// doTwoStepLogin runs the full /login + /auth/mfa/verify handshake
// and returns the final session cookie. The test asserts on each
// step's shape along the way.
func doTwoStepLogin(t *testing.T, r *gin.Engine, email, password string, factorID uint, code string) (finalCookie string) {
	t.Helper()

	// Step 1: password.
	body, _ := json.Marshal(models.LoginRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step1 status=%d body=%s", w.Code, w.Body.String())
	}
	var step1 models.LoginMFARequiredResponse
	if err := json.Unmarshal(w.Body.Bytes(), &step1); err != nil {
		t.Fatalf("decode step1: %v", err)
	}
	if step1.Status != models.LoginStatusMFARequired {
		t.Fatalf("step1 status = %q, want mfa_required", step1.Status)
	}
	// Confirm NO session cookie was set on step 1.
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			t.Fatalf("step1 unexpectedly set session cookie: %q", c.Value)
		}
	}

	// Step 2: pending_token + code.
	body, _ = json.Marshal(models.MFAVerifyRequest{
		PendingToken: step1.PendingToken,
		FactorID:     factorID,
		Code:         code,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("step2 status=%d body=%s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			return c.Value
		}
	}
	t.Fatal("step2: no session cookie")
	return ""
}

// Full end-to-end: password → MFA code → authenticated API call.
func TestAuthFlow_TwoStepLogin_HappyPath(t *testing.T) {
	cleanupDatabase()
	flow := setupAuthFlowRouter(t)

	userID, factorID, secret := seedUserWithTOTP(t, "mfa-user@test.local", "correct-pw")

	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	cookie := doTwoStepLogin(t, flow.router, "mfa-user@test.local", "correct-pw", factorID, code)

	// Cookie works on an authenticated endpoint.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: cookie})
	w := httptest.NewRecorder()
	flow.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/me with session cookie: status=%d body=%s", w.Code, w.Body.String())
	}

	// Pending row has been consumed; exactly 1 regular session for
	// the user.
	var regular, pending int64
	_ = testDB.Model(&models.Session{}).Where("user_id = ? AND kind = ?", userID, models.SessionKindRegular).Count(&regular).Error
	_ = testDB.Model(&models.Session{}).Where("user_id = ? AND kind = ?", userID, models.SessionKindPendingMFA).Count(&pending).Error
	if regular != 1 {
		t.Errorf("regular sessions = %d, want 1", regular)
	}
	if pending != 0 {
		t.Errorf("pending sessions = %d, want 0", pending)
	}

	// The new regular session has the right kind column at DB level.
	var row models.Session
	_ = testDB.Where("id = ?", store.HashSessionToken(cookie)).First(&row).Error
	if row.Kind != models.SessionKindRegular {
		t.Errorf("new session kind = %q, want regular", row.Kind)
	}
}

// Non-MFA user: /login continues to set the session cookie directly
// (backward compatibility with users who haven't enrolled a factor).
func TestAuthFlow_TwoStepLogin_NonMFAUserUnchanged(t *testing.T) {
	cleanupDatabase()
	flow := setupAuthFlowRouter(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pw-123456"), bcrypt.DefaultCost)
	u := &models.User{Name: "N", Email: "nomfa@test.local", Password: string(hash), Active: true}
	if err := testDB.Create(u).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	body, _ := json.Marshal(models.LoginRequest{Email: u.Email, Password: "pw-123456"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	flow.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var resp models.LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != models.LoginStatusAuthenticated {
		t.Errorf("status = %q, want authenticated", resp.Status)
	}
	// Cookie set.
	hasSess := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			hasSess = true
		}
	}
	if !hasSess {
		t.Error("no session cookie on non-MFA login")
	}
}

// Pending token expires → /verify returns 401.
func TestAuthFlow_TwoStepLogin_PendingExpired(t *testing.T) {
	cleanupDatabase()
	flow := setupAuthFlowRouter(t)

	_, factorID, secret := seedUserWithTOTP(t, "expire@test.local", "correct-pw")

	// Step 1.
	body, _ := json.Marshal(models.LoginRequest{Email: "expire@test.local", Password: "correct-pw"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	flow.router.ServeHTTP(w, req)
	var step1 models.LoginMFARequiredResponse
	_ = json.Unmarshal(w.Body.Bytes(), &step1)

	// Simulate expiry by rewriting the pending row's expires_at back.
	if err := testDB.Model(&models.Session{}).
		Where("id = ?", store.HashSessionToken(step1.PendingToken)).
		Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire: %v", err)
	}

	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	body, _ = json.Marshal(models.MFAVerifyRequest{
		PendingToken: step1.PendingToken, FactorID: factorID, Code: code,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	flow.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expired pending: expected 401, got %d", w.Code)
	}
}

// Pending token cannot be used in place of a session cookie against
// protected endpoints. Critical security property.
func TestAuthFlow_TwoStepLogin_PendingCannotAccessProtectedEndpoints(t *testing.T) {
	cleanupDatabase()
	flow := setupAuthFlowRouter(t)

	_, _, _ = seedUserWithTOTP(t, "mfa-user@test.local", "correct-pw")

	body, _ := json.Marshal(models.LoginRequest{Email: "mfa-user@test.local", Password: "correct-pw"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	flow.router.ServeHTTP(w, req)
	var step1 models.LoginMFARequiredResponse
	_ = json.Unmarshal(w.Body.Bytes(), &step1)

	// Try using pending_token as a session cookie.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: step1.PendingToken})
	w = httptest.NewRecorder()
	flow.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("pending-as-cookie: expected 401, got %d", w.Code)
	}

	// And as Authorization Bearer.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+step1.PendingToken)
	w = httptest.NewRecorder()
	flow.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("pending-as-bearer: expected 401, got %d", w.Code)
	}
}

// Wrong code 5 times → pending row destroyed, 429 returned, the
// next attempt returns 401 (unknown).
func TestAuthFlow_TwoStepLogin_PendingDestroyedAfterLimit(t *testing.T) {
	cleanupDatabase()
	flow := setupAuthFlowRouter(t)

	_, factorID, _ := seedUserWithTOTP(t, "bf@test.local", "correct-pw")

	// Step 1.
	body, _ := json.Marshal(models.LoginRequest{Email: "bf@test.local", Password: "correct-pw"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	flow.router.ServeHTTP(w, req)
	var step1 models.LoginMFARequiredResponse
	_ = json.Unmarshal(w.Body.Bytes(), &step1)

	verify := func(code string) int {
		body, _ := json.Marshal(models.MFAVerifyRequest{
			PendingToken: step1.PendingToken, FactorID: factorID, Code: code,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		flow.router.ServeHTTP(w, req)
		return w.Code
	}

	for i := 1; i < service.MFAChallengeFailureLimit; i++ {
		if got := verify("000000"); got != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", i, got)
		}
	}
	if got := verify("000000"); got != http.StatusTooManyRequests {
		t.Errorf("limit attempt: expected 429, got %d", got)
	}
	if got := verify("000000"); got != http.StatusUnauthorized {
		t.Errorf("post-limit: expected 401, got %d", got)
	}
}
