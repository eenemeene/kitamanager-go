package models

import "time"

// Period represents a date range where both From and To are INCLUSIVE.
// A contract with From=2025-01-01 and To=2025-12-31 is active on both dates.
// If To is nil, the period is ongoing (no end date).
type Period struct {
	From time.Time  `gorm:"column:from_date;type:date;not null" json:"from"`
	To   *time.Time `gorm:"column:to_date;type:date" json:"to"`
}

// PeriodRecord interface for any time-bounded record
type PeriodRecord interface {
	GetFrom() time.Time
	GetTo() *time.Time
	GetOwnerID() uint
}

// GetFrom returns the start date of the period.
func (p Period) GetFrom() time.Time {
	return p.From
}

// GetTo returns the end date of the period (nil if ongoing).
func (p Period) GetTo() *time.Time {
	return p.To
}

// IsActiveOn checks if the given date falls within the period (inclusive).
func (p Period) IsActiveOn(date time.Time) bool {
	date = truncateToDate(date)
	from := truncateToDate(p.From)
	if date.Before(from) {
		return false
	}
	if p.To == nil {
		return true
	}
	to := truncateToDate(*p.To)
	return !date.After(to) // date <= To
}

// IsOngoing returns true if the period has no end date.
func (p Period) IsOngoing() bool {
	return p.To == nil
}

// Overlaps checks if this period overlaps with another (both inclusive).
func (p Period) Overlaps(other Period) bool {
	// Two inclusive ranges [A.From, A.To] and [B.From, B.To] overlap if:
	// A.From <= B.To AND B.From <= A.To

	aFrom := truncateToDate(p.From)
	bFrom := truncateToDate(other.From)

	// A.From <= B.To (if B.To is nil, this is always true)
	if other.To != nil {
		bTo := truncateToDate(*other.To)
		if aFrom.After(bTo) {
			return false
		}
	}
	// B.From <= A.To (if A.To is nil, this is always true)
	if p.To != nil {
		aTo := truncateToDate(*p.To)
		if bFrom.After(aTo) {
			return false
		}
	}
	return true
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
