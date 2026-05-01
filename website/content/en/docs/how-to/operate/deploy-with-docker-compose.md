---
title: Deploy with Docker Compose
weight: 1
---

You want to bring up KitaManager (Postgres + API + frontend) on a host using Docker Compose. This is the same path as the [Deploy KitaManager](../../../tutorials/deploy-kitamanager/) tutorial, but condensed into a recipe for someone who already knows what they're doing.

## Prerequisites

- Docker and Docker Compose installed.
- Open ports for the frontend and API (defaults `3000` and `8080`).
- A long random `JWT_SECRET` (at least 32 chars) and a 64-char hex `TOTP_ENCRYPTION_KEY`.

## Steps

1. Clone or check out the repo at the version you want to run.
2. Copy `.env.production.example` to `.env` and fill in:
   - `DB_USER`, `DB_PASSWORD`, `DB_NAME`
   - `JWT_SECRET` — `openssl rand -hex 32`
   - `TOTP_ENCRYPTION_KEY` — `openssl rand -hex 32`
   - `WEBAUTHN_RP_ID`, `WEBAUTHN_RP_NAME`, `WEBAUTHN_ORIGINS` if you'll use security keys
   - `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`, `SEED_ADMIN_NAME` (for the bootstrap superadmin)
   - `GOVERNMENT_FUNDING_SEED_PATH=configs/government-fundings/berlin.yaml` to load Berlin rates on first start
3. Bring it up:
   ```bash
   docker compose up -d
   ```
4. Wait for the API health check to pass:
   ```bash
   curl -sf http://localhost:8080/api/v1/health
   ```
5. Sign in at `http://localhost:3000` with your `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD`. Change the password immediately, enable 2FA.

## Notes

- For the full env-var reference, see [Environment variables](../../../reference/cli-and-config/env-vars/).
- For the *first-time* walkthrough with seed data, follow the [Deploy KitaManager](../../../tutorials/deploy-kitamanager/) tutorial instead.
- Production checklist:
  - `SECURE_COOKIES=true`
  - `DB_SSLMODE=require` (or stricter)
  - `LOGIN_RATE_LIMIT_PER_MINUTE` left at default (5) and `API_RATE_LIMIT_PER_MINUTE` (60), or tuned to your traffic
  - **Never** set `SEED_TEST_DATA=true` in production
  - Take the first backup immediately — see [Back up and restore](../back-up-and-restore/)
