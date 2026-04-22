# KitaManager

A web application for managing kindergarten (Kita) organizations, employees, children, and contracts.

**Documentation:** [eenemeene.github.io/kitamanager-go](https://eenemeene.github.io/kitamanager-go/)

## Development setup

The API refuses to start unless every required env var is set — there is no dev/staging/production flag that relaxes security rules. You configure the system by giving it the right env vars for what you are running.

### Quick start (local development)

```bash
cp .env.dev.example .env
make web-install
make api-build
make dev
```

`make dev` brings up Postgres in Docker, starts the API, and runs the frontend with hot reload. If `.env` is missing it creates one from `.env.dev.example` automatically.

Or run the full stack in Docker:

```bash
cp .env.dev.example .env
docker compose up -d --build
```

Both paths read config from `.env`. Docker compose overrides `DB_HOST` to the in-network service name (`db`) so you can use the same `.env` for running the binary directly and inside compose.

- **Web UI:** http://localhost:3000
- **API:** http://localhost:8080
- **Login:** `admin@example.com` / `supersecret`

The `.env.dev.example` file ships with dev-only values (`DB_SSLMODE=disable`, a well-known `JWT_SECRET`, `SECURE_COOKIES=false`, rate limiters disabled, test data seeded). These values are safe to use on your laptop and nowhere else.

### Required env vars (all environments)

| Var | Rule |
|---|---|
| `JWT_SECRET` | Required. ≥32 characters. Must not be a known placeholder string. Used for CSRF-token HMAC derivation; name is kept for backwards compatibility. |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | Required. |
| `DB_SSLMODE` | One of `disable`, `require`, `verify-ca`, `verify-full`. |
| `SERVER_PORT` | Valid TCP port. Defaults to `8080`. |
| `CORS_ALLOW_ORIGINS` | Exact origins, comma-separated. `*` is permitted only when `CORS_ALLOW_CREDENTIALS=false`. |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error`. |
| `LOG_FORMAT` | `json` or `text`. |

Anything missing or invalid fails startup with an explicit error.

### Useful commands

| Command | What it does |
|---|---|
| `make dev` | Database + API + hot-reloaded frontend |
| `make dev-fresh` | Same, but wipes the DB volume first |
| `make test` | Web + API unit tests |
| `make ci` | Lint, test, and build — everything CI runs except integration/e2e |
| `make api-test-integration` | Integration tests against Postgres |
| `make web-test-e2e` | Playwright E2E tests |
| `make api-lint` | `golangci-lint run ./...` |
| `make swagger-docs` | Regenerate the OpenAPI spec in `docs/` |

See [DEVELOPMENT.md](DEVELOPMENT.md) for the full set of Makefile targets and developer notes.

### Testing

Unit tests: `make api-test-unit` (requires nothing; uses in-process SQLite via testutil). Integration tests: `make api-test-integration` (requires a running Postgres — `docker compose up -d db` is enough).

## Running in production

Use `.env.production.example` as a template:

```bash
cp .env.production.example .env
$EDITOR .env   # replace every REPLACE_ME
openssl rand -hex 32   # for JWT_SECRET
```

Then bring the stack up with the demo compose file (which enforces `DB_SSLMODE=require`, `SECURE_COOKIES=true`, and refuses to start without `JWT_SECRET` / `DB_PASSWORD` / `SEED_ADMIN_PASSWORD`):

```bash
docker compose -f docker-compose.demo.yml up -d --build
```

Behind a reverse proxy you must set `TRUSTED_PROXIES` to the proxy's CIDRs, otherwise `c.ClientIP()` returns the proxy's address and rate limits group every request together.

Rotate `SEED_ADMIN_PASSWORD` after the first login. Do not set `SEED_TEST_DATA=true` in production.

## License

[AGPL-3.0](LICENSE)
