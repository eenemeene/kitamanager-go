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
