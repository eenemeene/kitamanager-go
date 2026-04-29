---
title: Erste Schritte
weight: 1
---

Diese Anleitung hilft Ihnen, KitaManager schnell zum Laufen zu bringen.

## Voraussetzungen

- [Docker](https://docs.docker.com/get-docker/) und Docker Compose
- [Go 1.25+](https://go.dev/dl/) (für Entwicklung)
- [Node.js 24+](https://nodejs.org/) (für Frontend-Entwicklung)

## Schnellstart mit Docker

Der schnellste Weg zum Starten ist mit Docker Compose:

```bash
# Alle Dienste starten
docker compose up -d
```

Dies startet:
- PostgreSQL-Datenbank
- KitaManager API-Server
- Next.js-Frontend

Zugriff auf die Anwendung unter `http://localhost:3000`.

## Entwicklungsumgebung

Für lokale Entwicklung:

```bash
# Frontend-Abhängigkeiten installieren
make web-install

# API bauen
make api-build

# Entwicklungsumgebung starten
make dev
```

### Verfügbare Make-Befehle

| Befehl | Beschreibung |
|--------|--------------|
| `make dev` | Vollständige Entwicklungsumgebung starten |
| `make api-build` | Go API bauen |
| `make api-test` | API-Tests ausführen |
| `make web-install` | Frontend-Abhängigkeiten installieren |
| `make web-dev` | Frontend-Entwicklungsserver starten |
| `make swagger-docs` | API-Dokumentation generieren |

## Standard-Anmeldedaten

Nach dem Start können Sie sich mit einem der angelegten Testkonten anmelden. Jedes zeigt eine andere Rolle:

| E-Mail | Rolle | Passwort |
|--------|-------|----------|
| `superadmin@example.com` | Superadmin | `supersecret` |
| `admin@example.com` | Admin (Kita Sonnenschein) | `supersecret` |
| `manager@example.com` | Manager (Kita Sonnenschein) | `supersecret` |

{{% callout type="warning" %}}
Diese Zugangsdaten sind ausschließlich für die lokale Entwicklung gedacht. Ändern Sie alle Standard-Passwörter sofort in Produktionsumgebungen und aktivieren Sie die Zwei-Faktor-Authentifizierung für jedes Konto, das Daten bearbeiten kann.
{{% /callout %}}

## Testdaten

Die Entwicklungsumgebung enthält Testdaten mit:

- Einer Beispielorganisation „Kita Sonnenschein" mit drei Bereichen (Nest, Nestflüchter, Große)
- ~120 Kindern mit Betreuungsverträgen und realistischer Altersverteilung
- ~35 Mitarbeiterinnen und Mitarbeitern mit Arbeitsverträgen in allen Bereichen
- Entgelttabellen (TVöD-SuE 2024 und Minijob)
- Haushaltsposten (Elternbeiträge und Betriebskosten)
- Berliner Kita-Fördersätze
- Drei Testbenutzer mit verschiedenen Rollen (Superadmin, Admin, Manager)

## Nächste Schritte

- [Architektur-Übersicht](../architecture/) - Systemdesign verstehen
- [API-Referenz](../api/) - Die REST-API erkunden
