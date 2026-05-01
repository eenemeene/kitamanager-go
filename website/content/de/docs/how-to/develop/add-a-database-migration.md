---
title: Datenbank-Migration hinzufügen
weight: 3
---

Sie wollen das Schema ändern. Migrationen werden mit [golang-migrate](https://github.com/golang-migrate/migrate) verwaltet und liegen in `internal/database/migrations/`.

## Schritte

### 1. Migrationsdateien anlegen

```bash
# Im Repo-Root
NEXT=$(ls internal/database/migrations | grep -E '^[0-9]+' | sort -n | tail -1 | cut -d_ -f1 | awk '{printf "%06d\n", $1+1}')
touch internal/database/migrations/${NEXT}_<short_description>.up.sql
touch internal/database/migrations/${NEXT}_<short_description>.down.sql
```

### 2. Up-SQL schreiben

In `${NEXT}_<short_description>.up.sql`:

```sql
ALTER TABLE children ADD COLUMN allergy_notes TEXT;
```

Bei einfachem SQL bleiben. Hier keine ORM-Features — die Migration läuft ohne GORM.

### 3. Down-SQL schreiben

In `${NEXT}_<short_description>.down.sql`:

```sql
ALTER TABLE children DROP COLUMN allergy_notes;
```

Die Down-Migration muss den vorherigen Zustand exakt wiederherstellen. Migrationen laufen automatisch innerhalb von `database.Connect`. Um lokal einen Down/Up-Zyklus zu testen, das `migrate`-CLI direkt gegen die Dev-DB nutzen:

```bash
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
DB_URL="postgres://kitamanager:kitamanager@localhost:5432/kitamanager?sslmode=disable"
migrate -path internal/database/migrations -database "$DB_URL" down 1
migrate -path internal/database/migrations -database "$DB_URL" up 1
```

Oder einfacher: `make dev-fresh` baut die DB von Grund auf neu auf und wendet jede Up-Migration in Reihenfolge an.

### 4. GORM-Modell aktualisieren

In `internal/models/children.go`:

```go
type Child struct {
    ...
    AllergyNotes string `gorm:"type:text" json:"allergy_notes,omitempty" example:"Peanuts"`
}
```

### 5. Schema-Doku neu generieren

```bash
tbls doc --force postgres://kitamanager:kitamanager@localhost:5432/kitamanager_dev docs/schema
```

Die aktualisierten `docs/schema/`-Dateien mit committen.

### 6. Test-Suite ausführen

```bash
make api-test-integration
```

Die Integration-Suite zieht Postgres hoch, führt Migrationen von Grund auf aus und übt das Schema. Wenn Ihre Migration etwas bricht, fängt das es ab.

## Soft-Delete-Überlegungen

Wenn Ihre neue Spalte oder Abfrage `users` oder `organizations` (die zwei soft-deleted Tabellen) referenziert, gilt die **Raw-Query-Regel**. GORM scopt das primäre Modell automatisch — `db.First(&User{}, id)` fügt `WHERE deleted_at IS NULL` hinzu — aber **es scopt JOIN'ed Tabellen nicht**:

```go
// SCHLECHT — soft-gelöschte Nutzer:innen authentifizieren sich noch
db.Table("sessions").Joins("JOIN users ON users.id = sessions.user_id").
   Where("sessions.id = ?", idHash).Take(&row)

// GUT — expliziter Filter über die Helfer-Funktion
q := db.Table("sessions").Joins("JOIN users ON users.id = sessions.user_id").
   Where("sessions.id = ?", idHash)
err := store.ExcludeSoftDeletedUsers(q).Take(&row).Error
```

Helfer liegen in `internal/store/scoping.go` (`ExcludeSoftDeletedUsers`, `ExcludeSoftDeletedOrganizations`). Nutzen Sie `db.Unscoped()` nur für Admin-Trash-View-Endpunkte, `HardDelete`-Methoden und `FindByIDUnscoped`. Niemals in einem Standard-Lesepfad.

Für die Designentscheidung siehe [Architektur: Soft-Delete](../../../explanation/architecture/). Die vollständige Regel liegt in `.claude/rules/database.md`.

## Hinweise

- **Niemals** sich auf GORM `AutoMigrate` für produktive Schema-Änderungen verlassen — eine echte Migrationsdatei schreiben. AutoMigrate erhält keine Spalten-Reihenfolge, droppt keine Spalten, und generiert keinen Down-Pfad.
- **Niemals** eine Migration nach dem Merge bearbeiten. Eine neue hinzufügen.
- Migrationsdateien sind mit einem sechsstelligen Präfix nummeriert, um die Reihenfolge stabil zu halten.
- Die vollständigen Datenbank-Konventionen liegen in `.claude/rules/database.md`.
