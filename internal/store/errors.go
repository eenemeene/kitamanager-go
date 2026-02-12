package store

import "errors"

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("record not found")
