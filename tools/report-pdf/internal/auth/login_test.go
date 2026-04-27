package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/login" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected content-type: %s", ct)
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.Email != "user@test.com" {
			t.Errorf("Email = %q, want %q", req.Email, "user@test.com")
		}
		if req.Password != "secret" {
			t.Errorf("Password = %q, want %q", req.Password, "secret")
		}

		http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "tok123", Path: "/", HttpOnly: true})
		http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "ref456", Path: "/api/v1/refresh", HttpOnly: true})
		http.SetCookie(w, &http.Cookie{Name: "csrf_token", Value: "csrf789", Path: "/"})
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "authenticated", "expires_in": 3600})
	}))
	defer server.Close()

	cookies, err := Login(server.URL, "user@test.com", "secret", "http://localhost:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cookies) != 3 {
		t.Fatalf("got %d cookies, want 3", len(cookies))
	}

	cookieMap := make(map[string]string)
	for _, c := range cookies {
		cookieMap[c.Name] = c.Value
	}

	if cookieMap["access_token"] != "tok123" {
		t.Errorf("access_token = %q", cookieMap["access_token"])
	}
	if cookieMap["refresh_token"] != "ref456" {
		t.Errorf("refresh_token = %q", cookieMap["refresh_token"])
	}
	if cookieMap["csrf_token"] != "csrf789" {
		t.Errorf("csrf_token = %q", cookieMap["csrf_token"])
	}

	for _, c := range cookies {
		if *c.Domain != "localhost" {
			t.Errorf("cookie %q domain = %q, want %q", c.Name, *c.Domain, "localhost")
		}
	}
}

func TestLogin_ExtractsDomainFromBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// API doesn't set a Domain attribute (host-only cookie) — the
		// dev / same-host case. Login should fall back to the
		// frontend hostname so Playwright stores it where page.Goto
		// will read it.
		http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "tok", Path: "/"})
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
	}))
	defer server.Close()

	cookies, err := Login(server.URL, "a@b.com", "pw", "https://app.example.com:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	if *cookies[0].Domain != "app.example.com" {
		t.Errorf("domain = %q, want %q", *cookies[0].Domain, "app.example.com")
	}
}

func TestLogin_PreservesExplicitCookieDomain(t *testing.T) {
	// Cross-subdomain prod deployment: the API at api.example.com sets
	// Domain=.example.com so the cookie is shared with the frontend at
	// app.example.com. Login must NOT rewrite this to the frontend
	// hostname — that would scope the cookie too tightly and break the
	// shared-parent setup.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "access_token",
			Value:  "tok",
			Path:   "/",
			Domain: ".example.com",
		})
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
	}))
	defer server.Close()

	cookies, err := Login(server.URL, "a@b.com", "pw", "https://app.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	// Go's http parser strips the leading dot per RFC 6265 §4.1.2.3,
	// but the semantics (parent-domain scoping) are preserved as long
	// as we forward whatever the API gave us.
	if got := *cookies[0].Domain; got != "example.com" && got != ".example.com" {
		t.Errorf("domain = %q, want example.com or .example.com (preserved from API)", got)
	}
}

func TestLogin_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := Login(server.URL, "a@b.com", "wrong", "http://localhost:3000")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestLogin_NoCookies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "authenticated", "expires_in": 3600})
	}))
	defer server.Close()

	_, err := Login(server.URL, "a@b.com", "pw", "http://localhost:3000")
	if err == nil {
		t.Fatal("expected error when no cookies returned")
	}
	if !strings.Contains(err.Error(), "no session cookie") {
		t.Errorf("error = %q, want it to mention missing session cookie", err)
	}
}

func TestLogin_MFARequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Real API behaviour: 200 OK with status=mfa_required and *no* cookie.
		// Without explicit MFA detection, the report tool would surface this
		// as "no cookies received" — a confusing error for what is really a
		// service-account configuration problem.
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":        "mfa_required",
			"pending_token": "abc123",
			"expires_at":    "Wed, 23 Apr 2026 10:05:00 GMT",
			"factors":       []map[string]string{{"id": "f1", "type": "totp"}},
		})
	}))
	defer server.Close()

	_, err := Login(server.URL, "a@b.com", "pw", "http://localhost:3000")
	if err == nil {
		t.Fatal("expected error for MFA-required response")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "mfa") {
		t.Errorf("error = %q, want it to mention MFA", err)
	}
}

func TestLogin_UnknownStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "something_new"})
	}))
	defer server.Close()

	_, err := Login(server.URL, "a@b.com", "pw", "http://localhost:3000")
	if err == nil {
		t.Fatal("expected error for unknown status value")
	}
}

func TestLogin_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := Login(server.URL, "a@b.com", "pw", "http://localhost:3000")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestLogin_ConnectionRefused(t *testing.T) {
	_, err := Login("http://localhost:1", "a@b.com", "pw", "http://localhost:3000")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestLogin_PreservesCookiePaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "tok", Path: "/", HttpOnly: true})
		http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "ref", Path: "/api/v1/refresh", HttpOnly: true})
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
	}))
	defer server.Close()

	cookies, err := Login(server.URL, "a@b.com", "pw", "http://localhost:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pathMap := make(map[string]string)
	for _, c := range cookies {
		pathMap[c.Name] = *c.Path
	}

	if pathMap["access_token"] != "/" {
		t.Errorf("access_token path = %q, want %q", pathMap["access_token"], "/")
	}
	if pathMap["refresh_token"] != "/api/v1/refresh" {
		t.Errorf("refresh_token path = %q, want %q", pathMap["refresh_token"], "/api/v1/refresh")
	}
}
