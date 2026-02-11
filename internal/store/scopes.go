package store

import (
	"strings"
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

// PersonNameSearch returns a GORM scope filtering person-based records by first/last name.
// The search term is matched case-insensitively against both first_name and last_name.
// Uses LOWER()+LIKE for portability across PostgreSQL and SQLite.
// The tablePrefix should be the table name (e.g., "children", "employees").
func PersonNameSearch(tablePrefix, search string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		pattern := "%" + strings.ToLower(search) + "%"
		return db.Where(
			"LOWER("+tablePrefix+".first_name) LIKE ? OR LOWER("+tablePrefix+".last_name) LIKE ?",
			pattern, pattern,
		)
	}
}
