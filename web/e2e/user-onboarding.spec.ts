import { test, expect } from 'playwright/test'
import {
  login,
  logout,
  selectOrganization,
  createOrganization,
  createGroup,
  createUser,
  getApiToken,
  getUserByEmail,
  getOrganizationByName,
  getGroupByName,
  addUserToGroup,
  SUPERADMIN_EMAIL,
  SUPERADMIN_PASSWORD
} from './utils/test-helpers'

/**
 * User onboarding E2E test:
 * 1. Superadmin creates a new organization
 * 2. Superadmin creates a group within that organization
 * 3. Superadmin creates an admin user and assigns them to the group
 * 4. Admin user logs in and creates a manager user
 * 5. Admin user creates a member (viewer) user
 */
test.describe('User Onboarding', () => {
  // Generate unique names for this test run
  const timestamp = Date.now()
  const orgName = `Test Org ${timestamp}`
  const groupName = `Test Group ${timestamp}`

  // Admin user (created by superadmin)
  const adminUserName = `Test Admin ${timestamp}`
  const adminUserEmail = `admin${timestamp}@example.com`
  const adminUserPassword = 'testpassword123'

  // Manager user (created by admin)
  const managerUserName = `Test Manager ${timestamp}`
  const managerUserEmail = `manager${timestamp}@example.com`
  const managerUserPassword = 'testpassword123'

  // Member user (created by admin)
  const memberUserName = `Test Member ${timestamp}`
  const memberUserEmail = `member${timestamp}@example.com`
  const memberUserPassword = 'testpassword123'

  // This test performs many operations, increase timeout
  test.setTimeout(180000)

  test('complete user onboarding flow with role hierarchy', async ({ page }) => {
    // =====================================
    // PART 1: Superadmin sets up org and admin user
    // =====================================

    // Step 1: Login as superadmin
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)

    // Get superadmin token for API operations
    const superadminToken = await getApiToken(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)

    // Step 2: Create Organization
    await createOrganization(page, orgName, 'berlin')

    // Step 3: Select new organization in sidebar
    await selectOrganization(page, orgName, timestamp.toString())

    // Step 4: Create Group for the organization
    await createGroup(page, groupName)

    // Step 5: Create admin user
    await createUser(page, adminUserName, adminUserEmail, adminUserPassword)

    // Step 6: Add admin user to group with admin role via API
    const adminUser = await getUserByEmail(page, superadminToken, adminUserEmail)
    const org = await getOrganizationByName(page, superadminToken, orgName)
    const group = await getGroupByName(page, superadminToken, org.id, groupName)
    await addUserToGroup(page, superadminToken, adminUser.id, group.id, 'admin')

    // Verify admin user appears in table
    await page.reload()
    await page.waitForLoadState('networkidle')
    await page.getByRole('link', { name: /user/i }).first().click()
    await expect(page.getByRole('cell', { name: adminUserName })).toBeVisible({ timeout: 10000 })

    // Step 7: Logout superadmin
    await logout(page, SUPERADMIN_EMAIL)

    // =====================================
    // PART 2: Admin user creates manager and member users
    // =====================================

    // Step 8: Login as admin user
    await login(page, adminUserEmail, adminUserPassword)

    // Step 9: Select the organization
    await selectOrganization(page, orgName, timestamp.toString())

    // Step 10: Admin creates manager user
    await createUser(page, managerUserName, managerUserEmail, managerUserPassword)

    // Step 11: Add manager user to group with manager role via API
    // Note: Use superadmin token because admin can't see users not in their org yet
    const managerUser = await getUserByEmail(page, superadminToken, managerUserEmail)
    await addUserToGroup(page, superadminToken, managerUser.id, group.id, 'manager')

    // Step 12: Admin creates member user
    await createUser(page, memberUserName, memberUserEmail, memberUserPassword)

    // Step 13: Add member user to group with member role via API
    const memberUser = await getUserByEmail(page, superadminToken, memberUserEmail)
    await addUserToGroup(page, superadminToken, memberUser.id, group.id, 'member')

    // Verify all users appear in table
    await page.reload()
    await page.waitForLoadState('networkidle')
    await page.getByRole('link', { name: /user/i }).first().click()

    await expect(page.getByRole('cell', { name: adminUserName })).toBeVisible({ timeout: 10000 })
    await expect(page.getByRole('cell', { name: managerUserName })).toBeVisible({ timeout: 10000 })
    await expect(page.getByRole('cell', { name: memberUserName })).toBeVisible({ timeout: 10000 })

    // Step 14: Logout admin
    await logout(page, adminUserEmail)

    // =====================================
    // PART 3: Verify manager and member can login
    // =====================================

    // Step 15: Login as manager user
    await login(page, managerUserEmail, managerUserPassword)
    await expect(page.locator('body')).toContainText(/dashboard/i)
    await logout(page, managerUserEmail)

    // Step 16: Login as member user
    await login(page, memberUserEmail, memberUserPassword)
    await expect(page.locator('body')).toContainText(/dashboard/i)
  })
})
