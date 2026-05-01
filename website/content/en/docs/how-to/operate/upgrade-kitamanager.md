---
title: Upgrade KitaManager to a new version
weight: 11
---

You want to move from version vX to vY.

## Steps

1. **Read the release notes** at https://github.com/eenemeene/kitamanager-go/releases/tag/vY for breaking changes, new env vars, and migration cautions.
2. **Take a backup** of the current database. See [Back up and restore](../back-up-and-restore/). Test that the dump restores into a scratch database.
3. **Update your image tag** in the production `docker-compose.yml`. The bundled compose file in this repo uses `build:` for local dev — for production you'd have a separate compose with `image:` lines pinned to specific tags:
   ```bash
   sed -i 's|kitamanager:vX|kitamanager:vY|; s|kitamanager-ui:vX|kitamanager-ui:vY|; s|kitamanager-report:vX|kitamanager-report:vY|' docker-compose.yml
   ```
   For the canonical image names see [Publish a release](../publish-a-release/).
4. **Pull the new images**:
   ```bash
   docker compose pull
   ```
5. **Apply env-var changes** (if the release notes added or renamed any).
6. **Restart**:
   ```bash
   docker compose up -d
   ```
   Migrations run automatically on API start.
7. **Verify**:
   ```bash
   curl -sf http://localhost:8080/api/v1/health
   ```
   Sign in. Check the dashboard for unexpected errors. Check the audit log for failed migrations.

## Downtime expectation

The API is unavailable from "old container stopping" to "new container's migration finishes and health check passes". For most upgrades this is seconds. A schema migration that touches a large table (rare) can take minutes.

## If a migration fails

The API exits non-zero and Docker Compose marks it unhealthy. The schema is left in whatever partial state the migration reached. Restore from your backup, file an issue with the failing migration's number, and roll back to the previous image tag while waiting for a fix.

## Notes

- Pin to specific tags in production (`kitamanager:v0.34.0`), never `:latest`. The release workflow guarantees a tag never moves.
- The frontend and API are released as one tagged set; mixing versions across them is unsupported.
- For the release workflow itself (cut a release), see [Publish a release](../publish-a-release/).
