---
title: kitamanager-api-Schalter
weight: 2
---

Die `kitamanager-api`-Binärdatei nimmt **keine Kommandozeilen-Schalter**. Konfiguration über [Umgebungsvariablen](../env-vars/).

Build-Version: `GET /api/v1/health` liefert `{"status":"ok","version":"...","commit":"...","build_time":"..."}`.

Für den Schalter-Satz des report-pdf-Sidecars siehe das [README des Tools](https://github.com/eenemeene/kitamanager-go/tree/main/tools/report-pdf) (seine Schalter lesen auch aus `KITAMANAGER_REPORT_*`-Env-Vars).
