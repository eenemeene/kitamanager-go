package store

import (
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// validIdentifierRe matches valid SQL identifiers: bare (from_date),
// table-qualified (child_contracts.from_date), or double-quoted ("from").
var validIdentifierRe = regexp.MustCompile(`^("[a-z_][a-z0-9_]*"|[a-z_][a-z0-9_]*)(\.[a-z_][a-z0-9_]*)?$`)

// mustBeIdentifier panics if s is not a valid SQL identifier.
// All callers pass hardcoded string literals, so a panic signals a programming error.
func mustBeIdentifier(s string) {
	if !validIdentifierRe.MatchString(s) {
		panic("invalid SQL identifier: " + s)
	}
}

// PeriodActiveOn returns a GORM scope filtering period-based records to those
// active on the given date: fromCol <= date AND (toCol IS NULL OR toCol >= date).
func PeriodActiveOn(fromCol, toCol string, date time.Time) func(*gorm.DB) *gorm.DB {
	mustBeIdentifier(fromCol)
	mustBeIdentifier(toCol)
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Where(fromCol+" <= ?", date).
			Where(toCol+" IS NULL OR "+toCol+" >= ?", date)
	}
}

// NameSearch returns a GORM scope filtering records by a single column (e.g., name, email).
// The search term is matched case-insensitively using LOWER()+LIKE.
// The tablePrefix should be the table name (e.g., "sections", "groups").
func NameSearch(tablePrefix, column, search string) func(*gorm.DB) *gorm.DB {
	mustBeIdentifier(tablePrefix)
	mustBeIdentifier(column)
	return func(db *gorm.DB) *gorm.DB {
		pattern := "%" + strings.ToLower(search) + "%"
		return db.Where("LOWER("+tablePrefix+"."+column+") LIKE ?", pattern)
	}
}

// PersonNameSearch returns a GORM scope filtering person-based records by first/last name.
// The search term is matched case-insensitively against both first_name and last_name.
// The tablePrefix should be the table name (e.g., "children", "employees").
func PersonNameSearch(tablePrefix, search string) func(*gorm.DB) *gorm.DB {
	mustBeIdentifier(tablePrefix)
	return func(db *gorm.DB) *gorm.DB {
		pattern := "%" + strings.ToLower(search) + "%"
		return db.Where(
			"LOWER("+tablePrefix+".first_name) LIKE ? OR LOWER("+tablePrefix+".last_name) LIKE ?",
			pattern, pattern,
		)
	}
}
