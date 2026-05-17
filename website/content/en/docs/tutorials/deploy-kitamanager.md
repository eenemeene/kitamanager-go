---
title: Deploy KitaManager
weight: 2
---

This tutorial walks you through bringing up a fresh KitaManager instance with Docker Compose, on a machine you control. By the end you'll have a working stack with the Berlin funding rates loaded, a superadmin account, and a sane production-ish configuration.

You'll need:

- A Linux host with Docker and Docker Compose installed.
- About 30 minutes the first time, mostly waiting for image pulls.
- Open ports for the frontend (default `3000`) and API (default `8080`). For a production deploy, you'd put both behind an HTTPS reverse proxy — that's covered at the end.

## Step 1 — Get the code

```bash
git clone https://github.com/eenemeene/kitamanager-go.git
cd kitamanager-go
git checkout v0.34.0   # or your chosen release tag
```

Pinning to a tag is recommended over tracking `main` for production. New tags ship as Docker images automatically.

## Step 2 — Configure secrets

Copy the production env example and fill it in:

```bash
cp .env.production.example .env
```

Open `.env` in your editor. The required values are:

```
DB_USER=kitamanager
DB_PASSWORD=<long random>
DB_NAME=kitamanager

CSRF_HMAC_KEY=<openssl rand -hex 32>
TOTP_ENCRYPTION_KEY=<openssl rand -hex 32>

WEBAUTHN_RP_ID=<your domain, e.g. kitamanager.example.org>
WEBAUTHN_RP_NAME=KitaManager
WEBAUTHN_ORIGINS=https://kitamanager.example.org

SEED_ADMIN_EMAIL=admin@your-org.example
SEED_ADMIN_PASSWORD=<temporary password — change immediately after first sign-in>
SEED_ADMIN_NAME=System Admin

GOVERNMENT_FUNDING_SEED_PATH=configs/government-fundings/berlin.yaml
GOVERNMENT_FUNDING_SEED_STATE=berlin

SECURE_COOKIES=true
DB_SSLMODE=require
```

**Do not skip `CSRF_HMAC_KEY` or `TOTP_ENCRYPTION_KEY`.** The validator refuses to start without strong values.

For the full reference, see [Environment variables](../../reference/cli-and-config/env-vars/).

## Step 3 — Bring it up

```bash
docker compose up -d
```

The first start downloads images, runs migrations, loads the Berlin funding rates from YAML, and creates your bootstrap superadmin. Watch the logs:

```bash
docker compose logs -f api
```

You're looking for `KitaManager API ready` (or similar) and no migration errors.

## Step 4 — Verify

Health check the API:

```bash
curl -sf http://localhost:8080/api/v1/health
# Expect: {"status":"ok"}
```

Open `http://localhost:3000` and sign in with `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD`. You land on the dashboard.

## Step 5 — Lock the bootstrap account down

Immediately:

1. Change the bootstrap password in **Settings → Password**.
2. Enable 2FA in **Settings → Two-factor authentication**. Save the recovery codes somewhere durable.
3. (Optional but recommended) add a security key under **Add security key**.

Until this is done, anyone with the env file can sign in.

## Step 6 — Reverse proxy with HTTPS (production)

For anything beyond a local demo, put HTTPS in front. Caddy is the simplest:

```caddy
kitamanager.example.org {
    reverse_proxy localhost:3000
}

api.kitamanager.example.org {
    reverse_proxy localhost:8080
}
```

Update `WEBAUTHN_RP_ID`, `WEBAUTHN_ORIGINS`, and `CORS_ALLOW_ORIGINS` to match the public URLs. Restart the API.

For nginx, Apache, Traefik: configure the same — terminate TLS at the proxy, forward to ports `3000` (frontend) and `8080` (API).

## Step 7 — Take a backup

Before you put real data in, prove your backup pipeline works. Follow the [Back up and restore](../../how-to/operate/back-up-and-restore/) recipe.

## You're done

You have a deployed KitaManager. From here:

- Set up users for your team — see [Manage users and roles](../../how-to/administer/create-a-user/).
- Walk a teacher through the day-to-day — point them at [First day in KitaManager](../first-day-in-kitamanager/).
- Schedule the [recurring tasks](../../how-to/operate/) that keep the system healthy: backups, funding-rate updates, key rotation, releases.
