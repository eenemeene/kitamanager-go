import { test, expect } from '@playwright/test';
import { login, getApiToken, getFirstOrganization, uniqueName } from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

test.describe('Form Validation Errors', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should show validation error when creating organization with empty name', async ({
    page,
  }) => {
    await page.goto('/organizations');
    await page.waitForLoadState('networkidle');

    // Open create dialog
    await page.getByRole('button', { name: /new organization|neue organisation/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Try to submit empty form
    await page.getByRole('button', { name: /save|speichern/i }).click();

    // Should stay on dialog (not close) - form validation prevents submission
    await expect(page.getByRole('dialog')).toBeVisible();
  });

  test('should show validation error for invalid employee data', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/employees`);
    await page.waitForLoadState('networkidle');

    // Open create dialog
    await page.getByRole('button', { name: /new employee|neuer mitarbeiter/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Try to submit empty form
    await page.getByRole('button', { name: /save|speichern/i }).click();

    // Dialog should remain open because of validation errors
    await expect(page.getByRole('dialog')).toBeVisible();
  });

  test('should show validation error for invalid child data', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/children`);
    await page.waitForLoadState('networkidle');

    // Open create dialog
    await page.getByRole('button', { name: /new child|neues kind/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Try to submit empty form
    await page.getByRole('button', { name: /save|speichern/i }).click();

    // Dialog should remain open because of validation errors
    await expect(page.getByRole('dialog')).toBeVisible();
  });
});

test.describe('Authentication Error Scenarios', () => {
  test('should redirect to login when accessing protected page without auth', async ({ page }) => {
    // Access protected page directly without logging in
    await page.goto('/organizations');

    // Should redirect to login page
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
  });

  test('should redirect to login when accessing nested protected page without auth', async ({
    page,
  }) => {
    // Access deeply nested protected page
    await page.goto('/organizations/1/employees');

    // Should redirect to login page
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
  });

  test('should show error for invalid login credentials', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });

    // Fill form with invalid credentials
    await page.getByLabel(/email/i).fill('nonexistent@example.com');
    await page.getByLabel(/password/i).fill('wrongpassword123');
    await page.getByRole('button', { name: /sign in|login|anmelden/i }).click();

    // Should remain on login page
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });

    // Should show an error indication (toast, alert, or form stays)
    // The exact error display depends on implementation, but user should not be redirected
    await page.waitForTimeout(2000);
    await expect(page).toHaveURL(/.*login/);
  });

  test('should show error for empty login form submission', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });

    // Click login without filling fields
    await page.getByRole('button', { name: /sign in|login|anmelden/i }).click();

    // Should remain on login page
    await expect(page).toHaveURL(/.*login/);
  });
});

test.describe('Not Found Scenarios', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should handle non-existent organization gracefully', async ({ page }) => {
    // Navigate to a non-existent organization
    await page.goto('/organizations/99999/employees');
    await page.waitForLoadState('networkidle');

    // Page should either show an error/empty state or redirect
    // It should NOT show an unhandled error or blank page
    const bodyText = await page.locator('body').textContent();
    expect(bodyText).toBeTruthy();
    expect(bodyText!.length).toBeGreaterThan(0);

    // Should not show a raw error stack trace
    await expect(page.getByText(/TypeError|ReferenceError|Cannot read/i)).not.toBeVisible({
      timeout: 3000,
    });
  });

  test('should handle non-existent employee gracefully', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/employees/99999/contracts`);
    await page.waitForLoadState('networkidle');

    // Should not show a raw error stack trace
    await expect(page.getByText(/TypeError|ReferenceError|Cannot read/i)).not.toBeVisible({
      timeout: 3000,
    });
  });

  test('should handle non-existent child gracefully', async ({ page }) => {
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto(`/organizations/${org.id}/children/99999/contracts`);
    await page.waitForLoadState('networkidle');

    // Should not show a raw error stack trace
    await expect(page.getByText(/TypeError|ReferenceError|Cannot read/i)).not.toBeVisible({
      timeout: 3000,
    });
  });
});

test.describe('Duplicate Resource Errors', () => {
  test('should allow creating organization with same name (no unique constraint)', async ({
    page,
  }) => {
    await login(page);
    const token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);

    await page.goto('/organizations');
    await page.waitForLoadState('networkidle');

    // Create org with same name as existing one
    await page.getByRole('button', { name: /new organization|neue organisation/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.getByLabel('Name', { exact: true }).fill(org.name);
    await page.getByLabel(/Default Section Name/i).fill('Default');

    // Submit
    await page.getByRole('button', { name: /save|speichern/i }).click();

    // Dialog should close - duplicate names are allowed
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Cleanup: delete the duplicate org
    const orgs = await page.evaluate(
      async ({ token }) => {
        const res = await fetch('/api/v1/organizations?limit=100', {
          headers: { Authorization: `Bearer ${token}` },
        });
        const data = await res.json();
        return data.data || [];
      },
      { token }
    );
    // Delete the last org with the matching name (the one we just created)
    const duplicates = orgs.filter((o: { name: string }) => o.name === org.name);
    if (duplicates.length > 1) {
      const newest = duplicates[duplicates.length - 1];
      await page.evaluate(
        async ({ token, orgId }) => {
          const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
          const csrfToken = csrfMatch ? csrfMatch[1] : null;
          const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
          if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
          await fetch(`/api/v1/organizations/${orgId}`, { method: 'DELETE', headers });
        },
        { token, orgId: newest.id }
      );
    }
  });
});
