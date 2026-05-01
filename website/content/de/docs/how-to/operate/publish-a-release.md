---
title: Release veröffentlichen
weight: 7
---

Sie wollen ein neues KitaManager-Release schneiden, das Container-Images baut und nach GHCR pusht.

## Schritte

1. Sicherstellen, dass `main` grün ist.
2. Nächsten Semver-Tag wählen (z. B. `v0.35.0`).
3. GitHub-Release mit auto-generierten Notes anlegen:
   ```bash
   gh release create v0.35.0 --generate-notes
   ```
4. Der Release-Workflow übernimmt: er baut Multi-Arch-Container-Images und pusht sie nach GHCR (und Docker Hub):
   - `ghcr.io/eenemeene/kitamanager:v0.35.0` — die API
   - `ghcr.io/eenemeene/kitamanager-ui:v0.35.0` — das Next.js-Frontend
   - `ghcr.io/eenemeene/kitamanager-report:v0.35.0` — das report-pdf-Sidecar

## Hinweise

- **Nutzen Sie `gh release create`, niemals einen nackten `git tag` + `git push --tags`.** Ein nackter Tag erzeugt kein GitHub-Release, und der Container-Build feuert nicht. Es gibt keinen Fallback — das Release ist weg.
- Der Release-Workflow ist pfad-unabhängig: jedes Release baut jedes Artefakt, auch wenn kein relevanter Code geändert wurde. Das ist Absicht — Release-Tags sollten ein kohärenter Schnappschuss sein.
- Für die Begründung und den vollständigen Workflow siehe `.github/workflows/release.yml`.
- Container-Images sind das **einzige** Release-Artefakt. Es werden keine eigenständigen Binaries veröffentlicht.
- Aktualisieren Sie Konsumenten (z. B. Ihr `docker-compose.yml`), um den neuen Tag explizit anzupinnen. Verfolgen Sie nicht `:latest` in Produktion.
