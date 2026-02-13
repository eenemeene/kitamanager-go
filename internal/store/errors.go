package store

import (
	"errors"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("record not found")

// WrapNotFound converts gorm.ErrRecordNotFound to ErrNotFound for consistent error handling.
// Other errors are returned unchanged.
func WrapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
