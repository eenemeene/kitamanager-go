package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// ErrPeriodOverlap is returned when a period record would overlap with an existing one.
var ErrPeriodOverlap = errors.New("period would overlap with existing record")

// PeriodStore provides common queries for time-bounded records.
type PeriodStore[T models.PeriodRecord] struct {
	db         *gorm.DB
	ownerIDCol string // "employee_id", "child_id", or "cost_id"
}

// NewPeriodStore creates a new store for time-bounded records.
func NewPeriodStore[T models.PeriodRecord](db *gorm.DB, ownerIDCol string) *PeriodStore[T] {
	return &PeriodStore[T]{db: db, ownerIDCol: ownerIDCol}
}

// GetCurrentRecord returns the active record for an owner (if any).
func (s *PeriodStore[T]) GetCurrentRecord(ctx context.Context, ownerID uint) (*T, error) {
	return s.GetRecordOn(ctx, ownerID, time.Now())
}

// GetRecordOn returns the record valid on a specific date.
// Returns nil if no record exists for that date.
func (s *PeriodStore[T]) GetRecordOn(ctx context.Context, ownerID uint, date time.Time) (*T, error) {
	var record T
	err := DBFromContext(ctx, s.db).Where(
		s.ownerIDCol+" = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
		ownerID, date, date,
	).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListRecords returns all records for an owner ordered by from_date.
func (s *PeriodStore[T]) ListRecords(ctx context.Context, ownerID uint) ([]T, error) {
	var records []T
	err := DBFromContext(ctx, s.db).Where(s.ownerIDCol+" = ?", ownerID).
		Order("from_date ASC").
		Find(&records).Error
	return records, err
}

// ListRecordsPaginated returns paginated records for an owner ordered by from_date desc.
func (s *PeriodStore[T]) ListRecordsPaginated(ctx context.Context, ownerID uint, limit, offset int) ([]T, int64, error) {
	var records []T
	var total int64

	// Count total
	if err := DBFromContext(ctx, s.db).Model(new(T)).Where(s.ownerIDCol+" = ?", ownerID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results (desc for most recent first)
	err := DBFromContext(ctx, s.db).Where(s.ownerIDCol+" = ?", ownerID).
		Order("from_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	return records, total, err
}

// HasActiveRecord checks if an owner has a record on the given date.
func (s *PeriodStore[T]) HasActiveRecord(ctx context.Context, ownerID uint, date time.Time) (bool, error) {
	var count int64
	err := DBFromContext(ctx, s.db).Model(new(T)).Where(
		s.ownerIDCol+" = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
		ownerID, date, date,
	).Count(&count).Error
	return count > 0, err
}

// ValidateNoOverlap checks if a new record would overlap with existing ones.
// Use excludeID to exclude a specific record (for updates).
func (s *PeriodStore[T]) ValidateNoOverlap(ctx context.Context, ownerID uint, from time.Time, to *time.Time, excludeID *uint) error {
	query := DBFromContext(ctx, s.db).Model(new(T)).Where(s.ownerIDCol+" = ?", ownerID)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	// For inclusive ranges [from, to], overlap occurs when:
	// existing.from <= new.to AND new.from <= existing.to
	//
	// With NULL handling (NULL means ongoing/infinity):
	// - If new.to is NULL: overlaps if existing.to is NULL OR existing.to >= new.from
	// - If existing.to is NULL: overlaps if new.to is NULL OR new.to >= existing.from

	if to != nil {
		// New record has end date
		query = query.Where(
			"from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
			to, from,
		)
	} else {
		// New record is ongoing
		query = query.Where(
			"to_date IS NULL OR to_date >= ?",
			from,
		)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPeriodOverlap
	}
	return nil
}

// CloseCurrentRecord sets the end date of the current ongoing record.
func (s *PeriodStore[T]) CloseCurrentRecord(ctx context.Context, ownerID uint, endDate time.Time) error {
	return DBFromContext(ctx, s.db).Model(new(T)).
		Where(s.ownerIDCol+" = ? AND to_date IS NULL", ownerID).
		Update("to_date", endDate).Error
}
