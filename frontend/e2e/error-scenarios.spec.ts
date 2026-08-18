import { test, expect } from '@playwright/test';
import {
  login,
  createTestOrg,
  deleteTestOrg,
  getOrganizationsViaApi,
  deleteOrganizationViaApi,
  uniqueName,
} from './utils/test-helpers';

// Ensure English locale for all tests
test.use({ locale: 'en-US' });

/**
 * The app's own alerts. Next.js mounts a permanently-present, permanently-empty
 * `role="alert"` route announcer, so a bare getByRole('alert') either matches it
 * or trips strict mode -- either way it says nothing about the app.
 */
function appAlert(page: import('@playwright/test').Page) {
  return page.locator('[role="alert"]:not(#__next-route-announcer__)').first();
}

test.describe('Form Validation Errors', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'ErrorTests');
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
  });

  /**
   * Opens a create dialog, submits it empty, and returns the dialog.
   *
   * These used to assert only that the dialog was still open afterwards, which
   * the dialog already was before the click. That catches "the submit went
   * through when it should not have" and nothing else -- in particular not the
   * thing all three names promise, which is that the user was told what to fix.
   */
  async function submitEmpty(page: import('@playwright/test').Page, openButton: RegExp) {
    await page.getByRole('button', { name: openButton }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5000 });
    await dialog
      .getByRole('button', { name: /^(save|create|add)/i })
      .first()
      .click();
    return dialog;
  }

  /**
   * The bar every rejected form has to clear: the submit did not go through, at
   * least one input is marked for assistive technology, and there is prose next
   * to it saying why. A marked field with no message is a red outline the user
   * cannot act on; a message with no marking cannot be found by a screen reader.
   */
  async function expectRejectedWithReason(dialog: import('@playwright/test').Locator) {
    await expect(dialog).toBeVisible();
    await expect(dialog.locator('[aria-invalid="true"]').first()).toBeVisible({ timeout: 5000 });

    const marked = dialog.locator('[aria-invalid="true"]').first();
    const describedBy = await marked.getAttribute('aria-describedby');
    expect(describedBy, 'a marked field must point at the text explaining it').toBeTruthy();
    await expect(dialog.locator(`#${describedBy}`)).not.toBeEmpty();
  }

  test('rejecting an organization with no name says which field is wrong', async ({ page }) => {
    await page.goto('/organizations');
    await page.waitForLoadState('load');
    await expectRejectedWithReason(await submitEmpty(page, /new organization/i));
  });

  test('rejecting an employee with no name says which field is wrong', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees`);
    await page.waitForLoadState('load');
    await expectRejectedWithReason(await submitEmpty(page, /new employee/i));
  });

  test('rejecting a child with no name says which field is wrong', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children`);
    await page.waitForLoadState('load');
    const dialog = await submitEmpty(page, /new child/i);
    await expectRejectedWithReason(dialog);

    // The child dialog is one of the four that also renders the summary, so it
    // can be held to the higher bar: every problem listed in one place.
    const summary = dialog.getByTestId('form-error-summary');
    await expect(summary).toBeVisible();
    expect(Number(await summary.getAttribute('data-count'))).toBeGreaterThan(0);
  });
});

test.describe('Authentication Error Scenarios', () => {
  test('should redirect to login when accessing protected page without auth', async ({ page }) => {
    await page.goto('/organizations');
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
  });

  test('should redirect to login when accessing nested protected page without auth', async ({
    page,
  }) => {
    await page.goto('/organizations/1/employees');
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
  });

  test('rejected credentials are explained, not just refused', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });

    await page.getByLabel(/email/i).fill('nonexistent@example.com');
    await page.getByLabel(/password/i).fill('wrongpassword123');
    await page.getByRole('button', { name: /sign in|login/i }).click();

    // Staying on /login was the whole of this assertion, twice over, on
    // consecutive lines. Wrong credentials keep you here whether or not the
    // page says anything, so it could not see the failure it was named for.
    const alert = appAlert(page);
    await expect(alert).toBeVisible({ timeout: 10000 });
    await expect(alert).not.toBeEmpty();
    await expect(page).toHaveURL(/.*login/);
  });

  test('an empty login form names the fields it needs', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByLabel(/email/i)).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /sign in|login/i }).click();

    // Client-side rules, so this is field marking rather than the server's
    // alert -- and the marking has to carry its reason with it.
    const marked = page.locator('[aria-invalid="true"]').first();
    await expect(marked).toBeVisible({ timeout: 5000 });
    const describedBy = await marked.getAttribute('aria-describedby');
    expect(describedBy, 'a marked field must point at the text explaining it').toBeTruthy();
    await expect(page.locator(`#${describedBy}`)).not.toBeEmpty();
    await expect(page).toHaveURL(/.*login/);
  });
});

test.describe('Not Found Scenarios', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'NotFound');
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
  });

  /**
   * These three asserted that the body was a non-empty string and that the text
   * "TypeError" was not on screen. Both hold for a blank page, a 500, and -- as
   * the organization case turned out -- a fully working page belonging to
   * somebody else. They could not fail.
   */

  test('a missing employee is reported, not rendered as an empty history', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/employees/99999/contracts`);
    await page.waitForLoadState('load');

    const alert = appAlert(page);
    await expect(alert).toBeVisible({ timeout: 10000 });
    await expect(alert).not.toBeEmpty();
  });

  test('a missing child is reported, not rendered as an empty history', async ({ page }) => {
    await page.goto(`/organizations/${orgId}/children/99999/contracts`);
    await page.waitForLoadState('load');

    const alert = appAlert(page);
    await expect(alert).toBeVisible({ timeout: 10000 });
    await expect(alert).not.toBeEmpty();
  });

  // Known defect, deliberately not skipped quietly. /organizations/99999/...
  // renders a working page for an organization that does not exist, with a
  // *different* organization's name in the selector and an empty state inviting
  // the user to add their first employee. Nothing tells them the org is wrong.
  //
  // fixme rather than a weakened assertion: what a user should see here is a
  // product decision, and a passing test would go on claiming this is handled.
  test.fixme('a missing organization is reported to the user', async ({ page }) => {
    await page.goto('/organizations/99999/employees');
    await page.waitForLoadState('load');

    await expect(appAlert(page)).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Duplicate Resource Errors', () => {
  test('should show error when creating organization with duplicate name', async ({ page }) => {
    await login(page);

    // Create a temporary org to test duplication against
    const testOrg = await createTestOrg(page, 'DupTest');

    try {
      await page.goto('/organizations');
      await page.waitForLoadState('load');

      await page.getByRole('button', { name: /new organization/i }).click();
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // Get the test org's actual name for duplication
      const orgs = await getOrganizationsViaApi(page);
      const targetOrg = orgs.find((o) => o.id === testOrg.orgId);
      const orgName = targetOrg!.name;

      await page.getByLabel('Name', { exact: true }).fill(orgName);
      // Explicitly interact with the State combobox inside the dialog
      const dialog = page.getByRole('dialog');
      await dialog.locator('[role="combobox"]').first().click();
      await page.getByRole('option', { name: /berlin/i }).click();
      await page.getByLabel(/Default Section Name/i).fill('Default');

      await page.getByRole('button', { name: /save/i }).click();

      // Should show an error toast (duplicate name causes API error)
      // and the dialog should remain open
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    } finally {
      await deleteTestOrg(page, testOrg.orgId);
    }
  });
});
