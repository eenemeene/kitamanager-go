---
title: kitamanager-api-Schalter
weight: 2
---

Die `kitamanager-api`-Binärdatei nimmt **keine Kommandozeilen-Schalter**. Jeder Aspekt des Verhaltens wird über [Umgebungsvariablen](../env-vars/) konfiguriert — die richtige Form für Container-Bereitstellungen.

```bash
./bin/kitamanager-api
# Komplette Konfiguration über Env oder eine .env-Datei im Arbeitsverzeichnis.
```

Die Build-Version liefert die API selbst:

```bash
curl -sf http://localhost:8080/api/v1/health
# liefert {"status":"ok","version":"v0.34.0","commit":"...","build_time":"..."}
```

Für den deutlich reichhaltigeren Schalter-Satz des report-pdf-Tools siehe das [README des Tools](https://github.com/eenemeene/kitamanager-go/tree/main/tools/report-pdf).
