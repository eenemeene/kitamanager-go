---
title: Administrationsleitfaden
weight: 5
---

Dieser Leitfaden behandelt administrative Aufgaben in KitaManager: die Verwaltung von Organisationen, Benutzern, Rollen, Landesfoerderungskonfigurationen und Verguetungsplaenen. Fuer die meisten hier beschriebenen Aktionen benoetigen Sie Admin- oder Superadmin-Zugang.

## Organisationen verwalten

Organisationen repraesentieren einzelne Kindertagesstaetten (Kitas). Jede Organisation ist ein separater Datenbereich -- Kinder, Personal, Vertraege und andere Datensaetze gehoeren zu genau einer Organisation.

### Organisation erstellen

Nur **Superadmins** koennen Organisationen erstellen. Beim Erstellen einer Organisation muessen folgende Angaben gemacht werden:

- **Name** -- der Anzeigename der Kindertagesstaette (z.B. "Kita Sonnenschein")
- **Bundesland** -- das Bundesland, in dem sich die Organisation befindet

Das Bundesland bestimmt, welche staatlichen Foerderungsrichtlinien gelten. Unterstuetzte Bundeslaender sind unter anderem Berlin, Brandenburg, Bayern und weitere.

### Organisation bearbeiten

Admins und Superadmins koennen Organisationsdetails wie Name und Bundesland aktualisieren.

### Organisation loeschen

Nur **Superadmins** koennen Organisationen loeschen. Das Loeschen einer Organisation entfernt alle zugehoerigen Daten (Kinder, Personal, Vertraege usw.).

## Benutzerverwaltung

Die Benutzerverwaltung steht Admins und Superadmins zur Verfuegung.

### Benutzer erstellen

Beim Erstellen eines Benutzers sind folgende Angaben erforderlich:

- **Name** -- der Anzeigename des Benutzers
- **E-Mail** -- wird fuer die Anmeldung verwendet, muss eindeutig sein
- **Passwort** -- muss die Mindestlaengenanforderungen erfuellen
- **Aktiv** -- ob das Konto aktiviert ist

### Benutzer auflisten

Admins koennen alle Benutzer einsehen. Die Benutzerliste unterstuetzt Paginierung und zeigt Name, E-Mail und Aktivstatus jedes Benutzers an.

### Benutzer bearbeiten und loeschen

Admins koennen Benutzerdetails (Name, E-Mail, Aktivstatus) aktualisieren und Benutzerkonten loeschen. Das Loeschen eines Benutzers entfernt dessen Rollenzuweisungen und Zugriffsrechte.

### Passwoerter zuruecksetzen

Admins koennen das Passwort eines Benutzers ueber die Benutzerverwaltungsoberflaeche zuruecksetzen.

### Superadmin-Status

Nur bestehende Superadmins koennen den Superadmin-Status anderen Benutzern erteilen oder entziehen. Dies erfolgt ueber eine eigene Umschaltfunktion in der Benutzerverwaltungsoberflaeche.

## Rollenbasierte Zugriffskontrolle

KitaManager verwendet fuenf Rollen zur Zugriffskontrolle. Jede Rolle verfuegt ueber definierte Berechtigungen, die bestimmen, was ein Benutzer ausfuehren darf.

### Rollenuebersicht

- **Superadmin** -- globaler Systemadministrator mit vollem Zugriff auf alle Organisationen. Kann Organisationen erstellen und loeschen, Landesfoerderungskonfigurationen verwalten und alle Operationen durchfuehren.
- **Admin** -- volle Kontrolle innerhalb zugewiesener Organisationen. Kann Personal, Kinder, Vertraege, Gruppen, Verguetungsplaene und Benutzer verwalten. Kann keine Organisationen erstellen oder loeschen und keine Landesfoerderungskonfigurationen verwalten.
- **Manager** -- erledigt taegliche operative Aufgaben innerhalb zugewiesener Organisationen. Kann Personal, Kinder und Vertraege verwalten. Hat Lesezugriff auf Benutzer, Gruppen und Verguetungsplaene.
- **Mitglied** -- Lesezugriff innerhalb zugewiesener Organisationen. Kann Personal, Kinder, Vertraege, Gruppen und Verguetungsplaene einsehen, aber nichts aendern.
- **Personal** -- konzipiert fuer Erzieher/innen und Assistenzkraefte, die Anwesenheiten erfassen muessen. Kann Kinder, Kindervertraege und Gruppen einsehen. Hat vollen Lese-/Schreibzugriff ausschliesslich auf Anwesenheitsdaten.

### Berechtigungsmatrix

| Ressource | Superadmin | Admin | Manager | Mitglied | Personal |
|-----------|-----------|-------|---------|----------|----------|
| Organisationen | CRUD | Lesen/Aktualisieren | Lesen | Lesen | Lesen |
| Personal | CRUD | CRUD | CRUD | Lesen | -- |
| Kinder | CRUD | CRUD | CRUD | Lesen | Lesen |
| Vertraege | CRUD | CRUD | CRUD | Lesen | Lesen (nur Kind) |
| Anwesenheit | CRUD | CRUD | CRUD | Lesen | CRUD |
| Gruppen | CRUD | CRUD | Lesen | Lesen | Lesen |
| Landesfoerderung | CRUD | -- | -- | -- | -- |
| Verguetungsplaene | CRUD | CRUD | Lesen | Lesen | -- |
| Budget | CRUD | CRUD | Lesen | Lesen | -- |
| Statistiken | Lesen | Lesen | Lesen | Lesen | -- |
| Benutzer | CRUD | CRUD | Lesen | -- | -- |
| Landesfoerderungs-Abrechnungen | Erstellen/Lesen/Loeschen | Erstellen/Lesen/Loeschen | Erstellen/Lesen/Loeschen | -- | -- |

**Geltungsbereich:** Superadmins agieren organisationsuebergreifend. Alle anderen Rollen sind auf ihre zugewiesenen Organisationen beschraenkt.

## Organisationsmitgliedschaft

Benutzer werden Organisationen mit einer bestimmten Rolle zugewiesen. Dies bestimmt, worauf sie zugreifen koennen und in welchem Bereich.

### Wichtige Konzepte

- Ein Benutzer kann **mehreren Organisationen** mit unterschiedlichen Rollen angehoeren. Beispielsweise kann ein Benutzer in einer Kita Admin und in einer anderen Manager sein.
- Rollenzuweisungen werden ueber die Benutzerverwaltungsoberflaeche verwaltet. Admins koennen Organisationsmitgliedschaften fuer Benutzer innerhalb ihrer eigenen Organisationen hinzufuegen oder entfernen.
- Superadmins koennen Mitgliedschaften in allen Organisationen verwalten.

### Rolle zuweisen

Um einen Benutzer einer Organisation zuzuweisen, waehlen Sie den Benutzer in der Benutzerverwaltungsoberflaeche aus, waehlen die Zielorganisation und weisen die gewuenschte Rolle zu (Admin, Manager, Mitglied oder Personal).

### Mitgliedschaft entfernen

Das Entfernen der Mitgliedschaft eines Benutzers aus einer Organisation widerruft dessen Zugriff auf die Daten dieser Organisation. Das Benutzerkonto selbst wird nicht geloescht.

## Landesfoerderung konfigurieren

Die Konfiguration der Landesfoerderung ist eine **Superadmin-exklusive** Operation. Sie definiert die Foerderungssaetze, die staatliche Stellen fuer die Kinderbetreuung basierend auf den Landesvorschriften zahlen.

### Struktur

Eine Landesfoerderungskonfiguration besteht aus:

1. **Foerderungskonfiguration** -- ein uebergeordneter Eintrag mit einem Namen und dem zugehoerigen Bundesland
2. **Zeitraeume** -- Datumsbereiche (von/bis) innerhalb einer Konfiguration, jeweils mit einem Wert fuer die woechentliche Vollzeitstundenzahl
3. **Eigenschaften** -- einzelne Foerderungssatzeintraege innerhalb eines Zeitraums

### Eigenschaften

Jede Eigenschaft definiert einen bestimmten Foerderungssatz mit folgenden Feldern:

| Feld | Beschreibung | Beispiel |
|------|-------------|---------|
| Key | Kategoriebezeichner | `care_type` |
| Value | Spezifischer Wert innerhalb der Kategorie | `ganztag` |
| Label | Menschenlesbare Beschreibung | "Ganztagsbetreuung" |
| Payment | Betrag in Cent | `166847` (= 1.668,47 EUR) |
| Min Age | Mindestalter des Kindes (Monate) | `0` |
| Max Age | Hoechstalter des Kindes (Monate) | `36` |
| Apply to All | Ob dieser Satz universell gilt | `true` / `false` |

{{% callout type="info" %}}
Alle Geldbetraege werden als Ganzzahlen in Cent gespeichert, um Gleitkomma-Praezisionsfehler zu vermeiden. Beispielsweise wird 1.668,47 EUR als `166847` gespeichert.
{{% /callout %}}

### Foerderungssaetze importieren

Foerderungssaetze koennen aus YAML-Dateien importiert werden. Dies ist nuetzlich fuer das massenhafte Laden offizieller staatlicher Foerderungstabellen. Das YAML-Format definiert die vollstaendige Konfiguration einschliesslich Zeitraeume und Eigenschaften.

## Verguetungsplaene konfigurieren

Verguetungsplaene definieren Gehaltsstrukturen fuer das Personal, typischerweise nach Tarifvertraegen wie TVoeD-SuE.

### Struktur

Ein Verguetungsplan besteht aus:

1. **Verguetungsplan** -- ein benannter Plan (z.B. "TVoeD-SuE"), der einer Organisation zugeordnet ist
2. **Zeitraeume** -- Datumsbereiche mit zugehoerigen Wochenstunden und Arbeitgeberbeitragssatz
3. **Eintraege** -- einzelne Gehaltseintraege innerhalb eines Zeitraums

### Zeitraeume

Jeder Zeitraum definiert:

| Feld | Beschreibung | Beispiel |
|------|-------------|---------|
| Von | Startdatum | 2025-01-01 |
| Bis | Enddatum | 2025-12-31 |
| Wochenstunden | Regulaere woechentliche Arbeitszeit | 39,0 |
| Arbeitgeberbeitragssatz | Satz in Hundertstel Prozent | `2050` (= 20,50%) |

### Eintraege

Jeder Eintrag innerhalb eines Zeitraums definiert:

| Feld | Beschreibung | Beispiel |
|------|-------------|---------|
| Entgeltgruppe | Verguetungsstufe | `S8a` |
| Stufe | Erfahrungsstufe (1--6) | `3` |
| Monatsbetrag | Gehalt in Cent | `385000` (= 3.850,00 EUR) |
| Mindestjahre | Mindestberufserfahrung fuer diese Stufe | `5` |

### Import und Export

Verguetungsplaene koennen aus YAML-Dateien importiert und in YAML-Dateien exportiert werden. Dies vereinfacht die Einrichtung standardisierter Gehaltsstrukturen und deren Weitergabe an andere Organisationen.

## Audit-Protokollierung

Alle Erstell-, Aktualisierungs- und Loeschvorgaenge in KitaManager werden im Audit-Protokoll erfasst. Dies unterstuetzt Compliance-Anforderungen und ermoeglicht die Nachverfolgung, wer was und wann geaendert hat.

### Protokollierte Informationen

Jeder Audit-Protokolleintrag enthaelt:

| Feld | Beschreibung |
|------|-------------|
| Akteur | Der Benutzer, der die Aktion durchgefuehrt hat |
| Ressourcentyp | Der Typ der betroffenen Ressource (z.B. Personal, Kind, Vertrag) |
| Ressourcen-ID | Die Datenbank-ID der betroffenen Ressource |
| Ressourcenname | Ein menschenlesbarer Name der betroffenen Ressource |
| IP-Adresse | Die IP-Adresse, von der die Aktion ausgefuehrt wurde |
| Zeitstempel | Wann die Aktion stattfand |

Audit-Protokolle sind schreibgeschuetzt und koennen weder geaendert noch geloescht werden.

## Testdaten

Entwicklungs- und Testumgebungen koennen mit Beispieldaten befuellt werden, um die Einrichtung und das Testen zu erleichtern.

### Was wird befuellt

- Eine Beispielorganisation ("Kita Sonnenschein")
- Testkinder mit Vertraegen
- Beispielpersonal
- Berliner Landesfoerderungskonfiguration

### Testdaten ausfuehren

Verwenden Sie das Makefile-Target, um die Datenbank zu befuellen:

```bash
make seed
```

Alternativ kann der Seeding-API-Endpunkt im Entwicklungsmodus direkt aufgerufen werden.

{{% callout type="warning" %}}
Testdaten sind ausschliesslich fuer Entwicklungs- und Testumgebungen vorgesehen. Fuehren Sie das Seeding nicht auf Produktionsdatenbanken aus.
{{% /callout %}}

## Naechste Schritte

- [Erste Schritte](../getting-started/) -- Anwendung einrichten
- [Architekturuebersicht](../architecture/) -- Systemdesign verstehen
- [API-Referenz](../api/) -- REST API erkunden
