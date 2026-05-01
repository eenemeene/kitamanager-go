---
title: Add an end-to-end test
weight: 4
---

You want to add a Playwright E2E test that exercises a user flow against the running stack.

## Prerequisites

- Stack running locally: `make dev`.
- Playwright browsers installed: `make web-playwright-install`.

## Steps

### 1. Add the spec file

Create `frontend/e2e/<feature>.spec.ts` following the existing pattern:

```typescript
import { test, expect } from '@playwright/test';
import { login } from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('My feature', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('does the thing', async ({ page }) => {
    await page.goto('/some/route');
    await page.waitForLoadState('load');

    await expect(
      page.getByRole('heading', { name: /expected title/i })
    ).toBeVisible({ timeout: 10000 });
  });
});
```

### 2. Run it

```bash
cd frontend
npx playwright test --project=chromium e2e/<feature>.spec.ts
```

### 3. The conventions you must follow

- **`test.use({ locale: 'en-US' })`** at the top of every file. The codebase uses English locale for stable text matching regardless of the developer's system locale.
- **Never `waitForLoadState('networkidle')`.** react-query keeps background requests alive; networkidle never resolves and the test eventually times out at 30s. Use `'load'` plus explicit element-level assertions.
- **Never `waitForTimeout(...)`.** Replace with `expect(...).toHaveText(...)` or other explicit waits. Existing TOTP exceptions are documented inline.
- **No date-dependent assertions** — don't assert on "Active", "Upcoming", "Ended" status text that depends on today. Use fixed past dates for test data and assert on the data, not the computed status.

The full conventions are in `.claude/rules/e2e-tests.md`.

## CI

The E2E suite runs on every PR, sharded across two GitHub Actions runners (`--shard=1/2` and `--shard=2/2`). For local debugging, run:

```bash
npx playwright test --project=chromium --headed e2e/<feature>.spec.ts
```

## Notes

- For component-level tests, use Jest instead — Playwright is for full-stack flows.
- The test helpers in `frontend/e2e/utils/` cover login, MFA enrolment, and common selectors.
- A single `playwright.config.ts` lives at `frontend/playwright.config.ts`; it pins one worker on CI to avoid contention with the seeded backend.
