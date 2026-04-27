package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/playwright-community/playwright-go"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is the subset of the /login body the report tool cares
// about. The API returns either {status:"authenticated"} (session cookie
// set) or {status:"mfa_required", ...} (no cookie, second factor needed).
// We don't model the MFA-required body fully — just enough to surface a
// clear error to the operator.
type loginResponse struct {
	Status string `json:"status"`
}

const (
	loginStatusAuthenticated = "authenticated"
	loginStatusMFARequired   = "mfa_required"
)

// Login authenticates against the API and returns cookies suitable for Playwright.
func Login(apiURL, email, password, baseURL string) ([]playwright.OptionalCookie, error) {
	body, err := json.Marshal(loginRequest{Email: email, Password: password})
	if err != nil {
		return nil, fmt.Errorf("marshal login request: %w", err)
	}

	// Use a raw http.Client that does NOT follow redirects, so we capture Set-Cookie headers.
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Post(apiURL+"/api/v1/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	// Parse the body to detect the MFA-required branch. Without this, an
	// MFA-protected account sees only "no cookies received" — a confusing
	// surface for what is really a configuration problem (the report tool
	// is non-interactive and cannot complete the second factor).
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read login response: %w", err)
	}
	var parsed loginResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse login response: %w", err)
	}
	switch parsed.Status {
	case loginStatusAuthenticated:
		// fall through to cookie extraction
	case loginStatusMFARequired:
		return nil, fmt.Errorf("login requires MFA — the report tool is non-interactive and cannot complete a second factor; use a dedicated service account with MFA disabled")
	default:
		return nil, fmt.Errorf("login response had unexpected status %q", parsed.Status)
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	frontendHost := parsedURL.Hostname()

	var pwCookies []playwright.OptionalCookie
	for _, c := range resp.Cookies() {
		// Preserve the API's original Domain attribute when set:
		// in a cross-subdomain prod deployment (api.example.com +
		// app.example.com) the API issues `Domain=.example.com` and
		// rewriting it to the frontend hostname would defeat the
		// shared-parent setup. When the API issues a host-only
		// cookie (no Domain attribute — the dev/same-host case),
		// fall back to the frontend hostname so Playwright stores
		// it where the next page.Goto will read it.
		cookieDomain := c.Domain
		if cookieDomain == "" {
			cookieDomain = frontendHost
		}
		pwCookie := playwright.OptionalCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   &cookieDomain,
			Path:     playwright.String(c.Path),
			HttpOnly: playwright.Bool(c.HttpOnly),
			Secure:   playwright.Bool(c.Secure),
			SameSite: playwright.SameSiteAttributeStrict,
		}
		pwCookies = append(pwCookies, pwCookie)
	}

	if len(pwCookies) == 0 {
		return nil, fmt.Errorf("login succeeded but no session cookie was set")
	}

	return pwCookies, nil
}
