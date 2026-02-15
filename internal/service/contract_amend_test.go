package service

import (
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
)

func TestDetermineAmendMode_FromToday(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	mode, err := determineAmendMode(today, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeInPlace {
		t.Errorf("expected amendModeInPlace, got %d", mode)
	}
}

func TestDetermineAmendMode_FromTomorrow(t *testing.T) {
	tomorrow := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)
	mode, err := determineAmendMode(tomorrow, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeInPlace {
		t.Errorf("expected amendModeInPlace, got %d", mode)
	}
}

func TestDetermineAmendMode_FromYesterday(t *testing.T) {
	yesterday := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)
	mode, err := determineAmendMode(yesterday, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeAmend {
		t.Errorf("expected amendModeAmend, got %d", mode)
	}
}

func TestDetermineAmendMode_FromSixMonthsAgo(t *testing.T) {
	sixMonthsAgo := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -6, 0)
	mode, err := determineAmendMode(sixMonthsAgo, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeAmend {
		t.Errorf("expected amendModeAmend, got %d", mode)
	}
}

func TestDetermineAmendMode_EndedYesterday(t *testing.T) {
	past := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, -6, 0)
	yesterday := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)
	_, err := determineAmendMode(past, &yesterday)
	if err == nil {
		t.Fatal("expected error for ended contract, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestDetermineAmendMode_EndsToday(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	past := today.AddDate(0, -6, 0)
	// Contract ends today — still active today, should be amendModeAmend (started in the past)
	mode, err := determineAmendMode(past, &today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeAmend {
		t.Errorf("expected amendModeAmend, got %d", mode)
	}
}

func TestDetermineAmendMode_EndsTomorrow(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	tomorrow := today.AddDate(0, 0, 1)
	// Contract from today, ends tomorrow → in-place
	mode, err := determineAmendMode(today, &tomorrow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeInPlace {
		t.Errorf("expected amendModeInPlace, got %d", mode)
	}
}

func TestDetermineAmendMode_FromTodayToNil(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	mode, err := determineAmendMode(today, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeInPlace {
		t.Errorf("expected amendModeInPlace, got %d", mode)
	}
}

// Edge case: time truncation — non-midnight time should still be treated as the same date
func TestDetermineAmendMode_NonMidnightTime(t *testing.T) {
	// Contract from today at 15:30:45 — should still be "today" after truncation
	now := time.Now()
	todayWithTime := time.Date(now.Year(), now.Month(), now.Day(), 15, 30, 45, 0, time.UTC)
	mode, err := determineAmendMode(todayWithTime, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeInPlace {
		t.Errorf("expected amendModeInPlace for same day with non-midnight time, got %d", mode)
	}
}

// Edge case: same-day contract (From == To == today) — should be in-place
func TestDetermineAmendMode_SameDayContract(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	mode, err := determineAmendMode(today, &today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeInPlace {
		t.Errorf("expected amendModeInPlace for same-day contract, got %d", mode)
	}
}

// Edge case: contract ended exactly today with From in past — still active, should be amend mode
func TestDetermineAmendMode_EndsToday_FromPast(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	past := today.AddDate(0, -1, 0)
	mode, err := determineAmendMode(past, &today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != amendModeAmend {
		t.Errorf("expected amendModeAmend for contract ending today with past start, got %d", mode)
	}
}

// Edge case: contract ended long ago — should return error
func TestDetermineAmendMode_EndedLongAgo(t *testing.T) {
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err := determineAmendMode(past, &endDate)
	if err == nil {
		t.Fatal("expected error for contract that ended long ago, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}
