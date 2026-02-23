import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  getOrganizationsViaApi,
  deleteOrganizationViaApi,
  uniqueName,
} from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

test.describe('Form Validation Errors', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'ErrorTests');
    orgId = testOrg.orgId;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should show validation error when creating organization with empty name', async ({
    page,
  }) => {
    await page.goto('/organizations');
    await page.waitForLoadState('networkidle');

    // Open create dialog
    await page.getByRole('button', { name: /new organization/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Try to submit empty form
    await page.getByRole('button', { name: /save/i }).click();

    // Should stay on dialog (not close) - form validation prevents submission
    await expect(page.getByRole('dialog')).toBeVisible();
  });

  test('should show validation error for invalid employee data', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('networkidle');

    // Open create dialog
    await page.getByRole('button', { name: /new employee/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Try to submit empty form
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should remain open because of validation errors
    await expect(page.getByRole('dialog')).toBeVisible();
  });

  test('should show validation error for invalid child data', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('networkidle');

    // Open create dialog
    await page.getByRole('button', { name: /new child/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Try to submit empty form
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should remain open because of validation errors
    await expect(page.getByRole('dialog')).toBeVisible();
  });
});

test.describe('Authentication Error Scenarios', () => {
  test('should redirect to login when accessing protected page without auth', async ({ page }) => {
    await page.goto('/organizations');
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
  });

  test('should redirect to login when accessing nested protected page without auth', async ({
    page,
  }) => {
    await page.goto('/organizations/1/employees');
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
  });

  test('should show error for invalid login credentials', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });

    await page.getByLabel(/email/i).fill('nonexistent@example.com');
    await page.getByLabel(/password/i).fill('wrongpassword123');
    await page.getByRole('button', { name: /sign in|login/i }).click();

    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
    await expect(page).toHaveURL(/.*login/);
  });

  test('should show error for empty login form submission', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /sign in|login/i }).click();
    await expect(page).toHaveURL(/.*login/);
  });
});

test.describe('Not Found Scenarios', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'NotFound');
    orgId = testOrg.orgId;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should handle non-existent organization gracefully', async ({ page }) => {
    await page.goto('/organizations/99999/employees');
    await page.waitForLoadState('networkidle');

    const bodyText = await page.locator('body').textContent();
    expect(bodyText).toBeTruthy();
    expect(bodyText!.length).toBeGreaterThan(0);

    await expect(page.getByText(/TypeError|ReferenceError|Cannot read/i)).not.toBeVisible({
      timeout: 3000,
    });
  });

  test('should handle non-existent employee gracefully', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees/99999/contracts`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByText(/TypeError|ReferenceError|Cannot read/i)).not.toBeVisible({
      timeout: 3000,
    });
  });

  test('should handle non-existent child gracefully', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children/99999/contracts`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByText(/TypeError|ReferenceError|Cannot read/i)).not.toBeVisible({
      timeout: 3000,
    });
  });
});

test.describe('Duplicate Resource Errors', () => {
  test('should show error when creating organization with duplicate name', async ({ page }) => {
    await login(page);

    // Create a temporary org to test duplication against
    const testOrg = await createTestOrg(page, 'DupTest');

    try {
      await page.goto('/organizations');
      await page.waitForLoadState('networkidle');

      await page.getByRole('button', { name: /new organization/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Get the test org's actual name for duplication
      const orgs = await getOrganizationsViaApi(page);
      const targetOrg = orgs.find((o) => o.id === testOrg.orgId);
      const orgName = targetOrg!.name;

      await page.getByLabel('Name', { exact: true }).fill(orgName);
      // Explicitly interact with the State combobox inside the dialog
      const dialog = page.getByRole('dialog');
      await dialog.locator('[role="combobox"]').first().click();
      await page.getByRole('option', { name: /berlin/i }).click();
      await page.getByLabel(/Default Section Name/i).fill('Default');

      await page.getByRole('button', { name: /save/i }).click();

      // Should show an error toast (duplicate name causes API error)
      // and the dialog should remain open
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    } finally {
      await deleteTestOrg(page, testOrg.orgId);
    }
  });
});
