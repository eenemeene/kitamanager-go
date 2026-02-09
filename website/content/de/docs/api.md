---
title: API-Referenz
weight: 3
---

KitaManager Go bietet eine umfassende REST-API mit OpenAPI/Swagger-Dokumentation.

## API-Dokumentation

Die API-Dokumentation ist unter `/swagger/index.html` verfügbar, wenn die Anwendung läuft.

## Authentifizierung

Alle API-Endpunkte (außer Login) erfordern JWT-Authentifizierung.

### Anmeldung

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "admin123"}'
```

Antwort:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Token verwenden

Fügen Sie das Token im `Authorization`-Header ein:

```bash
curl http://localhost:8080/api/v1/organizations \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

## API-Endpunkte

### Organisationen

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/organizations` | Alle Organisationen auflisten |
| POST | `/api/v1/organizations` | Organisation erstellen |
| GET | `/api/v1/organizations/{id}` | Organisation abrufen |
| PUT | `/api/v1/organizations/{id}` | Organisation aktualisieren |
| DELETE | `/api/v1/organizations/{id}` | Organisation löschen |

### Mitarbeiter

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/organizations/{orgId}/employees` | Mitarbeiter auflisten |
| POST | `/api/v1/organizations/{orgId}/employees` | Mitarbeiter erstellen |
| GET | `/api/v1/organizations/{orgId}/employees/{id}` | Mitarbeiter abrufen |
| PUT | `/api/v1/organizations/{orgId}/employees/{id}` | Mitarbeiter aktualisieren |
| DELETE | `/api/v1/organizations/{orgId}/employees/{id}` | Mitarbeiter löschen |

### Mitarbeiterverträge

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/organizations/{orgId}/employees/{empId}/contracts` | Verträge auflisten |
| POST | `/api/v1/organizations/{orgId}/employees/{empId}/contracts` | Vertrag erstellen |
| PUT | `/api/v1/organizations/{orgId}/employees/{empId}/contracts/{id}` | Vertrag aktualisieren |
| DELETE | `/api/v1/organizations/{orgId}/employees/{empId}/contracts/{id}` | Vertrag löschen |

### Kinder

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/organizations/{orgId}/children` | Kinder auflisten |
| POST | `/api/v1/organizations/{orgId}/children` | Kind erstellen |
| GET | `/api/v1/organizations/{orgId}/children/{id}` | Kind abrufen |
| PUT | `/api/v1/organizations/{orgId}/children/{id}` | Kind aktualisieren |
| DELETE | `/api/v1/organizations/{orgId}/children/{id}` | Kind löschen |

### Kinderverträge

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/organizations/{orgId}/children/{childId}/contracts` | Verträge auflisten |
| POST | `/api/v1/organizations/{orgId}/children/{childId}/contracts` | Vertrag erstellen |
| PUT | `/api/v1/organizations/{orgId}/children/{childId}/contracts/{id}` | Vertrag aktualisieren |
| DELETE | `/api/v1/organizations/{orgId}/children/{childId}/contracts/{id}` | Vertrag löschen |

### Landesförderung

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/government-fundings` | Förderungskonfigurationen auflisten |
| POST | `/api/v1/government-fundings` | Förderungskonfiguration erstellen |
| GET | `/api/v1/government-fundings/{id}` | Förderungskonfiguration abrufen |
| DELETE | `/api/v1/government-fundings/{id}` | Förderungskonfiguration löschen |

### Benutzer & Gruppen

| Methode | Endpunkt | Beschreibung |
|---------|----------|--------------|
| GET | `/api/v1/users` | Benutzer auflisten |
| POST | `/api/v1/users` | Benutzer erstellen |
| GET | `/api/v1/groups` | Gruppen auflisten |
| POST | `/api/v1/groups` | Gruppe erstellen |

## Paginierung

Listen-Endpunkte unterstützen Paginierung mit Query-Parametern:

```bash
curl "http://localhost:8080/api/v1/organizations?page=1&limit=10"
```

Die Antwort enthält Paginierungs-Metadaten:
```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "limit": 10
}
```

## Fehlerantworten

Fehler werden mit entsprechenden HTTP-Statuscodes zurückgegeben:

```json
{
  "error": "Beschreibung des Fehlers"
}
```

| Status | Bedeutung |
|--------|-----------|
| 400 | Ungültige Anfrage - Ungültige Eingabe |
| 401 | Nicht autorisiert - Fehlendes oder ungültiges Token |
| 403 | Verboten - Unzureichende Berechtigungen |
| 404 | Nicht gefunden - Ressource existiert nicht |
| 500 | Serverfehler - Interner Fehler |
