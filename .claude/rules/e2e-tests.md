---
paths:
  - "frontend/e2e/**/*.ts"
---

# E2E test conventions (Playwright)

## Page load waits — never `networkidle`

react-query keeps background requests alive, so `waitForLoadState('networkidle')` never resolves. This causes consistent 30-second CI timeouts. Always `'load'`:

```typescript
// CORRECT
await page.waitForLoadState('load');
await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible({ timeout: 10000 });

// WRONG — will timeout because react-query keeps fetching
await page.waitForLoadState('networkidle');
```

For dynamic content, assert on elements with explicit timeouts.

## Locale: always English

```typescript
// At the top of each test file
test.use({ locale: 'en-US' });
```

Use English text in assertions and test data ("Deputy Manager" not "Gruppenleitung") so tests pass regardless of the developer's system locale.

## Avoid date-dependent assertions

Don't test status values that depend on "today's date" (`Active`, `Upcoming`, `Ended`) — they go flaky as dates pass.

```typescript
// BAD — will fail when 2024-06-01 passes
await page.getByLabel(/Start Date/i).fill('2024-06-01');
await expect(page.getByText('Upcoming')).toBeVisible();

// GOOD — test the data, not the status
await page.getByLabel(/Start Date/i).fill('2024-01-01');
await expect(page.getByText(/fulltime/i)).toBeVisible();
```

Use fixed past dates (e.g. `2024-01-01`) when creating test data; assert the data, not the computed status. If status must be tested, mock the date or use a date range that won't expire.

## No `waitForTimeout`

Replace with explicit waits — `expect(...).toHaveText(...)`, `expect(...).toBeVisible()`, etc.

Existing exceptions are documented inline at the call site. The 31s `waitForTimeout` calls in `two-factor-auth.spec.ts` (TOTP step boundary for replay-prevention) are tracked in the auto-memory file `project_e2e_totp_sleep_debt.md`; the planned fix is a configurable test-mode TOTP period or a test-only reset endpoint, both gated on `SEED_TEST_DATA`.

## Visual regression baselines

A **baseline** is the committed PNG in `e2e/visual-regression.spec.ts-snapshots/`
that a page is expected to look like. The test screenshots the page and compares
it pixel-for-pixel; a mismatch fails CI and writes `expected` / `actual` / `diff`
images into the report. It catches what assertions cannot describe — a shifted
button, a table overflowing at tablet width, a font that stopped loading.

### Regenerate with `make web-visual-baselines`. Never by hand.

```bash
make web-visual-baselines        # builds prod, verifies parity, rewrites
make web-visual-baselines-stop   # if a run was interrupted
```

A baseline is only worth as much as the environment that produced it, and the
naive way to make one is wrong in a way nothing catches afterwards:

- **Never regenerate against `make dev`.** CI serves a production build;
  `next dev` paints a dev-mode issues badge that lands inside the 412px mobile
  viewport, so `--update-snapshots` commits the badge as the page's expected
  appearance. Desktop captures usually escape it, which is worse — the mistake
  ships looking fine.
- The target builds production on its own port and dist directory, so a running
  `make dev` is untouched, and against an API on its own port because the
  running one allows only `:3000` as an origin.
- It **re-runs the existing baselines first**. If the machine renders unlike the
  runner, that fails there rather than on a PR after the new PNGs are committed.

Review the diff before committing: `git status --short` on the snapshots
directory should list only the pages you meant to change.

### A baseline that expires on its own is worse than no baseline

Anything derived from *today* drifts without anyone touching the page, and reds
CI on an unrelated PR. Mask it at source with `data-visual-mask="<category>"`
rather than widening `maxDiffPixelRatio`, which hides real regressions too.
Both of these were found by opening the committed PNGs, not by the suite passing:

- the week stepper's label (`Mon 17.08 – Fri 21.08 2026`) changes every Monday
- the dashboard stat cards show a staffing requirement computed from the
  children's ages, so it moves whenever one crosses an age bracket

Masked regions render as pink blocks in the committed PNG. **Open the image
after regenerating** — a green suite today says nothing about a baseline that
expires next week.
