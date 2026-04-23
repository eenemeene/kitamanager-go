//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
)

// clientCookies is what a real curl script keeps in its cookie jar:
// the opaque values the server hands us via Set-Cookie. A real client
// never computes CSRF tokens itself — it reads the csrf_token cookie
// the server returns and echoes it in the X-CSRF-Token header.
type clientCookies struct {
	Session   string
	CSRFToken string
}

// extractCookies pulls session + csrf_token from a response. An empty
// field means the server didn't set that cookie on this response.
func extractCookies(resp *http.Response) clientCookies {
	var cc clientCookies
	for _, c := range resp.Cookies() {
		switch c.Name {
		case "session":
			cc.Session = c.Value
		case "csrf_token":
			cc.CSRFToken = c.Value
		}
	}
	return cc
}

// doRequest is the single "curl" primitive: accepts optional cookies
// from a prior response, sets them on the request, and returns the
// recorder. No computation of tokens — the caller hands in exactly
// what they received from the server.
func doRequest(t *testing.T, r *gin.Engine, method, path string, cc clientCookies, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = strings.NewReader(string(b))
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if cc.Session != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: cc.Session})
	}
	if cc.CSRFToken != "" {
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: cc.CSRFToken})
		req.Header.Set("X-CSRF-Token", cc.CSRFToken)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestAuthFlow_MFAFullLifecycle is the end-to-end CLI-equivalent flow
// for adding 2FA to a real account and then signing back in with it.
// Everything below goes over the wire via httptest (same code path a
// curl script would hit):
//
//  1. Superadmin logs in (non-MFA) and creates a new user via POST /users.
//  2. The new user logs in with password only — no MFA yet — and gets
//     a real session cookie.
//  3. The user enrolls TOTP: POST /users/me/factors {type:totp,password}
//     receives the base32 secret, then POST /users/me/factors/:id/activate
//     with a valid code finalises the factor and returns backup codes.
//  4. The user logs out: POST /logout clears the session.
//  5. The user logs back in — this time /login returns mfa_required +
//     pending_token + factor list because the user now has MFA. They
//     complete /auth/mfa/verify with a TOTP code and receive a real
//     session cookie.
//  6. The new cookie actually works against a protected endpoint (/me).
//  7. Backup codes still work as an alternative factor on a separate
//     verify flow.
//
// A regression anywhere in the enrolment or two-step chain breaks this.
func TestAuthFlow_MFAFullLifecycle(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	enforcer, _ := rbac.NewEnforcer(testDB, findRBACModel(t))
	_, superEmail, superPass := seedSuperadmin(t, enforcer)

	// ---------- Step 1: superadmin logs in and creates a user ----------
	superCookies := loginPasswordStep(t, fr.router, superEmail, superPass)
	if superCookies.Session == "" || superCookies.CSRFToken == "" {
		t.Fatal("superadmin login: missing session or csrf cookie")
	}

	const newEmail = "mfa-lifecycle@test.local"
	const newPass = "pw-lifecycle-123"
	w := doRequest(t, fr.router, http.MethodPost, "/api/v1/users", superCookies, models.UserCreateRequest{
		Name:     "Lifecycle User",
		Email:    newEmail,
		Password: newPass,
		Active:   true,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: status=%d body=%s", w.Code, w.Body.String())
	}
	var created models.UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	// ---------- Step 2: new user logs in with password only ----------
	userCookies := loginPasswordStep(t, fr.router, newEmail, newPass)
	if userCookies.Session == "" || userCookies.CSRFToken == "" {
		t.Fatal("user login: missing session or csrf cookie")
	}
	// /me works with session cookie (no CSRF needed on GET).
	w = doRequest(t, fr.router, http.MethodGet, "/api/v1/me", userCookies, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("/me with password-only cookie: status=%d", w.Code)
	}

	// ---------- Step 3: user enrols TOTP ----------
	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/users/me/factors", userCookies, models.FactorEnrollRequest{
		Type:     models.FactorTypeTOTP,
		Password: newPass,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("enrol: status=%d body=%s", w.Code, w.Body.String())
	}
	var enrollResp models.FactorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &enrollResp); err != nil {
		t.Fatalf("decode enrol: %v", err)
	}
	// Enrollment is `any` after JSON round-trip; remarshal to extract.
	raw, _ := json.Marshal(enrollResp.Enrollment)
	var payload models.TOTPEnrollmentPayload
	_ = json.Unmarshal(raw, &payload)
	if payload.Secret == "" || payload.OTPAuthURI == "" {
		t.Fatal("enrollment payload missing secret or otpauth uri")
	}

	// Activate with a valid code.
	code, _ := totp.GenerateCode(payload.Secret, time.Now().UTC())
	w = doRequest(t, fr.router, http.MethodPost,
		fmt.Sprintf("/api/v1/users/me/factors/%d/activate", enrollResp.ID),
		userCookies, models.FactorActivateRequest{Code: code})
	if w.Code != http.StatusOK {
		t.Fatalf("activate: status=%d body=%s", w.Code, w.Body.String())
	}
	var activateResp models.FactorActivateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &activateResp); err != nil {
		t.Fatalf("decode activate: %v", err)
	}
	if !activateResp.Activated {
		t.Error("activation did not report success")
	}
	if activateResp.BackupCodes == nil || len(activateResp.BackupCodes.Codes) == 0 {
		t.Fatal("expected backup codes on first primary activation")
	}
	backupCodes := activateResp.BackupCodes.Codes
	backupFactorID := activateResp.BackupCodes.FactorID

	// ---------- Step 4: logout ----------
	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/logout", userCookies, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("logout: status=%d body=%s", w.Code, w.Body.String())
	}
	// The old cookie now 401s.
	w = doRequest(t, fr.router, http.MethodGet, "/api/v1/me", userCookies, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("post-logout /me: expected 401, got %d", w.Code)
	}

	// ---------- Step 5: login again — now MFA required ----------
	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/login", clientCookies{}, models.LoginRequest{
		Email: newEmail, Password: newPass,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login (step 1): status=%d body=%s", w.Code, w.Body.String())
	}
	var step1 models.LoginMFARequiredResponse
	if err := json.Unmarshal(w.Body.Bytes(), &step1); err != nil {
		t.Fatalf("decode step1: %v", err)
	}
	if step1.Status != models.LoginStatusMFARequired {
		t.Fatalf("login (step 1): status=%q, want mfa_required", step1.Status)
	}
	if step1.PendingToken == "" {
		t.Fatal("login (step 1): no pending_token")
	}
	// No session cookie on the MFA response.
	if gotCookies := extractCookies(w.Result()); gotCookies.Session != "" {
		t.Fatalf("login (step 1) wrongly set session cookie: %q", gotCookies.Session)
	}
	// Factor list contains both TOTP and backup_codes.
	sawTOTP, sawBackup := false, false
	for _, f := range step1.Factors {
		switch f.Type {
		case models.FactorTypeTOTP:
			sawTOTP = true
		case models.FactorTypeBackupCodes:
			sawBackup = true
		}
	}
	if !sawTOTP {
		t.Error("factor descriptor missing totp")
	}
	if !sawBackup {
		t.Error("factor descriptor missing backup_codes")
	}

	// ---------- Step 5b: complete with TOTP code ----------
	code2, _ := totp.GenerateCode(payload.Secret, time.Now().UTC())
	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/auth/mfa/verify", clientCookies{}, models.MFAVerifyRequest{
		PendingToken: step1.PendingToken,
		FactorID:     enrollResp.ID,
		Code:         code2,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("mfa verify: status=%d body=%s", w.Code, w.Body.String())
	}
	// Grab the fresh cookies the server set.
	mfaCookies := extractCookies(w.Result())
	if mfaCookies.Session == "" {
		t.Fatal("mfa verify returned no session cookie")
	}
	if mfaCookies.CSRFToken == "" {
		t.Fatal("mfa verify returned no csrf cookie")
	}
	if mfaCookies.Session == step1.PendingToken {
		t.Error("session cookie equals pending token — separation broken")
	}

	// ---------- Step 6: new cookie works against /me ----------
	w = doRequest(t, fr.router, http.MethodGet, "/api/v1/me", mfaCookies, nil)
	if w.Code != http.StatusOK {
		t.Errorf("/me with MFA-issued cookie: status=%d body=%s", w.Code, w.Body.String())
	}

	// ---------- Step 7: backup code path ----------
	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/logout", mfaCookies, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("logout #2: status=%d", w.Code)
	}
	// Login again.
	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/login", clientCookies{}, models.LoginRequest{
		Email: newEmail, Password: newPass,
	})
	var step1b models.LoginMFARequiredResponse
	_ = json.Unmarshal(w.Body.Bytes(), &step1b)

	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/auth/mfa/verify", clientCookies{}, models.MFAVerifyRequest{
		PendingToken: step1b.PendingToken,
		FactorID:     backupFactorID,
		Code:         backupCodes[0],
	})
	if w.Code != http.StatusOK {
		t.Fatalf("backup code verify: status=%d body=%s", w.Code, w.Body.String())
	}

	// Reuse of the same backup code on a fresh pending fails.
	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/login", clientCookies{}, models.LoginRequest{
		Email: newEmail, Password: newPass,
	})
	var step1c models.LoginMFARequiredResponse
	_ = json.Unmarshal(w.Body.Bytes(), &step1c)
	w = doRequest(t, fr.router, http.MethodPost, "/api/v1/auth/mfa/verify", clientCookies{}, models.MFAVerifyRequest{
		PendingToken: step1c.PendingToken,
		FactorID:     backupFactorID,
		Code:         backupCodes[0], // reused
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("reused backup code: expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

// loginPasswordStep runs /login with a password and returns the
// Set-Cookie values the server returned. Works only for non-MFA
// users (which is what the lifecycle test uses before enrolment).
// After enrolment, login returns mfa_required instead and the caller
// MUST use the /auth/mfa/verify endpoint.
func loginPasswordStep(t *testing.T, r *gin.Engine, email, password string) clientCookies {
	t.Helper()
	w := doRequest(t, r, http.MethodPost, "/api/v1/login", clientCookies{}, models.LoginRequest{
		Email: email, Password: password,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login %s: status=%d body=%s", email, w.Code, w.Body.String())
	}
	var resp models.LoginResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != models.LoginStatusAuthenticated {
		t.Fatalf("login %s: status=%q, want authenticated (this helper is for non-MFA users)", email, resp.Status)
	}
	return extractCookies(w.Result())
}
