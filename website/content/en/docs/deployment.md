---
title: Deployment
weight: 4
---

This guide covers deployment options for KitaManager Go.

## Docker Deployment

The recommended deployment method is using Docker.

### Production Docker Compose

Create a `docker-compose.prod.yml`:

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

### Building the Docker Image

```bash
docker build -t kitamanager-go:latest .
```

The multi-stage Dockerfile builds:
1. Go API binary
2. Next.js frontend (embedded into the binary)
3. Final minimal image

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | PostgreSQL connection string | Yes |
| `JWT_SECRET` | Secret for signing JWT tokens | Yes |
| `GIN_MODE` | Set to `release` for production | Recommended |
| `PORT` | API server port (default: 8080) | No |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | No |

## Database Setup

### Initial Migration

The application runs migrations automatically on startup. For manual control:

```bash
# Run migrations
./kitamanager-go migrate up

# Seed initial data
./kitamanager-go seed
```

### Backups

Regular PostgreSQL backups are recommended:

```bash
# Backup
pg_dump -h localhost -U kitamanager kitamanager > backup.sql

# Restore
psql -h localhost -U kitamanager kitamanager < backup.sql
```

## Security Considerations

### Production Checklist

{{% callout type="warning" %}}
Before deploying to production, ensure:
{{% /callout %}}

- [ ] Change default admin password
- [ ] Set strong `JWT_SECRET`
- [ ] Use HTTPS (configure reverse proxy)
- [ ] Set `GIN_MODE=release`
- [ ] Configure proper CORS origins
- [ ] Enable rate limiting
- [ ] Set up database backups

### Reverse Proxy

Use nginx or Caddy as a reverse proxy with HTTPS:

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

### Health Checks

The API provides health check endpoints:

```bash
# Liveness probe
curl http://localhost:8080/health/live

# Readiness probe
curl http://localhost:8080/health/ready
```

### Logging

Structured JSON logs are output to stdout:

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "Request completed",
  "method": "GET",
  "path": "/api/v1/organizations",
  "status": 200,
  "duration": "15ms"
}
```

Use a log aggregator like Loki, ELK, or CloudWatch for centralized logging.

## Scaling

### Horizontal Scaling

The API is stateless and can be horizontally scaled:

```yaml
services:
  api:
    image: kitamanager-go:latest
    deploy:
      replicas: 3
```

Use a load balancer to distribute traffic across instances.

### Database Scaling

For high-load scenarios:
- Enable connection pooling (PgBouncer)
- Configure read replicas
- Consider managed PostgreSQL services
