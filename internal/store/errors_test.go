package store

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// TestPGErrorClassifiers covers the small wrappers around pgconn.PgError.Code
// in one table. They are trivial today but easy to break by accident
// (wrong code string, wrong errors.As target type) and are load-bearing
// for the service-layer error → HTTP-status mapping.
func TestPGErrorClassifiers(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		dup      bool
		fk       bool
		excl     bool
		deadlock bool
	}{
		{"nil", nil, false, false, false, false},
		{"plain stdlib error", errors.New("oops"), false, false, false, false},
		{"gorm record-not-found", gorm.ErrRecordNotFound, false, false, false, false},
		{"unique violation 23505", &pgconn.PgError{Code: "23505"}, true, false, false, false},
		{"foreign-key violation 23503", &pgconn.PgError{Code: "23503"}, false, true, false, false},
		{"exclusion violation 23P01", &pgconn.PgError{Code: "23P01"}, false, false, true, false},
		{"deadlock 40P01", &pgconn.PgError{Code: "40P01"}, false, false, false, true},
		{"unrelated pg error 42P01", &pgconn.PgError{Code: "42P01"}, false, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsDuplicateKeyError(tc.err); got != tc.dup {
				t.Errorf("IsDuplicateKeyError = %v, want %v", got, tc.dup)
			}
			if got := IsForeignKeyViolation(tc.err); got != tc.fk {
				t.Errorf("IsForeignKeyViolation = %v, want %v", got, tc.fk)
			}
			if got := IsExclusionViolation(tc.err); got != tc.excl {
				t.Errorf("IsExclusionViolation = %v, want %v", got, tc.excl)
			}
			if got := IsDeadlock(tc.err); got != tc.deadlock {
				t.Errorf("IsDeadlock = %v, want %v", got, tc.deadlock)
			}
		})
	}
}

// TestWrapNotFound verifies the only non-classifier helper in this file:
// gorm.ErrRecordNotFound becomes ErrNotFound; everything else passes through.
func TestWrapNotFound(t *testing.T) {
	if got := WrapNotFound(nil); got != nil {
		t.Errorf("nil should pass through, got %v", got)
	}
	if got := WrapNotFound(gorm.ErrRecordNotFound); !errors.Is(got, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", got)
	}
	other := errors.New("some other error")
	if got := WrapNotFound(other); got != other {
		t.Errorf("other errors should pass through unchanged, got %v", got)
	}
}
