package store

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("record not found")

// ErrDuplicateKey is returned when a unique constraint is violated.
var ErrDuplicateKey = errors.New("duplicate key")

// WrapNotFound converts gorm.ErrRecordNotFound to ErrNotFound for consistent error handling.
// Other errors are returned unchanged.
func WrapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

// IsDuplicateKeyError checks if the error is a unique constraint violation.
// Works with both PostgreSQL and SQLite.
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || // PostgreSQL
		strings.Contains(msg, "UNIQUE constraint failed") // SQLite
}
