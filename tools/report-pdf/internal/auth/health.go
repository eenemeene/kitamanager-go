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
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
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
