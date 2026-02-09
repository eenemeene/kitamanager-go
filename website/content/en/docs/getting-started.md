---
title: Getting Started
weight: 1
---

This guide will help you get KitaManager Go up and running quickly.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [Go 1.25+](https://go.dev/dl/) (for development)
- [Node.js 18+](https://nodejs.org/) (for frontend development)

## Quick Start with Docker

The fastest way to get started is using Docker Compose:

```bash
# Clone the repository
git clone https://github.com/toabctl/kitamanager-go.git
cd kitamanager-go

# Start all services
docker compose up -d
```

This will start:
- PostgreSQL database
- KitaManager API server
- Next.js frontend

Access the application at `http://localhost:3000`.

## Development Setup

For local development:

```bash
# Install frontend dependencies
make web-install

# Build the API
make api-build

# Start development environment
make dev
```

### Available Make Targets

| Command | Description |
|---------|-------------|
| `make dev` | Start full development environment |
| `make api-build` | Build the Go API |
| `make api-test` | Run API tests |
| `make web-install` | Install frontend dependencies |
| `make web-dev` | Start frontend dev server |
| `make swagger-docs` | Generate API documentation |

## Default Credentials

After starting, you can log in with the default admin credentials:

| Field | Value |
|-------|-------|
| Email | `admin@example.com` |
| Password | `admin123` |

{{% callout type="warning" %}}
Change the default password immediately in production environments!
{{% /callout %}}

## Seed Data

The development environment includes seed data with:

- A sample organization "Kita Sonnenschein"
- 50 test children with contracts
- Sample employees
- Berlin state government funding configuration

## Next Steps

- [Architecture Overview](../architecture) - Understand the system design
- [API Reference](../api) - Explore the REST API
- [Deployment Guide](../deployment) - Production deployment options
