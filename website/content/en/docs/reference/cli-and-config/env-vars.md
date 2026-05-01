---
title: Environment variables
weight: 1
---

KitaManager is configured almost entirely through environment variables. The API binary loads them from process environment plus an optional `.env` file in the working directory at startup. The list below mirrors `internal/config/config.go`; **required** variables make `Load()` fail if missing or placeholder.

## Database

| Variable | Default | Required | Notes |
|---|---|---|---|
| `DB_HOST` | `localhost` | no | Postgres host |
| `DB_PORT` | `5432` | no | Postgres port |
| `DB_USER` |  | **yes** | Postgres user |
| `DB_PASSWORD` |  | **yes** | Postgres password |
| `DB_NAME` |  | **yes** | Postgres database name |
| `DB_SSLMODE` | `require` | no | One of `disable`, `require`, `verify-ca`, `verify-full`. Use `disable` only for local development. |
| `DB_MAX_IDLE_CONNS` | `10` | no | GORM connection-pool tuning. |
| `DB_MAX_OPEN_CONNS` | `100` | no | Same. |
| `DB_CONN_MAX_LIFE_MIN` | `60` | no | Recycle connection after this many minutes. |
| `DB_CONN_MAX_IDLE_MIN` | `10` | no | Close idle connection after this many minutes. |

## HTTP server

| Variable | Default | Required | Notes |
|---|---|---|---|
| `SERVER_PORT` | `8080` | no | Listen port. |
| `TRUSTED_PROXIES` | (none) | no | Comma-separated list of CIDRs/IPs whose `X-Forwarded-*` headers Gin will trust. Behind one reverse proxy, set this to the proxy's IP. Empty means trust no proxy. |
| `SECURE_COOKIES` | `true` | no | When `true`, session cookies are `Secure` (HTTPS only). Set `false` only for local HTTP dev. |
| `CORS_ALLOW_ORIGINS` | (none) | no | Comma-separated list of allowed origins. Empty disables CORS. |
| `CORS_ALLOW_CREDENTIALS` | `true` | no | Whether CORS responses include `Access-Control-Allow-Credentials: true`. |

## Authentication and sessions

| Variable | Default | Required | Notes |
|---|---|---|---|
| `JWT_SECRET` |  | **yes** | At least 32 chars. Used to sign session tokens. Rotating it logs out every active session. |
| `LOGIN_RATE_LIMIT_PER_MINUTE` | `5` | no | Per-IP login attempts per minute. `0` disables. |
| `API_RATE_LIMIT_PER_MINUTE` | `60` | no | Per-IP authenticated request rate. `0` disables. |

## Multi-factor authentication

| Variable | Default | Required | Notes |
|---|---|---|---|
| `TOTP_ENCRYPTION_KEY` |  | **yes** | Exactly 64 hex chars (32 bytes). Encrypts stored TOTP secrets at rest. **Rotating this key invalidates every stored TOTP factor** — affected users must re-enrol. Generate with `openssl rand -hex 32`. |
| `TOTP_ISSUER` | `KitaManager` | no | The issuer string shown in the user's authenticator app. |
| `WEBAUTHN_RP_ID` |  | only if WebAuthn is used | The Relying Party ID — the host part of your URL (e.g. `kitamanager.example.org`). Must match the URL the browser sees. |
| `WEBAUTHN_RP_NAME` |  | only if WebAuthn is used | Human-readable name shown in the browser's security-key prompt. |
| `WEBAUTHN_ORIGINS` |  | only if WebAuthn is used | Comma-separated origin URLs (e.g. `https://kitamanager.example.org`). |

## RBAC

| Variable | Default | Required | Notes |
|---|---|---|---|
| `RBAC_MODEL_PATH` | `configs/rbac_model.conf` | no | Path to the Casbin model file. Default works for both the binary and the Docker image. |

## Seed data and bootstrap

These let an empty database bootstrap with a usable account on first start. Set them only on first run; subsequent restarts ignore them if the user already exists.

| Variable | Default | Required | Notes |
|---|---|---|---|
| `SEED_ADMIN_EMAIL` |  | required to seed the bootstrap superadmin (skipped if empty) | Email of the initial superadmin. |
| `SEED_ADMIN_PASSWORD` |  | required if `SEED_ADMIN_EMAIL` is set | Initial superadmin password. Change it after first login. |
| `SEED_ADMIN_NAME` | `admin` | no | Display name for the seeded superadmin. |
| `SEED_TEST_DATA` | `false` | no | When `true`, seeds the demo "Kita Sonnenschein" organisation, sections, employees, children, contracts. **Never set this in production.** |
| `GOVERNMENT_FUNDING_SEED_PATH` |  | no | Path to a YAML funding-rate file to load on startup. Empty skips. Typical value: `configs/government-fundings/berlin.yaml`. |
| `GOVERNMENT_FUNDING_SEED_STATE` | `berlin` | no | The state name used to label the loaded funding configuration. |

## Time zone

| Variable | Default | Required | Notes |
|---|---|---|---|
| `KITAMANAGER_TIMEZONE` | `Europe/Berlin` | no | The application's calendar time zone. Used by every `models.Today()` call. Override with an IANA zone name. |

## Logging

| Variable | Default | Required | Notes |
|---|---|---|---|
| `LOG_LEVEL` | `info` | no | One of `debug`, `info`, `warn`, `error`. |
| `LOG_FORMAT` | `json` | no | Either `json` or `text`. |

## Email (optional)

Used for password reset / notifications. If `SMTP_HOST` is empty, email features are disabled.

| Variable | Default | Required |
|---|---|---|
| `SMTP_HOST` |  | no |
| `SMTP_PORT` | `587` | no |
| `SMTP_USER` |  | no |
| `SMTP_PASSWORD` |  | no |
| `SMTP_FROM` |  | no |

## report-pdf tool

The PDF-generating sidecar reads its own `KITAMANAGER_REPORT_*` variables. Each one mirrors a CLI flag of the same name. See [The report tool](../../../explanation/architecture/the-report-tool/) and the tool's [README](https://github.com/eenemeene/kitamanager-go/tree/main/tools/report-pdf) for the full list.
