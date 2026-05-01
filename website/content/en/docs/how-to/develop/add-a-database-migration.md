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

The down migration must restore the prior state exactly. Test it locally with `make dev-fresh && make api-build && ./bin/kitamanager-api migrate down 1 && ./bin/kitamanager-api migrate up 1`.

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

If your new column is referenced from `users` or `organizations`, read `.claude/rules/database.md`'s soft-delete section before joining against either table in raw queries. The GORM-auto-scoping rule does not apply to JOINs.

For the rationale, see [Architecture: Soft-delete](../../../explanation/architecture/#soft-delete-for-users-and-organisations).

## Notes

- **Never** rely on GORM `AutoMigrate` for production schema changes — write a real migration file. AutoMigrate doesn't preserve column order, doesn't drop columns, and won't generate a down path.
- **Never** edit a migration after it's been merged. Add a new one.
- Migration files are numbered with a six-digit prefix to keep ordering stable.
- The full database conventions are in `.claude/rules/database.md`.
