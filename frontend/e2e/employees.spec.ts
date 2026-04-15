import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createPayPlanViaApi,
  createPayPlanPeriodViaApi,
  createEmployeeWithContractViaApi,
  deleteEmployeeViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Employees', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'Employees');
    orgId = testOrg.orgId;
    // Create a pay plan with period (needed for employee contracts)
    const payplan = await createPayPlanViaApi(page, orgId, 'Test Pay Plan');
    await createPayPlanPeriodViaApi(page, orgId, payplan.id, {
      from: '2020-01-01',
      weekly_hours: 39,
    });
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
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('load');
  });

  test('should display employees list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /employee/i }).first()).toBeVisible();
  });

  test('should create a new employee via UI', async ({ page }) => {
    const firstName = uniqueName('EmpFirst');
    const lastName = uniqueName('EmpLast');

    // Click "New Employee" button
    await page.getByRole('button', { name: /new employee/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Fill form fields
    await page.getByLabel(/first name/i).fill(firstName);
    await page.getByLabel(/last name/i).fill(lastName);

    // Select gender via Shadcn Select
    await page.getByRole('combobox').click();
    await page.getByRole('option', { name: /female/i }).click();

    // Fill birthdate
    await page.getByLabel(/birthdate/i).fill('1990-05-15');

    // Capture the API response when submitting
    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/employees') && resp.request().method() === 'POST'
    );

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Verify API returned 201 Created
    const response = await responsePromise;
    expect(response.status()).toBe(201);
    const body = await response.json();

    // Dialog should close (confirms successful creation)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Cleanup via API
    await deleteEmployeeViaApi(page, orgId, body.id);
  });

  test('should edit an employee via UI', async ({ page }) => {
    // Setup: create employee with active contract so it appears in list
    const origFirst = uniqueName('EditEmp');
    const emp = await createEmployeeWithContractViaApi(page, orgId, {
      first_name: origFirst,
      last_name: 'Original',
      gender: 'male',
      birthdate: '1985-03-20',
    });

    // Reload and search for the employee
    await page.reload();
    await page.waitForLoadState('load');
    await page.getByRole('textbox', { name: /search/i }).fill(origFirst);
    await expect(page.getByText(origFirst)).toBeVisible({ timeout: 10000 });

    // Click edit button on the employee's row
    const row = page.getByRole('row').filter({ hasText: origFirst });
    await row.getByRole('button', { name: /edit/i }).click();

    // Dialog should open
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    // Modify first name
    const updatedFirst = uniqueName('Updated');
    await page.getByLabel(/first name/i).clear();
    await page.getByLabel(/first name/i).fill(updatedFirst);

    // Submit
    await page.getByRole('button', { name: /save/i }).click();

    // Dialog should close
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    // Search for updated name
    await page.getByRole('textbox', { name: /search/i }).clear();
    await page.getByRole('textbox', { name: /search/i }).fill(updatedFirst);

    // Updated name should appear
    await expect(page.getByText(updatedFirst)).toBeVisible({ timeout: 10000 });

    // Cleanup
    await deleteEmployeeViaApi(page, orgId, emp.id);
  });

  test('should delete an employee via UI', async ({ page }) => {
    // Setup: create employee with active contract so it appears in list
    const firstName = uniqueName('DelEmp');
    await createEmployeeWithContractViaApi(page, orgId, {
      first_name: firstName,
      last_name: 'ToDelete',
      gender: 'female',
      birthdate: '1992-07-10',
    });

    // Reload and search for the employee
    await page.reload();
    await page.waitForLoadState('load');
    await page.getByRole('textbox', { name: /search/i }).fill(firstName);
    await expect(page.getByText(firstName)).toBeVisible({ timeout: 10000 });

    // Click delete button on the employee's row
    const row = page.getByRole('row').filter({ hasText: firstName });
    await row.getByRole('button', { name: /delete/i }).click();

    // Confirm deletion in alert dialog
    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    // Employee should disappear
    await expect(page.getByText(firstName)).not.toBeVisible({ timeout: 10000 });
  });
});
