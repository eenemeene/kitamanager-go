import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createUserViaApi,
  deleteUserViaApi,
  getUsersViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Users', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'Users');
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

    await page.getByRole('button', { name: /new user/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.getByLabel(/name/i).fill(userName);
    await page.getByLabel(/email/i).fill(userEmail);
    await page.getByLabel(/password/i).fill('testpassword123');

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText(userName)).toBeVisible({ timeout: 10000 });

    const users = await getUsersViaApi(page);
    const created = users.find((u) => u.email === userEmail);
    if (created) {
      await deleteUserViaApi(page, created.id);
    }
  });

  test('should edit a user via UI', async ({ page }) => {
    const origName = uniqueName('EditUser');
    const email = `edituser-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: origName,
      email,
      password: 'testpassword123',
    });

    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(origName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: origName });
    // Action buttons are icon-only; skip the toggle switch (role=switch)
    // Order: edit (pencil), membership (users), delete (trash)
    await row.locator('button:not([role="switch"])').first().click();

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    const updatedName = uniqueName('Updated');
    await page.getByLabel(/name/i).clear();
    await page.getByLabel(/name/i).fill(updatedName);

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText(updatedName)).toBeVisible({ timeout: 10000 });

    await deleteUserViaApi(page, user.id);
  });

  test('should delete a user via UI', async ({ page }) => {
    const userName = uniqueName('DelUser');
    const email = `deluser-${Date.now()}@example.com`;
    await createUserViaApi(page, {
      name: userName,
      email,
      password: 'testpassword123',
    });

    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(userName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: userName });
    const actionButtons = row.locator('button');
    await actionButtons.last().click();

    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByText(userName)).not.toBeVisible({ timeout: 10000 });
  });
});
