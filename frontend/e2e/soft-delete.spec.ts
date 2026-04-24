import { test, expect } from '@playwright/test';

import {
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
  login,
  loginViaForm,
  createOrganizationViaApi,
  createUserViaApi,
  deleteOrganizationViaApi,
  deleteUserViaApi,
  addUserToOrgViaApi,
  getOrganizationsViaApi,
  logoutViaApi,
  uniqueName,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

/**
 * End-to-end coverage for the soft-delete (migration 000015) workflows
 * on users and organizations. These tests exercise the real auth
 * stack — the frontend hits the Go API, the API soft-deletes + revokes
 * sessions, subsequent requests go through the full middleware chain —
 * so a regression in any layer (SessionStore.Lookup filter, handler
 * behaviour, cookie clearing) is loud.
 *
 * Each test is a complete user journey rather than a narrow probe:
 * the journeys that actually matter in production are the ones an
 * incident response would replay.
 */

// -----------------------------------------------------------------
// User soft-delete
// -----------------------------------------------------------------

test.describe('User soft-delete', () => {
  test('deleted user cannot log in; the email can be reused by a fresh registration', async ({
    page,
  }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const email = `sd-login-${Date.now()}@example.com`;
    const originalPassword = 'original-pw-123';

    // 1. Admin creates a user.
    const original = await createUserViaApi(page, {
      name: uniqueName('Soft-Delete Login'),
      email,
      password: originalPassword,
      active: true,
    });

    // 2. Confirm the fresh user can log in. Rules out pre-existing
    //    login failure polluting the subsequent assertion.
    await logoutViaApi(page);
    await loginViaForm(page, email, originalPassword);
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });

    // 3. Admin soft-deletes the user.
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await deleteUserViaApi(page, original.id);

    // 4. Fresh login attempt with the same credentials must fail.
    //    The row exists in the DB (tombstoned) but FindByEmail
    //    returns NotFound via GORM auto-scoping, so the login
    //    handler can't even match the email.
    await logoutViaApi(page);
    await page.goto('/login');
    await page.getByLabel(/email/i).fill(email);
    await page.getByLabel(/password/i).fill(originalPassword);
    await page.getByRole('button', { name: /sign ?in|login/i }).click();
    await expect(page.locator('[role="alert"]:not(#__next-route-announcer__)').first()).toBeVisible(
      { timeout: 10000 }
    );
    await expect(page).toHaveURL(/\/login/);

    // 5. Admin re-registers the same email. The partial unique index
    //    (`lower(email) WHERE deleted_at IS NULL`) allows reuse, and
    //    the fresh account must then be able to log in.
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const fresh = await createUserViaApi(page, {
      name: uniqueName('Reborn'),
      email,
      password: 'fresh-pw-789',
      active: true,
    });
    expect(fresh.id).not.toEqual(original.id);
    await logoutViaApi(page);
    await loginViaForm(page, email, 'fresh-pw-789');
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 });

    // Cleanup.
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await deleteUserViaApi(page, fresh.id).catch(() => {});
  });

  test('existing session for a user is invalidated the moment the user is deleted', async ({
    browser,
  }) => {
    // Scenario: user is working on a tab, admin deletes them from
    // another tab, user's next request must 401. This is the
    // security-critical edge case — without it, a compromised admin
    // who deletes an account thinks the account is gone, but the
    // account's live sessions keep working until token expiry.
    const admin = await browser.newContext();
    const adminPage = await admin.newPage();
    await login(adminPage, ADMIN_EMAIL, ADMIN_PASSWORD);

    const victimContext = await browser.newContext();
    const victimPage = await victimContext.newPage();

    const email = `sd-session-${Date.now()}@example.com`;
    const pw = 'victim-pw-456';
    const user = await createUserViaApi(adminPage, {
      name: uniqueName('Session Victim'),
      email,
      password: pw,
      active: true,
    });

    // Victim logs in on their own browsing context.
    await loginViaForm(victimPage, email, pw);
    await expect(victimPage).not.toHaveURL(/\/login/, { timeout: 10000 });

    // Admin deletes the victim.
    await deleteUserViaApi(adminPage, user.id);

    // Victim's next API call must 401 and the UI lands them back at
    // /login. We probe via /api/v1/me which is the canonical "am I
    // still authenticated" endpoint.
    const status = await victimPage.evaluate(async () => {
      const r = await fetch('/api/v1/me', { credentials: 'same-origin' });
      return r.status;
    });
    expect(status).toBe(401);

    await victimContext.close();
    await admin.close();
  });
});

// -----------------------------------------------------------------
// Organization soft-delete
// -----------------------------------------------------------------

test.describe('Organization soft-delete', () => {
  test('deleted org becomes invisible to its members; the name can be reused', async ({
    page,
  }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const name = uniqueName('Soft-Delete Org');
    const original = await createOrganizationViaApi(page, name);

    // Sanity: the org is fetchable before delete.
    const beforeStatus = await page.evaluate(
      async (id: number) => {
        const r = await fetch(`/api/v1/organizations/${id}`, { credentials: 'same-origin' });
        return r.status;
      },
      original.id
    );
    expect(beforeStatus).toBe(200);

    // Admin deletes the organization.
    await deleteOrganizationViaApi(page, original.id);

    // GET /organizations/:id returns 404 afterwards. The row is
    // tombstoned, not purged, but GORM auto-scoping makes it
    // invisible to default paths.
    const afterStatus = await page.evaluate(
      async (id: number) => {
        const r = await fetch(`/api/v1/organizations/${id}`, { credentials: 'same-origin' });
        return r.status;
      },
      original.id
    );
    expect(afterStatus).toBe(404);

    // List must also exclude the soft-deleted org. Using the
    // shared helper so the response-shape decoding matches the
    // rest of the E2E suite (the raw fetch path tripped over a
    // different envelope shape in CI).
    const listed = await getOrganizationsViaApi(page);
    expect(listed.map((o) => o.id)).not.toContain(original.id);

    // Admin re-registers the same name. Partial unique index
    // (`name WHERE deleted_at IS NULL`) lets this succeed with a
    // fresh id.
    const fresh = await createOrganizationViaApi(page, name);
    expect(fresh.id).not.toEqual(original.id);
    expect(fresh.name).toBe(name);

    // Cleanup.
    await deleteOrganizationViaApi(page, fresh.id).catch(() => {});
  });

  test('member of a deleted org loses access to org-scoped routes', async ({ browser }) => {
    // Critical authorisation boundary: a user who was granted access
    // to an organization must lose that access the moment the org
    // is deleted, independent of their own session state.
    const admin = await browser.newContext();
    const adminPage = await admin.newPage();
    await login(adminPage, ADMIN_EMAIL, ADMIN_PASSWORD);

    const org = await createOrganizationViaApi(adminPage, uniqueName('Member-Access Org'));

    const email = `sd-orgmember-${Date.now()}@example.com`;
    const pw = 'member-pw-789';
    const user = await createUserViaApi(adminPage, {
      name: uniqueName('Org Member'),
      email,
      password: pw,
      active: true,
    });
    await addUserToOrgViaApi(adminPage, user.id, org.id, 'member');

    const memberContext = await browser.newContext();
    const memberPage = await memberContext.newPage();
    await loginViaForm(memberPage, email, pw);

    // Sanity: member can reach the org's data.
    const beforeStatus = await memberPage.evaluate(
      async (id: number) => {
        const r = await fetch(`/api/v1/organizations/${id}`, { credentials: 'same-origin' });
        return r.status;
      },
      org.id
    );
    expect(beforeStatus).toBe(200);

    // Admin soft-deletes the org.
    await deleteOrganizationViaApi(adminPage, org.id);

    // Member's org-scoped request now 404s (org appears not to
    // exist) rather than 403, because the org resolver can't even
    // load the row.
    const afterStatus = await memberPage.evaluate(
      async (id: number) => {
        const r = await fetch(`/api/v1/organizations/${id}`, { credentials: 'same-origin' });
        return r.status;
      },
      org.id
    );
    expect(afterStatus).toBe(404);

    // Cleanup.
    await deleteUserViaApi(adminPage, user.id).catch(() => {});
    await memberContext.close();
    await admin.close();
  });
});
