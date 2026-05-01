---
title: Rolle in einer Organisation zuweisen
weight: 5
---

Sie wollen einer Nutzer:in Zugang zu Ihrer Organisation geben oder ihre Rolle ändern.

## Schritte

1. Öffnen Sie **Einstellungen** → **Nutzer** → klicken Sie die Nutzer:in an.
2. Im Abschnitt **Organisationsmitgliedschaften** klicken Sie auf **Organisation hinzufügen** (für eine neue Mitgliedschaft) oder auf eine bestehende Organisationszeile, um die Rolle zu ändern.
3. Wählen Sie die Organisation und die Rolle: `admin`, `manager`, `member` oder `staff`. (`superadmin` ist global, nicht org-bezogen — siehe unten.)
4. Klicken Sie auf **Speichern**.

Die Berechtigungen aktualisieren sich sofort beim nächsten Request.

## Hinweise

- Für die Rollendefinitionen siehe [Referenz: RBAC](../../../reference/rbac/).
- Eine Person kann Mitglied **mehrerer** Organisationen mit jeweils unterschiedlicher Rolle sein. Nutzen Sie das für jemanden, der zwei Kitas betreut.
- **Superadmin** zu vergeben ist eine separate, dedizierte Funktion auf der Nutzer-Detailseite (nur bestehende Superadmins können sie vergeben). Sie ist global, nicht pro Organisation. Siehe [RBAC-Referenz](../../../reference/rbac/) für die Privilegien des Superadmins.
- Audit-Log dokumentiert die Zuweisung.
