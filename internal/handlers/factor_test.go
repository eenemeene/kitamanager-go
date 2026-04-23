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
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

const factorHandlerTestAEADKey = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

// newFactorHandlerForTest wires a real FactorService against the test
// DB. Separate from `createAuthHandler` etc. because factors also need
// the TOTP AEAD.
func newFactorHandlerForTest(t *testing.T, db *gorm.DB) *FactorHandler {
	t.Helper()
	keyBytes, _ := hex.DecodeString(factorHandlerTestAEADKey)
	aead, err := cryptopkg.NewAEAD(keyBytes)
	if err != nil {
		t.Fatalf("aead: %v", err)
	}
	audit := createAuditService(db)
	svc := service.NewFactorService(
		store.NewFactorStore(db),
		store.NewUserStore(db),
		aead,
		"KitaManager (test)",
		audit,
	)
	return NewFactorHandler(svc)
}

// routerAs wires a minimal router that injects ctxkeys.UserID/Email
// the way the real auth middleware would. Each test scopes the caller
// by calling this helper.
func routerAs(user *models.User) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ctxkeys.UserID, user.ID)
		c.Set(ctxkeys.UserEmail, user.Email)
		c.Next()
	})
	return r
}

// newFactorUser is a test fixture — a user with a real bcrypt password
// so step-up checks work end-to-end.
func newFactorUser(t *testing.T, db *gorm.DB, email, password string) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	u := &models.User{
		Name:     email,
		Email:    email,
		Password: string(hash),
		Active:   true,
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// enrollTOTPViaHandler runs the POST /factors request via the
// already-registered handler. Caller MUST have mounted
// POST /api/v1/users/:userId/factors on `r` before calling this.
func enrollTOTPViaHandler(t *testing.T, r *gin.Engine, userID uint, password string) (uint, string) {
	t.Helper()
	body := models.FactorEnrollRequest{
		Type:     models.FactorTypeTOTP,
		Password: password,
	}
	w := performRequest(r, "POST", fmt.Sprintf("/api/v1/users/%d/factors", userID), body)
	if w.Code != http.StatusOK {
		t.Fatalf("enroll status=%d body=%s", w.Code, w.Body.String())
	}
	var resp models.FactorResponse
	parseResponse(t, w, &resp)
	raw, err := json.Marshal(resp.Enrollment)
	if err != nil {
		t.Fatalf("marshal enrollment: %v", err)
	}
	var payload models.TOTPEnrollmentPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return resp.ID, payload.Secret
}

func TestFactorHandler_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw-12345")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.GET("/api/v1/users/:userId/factors", h.List)

	w := performRequest(r, "GET", "/api/v1/users/me/factors", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var list models.FactorListResponse
	parseResponse(t, w, &list)
	if len(list.Factors) != 0 {
		t.Errorf("expected no factors, got %d", len(list.Factors))
	}
}

func TestFactorHandler_Enroll_RequiresPassword(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "correct-pw")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)

	w := performRequest(r, "POST", "/api/v1/users/me/factors", models.FactorEnrollRequest{
		Type: models.FactorTypeTOTP, Password: "wrong-pw",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}
}

func TestFactorHandler_Enroll_ReturnsSecretAndURI(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw-12345")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)

	w := performRequest(r, "POST", "/api/v1/users/me/factors", models.FactorEnrollRequest{
		Type: models.FactorTypeTOTP, Password: "pw-12345",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp models.FactorResponse
	parseResponse(t, w, &resp)
	if resp.Type != models.FactorTypeTOTP {
		t.Errorf("type=%q", resp.Type)
	}
	if resp.Activated {
		t.Error("newly enrolled factor must not be activated")
	}
	// Enrollment is a generic `any` after JSON round-trip; re-marshal.
	raw, _ := json.Marshal(resp.Enrollment)
	var payload models.TOTPEnrollmentPayload
	_ = json.Unmarshal(raw, &payload)
	if payload.Secret == "" {
		t.Error("expected non-empty secret in enrollment payload")
	}
	if payload.OTPAuthURI == "" {
		t.Error("expected non-empty otpauth URI")
	}
}

func TestFactorHandler_FullEnrollActivateFlow(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw-12345")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.GET("/api/v1/users/:userId/factors", h.List)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)
	r.POST("/api/v1/users/:userId/factors/:id/activate", h.Activate)

	// Enroll
	fid, secret := enrollTOTPViaHandler(t, r, user.ID, "pw-12345")

	// Activate with a valid code.
	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	w := performRequest(r, "POST",
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", fid),
		models.FactorActivateRequest{Code: code})
	if w.Code != http.StatusOK {
		t.Fatalf("activate: status=%d body=%s", w.Code, w.Body.String())
	}
	var activation models.FactorActivateResponse
	parseResponse(t, w, &activation)
	if !activation.Activated {
		t.Error("expected Activated=true")
	}
	if activation.BackupCodes == nil || len(activation.BackupCodes.Codes) == 0 {
		t.Error("expected backup codes payload on first primary activation")
	}

	// List now returns both TOTP and backup_codes.
	w = performRequest(r, "GET", "/api/v1/users/me/factors", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: status=%d", w.Code)
	}
	var list models.FactorListResponse
	parseResponse(t, w, &list)
	if len(list.Factors) != 2 {
		t.Errorf("expected 2 factors (totp + backup_codes), got %d", len(list.Factors))
	}
}

func TestFactorHandler_Activate_InvalidCode(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw-12345")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)
	r.POST("/api/v1/users/:userId/factors/:id/activate", h.Activate)

	fid, _ := enrollTOTPViaHandler(t, r, user.ID, "pw-12345")

	w := performRequest(r, "POST",
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", fid),
		models.FactorActivateRequest{Code: "000000"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong code, got %d", w.Code)
	}
}

func TestFactorHandler_CrossUser_Returns404(t *testing.T) {
	// Bob sends requests addressed at Alice's factor via explicit id.
	// The handler must return 404 — never leak Alice's factor existence.
	db := setupTestDB(t)
	alice := newFactorUser(t, db, "alice@example.com", "alice-pw")
	bob := newFactorUser(t, db, "bob@example.com", "bob-pw")
	h := newFactorHandlerForTest(t, db)

	// Alice enrolls + activates.
	aliceR := routerAs(alice)
	aliceR.POST("/api/v1/users/:userId/factors", h.Enroll)
	aliceR.POST("/api/v1/users/:userId/factors/:id/activate", h.Activate)
	aliceFID, aliceSecret := enrollTOTPViaHandler(t, aliceR, alice.ID, "alice-pw")
	code, _ := totp.GenerateCode(aliceSecret, time.Now().UTC())
	w := performRequest(aliceR, "POST",
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", aliceFID),
		models.FactorActivateRequest{Code: code})
	if w.Code != http.StatusOK {
		t.Fatalf("alice activate: %d", w.Code)
	}

	// Bob addresses Alice's factor — by its numeric user id AND by
	// direct factor id. Both must be 404.
	bobR := routerAs(bob)
	bobR.GET("/api/v1/users/:userId/factors/:id", h.Get)
	bobR.DELETE("/api/v1/users/:userId/factors/:id", h.Delete)
	bobR.POST("/api/v1/users/:userId/factors/:id/activate", h.Activate)
	bobR.POST("/api/v1/users/:userId/factors/:id/regenerate", h.Regenerate)

	// Bob GET via /users/alice.ID/factors/X : self-scope rule → 404.
	w = performRequest(bobR, "GET",
		fmt.Sprintf("/api/v1/users/%d/factors/%d", alice.ID, aliceFID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("bob GET via alice's user id: got %d, want 404", w.Code)
	}

	// Bob GET via /users/me/factors/<alice-factor-id> : ownership check → 404.
	w = performRequest(bobR, "GET",
		fmt.Sprintf("/api/v1/users/me/factors/%d", aliceFID), nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("bob GET /me/factors/<alice-id>: got %d, want 404", w.Code)
	}

	// Bob DELETE at /users/me/factors/<alice-id>: must 404 even with bob's own password.
	w = performRequest(bobR, "DELETE",
		fmt.Sprintf("/api/v1/users/me/factors/%d", aliceFID),
		models.FactorDeleteRequest{Password: "bob-pw"})
	if w.Code != http.StatusNotFound {
		t.Errorf("bob DELETE /me/factors/<alice-id>: got %d, want 404", w.Code)
	}

	// Alice's factor must still exist.
	aliceCheckR := routerAs(alice)
	aliceCheckR.GET("/api/v1/users/:userId/factors/:id", h.Get)
	w = performRequest(aliceCheckR, "GET",
		fmt.Sprintf("/api/v1/users/me/factors/%d", aliceFID), nil)
	if w.Code != http.StatusOK {
		t.Errorf("alice's factor was wrongly affected: got %d", w.Code)
	}
}

func TestFactorHandler_UpdateLabel(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)
	r.POST("/api/v1/users/:userId/factors/:id/activate", h.Activate)
	r.PATCH("/api/v1/users/:userId/factors/:id", h.UpdateLabel)

	fid, secret := enrollTOTPViaHandler(t, r, user.ID, "pw")
	code, _ := totp.GenerateCode(secret, time.Now().UTC())
	_ = performRequest(r, "POST",
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", fid),
		models.FactorActivateRequest{Code: code})

	// Rename.
	label := "Old phone"
	w := performRequest(r, "PATCH",
		fmt.Sprintf("/api/v1/users/me/factors/%d", fid),
		models.FactorLabelUpdateRequest{Label: &label})
	if w.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	var resp models.FactorResponse
	parseResponse(t, w, &resp)
	if resp.Label == nil || *resp.Label != label {
		t.Errorf("label = %v, want %q", resp.Label, label)
	}
}

func TestFactorHandler_NotAuthenticated(t *testing.T) {
	db := setupTestDB(t)
	h := newFactorHandlerForTest(t, db)

	// No middleware — ctxkeys.UserID is absent.
	r := gin.New()
	r.GET("/api/v1/users/:userId/factors", h.List)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)

	w := performRequest(r, "GET", "/api/v1/users/me/factors", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestFactorHandler_MalformedFactorID(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.GET("/api/v1/users/:userId/factors/:id", h.Get)

	// Non-numeric id → 404 (uniform with "not found" — don't leak bad format).
	w := performRequest(r, "GET", "/api/v1/users/me/factors/not-a-number", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for malformed id, got %d", w.Code)
	}
}

// TestFactorHandler_Enroll_UnsupportedType: a factor type the server
// doesn't know about must be rejected at the handler boundary with a
// clean 400 — not fall through to some default path that would create
// a row the rest of the system can't reason about.
func TestFactorHandler_Enroll_UnsupportedType(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw-12345")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)

	w := performRequest(r, "POST", "/api/v1/users/me/factors", models.FactorEnrollRequest{
		Type:     "webauthn", // not yet implemented
		Password: "pw-12345",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("unsupported type should be 400, got %d body=%s", w.Code, w.Body.String())
	}
}

// TestFactorHandler_Enroll_BackupCodesDirectly_Rejected: backup_codes
// is a server-managed singleton auto-created on first primary factor
// activation. Allowing the client to enrol one directly would create
// either duplicate rows or split-brain state, so the handler must 400.
func TestFactorHandler_Enroll_BackupCodesDirectly_Rejected(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw-12345")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)

	w := performRequest(r, "POST", "/api/v1/users/me/factors", models.FactorEnrollRequest{
		Type:     models.FactorTypeBackupCodes,
		Password: "pw-12345",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("direct backup_codes enrol should be 400, got %d body=%s", w.Code, w.Body.String())
	}
}

// TestFactorHandler_Activate_FiveWrongCodes_Returns429: mirrors the
// service-level test at the HTTP boundary. After FactorActivationFailureLimit
// wrong codes the endpoint responds 429 and the pending factor vanishes.
func TestFactorHandler_Activate_FiveWrongCodes_Returns429(t *testing.T) {
	db := setupTestDB(t)
	user := newFactorUser(t, db, "u@example.com", "pw")
	h := newFactorHandlerForTest(t, db)

	r := routerAs(user)
	r.POST("/api/v1/users/:userId/factors", h.Enroll)
	r.POST("/api/v1/users/:userId/factors/:id/activate", h.Activate)

	fid, _ := enrollTOTPViaHandler(t, r, user.ID, "pw")

	for i := 1; i <= service.FactorActivationFailureLimit-1; i++ {
		w := performRequest(r, "POST",
			fmt.Sprintf("/api/v1/users/me/factors/%d/activate", fid),
			models.FactorActivateRequest{Code: "000000"})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d body=%s", i, w.Code, w.Body.String())
		}
	}
	// Nth wrong code trips the limit.
	w := performRequest(r, "POST",
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", fid),
		models.FactorActivateRequest{Code: "000000"})
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("limit attempt: expected 429, got %d body=%s", w.Code, w.Body.String())
	}
	// Subsequent request is 404 — factor row is gone.
	w = performRequest(r, "POST",
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", fid),
		models.FactorActivateRequest{Code: "000000"})
	if w.Code != http.StatusNotFound {
		t.Errorf("post-limit attempt: expected 404, got %d", w.Code)
	}
}

// Trivial context-carrying helper to satisfy the unused-import guard
// if service/store happen not to be referenced in any test.
var _ = context.Background
