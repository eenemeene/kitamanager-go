import { test, expect } from '@playwright/test';
import {
  login,
  getFirstOrganization,
  createGroupViaApi,
  deleteGroupViaApi,
  getGroupsViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Groups', () => {
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    const org = await getFirstOrganization(page);
    orgId = org.id;
    await page.goto(`/organizations/${orgId}/groups`);
    await page.waitForLoadState('networkidle');
  });

  test('should display groups list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /group/i }).first()).toBeVisible();
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
  });

  test('should create a new group via UI', async ({ page }) => {
    const groupName = uniqueName('TestGroup');

    // Click "New Group" button
    await page.getByRole('button', { name: /new group/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Fill form fields
    await page.getByLabel(/name/i).fill(groupName);

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Group should appear in list
    await expect(page.getByText(groupName)).toBeVisible({ timeout: 10000 });

    // Cleanup via API
    const groups = await getGroupsViaApi(page, orgId);
    const created = groups.find((g) => g.name === groupName);
    if (created) {
      await deleteGroupViaApi(page, orgId, created.id);
    }
  });

  test('should edit a group via UI', async ({ page }) => {
    // Setup: create group via API
    const origName = uniqueName('EditGroup');
    const group = await createGroupViaApi(page, orgId, { name: origName });

    // Reload to see the group
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(origName)).toBeVisible({ timeout: 10000 });

    // Click edit button on the group's row
    const row = page.getByRole('row').filter({ hasText: origName });
    await row.getByRole('button', { name: /edit/i }).click();

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
    await deleteGroupViaApi(page, orgId, group.id);
  });

  test('should delete a group via UI', async ({ page }) => {
    // Setup: create group via API
    const groupName = uniqueName('DelGroup');
    await createGroupViaApi(page, orgId, { name: groupName });

    // Reload to see the group
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(groupName)).toBeVisible({ timeout: 10000 });

    // Click delete button on the group's row
    const row = page.getByRole('row').filter({ hasText: groupName });
    await row.getByRole('button', { name: /delete/i }).click();

    // Confirm deletion in alert dialog
    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    // Group should disappear
    await expect(page.getByText(groupName)).not.toBeVisible({ timeout: 10000 });
  });
});
