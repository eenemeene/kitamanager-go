# Development Guide

## Prerequisites

- Go 1.24+
- Node.js 22+
- Docker (for database)
- [pre-commit](https://pre-commit.com/) (optional, for git hooks)

## Quick Start

```bash
# Install dependencies
make web-install

# Build the API
make api-build

# Start full dev environment (database + API + web with hot reload)
make dev
```

This starts:
- **Web UI** at http://localhost:3000 (with hot reload)
- **API** at http://localhost:8080
- **PostgreSQL** via Docker

Press `Ctrl+C` to stop all services.

## Test Data Seeding

When `SEED_TEST_DATA=true` (enabled by default in `make dev`), the following test data is created:

### Test Users

All test users have password: `supersecret`

| Email | Role | Permissions |
|-------|------|-------------|
| `superadmin@example.com` | Superadmin | Full system access across all organizations |
| `admin@example.com` | Admin | Full access within the test organization |
| `manager@example.com` | Manager | Operational access (employees, children, contracts); read-only for users/groups |

### Test Organization

- **Name**: Kita Sonnenschein
- **Group**: Mitarbeiter (default group)
- **Government Funding**: Berlin funding plan
- **Children**: 50 children with age-appropriate distribution and contracts

See [RBAC documentation](docs/RBAC.md) for detailed permission matrix.

## Makefile Targets

### Development

| Target | Description |
|--------|-------------|
| `make dev` | Start full dev environment (DB + API + Web with hot reload) |
| `make build` | Build both web and API |
| `make test` | Run all tests (web + API) |
| `make lint` | Run linters (web + API) |
| `make ci` | Run all CI checks locally |
| `make clean` | Remove build artifacts |

### API

| Target | Description |
|--------|-------------|
| `make api-build` | Build API to `bin/kitamanager-api` |
| `make api-run` | Run API with `go run` |
| `make api-test-unit` | Run API unit tests |
| `make api-test-integration` | Run integration tests (requires DB) |
| `make api-test-contract` | Run contract tests (requires DB) |
| `make api-test-coverage` | Run tests with coverage report |
| `make api-lint` | Run Go linter |

### Web (Next.js Frontend)

| Target | Description |
|--------|-------------|
| `make web-install` | Install npm dependencies |
| `make web-dev` | Start Next.js dev server only |
| `make web-build` | Build for production |
| `make web-test` | Run Jest tests |
| `make web-test-coverage` | Run tests with coverage |
| `make web-test-e2e` | Run Playwright E2E tests |
| `make web-test-e2e-headed` | Run E2E tests with browser visible |
| `make web-lint` | Run ESLint |
| `make web-format` | Format code with Prettier |
| `make web-type-check` | TypeScript type checking |

### Docker

| Target | Description |
|--------|-------------|
| `make docker-up` | Start docker containers |
| `make docker-down` | Stop docker containers |
| `make docker-rebuild` | Rebuild and restart containers |

### Documentation

| Target | Description |
|--------|-------------|
| `make docs` | Generate all documentation |
| `make swagger-docs` | Generate OpenAPI/Swagger docs |
| `make schema-docs` | Generate database schema docs |

### Git Hooks

| Target | Description |
|--------|-------------|
| `make install-hooks` | Install pre-commit hooks |
| `make uninstall-hooks` | Uninstall pre-commit hooks |
| `make pre-commit` | Run pre-commit on all files |

## Project Structure

```
.
├── cmd/api/            # API entry point
├── internal/
│   ├── config/         # Configuration
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # HTTP middleware (auth, CORS, etc.)
│   ├── models/         # Database models
│   ├── rbac/           # Role-based access control
│   ├── store/          # Database stores
│   ├── integration/    # Integration tests
│   └── contract/       # API contract tests
├── frontend/           # Next.js frontend
│   ├── src/
│   │   ├── app/        # Next.js App Router pages
│   │   ├── components/ # React components
│   │   ├── lib/        # API client, hooks, utilities
│   │   ├── i18n/       # Internationalization
│   │   └── stores/     # Zustand stores
│   └── e2e/            # Playwright E2E tests
├── docs/               # Documentation
└── configs/            # Configuration files
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run only API tests
make api-test-unit

# Run only web tests
make web-test

# Run E2E tests
make web-test-e2e

# Run with coverage
make api-test-coverage
make web-test-coverage
```

### Test Tags

- **Unit tests**: No tags, run with `go test ./...`
- **Integration tests**: Use `-tags=integration`, require database
- **Contract tests**: Use `-tags=contract`, require database

## Code Style

- **Go**: Uses `golangci-lint` with project configuration
- **TypeScript/React**: Uses ESLint and Prettier
- **Commits**: Follow [Conventional Commits](https://www.conventionalcommits.org/)

Pre-commit hooks enforce these standards automatically.

## API Documentation

Swagger/OpenAPI documentation is available at http://localhost:8080/swagger/index.html when the API is running.

To regenerate after handler changes:

```bash
make swagger-docs
```
