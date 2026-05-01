---
title: Umgebungsvariablen
weight: 1
---

KitaManager wird fast vollständig über Umgebungsvariablen konfiguriert. Die API-Binärdatei lädt sie beim Start aus der Prozess-Umgebung plus einer optionalen `.env`-Datei im Arbeitsverzeichnis. Die Liste unten spiegelt `internal/config/config.go`; **Pflicht**-Variablen lassen `Load()` fehlschlagen, wenn sie fehlen oder Platzhalter sind. Einige Einträge (`KITAMANAGER_TIMEZONE`, `SEED_RBAC_POLICIES`) werden direkt von anderen Paketen gelesen (`internal/models/clock.go`, `cmd/api/main.go`) statt von `config.Load()`, folgen aber denselben Konventionen.

## Datenbank

| Variable | Default | Pflicht | Hinweise |
|---|---|---|---|
| `DB_HOST` | `localhost` | nein | Postgres-Host |
| `DB_PORT` | `5432` | nein | Postgres-Port |
| `DB_USER` |  | **ja** | Postgres-Benutzer |
| `DB_PASSWORD` |  | **ja** | Postgres-Passwort |
| `DB_NAME` |  | **ja** | Postgres-Datenbankname |
| `DB_SSLMODE` | `require` | nein | Eines von `disable`, `require`, `verify-ca`, `verify-full`. `disable` nur für lokale Entwicklung. |
| `DB_MAX_IDLE_CONNS` | `10` | nein | GORM-Connection-Pool-Tuning. |
| `DB_MAX_OPEN_CONNS` | `100` | nein | dito. |
| `DB_CONN_MAX_LIFE_MIN` | `60` | nein | Verbindung nach so vielen Minuten erneuern. |
| `DB_CONN_MAX_IDLE_MIN` | `10` | nein | Idle-Verbindung nach so vielen Minuten schließen. |

## HTTP-Server

| Variable | Default | Pflicht | Hinweise |
|---|---|---|---|
| `SERVER_PORT` | `8080` | nein | Listen-Port. |
| `TRUSTED_PROXIES` | (keiner) | nein | Komma-getrennte Liste von CIDRs/IPs, deren `X-Forwarded-*`-Header Gin vertraut. Hinter einem Reverse-Proxy: dessen IP eintragen. Leer = keinem Proxy vertrauen. |
| `SECURE_COOKIES` | `true` | nein | Bei `true` sind Sitzungs-Cookies `Secure` (nur HTTPS). `false` nur für lokale HTTP-Dev. |
| `CORS_ALLOW_ORIGINS` | (keiner) | nein | Komma-getrennte Liste erlaubter Origins. Leer deaktiviert CORS. |
| `CORS_ALLOW_CREDENTIALS` | `true` | nein | Ob CORS-Antworten `Access-Control-Allow-Credentials: true` enthalten. |

## Authentifizierung und Sitzungen

| Variable | Default | Pflicht | Hinweise |
|---|---|---|---|
| `JWT_SECRET` |  | **ja** | Mindestens 32 Zeichen. Signiert Sitzungs-Tokens. Rotation meldet jede aktive Sitzung ab. |
| `LOGIN_RATE_LIMIT_PER_MINUTE` | `5` | nein | Login-Versuche pro IP pro Minute. `0` deaktiviert. |
| `API_RATE_LIMIT_PER_MINUTE` | `60` | nein | Authentifizierte Request-Rate pro IP. `0` deaktiviert. |

## Multi-Faktor-Authentifizierung

| Variable | Default | Pflicht | Hinweise |
|---|---|---|---|
| `TOTP_ENCRYPTION_KEY` |  | **ja** | Genau 64 Hex-Zeichen (32 Bytes). Verschlüsselt gespeicherte TOTP-Secrets ruhend. **Rotation invalidiert jeden gespeicherten TOTP-Faktor** — betroffene Personen müssen sich neu einrichten. Erzeugen mit `openssl rand -hex 32`. |
| `TOTP_ISSUER` | `KitaManager` | nein | Der Issuer-String, der in der Authenticator-App angezeigt wird. |
| `WEBAUTHN_RP_ID` |  | nur bei WebAuthn-Nutzung | Die Relying-Party-ID — der Host-Teil Ihrer URL (z. B. `kitamanager.example.org`). Muss zur URL passen, die der Browser sieht. |
| `WEBAUTHN_RP_NAME` |  | nur bei WebAuthn-Nutzung | Lesbarer Name in der WebAuthn-Aufforderung des Browsers. |
| `WEBAUTHN_ORIGINS` |  | nur bei WebAuthn-Nutzung | Komma-getrennte Origin-URLs (z. B. `https://kitamanager.example.org`). |

## RBAC

| Variable | Default | Pflicht | Hinweise |
|---|---|---|---|
| `RBAC_MODEL_PATH` | `configs/rbac_model.conf` | nein | Pfad zur Casbin-Modelldatei. Default funktioniert für Binärdatei und Docker-Image. |

## Seed-Daten und Bootstrap

Diese erlauben einer leeren Datenbank, beim ersten Start mit einem nutzbaren Konto zu booten. Nur beim ersten Start setzen; spätere Neustarts ignorieren sie, wenn der Nutzer bereits existiert.

| Variable | Default | Pflicht | Hinweise |
|---|---|---|---|
| `SEED_ADMIN_EMAIL` |  | erforderlich, um den Bootstrap-Superadmin zu seeden (übersprungen wenn leer) | E-Mail des initialen Superadmins. |
| `SEED_ADMIN_PASSWORD` |  | erforderlich, falls `SEED_ADMIN_EMAIL` gesetzt ist | Initiales Superadmin-Passwort. Nach dem ersten Login ändern. |
| `SEED_ADMIN_NAME` | `admin` | nein | Anzeigename für den geseedeten Superadmin. |
| `SEED_TEST_DATA` | `false` | nein | Bei `true` wird die Demo-Organisation „Kita Sonnenschein" mit Bereichen, Mitarbeitenden, Kindern und Verträgen geseedet. **Niemals in Produktion setzen.** |
| `SEED_RBAC_POLICIES` | `false` | nein | Bei `true` werden die Casbin-Standardrichtlinien für Rollen und Berechtigungen beim Start geseedet. Genau einmal auf einer frischen Datenbank nötig; bei späteren Starts harmlos, aber unnötig. |
| `GOVERNMENT_FUNDING_SEED_PATH` |  | nein | Pfad zu einer YAML-Fördersatz-Datei, die beim Start geladen wird. Leer überspringt. Typischer Wert: `configs/government-fundings/berlin.yaml`. |
| `GOVERNMENT_FUNDING_SEED_STATE` | `berlin` | nein | Der Bundesland-Name, mit dem die geladene Förder-Konfiguration beschriftet wird. |

## Zeitzone

| Variable | Default | Pflicht | Hinweise |
|---|---|---|---|
| `KITAMANAGER_TIMEZONE` | `Europe/Berlin` | nein | Die Kalender-Zeitzone der Anwendung. Wird von jedem `models.Today()`-Aufruf genutzt. Mit IANA-Zonennamen überschreiben. |

## Logging

| Variable | Default | Pflicht | Hinweise |
|---|---|---|---|
| `LOG_LEVEL` | `info` | nein | Eines von `debug`, `info`, `warn`, `error`. |
| `LOG_FORMAT` | `json` | nein | Entweder `json` oder `text`. |

## E-Mail (optional)

Für Passwort-Reset / Benachrichtigungen. Wenn `SMTP_HOST` leer ist, sind E-Mail-Funktionen deaktiviert.

| Variable | Default | Pflicht |
|---|---|---|
| `SMTP_HOST` |  | nein |
| `SMTP_PORT` | `587` | nein |
| `SMTP_USER` |  | nein |
| `SMTP_PASSWORD` |  | nein |
| `SMTP_FROM` |  | nein |

## report-pdf-Tool

Das PDF-erzeugende Sidecar liest seine eigenen `KITAMANAGER_REPORT_*`-Variablen. Jede entspricht einem CLI-Schalter gleichen Namens. Siehe [Das Report-Tool](../../../explanation/architecture/the-report-tool/) und das [README des Tools](https://github.com/eenemeene/kitamanager-go/tree/main/tools/report-pdf) für die vollständige Liste.
