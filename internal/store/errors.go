package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
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

// IsDuplicateKeyError checks if the error is a PostgreSQL unique constraint violation (23505).
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsForeignKeyViolation checks if the error is a PostgreSQL foreign key violation (23503).
// Returned when a delete is blocked by a referencing row, or an insert/update points at
// a non-existent parent.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// IsExclusionViolation checks if the error is a PostgreSQL exclusion constraint
// violation (23P01). Raised when an EXCLUDE constraint (e.g., the GiST overlap
// guard on child_contracts / employee_contracts) detects a conflict. With
// DEFERRABLE INITIALLY DEFERRED constraints this surfaces at COMMIT time rather
// than on the offending statement, so callers must inspect the error returned
// from transactor.InTransaction, not just statement results.
func IsExclusionViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23P01"
}
