package handlers

// If-Match preconditions on contract writes.
//
// A contract's care type and supplements decide what the Kita is paid, so a lost
// update quietly changes money. The version column and the version-guarded store
// updates make a stale write fail — but only if the client says which version it
// believes it is editing. If-Match is how it says that.
//
// Required, not optional: an optional precondition is one clients forget, and
// the failure mode of forgetting is silent. A write without If-Match is answered
// 428 (nothing was compared — read the contract and retry with its version),
// which is a different situation from 412 (you lost a race).

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
)

// versionedResponse is implemented by the contract response DTOs. Used through a
// type assertion so the generic contract helpers do not need another type
// parameter just to reach one field.
type versionedResponse interface {
	GetVersion() int64
}

// setVersionETag publishes a contract's version as an ETag so a client can echo
// it back as a precondition. Quoted per RFC 9110; strong, because the version is
// exact rather than a heuristic.
func setVersionETag(c *gin.Context, resp any) {
	if v, ok := resp.(versionedResponse); ok {
		c.Header("ETag", strconv.Quote(strconv.FormatInt(v.GetVersion(), 10)))
	}
}

// requireIfMatch reads the If-Match precondition from the request.
//
// Returns (expected, true) on success, where a nil expected means the wildcard
// `*` — RFC 9110's "any current version", i.e. the caller deliberately opts out
// of the check. On failure it has already written the response.
func requireIfMatch(c *gin.Context) (*int64, bool) {
	raw := strings.TrimSpace(c.GetHeader("If-Match"))

	if raw == "" {
		respondError(c, apperror.PreconditionRequired(
			`this write requires an If-Match header carrying the contract's version, e.g. If-Match: "3" — `+
				`GET the contract (or read `+"`version`"+` from the list response) and echo it back`))
		return nil, false
	}

	// `*` means "whatever the current version is". Kept because it is the
	// standard way to say "I know what I am doing", not because anything here
	// needs it.
	if raw == "*" {
		return nil, true
	}

	// A list is legal HTTP but meaningless for a single integer version, and
	// silently taking the first entry would hide a client bug.
	if strings.Contains(raw, ",") {
		respondError(c, apperror.BadRequest("If-Match must carry exactly one version, not a list"))
		return nil, false
	}

	// Weak validators cannot be used with If-Match: the comparison is defined as
	// strong, so `W/"3"` is a client error rather than something to interpret.
	if strings.HasPrefix(raw, "W/") || strings.HasPrefix(raw, "w/") {
		respondError(c, apperror.BadRequest(`If-Match requires a strong validator, so W/"..." is not accepted`))
		return nil, false
	}

	// Quoted is the canonical form; a bare number is tolerated because it is
	// unambiguous and a plausible thing for a hand-written request to send.
	value := raw
	if unquoted, err := strconv.Unquote(raw); err == nil {
		value = unquoted
	}

	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil || version < 1 {
		respondError(c, apperror.BadRequest(
			`If-Match must be a contract version, e.g. If-Match: "3"`))
		return nil, false
	}
	return &version, true
}

// versionPrecondition is implemented by the request DTOs that carry an If-Match
// expectation into the service, where it is compared against the contract as
// freshly loaded. It has to be compared there, not here: comparing in the
// handler would leave a window in which another writer bumps the version between
// the check and the load, and the store's guard would then happily write on top
// of that newer state.
type versionPrecondition interface {
	SetExpectedVersion(*int64)
}

// applyIfMatch reads the precondition and hands it to the request. Returns false
// when the response has already been written.
func applyIfMatch(c *gin.Context, req any) bool {
	expected, ok := requireIfMatch(c)
	if !ok {
		return false
	}
	if p, ok := req.(versionPrecondition); ok {
		p.SetExpectedVersion(expected)
	}
	return true
}
