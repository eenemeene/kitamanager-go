package store

import (
	"time"

	"gorm.io/gorm"
)

// PeriodActiveOn returns a GORM scope filtering period-based records to those
// active on the given date: fromCol <= date AND (toCol IS NULL OR toCol >= date).
func PeriodActiveOn(fromCol, toCol string, date time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Where(fromCol+" <= ?", date).
			Where(toCol+" IS NULL OR "+toCol+" >= ?", date)
	}
}
