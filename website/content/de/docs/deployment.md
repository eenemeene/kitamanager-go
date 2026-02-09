---
title: Bereitstellung
weight: 4
---

Diese Anleitung behandelt Bereitstellungsoptionen für KitaManager Go.

## Docker-Bereitstellung

Die empfohlene Bereitstellungsmethode ist Docker.

### Produktions-Docker-Compose

Erstellen Sie eine `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:18-alpine
    environment:
      POSTGRES_DB: kitamanager
      POSTGRES_USER: kitamanager
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  api:
    image: kitamanager-go:latest
    environment:
      DATABASE_URL: postgres://kitamanager:${DB_PASSWORD}@postgres:5432/kitamanager
      JWT_SECRET: ${JWT_SECRET}
      GIN_MODE: release
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    restart: unless-stopped

volumes:
  postgres_data:
```

### Docker-Image bauen

```bash
docker build -t kitamanager-go:latest .
```

Das mehrstufige Dockerfile baut:
1. Go API-Binary
2. Next.js-Frontend (eingebettet in das Binary)
3. Finales minimales Image

## Umgebungsvariablen

| Variable | Beschreibung | Erforderlich |
|----------|--------------|--------------|
| `DATABASE_URL` | PostgreSQL-Verbindungsstring | Ja |
| `JWT_SECRET` | Geheimnis für JWT-Token-Signierung | Ja |
| `GIN_MODE` | Auf `release` für Produktion setzen | Empfohlen |
| `PORT` | API-Server-Port (Standard: 8080) | Nein |
| `LOG_LEVEL` | Logging-Level (debug, info, warn, error) | Nein |

## Datenbank-Einrichtung

### Initiale Migration

Die Anwendung führt Migrationen beim Start automatisch aus. Für manuelle Kontrolle:

```bash
# Migrationen ausführen
./kitamanager-go migrate up

# Initiale Daten einspielen
./kitamanager-go seed
```

### Backups

Regelmäßige PostgreSQL-Backups werden empfohlen:

```bash
# Backup
pg_dump -h localhost -U kitamanager kitamanager > backup.sql

# Wiederherstellen
psql -h localhost -U kitamanager kitamanager < backup.sql
```

## Sicherheitsüberlegungen

### Produktions-Checkliste

{{% callout type="warning" %}}
Vor der Produktionsbereitstellung sicherstellen:
{{% /callout %}}

- [ ] Standard-Admin-Passwort ändern
- [ ] Starkes `JWT_SECRET` setzen
- [ ] HTTPS verwenden (Reverse-Proxy konfigurieren)
- [ ] `GIN_MODE=release` setzen
- [ ] Korrekte CORS-Origins konfigurieren
- [ ] Rate-Limiting aktivieren
- [ ] Datenbank-Backups einrichten

### Reverse-Proxy

Verwenden Sie nginx oder Caddy als Reverse-Proxy mit HTTPS:

```nginx
server {
    listen 443 ssl http2;
    server_name kitamanager.example.org;

    ssl_certificate /etc/letsencrypt/live/kitamanager.example.org/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/kitamanager.example.org/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Monitoring

### Health-Checks

Die API bietet Health-Check-Endpunkte:

```bash
# Liveness-Probe
curl http://localhost:8080/health/live

# Readiness-Probe
curl http://localhost:8080/health/ready
```

### Logging

Strukturierte JSON-Logs werden nach stdout ausgegeben:

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "Anfrage abgeschlossen",
  "method": "GET",
  "path": "/api/v1/organizations",
  "status": 200,
  "duration": "15ms"
}
```

Verwenden Sie einen Log-Aggregator wie Loki, ELK oder CloudWatch für zentralisiertes Logging.

## Skalierung

### Horizontale Skalierung

Die API ist zustandslos und kann horizontal skaliert werden:

```yaml
services:
  api:
    image: kitamanager-go:latest
    deploy:
      replicas: 3
```

Verwenden Sie einen Load-Balancer, um den Traffic auf die Instanzen zu verteilen.

### Datenbank-Skalierung

Für Hochlast-Szenarien:
- Connection-Pooling aktivieren (PgBouncer)
- Lese-Replikate konfigurieren
- Managed PostgreSQL-Dienste in Betracht ziehen
