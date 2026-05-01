---
paths:
  - "internal/database/**/*.go"
  - "internal/models/**/*.go"
  - "internal/store/**/*.go"
---

# Database conventions

Loaded when working in the database, model, or store layers. The wider Go conventions in `go-conventions.md` also apply.

## Migrations

Schema changes need a numbered migration file (golang-migrate). Never rely solely on GORM `AutoMigrate` for production changes — it doesn't preserve column order, drop columns, or generate the down-migration.

After any schema change, regenerate the diagram:
```bash
tbls doc --force postgres://user:pass@localhost:5432/kitamanager docs/schema
```

Install: `go install github.com/k1LoW/tbls@latest`. Settings: `.tbls.yml`.

## Soft-delete: the raw-query rule

Migration 000015 made `users` and `organizations` soft-deleted. GORM auto-scopes the **primary** model, but **not joined tables**.

Any hand-written query (`.Table()`, `.Joins()`, `.Raw()`) that references `users` or `organizations` as a JOINed entity MUST filter out tombstones explicitly. Use the helpers in `internal/store/scoping.go`:

```go
// GOOD — raw JOIN through users with explicit filter
q := db.Table("sessions").Joins("JOIN users ON users.id = sessions.user_id").Where(...)
err := store.ExcludeSoftDeletedUsers(q).Take(&row).Error

// GOOD — raw JOIN through organizations
err := store.ExcludeSoftDeletedOrganizations(q).Take(&row).Error

// BAD — forgotten filter; soft-deleted users would still authenticate
db.Table("sessions").Joins("JOIN users ...").Where("sessions.id = ?", idHash).Take(...)
```

Queries that **start** from a GORM model (`db.First(&User{}, id)`, `db.Model(&User{}).Joins(...)`) auto-scope and don't need the helper.

### When `.Unscoped()` is correct

- Admin "trash view" endpoints that list tombstoned rows
- `HardDelete` methods (Art. 17 erasure flow, retention TTL purge)
- `FindByIDUnscoped` when a purge target may already be tombstoned

**Never** call `.Unscoped()` in a default read path.

## Currency reminder

All monetary fields are `int` cents — see the cross-cutting rule in the top-level CLAUDE.md.
