---
title: Warum Nutzer:innen und Organisationen soft-gelöscht sind
weight: 2
---

Migration 000015 machte die Tabellen `users` und `organizations` soft-deleted: ein `DELETE` auf Anwendungs-Ebene stempelt `deleted_at` statt die Zeile physisch zu entfernen. Drei Gründe:

1. **Audit-Trail-Erhaltung.** Audit-Log-Einträge referenzieren Nutzer:innen über ID. Verschwände der Datensatz, würden die Einträge entweder hängen oder müssten umgeschrieben werden.
2. **Reversibilität.** „Hoppla, das Konto war wichtig“ braucht ein Undo ohne Backup-Restore.
3. **Kontrollierte DSGVO-Art.-17-Löschung.** Echte Löschung hat spezifische Anforderungen (Freitextfelder gelöscht). Ein dedizierter `HardDelete`-Codepfad ist sicherer als ein Default-Delete, das die Anforderungen gelegentlich erfüllt.

Admin-Wiederherstellung erfolgt heute nur direkt-DB — siehe [Soft-gelöschte Nutzer:in oder Organisation wiederherstellen](../../../how-to/operate/restore-a-soft-deleted-user-or-organization/).

## Asymmetrie zwischen Tabellen

Nur `users` und `organizations` werden tombstoned. Kinder, Mitarbeiter, Verträge, Bereiche, Bescheide, Audit-Log-Einträge hard-deleten beim `DELETE`. Identitätstragende Zeilen brauchen den Tombstone; Datensatz-Zeilen können physisch entfernt werden, ohne den Audit-Trail zu brechen (der unabhängig erhalten wird).

## Mitwirkenden-Regel

Für die Regel zum Schreiben von Queries, die das Soft-Delete-Invariant respektieren (Auto-Scoping vs. JOIN'ed Tabellen), siehe [Datenbank-Migration hinzufügen](../../../how-to/develop/add-a-database-migration/) und `.claude/rules/database.md`.
