import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
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
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'Sections');
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
    // On tablet (768px), the desktop sidebar is expanded and overlaps main content.
    // Collapse it so tab/button clicks are not intercepted.
    const toggleBtn = page.getByRole('button', { name: /toggle sidebar/i });
    if (await toggleBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await toggleBtn.click();
    }
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
    await page.getByRole('tab', { name: /manage/i }).click({ force: true });

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
    const sections = await getSectionsViaApi(page, orgId);
    const created = sections.find((s) => s.name === sectionName);
    if (created) {
      await deleteSectionViaApi(page, orgId, created.id);
    }
  });

  test('should delete a section from Manage tab', async ({ page }) => {
    // Create section via API
    const sectionName = uniqueName('DeleteMe');
    const section = await createSectionViaApi(page, orgId, sectionName);

    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Switch to Manage tab
    await page.getByRole('tab', { name: /manage/i }).click({ force: true });

    // Wait for section to appear
    await expect(page.getByText(sectionName)).toBeVisible({ timeout: 10000 });

    // Find the row with the section and click delete button
    const row = page.getByRole('row').filter({ hasText: sectionName });
    await row.getByRole('button', { name: /delete/i }).click();

    // Confirm deletion in dialog
    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    // Section should disappear
    await expect(page.getByText(sectionName)).not.toBeVisible({ timeout: 10000 });
  });

  test('should edit a section from Manage tab', async ({ page }) => {
    // Create section via API
    const origName = uniqueName('EditSec');
    const section = await createSectionViaApi(page, orgId, origName);

    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Switch to Manage tab
    await page.getByRole('tab', { name: /manage/i }).click({ force: true });

    // Wait for section to appear
    await expect(page.getByText(origName)).toBeVisible({ timeout: 10000 });

    // Click edit button on the section's row
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
    await deleteSectionViaApi(page, orgId, section.id);
  });

  test('should show children grouped by section on board', async ({ page }) => {
    // Create section and child via API
    const sectionName = uniqueName('BoardSec');
    const section = await createSectionViaApi(page, orgId, sectionName);

    const childFirstName = uniqueName('BoardChild');
    const child = await createChildViaApi(page, orgId, {
      first_name: childFirstName,
      last_name: 'Test',
      birthdate: '2020-01-15',
      gender: 'female',
    });

    // Create a contract with the section so the child appears on the board
    await createChildContractViaApi(page, orgId, child.id, {
      from: formatDateForApi('2020-02-01'),
      section_id: section.id,
    });

    await page.goto(`/organizations/${orgId}/sections`);
    await page.waitForLoadState('networkidle');

    // Should see section column and child
    await expect(page.getByText(sectionName)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteChildViaApi(page, orgId, child.id);
    await deleteSectionViaApi(page, orgId, section.id);
  });
});
