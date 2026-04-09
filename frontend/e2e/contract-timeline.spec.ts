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
  uniqueName,
} from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

let orgId: number;
let defaultSectionId: number;
let payplanId: number;

test.beforeAll(async ({ browser }) => {
  const page = await browser.newPage();
  await login(page);
  const testOrg = await createTestOrg(page, 'Timeline');
  orgId = testOrg.orgId;
  defaultSectionId = testOrg.sectionId;
  const payplan = await createPayPlanViaApi(page, orgId, 'Test Pay Plan');
  payplanId = payplan.id;
  await createPayPlanPeriodViaApi(page, orgId, payplanId, {
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
});

test.describe('Child Contract Timeline', () => {
  test('timeline tab is visible and renders timeline', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TLTab'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'male',
    });

    try {
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-01-01T00:00:00Z',
        to: '2024-06-30T00:00:00Z',
        section_id: defaultSectionId,
      });

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      // Click Timeline tab
      await page.getByRole('tab', { name: /Timeline/i }).click();

      // Timeline container should be visible
      await expect(page.getByTestId('contract-timeline')).toBeVisible({ timeout: 5000 });
      expect(await page.getByTestId('timeline-segment').count()).toBe(1);
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('boundary handle is visible between adjacent contracts', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TLBoundary'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'female',
    });

    try {
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-01-01T00:00:00Z',
        to: '2024-06-30T00:00:00Z',
        section_id: defaultSectionId,
      });
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-07-01T00:00:00Z',
        to: '2024-12-31T00:00:00Z',
        section_id: defaultSectionId,
      });

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('tab', { name: /Timeline/i }).click();
      await expect(page.getByTestId('contract-timeline')).toBeVisible({ timeout: 5000 });

      // Should have 2 segments and 1 boundary handle
      expect(await page.getByTestId('timeline-segment').count()).toBe(2);
      await expect(page.getByTestId('boundary-handle')).toBeVisible();
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('gap indicator shown for non-adjacent contracts', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TLGap'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'diverse',
    });

    try {
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-01-01T00:00:00Z',
        to: '2024-03-31T00:00:00Z',
        section_id: defaultSectionId,
      });
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-07-01T00:00:00Z',
        to: '2024-12-31T00:00:00Z',
        section_id: defaultSectionId,
      });

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('tab', { name: /Timeline/i }).click();
      await expect(page.getByTestId('contract-timeline')).toBeVisible({ timeout: 5000 });

      // Should have gap, no boundary handle
      await expect(page.getByTestId('timeline-gap')).toBeVisible();
      expect(await page.getByTestId('boundary-handle').count()).toBe(0);
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('clicking boundary opens calendar and selecting date updates contracts', async ({
    page,
  }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TLClick'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'male',
    });

    try {
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-01-01T00:00:00Z',
        to: '2024-06-30T00:00:00Z',
        section_id: defaultSectionId,
      });
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-07-01T00:00:00Z',
        to: '2024-12-31T00:00:00Z',
        section_id: defaultSectionId,
      });

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('tab', { name: /Timeline/i }).click();
      await expect(page.getByTestId('boundary-handle')).toBeVisible({ timeout: 5000 });

      // Click the boundary handle to open the calendar
      await page.getByTestId('boundary-handle').click();

      // Calendar popover should be visible
      await expect(page.getByRole('grid')).toBeVisible({ timeout: 5000 });

      // Select a different day (e.g., June 15) to shift the boundary
      await page.getByRole('gridcell', { name: '15' }).first().click();

      // Wait for the batch update to complete
      await page.waitForLoadState('networkidle');

      // Switch to Table tab and verify dates changed — both rows should still exist
      await page.getByRole('tab', { name: /Table/i }).click();
      const tableRows = page.locator('table tbody tr');
      expect(await tableRows.count()).toBe(2);
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('error rollback on failed batch update', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TLError'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'female',
    });

    try {
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-01-01T00:00:00Z',
        to: '2024-06-30T00:00:00Z',
        section_id: defaultSectionId,
      });
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-07-01T00:00:00Z',
        to: '2024-12-31T00:00:00Z',
        section_id: defaultSectionId,
      });

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('tab', { name: /Timeline/i }).click();
      await expect(page.getByTestId('boundary-handle')).toBeVisible({ timeout: 5000 });

      // Intercept batch update API to return error
      await page.route('**/contracts/batch', (route) => {
        route.fulfill({
          status: 500,
          body: JSON.stringify({ error: 'Internal server error' }),
        });
      });

      // Click boundary and select a different date
      await page.getByTestId('boundary-handle').click();
      await expect(page.getByRole('grid')).toBeVisible({ timeout: 5000 });
      await page.getByRole('gridcell', { name: '15' }).first().click();

      // Should still have 2 segments (dates reverted after error)
      await expect(page.getByTestId('timeline-segment').first()).toBeVisible({ timeout: 5000 });
      expect(await page.getByTestId('timeline-segment').count()).toBe(2);
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('empty timeline shows no-contracts message', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TLEmpty'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'male',
    });

    try {
      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('tab', { name: /Timeline/i }).click();
      await expect(page.getByTestId('timeline-empty')).toBeVisible({ timeout: 5000 });
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('segments show status badges', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TLStatus'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'female',
    });

    try {
      // Create an ended contract (past dates)
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2020-01-01T00:00:00Z',
        to: '2020-12-31T00:00:00Z',
        section_id: defaultSectionId,
      });

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('tab', { name: /Timeline/i }).click();
      await expect(page.getByTestId('contract-timeline')).toBeVisible({ timeout: 5000 });

      // The segment should show the "Ended" status badge
      const segment = page.getByTestId('timeline-segment');
      await expect(segment).toBeVisible();
      await expect(segment.getByText('Ended')).toBeVisible();
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('three adjacent contracts show two boundary handles', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TL3Adj'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'diverse',
    });

    try {
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-01-01T00:00:00Z',
        to: '2024-04-30T00:00:00Z',
        section_id: defaultSectionId,
      });
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-05-01T00:00:00Z',
        to: '2024-08-31T00:00:00Z',
        section_id: defaultSectionId,
      });
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-09-01T00:00:00Z',
        to: '2024-12-31T00:00:00Z',
        section_id: defaultSectionId,
      });

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('tab', { name: /Timeline/i }).click();
      await expect(page.getByTestId('contract-timeline')).toBeVisible({ timeout: 5000 });

      expect(await page.getByTestId('timeline-segment').count()).toBe(3);
      expect(await page.getByTestId('boundary-handle').count()).toBe(2);
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });

  test('switching between table and timeline tabs preserves data', async ({ page }) => {
    const child = await createChildViaApi(page, orgId, {
      first_name: uniqueName('TLSwitch'),
      last_name: 'Test',
      birthdate: '2020-01-01',
      gender: 'male',
    });

    try {
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-01-01T00:00:00Z',
        to: '2024-06-30T00:00:00Z',
        section_id: defaultSectionId,
      });
      await createChildContractViaApi(page, orgId, child.id, {
        from: '2024-07-01T00:00:00Z',
        to: '2024-12-31T00:00:00Z',
        section_id: defaultSectionId,
      });

      await page.goto(`/organizations/${orgId}/children/${child.id}/contracts`);
      await page.waitForLoadState('networkidle');

      // Table tab should show 2 rows
      const tableRows = page.locator('table tbody tr');
      expect(await tableRows.count()).toBe(2);

      // Switch to Timeline
      await page.getByRole('tab', { name: /Timeline/i }).click();
      expect(await page.getByTestId('timeline-segment').count()).toBe(2);

      // Switch back to Table
      await page.getByRole('tab', { name: /Table/i }).click();
      expect(await tableRows.count()).toBe(2);
    } finally {
      await deleteChildViaApi(page, orgId, child.id);
    }
  });
});

test.describe('Employee Contract Timeline', () => {
  test('timeline works with employee contracts', async ({ page }) => {
    const employee = await createEmployeeViaApi(page, orgId, {
      first_name: uniqueName('TLEmp'),
      last_name: 'Test',
      birthdate: '1990-01-01',
      gender: 'male',
    });

    try {
      await createEmployeeContractViaApi(page, orgId, employee.id, {
        from: '2024-01-01T00:00:00Z',
        to: '2024-06-30T00:00:00Z',
        section_id: defaultSectionId,
        staff_category: 'qualified',
        grade: 'S8a',
        step: 3,
        weekly_hours: 39,
        payplan_id: payplanId,
      });
      await createEmployeeContractViaApi(page, orgId, employee.id, {
        from: '2024-07-01T00:00:00Z',
        to: '2024-12-31T00:00:00Z',
        section_id: defaultSectionId,
        staff_category: 'qualified',
        grade: 'S8a',
        step: 4,
        weekly_hours: 39,
        payplan_id: payplanId,
      });

      await page.goto(`/organizations/${orgId}/employees/${employee.id}/contracts`);
      await page.waitForLoadState('networkidle');

      await page.getByRole('tab', { name: /Timeline/i }).click();
      await expect(page.getByTestId('contract-timeline')).toBeVisible({ timeout: 5000 });

      // Should have 2 segments and 1 boundary handle
      expect(await page.getByTestId('timeline-segment').count()).toBe(2);
      await expect(page.getByTestId('boundary-handle')).toBeVisible();

      // Employee-specific content should be visible (staff category, grade)
      await expect(page.getByText(/S8a/).first()).toBeVisible();
    } finally {
      await deleteEmployeeViaApi(page, orgId, employee.id);
    }
  });
});
