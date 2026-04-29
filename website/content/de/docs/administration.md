---
title: Administrationsleitfaden
weight: 5
---

Dieser Leitfaden behandelt administrative Aufgaben in KitaManager: die Verwaltung von Organisationen, Benutzern, Rollen, Landesförderungskonfigurationen und Vergütungsplänen. Für die meisten hier beschriebenen Aktionen benötigen Sie Admin- oder Superadmin-Zugang.

## Organisationen verwalten

Organisationen repräsentieren einzelne Kindertagesstätten (Kitas). Jede Organisation ist ein separater Datenbereich -- Kinder, Personal, Verträge und andere Datensätze gehören zu genau einer Organisation.

### Organisation erstellen

Nur **Superadmins** können Organisationen erstellen. Beim Erstellen einer Organisation müssen folgende Angaben gemacht werden:

- **Name** -- der Anzeigename der Kindertagesstätte (z. B. "Kita Sonnenschein")
- **Bundesland** -- das Bundesland, in dem sich die Organisation befindet

Das Bundesland bestimmt, welche staatlichen Förderungsrichtlinien gelten. Unterstützte Bundesländer sind unter anderem Berlin, Brandenburg, Bayern und weitere.

### Organisation bearbeiten

Admins und Superadmins können Organisationsdetails wie Name und Bundesland aktualisieren.

### Organisation löschen

Nur **Superadmins** können Organisationen löschen. Das Löschen einer Organisation entfernt alle zugehörigen Daten (Kinder, Personal, Verträge usw.).

## Benutzerverwaltung

Die Benutzerverwaltung steht Admins und Superadmins zur Verfügung.

### Benutzer erstellen

Beim Erstellen eines Benutzers sind folgende Angaben erforderlich:

- **Name** -- der Anzeigename des Benutzers
- **E-Mail** -- wird für die Anmeldung verwendet, muss eindeutig sein
- **Passwort** -- muss die Mindestlängenanforderungen erfüllen
- **Aktiv** -- ob das Konto aktiviert ist

### Benutzer auflisten

Admins können alle Benutzer einsehen. Die Benutzerliste unterstützt Paginierung und zeigt Name, E-Mail und Aktivstatus jedes Benutzers an.

### Benutzer bearbeiten und löschen

Admins können Benutzerdetails (Name, E-Mail, Aktivstatus) aktualisieren und Benutzerkonten löschen. Das Löschen eines Benutzers entfernt dessen Rollenzuweisungen und Zugriffsrechte.

### Passwörter zurücksetzen

Admins können das Passwort eines Benutzers über die Benutzerverwaltungsoberfläche zurücksetzen. Bei einem Reset werden alle anderen Sitzungen des Benutzers beendet.

### Superadmin-Status

Nur bestehende Superadmins können den Superadmin-Status anderen Benutzern erteilen oder entziehen. Dies erfolgt über eine eigene Umschaltfunktion in der Benutzerverwaltungsoberfläche.

### Zwei-Faktor-Authentifizierung

Jede Nutzerin und jeder Nutzer registriert eigene Faktoren über die [Einstellungsseite](../user-guide/#zwei-faktor-authentifizierung-2fa). Administratorinnen und Administratoren können **keine** Faktoren stellvertretend für andere registrieren oder deaktivieren.

Hat ein Benutzer den Zugriff auf Authenticator-App und Wiederherstellungscodes verloren, ist die heutige Wiederherstellung: Passwort des Benutzers zurücksetzen (deaktiviert 2FA **nicht**) und den Benutzer bitten, einen Wiederherstellungscode zu nutzen -- oder, falls auch diese verloren sind, das Konto löschen und neu anlegen. Ein direkter Admin-Reset der MFA-Faktoren ist im Roadmap; siehe SECURITY-TODO im Repository.

## Rollenbasierte Zugriffskontrolle

KitaManager verwendet fünf Rollen zur Zugriffskontrolle. Jede Rolle verfügt über definierte Berechtigungen, die bestimmen, was eine Nutzerin oder ein Nutzer ausführen darf.

### Rollenübersicht

- **Superadmin** -- globaler Systemadministrator mit vollem Zugriff auf alle Organisationen. Kann Organisationen erstellen und löschen, Landesförderungskonfigurationen verwalten und alle Operationen durchführen.
- **Admin** -- volle Kontrolle innerhalb zugewiesener Organisationen. Kann Personal, Kinder, Verträge, Bereiche, Vergütungspläne und Benutzer verwalten. Kann keine Organisationen erstellen oder löschen und keine Landesförderungskonfigurationen verwalten.
- **Manager** -- erledigt tägliche operative Aufgaben innerhalb zugewiesener Organisationen. Kann Personal, Kinder und Verträge verwalten. Hat Lesezugriff auf Benutzer, Bereiche und Vergütungspläne.
- **Mitglied** -- Lesezugriff innerhalb zugewiesener Organisationen. Kann Personal, Kinder, Verträge, Bereiche und Vergütungspläne einsehen, aber nichts ändern.
- **Personal** -- konzipiert für Erzieher/innen und Assistenzkräfte, die Anwesenheiten erfassen müssen. Kann Kinder, Kinderverträge und Bereiche einsehen. Hat vollen Lese-/Schreibzugriff ausschließlich auf Anwesenheitsdaten.

### Berechtigungsmatrix

| Ressource | Superadmin | Admin | Manager | Mitglied | Personal |
|-----------|-----------|-------|---------|----------|----------|
| Organisationen | CRUD | Lesen/Aktualisieren | Lesen | Lesen | Lesen |
| Personal | CRUD | CRUD | CRUD | Lesen | -- |
| Kinder | CRUD | CRUD | CRUD | Lesen | Lesen |
| Verträge | CRUD | CRUD | CRUD | Lesen | Lesen (nur Kind) |
| Anwesenheit | CRUD | CRUD | CRUD | Lesen | CRUD |
| Bereiche | CRUD | CRUD | Lesen | Lesen | Lesen |
| Landesförderung | CRUD | -- | -- | -- | -- |
| Vergütungspläne | CRUD | CRUD | Lesen | Lesen | -- |
| Budget | CRUD | CRUD | Lesen | Lesen | -- |
| Statistiken | Lesen | Lesen | Lesen | Lesen | -- |
| Benutzer | CRUD | CRUD | Lesen | -- | -- |
| ISBJ-Abrechnungen | Erstellen/Lesen/Löschen | Erstellen/Lesen/Löschen | Erstellen/Lesen/Löschen | -- | -- |

**Geltungsbereich:** Superadmins agieren organisationsübergreifend. Alle anderen Rollen sind auf ihre zugewiesenen Organisationen beschränkt.

## Organisationsmitgliedschaft

Benutzer werden Organisationen mit einer bestimmten Rolle zugewiesen. Dies bestimmt, worauf sie zugreifen können und in welchem Bereich.

### Wichtige Konzepte

- Eine Nutzerin oder ein Nutzer kann **mehreren Organisationen** mit unterschiedlichen Rollen angehören. Beispielsweise kann jemand in einer Kita Admin und in einer anderen Manager sein.
- Rollenzuweisungen werden über die Benutzerverwaltungsoberfläche verwaltet. Admins können Organisationsmitgliedschaften für Benutzer innerhalb ihrer eigenen Organisationen hinzufügen oder entfernen.
- Superadmins können Mitgliedschaften in allen Organisationen verwalten.

### Rolle zuweisen

Um einen Benutzer einer Organisation zuzuweisen, wählen Sie ihn in der Benutzerverwaltungsoberfläche aus, wählen die Zielorganisation und weisen die gewünschte Rolle zu (Admin, Manager, Mitglied oder Personal).

### Mitgliedschaft entfernen

Das Entfernen der Mitgliedschaft eines Benutzers aus einer Organisation widerruft dessen Zugriff auf die Daten dieser Organisation. Das Benutzerkonto selbst wird nicht gelöscht.

## Landesförderung konfigurieren

Die Konfiguration der Landesförderung ist eine **Superadmin-exklusive** Operation. Sie definiert die Förderungssätze, die staatliche Stellen für die Kinderbetreuung basierend auf den Landesvorschriften zahlen.

### Struktur

Eine Landesförderungskonfiguration besteht aus:

1. **Förderungskonfiguration** -- ein übergeordneter Eintrag mit einem Namen und dem zugehörigen Bundesland
2. **Zeiträume** -- Datumsbereiche (von/bis) innerhalb einer Konfiguration, jeweils mit einem Wert für die wöchentliche Vollzeitstundenzahl
3. **Eigenschaften** -- einzelne Förderungssatzeinträge innerhalb eines Zeitraums

### Eigenschaften

Jede Eigenschaft definiert einen bestimmten Förderungssatz mit folgenden Feldern:

| Feld | Beschreibung | Beispiel |
|------|-------------|---------|
| Key | Kategoriebezeichner | `care_type` |
| Value | Spezifischer Wert innerhalb der Kategorie | `ganztag` |
| Label | Menschenlesbare Beschreibung | "Ganztagsbetreuung" |
| Payment | Betrag in Cent | `166847` (= 1.668,47 EUR) |
| Min Age | Mindestalter des Kindes (Monate) | `0` |
| Max Age | Höchstalter des Kindes (Monate) | `36` |
| Apply to All | Ob dieser Satz universell gilt | `true` / `false` |

{{% callout type="info" %}}
Alle Geldbeträge werden als Ganzzahlen in Cent gespeichert, um Gleitkomma-Präzisionsfehler zu vermeiden. Beispielsweise wird 1.668,47 EUR als `166847` gespeichert.
{{% /callout %}}

### Förderungssätze importieren

Förderungssätze können aus YAML-Dateien importiert werden. Dies ist nützlich für das massenhafte Laden offizieller staatlicher Förderungstabellen. Das YAML-Format definiert die vollständige Konfiguration einschließlich Zeiträume und Eigenschaften.

## Vergütungspläne konfigurieren

Vergütungspläne definieren Gehaltsstrukturen für das Personal, typischerweise nach Tarifverträgen wie TVöD-SuE.

### Struktur

Ein Vergütungsplan besteht aus:

1. **Vergütungsplan** -- ein benannter Plan (z. B. "TVöD-SuE"), der einer Organisation zugeordnet ist
2. **Zeiträume** -- Datumsbereiche mit zugehörigen Wochenstunden und Arbeitgeberbeitragssatz
3. **Einträge** -- einzelne Gehaltseinträge innerhalb eines Zeitraums

### Zeiträume

Jeder Zeitraum definiert:

| Feld | Beschreibung | Beispiel |
|------|-------------|---------|
| Von | Startdatum | 2025-01-01 |
| Bis | Enddatum | 2025-12-31 |
| Wochenstunden | Reguläre wöchentliche Arbeitszeit | 39,0 |
| Arbeitgeberbeitragssatz | Satz in Hundertstel Prozent | `2050` (= 20,50%) |

### Einträge

Jeder Eintrag innerhalb eines Zeitraums definiert:

| Feld | Beschreibung | Beispiel |
|------|-------------|---------|
| Entgeltgruppe | Vergütungsstufe | `S8a` |
| Stufe | Erfahrungsstufe (1--6) | `3` |
| Monatsbetrag | Gehalt in Cent | `385000` (= 3.850,00 EUR) |
| Mindestjahre | Mindestberufserfahrung für diese Stufe | `5` |

### Import und Export

Vergütungspläne können aus YAML-Dateien importiert und in YAML-Dateien exportiert werden. Dies vereinfacht die Einrichtung standardisierter Gehaltsstrukturen und deren Weitergabe an andere Organisationen.

## Audit-Protokollierung

Alle Erstell-, Aktualisierungs- und Löschvorgänge in KitaManager werden im Audit-Protokoll erfasst. Dies unterstützt Compliance-Anforderungen und ermöglicht die Nachverfolgung, wer was und wann geändert hat.

### Protokollierte Informationen

Jeder Audit-Protokolleintrag enthält:

| Feld | Beschreibung |
|------|-------------|
| Akteur | Der Benutzer, der die Aktion durchgeführt hat |
| Aktion | Aktionsname (z. B. `child_create`, `section_delete`) |
| Ressourcentyp | Der Typ der betroffenen Ressource (z. B. Personal, Kind, Vertrag) |
| Ressourcen-ID | Die Datenbank-ID der betroffenen Ressource |
| Ressourcenname | Ein menschenlesbarer Name der betroffenen Ressource |
| IP-Adresse | Die IP-Adresse, von der die Aktion ausgeführt wurde |
| Ergebnis | Ob die Aktion erfolgreich war oder fehlschlug |
| Zeitstempel | Wann die Aktion stattfand |

Audit-Protokolle sind nur lesbar -- sie können in der Oberfläche weder geändert noch gelöscht werden.

### Audit-Protokolle einsehen

Der Zugriff auf Audit-Protokolle ist in zwei Bereiche aufgeteilt:

- **Org-Admins** sehen alle Änderungen in ihrer Organisation unter **Einstellungen → Protokoll** in der Seitenleiste (URL: `/organizations/{orgId}/audit-logs`). Anmelde- und Passwortereignisse werden in dieser Ansicht bewusst ausgeblendet, da sie organisationsübergreifend sensibel sind.
- **Superadmins** sehen über die API ein globales Protokoll inklusive Anmelde- und Authentifizierungsereignissen (`GET /api/v1/audit-logs`). Eine eigene UI für die globale Sicht gibt es derzeit nicht; Superadmins fragen sie direkt ab.

Beide Sichten unterstützen Filter nach Zeitraum und nach Aktionsname (Teilstring).

## Testdaten

Entwicklungs- und Testumgebungen können mit Beispieldaten befüllt werden, um die Einrichtung und das Testen zu erleichtern.

### Was wird befüllt

- Eine Beispielorganisation ("Kita Sonnenschein")
- Testkinder mit Verträgen
- Beispielpersonal
- Berliner Landesförderungskonfiguration

### Testdaten ausführen

Verwenden Sie das Makefile-Target, um die Datenbank zu befüllen:

```bash
make seed
```

Alternativ kann der Seeding-API-Endpunkt im Entwicklungsmodus direkt aufgerufen werden.

{{% callout type="warning" %}}
Testdaten sind ausschließlich für Entwicklungs- und Testumgebungen vorgesehen. Führen Sie das Seeding nicht auf Produktionsdatenbanken aus.
{{% /callout %}}

## Nächste Schritte

- [Erste Schritte](../getting-started/) -- Anwendung einrichten
- [Architekturübersicht](../architecture/) -- Systemdesign verstehen
- [API-Referenz](../api/) -- REST API erkunden
