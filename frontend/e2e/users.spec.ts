import { test, expect } from '@playwright/test';
import {
  login,
  loginViaForm,
  createTestOrg,
  deleteTestOrg,
  createUserViaApi,
  deleteUserViaApi,
  getUsersViaApi,
  uniqueName,
  addUserToOrgViaApi,
  logoutViaApi,
} from './utils/test-helpers';

test.use({ locale: 'en-US' });

test.describe('Users', () => {
  let orgId: number;

  test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage();
    await login(page);
    const testOrg = await createTestOrg(page, 'Users');
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
    await page.goto(`/organizations/${orgId}/users`);
    await page.waitForLoadState('load');
  });

  test('should display users list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /user/i }).first()).toBeVisible();
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 10000 });
  });

  test('should create a new user via UI', async ({ page }) => {
    const userName = uniqueName('TestUser');
    const userEmail = `testuser-${Date.now()}@example.com`;

    await page.getByRole('button', { name: /new user/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    await page.getByLabel(/name/i).fill(userName);
    await page.getByLabel(/email/i).fill(userEmail);
    await page.getByLabel(/password/i).fill('testpassword123');

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText(userName)).toBeVisible({ timeout: 10000 });

    const users = await getUsersViaApi(page);
    const created = users.find((u) => u.email === userEmail);
    if (created) {
      await deleteUserViaApi(page, created.id);
    }
  });

  test('should edit a user via UI', async ({ page }) => {
    const origName = uniqueName('EditUser');
    const email = `edituser-${Date.now()}@example.com`;
    const user = await createUserViaApi(page, {
      name: origName,
      email,
      password: 'testpassword123',
    });

    await page.reload();
    await page.waitForLoadState('load');
    await expect(page.getByText(origName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: origName });
    // Action buttons are icon-only; skip the toggle switch (role=switch)
    // Order: edit (pencil), membership (users), delete (trash)
    await row.locator('button:not([role="switch"])').first().click();

    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

    const updatedName = uniqueName('Updated');
    await page.getByLabel(/name/i).clear();
    await page.getByLabel(/name/i).fill(updatedName);

    await page.getByRole('button', { name: /save/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText(updatedName)).toBeVisible({ timeout: 10000 });

    await deleteUserViaApi(page, user.id);
  });

  test('should delete a user via UI', async ({ page }) => {
    const userName = uniqueName('DelUser');
    const email = `deluser-${Date.now()}@example.com`;
    await createUserViaApi(page, {
      name: userName,
      email,
      password: 'testpassword123',
    });

    await page.reload();
    await page.waitForLoadState('load');
    await expect(page.getByText(userName)).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: userName });
    const actionButtons = row.locator('button');
    await actionButtons.last().click();

    await expect(page.getByRole('alertdialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /delete/i }).click();

    await expect(page.getByText(userName)).not.toBeVisible({ timeout: 10000 });
  });

  // Real login flow for a non-superadmin user — mirrors the Go integration
  // coverage (internal/integration/auth_login_flow_test.go) from the browser
  // side. A superadmin provisions a `member`-role user and grants them
  // membership in the test org, then this test signs out and signs back in
  // as that member. If the browser session survives and /me returns the
  // expected identity, the whole auth chain (password hash on create,
  // session cookie on login, RequireAuth middleware, `member` Casbin
  // policies) is working end-to-end from the React app's perspective.
  test('member user with org membership can log in and access protected API', async ({ page }) => {
    const memberName = uniqueName('MemberUser');
    const memberEmail = `member-${Date.now()}@example.com`;
    const memberPassword = 'member-pw-12345';

    // Create the member user (superadmin session is already active from
    // the describe-level beforeEach).
    const member = await createUserViaApi(page, {
      name: memberName,
      email: memberEmail,
      password: memberPassword,
      active: true,
    });
    await addUserToOrgViaApi(page, member.id, orgId, 'member');

    try {
      // Drop the superadmin session; the next request has no auth cookie
      // so RequireAuth must refuse it.
      await logoutViaApi(page);

      const unauth = await page.evaluate(() =>
        fetch('/api/v1/me', { credentials: 'same-origin' }).then((r) => r.status)
      );
      expect(unauth).toBe(401);

      // Sign in as the freshly-created member. The password travels the same
      // bcrypt path as on creation; if the hash were wrong this would 401.
      await login(page, memberEmail, memberPassword);

      // /me must return the member's identity — proof that the session
      // issued by login is being accepted by RequireAuth and resolving to
      // the right user row.
      const me = await page.evaluate(async () => {
        const r = await fetch('/api/v1/me', { credentials: 'same-origin' });
        if (!r.ok) throw new Error(`/me ${r.status}`);
        return (await r.json()) as { email: string; is_superadmin: boolean };
      });
      expect(me.email).toBe(memberEmail);
      expect(me.is_superadmin).toBe(false);

      // Member has read permission on organizations they belong to —
      // verify the org-scoped gate resolves the user_organization row
      // through PermissionService to the `member` Casbin policies.
      const orgStatus = await page.evaluate(
        async (id) =>
          (await fetch(`/api/v1/organizations/${id}`, { credentials: 'same-origin' })).status,
        orgId
      );
      expect(orgStatus).toBe(200);

      // Org-scoped children list is also member-readable.
      const childrenStatus = await page.evaluate(
        async (id) =>
          (
            await fetch(`/api/v1/organizations/${id}/children?limit=1`, {
              credentials: 'same-origin',
            })
          ).status,
        orgId
      );
      expect(childrenStatus).toBe(200);
    } finally {
      // Re-auth as superadmin and clean up the member user. Swallow errors
      // in teardown so a failed assertion above still leaves a tidy DB.
      await login(page);
      await deleteUserViaApi(page, member.id).catch(() => {});
    }
  });

  // UI form login (as opposed to the API-level `login()` helper) for a
  // non-superadmin. Guards the login-page React component + its redirect
  // behaviour for accounts that aren't the seeded admin — a regression
  // there would leave real members stuck on the form even with valid
  // credentials.
  test('member user can sign in through the login form', async ({ page }) => {
    const memberEmail = `formlogin-${Date.now()}@example.com`;
    const memberPassword = 'form-login-pw-12345';
    const member = await createUserViaApi(page, {
      name: uniqueName('FormLoginUser'),
      email: memberEmail,
      password: memberPassword,
      active: true,
    });
    await addUserToOrgViaApi(page, member.id, orgId, 'member');

    try {
      await logoutViaApi(page);
      // loginViaForm navigates to /login and exercises the actual form
      // submit path, then asserts the URL moves off /login.
      await loginViaForm(page, memberEmail, memberPassword);

      // After the form submit the app should let the member land on an
      // authenticated page. Confirm via /me that the session is the
      // member's, not a leftover superadmin session.
      const me = await page.evaluate(async () => {
        const r = await fetch('/api/v1/me', { credentials: 'same-origin' });
        if (!r.ok) throw new Error(`/me ${r.status}`);
        return (await r.json()) as { email: string };
      });
      expect(me.email).toBe(memberEmail);
    } finally {
      await login(page);
      await deleteUserViaApi(page, member.id).catch(() => {});
    }
  });

  // Granting org membership through the UI (membership dialog). The API
  // path is already covered; this pins the dialog workflow an admin
  // actually uses in the browser — clicking the users icon on a row,
  // picking a role, and confirming. A regression here would break the
  // only practical way to onboard a new colleague without hitting the
  // raw API.
  test('superadmin can grant org membership through the UI dialog', async ({ page }) => {
    const candidateEmail = `uimembership-${Date.now()}@example.com`;
    const candidateName = uniqueName('UiMember');
    const candidate = await createUserViaApi(page, {
      name: candidateName,
      email: candidateEmail,
      password: 'ui-membership-pw-12345',
      active: true,
    });

    try {
      // Reload so the newly-created user is in the table.
      await page.reload();
      await page.waitForLoadState('load');
      await expect(page.getByText(candidateName)).toBeVisible({ timeout: 10000 });

      // Middle icon on the row opens the membership dialog. Action buttons
      // are rendered in order: edit, manage-memberships, delete — matching
      // the existing edit/delete tests above.
      const row = page.getByRole('row').filter({ hasText: candidateName });
      const actionButtons = row.locator('button:not([role="switch"])');
      await actionButtons.nth(1).click();

      // Dialog must be visible and default to "no membership in this org"
      // because we never granted one.
      await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });

      // The role select defaults to member; switch it to manager so the
      // assertion below is not a no-op if the default ever changes.
      await page.getByRole('combobox').first().click();
      await page.getByRole('option', { name: /^Manager$/ }).click();

      // Click the "Add to {orgName}" button to commit.
      await page.getByRole('button', { name: /add to /i }).click();

      // Poll the memberships API until the row has been created. This is
      // more robust than asserting UI state, which depends on the dialog
      // re-rendering without flicker.
      await expect
        .poll(
          async () =>
            page.evaluate(async (uid) => {
              const r = await fetch(`/api/v1/users/${uid}/memberships`, {
                credentials: 'same-origin',
              });
              if (!r.ok) return null;
              const data = (await r.json()) as {
                memberships: Array<{ organization_id: number; role: string }>;
              };
              return data.memberships;
            }, candidate.id),
          { timeout: 10000 }
        )
        .toEqual(
          expect.arrayContaining([
            expect.objectContaining({ organization_id: orgId, role: 'manager' }),
          ])
        );
    } finally {
      await deleteUserViaApi(page, candidate.id).catch(() => {});
    }
  });

  // Negative pair for the test above: a user with no org membership must
  // still be refused by org-scoped endpoints even after they log in
  // successfully. Without this, a misconfigured Casbin fallthrough would
  // let an orphan account read every org's data.
  test('member user without org membership is refused on org endpoints', async ({ page }) => {
    const orphanEmail = `orphan-${Date.now()}@example.com`;
    const orphanPassword = 'orphan-pw-54321';
    const orphan = await createUserViaApi(page, {
      name: uniqueName('OrphanUser'),
      email: orphanEmail,
      password: orphanPassword,
      active: true,
    });
    // Intentionally no addUserToOrgViaApi — that's the scenario under test.

    try {
      await logoutViaApi(page);
      await login(page, orphanEmail, orphanPassword);

      // Login succeeded — identity is valid without a role.
      const me = await page.evaluate(async () => {
        const r = await fetch('/api/v1/me', { credentials: 'same-origin' });
        return { status: r.status, body: r.ok ? await r.json() : null };
      });
      expect(me.status).toBe(200);
      expect(me.body?.email).toBe(orphanEmail);

      // Org-scoped reads must fail with 403 — no role in this org.
      const orgStatus = await page.evaluate(
        async (id) =>
          (await fetch(`/api/v1/organizations/${id}`, { credentials: 'same-origin' })).status,
        orgId
      );
      expect(orgStatus).toBe(403);
    } finally {
      await login(page);
      await deleteUserViaApi(page, orphan.id).catch(() => {});
    }
  });
});
