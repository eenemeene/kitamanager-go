import { test, expect } from '@playwright/test';
import { login, getFirstOrganization } from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Dashboard', () => {
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    await page.goto(`/organizations/${orgId}/dashboard`);
    await page.waitForLoadState('load');
  });

  test('should display dashboard heading', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /dashboard/i }).first()).toBeVisible();
  });

  test('should display stat cards', async ({ page }) => {
    // Verify stat card titles are visible
    await expect(page.getByText(/active employees/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/active children/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/staffing coverage/i)).toBeVisible({ timeout: 10000 });
  });

  test('renders its widgets, not an error boundary', async ({ page }) => {
    // The error-boundary check here used to be written as
    // `expect(...).not.toBeVisible().catch(() => {})`. A failed expectation
    // returns a rejected promise and the catch discarded it, so the assertion
    // was a no-op even with the boundary on screen. Nothing else in the test
    // looked at a widget, despite the name.
    await expect(page.getByText(/active employees/i)).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-testid="error-boundary"]')).toHaveCount(0);
    await expect(page.getByText(/something went wrong/i)).toHaveCount(0);

    // Which widgets render depends on the data, so the assertion is that the
    // dashboard put *something* below the stat cards -- a card with a heading.
    // Zero of them means every widget query failed, which is the regression
    // this test exists for and the one it could not previously see.
    const headings = page.getByRole('heading');
    expect(await headings.count(), 'dashboard should render widget headings').toBeGreaterThan(3);
  });
});
