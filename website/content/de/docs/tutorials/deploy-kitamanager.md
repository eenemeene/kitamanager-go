---
title: KitaManager bereitstellen
weight: 2
---

Dieses Tutorial führt Sie durch das Hochfahren einer frischen KitaManager-Instanz mit Docker Compose auf einem Rechner, den Sie kontrollieren. Am Ende haben Sie einen lauffähigen Stack mit geladenen Berliner Fördersätzen, einem Superadmin-Konto und einer vernünftigen produktionsähnlichen Konfiguration.

Sie brauchen:

- Einen Linux-Host mit installiertem Docker und Docker Compose.
- Etwa 30 Minuten beim ersten Mal, hauptsächlich für Image-Downloads.
- Offene Ports für Frontend (Standard `3000`) und API (Standard `8080`). Für eine Produktivinstallation würden Sie beides hinter einem HTTPS-Reverse-Proxy betreiben — am Ende beschrieben.

## Schritt 1 — Code holen

```bash
git clone https://github.com/eenemeene/kitamanager-go.git
cd kitamanager-go
git checkout v0.34.0   # oder Ihr gewähltes Release-Tag
```

Für Produktion ist das Anpinnen an ein Tag dem Verfolgen von `main` vorzuziehen. Neue Tags werden automatisch als Docker-Images veröffentlicht.

## Schritt 2 — Geheimnisse konfigurieren

Kopieren Sie die Produktions-Env-Vorlage und füllen Sie sie aus:

```bash
cp .env.production.example .env
```

Öffnen Sie `.env` in einem Editor. Die Pflichtwerte sind:

```
DB_USER=kitamanager
DB_PASSWORD=<langes Zufallspasswort>
DB_NAME=kitamanager

CSRF_HMAC_KEY=<openssl rand -hex 32>
TOTP_ENCRYPTION_KEY=<openssl rand -hex 32>

WEBAUTHN_RP_ID=<Ihre Domain, z. B. kitamanager.example.org>
WEBAUTHN_RP_NAME=KitaManager
WEBAUTHN_ORIGINS=https://kitamanager.example.org

SEED_ADMIN_EMAIL=admin@your-org.example
SEED_ADMIN_PASSWORD=<temporäres Passwort — sofort nach erstem Login ändern>
SEED_ADMIN_NAME=System Admin

GOVERNMENT_FUNDING_SEED_PATH=configs/government-fundings/berlin.yaml
GOVERNMENT_FUNDING_SEED_STATE=berlin

SECURE_COOKIES=true
DB_SSLMODE=require
```

**Überspringen Sie weder `CSRF_HMAC_KEY` noch `TOTP_ENCRYPTION_KEY`.** Der Validator weigert sich ohne starke Werte zu starten.

Die vollständige Referenz: [Umgebungsvariablen](../../reference/cli-and-config/env-vars/).

## Schritt 3 — Hochfahren

```bash
docker compose up -d
```

Der erste Start lädt Images herunter, führt Migrationen aus, lädt die Berliner Fördersätze aus YAML und legt Ihren Bootstrap-Superadmin an. Beobachten Sie das Log:

```bash
docker compose logs -f api
```

Sie warten auf `KitaManager API ready` (oder Ähnliches) und auf das Ausbleiben von Migrationsfehlern.

## Schritt 4 — Verifizieren

Health-Check der API:

```bash
curl -sf http://localhost:8080/api/v1/health
# Erwartet: {"status":"ok"}
```

Öffnen Sie `http://localhost:3000` und melden Sie sich mit `SEED_ADMIN_EMAIL` und `SEED_ADMIN_PASSWORD` an. Sie landen auf dem Dashboard.

## Schritt 5 — Bootstrap-Konto absichern

Sofort:

1. Bootstrap-Passwort in **Einstellungen → Passwort** ändern.
2. 2FA in **Einstellungen → Zwei-Faktor-Authentifizierung** aktivieren. Wiederherstellungscodes an einem dauerhaft auffindbaren Ort speichern.
3. (Optional, aber empfohlen) Sicherheitsschlüssel unter **Sicherheitsschlüssel hinzufügen** registrieren.

Bis das erledigt ist, kann sich jede:r mit Zugriff auf die Env-Datei anmelden.

## Schritt 6 — Reverse-Proxy mit HTTPS (Produktion)

Für alles über eine lokale Demo hinaus brauchen Sie HTTPS davor. Caddy ist die einfachste Lösung:

```caddy
kitamanager.example.org {
    reverse_proxy localhost:3000
}

api.kitamanager.example.org {
    reverse_proxy localhost:8080
}
```

Aktualisieren Sie `WEBAUTHN_RP_ID`, `WEBAUTHN_ORIGINS` und `CORS_ALLOW_ORIGINS` so, dass sie zu den öffentlichen URLs passen. Starten Sie die API neu.

Für nginx, Apache, Traefik: das gleiche Prinzip — TLS am Proxy terminieren, an Port `3000` (Frontend) und `8080` (API) weiterleiten.

## Schritt 7 — Backup machen

Bevor Sie echte Daten einspielen, beweisen Sie, dass Ihr Backup funktioniert. Folgen Sie dem Rezept [Datenbank sichern und wiederherstellen](../../how-to/operate/back-up-and-restore/).

## Sie sind fertig

Sie haben einen laufenden KitaManager. Von hier aus:

- Nutzer für Ihr Team einrichten — siehe [Nutzer:in anlegen](../../how-to/administer/create-a-user/).
- Eine Erzieherin durch den Alltag begleiten — verweisen Sie sie auf [Erster Tag in KitaManager](../first-day-in-kitamanager/).
- Die [wiederkehrenden Aufgaben](../../how-to/operate/) planen, die das System gesund halten: Backups, Aktualisierung der Fördersätze, Schlüsselrotation, Releases.
