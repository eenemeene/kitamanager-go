---
title: Getting Started
weight: 1
---

This guide will help you get KitaManager up and running quickly.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [Go 1.25+](https://go.dev/dl/) (for development)
- [Node.js 24+](https://nodejs.org/) (for frontend development)

## Quick Start with Docker

The fastest way to get started is using Docker Compose:

```bash
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

After starting, you can sign in with one of the seeded test accounts. Each demonstrates a different role:

| Email | Role | Password |
|-------|------|----------|
| `superadmin@example.com` | Superadmin | `supersecret` |
| `admin@example.com` | Admin (Kita Sonnenschein) | `supersecret` |
| `manager@example.com` | Manager (Kita Sonnenschein) | `supersecret` |

{{% callout type="warning" %}}
These credentials are for local development only. Change all default passwords immediately in production environments and enable two-factor authentication for every account that can edit data.
{{% /callout %}}

## Seed Data

The development environment includes seed data with:

- A sample organization "Kita Sonnenschein" with three sections (Nest, Nestflüchter, Große)
- ~120 children with care contracts and realistic age distributions
- ~35 employees with employment contracts across all sections
- Pay plans (TVöD-SuE 2024 and Minijob)
- Budget items (parent contributions and operational costs)
- Berlin state government funding rates
- Three test users with different roles (superadmin, admin, manager)

## Next Steps

- [Architecture Overview](../architecture/) - Understand the system design
- [API Reference](../api/) - Explore the REST API
