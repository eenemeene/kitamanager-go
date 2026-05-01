# KitaManager Go — Development Guidelines

KitaManager is a Kita management application: Go backend (Gin + GORM + Postgres) and a Next.js 16 frontend. Targets German Kita operators, with first-class support for the Berlin ISBJ funding model.

Topic-specific conventions live under `.claude/rules/` and load only when working with matching files. Everything in this file applies to **all** changes.

## Foundational principle

**Prefer correctness over speed when planning. Always deeply analyze and root-cause — never guess.**

If a behaviour, error, or test failure is not understood, stop and investigate until it is. Do not pattern-match a fix from surface symptoms. Do not paper over an unexpected state with `--no-verify`, `git reset --hard`, deleted lock files, or "just retry until it works". Read the code, follow the data, reproduce the failure, then fix the root cause. Document non-obvious findings in the commit so the next person doesn't have to re-derive them.

## Cross-cutting rules

### Use the latest versions of all dependencies
Hugo, Hextra, Go, Node, libraries — never pin to older versions or downgrade to work around compatibility. Upgrade the dependency chain to make everything work with the latest releases. **Exception:** when an upstream lag forces it (e.g. eslint-plugin-react vs. eslint 10), pin to the latest *compatible* version and document why in the commit.

### Money is stored as integer cents
All monetary values are `int` cents (or the smallest currency unit). Never `float64`. €1.50 → `150`. Convert at the boundary: `int(math.Round(eur * 100))` going in, `cents/100` for display. The frontend formats with `(cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' })`.

This avoids floating-point errors like `0.1 + 0.2 = 0.30000000000000004`.

### Use proper date/time types — never strings or regex
- Go: `time.Time` and the `time` package. No string compare, no regex on dates.
- TypeScript: `Date` or `date-fns`. No string slicing.

### `models.Today()` is the only source of truth for "today's calendar date"
Every "what date is it today?" decision MUST go through `models.Today()` (timezone defaults to `Europe/Berlin`, override via `KITAMANAGER_TIMEZONE`). Never `time.Now().UTC().Truncate(24*time.Hour)` — that's the *server's* clock day, off by one in the late-evening window for a Berlin user.

`time.Now()` remains correct for *instant* timestamps (audit log times, JWT issued-at, MFA expiry, attendance check-in/check-out times).

In tests, pin "today" with `models.SetNow(instant)` and `defer` the returned restore function. For tests that need a specific zone, call `models.DateIn(t, loc)` directly.

This rule applies to: amend-mode contract threshold, list defaults (`active_on=today`), auto-derived attendance dates, future-birthdate guards, and anywhere "today" is implicit.

### Never commit real personal data — DSGVO / GDPR
No real names, birthdates, addresses, voucher numbers, emails, financial amounts of real Kitas/children/employees/families. Applies to screenshots, fixtures, log dumps, configs, commit messages, PR descriptions. **Children's personal data is the highest legal protection level in Germany.**

Use the built-in seed data ("Kita Sonnenschein") for any examples — it contains only fictional names and generated records.

### Container images: `Dockerfile.*` is the only source of truth
- `Dockerfile.api` — multi-stage Chainguard `go` builder + `static` runtime
- `Dockerfile.frontend` — multi-stage Chainguard `node` builder + runtime
- `Dockerfile.report` — report-pdf binary

Container images are the only release artifact. The release workflow builds + pushes multi-arch images to GHCR when a GitHub release is published.

### Git workflow: feature branch + PR, never direct to `main`
```bash
git checkout -b feat/my-feature
# ... make changes ...
git push -u origin feat/my-feature
gh pr create --fill
```
Merge only after CI is green.

### Releases: `gh release create`, never a bare `git tag`
A bare tag does not create the GitHub release and the container image build won't fire.
```bash
gh release create v1.2.3 --generate-notes
```

## Topic-specific rules

Detailed conventions live under `.claude/rules/` and load only when Claude reads matching files:

| File | Loads when working on |
|---|---|
| `go-conventions.md` | any `**/*.go` — modern idioms (`any`, `slices`/`maps`/`cmp`, `for range N`, `log/slog`) |
| `api-handlers.md` | `internal/handlers/**/*.go` — swaggo annotations, DTO naming, REST conventions, audit logging |
| `database.md` | `internal/{database,models,store}/**/*.go` — migrations, schema diagram, soft-delete raw-query rule |
| `rbac.md` | `internal/{handlers,middleware,rbac}/**/*.go` — roles, organization scoping, authz middleware |
| `frontend-ui.md` | `frontend/src/**/*.{ts,tsx}` — responsive breakpoints, 44px touch targets, table patterns |
| `e2e-tests.md` | `frontend/e2e/**/*.ts` — Playwright (`waitForLoadState('load')`, `en-US` locale, no date-dependent assertions) |

## Other references

- `README.md` — repo overview, quickstart
- `DEVELOPMENT.md` — local-dev setup, common commands
- `website/content/{en,de}/docs/` — user-facing documentation; canonical reference for product terminology

### Berlin Kita terminology (used across the codebase)

- **Bezirks-Jugendamt** — issues Kita-Gutscheine, parent-facing voucher questions (12 districts in Berlin)
- **Senatsverwaltung für Bildung, Jugend und Familie** — sets the Kita funding rates, operates the ISBJ procedure on behalf of the districts
- **ISBJ** (Integriertes Software-System Berliner Jugendhilfe) — the software/procedure for the monthly bill exchange between Kita and Senate
- **NdH** — *nichtdeutsche Herkunftssprache* (family communication language is not German)
- **QM/MSS** — *Quartiersmanagement / Monitoring Soziale Stadtentwicklung* (Kita is in such a classified area)
- **Integrationsstatus A/B** — Berlin Kita classification under Eingliederungshilfe (SGB IX/SGB VIII)

When in doubt about wording, mirror what's in `configs/government-fundings/berlin.yaml` and the user-guide pages.
