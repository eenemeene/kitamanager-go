package store

import "gorm.io/gorm"

// Soft-delete scoping helpers.
//
// GORM's `gorm.DeletedAt` sentinel field gives us automatic
// `deleted_at IS NULL` filtering ONLY for queries that start from the
// model in question — i.e. `db.First(&models.User{}, id)` is scoped,
// but `db.Table("sessions").Joins("JOIN users ...")` is NOT. Any
// hand-written query that references users or organizations as a
// JOINed table must apply the filter explicitly.
//
// Use these helpers rather than hard-coding the predicate so the rule
// is grep-able and the call site reads as a deliberate choice rather
// than a forgotten detail.

// ExcludeSoftDeletedUsers appends `AND users.deleted_at IS NULL` to
// the query. Call this from any raw query that references the
// `users` table via `.Table()` or `.Joins()` — GORM auto-scoping
// does not reach joined tables, only the primary model.
func ExcludeSoftDeletedUsers(q *gorm.DB) *gorm.DB {
	return q.Where("users.deleted_at IS NULL")
}

// ExcludeSoftDeletedOrganizations appends the analogous filter for
// the organizations table.
func ExcludeSoftDeletedOrganizations(q *gorm.DB) *gorm.DB {
	return q.Where("organizations.deleted_at IS NULL")
}
