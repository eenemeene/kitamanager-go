package models

import (
	"log/slog"
	"os"
	"sync"
	"time"

	// Embed tzdata so Europe/Berlin (and any other zone configured via the
	// KITAMANAGER_TIMEZONE env var) resolves regardless of whether the
	// runtime container ships /usr/share/zoneinfo. The Chainguard `static`
	// base image used by Dockerfile.api strips zoneinfo to keep the image
	// minimal — without this import, time.LoadLocation("Europe/Berlin")
	// returns "unknown time zone" and the app silently falls back to UTC.
	_ "time/tzdata"
)

const (
	// AppTimezoneEnv overrides the application's calendar timezone. The
	// value must be a name resolvable by time.LoadLocation (an IANA zone
	// like "Europe/Berlin" or "UTC"). Tests can pin behavior by setting
	// this before the first call to AppLocation.
	AppTimezoneEnv = "KITAMANAGER_TIMEZONE"

	// DefaultAppTimezone is the calendar timezone used when AppTimezoneEnv
	// is unset. KitaManager today serves German Bundesländer exclusively
	// (Organization.State defaults to "berlin") and all of them share the
	// same zone, so a single default is correct. If the app expands beyond
	// Germany, this becomes per-organization.
	DefaultAppTimezone = "Europe/Berlin"
)

var (
	appLocationOnce sync.Once
	appLocation     *time.Location

	// nowMu guards nowFn so SetNow can be called from one goroutine while
	// Today() is read from another. Tests that pin the clock should NOT
	// also enable t.Parallel() within the same package — the seam itself
	// is concurrency-safe, but two pinned values racing each other is
	// nonsense.
	nowMu sync.RWMutex
	nowFn = time.Now
)

// AppLocation returns the application's calendar timezone, read from
// AppTimezoneEnv on first call (default: Europe/Berlin). Subsequent calls
// return the cached *time.Location.
//
// On a malformed or unknown zone name the loader logs a slog.Warn and falls
// back to UTC so the process keeps serving requests. The fallback is
// deliberately silent-but-noisy: refusing to start would convert a config
// typo into an outage, while ignoring it entirely would mask drift between
// "today" and what users expect.
func AppLocation() *time.Location {
	appLocationOnce.Do(func() {
		name := os.Getenv(AppTimezoneEnv)
		if name == "" {
			name = DefaultAppTimezone
		}
		loc, err := time.LoadLocation(name)
		if err != nil {
			slog.Warn("invalid app timezone, falling back to UTC",
				"env", AppTimezoneEnv, "value", name, "error", err)
			loc = time.UTC
		}
		appLocation = loc
	})
	return appLocation
}

// DateIn returns the UTC midnight of the calendar date that t falls on when
// viewed in `loc`. The result is at UTC midnight (matching how GORM stores /
// reads DATE columns) so it composes with TruncateToDate-using callers.
//
// Use this directly in tests when pinning behavior to a specific instant +
// zone. Production code should call Today() instead.
func DateIn(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// Today returns the UTC midnight of the current calendar date in the
// application's configured timezone (see AppLocation).
//
// Every "what date is it today" decision in the codebase MUST go through
// this helper. Reading time.Now() (or time.Now().UTC()) directly produces
// the server's clock-day, which can disagree with the user's calendar day
// in the late-evening window — for a Berlin user, 23:30 UTC is already
// the next calendar day. The amend-mode threshold, list-default
// active_on, attendance auto-date, and the future-birthdate guard all
// share this rule, so a mismatched "today" surfaces as off-by-one in any
// of them.
func Today() time.Time {
	nowMu.RLock()
	now := nowFn()
	nowMu.RUnlock()
	return DateIn(now, AppLocation())
}

// SetNow overrides the time source used by Today() for the duration of a
// test. The returned function restores the previous source — typical use:
//
//	defer models.SetNow(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC))()
//
// or, with t.Cleanup:
//
//	t.Cleanup(models.SetNow(myInstant))
//
// Pin behavior, don't pin output: SetNow takes an *instant*, so Today()
// still routes through DateIn(...) and AppLocation(), preserving the
// timezone semantics that production code depends on.
//
// This seam is the answer to CLAUDE.md's "always use models.Today()" rule:
// production code stays free of clock injection while tests can deterministically
// pin "today" without resorting to per-call function injection or string-formatted
// date comparisons.
func SetNow(now time.Time) func() {
	nowMu.Lock()
	prev := nowFn
	nowFn = func() time.Time { return now }
	nowMu.Unlock()
	return func() {
		nowMu.Lock()
		nowFn = prev
		nowMu.Unlock()
	}
}
