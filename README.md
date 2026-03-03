# KitaManager

A web application for managing kindergarten (Kita) organizations, employees, children, and contracts.

**Documentation:** [eenemeene.github.io/kitamanager-go](https://eenemeene.github.io/kitamanager-go/)

## Quick Start

### Using Docker Compose (Recommended)

Start the API and PostgreSQL database:

```bash
docker compose up -d
```

This starts:
- **API** at http://localhost:8080
- **Frontend** at http://localhost:3000
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

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed development setup and Makefile targets.

Quick start:

```bash
make web-install   # Install dependencies
make api-build     # Build API
make dev           # Start dev environment with hot reload
```

## License

[AGPL-3.0](LICENSE)
