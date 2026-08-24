package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/problem"
)

type deniedCall struct {
	userID     *uint
	email      string
	method     string
	route      string
	path       string
	code       string
	reason     string
	ip         string
	userAgent  string
	orgID      *uint
	suppressed int
}

type recordingAuditor struct {
	mu    sync.Mutex
	calls []deniedCall
}

func (r *recordingAuditor) LogAccessDenied(_ context.Context, userID *uint, email, method, route, path, code, reason, ipAddress, userAgent string, orgID *uint, suppressed int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, deniedCall{userID, email, method, route, path, code, reason, ipAddress, userAgent, orgID, suppressed})
}

func (r *recordingAuditor) snapshot() []deniedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]deniedCall(nil), r.calls...)
}

// denialRouter builds a router shaped like the production one: an
// authentication stand-in that seeds the actor keys, the denial auditor
// wrapping everything, and a terminal handler the test controls.
func denialRouter(auditor AccessDenialAuditor, actorID uint, terminal gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(func(c *gin.Context) {
		if actorID != 0 {
			c.Set(ctxkeys.UserID, actorID)
			c.Set(ctxkeys.UserEmail, "staff@example.com")
		}
		c.Next()
	})
	g.Use(AuditAccessDenials(auditor))
	g.GET("/organizations/:orgId/children", terminal)
	g.DELETE("/organizations/:orgId/children/:childId", terminal)
	g.POST("/users", terminal)
	return r
}

func forbid(detail string) gin.HandlerFunc {
	return func(c *gin.Context) {
		problem.Write(c, http.StatusForbidden, apperror.CodeForbidden, detail)
	}
}

func TestAuditAccessDenials_RecordsForbidden(t *testing.T) {
	auditor := &recordingAuditor{}
	r := denialRouter(auditor, 7, forbid("forbidden"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/3/children/42", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	calls := auditor.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 denial audit, got %d", len(calls))
	}
	got := calls[0]
	if got.userID == nil || *got.userID != 7 {
		t.Errorf("expected actor 7, got %v", got.userID)
	}
	if got.email != "staff@example.com" {
		t.Errorf("expected actor email, got %q", got.email)
	}
	if got.method != http.MethodDelete {
		t.Errorf("expected DELETE, got %q", got.method)
	}
	if got.route != "/api/v1/organizations/:orgId/children/:childId" {
		t.Errorf("expected the route pattern, got %q", got.route)
	}
	if got.path != "/api/v1/organizations/3/children/42" {
		t.Errorf("expected the concrete path, got %q", got.path)
	}
	if got.code != apperror.CodeForbidden {
		t.Errorf("expected problem code %q, got %q", apperror.CodeForbidden, got.code)
	}
	if got.reason != "forbidden" {
		t.Errorf("expected the refusal detail, got %q", got.reason)
	}
	if got.userAgent != "TestAgent/1.0" {
		t.Errorf("expected the user agent, got %q", got.userAgent)
	}
	if got.orgID == nil || *got.orgID != 3 {
		t.Errorf("expected requested org 3, got %v", got.orgID)
	}
	if got.suppressed != 0 {
		t.Errorf("expected no suppressed count on a lone denial, got %d", got.suppressed)
	}
}

// The refusal reason is what separates a role denial from a mis-wired route or
// a CSRF rejection, all of which arrive as a bare 403.
func TestAuditAccessDenials_CarriesTheSpecificReason(t *testing.T) {
	for _, tc := range []struct{ code, detail string }{
		{apperror.CodeForbidden, "superadmin access required"},
		{apperror.CodeForbidden, "organization context required"},
		{"csrf_error", "CSRF token validation failed"},
	} {
		auditor := &recordingAuditor{}
		r := denialRouter(auditor, 7, func(c *gin.Context) {
			problem.Write(c, http.StatusForbidden, tc.code, tc.detail)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/users", nil))

		calls := auditor.snapshot()
		if len(calls) != 1 {
			t.Fatalf("%s: expected 1 denial audit, got %d", tc.detail, len(calls))
		}
		if calls[0].reason != tc.detail || calls[0].code != tc.code {
			t.Errorf("expected %q/%q, got %q/%q", tc.code, tc.detail, calls[0].code, calls[0].reason)
		}
		if calls[0].orgID != nil {
			t.Errorf("expected no org id on a route without :orgId, got %v", calls[0].orgID)
		}
	}
}

func TestAuditAccessDenials_IgnoresNonForbidden(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusUnauthorized, http.StatusNotFound, http.StatusBadRequest, http.StatusInternalServerError} {
		auditor := &recordingAuditor{}
		r := denialRouter(auditor, 7, func(c *gin.Context) { c.Status(status) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/users", nil))

		if n := len(auditor.snapshot()); n != 0 {
			t.Errorf("status %d: expected no denial audit, got %d", status, n)
		}
	}
}

// A 404 for a tombstoned organization must not read as a denial: the caller was
// authorized, the data is simply gone.
func TestAuditAccessDenials_TombstonedOrg404IsNotADenial(t *testing.T) {
	auditor := &recordingAuditor{}
	r := denialRouter(auditor, 7, func(c *gin.Context) {
		problem.Write(c, http.StatusNotFound, apperror.CodeNotFound, "organization not found")
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/organizations/3/children", nil))

	if n := len(auditor.snapshot()); n != 0 {
		t.Errorf("expected no denial audit for a 404, got %d", n)
	}
}

func TestAuditAccessDenials_NilAuditorIsAPassThrough(t *testing.T) {
	r := denialRouter(nil, 7, forbid("forbidden"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/users", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected the 403 to pass through untouched, got %d", w.Code)
	}
}

// A non-numeric :orgId is refused before anything parses it. The audit row must
// not invent an id for it.
func TestAuditAccessDenials_NonNumericOrgIDIsDropped(t *testing.T) {
	auditor := &recordingAuditor{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(func(c *gin.Context) { c.Set(ctxkeys.UserID, uint(7)); c.Next() })
	g.Use(AuditAccessDenials(auditor))
	g.GET("/organizations/:orgId/children", forbid("forbidden"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/organizations/not-a-number/children", nil))

	calls := auditor.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 denial audit, got %d", len(calls))
	}
	if calls[0].orgID != nil {
		t.Errorf("expected no org id for an unparseable :orgId, got %v", *calls[0].orgID)
	}
}

func TestAuditAccessDenials_ThrottlesAndReportsSuppressed(t *testing.T) {
	auditor := &recordingAuditor{}
	r := denialRouter(auditor, 7, forbid("forbidden"))

	const attempts = denialAuditLimit + 15
	for range attempts {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/users", nil))
	}

	calls := auditor.snapshot()
	if len(calls) != denialAuditLimit {
		t.Fatalf("expected the burst to be capped at %d rows, got %d", denialAuditLimit, len(calls))
	}
	for i, c := range calls {
		if c.suppressed != 0 {
			t.Errorf("row %d: nothing was suppressed before the cap was reached, got %d", i, c.suppressed)
		}
	}
}

// The suppressed count is owed to the next row that is actually written, even
// when that row lands in a later window — otherwise a burst that straddles a
// window boundary would lose the fact that it was a burst.
func TestDenialThrottle_CarriesSuppressedCountIntoTheNextWindow(t *testing.T) {
	throttle := newDenialThrottle(2)
	base := time.Now()

	for range 2 {
		if write, suppressed := throttle.admit(1, base); !write || suppressed != 0 {
			t.Fatalf("expected an unsuppressed write, got write=%v suppressed=%d", write, suppressed)
		}
	}
	for range 5 {
		if write, _ := throttle.admit(1, base); write {
			t.Fatal("expected the denial to be suppressed once over the limit")
		}
	}

	write, suppressed := throttle.admit(1, base.Add(2*time.Minute))
	if !write {
		t.Fatal("expected the new window to admit a write")
	}
	if suppressed != 5 {
		t.Errorf("expected the 5 suppressed denials to be reported, got %d", suppressed)
	}

	// Reported once, not on every subsequent row.
	if _, suppressed := throttle.admit(1, base.Add(2*time.Minute)); suppressed != 0 {
		t.Errorf("expected the suppressed count to be cleared once reported, got %d", suppressed)
	}
}

// One noisy actor must not consume another actor's budget, or a single script
// could flush real denials out of the log.
func TestDenialThrottle_IsPerActor(t *testing.T) {
	throttle := newDenialThrottle(2)
	now := time.Now()

	for range 5 {
		throttle.admit(1, now)
	}
	if write, _ := throttle.admit(2, now); !write {
		t.Error("expected a second actor to have their own budget")
	}
}

func TestDenialThrottle_ForgetKeepsActorsWithOutstandingCounts(t *testing.T) {
	throttle := newDenialThrottle(1)
	base := time.Now()

	throttle.admit(1, base) // written
	throttle.admit(1, base) // suppressed, owed to a later row
	throttle.admit(2, base) // written, nothing outstanding

	throttle.forget(base.Add(2 * time.Minute))

	throttle.mu.Lock()
	_, keptNoisy := throttle.entries[1]
	_, keptQuiet := throttle.entries[2]
	throttle.mu.Unlock()

	if !keptNoisy {
		t.Error("expected the actor with a suppressed count still owed to be kept")
	}
	if keptQuiet {
		t.Error("expected the settled actor to be forgotten")
	}
}

func TestDenialThrottle_ConcurrentAdmitIsRaceFree(t *testing.T) {
	throttle := newDenialThrottle(denialAuditLimit)
	now := time.Now()

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(actor uint) {
			defer wg.Done()
			for range 20 {
				throttle.admit(actor%5, now)
			}
		}(uint(i))
	}
	wg.Wait()
}
