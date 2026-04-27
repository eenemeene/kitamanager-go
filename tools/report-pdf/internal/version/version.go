// Package version exposes build-time identifiers for the report-pdf tool.
// Mirrors the API's internal/version package so the same Makefile pattern
// (ldflags -X $PKG.GitVersion=... etc.) populates both binaries the same way.
package version

// These variables are set at build time via -ldflags. The defaults make
// `go run` and `go build` produce a binary that says "dev" rather than
// failing; production builds (Makefile + Dockerfile.report) override them
// with `git describe`, the commit SHA, and an ISO timestamp.
var (
	// GitVersion is the version string from git describe (e.g. "v0.27.0" or "v0.27.0-3-g1a2b3c4").
	GitVersion = "dev"

	// GitCommit is the git commit hash.
	GitCommit = "unknown"

	// BuildTime is the time the binary was built (RFC3339 / ISO 8601).
	BuildTime = "unknown"
)

// Version returns the human-friendly version string. Prefers a real
// git-describe value, falls back to a short commit hash, finally to
// the literal sentinel for unbuilt sources.
func Version() string {
	if GitVersion != "" && GitVersion != "dev" {
		return GitVersion
	}
	if len(GitCommit) >= 7 && GitCommit != "unknown" {
		return GitCommit[:7]
	}
	return GitVersion
}
