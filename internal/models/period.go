package models

import "time"

// Period represents a date range where both From and To are INCLUSIVE.
// A contract with From=2025-01-01 and To=2025-12-31 is active on both dates.
// If To is nil, the period is ongoing (no end date).
type Period struct {
	From time.Time  `gorm:"column:from_date;type:date;not null" json:"from" format:"date-time"`
	To   *time.Time `gorm:"column:to_date;type:date" json:"to" format:"date-time"`
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
	date = TruncateToDate(date)
	from := TruncateToDate(p.From)
	if date.Before(from) {
		return false
	}
	if p.To == nil {
		return true
	}
	to := TruncateToDate(*p.To)
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

	aFrom := TruncateToDate(p.From)
	bFrom := TruncateToDate(other.From)

	// A.From <= B.To (if B.To is nil, this is always true)
	if other.To != nil {
		bTo := TruncateToDate(*other.To)
		if aFrom.After(bTo) {
			return false
		}
	}
	// B.From <= A.To (if A.To is nil, this is always true)
	if p.To != nil {
		aTo := TruncateToDate(*p.To)
		if bFrom.After(aTo) {
			return false
		}
	}
	return true
}

// TruncateToDate truncates a time to date-only (midnight UTC).
func TruncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
