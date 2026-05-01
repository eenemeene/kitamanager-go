---
title: Soft-gelöschte Nutzer:in oder Organisation wiederherstellen
weight: 8
---

Sie haben eine Nutzer:in (oder Organisation) gelöscht und brauchen sie zurück. Möglich — KitaManager **soft-deleted** Nutzer:innen und Organisationen, die Zeile ist also noch in der Datenbank, nur ausgeblendet.

{{< callout type="warning" >}}
Es gibt heute **keine UI und keinen API-Endpunkt** für Wiederherstellung. Dieses Rezept erfordert direkten Datenbankzugriff. Die in einigen Kommentaren erwähnte „Admin-Trash-View" ist geplant, aber noch nicht umgesetzt.
{{< /callout >}}

## Zeile finden

Mit der Postgres-Datenbank verbinden (Docker Compose: `docker compose exec -it postgres psql -U $DB_USER $DB_NAME`).

```sql
-- Eine soft-gelöschte Nutzer:in finden
SELECT id, name, email, deleted_at FROM users WHERE deleted_at IS NOT NULL;

-- Eine soft-gelöschte Organisation finden
SELECT id, name, deleted_at FROM organizations WHERE deleted_at IS NOT NULL;
```

## Zeile wiederherstellen

```sql
UPDATE users SET deleted_at = NULL WHERE id = <ID>;
-- oder
UPDATE organizations SET deleted_at = NULL WHERE id = <ID>;
```

Die Nutzer:in kann sich sofort wieder anmelden. Die Organisation erscheint wieder bei jedem Mitglied.

## Was ist mit anderen Tabellen?

Nur `users` und `organizations` sind soft-deleted. Kinder, Mitarbeiter, Verträge, Bereiche, Abrechnungen — alle hard-deleten beim `DELETE`. **Wenn Sie eines davon versehentlich gelöscht haben, ist die einzige Wiederherstellung ein Datenbank-Restore aus dem Backup.** Siehe [Datenbank sichern und wiederherstellen](../back-up-and-restore/).

## Warum noch keine Admin-Oberfläche

Das Muster wurde in Migration 000015 hinzugefügt, um eine zukünftige Admin-Trash-View zu unterstützen; API + UI sind noch nicht gebaut. Wenn Ihr Team häufig gelöschte Nutzer:innen wiederherstellt, priorisieren Sie das Feature.

## Hinweise

- Der Audit-Log dokumentiert das ursprüngliche Lösch-Ereignis. Nach der Wiederherstellung optional eine manuelle Notiz schreiben (z. B. ein weiterer Audit-Eintrag oder ein Kommentar im Ticket-Tracker), warum die Wiederherstellung erfolgte.
- Wiederherstellen einer Nutzer:in stellt nicht ihre Organisationsmitgliedschaften wieder her — diese liegen in einer separaten Tabelle, die hart löscht. Mitgliedschaften erneut über [Rolle in einer Organisation zuweisen](../../administer/assign-role-in-organization/) hinzufügen.
