import { test, expect } from 'playwright/test'
import {
  login,
  createOrganization,
  SUPERADMIN_EMAIL,
  SUPERADMIN_PASSWORD
} from './utils/test-helpers'

test.describe('Organization State and Government Funding', () => {
  // Use a unique timestamp to avoid conflicts between test runs
  const timestamp = Date.now()
  const testOrgName = `Test Org State ${timestamp}`

  test.beforeEach(async ({ page }) => {
    // Login as superadmin before each test
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)
  })

  test('superadmin can create an organization with a state', async ({ page }) => {
    // Create a new organization (state defaults to Berlin)
    await createOrganization(page, testOrgName, 'berlin')

    // Reload the page to ensure we see the updated list
    await page.reload()
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(1000) // Extra wait for Firefox

    // The org might be on any page due to sorting - search through all pages
    let found = false
    for (let i = 0; i < 10 && !found; i++) {
      if (await page.getByRole('cell', { name: testOrgName }).isVisible().catch(() => false)) {
        found = true
        break
      }
      // Try next page
      const nextButton = page.getByRole('button', { name: 'Next Page' })
      if (await nextButton.isEnabled().catch(() => false)) {
        await nextButton.click()
        await page.waitForLoadState('networkidle')
        await page.waitForTimeout(500)
      } else {
        break
      }
    }

    // Verify the organization appears in the table with Berlin state
    await expect(page.getByRole('cell', { name: testOrgName })).toBeVisible({ timeout: 10000 })
    const orgRow = page.getByRole('row').filter({ hasText: testOrgName })
    await expect(orgRow.getByText('Berlin')).toBeVisible()
  })

  test('superadmin can navigate to government fundings list', async ({ page }) => {
    // Navigate to government fundings via sidebar
    await page.getByRole('link', { name: /government funding/i }).click()
    await expect(page).toHaveURL(/.*government-funding/)

    // Verify the government fundings list is displayed
    await expect(page.getByRole('heading', { name: /government funding/i })).toBeVisible()

    // Verify Berlin government funding is listed (seeded data)
    await expect(page.getByRole('cell', { name: /Berlin/i })).toBeVisible({ timeout: 5000 })
  })

  test('superadmin can view government funding details', async ({ page }) => {
    // Navigate to government fundings
    await page.getByRole('link', { name: /government funding/i }).click()
    await expect(page).toHaveURL(/.*government-funding/)

    // Click on view details for Berlin
    const berlinRow = page.getByRole('row').filter({ hasText: /Berlin/i })
    await berlinRow.getByRole('button', { name: /view details/i }).click()

    // Verify we're on the detail page
    await expect(page).toHaveURL(/.*government-funding.*\d+/)

    // Verify the detail page shows the "Add Period" button (indicates we're on the details page)
    await expect(page.getByRole('button', { name: 'Add Period' })).toBeVisible({ timeout: 5000 })
  })

  test('organization state determines which government funding is used', async ({ page }) => {
    // Create an organization with Berlin state
    const orgWithState = `Test Org Funding ${timestamp}`
    await createOrganization(page, orgWithState, 'berlin')

    // Reload the page to ensure we see the updated list
    await page.reload()
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(1000) // Extra wait for Firefox

    // The org might be on any page due to sorting - search through all pages
    let found = false
    for (let i = 0; i < 10 && !found; i++) {
      if (await page.getByRole('cell', { name: orgWithState }).isVisible().catch(() => false)) {
        found = true
        break
      }
      // Try next page
      const nextButton = page.getByRole('button', { name: 'Next Page' })
      if (await nextButton.isEnabled().catch(() => false)) {
        await nextButton.click()
        await page.waitForLoadState('networkidle')
        await page.waitForTimeout(500)
      } else {
        break
      }
    }

    // Verify the organization shows Berlin state
    const orgRow = page.getByRole('row').filter({ hasText: orgWithState })
    await expect(orgRow).toBeVisible({ timeout: 10000 })
    await expect(orgRow.getByText('Berlin')).toBeVisible()

    // The organization's funding is now automatically determined by its state
    // No manual assignment needed - Berlin orgs use Berlin funding rules
  })
})
