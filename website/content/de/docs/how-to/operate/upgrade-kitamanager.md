---
title: KitaManager auf eine neue Version aktualisieren
weight: 11
---

Sie wollen von Version vX auf vY wechseln.

## Schritte

1. **Release-Notes lesen** unter https://github.com/eenemeene/kitamanager-go/releases/tag/vY: Breaking Changes, neue Env-Vars, Migrations-Hinweise.
2. **Backup machen** der aktuellen Datenbank. Siehe [Datenbank sichern und wiederherstellen](../back-up-and-restore/). Testen, dass der Dump in eine Scratch-Datenbank zurückspielt.
3. **Image-Tag in der produktiven `docker-compose.yml` aktualisieren.** Die in diesem Repo enthaltene Compose-Datei nutzt `build:` für die lokale Entwicklung — für Produktion hat man eine separate Compose mit `image:`-Zeilen, die auf bestimmte Tags gepinnt sind:
   ```bash
   sed -i 's|kitamanager:vX|kitamanager:vY|; s|kitamanager-ui:vX|kitamanager-ui:vY|; s|kitamanager-report:vX|kitamanager-report:vY|' docker-compose.yml
   ```
   Für die kanonischen Image-Namen siehe [Release veröffentlichen](../publish-a-release/).
4. **Neue Images ziehen**:
   ```bash
   docker compose pull
   ```
5. **Env-Var-Änderungen einarbeiten** (falls die Release-Notes welche hinzugefügt oder umbenannt haben).
6. **Neu starten**:
   ```bash
   docker compose up -d
   ```
   Migrationen laufen automatisch beim API-Start.
7. **Verifizieren**:
   ```bash
   curl -sf http://localhost:8080/api/v1/health
   ```
   Anmelden. Dashboard auf unerwartete Fehler prüfen. Audit-Log auf Migrations-Fehler prüfen.

## Wenn der neue Container nicht startet

Der Config-Loader bricht beim ersten Fehler ab und gibt alle Probleme in einer einzigen Zeile aus. Wenn `docker compose up -d` die API hochfährt, sie aber beendet, folgendes ausführen:

```bash
docker logs <api-container> 2>&1 | grep -A1 'configuration validation failed'
```

Alle erforderlichen oder fehlerhaften Env-Vars sind dort aufgeführt. Mit den Release-Notes (für Umbenennungen oder neue Variablen) und der [Umgebungsvariablen-Referenz](../../../reference/cli-and-config/env-vars/) abgleichen.

## Erwartete Ausfallzeit

Die API ist nicht verfügbar von „alter Container stoppt" bis „neuer Container hat Migration abgeschlossen und Health-Check besteht". Für die meisten Aktualisierungen sind das Sekunden. Eine Schema-Migration auf einer großen Tabelle (selten) kann Minuten dauern.

## Wenn eine Migration scheitert

Die API beendet mit Non-Zero, und Docker Compose markiert sie als unhealthy. Das Schema bleibt im teilweise migrierten Zustand. Aus dem Backup wiederherstellen, ein Issue mit der Nummer der scheiternden Migration eröffnen, auf den vorherigen Image-Tag zurückrollen, bis ein Fix vorliegt.

## Hinweise

- In Produktion an spezifische Tags pinnen (`kitamanager:v0.34.0`), niemals `:latest`. Der Release-Workflow garantiert, dass ein Tag sich nicht bewegt.
- Frontend und API werden als ein Tagged-Set veröffentlicht; Versionsmischung wird nicht unterstützt.
- Für den Release-Workflow selbst (ein Release schneiden) siehe [Release veröffentlichen](../publish-a-release/).
