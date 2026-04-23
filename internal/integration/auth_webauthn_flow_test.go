//go:build integration

package integration

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// webauthnFlowFixture bundles the test user + synthetic authenticator
// + router so each phase of the lifecycle test can share state.
type webauthnFlowFixture struct {
	router        *gin.Engine
	userID        uint
	email         string
	password      string
	cookieHeader  string
	csrfToken     string
	authenticator *syntheticAuthenticator
}

func setupWebAuthnFixture(t *testing.T) *webauthnFlowFixture {
	t.Helper()
	cleanupDatabase()

	const email = "webauthn@test.local"
	const password = "webauthn-password-123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := &models.User{
		Name:     "WebAuthn User",
		Email:    email,
		Password: string(hash),
		Active:   true,
	}
	if err := testDB.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	fr := setupAuthFlowRouter(t)
	return &webauthnFlowFixture{
		router:        fr.router,
		userID:        u.ID,
		email:         email,
		password:      password,
		authenticator: newSyntheticAuthenticator(t, testWebAuthnRPID),
	}
}

// loginPasswordOnly performs the password-only phase of login; for
// a user without any factor enrolled yet this sets a regular
// session cookie. Captures the session + csrf cookies for reuse on
// authenticated requests.
func (f *webauthnFlowFixture) loginPasswordOnly(t *testing.T) {
	t.Helper()
	body, _ := json.Marshal(models.LoginRequest{Email: f.email, Password: f.password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", w.Code, w.Body.String())
	}
	f.captureCookies(w.Result().Cookies())
}

func (f *webauthnFlowFixture) captureCookies(cookies []*http.Cookie) {
	parts := []string{}
	for _, c := range cookies {
		if c.Name == "session" || c.Name == "csrf_token" {
			parts = append(parts, c.Name+"="+c.Value)
		}
		if c.Name == "csrf_token" {
			f.csrfToken = c.Value
		}
	}
	if len(parts) > 0 {
		f.cookieHeader = strings.Join(parts, "; ")
	}
}

// do is a cookie-aware request helper. Mirrors what the
// auth_mfa_full_lifecycle test's doRequest does, but lives here so
// this file stays self-contained.
func (f *webauthnFlowFixture) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = strings.NewReader(string(b))
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if f.cookieHeader != "" {
		req.Header.Set("Cookie", f.cookieHeader)
	}
	if f.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", f.csrfToken)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

// enrolWebAuthn drives the two-step registration ceremony end to
// end using the synthetic authenticator. Returns the factor id on
// success.
func (f *webauthnFlowFixture) enrolWebAuthn(t *testing.T) uint {
	t.Helper()
	// Step 1: POST /factors returns creation options (with the
	// server-issued challenge).
	w := f.do(t, http.MethodPost, "/api/v1/users/me/factors", models.FactorEnrollRequest{
		Type:     models.FactorTypeWebAuthn,
		Password: f.password,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("enrol status=%d body=%s", w.Code, w.Body.String())
	}
	var resp models.FactorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode enrol: %v", err)
	}
	// The enrollment field is the raw go-webauthn creation options
	// JSON; pull the challenge out.
	raw, _ := json.Marshal(resp.Enrollment)
	challenge := extractChallengeFromOptions(t, raw, "creation_options")

	// Step 2: synthetic authenticator produces the attestation
	// object; POST to /activate.
	attObj, cdJSON, err := f.authenticator.makeAttestationObject(challenge, testWebAuthnOrigin)
	if err != nil {
		t.Fatalf("attestation: %v", err)
	}
	activateBody := models.FactorActivateRequest{
		WebAuthnResponse: marshalRawMessage(t, map[string]any{
			"id":    base64.RawURLEncoding.EncodeToString(f.authenticator.credentialID),
			"rawId": base64.RawURLEncoding.EncodeToString(f.authenticator.credentialID),
			"type":  "public-key",
			"response": map[string]any{
				"attestationObject": attObj,
				"clientDataJSON":    cdJSON,
				"transports":        []string{"internal"},
			},
			"clientExtensionResults": map[string]any{},
		}),
	}
	w = f.do(t, http.MethodPost, "/api/v1/users/me/factors/"+uintToStr(resp.ID)+"/activate", activateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("activate status=%d body=%s", w.Code, w.Body.String())
	}
	return resp.ID
}

// completeLoginWebAuthn runs the full two-step login: password →
// /auth/mfa/challenge → synthetic-authenticator assertion →
// /auth/mfa/verify. Updates the fixture's cookies on success.
func (f *webauthnFlowFixture) completeLoginWebAuthn(t *testing.T, factorID uint) {
	t.Helper()
	// Reset cookies so we start unauthenticated.
	f.cookieHeader = ""
	f.csrfToken = ""

	body, _ := json.Marshal(models.LoginRequest{Email: f.email, Password: f.password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login step 1: status=%d body=%s", w.Code, w.Body.String())
	}
	var step1 models.LoginMFARequiredResponse
	if err := json.Unmarshal(w.Body.Bytes(), &step1); err != nil {
		t.Fatalf("decode step1: %v", err)
	}
	if step1.Status != models.LoginStatusMFARequired {
		t.Fatalf("step1 status=%q, want mfa_required", step1.Status)
	}

	// Fetch challenge.
	body, _ = json.Marshal(models.MFAChallengeRequest{
		PendingToken: step1.PendingToken,
		FactorID:     factorID,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/challenge", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("challenge status=%d body=%s", w.Code, w.Body.String())
	}
	var chalResp models.MFAChallengeResponse
	_ = json.Unmarshal(w.Body.Bytes(), &chalResp)
	challenge := extractChallengeFromOptions(t, chalResp.RequestOptions, "")

	// Sign assertion.
	userHandle := encodeUserHandle(f.userID)
	assertion, err := f.authenticator.makeAssertionResponse(challenge, testWebAuthnOrigin, userHandle)
	if err != nil {
		t.Fatalf("assertion: %v", err)
	}
	body, _ = json.Marshal(models.MFAVerifyRequest{
		PendingToken:     step1.PendingToken,
		FactorID:         factorID,
		WebAuthnResponse: marshalRawMessage(t, assertion),
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", w.Code, w.Body.String())
	}
	f.captureCookies(w.Result().Cookies())
}

// TestAuthFlow_WebAuthnFullLifecycle exercises the full WebAuthn
// flow against the real backend + real go-webauthn verification
// with a synthetic authenticator.
func TestAuthFlow_WebAuthnFullLifecycle(t *testing.T) {
	f := setupWebAuthnFixture(t)

	// 1. Password-only login (no factor yet).
	f.loginPasswordOnly(t)

	// 2. Enrol a WebAuthn factor via the full two-step ceremony.
	factorID := f.enrolWebAuthn(t)

	// 3. Confirm the factor is visible on the list endpoint.
	w := f.do(t, http.MethodGet, "/api/v1/users/me/factors", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"type":"webauthn"`) {
		t.Errorf("factor list missing webauthn entry: %s", w.Body.String())
	}

	// 4. Log out.
	w = f.do(t, http.MethodPost, "/api/v1/logout", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("logout: status=%d", w.Code)
	}

	// 5. Log back in via the full WebAuthn assertion ceremony.
	f.completeLoginWebAuthn(t, factorID)

	// 6. The reissued cookies hit /me successfully.
	w = f.do(t, http.MethodGet, "/api/v1/me", nil)
	if w.Code != http.StatusOK {
		t.Errorf("/me after webauthn login: status=%d body=%s", w.Code, w.Body.String())
	}
}

// TestAuthFlow_WebAuthnAssertionReplayRejected proves an assertion
// can't be replayed against a second pending_mfa row: the
// go-webauthn library's challenge-match check fires and returns 401.
func TestAuthFlow_WebAuthnAssertionReplayRejected(t *testing.T) {
	f := setupWebAuthnFixture(t)
	f.loginPasswordOnly(t)
	factorID := f.enrolWebAuthn(t)
	w := f.do(t, http.MethodPost, "/api/v1/logout", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("logout: %d", w.Code)
	}

	// First login — capture the assertion we'll try to replay.
	body, _ := json.Marshal(models.LoginRequest{Email: f.email, Password: f.password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	var step1 models.LoginMFARequiredResponse
	_ = json.Unmarshal(w.Body.Bytes(), &step1)

	body, _ = json.Marshal(models.MFAChallengeRequest{PendingToken: step1.PendingToken, FactorID: factorID})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/challenge", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	var chalResp models.MFAChallengeResponse
	_ = json.Unmarshal(w.Body.Bytes(), &chalResp)
	challenge := extractChallengeFromOptions(t, chalResp.RequestOptions, "")

	userHandle := encodeUserHandle(f.userID)
	assertion, err := f.authenticator.makeAssertionResponse(challenge, testWebAuthnOrigin, userHandle)
	if err != nil {
		t.Fatalf("assertion: %v", err)
	}

	// Start a SECOND login without verifying the first. The replay
	// target is a fresh pending_mfa row with a different challenge.
	body, _ = json.Marshal(models.LoginRequest{Email: f.email, Password: f.password})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	var step1b models.LoginMFARequiredResponse
	_ = json.Unmarshal(w.Body.Bytes(), &step1b)
	_ = f.do(t, http.MethodPost, "/api/v1/auth/mfa/challenge", models.MFAChallengeRequest{
		PendingToken: step1b.PendingToken, FactorID: factorID,
	})

	// Submit the first ceremony's assertion against the second
	// pending_mfa row — the challenge in the assertion doesn't
	// match the server's stored one, so verify must 401.
	body, _ = json.Marshal(models.MFAVerifyRequest{
		PendingToken:     step1b.PendingToken,
		FactorID:         factorID,
		WebAuthnResponse: marshalRawMessage(t, assertion),
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("replayed assertion: expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}
