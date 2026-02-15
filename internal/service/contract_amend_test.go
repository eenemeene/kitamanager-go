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
