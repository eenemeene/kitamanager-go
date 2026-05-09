import { test, expect, Page } from '@playwright/test';
import {
  login,
  loginViaForm,
  logoutViaApi,
  createUserViaApi,
  deleteUserViaApi,
  addUserToOrgViaApi,
  createTestOrg,
  deleteTestOrg,
  uniqueName,
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
} from './utils/test-helpers';

// English locale so the sidebar labels match what the assertions look
// for via accessible-name. Assertions also fall back to href patterns
// so most don't actually depend on locale, but pinning here matches
// the rest of the suite.
test.use({ locale: 'en-US' });

// Why this file exists: the four organizational roles (staff, member,
// manager, admin) are the ground truth for what each user should see
// in the UI. The sidebar's `minRole` gates the nav, the Casbin policy
// gates the API. The two are written separately and historically have
// drifted (we just fixed two such drifts on 2026-05-09: staff couldn't
// reach Attendance, manager had funding-bills perms but no nav link).
// This file pins the sidebar half so the next drift trips a test, not
// a user. The Casbin half is pinned by rbac_test.go.

// What each role MUST see (positive) and MUST NOT see (negative).
// Every entry is identified by an href substring that survives both
// translation and the `/organizations/{id}` prefix the org-scoped
// nav prepends to its hrefs in app-sidebar.tsx:188-191.
type Visibility = { hrefMatches: string; visible: boolean };

const STAFF_NAV: Visibility[] = [
  { hrefMatches: '/attendance', visible: true },
  // Everything else is gated at member or higher, so staff sees nothing
  // beyond Attendance. The org selector and global nav still show.
  { hrefMatches: '/dashboard', visible: false },
  { hrefMatches: '/sections', visible: false },
  { hrefMatches: '/children', visible: false },
  { hrefMatches: '/employees', visible: false },
  { hrefMatches: '/government-funding-bills', visible: false },
  { hrefMatches: '/budget-items', visible: false },
  { hrefMatches: '/statistics', visible: false },
  { hrefMatches: '/payplans', visible: false },
  { hrefMatches: '/users', visible: false },
  { hrefMatches: '/audit-logs', visible: false },
  { hrefMatches: '/government-funding-rates', visible: false },
];

const MEMBER_NAV: Visibility[] = [
  { hrefMatches: '/dashboard', visible: true },
  { hrefMatches: '/attendance', visible: true },
  { hrefMatches: '/children', visible: true },
  { hrefMatches: '/sections', visible: false },
  { hrefMatches: '/employees', visible: false },
  { hrefMatches: '/government-funding-bills', visible: false },
  { hrefMatches: '/budget-items', visible: false },
  { hrefMatches: '/statistics', visible: false },
  { hrefMatches: '/payplans', visible: false },
  { hrefMatches: '/users', visible: false },
  { hrefMatches: '/audit-logs', visible: false },
  { hrefMatches: '/government-funding-rates', visible: false },
];

const MANAGER_NAV: Visibility[] = [
  { hrefMatches: '/dashboard', visible: true },
  { hrefMatches: '/attendance', visible: true },
  { hrefMatches: '/sections', visible: true },
  { hrefMatches: '/children', visible: true },
  { hrefMatches: '/employees', visible: true },
  { hrefMatches: '/government-funding-bills', visible: true },
  { hrefMatches: '/budget-items', visible: true },
  { hrefMatches: '/statistics', visible: true },
  { hrefMatches: '/government-funding-rates', visible: true },
  // Settings group stays admin-only.
  { hrefMatches: '/payplans', visible: false },
  { hrefMatches: '/users', visible: false },
  { hrefMatches: '/audit-logs', visible: false },
];

const ADMIN_NAV: Visibility[] = [
  { hrefMatches: '/dashboard', visible: true },
  { hrefMatches: '/attendance', visible: true },
  { hrefMatches: '/sections', visible: true },
  { hrefMatches: '/children', visible: true },
  { hrefMatches: '/employees', visible: true },
  { hrefMatches: '/government-funding-bills', visible: true },
  { hrefMatches: '/budget-items', visible: true },
  { hrefMatches: '/statistics', visible: true },
  { hrefMatches: '/government-funding-rates', visible: true },
  { hrefMatches: '/payplans', visible: true },
  { hrefMatches: '/users', visible: true },
  { hrefMatches: '/audit-logs', visible: true },
];

// Open the mobile sidebar drawer if the viewport is narrow enough that
// the hamburger is shown. Mirrored from navigation.spec.ts so the test
// is robust to different Playwright project widths.
async function ensureSidebarVisible(page: Page) {
  await page.waitForLoadState('load');
  const hamburger = page.getByRole('button', { name: /menu/i });
  if (await hamburger.isVisible().catch(() => false)) {
    await page.waitForTimeout(300);
    await hamburger.click();
    await page.locator('div.fixed.inset-0.z-50').waitFor({ state: 'visible', timeout: 5000 });
  }
}

// Assert exactly the visibility table for the currently logged-in
// user. Uses href substrings so the assertion survives translation
// changes and the `/organizations/{id}` prefix on org-scoped items.
async function assertNavMatches(page: Page, expected: Visibility[]) {
  await ensureSidebarVisible(page);
  // The sidebar renders inside <nav>; org-scoped items only appear
  // once an org is selected, which fetchOrganizations does
  // automatically when the user has at least one org assigned.
  await page.locator('nav').first().waitFor({ state: 'visible', timeout: 10000 });

  for (const { hrefMatches, visible } of expected) {
    const links = page.locator(`nav a[href*="${hrefMatches}"]`);
    if (visible) {
      // Must have at least one matching link in the nav. Some entries
      // (e.g. Statistics) render a parent link plus children, so >=1
      // is the right floor.
      await expect(
        links.first(),
        `expected nav to contain a link matching ${hrefMatches}`
      ).toBeVisible({ timeout: 10000 });
    } else {
      // Must have zero matching links. count() is the right primitive
      // here — a hidden-but-rendered link still indicates a regression.
      await expect(
        links,
        `expected nav to NOT contain any link matching ${hrefMatches}`
      ).toHaveCount(0);
    }
  }
}

// Provision a user with a single role in a single org. Returns the
// IDs so the cleanup hook can tear them down. All work goes through
// the API helpers — no UI interaction in setup, so failures are
// isolated to the actual sidebar assertions.
async function provisionRoleUser(
  page: Page,
  role: 'staff' | 'member' | 'manager' | 'admin',
  orgId: number
): Promise<{ userId: number; email: string; password: string }> {
  const email = `${role}-${Date.now()}@test.local`;
  const password = 'role-test-password-123';
  const user = await createUserViaApi(page, {
    name: uniqueName(role),
    email,
    password,
    active: true,
  });
  await addUserToOrgViaApi(page, user.id, orgId, role);
  return { userId: user.id, email, password };
}

// Each role gets its own test so Playwright's per-test browser
// context isolation prevents the previous role's persisted localStorage
// (selectedOrganizationId, auth tokens) from leaking into the next.
test.describe('Sidebar visibility per role', () => {
  let orgId: number;
  const createdUserIds: number[] = [];

  test.beforeAll(async ({ browser }) => {
    // Use a throwaway page just for the API setup; the per-test pages
    // start empty. createTestOrg + addUserToOrgViaApi need a logged-in
    // superadmin context.
    const ctx = await browser.newContext({ locale: 'en-US' });
    const page = await ctx.newPage();
    await login(page);
    const created = await createTestOrg(page, 'RoleSidebarTest');
    orgId = created.orgId;
    await ctx.close();
  });

  test.afterAll(async ({ browser }) => {
    const ctx = await browser.newContext({ locale: 'en-US' });
    const page = await ctx.newPage();
    await login(page);
    for (const userId of createdUserIds) {
      await deleteUserViaApi(page, userId).catch(() => {});
    }
    await deleteTestOrg(page, orgId).catch(() => {});
    await ctx.close();
  });

  for (const [role, expected] of [
    ['staff', STAFF_NAV],
    ['member', MEMBER_NAV],
    ['manager', MANAGER_NAV],
    ['admin', ADMIN_NAV],
  ] as const) {
    test(`${role} sees the right nav items`, async ({ page }) => {
      // Provision happens as superadmin; we then log out and log in
      // as the freshly-created user before asserting.
      await login(page);
      const { userId, email, password } = await provisionRoleUser(page, role, orgId);
      createdUserIds.push(userId);
      await logoutViaApi(page);

      // Form login (rather than the API helper) because we want the
      // browser to land on the dashboard with a real React render —
      // that's where the sidebar mounts and runs its filter logic.
      await loginViaForm(page, email, password);

      await assertNavMatches(page, expected);
    });
  }

  // Sanity check that confirms superadmin still sees the global-nav
  // entries (Organizations + Government Funding Rates) that no other
  // role gets. Without this, a future "lower minRole on globalNavigation"
  // change could pass the per-role tests above without anyone noticing
  // those items disappeared from superadmin too.
  test('superadmin sees the global-nav entries', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await ensureSidebarVisible(page);
    await expect(
      page.locator('nav a[href="/organizations"]').first()
    ).toBeVisible({ timeout: 10000 });
    await expect(
      page.locator('nav a[href="/government-funding-rates"]').first()
    ).toBeVisible({ timeout: 10000 });
  });
});
