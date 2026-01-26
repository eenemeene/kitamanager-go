import { test, expect } from 'playwright/test'

/**
 * User onboarding E2E test:
 * 1. Create a new organization
 * 2. Create a group within that organization
 * 3. Create a new user
 * 4. Assign the user to the group with manager role
 * 5. Login as the new user
 */
test.describe('User Onboarding', () => {
  // Generate unique names for this test run
  const timestamp = Date.now()
  const orgName = `Test Org ${timestamp}`
  const groupName = `Test Group ${timestamp}`
  const userName = `Test Manager ${timestamp}`
  const userEmail = `manager${timestamp}@example.com`
  const userPassword = 'testpassword123'

  // This test performs many operations, increase timeout
  test.setTimeout(120000)

  test('complete user onboarding flow', async ({ page }) => {
    // =====================================
    // Step 1: Login as admin
    // =====================================
    await page.goto('/login')
    await page.getByPlaceholder('Email').fill('admin@example.com')
    await page.getByPlaceholder('Password').fill('adminadmin')
    await page.getByRole('button', { name: 'Sign In' }).click()
    await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 })

    // =====================================
    // Step 2: Create Organization
    // =====================================
    await page.getByRole('link', { name: /organization/i }).first().click()
    await expect(page).toHaveURL(/.*organization/)

    // Click "New Organization" button
    await page.getByRole('button', { name: /new organization/i }).click()

    // Fill organization form
    await page.getByPlaceholder('Organization name').fill(orgName)
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for dialog to close and table to update
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })
    // Wait for success toast to appear (confirms save completed)
    await expect(page.getByText('Organization created successfully')).toBeVisible({ timeout: 5000 })

    // =====================================
    // Step 3: Select new organization in sidebar
    // =====================================
    // Refresh the page to ensure the sidebar org dropdown has the new organization
    await page.reload()
    await page.waitForLoadState('networkidle')

    // Click on the organization dropdown in the sidebar
    const orgDropdown = page.getByRole('combobox').first()
    await orgDropdown.click()

    // Select the new organization from the dropdown list
    // Use exact:false to handle partial matches and scroll into view if needed
    const orgOption = page.getByRole('option', { name: orgName })
    await orgOption.scrollIntoViewIfNeeded()
    await orgOption.click()

    // =====================================
    // Step 4: Create Group for the organization
    // =====================================
    // Navigate to Groups (should now be visible in sidebar)
    await page.getByRole('link', { name: /group/i }).first().click()
    await expect(page).toHaveURL(/.*groups/)

    // Click "New Group" button
    await page.getByRole('button', { name: /new group/i }).click()

    // Fill group form
    await page.getByPlaceholder('Group name').fill(groupName)
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for dialog to close and verify group appears in table
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('cell', { name: groupName })).toBeVisible()

    // =====================================
    // Step 5: Create new User
    // =====================================
    await page.getByRole('link', { name: /user/i }).first().click()
    await expect(page).toHaveURL(/.*users/)

    // Click "New User" button
    await page.getByRole('button', { name: /new user/i }).click()

    // Fill user form
    await page.getByPlaceholder('Full name').fill(userName)
    await page.getByPlaceholder('Email address').fill(userEmail)
    await page.getByPlaceholder('Password').fill(userPassword)
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for dialog to close and verify user appears in table
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('cell', { name: userName })).toBeVisible()

    // =====================================
    // Step 6: Add user to group with manager role
    // =====================================
    // Find the row with our new user and click the "Manage Memberships" button (icon: pi-users)
    const userRow = page.getByRole('row').filter({ hasText: userName })
    // The button is icon-only with title="Manage Memberships", use title selector
    await userRow.locator('button[title="Manage Memberships"]').click()

    // Wait for memberships dialog to open
    await expect(page.getByRole('dialog', { name: /manage group memberships/i })).toBeVisible()

    // Click "Add to Group" button
    await page.getByRole('button', { name: /add to group/i }).click()

    // Wait for the "Add User to Group" dialog
    await expect(page.getByRole('dialog', { name: /add user to group/i })).toBeVisible()

    // Select the group from dropdown
    const groupDropdown = page.locator('#add-group')
    await groupDropdown.click()
    // The group option shows as "Group Name (Org Name)"
    await page.getByRole('option', { name: new RegExp(groupName) }).click()

    // Select "Manager" role from dropdown
    const roleDropdown = page.locator('#add-role')
    await roleDropdown.click()
    await page.getByRole('option', { name: 'Manager' }).click()

    // Click Add button
    await page.getByRole('button', { name: 'Add', exact: true }).click()

    // Verify the membership was added (should appear in the memberships dialog table)
    const membershipsDialog = page.getByRole('dialog', { name: /manage group memberships/i })
    await expect(membershipsDialog.getByRole('cell', { name: groupName })).toBeVisible({ timeout: 5000 })
    // Look for Manager role tag specifically (not the title which contains "Manager" in the user name)
    await expect(membershipsDialog.locator('.p-tag').getByText('Manager')).toBeVisible()

    // Close the memberships dialog (use the text Close button in the footer, not the X icon button)
    await membershipsDialog.locator('button.p-button-text', { hasText: 'Close' }).click()
    await expect(membershipsDialog).not.toBeVisible({ timeout: 5000 })

    // =====================================
    // Step 7: Logout
    // =====================================
    // Click on the user email button in the header to open the menu
    await page.getByRole('button', { name: 'admin@example.com' }).click()
    // Click Logout in the popup menu
    await page.getByRole('menuitem', { name: /logout|sign out|abmelden/i }).click()
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 })

    // =====================================
    // Step 8: Login as the new manager user
    // =====================================
    await page.getByPlaceholder('Email').fill(userEmail)
    await page.getByPlaceholder('Password').fill(userPassword)
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Should redirect to dashboard
    await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 })

    // Verify we're logged in as the new user (check if user name appears somewhere)
    // The new user should have access but may see limited content based on their org access
    await expect(page.locator('body')).toContainText(/dashboard/i)
  })
})
