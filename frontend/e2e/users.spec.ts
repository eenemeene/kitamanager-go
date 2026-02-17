import { test, expect } from '@playwright/test';
import {
  login,
  getFirstOrganization,
  createUserViaApi,
  deleteUserViaApi,
  getUsersViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Users', () => {
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    await page.goto(`/organizations/${orgId}/users`);
    await page.waitForLoadState('networkidle');
  });

  test('should display users list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /user/i }).first()).toBeVisible();
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
  });

  test('should create a new user via UI', async ({ page }) => {
    const userName = uniqueName('TestUser');
    const userEmail = `testuser-${Date.now()}@example.com`;

    // Click "New User" button
    await page.getByRole('button', { name: /new user/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Fill form fields
    await page.getByLabel(/name/i).fill(userName);
    await page.getByLabel(/email/i).fill(userEmail);
    await page.getByLabel(/password/i).fill('testpassword123');

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // User should appear in list
    await expect(page.getByText(userName)).toBeVisible({ timeout: 10000 });

    // Cleanup via API
    const users = await getUsersViaApi(page);
    const created = users.find((u) => u.email === userEmail);
    if (created) {
      await deleteUserViaApi(page, created.id);
    }
  });

  test('should edit a user via UI', async ({ page }) => {
    // Setup: create user via API
    const origName = uniqueName('EditUser');
    const email = `edituser-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: origName,
      email,
      password: 'testpassword123',
    });

    // Reload to see the user
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(origName)).toBeVisible({ timeout: 10000 });

    // Click edit button (first icon button) on the user's row
    // The users table renders icon-only buttons without aria-labels:
    // first button = edit (pencil), second button = delete (trash)
    const row = page.getByRole('row').filter({ hasText: origName });
    const actionButtons = row.locator('button');
    // The row has multiple buttons; the last two are edit and delete
    await actionButtons.nth(-2).click();

    // Dialog should open
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Modify name
    const updatedName = uniqueName('Updated');
    await page.getByLabel(/name/i).clear();
    await page.getByLabel(/name/i).fill(updatedName);

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Updated name should appear
    await expect(page.getByText(updatedName)).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteUserViaApi(page, user.id);
  });

  test('should delete a user via UI', async ({ page }) => {
    // Setup: create user via API
    const userName = uniqueName('DelUser');
    const email = `deluser-${Date.now()}@example.com`;
    await createUserViaApi(page, {
      name: userName,
      email,
      password: 'testpassword123',
    });

    // Reload to see the user
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(userName)).toBeVisible({ timeout: 10000 });

    // Click delete button (last icon button) on the user's row
    const row = page.getByRole('row').filter({ hasText: userName });
    const actionButtons = row.locator('button');
    await actionButtons.last().click();

    // Confirm deletion in alert dialog
    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    // User should disappear
    await expect(page.getByText(userName)).not.toBeVisible({ timeout: 10000 });
  });
});
