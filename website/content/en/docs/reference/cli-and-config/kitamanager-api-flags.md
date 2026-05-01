---
title: kitamanager-api flags
weight: 2
---

The `kitamanager-api` binary takes **no command-line flags**. Every aspect of behaviour is configured through [environment variables](../env-vars/), which is the right shape for container deployments.

```bash
./bin/kitamanager-api
# All configuration via env or a .env file in the working directory.
```

The build version is reported by the API itself:

```bash
curl -sf http://localhost:8080/api/v1/health
# returns {"status":"ok","version":"v0.34.0","commit":"...","build_time":"..."}
```

For the report-pdf tool's flag set (which is much richer), see the [tool's README](https://github.com/eenemeene/kitamanager-go/tree/main/tools/report-pdf).
