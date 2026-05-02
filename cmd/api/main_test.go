package main

import (
	"net/http"
	"testing"
	"time"
)

// TestBuildHTTPServer_SetsAllTimeouts is the M3 regression guard. It
// asserts every timeout field that prevents a class of resource-
// exhaustion attack is set. Removing any one of these (especially
// ReadHeaderTimeout) would re-open the slow-loris attack the review
// flagged.
func TestBuildHTTPServer_SetsAllTimeouts(t *testing.T) {
	srv := buildHTTPServer("8080", http.NewServeMux())

	if srv.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", srv.Addr)
	}

	cases := []struct {
		name string
		got  time.Duration
	}{
		// ReadHeaderTimeout is the M3 fix. Without it a slow-loris
		// attacker can hold connections open indefinitely by trickling
		// header bytes; ReadTimeout only starts after headers are
		// fully read, so it does not save you.
		{"ReadHeaderTimeout", srv.ReadHeaderTimeout},
		{"ReadTimeout", srv.ReadTimeout},
		{"WriteTimeout", srv.WriteTimeout},
		{"IdleTimeout", srv.IdleTimeout},
	}
	for _, c := range cases {
		if c.got <= 0 {
			t.Errorf("%s must be set to a positive duration; got %v", c.name, c.got)
		}
	}

	// Hard upper bound on ReadHeaderTimeout — generous defaults are
	// fine, but anything over a minute defeats the purpose.
	if srv.ReadHeaderTimeout > time.Minute {
		t.Errorf("ReadHeaderTimeout = %v, want <= 1m (slow-loris guard would be too lax)", srv.ReadHeaderTimeout)
	}
}
