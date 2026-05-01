---
title: Nutzer:in bearbeiten oder deaktivieren
weight: 3
---

Sie wollen Name oder E-Mail einer Nutzer:in aktualisieren oder das Anmelden verhindern, ohne den Datensatz zu löschen.

## Name oder E-Mail bearbeiten

1. Öffnen Sie **Einstellungen** → **Nutzer** → klicken Sie die Nutzer:in an.
2. Aktualisieren Sie **Name** und/oder **E-Mail** und klicken Sie auf **Speichern**.

Die Änderung wird im Audit-Log dokumentiert.

## Nutzer:in deaktivieren

Häkchen bei **Aktiv** entfernen und speichern. Eine inaktive Nutzer:in kann sich nicht anmelden. Bestehende Daten (Audit-Log-Einträge, von ihr erstellte Verträge) bleiben erhalten.

## Nutzer:in löschen

Klicken Sie auf der Detailseite auf **Löschen**. Die Organisationsmitgliedschaften werden entfernt, und die Person kann sich nicht mehr anmelden. Der Nutzer-Datensatz wird **soft-deleted**: ausgeblendet aus normalen Lesevorgängen, aber noch in der Datenbank, daher ohne Backup wiederherstellbar.

## Hinweise

- Bevorzugen Sie **deaktivieren** gegenüber **löschen**, wenn jemand geht: Deaktivierung erhält den Audit-Trail und die Möglichkeit zur Reaktivierung ohne DB-Zugriff.
- Wiederherstellung einer gelöschten Person ist möglich, erfordert heute aber direkten Datenbankzugriff — siehe [Soft-gelöschte Nutzer:in oder Organisation wiederherstellen](../../operate/restore-a-soft-deleted-user-or-organization/). Eine Admin-Oberfläche dafür gibt es noch nicht.
