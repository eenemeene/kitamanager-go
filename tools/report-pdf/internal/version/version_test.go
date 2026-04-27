package version

import "testing"

func TestVersion_PrefersGitDescribe(t *testing.T) {
	orig := GitVersion
	defer func() { GitVersion = orig }()
	GitVersion = "v0.27.0"
	if got := Version(); got != "v0.27.0" {
		t.Errorf("Version() = %q, want %q", got, "v0.27.0")
	}
}

func TestVersion_FallsBackToShortCommit(t *testing.T) {
	origVer := GitVersion
	origCommit := GitCommit
	defer func() {
		GitVersion = origVer
		GitCommit = origCommit
	}()
	GitVersion = "dev"
	GitCommit = "1234567890abcdef"
	if got := Version(); got != "1234567" {
		t.Errorf("Version() = %q, want short commit %q", got, "1234567")
	}
}

func TestVersion_DefaultsToDevWhenUnbuilt(t *testing.T) {
	origVer := GitVersion
	origCommit := GitCommit
	defer func() {
		GitVersion = origVer
		GitCommit = origCommit
	}()
	GitVersion = "dev"
	GitCommit = "unknown"
	if got := Version(); got != "dev" {
		t.Errorf("Version() = %q, want %q", got, "dev")
	}
}
