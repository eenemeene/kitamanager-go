package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
)

// AccessDenialAuditor is the slice of *service.AuditService this middleware
// needs.
//
// It is an interface declared here rather than a direct dependency because
// service already imports middleware (for RequestIDFromContext), so importing
// service back would be a cycle. Go's structural interfaces make the seam free:
// *service.AuditService satisfies this without knowing the type exists.
type AccessDenialAuditor interface {
	LogAccessDenied(ctx context.Context, userID *uint, email, method, route, path, code, reason, ipAddress, userAgent string, orgID *uint, suppressed int)
}

// denialAuditLimit is how many 403s one actor can have recorded per window
// before the rest of the window is summarised instead of written row by row.
//
// The number is set for the shape of real traffic rather than as a security
// parameter. A user who lands on a page their role cannot read produces a
// handful of denials in one burst — a page with several panels can produce a
// dozen — so the limit has to sit above "normal misnavigation" or the throttle
// would be summarising the common case. Above it, the audit value of the
// hundredth identical denial is nil while its cost is another row.
const denialAuditLimit = 20

// denialAuditWindow is the period denialAuditLimit applies to.
const denialAuditWindow = time.Minute

// denialThrottle bounds how many access_denied rows a single actor can cause.
//
// Denials are audited for reads as well as writes, and only mutations are rate
// limited at the API edge, so a loop against a forbidden GET would otherwise
// translate one HTTP request into one audit INSERT with nothing in between.
// On the single-instance deployments this app targets that is a self-inflicted
// write amplification against the same database the app serves from.
//
// Suppression never loses the event, only its individual row: the count of
// what was swallowed rides along on the next row that is written, so a burst
// reads as a burst. Suppressed counts are therefore reset when reported, not
// when the window rolls over.
//
// Keyed by user id, so one actor cannot flush another's denials out. The map is
// bounded by the number of users that have ever been refused something, which
// is bounded by the user table; entries are dropped on the next access after
// their window expires with nothing outstanding to report.
type denialThrottle struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[uint]*denialWindow
}

type denialWindow struct {
	windowEnd  time.Time
	emitted    int
	suppressed int
}

// newDenialThrottle takes the limit but not the window: tests drive time
// through admit's `now` argument rather than by shortening the window, so
// there has never been a reason for a second value of it.
func newDenialThrottle(limit int) *denialThrottle {
	return &denialThrottle{
		limit:   limit,
		window:  denialAuditWindow,
		entries: make(map[uint]*denialWindow),
	}
}

// admit reports whether this denial should be written as its own audit row and,
// if so, how many denials were suppressed since the previous written one.
func (t *denialThrottle) admit(userID uint, now time.Time) (write bool, suppressed int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.entries[userID]
	if !ok {
		e = &denialWindow{}
		t.entries[userID] = e
	}
	if now.After(e.windowEnd) {
		// New window. The suppressed count deliberately survives the roll:
		// it is owed to the next row that gets written, whenever that is.
		e.windowEnd = now.Add(t.window)
		e.emitted = 0
	}

	if e.emitted >= t.limit {
		e.suppressed++
		return false, 0
	}

	e.emitted++
	suppressed, e.suppressed = e.suppressed, 0
	return true, suppressed
}

// forget drops per-actor state that has nothing left to say. Called
// opportunistically so the map does not accumulate one entry per user who was
// ever refused anything for the lifetime of the process.
func (t *denialThrottle) forget(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, e := range t.entries {
		if now.After(e.windowEnd) && e.suppressed == 0 {
			delete(t.entries, id)
		}
	}
}

// AuditAccessDenials records every authenticated request that ends in 403.
//
// It is written as a wrapping middleware rather than as a call inside each
// authorization check because a 403 has four unrelated sources — the RBAC
// permission check, the superadmin gate, the service-layer superadmin guards
// reached through respondError, and CSRF validation — and only the response
// carries all four. Registering it once, immediately inside RequireAuth and
// outside everything else, covers them by construction, including any future
// fifth source.
//
// It must sit inside RequireAuth: an unauthenticated request is refused with
// 401 before this runs, which is deliberate. Expired sessions are ordinary
// traffic and auditing them would bury the denials that mean something.
//
// A nil auditor makes this a pass-through, so tests and any deployment that
// wires routes without an audit service keep working.
func AuditAccessDenials(auditor AccessDenialAuditor) gin.HandlerFunc {
	throttle := newDenialThrottle(denialAuditLimit)
	var sweeps int

	return func(c *gin.Context) {
		c.Next()

		if auditor == nil || c.Writer.Status() != http.StatusForbidden {
			return
		}

		now := time.Now()

		// Sweep occasionally rather than on a timer: this middleware has no
		// lifecycle to hang a goroutine off, and the map is small enough that
		// an amortised pass is cheaper than owning a background worker.
		sweeps++
		if sweeps%256 == 0 {
			throttle.forget(now)
		}

		var userID *uint
		if v, ok := c.Get(ctxkeys.UserID); ok {
			if id, ok := v.(uint); ok && id != 0 {
				userID = &id
			}
		}

		// An actor we cannot name cannot be throttled per actor. That should
		// not happen inside the protected group, but if it ever does, record
		// the event rather than dropping it — an unattributable 403 is more
		// interesting than an ordinary one, not less.
		write, suppressed := true, 0
		if userID != nil {
			write, suppressed = throttle.admit(*userID, now)
		}
		if !write {
			return
		}

		email, _ := c.Get(ctxkeys.UserEmail)
		emailStr, _ := email.(string)
		code, _ := c.Get(ctxkeys.ProblemCode)
		codeStr, _ := code.(string)
		reason, _ := c.Get(ctxkeys.ProblemDetail)
		reasonStr, _ := reason.(string)

		auditor.LogAccessDenied(
			c.Request.Context(),
			userID,
			emailStr,
			c.Request.Method,
			c.FullPath(),
			c.Request.URL.Path,
			codeStr,
			reasonStr,
			c.ClientIP(),
			c.Request.UserAgent(),
			denialOrgID(c),
			suppressed,
		)
	}
}

// denialOrgID resolves the :orgId path parameter of the route that was refused.
//
// Returns nil when the route is not org-scoped, or when the parameter is not a
// number — a request for /organizations/abc/children is refused before anything
// parses it, and an audit row is not the place to start inventing an org id.
func denialOrgID(c *gin.Context) *uint {
	raw := c.Param("orgId")
	if raw == "" {
		return nil
	}
	parsed, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return nil
	}
	id := uint(parsed)
	return &id
}
