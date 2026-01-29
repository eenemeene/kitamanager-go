package store

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// ErrContractOverlap is returned when a contract would overlap with an existing one.
var ErrContractOverlap = errors.New("contract would overlap with existing contract")

// PeriodStore provides common queries for time-bounded records.
type PeriodStore[T models.HasPeriod] struct {
	db          *gorm.DB
	personIDCol string // "employee_id" or "child_id"
}

// NewPeriodStore creates a new store for time-bounded records.
func NewPeriodStore[T models.HasPeriod](db *gorm.DB, personIDCol string) *PeriodStore[T] {
	return &PeriodStore[T]{db: db, personIDCol: personIDCol}
}

// GetCurrentContract returns the active contract for a person (if any).
func (s *PeriodStore[T]) GetCurrentContract(personID uint) (*T, error) {
	return s.GetContractOn(personID, time.Now())
}

// GetContractOn returns the contract valid on a specific date.
// Returns nil if no contract exists for that date.
func (s *PeriodStore[T]) GetContractOn(personID uint, date time.Time) (*T, error) {
	var contract T
	err := s.db.Where(
		s.personIDCol+" = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
		personID, date, date,
	).First(&contract).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &contract, nil
}

// GetHistory returns all contracts for a person ordered by from_date.
func (s *PeriodStore[T]) GetHistory(personID uint) ([]T, error) {
	var contracts []T
	err := s.db.Where(s.personIDCol+" = ?", personID).
		Order("from_date ASC").
		Find(&contracts).Error
	return contracts, err
}

// GetHistoryPaginated returns paginated contracts for a person ordered by from_date desc.
func (s *PeriodStore[T]) GetHistoryPaginated(personID uint, limit, offset int) ([]T, int64, error) {
	var contracts []T
	var total int64

	// Count total
	if err := s.db.Model(new(T)).Where(s.personIDCol+" = ?", personID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results (desc for most recent first)
	err := s.db.Where(s.personIDCol+" = ?", personID).
		Order("from_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&contracts).Error
	return contracts, total, err
}

// HasActiveContract checks if a person has a contract on the given date.
func (s *PeriodStore[T]) HasActiveContract(personID uint, date time.Time) (bool, error) {
	var count int64
	err := s.db.Model(new(T)).Where(
		s.personIDCol+" = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
		personID, date, date,
	).Count(&count).Error
	return count > 0, err
}

// ValidateNoOverlap checks if a new contract would overlap with existing ones.
// Use excludeID to exclude a specific contract (for updates).
func (s *PeriodStore[T]) ValidateNoOverlap(personID uint, from time.Time, to *time.Time, excludeID *uint) error {
	query := s.db.Model(new(T)).Where(s.personIDCol+" = ?", personID)

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
		// New contract has end date
		query = query.Where(
			"from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
			to, from,
		)
	} else {
		// New contract is ongoing
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
		return ErrContractOverlap
	}
	return nil
}

// CloseCurrentContract sets the end date of the current ongoing contract.
func (s *PeriodStore[T]) CloseCurrentContract(personID uint, endDate time.Time) error {
	return s.db.Model(new(T)).
		Where(s.personIDCol+" = ? AND to_date IS NULL", personID).
		Update("to_date", endDate).Error
}
