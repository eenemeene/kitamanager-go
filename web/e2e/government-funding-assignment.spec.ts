import { test, expect } from 'playwright/test'
import {
  login,
  createOrganization,
  assignGovernmentFundingToOrganization,
  SUPERADMIN_EMAIL,
  SUPERADMIN_PASSWORD
} from './utils/test-helpers'

test.describe('Government Funding Assignment', () => {
  // Use a unique timestamp to avoid conflicts between test runs
  const timestamp = Date.now()
  const testOrgName = `Test Org GovFunding ${timestamp}`

  test.beforeEach(async ({ page }) => {
    // Login as superadmin before each test
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)
  })

  test('superadmin can create an organization and assign government funding', async ({ page }) => {
    // Step 1: Create a new organization
    await createOrganization(page, testOrgName)

    // Step 2: Navigate to last page to find the newly created org (pagination)
    const lastPageButton = page.getByRole('button', { name: 'Last Page' })
    if (await lastPageButton.isEnabled()) {
      await lastPageButton.click()
      await page.waitForTimeout(500)
    }

    // Step 3: Verify the organization appears in the table
    await expect(page.getByRole('cell', { name: testOrgName })).toBeVisible({ timeout: 5000 })

    // Step 4: Verify the organization has no government funding assigned (shows "-")
    const orgRow = page.getByRole('row').filter({ hasText: testOrgName })
    await expect(orgRow).toBeVisible()

    // Step 5: Assign the "Berlin" government funding to the organization
    // (Berlin government funding is seeded by default)
    await assignGovernmentFundingToOrganization(page, testOrgName, 'Berlin')

    // Step 6: Verify the organization now shows the assigned government funding
    await expect(orgRow.getByText('Berlin')).toBeVisible({ timeout: 5000 })
  })

  test('superadmin can navigate to government fundings list', async ({ page }) => {
    // Navigate to government fundings via sidebar
    await page.getByRole('link', { name: /government funding/i }).click()
    await expect(page).toHaveURL(/.*government-funding/)

    // Verify the government fundings list is displayed
    await expect(page.getByRole('heading', { name: /government funding/i })).toBeVisible()

    // Verify Berlin government funding is listed (seeded data)
    await expect(page.getByRole('cell', { name: 'Berlin' })).toBeVisible({ timeout: 5000 })
  })

  test('superadmin can view government funding details', async ({ page }) => {
    // Navigate to government fundings
    await page.getByRole('link', { name: /government funding/i }).click()
    await expect(page).toHaveURL(/.*government-funding/)

    // Click on view details for Berlin
    const berlinRow = page.getByRole('row').filter({ hasText: 'Berlin' })
    await berlinRow.getByRole('button', { name: /view details/i }).click()

    // Verify we're on the detail page
    await expect(page).toHaveURL(/.*government-funding.*\d+/)

    // Verify the detail page shows periods (Berlin has multiple periods)
    await expect(page.getByText(/period/i)).toBeVisible({ timeout: 5000 })
  })

  test('superadmin can remove government funding from organization', async ({ page }) => {
    // First create an org and assign funding
    const removeTestOrgName = `Test Org Remove GovFunding ${timestamp}`
    await createOrganization(page, removeTestOrgName)

    // Navigate to last page to find the newly created org (pagination)
    const lastPageButton = page.getByRole('button', { name: 'Last Page' })
    if (await lastPageButton.isEnabled()) {
      await lastPageButton.click()
      await page.waitForTimeout(500)
    }

    await assignGovernmentFundingToOrganization(page, removeTestOrgName, 'Berlin')

    // Verify it's assigned
    const orgRow = page.getByRole('row').filter({ hasText: removeTestOrgName })
    await expect(orgRow.getByText('Berlin')).toBeVisible({ timeout: 5000 })

    // Open the assignment dialog again
    await orgRow.getByRole('button', { name: /assign government funding/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // Clear the selection by hovering over the dropdown container and clicking the clear icon
    // The clear icon appears when hovering over a dropdown that has a value
    const dropdownContainer = page.getByRole('dialog').locator('[data-pc-name="dropdown"]')
    await dropdownContainer.hover()
    await page.waitForTimeout(200)

    // Click the clear icon (the X button that appears on hover)
    const clearIcon = dropdownContainer.locator('[data-pc-section="clearicon"]')
    await clearIcon.click()

    // Save to remove the assignment
    await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

    // Wait for success toast
    await expect(page.getByText(/government funding removed successfully/i)).toBeVisible({
      timeout: 5000
    })

    // Verify the organization no longer shows Berlin (shows "-" instead)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })
    await expect(orgRow.getByText('Berlin')).not.toBeVisible({ timeout: 2000 })
  })
})
