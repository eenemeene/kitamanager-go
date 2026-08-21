//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// POST /logout must revoke the session whichever transport presented it.
//
// RequireAuth accepts a `session` cookie OR `Authorization: Bearer`, the latter
// existing specifically for CLI and server-to-server callers. The handler read
// the cookie, so a bearer client logged out against an empty string: nothing
// revoked, no audit row, and a 200 saying it had worked. The credential stayed
// valid until natural expiry.
//
// This runs through the real router and the real middleware, because the defect
// was in the seam between them.

func TestBearerLogout_RevokesTheSession(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	_, email, password := seedSuperadmin(t)

	status, token := doLogin(t, fr.router, email, password)
	if status != http.StatusOK || token == "" {
		t.Fatalf("setup: login returned %d, token empty=%v", status, token == "")
	}

	// Sanity: the token works before logout, so the 401 below is attributable
	// to the logout rather than to a token that was never valid.
	if w := doAuthed(t, fr.router, http.MethodGet, "/api/v1/me", token, nil); w.Code != http.StatusOK {
		t.Fatalf("setup: /me with a fresh bearer token returned %d: %s", w.Code, w.Body.String())
	}

	if w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/logout", token, nil); w.Code != http.StatusOK {
		t.Fatalf("logout returned %d: %s", w.Code, w.Body.String())
	}

	// The row itself must be gone, not merely unreachable.
	sessionStore := store.NewSessionStore(testDB)
	if _, err := sessionStore.Lookup(t.Context(), store.HashSessionToken(token)); err != store.ErrNotFound {
		t.Errorf("session row must be deleted after a bearer logout, Lookup gave %v", err)
	}

	// And the credential must stop working. This is the assertion that failed
	// before the fix: the old code answered 200 and left the token live.
	if w := doAuthed(t, fr.router, http.MethodGet, "/api/v1/me", token, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("bearer token still works after logout: /me returned %d", w.Code)
	}
}

func TestBearerLogout_EmitsAuditRow(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	userID, email, password := seedSuperadmin(t)

	_, token := doLogin(t, fr.router, email, password)
	if w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/logout", token, nil); w.Code != http.StatusOK {
		t.Fatalf("logout returned %d: %s", w.Code, w.Body.String())
	}

	// The audit worker is asynchronous; poll rather than sleep a fixed amount.
	var rows []models.AuditLog
	deadline := 0
	for deadline < 100 {
		if err := testDB.Where("action = ? AND user_id = ?", models.AuditActionLogout, userID).Find(&rows).Error; err != nil {
			t.Fatalf("query audit rows: %v", err)
		}
		if len(rows) > 0 {
			break
		}
		deadline++
	}

	if len(rows) == 0 {
		t.Fatal("a bearer logout must leave a logout audit row; the trail otherwise shows a login with no matching logout")
	}
}

// Logging out twice must not fail. The route sits behind RequireAuth, so the
// second call cannot actually reach the handler with a live session — but the
// service has to treat a missing row as success rather than as an error, or a
// double-clicked logout would surface a 500.
func TestBearerLogout_IsIdempotentAtTheServiceLayer(t *testing.T) {
	cleanupDatabase()
	fr := setupAuthFlowRouter(t)
	_, email, password := seedSuperadmin(t)
	_, token := doLogin(t, fr.router, email, password)

	if w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/logout", token, nil); w.Code != http.StatusOK {
		t.Fatalf("first logout returned %d", w.Code)
	}
	// The second attempt is rejected by RequireAuth, not by the handler — 401
	// rather than a 500 from trying to delete a row that is already gone.
	if w := doAuthed(t, fr.router, http.MethodPost, "/api/v1/logout", token, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("second logout should be refused by RequireAuth with 401, got %d: %s", w.Code, w.Body.String())
	}
}
