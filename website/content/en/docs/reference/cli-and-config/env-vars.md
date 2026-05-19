---
title: Environment variables
weight: 1
---

KitaManager is configured almost entirely through environment variables. The API binary loads them from process environment plus an optional `.env` file in the working directory at startup. The list below mirrors `internal/config/config.go`; **required** variables make `Load()` fail if missing or placeholder. A few entries (`KITAMANAGER_TIMEZONE`, `SEED_RBAC_POLICIES`) are read directly by other packages (`internal/models/clock.go`, `cmd/api/main.go`) rather than by `config.Load()`, but they share the same conventions.

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
| `TRUSTED_PROXIES` | (none) | no | Comma-separated list of CIDRs/IPs whose `X-Forwarded-*` headers Gin will trust. Behind one reverse proxy, set this to the proxy's IP. Empty means trust no proxy. `0.0.0.0/0` and `::/0` are rejected — they would defeat per-IP rate limiting by letting any client spoof its source address. |
| `SECURE_COOKIES` | `true` | no | When `true`, session cookies are `Secure` (HTTPS only) and the loader enforces production gates (no `DB_SSLMODE=disable`, rate limits > 0, no `SEED_TEST_DATA`). Set `false` only for local HTTP dev. |
| `CORS_ALLOW_ORIGINS` | (none) | no | Comma-separated list of allowed origins. Empty disables CORS. |
| `CORS_ALLOW_CREDENTIALS` | `true` | no | Whether CORS responses include `Access-Control-Allow-Credentials: true`. |

## Production gate opt-outs

These exist only as conscious escape hatches for the production gates that `SECURE_COOKIES=true` activates. Default to leaving them unset; use them only for known constrained environments (e.g. an isolated LAN where the DB really cannot serve TLS).

| Variable | Default | Required | Notes |
|---|---|---|---|
| `ALLOW_RATE_LIMIT_DISABLED_IN_PRODUCTION` | `false` | no | When `true`, permits `LOGIN_RATE_LIMIT_PER_MINUTE=0` or `API_RATE_LIMIT_PER_MINUTE=0` even with `SECURE_COOKIES=true`. |
| `ALLOW_DB_SSLMODE_DISABLE_IN_PRODUCTION` | `false` | no | When `true`, permits `DB_SSLMODE=disable` even with `SECURE_COOKIES=true`. |
| `AUDIT_LOG_RETENTION_DAYS` | `730` | no | How long mutating-action audit log rows are kept before the retention worker deletes them. DSGVO Art. 17 obligation; do not lower below 365 without legal review. Audit writes are asynchronous (channel buffer 4096) with a 5 s synchronous fallback; on a double failure the row is dropped and `audit_entries_dropped_total` increments — alert on any non-zero rate. |

## Authentication and sessions

| Variable | Default | Required | Notes |
|---|---|---|---|
| `CSRF_HMAC_KEY` |  | **yes** | At least 32 chars; not a known placeholder. HMAC key used to derive the double-submit CSRF token from the session cookie. Rotating it invalidates every outstanding CSRF token (users get a 403 once on their next state-changing request, then a fresh token is re-issued). |
| `LOGIN_RATE_LIMIT_PER_MINUTE` | `5` | no | Per-IP login attempts per minute. `0` disables. |
| `API_RATE_LIMIT_PER_MINUTE` | `60` | no | Per-IP authenticated request rate. `0` disables. |

## Multi-factor authentication

| Variable | Default | Required | Notes |
|---|---|---|---|
| `TOTP_ENCRYPTION_KEY` |  | **yes** | Exactly 64 hex chars (32 bytes). Encrypts stored TOTP secrets at rest. **Rotating this key invalidates every stored TOTP factor** — affected users must re-enrol. Generate with `openssl rand -hex 32`. Uniform-byte values (e.g. 64 zeros or 64 ones) and known placeholders are rejected at startup — the loader requires real entropy, not a typed-by-hand string. |
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
| `SEED_RBAC_POLICIES` | `false` | no | When `true`, seeds the default Casbin role-permission policies on startup. Required exactly once on a fresh database; harmless on subsequent starts but unnecessary. |
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
