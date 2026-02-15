import { test, expect } from '@playwright/test';
import {
  login,
  getApiToken,
  createOrganizationViaApi,
  deleteOrganizationViaApi,
  uniqueName,
} from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

test.describe('Organizations', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/organizations');
    await expect(page).toHaveURL(/.*organization/);
  });

  test('should display organizations list', async ({ page }) => {
    // Use first() since there may be multiple headings with "organization"
    await expect(page.getByRole('heading', { name: /organization/i }).first()).toBeVisible();

    // Should have a table or list
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
  });

  test('should have new organization button', async ({ page }) => {
    const newButton = page.getByRole('button', { name: /new organization/i });
    await expect(newButton).toBeVisible();
  });

  test('should open create dialog', async ({ page }) => {
    await page.getByRole('button', { name: /new organization/i }).click();

    // Dialog should be visible
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Should have name input (use exact match to avoid "Default Section Name" field)
    await expect(page.getByLabel('Name', { exact: true })).toBeVisible();
  });

  test('should create a new organization', async ({ page }) => {
    const orgName = uniqueName('Test Org');

    // Open dialog
    await page.getByRole('button', { name: /new organization/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Fill form (use exact match to avoid "Default Section Name" field)
    await page.getByLabel('Name', { exact: true }).fill(orgName);
    await page.getByLabel(/Default Section Name/i).fill('Default');

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Organization should appear in list
    await expect(page.getByText(orgName)).toBeVisible({ timeout: 10000 });

    // Cleanup: delete the org via API
    const token = await getApiToken(page);
    const orgs = await page.evaluate(async ({ token }) => {
      const res = await fetch('/api/v1/organizations?limit=100', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      return data.data || [];
    }, { token });

    const createdOrg = orgs.find((o: { name: string }) => o.name === orgName);
    if (createdOrg) {
      await deleteOrganizationViaApi(page, token, createdOrg.id);
    }
  });

  test('should show organization in list after API creation', async ({ page }) => {
    const orgName = uniqueName('API Test Org');

    // Create via API
    const token = await getApiToken(page);
    const org = await createOrganizationViaApi(page, token, orgName);

    // Refresh page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Should appear in list
    await expect(page.getByText(orgName)).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteOrganizationViaApi(page, token, org.id);
  });

  test('should show table headers', async ({ page }) => {
    await expect(page.getByRole('columnheader', { name: /id/i })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('columnheader', { name: /name/i })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: /state/i })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: /status/i })).toBeVisible();
  });
});
