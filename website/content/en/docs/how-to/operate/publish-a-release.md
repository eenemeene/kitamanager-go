---
title: Publish a release
weight: 7
---

You want to cut a new KitaManager release that builds and pushes container images to GHCR.

## Steps

1. Make sure `main` is green.
2. Pick the next semver tag (e.g. `v0.35.0`).
3. Create the GitHub release with auto-generated notes:
   ```bash
   gh release create v0.35.0 --generate-notes
   ```
4. The release workflow takes over: it builds multi-arch container images and pushes them to GHCR (and Docker Hub):
   - `ghcr.io/eenemeene/kitamanager:v0.35.0` — the API
   - `ghcr.io/eenemeene/kitamanager-ui:v0.35.0` — the Next.js frontend
   - `ghcr.io/eenemeene/kitamanager-report:v0.35.0` — the report-pdf sidecar

## Notes

- **Use `gh release create`, never a bare `git tag` + `git push --tags`.** A bare tag does not create the GitHub release and the container build won't fire. There's no fallback — the release is gone.
- The release workflow is path-independent: every release builds every artifact, even if no relevant code changed. This is intentional — release tags should be a coherent snapshot.
- For the rationale and the full workflow, see `.github/workflows/release.yml`.
- Container images are the **only** release artifact. There are no standalone binaries published.
- Update consumers (e.g. your `docker-compose.yml`) to pin the new tag explicitly. Don't track `:latest` in production.
