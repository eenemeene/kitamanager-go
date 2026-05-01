---
title: CLI und Konfiguration
weight: 2
---

Referenz für alles, was Sie KitaManager von außen mitgeben können: Umgebungsvariablen, Kommandozeilen-Schalter, Make-Ziele.

{{< cards >}}
  {{< card link="env-vars/" title="Umgebungsvariablen" subtitle="Jede KITAMANAGER_*, DB_*, JWT_*, TOTP_*, WEBAUTHN_*, CORS_*, RATE_LIMIT_*, SEED_*. Pflicht vs. optional, Defaults, Format." icon="cog" >}}
  {{< card link="kitamanager-api-flags/" title="kitamanager-api-Schalter" subtitle="Kommandozeilen-Schalter für die API-Binärdatei. Das meiste ist über Env-Vars konfigurierbar; dies ist die kurze Schalterliste." icon="terminal" >}}
  {{< card link="make-targets/" title="Make-Ziele" subtitle="Jedes Makefile-Ziel: build, tests (unit / integration / contract / fuzz / coverage / race), web, docs, docker, hooks." icon="play" >}}
{{< /cards >}}
