---
title: kitamanager-api flags
weight: 2
---

The `kitamanager-api` binary takes very few command-line flags — almost everything is configured via [environment variables](../env-vars/), which is the right shape for container deployments.

| Flag | Default | Notes |
|---|---|---|
| `-version` | — | Print the build version (git tag, commit, build time) and exit. |
| `-help` | — | Print usage. |

The binary reads `.env` from its working directory at startup if present, then process environment overrides it. To see the build's version and commit:

```bash
./bin/kitamanager-api -version
```

For the report-pdf tool's flag set (which is much richer), see the [tool's README](https://github.com/eenemeene/kitamanager-go/tree/main/tools/report-pdf).
