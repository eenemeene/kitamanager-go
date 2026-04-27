package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// healthResponse mirrors models.HealthResponse on the API side. We only
// care about Version for the report colophon — the other fields are
// intentionally not modeled so a future addition to the response shape
// can't break this client.
type healthResponse struct {
	Version string `json:"version"`
}

// FetchAPIVersion calls GET {apiURL}/api/v1/health and returns the
// reported API version string. Used by the report tool to embed the
// API version (alongside the CLI's own embedded version) in the
// colophon stamp at the foot of the rendered PDF.
//
// Lives in the auth package because that's where the report tool's
// only other HTTP client lives — they share a Go module + the
// "thin standalone HTTP" style. A short timeout + a no-redirect
// client matches the Login() shape.
func FetchAPIVersion(apiURL string) (string, error) {
	client := newVersionClient()
	resp, err := client.Get(apiURL + "/api/v1/health")
	if err != nil {
		return "", fmt.Errorf("health request failed: %w", err)
	}
	defer resp.Body.Close()
	// /health returns 503 when the API can talk but the DB can't —
	// the version field is still populated in that case, so accept
	// 200 OR 503 here. 4xx responses we treat as "no version".
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return "", fmt.Errorf("health returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read health response: %w", err)
	}
	var parsed healthResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse health response: %w", err)
	}
	return parsed.Version, nil
}

// webVersionResponse is the shape served by the frontend's /version
// route handler (frontend/src/app/version/route.ts). Modeled here as
// the minimum field we need so an additive change on the frontend
// (e.g. adding `node_version`) doesn't break the parse.
type webVersionResponse struct {
	Version string `json:"version"`
}

// FetchWebVersion calls GET {baseURL}/version on the frontend and
// returns the reported web/UI version. The frontend ships as its own
// container image and may be at a different version than the API —
// the colophon needs both to be honest about which exact pair of
// images rendered the PDF.
func FetchWebVersion(baseURL string) (string, error) {
	client := newVersionClient()
	resp, err := client.Get(baseURL + "/version")
	if err != nil {
		return "", fmt.Errorf("web version request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web version returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read web version response: %w", err)
	}
	var parsed webVersionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse web version response: %w", err)
	}
	return parsed.Version, nil
}

// newVersionClient builds the http.Client both Fetch*Version helpers
// share: short timeout, no redirect-following. Factored out so the
// two helpers stay byte-for-byte consistent in their HTTP behaviour.
func newVersionClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
