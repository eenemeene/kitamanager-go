package version

// These variables are set at build time via -ldflags.
var (
	// GitCommit is the git commit hash.
	GitCommit = "unknown"

	// BuildTime is the time the binary was built.
	BuildTime = "unknown"
)

// Version returns the git commit short hash as the version string.
func Version() string {
	if len(GitCommit) >= 7 {
		return GitCommit[:7]
	}
	return GitCommit
}
