import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  createPayPlanViaApi,
  createPayPlanPeriodViaApi,
  createChildViaApi,
  createEmployeeViaApi,
  deleteChildViaApi,
  deleteEmployeeViaApi,
  createChildContractViaApi,
  createEmployeeContractViaApi,
  getSectionsViaApi,
  getPayPlansViaApi,
  ensureFundingHasProperties,
  uniqueName,
  formatDateForApi,
} from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

// Shared state across all describe blocks
let orgId: number;
let defaultSectionId: number;
let payplanId: number;

test.beforeAll(async ({ browser }) => {
  const page = await browser.newPage();
  await login(page);
  const testOrg = await createTestOrg(page, 'Contracts');
  orgId = testOrg.orgId;
  defaultSectionId = testOrg.sectionId;
  // Create a pay plan with period (needed for employee contracts)
  const payplan = await createPayPlanViaApi(page, orgId, 'Test Pay Plan');
  payplanId = payplan.id;
  await createPayPlanPeriodViaApi(page, orgId, payplanId, {
    from: '2020-01-01',
    weekly_hours: 39,
  });
  // Ensure Berlin funding has properties (for contract property suggestions)
  await ensureFundingHasProperties(page);
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

test.describe('Child Contracts - CRUD Operations', () => {
  test('should add a new contract from history page', async ({ page }) => {
    const childName = uniqueName('AddContract');
    const child = await createChildViaApi(page, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2020-06-15',
      gender: 'female',
    });

    try {
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/No contracts found/i).first()).toBeVisible({
        timeout: 10000,
      });

      await page.getByRole('button', { name: /New Contract/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      await page.getByLabel(/Start Date/i).fill('2024-01-01');

      await page.getByRole('combobox', { name: /Sections/i }).click();
      await page.getByRole('option').first().click();

      const suggestionsArea = page.getByTestId('property-suggestions');
      if (await suggestionsArea.isVisible({ timeout: 3000 }).catch(() => false)) {
        const gantzagSuggestion = suggestionsArea.locator('button', { hasText: 'ganztag' }).first();
        if (await gantzagSuggestion.isVisible({ timeout: 2000 }).catch(() => false)) {
          await gantzagSuggestion.click();
        }
      }

      await page.getByRole('button', { name: /Save/i }).click();
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      const contractRow = page.locator('tbody tr').first();
      await expect(contractRow).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('should show suggested properties from government funding', async ({ page }) => {
    await ensureFundingHasProperties(page);

    const childName = uniqueName('SuggestAttr');
    const child = await createChildViaApi(page, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2022-03-15',
      gender: 'male',
    });

    try {
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('button', { name: /New Contract/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      await page.getByLabel(/Start Date/i).fill('2025-03-01');

      await page.getByRole('combobox', { name: /Sections/i }).click();
      await page.getByRole('option').first().click();

      await expect(page.getByText(/Available:/i)).toBeVisible({ timeout: 15000 });

      const suggestionsArea = page.getByTestId('property-suggestions');
      const gantzagSuggestion = suggestionsArea.locator('button', { hasText: 'ganztag' }).first();
      await expect(gantzagSuggestion).toBeVisible({ timeout: 5000 });
      await gantzagSuggestion.click();

      const dialog = page.getByRole('dialog');
      await expect(dialog.getByText('ganztag').first()).toBeVisible();

      const ndhSuggestion = suggestionsArea.locator('button', { hasText: 'ndh' }).first();
      if (await ndhSuggestion.isVisible({ timeout: 2000 }).catch(() => false)) {
        await ndhSuggestion.click();
        await expect(dialog.getByText('ndh').first()).toBeVisible();
      }

      await page.getByRole('button', { name: /Save/i }).click();
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      await expect(page.getByText(/Full-Time/i)).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('should update a contract from history page', async ({ page }) => {
    const childName = uniqueName('UpdateContract');
    const child = await createChildViaApi(page, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2020-01-15',
      gender: 'female',
    });

    await createChildContractViaApi(page, orgId, child.id, {
      from: formatDateForApi('2024-01-01'),
      section_id: defaultSectionId,
      properties: { care_type: 'ganztag' },
    });

    try {
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      const contractRow = page.locator('tbody tr').first();
      await expect(contractRow).toBeVisible({ timeout: 10000 });
      await expect(contractRow.getByText(/Full-Time/i)).toBeVisible();

      await contractRow.getByRole('button', { name: /Edit/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      await page.getByLabel(/End Date/i).fill('2026-12-31');

      await page.getByRole('button', { name: /Save/i }).click();
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      await expect(page.getByText(/Full-Time/i).first()).toBeVisible({ timeout: 10000 });
      await expect(page.getByText(/Dec 31, 2026/)).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('should delete a contract from history page', async ({ page }) => {
    const childName = uniqueName('DeleteContract');
    const child = await createChildViaApi(page, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2019-08-20',
      gender: 'male',
    });

    await createChildContractViaApi(page, orgId, child.id, {
      from: formatDateForApi('2024-01-01'),
      section_id: defaultSectionId,
      properties: { care_type: 'halbtag' },
    });

    try {
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/Half-Day/i)).toBeVisible({ timeout: 10000 });

      const contractRow = page.locator('tbody tr').first();
      await contractRow.getByRole('button', { name: /Delete/i }).click();

      await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });

      await page
        .getByRole('alertdialog')
        .getByRole('button', { name: /Delete|Confirm|Yes/i })
        .click();

      await expect(page.getByText(/Half-Day/i)).not.toBeVisible({ timeout: 10000 });
      await expect(page.getByText(/No contracts found/i).first()).toBeVisible();
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });
});

test.describe('Child Contract Workflow - create child, add contract, move section', () => {
  test('should create child, add new contract (ending previous), move section, and verify', async ({
    page,
  }, testInfo) => {
    testInfo.setTimeout(60000);
    const childName = uniqueName('Workflow');
    const child = await createChildViaApi(page, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2021-03-15',
      gender: 'female',
    });

    await createChildContractViaApi(page, orgId, child.id, {
      from: formatDateForApi('2024-01-01'),
      section_id: defaultSectionId,
      properties: { care_type: 'ganztag' },
    });

    try {
      await page.goto(`/organizations/${orgId}/children`);
      await page.waitForLoadState('networkidle');

      await page.getByPlaceholder(/Search/i).fill(childName);
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(childName)).toBeVisible({ timeout: 10000 });

      const childRow = page.getByRole('row').filter({ hasText: childName });
      await childRow.getByRole('button', { name: /Add Contract/i }).click();

      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
      await expect(page.getByText(/has an active contract/i)).toBeVisible({ timeout: 5000 });

      const endContractCheckbox = page.locator('#endCurrentContract');
      await expect(endContractCheckbox).toBeChecked();

      await page.getByLabel(/Start Date/i).fill('2025-01-01');

      await page.getByRole('combobox', { name: /Sections/i }).click();
      await page.getByRole('option').first().click();

      await page.getByRole('button', { name: /Save/i }).click();
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      await expect(page.getByText(/previous contract.*ended|contract.*ended/i).first()).toBeVisible(
        { timeout: 5000 }
      );

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      const contractRows = page.locator('tbody tr');
      await expect(contractRows).toHaveCount(2, { timeout: 10000 });

      await expect(page.getByText(/Full-Time/i).first()).toBeVisible();
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });
});

test.describe('Employee Contracts - CRUD Operations', () => {
  test('should add a new contract from history page', async ({ page }) => {
    const employeeName = uniqueName('AddContract');
    const employee = await createEmployeeViaApi(page, orgId, {
      first_name: employeeName,
      last_name: 'Test',
      gender: 'male',
      birthdate: '1990-05-15',
    });

    try {
      await page.goto(`/organizations/${orgId}/employees/${employee.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/No contracts found/i).first()).toBeVisible({
        timeout: 10000,
      });

      await page.getByRole('button', { name: /New Contract/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      await page.getByLabel(/Start Date/i).fill('2024-01-01');

      await page.getByRole('combobox', { name: /Sections/i }).click();
      await page.getByRole('option').first().click();

      await page.getByRole('combobox', { name: /Pay Plan/i }).click();
      await page.getByRole('option').first().click();

      await page.getByRole('combobox', { name: /Staff Category/i }).click();
      await page.getByRole('option', { name: /Qualified/i }).click();

      await page.getByLabel(/Grade/i).fill('S8a');
      await page.getByLabel(/Step/i).clear();
      await page.getByLabel(/Step/i).fill('3');
      await page.getByLabel(/Weekly Hours/i).clear();
      await page.getByLabel(/Weekly Hours/i).fill('39');

      await page.getByRole('button', { name: /Save/i }).click();
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      await expect(page.getByText(/Qualified Staff/i)).toBeVisible({ timeout: 10000 });
      await expect(page.getByText('S8a')).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('should update a contract from history page', async ({ page }) => {
    const employeeName = uniqueName('UpdateContract');
    const employee = await createEmployeeViaApi(page, orgId, {
      first_name: employeeName,
      last_name: 'Test',
      gender: 'male',
      birthdate: '1988-03-12',
    });

    await createEmployeeContractViaApi(page, orgId, employee.id, {
      from: formatDateForApi('2024-01-01'),
      section_id: defaultSectionId,
      staff_category: 'qualified',
      grade: 'S8a',
      step: 2,
      weekly_hours: 39,
      payplan_id: payplanId,
    });

    try {
      await page.goto(`/organizations/${orgId}/employees/${employee.id}/contracts`);
      await page.waitForLoadState('networkidle');

      const contractRow = page.locator('tbody tr').first();
      await expect(contractRow).toBeVisible({ timeout: 10000 });
      await expect(contractRow.getByText(/Qualified Staff/i)).toBeVisible();

      await contractRow.getByRole('button', { name: /Edit/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      await page.getByRole('combobox', { name: /Staff Category/i }).click();
      await page.getByRole('option', { name: /Non-pedagogical/i }).click();

      const stepInput = page.getByLabel(/Step/i);
      await stepInput.clear();
      await stepInput.fill('4');

      await page.getByRole('button', { name: /Save/i }).click();
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      await expect(page.getByText(/Non-pedagogical/i)).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });

  test('should delete a contract from history page', async ({ page }) => {
    const employeeName = uniqueName('DeleteContract');
    const employee = await createEmployeeViaApi(page, orgId, {
      first_name: employeeName,
      last_name: 'Test',
      gender: 'female',
      birthdate: '1992-07-08',
    });

    await createEmployeeContractViaApi(page, orgId, employee.id, {
      from: formatDateForApi('2024-01-01'),
      section_id: defaultSectionId,
      staff_category: 'supplementary',
      grade: 'S2',
      step: 1,
      weekly_hours: 20,
      payplan_id: payplanId,
    });

    try {
      await page.goto(`/organizations/${orgId}/employees/${employee.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/Supplementary Staff/i)).toBeVisible({ timeout: 10000 });

      const contractRow = page.locator('tbody tr').first();
      await contractRow.getByRole('button', { name: /Delete/i }).click();

      await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });

      await page
        .getByRole('alertdialog')
        .getByRole('button', { name: /Delete|Confirm|Yes/i })
        .click();

      await expect(page.getByText(/Supplementary Staff/i)).not.toBeVisible({ timeout: 10000 });
      await expect(page.getByText(/No contracts found/i).first()).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });
});
