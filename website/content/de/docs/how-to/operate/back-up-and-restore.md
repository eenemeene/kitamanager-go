---
title: Datenbank sichern und wiederherstellen
weight: 2
---

Sie wollen einen Postgres-Dump der KitaManager-Datenbank machen oder einen wiederherstellen.

## Sichern

```bash
# Auf dem Host, der Postgres betreibt:
pg_dump --format=custom --file=kitamanager-$(date +%Y%m%d-%H%M%S).dump \
  --host=$DB_HOST --port=$DB_PORT --username=$DB_USER $DB_NAME
```

`--format=custom` produziert eine einzelne komprimierte Datei mit Unterstützung für parallele Wiederherstellung. Nutzen Sie `--format=plain`, wenn Sie einen SQL-Text-Dump zur Inspektion brauchen.

Für die Docker-Compose-Bereitstellung:

```bash
docker compose exec -T postgres pg_dump -U $DB_USER --format=custom $DB_NAME \
  > kitamanager-$(date +%Y%m%d-%H%M%S).dump
```

## Wiederherstellen

```bash
pg_restore --clean --if-exists --no-owner --no-privileges \
  --host=$DB_HOST --port=$DB_PORT --username=$DB_USER --dbname=$DB_NAME \
  kitamanager-YYYYMMDD-HHMMSS.dump
```

`--clean --if-exists` droppt bestehende Objekte vor dem Wiederherstellen; weglassen, wenn Sie in eine leere Datenbank wiederherstellen.

Für Docker Compose:

```bash
cat kitamanager-YYYYMMDD-HHMMSS.dump | \
  docker compose exec -T postgres pg_restore --clean --if-exists --no-owner -U $DB_USER -d $DB_NAME
```

## Wiederherstellung verifizieren

KitaManager hat einen Integrationstest, der ein Backup durch das Restore zurückspielt — `make api-test-backup`. Lassen Sie ihn gegen eine Staging-Kopie der wiederhergestellten Datenbank laufen, um Schema- und Datenintegrität zu bestätigen.

## Hinweise

- DSGVO/GDPR: Backups enthalten personenbezogene Daten von Kindern, Familien und Mitarbeitenden. Ruhend verschlüsseln, Zugriff einschränken, Aufbewahrungs-Plan festlegen.
- Backups per cron, systemd-timer oder Ihrem bevorzugten Scheduler einplanen. Täglich ist ein vernünftiger Startpunkt.
- Wiederherstellungen regelmäßig testen — ein Backup, das Sie noch nie wiederhergestellt haben, ist kein Backup.
- Das Sidecar `tools/report-pdf/` und die API selbst lesen aus derselben Datenbank; eine Wiederherstellung wirkt sich auf alles aus.
