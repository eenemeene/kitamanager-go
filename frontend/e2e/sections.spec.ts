import { test, expect } from '@playwright/test';
import {
  login,
  getApiToken,
  getFirstOrganization,
  createSectionViaApi,
  deleteSectionViaApi,
  getSectionsViaApi,
  createChildViaApi,
  createChildContractViaApi,
  deleteChildViaApi,
  uniqueName,
  formatDateForApi,
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

  test('should display sections board', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Board tab should be active by default and show drag hint
    await expect(page.getByText(/drag children/i)).toBeVisible({ timeout: 10000 });
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
    const sectionName = uniqueName('BoardSec');
    const section = await createSectionViaApi(page, token, orgId, sectionName);

    const childFirstName = uniqueName('BoardChild');
    const child = await createChildViaApi(page, token, orgId, {
      first_name: childFirstName,
      last_name: 'Test',
      birthdate: '2020-01-15',
      gender: 'female',
    });

    // Create a contract with the section so the child appears on the board
    await createChildContractViaApi(page, token, orgId, child.id, {
      from: formatDateForApi('2020-02-01'),
      section_id: section.id,
    });

    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Should see section column and child
    await expect(page.getByText(sectionName)).toBeVisible({ timeout: 10000 });
    // Reload once if child not visible (concurrent tests can shift pagination boundaries)
    if (!(await page.getByText(childFirstName).isVisible({ timeout: 5000 }).catch(() => false))) {
      await page.reload();
      await page.waitForLoadState('networkidle');
    }
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteChildViaApi(page, token, orgId, child.id);
    await deleteSectionViaApi(page, token, orgId, section.id);
  });

});
