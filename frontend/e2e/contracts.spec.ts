import { test, expect } from '@playwright/test';
import {
  login,
  getApiToken,
  getFirstOrganization,
  createChildViaApi,
  createEmployeeViaApi,
  deleteChildViaApi,
  deleteEmployeeViaApi,
  createChildContractViaApi,
  createEmployeeContractViaApi,
  getSectionsViaApi,
  getPayPlansViaApi,
  uniqueName,
  formatDateForApi,
} from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

test.describe('Child Contracts - CRUD Operations', () => {
  let token: string;
  let orgId: number;
  let defaultSectionId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);
    orgId = org.id;
    // Get the default section for creating contracts
    const sections = await getSectionsViaApi(page, token, orgId);
    defaultSectionId = sections[0].id;
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should add a new contract from history page', async ({ page }) => {
    // Create a fresh child without any contracts
    const childName = uniqueName('AddContract');
    const child = await createChildViaApi(page, token, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2020-06-15',
      gender: 'female',
    });

    try {
      // Navigate directly to contract history page
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      // Should show no contracts message (use first() since text may appear multiple times)
      await expect(page.getByText(/No contracts found/i).first()).toBeVisible({
        timeout: 10000,
      });

      // Click new contract button
      await page.getByRole('button', { name: /New Contract/i }).click();

      // Dialog should open
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Fill the form - use a past date to ensure Active status
      await page.getByLabel(/Start Date/i).fill('2024-01-01');

      // Select a section (required)
      await page.getByRole('combobox', { name: /Sections/i }).click();
      await page.getByRole('option').first().click();

      // Wait for property suggestions to load and click on them
      // The PropertyTagInput shows clickable suggestion buttons
      await expect(page.getByText(/Available:/i)).toBeVisible({ timeout: 10000 });
      const suggestionsArea = page.locator('text=Available:').locator('..');

      // Click on ganztag suggestion (or first available suggestion)
      const gantzagSuggestion = suggestionsArea.locator('button', { hasText: 'ganztag' }).first();
      if (await gantzagSuggestion.isVisible({ timeout: 2000 }).catch(() => false)) {
        await gantzagSuggestion.click();
      }

      // Click on ndh suggestion
      const ndhSuggestion = suggestionsArea.locator('button', { hasText: 'ndh' }).first();
      if (await ndhSuggestion.isVisible({ timeout: 2000 }).catch(() => false)) {
        await ndhSuggestion.click();
      }

      // Submit
      await page.getByRole('button', { name: /Save/i }).click();

      // Dialog should close
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      // Contract should appear in table with selected properties
      await expect(page.getByText(/ganztag/i)).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteChildViaApi(page, token, orgId, child.id);
    }
  });

  test('should show suggested properties from government funding', async ({ page }) => {
    // Create a child without contracts
    const childName = uniqueName('SuggestAttr');
    const child = await createChildViaApi(page, token, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2022-03-15',
      gender: 'male',
    });

    try {
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      // Click new contract button
      await page.getByRole('button', { name: /New Contract/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Fill a date that overlaps with Berlin funding period (2025-02-01 onwards)
      await page.getByLabel(/Start Date/i).fill('2025-03-01');

      // Select a section (required)
      await page.getByRole('combobox', { name: /Sections/i }).click();
      await page.getByRole('option').first().click();

      // Wait for suggestions to appear (Berlin funding has these properties)
      // The suggestions should show property values from the government funding
      await expect(page.getByText(/Available:/i)).toBeVisible({ timeout: 10000 });

      // Click on a suggested property - suggestions are buttons with icon + text
      // Look within the suggestions area (after "Available:" label)
      const suggestionsArea = page.locator('text=Available:').locator('..');
      const gantzagSuggestion = suggestionsArea.locator('button', { hasText: 'ganztag' }).first();
      await expect(gantzagSuggestion).toBeVisible({ timeout: 5000 });
      await gantzagSuggestion.click();

      // After clicking, the tag appears in the input and suggestion disappears
      const dialog = page.getByRole('dialog');
      // The selected tag now appears as a badge (not a button) in the input area
      // We check that "ganztag" text exists but the suggestion button is gone
      await expect(dialog.getByText('ganztag').first()).toBeVisible();

      // Add "ndh" by clicking its suggestion
      const ndhSuggestion = suggestionsArea.locator('button', { hasText: 'ndh' }).first();
      if (await ndhSuggestion.isVisible({ timeout: 2000 }).catch(() => false)) {
        await ndhSuggestion.click();
        await expect(dialog.getByText('ndh').first()).toBeVisible();
      }

      // Save the contract
      await page.getByRole('button', { name: /Save/i }).click();
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      // Verify attributes appear in the table
      await expect(page.getByText(/ganztag/i)).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteChildViaApi(page, token, orgId, child.id);
    }
  });

  test('should update a contract from history page', async ({ page }) => {
    // Create a child with a contract
    const childName = uniqueName('UpdateContract');
    const child = await createChildViaApi(page, token, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2020-01-15',
      gender: 'female',
    });

    await createChildContractViaApi(page, token, orgId, child.id, {
      from: formatDateForApi('2024-01-01'),
      section_id: defaultSectionId,
      properties: { care_type: 'ganztag' },
    });

    try {
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      // Find the contract row and click edit
      const contractRow = page.locator('tbody tr').first();
      await expect(contractRow).toBeVisible({ timeout: 10000 });
      await expect(contractRow.getByText(/ganztag/i)).toBeVisible();

      await contractRow.getByRole('button', { name: /Edit/i }).click();

      // Dialog should open
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Add a property by clicking on suggestion
      const suggestionsArea = page.locator('text=Available:').locator('..');
      const ndhSuggestion = suggestionsArea.locator('button', { hasText: 'ndh' }).first();
      if (await ndhSuggestion.isVisible({ timeout: 5000 }).catch(() => false)) {
        await ndhSuggestion.click();
      }

      // Submit
      await page.getByRole('button', { name: /Save/i }).click();

      // Dialog should close
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      // Should show updated property
      await expect(page.getByText(/ndh/i)).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteChildViaApi(page, token, orgId, child.id);
    }
  });

  test('should delete a contract from history page', async ({ page }) => {
    // Create a child with a contract
    const childName = uniqueName('DeleteContract');
    const child = await createChildViaApi(page, token, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2019-08-20',
      gender: 'male',
    });

    await createChildContractViaApi(page, token, orgId, child.id, {
      from: formatDateForApi('2024-01-01'),
      section_id: defaultSectionId,
      properties: { care_type: 'halbtag' },
    });

    try {
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      // Verify contract exists
      await expect(page.getByText(/halbtag/i)).toBeVisible({ timeout: 10000 });

      // Find the contract row and click delete
      const contractRow = page.locator('tbody tr').first();
      await contractRow.getByRole('button', { name: /Delete/i }).click();

      // Confirmation dialog should open
      await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });

      // Click the confirm/continue button in the alert dialog
      await page
        .getByRole('alertdialog')
        .getByRole('button', { name: /Delete|Confirm|Yes/i })
        .click();

      // Contract should be removed
      await expect(page.getByText(/halbtag/i)).not.toBeVisible({ timeout: 10000 });
      await expect(page.getByText(/No contracts found/i).first()).toBeVisible();
    } finally {
      await deleteChildViaApi(page, token, orgId, child.id);
    }
  });
});

test.describe('Child Contract Workflow - create child, add contract, move section', () => {
  let token: string;
  let orgId: number;
  let defaultSectionId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);
    orgId = org.id;
    const sections = await getSectionsViaApi(page, token, orgId);
    defaultSectionId = sections[0].id;
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should create child, add new contract (ending previous), move section, and verify', async ({
    page,
  }, testInfo) => {
    testInfo.setTimeout(60000);
    // 1. Create child with initial contract via API
    const childName = uniqueName('Workflow');
    const child = await createChildViaApi(page, token, orgId, {
      first_name: childName,
      last_name: 'Test',
      birthdate: '2021-03-15',
      gender: 'female',
    });

    await createChildContractViaApi(page, token, orgId, child.id, {
      from: formatDateForApi('2024-01-01'),
      section_id: defaultSectionId,
      properties: { care_type: 'ganztag' },
    });

    try {
      // 2. Navigate to children list and add a new contract via the UI
      await page.goto(`/organizations/${orgId}/children`);
      await page.waitForLoadState('networkidle');

      // Search for the child to find it in the paginated list
      await page.getByPlaceholder(/Search/i).fill(childName);
      await page.waitForLoadState('networkidle');

      // Find the child row
      await expect(page.getByText(childName)).toBeVisible({ timeout: 10000 });

      // Click the add-contract button (FileText icon)
      const childRow = page.getByRole('row').filter({ hasText: childName });
      await childRow.getByRole('button', { name: /Add Contract/i }).click();

      // Dialog should open showing active contract info
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
      await expect(page.getByText(/has an active contract/i)).toBeVisible({ timeout: 5000 });

      // The "End current contract" checkbox should be checked by default
      const endContractCheckbox = page.locator('#endCurrentContract');
      await expect(endContractCheckbox).toBeChecked();

      // Fill the start date for the new contract
      await page.getByLabel(/Start Date/i).fill('2025-01-01');

      // Select a section (required)
      await page.getByRole('combobox', { name: /Sections/i }).click();
      await page.getByRole('option').first().click();

      // Click on a different property suggestion to make the contract different
      await expect(page.getByText(/Available:/i)).toBeVisible({ timeout: 10000 });
      const suggestionsArea = page.locator('text=Available:').locator('..');
      const halbtagSuggestion = suggestionsArea.locator('button', { hasText: 'halbtag' }).first();
      if (await halbtagSuggestion.isVisible({ timeout: 3000 }).catch(() => false)) {
        await halbtagSuggestion.click();
      }

      // Submit - this should end the old contract and create the new one
      await page.getByRole('button', { name: /Save/i }).click();
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      // Verify the toast indicates the previous contract was ended
      await expect(page.getByText(/previous contract.*ended|contract.*ended/i).first()).toBeVisible(
        { timeout: 5000 }
      );

      // 3. Verify contract history shows both contracts
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      // Should have 2 contracts in the table
      const contractRows = page.locator('tbody tr');
      await expect(contractRows).toHaveCount(2, { timeout: 10000 });

      // The old contract should be ended (has a to date), the new one should be active
      await expect(page.getByText(/Ended/i).first()).toBeVisible();
      await expect(page.getByText(/Active/i).first()).toBeVisible();
    } finally {
      await deleteChildViaApi(page, token, orgId, child.id);
    }
  });
});

test.describe('Employee Contracts - CRUD Operations', () => {
  let token: string;
  let orgId: number;
  let defaultSectionId: number;
  let payplanId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    token = await getApiToken(page);
    const org = await getFirstOrganization(page, token);
    orgId = org.id;
    const sections = await getSectionsViaApi(page, token, orgId);
    defaultSectionId = sections[0].id;
    const payplans = await getPayPlansViaApi(page, token, orgId);
    payplanId = payplans[0].id;
    await page.close();
  });

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should add a new contract from history page', async ({ page }) => {
    // Create a fresh employee without any contracts
    const employeeName = uniqueName('AddContract');
    const employee = await createEmployeeViaApi(page, token, orgId, {
      first_name: employeeName,
      last_name: 'Test',
      gender: 'male',
      birthdate: '1990-05-15',
    });

    try {
      // Navigate directly to contract history page
      await page.goto(`/organizations/${orgId}/employees/${employee.id}/contracts`);
      await page.waitForLoadState('networkidle');

      // Should show no contracts message
      await expect(page.getByText(/No contracts found/i).first()).toBeVisible({
        timeout: 10000,
      });

      // Click new contract button
      await page.getByRole('button', { name: /New Contract/i }).click();

      // Dialog should open
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Fill the form - use a past date
      await page.getByLabel(/Start Date/i).fill('2024-01-01');

      // Select a section (required)
      await page.getByRole('combobox', { name: /Sections/i }).click();
      await page.getByRole('option').first().click();

      // Select a pay plan (required)
      await page.getByRole('combobox', { name: /Pay Plan/i }).click();
      await page.getByRole('option').first().click();

      // Select staff category
      await page.getByRole('combobox', { name: /Staff Category/i }).click();
      await page.getByRole('option', { name: /Qualified/i }).click();

      await page.getByLabel(/Grade/i).fill('S8a');
      await page.getByLabel(/Step/i).clear();
      await page.getByLabel(/Step/i).fill('3');
      await page.getByLabel(/Weekly Hours/i).clear();
      await page.getByLabel(/Weekly Hours/i).fill('39');

      // Submit
      await page.getByRole('button', { name: /Save/i }).click();

      // Dialog should close
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      // Contract should appear in table
      await expect(page.getByText(/Qualified Staff/i)).toBeVisible({ timeout: 10000 });
      await expect(page.getByText('S8a')).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, token, orgId, employee.id);
    }
  });

  test('should update a contract from history page', async ({ page }) => {
    // Create an employee with a contract
    const employeeName = uniqueName('UpdateContract');
    const employee = await createEmployeeViaApi(page, token, orgId, {
      first_name: employeeName,
      last_name: 'Test',
      gender: 'male',
      birthdate: '1988-03-12',
    });

    await createEmployeeContractViaApi(page, token, orgId, employee.id, {
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

      // Find the contract row and click edit
      const contractRow = page.locator('tbody tr').first();
      await expect(contractRow).toBeVisible({ timeout: 10000 });
      await expect(contractRow.getByText(/Qualified Staff/i)).toBeVisible();

      await contractRow.getByRole('button', { name: /Edit/i }).click();

      // Dialog should open
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Update staff category and step
      await page.getByRole('combobox', { name: /Staff Category/i }).click();
      await page.getByRole('option', { name: /Non-pedagogical/i }).click();

      const stepInput = page.getByLabel(/Step/i);
      await stepInput.clear();
      await stepInput.fill('4');

      // Submit
      await page.getByRole('button', { name: /Save/i }).click();

      // Dialog should close
      await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

      // Should show updated values
      await expect(page.getByText(/Non-pedagogical/i)).toBeVisible({ timeout: 10000 });
    } finally {
      await deleteEmployeeViaApi(page, token, orgId, employee.id);
    }
  });

  test('should delete a contract from history page', async ({ page }) => {
    // Create an employee with a contract
    const employeeName = uniqueName('DeleteContract');
    const employee = await createEmployeeViaApi(page, token, orgId, {
      first_name: employeeName,
      last_name: 'Test',
      gender: 'female',
      birthdate: '1992-07-08',
    });

    await createEmployeeContractViaApi(page, token, orgId, employee.id, {
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

      // Verify contract exists
      await expect(page.getByText(/Supplementary Staff/i)).toBeVisible({ timeout: 10000 });

      // Find the contract row and click delete
      const contractRow = page.locator('tbody tr').first();
      await contractRow.getByRole('button', { name: /Delete/i }).click();

      // Confirmation dialog should open
      await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });

      // Click the confirm button
      await page
        .getByRole('alertdialog')
        .getByRole('button', { name: /Delete|Confirm|Yes/i })
        .click();

      // Contract should be removed
      await expect(page.getByText(/Supplementary Staff/i)).not.toBeVisible({ timeout: 10000 });
      await expect(page.getByText(/No contracts found/i).first()).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, token, orgId, employee.id);
    }
  });
});
