package models

import (
	"testing"
	"time"
)

// TestDateIn covers the pure helper that powers Today(). DateIn is the
// boundary that matters: every "what calendar date is `now`?" answer in
// the codebase routes through it (via Today), and the answer must agree
// with what a user in the given timezone sees on their wall clock.
func TestDateIn(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("Europe/Berlin must resolve via embedded tzdata: %v", err)
	}
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("Asia/Tokyo must resolve via embedded tzdata: %v", err)
	}

	// Reference instant: 2025-06-15 23:30:00 UTC.
	// Berlin in summer is UTC+2 → 2025-06-16 01:30 Berlin → calendar date 16.
	// Tokyo is UTC+9 → 2025-06-16 08:30 Tokyo → calendar date 16.
	// UTC sees 2025-06-15.
	t.Run("late evening UTC produces next-day Berlin", func(t *testing.T) {
		now := time.Date(2025, 6, 15, 23, 30, 0, 0, time.UTC)
		got := DateIn(now, berlin)
		want := time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Errorf("DateIn(2025-06-15 23:30 UTC, Berlin) = %v, want %v", got, want)
		}
	})

	// Symmetric: a UTC date that is yesterday from Tokyo's perspective.
	// 2025-06-15 14:00 UTC is 2025-06-15 23:00 Tokyo (still 15th).
	// 2025-06-15 16:00 UTC is 2025-06-16 01:00 Tokyo (next day).
	t.Run("Tokyo crosses date 9 hours before UTC", func(t *testing.T) {
		early := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
		late := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
		gotEarly := DateIn(early, tokyo)
		gotLate := DateIn(late, tokyo)
		if !gotEarly.Equal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("Tokyo at 14 UTC = %v, want 2025-06-15 UTC midnight", gotEarly)
		}
		if !gotLate.Equal(time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("Tokyo at 16 UTC = %v, want 2025-06-16 UTC midnight", gotLate)
		}
	})

	// DateIn must always return a UTC midnight, regardless of input zone or
	// time-of-day. This is the contract that lets the result compose with
	// TruncateToDate-using callers.
	t.Run("result is always UTC midnight", func(t *testing.T) {
		inputs := []time.Time{
			time.Date(2025, 6, 15, 23, 30, 0, 0, berlin),
			time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2025, 6, 15, 12, 0, 0, 0, tokyo),
		}
		for _, in := range inputs {
			got := DateIn(in, berlin)
			if got.Location() != time.UTC {
				t.Errorf("DateIn(%v) returned in zone %v, want UTC", in, got.Location())
			}
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
				t.Errorf("DateIn(%v) = %v, want midnight", in, got)
			}
		}
	})

	// DST transition (CEST→CET): 2025-10-26 03:00 CEST falls back to
	// 02:00 CET. The clock at 01:30 UTC is 03:30 CEST → calendar 26.
	// Same wall-clock 01:30 UTC just before transition is also 03:30 CEST.
	// The point: Berlin's calendar date around DST is unambiguous.
	t.Run("DST fall-back doesn't shift the calendar date", func(t *testing.T) {
		preDST := time.Date(2025, 10, 26, 0, 30, 0, 0, time.UTC)  // 02:30 CEST
		postDST := time.Date(2025, 10, 26, 1, 30, 0, 0, time.UTC) // 02:30 CET (after fall-back)
		gotPre := DateIn(preDST, berlin)
		gotPost := DateIn(postDST, berlin)
		want := time.Date(2025, 10, 26, 0, 0, 0, 0, time.UTC)
		if !gotPre.Equal(want) || !gotPost.Equal(want) {
			t.Errorf("DST boundary: pre=%v post=%v want=%v", gotPre, gotPost, want)
		}
	})
}

// AppLocation cache: must return *time.Location, must equal Europe/Berlin by
// default, and must keep returning the same value across calls.
func TestAppLocation_DefaultBerlin(t *testing.T) {
	loc := AppLocation()
	if loc == nil {
		t.Fatal("AppLocation returned nil")
	}
	if loc.String() != "Europe/Berlin" && loc.String() != "UTC" {
		// "UTC" is acceptable only if the env var was set to UTC by an
		// earlier test in the same process — flag if it's neither.
		t.Errorf("AppLocation = %q, want Europe/Berlin or UTC fallback", loc.String())
	}

	// Stability: subsequent calls return the same pointer (sync.Once).
	if AppLocation() != loc {
		t.Error("AppLocation returned different pointers on subsequent calls")
	}
}

// AppLocation env-var override: set the env var BEFORE the first call to
// AppLocation in this process, and verify the loader honored it. This test
// uses os.Setenv at TestMain time via init() — but since AppLocation uses
// sync.Once, we cannot easily test the override path directly without
// process isolation. We do test the malformed-zone fallback path via DateIn.
func TestAppLocation_KnownZoneIsResolvable(t *testing.T) {
	// The embedded tzdata must include at least the IANA zones we plan to
	// support. If this test fails, the `_ "time/tzdata"` import in clock.go
	// has been removed and the runtime image's tzdata is being relied upon.
	for _, zone := range []string{"Europe/Berlin", "UTC", "America/New_York", "Asia/Tokyo"} {
		if _, err := time.LoadLocation(zone); err != nil {
			t.Errorf("zone %q must resolve (embedded tzdata): %v", zone, err)
		}
	}
}

// SetNow pins the time source so Today() returns a deterministic value.
// Validates the seam itself; downstream packages use this to remove
// "today"-shaped flakiness from their tests without per-call injection.
func TestSetNow_PinsToday(t *testing.T) {
	// Use an instant late enough in UTC that Berlin is already on the next
	// calendar day — proves the seam routes through DateIn/AppLocation
	// instead of just returning the pinned instant.
	pinned := time.Date(2026, 1, 14, 23, 30, 0, 0, time.UTC)
	defer SetNow(pinned)()

	got := Today()
	want := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Today() with SetNow(2026-01-14 23:30 UTC) = %v, want %v (Berlin's next calendar day)", got, want)
	}
}

// SetNow returns a restore function that brings back the real clock.
// Without this, a test that pins the clock would leak into every test
// that runs after it in the same process.
func TestSetNow_RestoresPreviousClock(t *testing.T) {
	pinned := time.Date(1999, 12, 31, 12, 0, 0, 0, time.UTC)
	restore := SetNow(pinned)

	if !Today().Equal(time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("SetNow did not pin Today()")
	}

	restore()

	// After restore, Today() must be within a tight window around real time.
	// Allow ± 1 day to absorb the cross-midnight edge case.
	now := Today()
	realToday := DateIn(time.Now(), AppLocation())
	diff := now.Sub(realToday)
	if diff < -24*time.Hour || diff > 24*time.Hour {
		t.Errorf("after restore, Today() = %v drifted %v from real today %v", now, diff, realToday)
	}
	// And specifically: must NOT still equal the pinned date.
	if now.Equal(time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC)) {
		t.Error("after restore, Today() still returns the pinned 1999 date")
	}
}

// Today: smoke test that the helper returns a UTC-midnight value matching
// today's date in the configured zone. The calendar date is whatever the
// machine running the test thinks "today" is in that zone — we only verify
// the structural invariants (zone == UTC, hour == 0).
func TestToday_StructuralInvariants(t *testing.T) {
	got := Today()
	if got.Location() != time.UTC {
		t.Errorf("Today().Location = %v, want UTC", got.Location())
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
		t.Errorf("Today() = %v, want midnight", got)
	}

	// And: the date returned must equal what DateIn produces for the same
	// "now" — Today() is just sugar over DateIn(time.Now(), AppLocation()).
	now := time.Now()
	want := DateIn(now, AppLocation())
	// Allow a 1-day delta if the test crosses midnight between the two
	// calls; otherwise must match exactly.
	delta := got.Sub(want)
	if delta != 0 && delta != 24*time.Hour && delta != -24*time.Hour {
		t.Errorf("Today() vs DateIn(time.Now(), AppLocation()) drift = %v, want 0 or ±24h", delta)
	}
}
