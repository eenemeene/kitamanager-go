---
title: Erste Schritte
weight: 1
---

Diese Anleitung hilft Ihnen, KitaManager Go schnell zum Laufen zu bringen.

## Voraussetzungen

- [Docker](https://docs.docker.com/get-docker/) und Docker Compose
- [Go 1.25+](https://go.dev/dl/) (für Entwicklung)
- [Node.js 18+](https://nodejs.org/) (für Frontend-Entwicklung)

## Schnellstart mit Docker

Der schnellste Weg zum Starten ist mit Docker Compose:

```bash
# Repository klonen
git clone https://github.com/toabctl/kitamanager-go.git
cd kitamanager-go

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

Nach dem Start können Sie sich mit den Standard-Admin-Anmeldedaten anmelden:

| Feld | Wert |
|------|------|
| E-Mail | `admin@example.com` |
| Passwort | `admin123` |

{{% callout type="warning" %}}
Ändern Sie das Standard-Passwort sofort in Produktionsumgebungen!
{{% /callout %}}

## Testdaten

Die Entwicklungsumgebung enthält Testdaten mit:

- Einer Beispielorganisation "Kita Sonnenschein"
- 50 Testkindern mit Verträgen
- Beispielmitarbeitern
- Berliner Landesförderungs-Konfiguration

## Nächste Schritte

- [Architektur-Übersicht](../architecture) - Systemdesign verstehen
- [API-Referenz](../api) - Die REST-API erkunden
- [Bereitstellungs-Anleitung](../deployment) - Produktions-Bereitstellungsoptionen
