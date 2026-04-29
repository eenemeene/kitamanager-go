---
title: API-Referenz
weight: 3
---

KitaManager bietet eine REST-API mit interaktiver OpenAPI/Swagger-Dokumentation unter `/swagger/index.html` beim Ausführen der Anwendung.

Die Authentifizierung erfolgt **per Cookie**: Eine erfolgreiche Anmeldung setzt ein HttpOnly-Sitzungscookie `access_token` und ein für JavaScript lesbares `csrf_token`-Cookie. Mutierende Anfragen (POST/PUT/PATCH/DELETE) müssen den CSRF-Token im `X-CSRF-Token`-Header mitsenden. Es gibt keinen separaten Refresh-Endpunkt -- Sitzungen bleiben gültig, bis Sie sich abmelden, das Cookie abläuft oder Sie die Sitzung über `/me/sessions` widerrufen.

## Authentifizierung

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| POST | `/api/v1/login` | Schritt 1 der Anmeldung: Passwort prüfen. Liefert entweder die Sitzungscookies (kein MFA) oder einen kurzlebigen MFA-Challenge-Token (MFA aktiv). |
| POST | `/api/v1/auth/mfa/challenge` | WebAuthn-Challenge für die laufende Anmeldung erzeugen (für Sicherheitsschlüssel). |
| POST | `/api/v1/auth/mfa/verify` | Schritt 2 der Anmeldung: TOTP-Code, Wiederherstellungscode oder WebAuthn-Assertion gegen den Challenge-Token prüfen. Bei Erfolg werden die Sitzungscookies gesetzt. |
| POST | `/api/v1/logout` | Aktuelle Sitzung beenden und Cookies löschen. |
| GET | `/api/v1/me` | Profil des aktuellen Benutzers abrufen. |
| PUT | `/api/v1/me/password` | Eigenes Passwort ändern. Beendet alle anderen Sitzungen. |
| GET | `/api/v1/me/sessions` | Aktive Sitzungen des aktuellen Benutzers auflisten (Gerät, IP, letzte Aktivität). |
| DELETE | `/api/v1/me/sessions/{id}` | Eine bestimmte Sitzung beenden. |

### Login-Beispiel (ohne MFA)

```bash
curl -i -c cookies.txt -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "supersecret"}'
```

Die Antwort setzt zwei Cookies:

```
Set-Cookie: access_token=...; HttpOnly; Path=/
Set-Cookie: csrf_token=...; Path=/
```

### Login-Beispiel (mit MFA)

Hat der Benutzer MFA aktiviert, enthält die Antwort ein Flag `mfa_required` und einen kurzlebigen `mfa_challenge_token`:

```json
{ "mfa_required": true, "mfa_challenge_token": "...", "factors": ["totp", "webauthn"] }
```

Die Anmeldung wird mit einem Code oder einer WebAuthn-Assertion gegen `/auth/mfa/verify` abgeschlossen:

```bash
curl -i -c cookies.txt -X POST http://localhost:8080/api/v1/auth/mfa/verify \
  -H "Content-Type: application/json" \
  -d '{"challenge_token": "...", "factor_type": "totp", "code": "123456"}'
```

### Cookie verwenden

Für nachfolgende Aufrufe wird die Cookie-Datei mitgesendet. Mutierende Aufrufe ergänzen den CSRF-Token aus dem `csrf_token`-Cookie:

```bash
curl http://localhost:8080/api/v1/organizations -b cookies.txt
curl -X POST http://localhost:8080/api/v1/organizations \
  -b cookies.txt \
  -H "X-CSRF-Token: $(grep csrf_token cookies.txt | awk '{print $7}')" \
  -H "Content-Type: application/json" \
  -d '{"name": "New Org", "state": "berlin"}'
```

## Mehrfaktor-Authentifizierung (MFA-Faktoren)

Benutzer verwalten ihre eigenen Faktoren über `/users/{userId}/factors`. Der Pfadparameter `:userId` akzeptiert das Schlüsselwort `me` als Selbst-Alias -- aktuell liefert das Adressieren eines anderen Benutzers 403.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/users/{userId}/factors` | Registrierte Faktoren auflisten. |
| POST | `/api/v1/users/{userId}/factors` | Neuen Faktor registrieren. Body enthält Faktortyp (`totp` oder `webauthn`) und aktuelles Passwort. Antwort enthält Registrierungsdaten (TOTP-Schlüssel + otpauth-URI bzw. WebAuthn-Registrierungs-Challenge). |
| GET | `/api/v1/users/{userId}/factors/{id}` | Faktor abrufen. |
| PATCH | `/api/v1/users/{userId}/factors/{id}` | Bezeichnung eines Faktors ändern. |
| DELETE | `/api/v1/users/{userId}/factors/{id}` | Faktor löschen. Erfordert Passwort und aktuellen TOTP-Code. Beim Entfernen des letzten Primärfaktors werden auch die Wiederherstellungscodes entfernt. |
| POST | `/api/v1/users/{userId}/factors/{id}/activate` | Pendierende Registrierung mit einem Verifikationscode aktivieren. Bei der ersten Aktivierung werden zugleich Einmal-Wiederherstellungscodes ausgegeben. |
| POST | `/api/v1/users/{userId}/factors/{id}/regenerate` | Wiederherstellungscodes neu erzeugen; alte Codes werden ungültig. |

## Organisationen

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/organizations` | Organisationen auflisten |
| POST | `/api/v1/organizations` | Organisation erstellen (Superadmin) |
| GET | `/api/v1/organizations/{orgId}` | Organisation abrufen |
| PUT | `/api/v1/organizations/{orgId}` | Organisation aktualisieren |
| DELETE | `/api/v1/organizations/{orgId}` | Organisation löschen (Superadmin) |

## Bereiche

Alle Bereich-Endpunkte sind einer Organisation zugeordnet: `/api/v1/organizations/{orgId}/sections`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../sections` | Bereiche auflisten |
| POST | `.../sections` | Bereich erstellen |
| GET | `.../sections/{sectionId}` | Bereich abrufen |
| PUT | `.../sections/{sectionId}` | Bereich aktualisieren |
| DELETE | `.../sections/{sectionId}` | Bereich löschen |

## Mitarbeiter

Alle Mitarbeiter-Endpunkte sind einer Organisation zugeordnet: `/api/v1/organizations/{orgId}/employees`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../employees` | Mitarbeiter auflisten |
| POST | `.../employees` | Mitarbeiter erstellen |
| GET | `.../employees/{id}` | Mitarbeiter abrufen |
| PUT | `.../employees/{id}` | Mitarbeiter aktualisieren |
| DELETE | `.../employees/{id}` | Mitarbeiter löschen |
| GET | `.../employees/export/excel` | Mitarbeiter als Excel exportieren |
| GET | `.../employees/export/yaml` | Mitarbeiter als YAML exportieren |
| POST | `.../employees/import` | Mitarbeiter aus YAML importieren |
| GET | `.../employees/step-promotions` | Stufenaufstiege abrufen |

### Mitarbeiterverträge

Verschachtelt unter einem Mitarbeiter: `.../employees/{id}/contracts`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../contracts` | Verträge auflisten |
| POST | `.../contracts` | Vertrag erstellen |
| GET | `.../contracts/current` | Aktuellen aktiven Vertrag abrufen |
| GET | `.../contracts/{contractId}` | Vertrag abrufen |
| PUT | `.../contracts/{contractId}` | Vertrag aktualisieren |
| DELETE | `.../contracts/{contractId}` | Vertrag löschen |

## Kinder

Alle Kind-Endpunkte sind einer Organisation zugeordnet: `/api/v1/organizations/{orgId}/children`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../children` | Kinder auflisten |
| POST | `.../children` | Kind erstellen |
| GET | `.../children/{id}` | Kind abrufen |
| PUT | `.../children/{id}` | Kind aktualisieren |
| DELETE | `.../children/{id}` | Kind löschen |
| GET | `.../children/export/excel` | Kinder als Excel exportieren |
| GET | `.../children/export/yaml` | Kinder als YAML exportieren |
| POST | `.../children/import` | Kinder aus YAML importieren |
| GET | `.../children/attendance` | Organisationsweite Anwesenheit nach Datum |
| GET | `.../children/attendance/summary` | Tägliche Anwesenheitsübersicht |

### Kinderverträge

Verschachtelt unter einem Kind: `.../children/{id}/contracts`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../contracts` | Verträge auflisten |
| POST | `.../contracts` | Vertrag erstellen |
| GET | `.../contracts/current` | Aktuellen aktiven Vertrag abrufen |
| GET | `.../contracts/{contractId}` | Vertrag abrufen |
| PUT | `.../contracts/{contractId}` | Vertrag aktualisieren |
| DELETE | `.../contracts/{contractId}` | Vertrag löschen |

### Anwesenheit

Verschachtelt unter einem Kind: `.../children/{id}/attendance`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| POST | `.../attendance` | Anwesenheitseintrag erstellen |
| GET | `.../attendance` | Anwesenheitseinträge des Kindes auflisten |
| GET | `.../attendance/{attendanceId}` | Anwesenheitseintrag abrufen |
| PUT | `.../attendance/{attendanceId}` | Anwesenheitseintrag aktualisieren |
| DELETE | `.../attendance/{attendanceId}` | Anwesenheitseintrag löschen |

## Landesförderungssätze

Globale Ressource, verwaltet von Superadmins.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/government-funding-rates` | Förderungskonfigurationen auflisten |
| POST | `/api/v1/government-funding-rates` | Förderungskonfiguration erstellen |
| GET | `/api/v1/government-funding-rates/{id}` | Förderungskonfiguration abrufen |
| PUT | `/api/v1/government-funding-rates/{id}` | Förderungskonfiguration aktualisieren |
| DELETE | `/api/v1/government-funding-rates/{id}` | Förderungskonfiguration löschen |
| POST | `/api/v1/government-funding-rates/import` | Förderungssätze aus YAML importieren |

### Förderungszeiträume

Verschachtelt unter einem Förderungssatz: `.../government-funding-rates/{id}/periods`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| POST | `.../periods` | Zeitraum erstellen |
| GET | `.../periods/{periodId}` | Zeitraum abrufen |
| PUT | `.../periods/{periodId}` | Zeitraum aktualisieren |
| DELETE | `.../periods/{periodId}` | Zeitraum löschen |

### Förderungseigenschaften

Verschachtelt unter einem Zeitraum: `.../periods/{periodId}/properties`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| POST | `.../properties` | Eigenschaft erstellen |
| GET | `.../properties/{propertyId}` | Eigenschaft abrufen |
| PUT | `.../properties/{propertyId}` | Eigenschaft aktualisieren |
| DELETE | `.../properties/{propertyId}` | Eigenschaft löschen |

## Landesförderungsabrechnungen

Einer Organisation zugeordnet: `/api/v1/organizations/{orgId}/government-funding-bills`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../government-funding-bills` | Abrechnungen auflisten |
| POST | `.../government-funding-bills` | ISBJ-Abrechnung hochladen |
| GET | `.../government-funding-bills/{billId}` | Abrechnung abrufen |
| GET | `.../government-funding-bills/{billId}/compare` | Berechnete und abgerechnete Beträge vergleichen |
| DELETE | `.../government-funding-bills/{billId}` | Abrechnung löschen |

## Vergütungspläne

Einer Organisation zugeordnet: `/api/v1/organizations/{orgId}/pay-plans`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../pay-plans` | Vergütungspläne auflisten |
| POST | `.../pay-plans` | Vergütungsplan erstellen |
| GET | `.../pay-plans/{id}` | Vergütungsplan abrufen |
| PUT | `.../pay-plans/{id}` | Vergütungsplan aktualisieren |
| DELETE | `.../pay-plans/{id}` | Vergütungsplan löschen |
| GET | `.../pay-plans/{id}/export` | Vergütungsplan als YAML exportieren |
| POST | `.../pay-plans/import` | Vergütungsplan aus YAML importieren |

### Vergütungsplan-Zeiträume

Verschachtelt unter einem Vergütungsplan: `.../pay-plans/{id}/periods`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| POST | `.../periods` | Zeitraum erstellen |
| GET | `.../periods/{periodId}` | Zeitraum abrufen |
| PUT | `.../periods/{periodId}` | Zeitraum aktualisieren |
| DELETE | `.../periods/{periodId}` | Zeitraum löschen |

### Vergütungsplan-Einträge

Verschachtelt unter einem Zeitraum: `.../periods/{periodId}/entries`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| POST | `.../entries` | Eintrag erstellen |
| GET | `.../entries/{entryId}` | Eintrag abrufen |
| PUT | `.../entries/{entryId}` | Eintrag aktualisieren |
| DELETE | `.../entries/{entryId}` | Eintrag löschen |

## Budgetposten

Einer Organisation zugeordnet: `/api/v1/organizations/{orgId}/budget-items`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../budget-items` | Budgetposten auflisten |
| POST | `.../budget-items` | Budgetposten erstellen |
| GET | `.../budget-items/{id}` | Budgetposten abrufen |
| PUT | `.../budget-items/{id}` | Budgetposten aktualisieren |
| DELETE | `.../budget-items/{id}` | Budgetposten löschen |

### Budgetposten-Einträge

Verschachtelt unter einem Budgetposten: `.../budget-items/{id}/entries`.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../entries` | Einträge auflisten |
| POST | `.../entries` | Eintrag erstellen |
| GET | `.../entries/{entryId}` | Eintrag abrufen |
| PUT | `.../entries/{entryId}` | Eintrag aktualisieren |
| DELETE | `.../entries/{entryId}` | Eintrag löschen |

## Statistiken

Einer Organisation zugeordnet: `/api/v1/organizations/{orgId}/statistics`. Alle Statistik-Endpunkte erfordern die Abfrageparameter `from` und `to` zur Angabe eines Datumsbereichs (Format: `YYYY-MM-DD`).

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `.../statistics/staffing-hours` | Übersicht der Personalstunden |
| GET | `.../statistics/staffing-hours/employees` | Personalstunden pro Mitarbeiter |
| GET | `.../statistics/financials` | Finanzübersicht |
| GET | `.../statistics/occupancy` | Belegungsstatistiken |
| GET | `.../statistics/age-distribution` | Altersverteilung |
| GET | `.../statistics/contract-properties` | Verteilung der Vertragseigenschaften |
| GET | `.../statistics/funding` | Förderungsstatistiken |

### Prognose und Schätzungen

Diese Endpunkte versorgen die **Prognose**-Seite. Sie nehmen denselben Datumsbereich plus eine Liste von Szenario-Änderungen (hinzugefügte/entfernte Kinder, hinzugefügte/entfernte Mitarbeiter) entgegen und liefern die projizierten Statistiken mit den Änderungen darüber.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| POST | `.../statistics/forecast` | Volle Kitajahresprojektion mit dem übergebenen Szenario auf Basis der aktuellen Daten. Liefert Finanzen, Personalstunden, Belegung und Mitarbeiterstunden für den Zeitraum. |
| POST | `.../statistics/estimates/child-funding` | Schnellschätzung der monatlichen Förderung für ein hypothetisches Kind, ohne gespeicherte Daten zu verändern. |
| POST | `.../statistics/estimates/employee-cost` | Schnellschätzung der monatlichen Kosten (brutto + Arbeitgeberanteil) für eine hypothetische Mitarbeiterin / einen hypothetischen Mitarbeiter. |

## Audit-Protokolle

Zwei Sichten:

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/audit-logs` | **Nur Superadmin**: globales Protokoll inklusive Anmelde- und Authentifizierungsereignissen. Filter `from`, `to`, `action`, `actor`. |
| GET | `/api/v1/audit-logs/{id}` | **Nur Superadmin**: einen einzelnen Eintrag abrufen. |
| GET | `/api/v1/organizations/{orgId}/audit-logs` | **Org-Admin**: auf eine Organisation eingeschränkte Sicht. Anmelde- und Passwortereignisse werden ausgeblendet. |

## Benutzer

Globale Endpunkte zur Benutzerverwaltung.

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/users` | Benutzer auflisten |
| POST | `/api/v1/users` | Benutzer erstellen |
| GET | `/api/v1/users/{id}` | Benutzer abrufen |
| PUT | `/api/v1/users/{id}` | Benutzer aktualisieren |
| DELETE | `/api/v1/users/{id}` | Benutzer löschen |
| GET | `/api/v1/users/{id}/memberships` | Organisationsmitgliedschaften des Benutzers abrufen |
| POST | `/api/v1/users/{id}/organizations` | Benutzer einer Organisation hinzufügen |
| PUT | `/api/v1/users/{id}/organizations/{orgId}` | Rolle des Benutzers in der Organisation aktualisieren |
| DELETE | `/api/v1/users/{id}/organizations/{orgId}` | Benutzer aus Organisation entfernen |
| PUT | `/api/v1/users/{id}/password` | Passwort des Benutzers zurücksetzen (Admin) |
| PUT | `/api/v1/users/{id}/superadmin` | Superadmin-Status festlegen |

### Organisationsbenutzer

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/organizations/{orgId}/users` | Benutzer einer Organisation auflisten |

## Paginierung

Listen-Endpunkte unterstützen Paginierung über Abfrageparameter:

```bash
curl "http://localhost:8080/api/v1/organizations?page=1&limit=10" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

Antwort:

```json
{
  "data": [],
  "total": 100,
  "page": 1,
  "limit": 10
}
```

## Fehlerantworten

Fehler werden mit dem entsprechenden HTTP-Statuscode und einem JSON-Body zurückgegeben:

```json
{
  "error": "Beschreibung des Fehlers"
}
```

| Status | Bedeutung |
|--------|-----------|
| 400 | Bad Request -- Ungültige Eingabe oder fehlende Pflichtparameter |
| 401 | Unauthorized -- Fehlender oder ungültiger Authentifizierungstoken |
| 403 | Forbidden -- Unzureichende Berechtigungen für die angeforderte Aktion |
| 404 | Not Found -- Die angeforderte Ressource existiert nicht |
| 500 | Internal Server Error -- Ein unerwarteter Fehler ist aufgetreten |
