package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// fakeSessionStore is a minimal in-memory SessionStorer so middleware tests
// don't require a database. It mirrors the contract of store.SessionStore:
// Lookup returns ErrNotFound when the id is absent, and it enforces the
// `expires_at > now()` + user_active semantics that the real JOIN query has.
type fakeSessionStore struct {
	sessions  map[string]fakeSession
	lookupErr error // if set, Lookup returns this error
	deleted   []string
}

type fakeSession struct {
	userID     uint
	userEmail  string
	userActive bool
	expiresAt  time.Time
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: make(map[string]fakeSession)}
}

func (f *fakeSessionStore) Create(ctx context.Context, sess *models.Session) error {
	f.sessions[sess.ID] = fakeSession{
		userID:     sess.UserID,
		userEmail:  "", // not relevant for middleware tests; set manually in tests that care
		userActive: true,
		expiresAt:  sess.ExpiresAt,
	}
	return nil
}

func (f *fakeSessionStore) Lookup(ctx context.Context, idHash string) (*store.SessionLookupResult, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	s, ok := f.sessions[idHash]
	if !ok {
		return nil, store.ErrNotFound
	}
	if time.Now().UTC().After(s.expiresAt) {
		return nil, store.ErrNotFound
	}
	return &store.SessionLookupResult{
		UserID:     s.userID,
		UserEmail:  s.userEmail,
		UserActive: s.userActive,
		ExpiresAt:  s.expiresAt,
	}, nil
}

func (f *fakeSessionStore) Delete(ctx context.Context, idHash string) error {
	f.deleted = append(f.deleted, idHash)
	delete(f.sessions, idHash)
	return nil
}

func (f *fakeSessionStore) DeleteAllForUser(ctx context.Context, userID uint) error {
	for h, s := range f.sessions {
		if s.userID == userID {
			delete(f.sessions, h)
		}
	}
	return nil
}

func (f *fakeSessionStore) DeleteAllForUserExcept(ctx context.Context, userID uint, keepIDHash string) error {
	for h, s := range f.sessions {
		if s.userID == userID && h != keepIDHash {
			delete(f.sessions, h)
		}
	}
	return nil
}

func (f *fakeSessionStore) CleanupExpired(ctx context.Context) error {
	now := time.Now().UTC()
	for h, s := range f.sessions {
		if now.After(s.expiresAt) {
			delete(f.sessions, h)
		}
	}
	return nil
}

func (f *fakeSessionStore) ListForUser(ctx context.Context, userID uint) ([]models.Session, error) {
	var out []models.Session
	for h, s := range f.sessions {
		if s.userID == userID && !time.Now().UTC().After(s.expiresAt) {
			out = append(out, models.Session{ID: h, UserID: s.userID, ExpiresAt: s.expiresAt})
		}
	}
	return out, nil
}

func (f *fakeSessionStore) DeleteForUser(ctx context.Context, idHash string, userID uint) (int64, error) {
	s, ok := f.sessions[idHash]
	if !ok || s.userID != userID {
		return 0, nil
	}
	delete(f.sessions, idHash)
	return 1, nil
}

// Pending-MFA methods: unused by the auth middleware tests (which are
// strictly concerned with regular-session gating), but must exist so
// *fakeSessionStore still satisfies SessionStorer.
func (f *fakeSessionStore) LookupPendingMFA(ctx context.Context, idHash string) (*store.SessionPendingLookupResult, error) {
	return nil, store.ErrNotFound
}
func (f *fakeSessionStore) BumpMFAChallengeFailures(ctx context.Context, idHash string) (int, error) {
	return 0, store.ErrNotFound
}
func (f *fakeSessionStore) DeletePendingMFA(ctx context.Context, idHash string) error {
	return nil
}

// addSession seeds a session as if it had been issued by Login, returning the
// raw cookie value.
func (f *fakeSessionStore) addSession(userID uint, email string, lifetime time.Duration) string {
	raw, hashed, err := store.GenerateSessionToken()
	if err != nil {
		panic(err)
	}
	f.sessions[hashed] = fakeSession{
		userID:     userID,
		userEmail:  email,
		userActive: true,
		expiresAt:  time.Now().UTC().Add(lifetime),
	}
	return raw
}

func newRouter(mw *AuthMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", mw.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestAuthMiddleware_NoCredentials(t *testing.T) {
	mw := NewAuthMiddleware(newFakeSessionStore())
	r := newRouter(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_MalformedAuthorizationHeader(t *testing.T) {
	mw := NewAuthMiddleware(newFakeSessionStore())
	r := newRouter(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "not-a-bearer-scheme token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for malformed Authorization, got %d", w.Code)
	}
}

func TestAuthMiddleware_UnknownSessionCookie_ClearsCookies(t *testing.T) {
	store := newFakeSessionStore()
	mw := NewAuthMiddleware(store)
	r := newRouter(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: authCookieSession, Value: "not-a-known-token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// Response must clear the two auth cookies with Max-Age=-1.
	wantCleared := map[string]bool{authCookieSession: false, authCookieCSRF: false}
	for _, c := range w.Result().Cookies() {
		if _, ok := wantCleared[c.Name]; !ok {
			continue
		}
		if c.MaxAge != -1 {
			t.Errorf("cookie %q MaxAge = %d, want -1", c.Name, c.MaxAge)
		}
		wantCleared[c.Name] = true
	}
	for name, seen := range wantCleared {
		if !seen {
			t.Errorf("expected clearing of cookie %q", name)
		}
	}
}

func TestAuthMiddleware_ValidSession_Cookie(t *testing.T) {
	sess := newFakeSessionStore()
	raw := sess.addSession(42, "test@example.com", time.Hour)
	mw := NewAuthMiddleware(sess)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	var gotUserID any
	var gotEmail any
	var gotSessionHash any
	r.GET("/test", mw.RequireAuth(), func(c *gin.Context) {
		gotUserID, _ = c.Get(ctxkeys.UserID)
		gotEmail, _ = c.Get(ctxkeys.UserEmail)
		gotSessionHash, _ = c.Get(ctxkeys.SessionIDHash)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: authCookieSession, Value: raw})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	uid, ok := gotUserID.(uint)
	if !ok || uid != 42 {
		t.Errorf("UserID ctxkey: got %v (%T), want uint(42)", gotUserID, gotUserID)
	}
	if email, ok := gotEmail.(string); !ok || email != "test@example.com" {
		t.Errorf("UserEmail ctxkey: got %v, want test@example.com", gotEmail)
	}
	if hash, ok := gotSessionHash.(string); !ok || hash != store.HashSessionToken(raw) {
		t.Errorf("SessionIDHash ctxkey: got %v, want %q", gotSessionHash, store.HashSessionToken(raw))
	}
}

func TestAuthMiddleware_ValidSession_BearerHeader(t *testing.T) {
	sess := newFakeSessionStore()
	raw := sess.addSession(7, "cli@example.com", time.Hour)
	mw := NewAuthMiddleware(sess)
	r := newRouter(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for Bearer session token, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredSession(t *testing.T) {
	sess := newFakeSessionStore()
	// Seed directly with an already-expired row (addSession uses a future expiry).
	raw, hashed, _ := store.GenerateSessionToken()
	sess.sessions[hashed] = fakeSession{
		userID:     1,
		userActive: true,
		expiresAt:  time.Now().UTC().Add(-time.Minute),
	}
	mw := NewAuthMiddleware(sess)
	r := newRouter(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: authCookieSession, Value: raw})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired session, got %d", w.Code)
	}
}

func TestAuthMiddleware_InactiveUser_SessionPruned(t *testing.T) {
	sess := newFakeSessionStore()
	raw, hashed, _ := store.GenerateSessionToken()
	sess.sessions[hashed] = fakeSession{
		userID:     1,
		userActive: false, // inactive
		expiresAt:  time.Now().UTC().Add(time.Hour),
	}
	mw := NewAuthMiddleware(sess)
	r := newRouter(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for inactive user, got %d", w.Code)
	}
	if len(sess.deleted) != 1 || sess.deleted[0] != hashed {
		t.Errorf("expected session row for inactive user to be deleted; deleted=%v", sess.deleted)
	}
}

func TestAuthMiddleware_LookupError_500(t *testing.T) {
	sess := newFakeSessionStore()
	sess.lookupErr = context.DeadlineExceeded
	mw := NewAuthMiddleware(sess)
	r := newRouter(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer something")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when Lookup returns a transport error, got %d", w.Code)
	}
}

func TestAuthMiddleware_CookieTakesPriorityOverBearer(t *testing.T) {
	// When both are present, the cookie wins. This matches the documented
	// behavior in extractRawToken and protects us from a client that sends a
	// stale Bearer header alongside a fresh cookie.
	sess := newFakeSessionStore()
	goodRaw := sess.addSession(9, "a@example.com", time.Hour)
	mw := NewAuthMiddleware(sess)
	r := newRouter(mw)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: authCookieSession, Value: goodRaw})
	req.Header.Set("Authorization", "Bearer stale-bearer-value")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (cookie should win), got %d", w.Code)
	}
}
