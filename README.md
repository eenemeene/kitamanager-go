# KitaManager Go

A REST API for managing kindergarten (Kita) organizations, employees, children, and contracts.

## Quick Start

### Using Docker Compose (Recommended)

Start the API and PostgreSQL database:

```bash
docker compose up -d
```

This starts:
- **API** at http://localhost:8080
- **PostgreSQL 18** at localhost:5432

To rebuild after code changes:

```bash
docker compose up -d --build
```

To stop:

```bash
docker compose down
```

To stop and remove data:

```bash
docker compose down -v
```

### Local Development

Prerequisites:
- Go 1.24+
- PostgreSQL 18+

```bash
# Copy environment template
cp .env.example .env

# Edit .env with your database settings
vim .env

# Run the API
make run
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile to `bin/kitamanager-api` |
| `make run` | Run locally with `go run` |
| `make test` | Run all tests |
| `make clean` | Remove build artifacts |
| `make schema-docs` | Regenerate database schema docs |
| `make docker-up` | Start docker containers |
| `make docker-down` | Stop docker containers |
| `make docker-rebuild` | Rebuild and restart containers |

## API Endpoints

### Authentication
- `POST /api/v1/login` - Login, returns JWT token

### Organizations (Superadmin only for create/delete)
- `GET /api/v1/organizations` - List all
- `POST /api/v1/organizations` - Create (superadmin)
- `GET /api/v1/organizations/:id` - Get by ID
- `PUT /api/v1/organizations/:id` - Update
- `DELETE /api/v1/organizations/:id` - Delete (superadmin)

### Employees (Organization-scoped)
- `GET /api/v1/organizations/:orgId/employees` - List
- `POST /api/v1/organizations/:orgId/employees` - Create
- `GET /api/v1/organizations/:orgId/employees/:id` - Get
- `PUT /api/v1/organizations/:orgId/employees/:id` - Update
- `DELETE /api/v1/organizations/:orgId/employees/:id` - Delete
- `POST /api/v1/organizations/:orgId/employees/:id/contracts` - Add contract
- `DELETE /api/v1/organizations/:orgId/employees/:id/contracts/:contractId` - Remove contract

### Children (Organization-scoped)
- `GET /api/v1/organizations/:orgId/children` - List
- `POST /api/v1/organizations/:orgId/children` - Create
- `GET /api/v1/organizations/:orgId/children/:id` - Get
- `PUT /api/v1/organizations/:orgId/children/:id` - Update
- `DELETE /api/v1/organizations/:orgId/children/:id` - Delete
- `POST /api/v1/organizations/:orgId/children/:id/contracts` - Add contract
- `DELETE /api/v1/organizations/:orgId/children/:id/contracts/:contractId` - Remove contract

### Users & Groups
- Standard CRUD at `/api/v1/users` and `/api/v1/groups`

## Documentation

- [RBAC System](docs/RBAC.md) - Role-based access control
- [Database Schema](docs/schema/README.md) - Auto-generated ER diagram

## Development Guidelines

See [CLAUDE.md](CLAUDE.md) for development guidelines including:
- API handler documentation requirements
- RBAC middleware usage
- Database schema change procedures
