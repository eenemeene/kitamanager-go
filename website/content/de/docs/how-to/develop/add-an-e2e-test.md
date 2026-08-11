---
title: End-to-end-Test hinzufügen
weight: 4
---

Sie wollen einen Playwright-E2E-Test hinzufügen, der einen User-Flow gegen den laufenden Stack ausführt.

## Voraussetzungen

- Stack lokal laufend: `make dev`.
- Playwright-Browser installiert: `make web-playwright-install`.

## Schritte

### 1. Spec-Datei hinzufügen

`frontend/e2e/<feature>.spec.ts` nach dem bestehenden Muster anlegen:

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

### 2. Ausführen

```bash
cd frontend
npx playwright test --project=chromium e2e/<feature>.spec.ts
```

### 3. Pflicht-Konventionen

- **`test.use({ locale: 'en-US' })`** am Anfang jeder Datei. Die Codebasis nutzt englisches Locale für stabiles Text-Matching unabhängig vom Locale des Entwickler-Systems.
- **Niemals `waitForLoadState('networkidle')`.** react-query hält Hintergrund-Requests am Leben; networkidle löst nie auf, und der Test läuft nach 30 s in einen Timeout. `'load'` plus explizite Element-Assertions nutzen.
- **Niemals `waitForTimeout(...)`.** Durch `expect(...).toHaveText(...)` oder andere explizite Waits ersetzen. Bestehende TOTP-Ausnahmen sind inline dokumentiert.
- **Keine datums-abhängigen Assertions** — kein Assert auf „Active“, „Upcoming“, „Ended“-Status, der von heute abhängt. Feste vergangene Daten für Testdaten nutzen und den Datensatz statt des berechneten Status prüfen.

Die vollständigen Konventionen liegen in `.claude/rules/e2e-tests.md`.

## CI

Die E2E-Suite läuft auf jedem PR, gesharded über zwei GitHub-Actions-Runner (`--shard=1/2` und `--shard=2/2`). Für lokales Debuggen:

```bash
npx playwright test --project=chromium --headed e2e/<feature>.spec.ts
```

## Hinweise

- Für Komponententests stattdessen Jest nutzen — Playwright ist für Full-Stack-Flows.
- Die Test-Helpers in `frontend/e2e/utils/` decken Login, MFA-Enrollment und gängige Selektoren ab.
- Ein einzelnes `playwright.config.ts` liegt in `frontend/playwright.config.ts`; es pinnt einen Worker auf CI, um Konflikte mit dem geseedeten Backend zu vermeiden.
