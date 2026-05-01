---
title: kitamanager-api flags
weight: 2
---

The `kitamanager-api` binary takes **no command-line flags**. Configure via [environment variables](../env-vars/).

Build version: `GET /api/v1/health` returns `{"status":"ok","version":"...","commit":"...","build_time":"..."}`.

For the report-pdf sidecar's flag set, see the [tool's README](https://github.com/eenemeene/kitamanager-go/tree/main/tools/report-pdf) (its flags also read from `KITAMANAGER_REPORT_*` env vars).
