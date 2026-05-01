---
title: CLI and configuration
weight: 2
---

Reference for everything you can pass to KitaManager from the outside: environment variables, command-line flags, Make targets.

{{< cards >}}
  {{< card link="env-vars/" title="Environment variables" subtitle="Every KITAMANAGER_*, DB_*, JWT_*, TOTP_*, WEBAUTHN_*, CORS_*, RATE_LIMIT_*, SEED_*. Required vs. optional, defaults, format." icon="cog" >}}
  {{< card link="kitamanager-api-flags/" title="kitamanager-api flags" subtitle="Command-line flags for the API binary. Most behaviour is env-driven; this is the short list of flags." icon="terminal" >}}
  {{< card link="make-targets/" title="Make targets" subtitle="Every Makefile target: build, test (unit / integration / contract / fuzz / coverage / race), web, docs, docker, hooks." icon="play" >}}
{{< /cards >}}
