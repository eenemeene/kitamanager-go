---
title: Funktionen
weight: 2
---

KitaManager Go bietet einen umfassenden Funktionsumfang für die Verwaltung von Kindertagesstätten.

## Organisationsverwaltung

### Multi-Mandanten-Architektur
- Unterstützung mehrerer unabhängiger Organisationen
- Vollständige Datenisolierung zwischen Organisationen
- Organisationsbezogene Zugriffskontrolle
- Konfiguration auf Bundesland-Ebene (z.B. Berlin, Bayern)

### Organisationsbereiche
- Kinder und Mitarbeiter in Abteilungen gruppieren
- Flexible Bereichsstruktur
- Kapazitätsverwaltung

## Personalverwaltung

### Vollständige Mitarbeiterdatenbank
- Verwaltung persönlicher Informationen
- Nachverfolgung der Beschäftigungshistorie
- Flexible Vertragsverwaltung
- Mehrere gleichzeitige Verträge pro Mitarbeiter

### Vertragsverwaltung
- Definition von Beschäftigungsbedingungen
- Positions- und Entgeltgruppen-Tracking
- Wochenstunden-Konfiguration
- Vertragsvalidierung (keine Überschneidungen)

### Vergütungsplan-Integration
- Definition von Vergütungsskalen und Gruppen
- Stufenbasierte Gehaltsentwicklung
- Entgeltgruppen-Stufen-Kombinationen (z.B. S8a Stufe 3)

## Kinderverwaltung

### Umfassende Kinderdaten
- Persönliche Daten und Kontaktinformationen
- Geburtsdatum und Geschlecht
- Anmeldungshistorie

### Betreuungsverträge
- Flexible Vertragslaufzeiten
- Benutzerdefinierte Eigenschaften (Betreuungsart usw.)
- Automatische Förderungsberechnung
- Validierung nicht überlappender Verträge

### Förderungsintegration
- Automatische Anzeige des Förderbetrags
- Eigenschaftsbasierte Förderungsberechnungen
- Konfiguration von Landesförderungen

## Zugriffskontrolle

### Rollenbasierte Berechtigungen (RBAC)
Hybrid-System, das Casbin-Richtlinien mit datenbankgespeicherten Zuweisungen kombiniert.

| Rolle | Beschreibung |
|-------|--------------|
| **Superadmin** | Vollständiger Systemzugriff über alle Organisationen |
| **Admin** | Vollständiger Zugriff innerhalb zugewiesener Organisation(en) |
| **Manager** | Operativer Zugriff (Mitarbeiter, Kinder, Verträge) |
| **Mitglied** | Nur-Lese-Zugriff |

### Audit-Protokollierung
- Nachverfolgung aller Datenänderungen
- Benutzeraktions-Historie
- Compliance-bereite Audit-Trails

## Landesförderung

### Bundeland-Konfiguration
- Förderungsregeln pro Bundesland definieren
- Eigenschaftsbasierte Förderungseinträge
- Flexible Zeitraumverwaltung

### Förderungszeiträume
- Zeitbasierte Förderungskonfigurationen
- Eigenschaftskombinationen (z.B. care_type + ndh)
- Betragsberechnungen in Cent (Präzision)

### Automatische Berechnungen
- Echtzeit-Förderungsanzeige
- Vertragseigenschafts-Abgleich
- Berichtsfunktionen

## API & Integration

### REST-API
- Umfassende REST-API
- OpenAPI/Swagger-Dokumentation
- JWT-Authentifizierung

### Sicherheitsfunktionen
- CORS-Konfiguration
- Rate-Limiting
- CSRF-Schutz
- Anfragegrößen-Limits
- Sicherheits-Header

## Moderne Technologie

### Leistung
- Mit Go für hohe Leistung entwickelt
- Effiziente PostgreSQL-Abfragen
- Optimiertes Frontend mit Next.js

### Entwicklererfahrung
- Umfassende Dokumentation
- TypeScript-Frontend
- Hot-Reloading in der Entwicklung
- Umfangreiche Testabdeckung

### Bereitstellung
- Docker-Unterstützung
- Single-Binary-Bereitstellung
- Eingebettete Frontend-Option
- Einfache Konfiguration
