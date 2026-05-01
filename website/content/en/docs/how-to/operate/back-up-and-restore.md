---
title: Back up and restore the database
weight: 2
---

You want to take a Postgres dump of the KitaManager database, or restore one.

## Back up

```bash
# From the host running Postgres:
pg_dump --format=custom --file=kitamanager-$(date +%Y%m%d-%H%M%S).dump \
  --host=$DB_HOST --port=$DB_PORT --username=$DB_USER $DB_NAME
```

`--format=custom` produces a single file that's compressed and supports parallel restore. Use `--format=plain` if you need a SQL text dump for inspection.

For the Docker Compose deployment:

```bash
docker compose exec -T postgres pg_dump -U $DB_USER --format=custom $DB_NAME \
  > kitamanager-$(date +%Y%m%d-%H%M%S).dump
```

## Restore

```bash
pg_restore --clean --if-exists --no-owner --no-privileges \
  --host=$DB_HOST --port=$DB_PORT --username=$DB_USER --dbname=$DB_NAME \
  kitamanager-YYYYMMDD-HHMMSS.dump
```

`--clean --if-exists` drops existing objects before restoring; omit if you're restoring into an empty database.

For Docker Compose:

```bash
cat kitamanager-YYYYMMDD-HHMMSS.dump | \
  docker compose exec -T postgres pg_restore --clean --if-exists --no-owner -U $DB_USER -d $DB_NAME
```

## Verify the restore

KitaManager has an integration test that round-trips a backup through restore — `make api-test-backup`. Run it against a staging copy of the restored database to confirm schema + data integrity.

## Notes

- DSGVO/GDPR: backups contain personal data of children, families, and employees. Encrypt at rest, restrict access, set a retention schedule.
- Schedule backups via cron, systemd-timer, or your scheduler of choice. Daily is a reasonable starting point.
- Test restores periodically — a backup you've never restored is not a backup.
- The `tools/report-pdf/` sidecar and the API itself read from the same database, so restoring affects them all.
