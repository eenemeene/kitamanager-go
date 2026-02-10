import { test, expect } from '@playwright/test';
import {
  login,
  getApiToken,
  getFirstOrganization,
  createSectionViaApi,
  deleteSectionViaApi,
  getSectionsViaApi,
  createChildViaApi,
  deleteChildViaApi,
  updateChildSectionViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Sections', () => {
  let token: string;
  let orgId: number;

  test.beforeEach(async ({ page }) => {
    await login(page);
    token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);
    orgId = org.id;
  });

  test('should display sections board with Unassigned column', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Board tab should be active by default
    await expect(page.getByText(/unassigned/i)).toBeVisible({ timeout: 10000 });
  });

  test('should switch to Manage tab and create a section', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Switch to Manage tab
    await page.getByRole('tab', { name: /manage/i }).click();

    const sectionName = uniqueName('Section');

    // Click new section button
    await page.getByRole('button', { name: /new section/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Fill form
    await page.getByLabel(/name/i).fill(sectionName);
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Section should appear in list
    await expect(page.getByText(sectionName)).toBeVisible({ timeout: 10000 });

    // Cleanup
    const sections = await getSectionsViaApi(page, token, orgId);
    const created = sections.find((s) => s.name === sectionName);
    if (created) {
      await deleteSectionViaApi(page, token, orgId, created.id);
    }
  });

  test('should delete a section from Manage tab', async ({ page }) => {
    // Create section via API
    const sectionName = uniqueName('DeleteMe');
    const section = await createSectionViaApi(page, token, orgId, sectionName);

    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Switch to Manage tab
    await page.getByRole('tab', { name: /manage/i }).click();

    // Wait for section to appear
    await expect(page.getByText(sectionName)).toBeVisible({ timeout: 10000 });

    // Find the row with the section and click delete button
    const row = page.getByRole('row').filter({ hasText: sectionName });
    await row.getByRole('button').last().click();

    // Confirm deletion in dialog
    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    // Section should disappear
    await expect(page.getByText(sectionName)).not.toBeVisible({ timeout: 10000 });
  });

  test('should show children grouped by section on board', async ({ page }) => {
    // Create section and child via API
    const sectionName = uniqueName('BoardTest');
    const section = await createSectionViaApi(page, token, orgId, sectionName);

    const child = await createChildViaApi(page, token, orgId, {
      first_name: 'BoardTest',
      last_name: uniqueName('Child'),
      birthdate: '2020-01-15',
      gender: 'female',
    });

    // Assign child to section
    await updateChildSectionViaApi(page, token, orgId, child.id, section.id);

    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Should see section column and child
    await expect(page.getByText(sectionName)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('BoardTest')).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteChildViaApi(page, token, orgId, child.id);
    await deleteSectionViaApi(page, token, orgId, section.id);
  });

  test('should show Unassigned column with unassigned children', async ({ page }) => {
    // Create a child without section assignment
    const child = await createChildViaApi(page, token, orgId, {
      first_name: 'Unassigned',
      last_name: uniqueName('Child'),
      birthdate: '2021-05-10',
      gender: 'male',
    });

    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Unassigned column should contain the child
    await expect(page.getByText(/unassigned/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Unassigned')).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteChildViaApi(page, token, orgId, child.id);
  });
});
