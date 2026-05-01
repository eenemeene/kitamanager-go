---
title: Add a database migration
weight: 3
---

You want to change the schema. Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate) and live in `internal/database/migrations/`.

## Steps

### 1. Create the migration files

```bash
# From the repo root
NEXT=$(ls internal/database/migrations | grep -E '^[0-9]+' | sort -n | tail -1 | cut -d_ -f1 | awk '{printf "%06d\n", $1+1}')
touch internal/database/migrations/${NEXT}_<short_description>.up.sql
touch internal/database/migrations/${NEXT}_<short_description>.down.sql
```

### 2. Write the up SQL

In `${NEXT}_<short_description>.up.sql`:

```sql
ALTER TABLE children ADD COLUMN allergy_notes TEXT;
```

Stick to plain SQL. Don't use ORM features here — the migration runs without GORM.

### 3. Write the down SQL

In `${NEXT}_<short_description>.down.sql`:

```sql
ALTER TABLE children DROP COLUMN allergy_notes;
```

The down migration must restore the prior state exactly. Migrations run automatically inside `database.Connect`, so to test a down/up cycle locally use the `migrate` CLI directly against the dev DB:

```bash
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
DB_URL="postgres://kitamanager:kitamanager@localhost:5432/kitamanager?sslmode=disable"
migrate -path internal/database/migrations -database "$DB_URL" down 1
migrate -path internal/database/migrations -database "$DB_URL" up 1
```

Or simpler: `make dev-fresh` rebuilds the DB from scratch, applying every up migration in order.

### 4. Update the GORM model

In `internal/models/children.go`:

```go
type Child struct {
    ...
    AllergyNotes string `gorm:"type:text" json:"allergy_notes,omitempty" example:"Peanuts"`
}
```

### 5. Regenerate the schema docs

```bash
tbls doc --force postgres://kitamanager:kitamanager@localhost:5432/kitamanager_dev docs/schema
```

Commit the updated `docs/schema/` files.

### 6. Run the test suite

```bash
make api-test-integration
```

The integration suite spins up Postgres, runs migrations from scratch, and exercises the schema. If your migration breaks anything, this catches it.

## Soft-delete considerations

If your new column or query references `users` or `organizations` (the two soft-deleted tables), the **raw-query rule** applies. GORM auto-scopes the primary model in a query — `db.First(&User{}, id)` adds `WHERE deleted_at IS NULL` for you — but **it does not auto-scope JOINed tables**:

```go
// BAD — soft-deleted users still authenticate
db.Table("sessions").Joins("JOIN users ON users.id = sessions.user_id").
   Where("sessions.id = ?", idHash).Take(&row)

// GOOD — explicit filter via the helper
q := db.Table("sessions").Joins("JOIN users ON users.id = sessions.user_id").
   Where("sessions.id = ?", idHash)
err := store.ExcludeSoftDeletedUsers(q).Take(&row).Error
```

Helpers live at `internal/store/scoping.go` (`ExcludeSoftDeletedUsers`, `ExcludeSoftDeletedOrganizations`). Use `db.Unscoped()` only for admin trash-view endpoints, `HardDelete` methods, and `FindByIDUnscoped`. Never in a default read path.

For the design rationale, see [Architecture: Soft-delete](../../../explanation/architecture/#soft-delete-for-users-and-organisations). The full rule is in `.claude/rules/database.md`.

## Notes

- **Never** rely on GORM `AutoMigrate` for production schema changes — write a real migration file. AutoMigrate doesn't preserve column order, doesn't drop columns, and won't generate a down path.
- **Never** edit a migration after it's been merged. Add a new one.
- Migration files are numbered with a six-digit prefix to keep ordering stable.
- The full database conventions are in `.claude/rules/database.md`.
