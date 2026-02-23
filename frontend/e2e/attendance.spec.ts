import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createChildViaApi,
  createChildContractViaApi,
  createChildWithContractViaApi,
  createSectionViaApi,
  deleteChildViaApi,
  deleteSectionViaApi,
  getSectionsViaApi,
  clearWeekAttendance,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Attendance', () => {
  let orgId: number;
  let childId: number;
  const childFirstName = uniqueName('AttChild');

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'Attendance');
    orgId = testOrg.orgId;
    const child = await createChildWithContractViaApi(page, orgId, {
      first_name: childFirstName,
      last_name: 'Test',
      gender: 'female',
      birthdate: '2021-06-10',
    });
    childId = child.id;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteChildViaApi(page, orgId, childId).catch(() => {});
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/attendance`);
    await page.waitForLoadState('domcontentloaded');
  });

  test('should display attendance page with heading and week stepper', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /attendance/i }).first()).toBeVisible();
    // Week stepper should show "This week" button
    await expect(page.getByRole('button', { name: /this week/i })).toBeVisible();
  });

  test('should show child in weekly attendance table', async ({ page }) => {
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });

    // Table should have 6 column headers (Name + 5 weekdays)
    const headers = page.locator('thead th');
    await expect(headers).toHaveCount(6);
  });

  test('should check-in child', async ({ page }) => {
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Click the first check-in button in the row
    await row
      .getByRole('button', { name: /check-in/i })
      .first()
      .click();

    // After check-in, a check-out button should appear
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });
  });

  test('should check-out child after check-in', async ({ page }) => {
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Check in if not already
    const checkInButton = row.getByRole('button', { name: /check-in/i }).first();
    if (await checkInButton.isVisible().catch(() => false)) {
      await checkInButton.click();
      await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
        timeout: 10000,
      });
    }

    // Click check-out
    const checkOutButton = row.getByRole('button', { name: /check-out/i }).first();
    await checkOutButton.click();

    // After check-out, should show a time range (check-in and check-out times displayed)
    await expect(
      row.locator('button[aria-label="Check-out"]').filter({ hasText: /\d{2}:\d{2}/ })
    ).toBeVisible({ timeout: 10000 });
  });

  test('should navigate between weeks', async ({ page }) => {
    // Navigate to previous week
    await page.getByRole('button', { name: /previous week/i }).click();
    await page.waitForLoadState('domcontentloaded');

    // Navigate to next week
    await page.getByRole('button', { name: /next week/i }).click();
    await page.waitForLoadState('domcontentloaded');

    // Click "This week" to return
    await page.getByRole('button', { name: /this week/i }).click();
    await page.waitForLoadState('domcontentloaded');

    // Child should still be visible
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Attendance Status Transitions', () => {
  let orgId: number;
  let childId: number;
  const childFirstName = uniqueName('StatusChild');

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'AttStatus');
    orgId = testOrg.orgId;
    const child = await createChildWithContractViaApi(page, orgId, {
      first_name: childFirstName,
      last_name: 'Test',
      gender: 'male',
      birthdate: '2022-03-15',
    });
    childId = child.id;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteChildViaApi(page, orgId, childId).catch(() => {});
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    // Navigate first so cookies are fully available for API calls
    await page.goto(`/organizations/${orgId}/attendance`);
    await page.waitForLoadState('domcontentloaded');
    await clearWeekAttendance(page, orgId, childId);
    await page.reload();
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });
  });

  test('check-in shows time and check-out button', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Click check-in
    await row
      .getByRole('button', { name: /check-in/i })
      .first()
      .click();

    // Should show a check-out button and a time (HH:MM)
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });
    await expect(row.locator('text=/\\d{2}:\\d{2}/')).toBeVisible();
  });

  test('mark sick via popover shows Sick status text', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Open the popover (⋯ button)
    await row
      .getByRole('button', { name: /quick mark/i })
      .first()
      .click();

    // Click the Sick status button in the popover
    await page.getByRole('button', { name: /^Sick$/i }).click();

    // Should show "Sick" status text
    await expect(row.getByText('Sick')).toBeVisible({ timeout: 10000 });
  });

  test('check-in then mark sick clears times and shows Sick', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Check-in first
    await row
      .getByRole('button', { name: /check-in/i })
      .first()
      .click();
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // Open popover and mark sick
    await row
      .getByRole('button', { name: /quick mark/i })
      .first()
      .click();
    await page.getByRole('button', { name: /^Sick$/i }).click();

    // Should show "Sick" text, check-out button from today's cell should be gone
    await expect(row.getByText('Sick')).toBeVisible({ timeout: 10000 });
    // Time range should not be visible (times were cleared)
    await expect(row.locator('button[aria-label="Check-out"]').filter({ hasText: /\d{2}:\d{2}/ })).toBeHidden();
  });

  test('mark sick then mark present auto-sets check-in time', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Mark as sick via popover
    await row
      .getByRole('button', { name: /quick mark/i })
      .first()
      .click();
    await page.getByRole('button', { name: /^Sick$/i }).click();
    await expect(row.getByText('Sick')).toBeVisible({ timeout: 10000 });

    // Now mark as present via popover
    await row
      .getByRole('button', { name: /quick mark/i })
      .first()
      .click();
    await page.getByRole('button', { name: /^Present$/i }).click();

    // Should show check-out button (meaning check-in time was auto-set)
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });
    // Should NOT show "Present" as italic text (should show time instead)
    await expect(row.getByText('Present')).toBeHidden();
  });

  test('check-in, check-out, then mark vacation clears both times', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Check-in — wait for API response to complete
    const checkInResp = page.waitForResponse(
      (r) => r.url().includes('/attendance') && r.request().method() === 'POST'
    );
    await row
      .getByRole('button', { name: /check-in/i })
      .first()
      .click();
    await checkInResp;
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // Check-out — wait for API response to complete
    const checkOutResp = page.waitForResponse(
      (r) => r.url().includes('/attendance') && r.request().method() === 'PUT'
    );
    await row
      .getByRole('button', { name: /check-out/i })
      .first()
      .click();
    await checkOutResp;
    // Should show editable check-out time
    await expect(row.locator('button[aria-label="Check-out"]').filter({ hasText: /\d{2}:\d{2}/ })).toBeVisible({
      timeout: 10000,
    });

    // Mark vacation via popover
    await row
      .getByRole('button', { name: /quick mark/i })
      .first()
      .click();
    await page.getByRole('button', { name: /^Vacation$/i }).click();

    // Should show "Vacation" text, times should be gone
    await expect(row.getByText('Vacation')).toBeVisible({ timeout: 10000 });
    await expect(row.locator('button[aria-label="Check-out"]').filter({ hasText: /\d{2}:\d{2}/ })).toBeHidden();
  });

  test('mark absent then mark present shows check-in time', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Mark absent
    await row
      .getByRole('button', { name: /quick mark/i })
      .first()
      .click();
    await page.getByRole('button', { name: /^Absent$/i }).click();
    await expect(row.getByText('Absent')).toBeVisible({ timeout: 10000 });

    // Mark present
    await row
      .getByRole('button', { name: /quick mark/i })
      .first()
      .click();
    await page.getByRole('button', { name: /^Present$/i }).click();

    // Should show check-out button (check-in auto-set)
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });
  });
});

test.describe('Attendance Editable Times', () => {
  let orgId: number;
  let childId: number;
  const childFirstName = uniqueName('TimeChild');

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'AttTime');
    orgId = testOrg.orgId;
    const child = await createChildWithContractViaApi(page, orgId, {
      first_name: childFirstName,
      last_name: 'Test',
      gender: 'female',
      birthdate: '2022-01-20',
    });
    childId = child.id;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteChildViaApi(page, orgId, childId).catch(() => {});
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/attendance`);
    await page.waitForLoadState('domcontentloaded');
    await clearWeekAttendance(page, orgId, childId);
    await page.reload();
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });
  });

  test('click check-in time opens editable input', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Check-in first
    await row.getByRole('button', { name: /check-in/i }).first().click();
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // Click the check-in time text (it's a button with aria-label "Check-in")
    // The EditableTime renders as a <button> showing the time
    const timeButton = row.locator('button[aria-label="Check-in"]').filter({ hasText: /\d{2}:\d{2}/ });
    await timeButton.click();

    // Should show a time input
    const timeInput = row.locator('input[type="time"][aria-label="Check-in"]');
    await expect(timeInput).toBeVisible();
  });

  test('edit check-in time and save via Enter', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Check-in
    await row.getByRole('button', { name: /check-in/i }).first().click();
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // The time may render as a button (view mode) or input (edit mode).
    const timeButton = row
      .locator('button[aria-label="Check-in"]')
      .filter({ hasText: /\d{2}:\d{2}/ });
    const timeInput = row.locator('input[type="time"][aria-label="Check-in"]');
    await expect(timeButton.or(timeInput)).toBeVisible({ timeout: 10000 });
    if (await timeButton.isVisible()) {
      await timeButton.click();
    }

    // Change the time
    await expect(timeInput).toBeVisible();
    await timeInput.fill('08:30');
    await timeInput.press('Enter');

    // Should show "Attendance updated" toast (confirms save succeeded)
    await expect(page.getByText('Attendance updated', { exact: true })).toBeVisible({ timeout: 10000 });

    // Input should be gone, replaced by a time button again
    await expect(timeInput).toBeHidden({ timeout: 5000 });
    await expect(
      row.locator('button[aria-label="Check-in"]').filter({ hasText: /\d{2}:\d{2}/ })
    ).toBeVisible({ timeout: 10000 });
  });

  test('edit check-out time after full check-in/out', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Check-in: wait for the API response before proceeding
    const checkInResponse = page.waitForResponse(
      (resp) => resp.url().includes('/attendance') && resp.request().method() === 'POST'
    );
    await row.getByRole('button', { name: /check-in/i }).first().click();
    await checkInResponse;
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // Check-out: wait for the API response before asserting
    const checkOutResponse = page.waitForResponse(
      (resp) => resp.url().includes('/attendance') && resp.request().method() === 'PUT'
    );
    await row.getByRole('button', { name: /check-out/i }).first().click();
    await checkOutResponse;
    await expect(row.locator('button[aria-label="Check-out"]').filter({ hasText: /\d{2}:\d{2}/ })).toBeVisible({
      timeout: 10000,
    });

    // Click the check-out time to edit it
    const checkOutButton = row
      .locator('button[aria-label="Check-out"]')
      .filter({ hasText: /\d{2}:\d{2}/ });
    await checkOutButton.click();

    // Change the time
    const timeInput = row.locator('input[type="time"][aria-label="Check-out"]');
    await expect(timeInput).toBeVisible();
    await timeInput.fill('16:45');
    await timeInput.press('Enter');

    // Should show toast (confirms save succeeded)
    await expect(page.getByText('Attendance updated', { exact: true })).toBeVisible({ timeout: 10000 });

    // Input should be gone, replaced by a time button again
    await expect(timeInput).toBeHidden({ timeout: 5000 });
    await expect(
      row.locator('button[aria-label="Check-out"]').filter({ hasText: /\d{2}:\d{2}/ })
    ).toBeVisible({ timeout: 10000 });
  });

  test('escape cancels time edit without saving', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Check-in
    await row.getByRole('button', { name: /check-in/i }).first().click();
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // The time may render as a button (view mode) or input (edit mode).
    const timeButton = row.locator('button[aria-label="Check-in"]').filter({ hasText: /\d{2}:\d{2}/ });
    const timeInput = row.locator('input[type="time"][aria-label="Check-in"]');
    await expect(timeButton.or(timeInput)).toBeVisible({ timeout: 10000 });

    // Get the original time from whichever element is showing
    let originalTime: string;
    if (await timeButton.isVisible()) {
      originalTime = (await timeButton.textContent())!;
      await timeButton.click();
    } else {
      originalTime = await timeInput.inputValue();
    }

    // Change value, then press Escape
    await expect(timeInput).toBeVisible();
    await timeInput.fill('06:00');
    await timeInput.press('Escape');

    // Should revert to original time (no toast)
    await expect(row.locator('button[aria-label="Check-in"]').filter({ hasText: originalTime })).toBeVisible({
      timeout: 5000,
    });
  });
});

test.describe('Attendance Note Saving', () => {
  let orgId: number;
  let childId: number;
  const childFirstName = uniqueName('NoteChild');

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'AttNote');
    orgId = testOrg.orgId;
    const child = await createChildWithContractViaApi(page, orgId, {
      first_name: childFirstName,
      last_name: 'Test',
      gender: 'male',
      birthdate: '2022-05-10',
    });
    childId = child.id;
    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteChildViaApi(page, orgId, childId).catch(() => {});
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    // Navigate first so cookies are fully available for API calls
    await page.goto(`/organizations/${orgId}/attendance`);
    await page.waitForLoadState('domcontentloaded');
    await clearWeekAttendance(page, orgId, childId);
    // Reload to reflect cleared data
    await page.reload();
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText(childFirstName)).toBeVisible({ timeout: 10000 });
  });

  test('note textarea not shown when no attendance record exists', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Open popover without checking in first
    await row.getByRole('button', { name: /quick mark/i }).first().click();

    // Status buttons should be visible
    await expect(page.getByRole('button', { name: /^Present$/i })).toBeVisible();

    // Note textarea should NOT be visible (no attendance record yet)
    await expect(page.locator('textarea[placeholder="Note"]')).toBeHidden();
  });

  test('save note via popover after check-in', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Check-in first (note textarea only shows when attendance record exists)
    await row.getByRole('button', { name: /check-in/i }).first().click();
    await expect(row.getByRole('button', { name: /check-out/i }).first()).toBeVisible({
      timeout: 10000,
    });

    // Open popover
    await row.getByRole('button', { name: /quick mark/i }).first().click();

    // Type a note
    const noteTextarea = page.locator('textarea[placeholder="Note"]');
    await expect(noteTextarea).toBeVisible();
    await noteTextarea.fill('Arrived late, picked up by grandma');

    // Click save
    await page.getByRole('button', { name: /^Save$/i }).click();

    // Should show success toast
    await expect(page.getByText('Attendance updated', { exact: true })).toBeVisible({ timeout: 10000 });
  });

  test('note textarea appears after setting status via popover', async ({ page }) => {
    const row = page.getByRole('row').filter({ hasText: childFirstName });

    // Mark sick (creates a record without check-in)
    await row.getByRole('button', { name: /quick mark/i }).first().click();
    await page.getByRole('button', { name: /^Sick$/i }).click();
    await expect(row.getByText('Sick')).toBeVisible({ timeout: 10000 });

    // Reopen popover — note textarea should now be visible
    await row.getByRole('button', { name: /quick mark/i }).first().click();
    await expect(page.locator('textarea[placeholder="Note"]')).toBeVisible();

    // Save a note
    await page.locator('textarea[placeholder="Note"]').fill('Has fever');
    await page.getByRole('button', { name: /^Save$/i }).click();
    await expect(page.getByText('Attendance updated', { exact: true })).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Attendance Section Filter', () => {
  let orgId: number;
  let section1Id: number;
  let section2Id: number;
  let child1Id: number;
  let child2Id: number;
  const child1FirstName = uniqueName('SecChild1');
  const child2FirstName = uniqueName('SecChild2');
  const section2Name = uniqueName('Section2');

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'AttFilter');
    orgId = testOrg.orgId;
    section1Id = testOrg.sectionId; // default section from createTestOrg

    // Create a second section
    const section2 = await createSectionViaApi(page, orgId, section2Name);
    section2Id = section2.id;

    // Create child1 in section1 (default)
    const child1 = await createChildViaApi(page, orgId, {
      first_name: child1FirstName,
      last_name: 'Test',
      gender: 'female',
      birthdate: '2021-03-01',
    });
    child1Id = child1.id;
    await createChildContractViaApi(page, orgId, child1.id, {
      from: '2024-01-01T00:00:00Z',
      section_id: section1Id,
    });

    // Create child2 in section2
    const child2 = await createChildViaApi(page, orgId, {
      first_name: child2FirstName,
      last_name: 'Test',
      gender: 'male',
      birthdate: '2022-07-15',
    });
    child2Id = child2.id;
    await createChildContractViaApi(page, orgId, child2.id, {
      from: '2024-01-01T00:00:00Z',
      section_id: section2Id,
    });

    await page.close();
  });

  test.afterAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    await deleteChildViaApi(page, orgId, child1Id).catch(() => {});
    await deleteChildViaApi(page, orgId, child2Id).catch(() => {});
    await deleteSectionViaApi(page, orgId, section2Id).catch(() => {});
    await deleteTestOrg(page, orgId);
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(`/organizations/${orgId}/attendance`);
    await page.waitForLoadState('domcontentloaded');
  });

  test('shows all children when no section filter is selected', async ({ page }) => {
    await expect(page.getByText(child1FirstName)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(child2FirstName)).toBeVisible({ timeout: 10000 });
  });

  test('filter by section shows only children in that section', async ({ page }) => {
    await expect(page.getByText(child1FirstName)).toBeVisible({ timeout: 10000 });

    // Open section filter dropdown and select section2
    await page.getByRole('combobox').click();
    await page.getByRole('option', { name: section2Name }).click();

    // Only child2 should be visible
    await expect(page.getByText(child2FirstName)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(child1FirstName)).toBeHidden();
  });

  test('switching back to All Sections shows all children', async ({ page }) => {
    await expect(page.getByText(child1FirstName)).toBeVisible({ timeout: 10000 });

    // Filter by section2
    await page.getByRole('combobox').click();
    await page.getByRole('option', { name: section2Name }).click();
    await expect(page.getByText(child2FirstName)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(child1FirstName)).toBeHidden();

    // Switch back to "All Sections"
    await page.getByRole('combobox').click();
    await page.getByRole('option', { name: /all sections/i }).click();

    // Both children should be visible
    await expect(page.getByText(child1FirstName)).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(child2FirstName)).toBeVisible({ timeout: 10000 });
  });
});
