import { test, expect } from 'playwright/test'
import {
  login,
  selectOrganization,
  SUPERADMIN_EMAIL,
  SUPERADMIN_PASSWORD
} from './utils/test-helpers'

test.describe('Children', () => {
  // Increase timeout for tests that depend on seed data
  test.setTimeout(60000)

  test.beforeEach(async ({ page }) => {
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)
  })

  test('should display all seeded children (pagination test)', async ({ page }) => {
    // Wait for dashboard to fully load after login
    await page.waitForLoadState('networkidle')

    // Select the test organization - use longer filter text for specificity
    await selectOrganization(page, 'Kita Sonnenschein', 'Kita Sonnenschein')

    // Navigate to children
    await page.getByRole('link', { name: /child/i }).first().click()
    await expect(page).toHaveURL(/.*children/)

    // Wait for the table to load
    await page.waitForLoadState('networkidle')

    // Get the total count from the API to know how many children we expect
    const token = await page.evaluate(() => localStorage.getItem('token'))
    const orgId = await page.evaluate(async (token) => {
      const res = await fetch('/api/v1/organizations?limit=100', {
        headers: { Authorization: `Bearer ${token}` }
      })
      const data = await res.json()
      const org = data.data.find((o: { name: string }) => o.name === 'Kita Sonnenschein')
      return org?.id
    }, token)

    const expectedCount = await page.evaluate(
      async ({ token, orgId }) => {
        const res = await fetch(`/api/v1/organizations/${orgId}/children?limit=1`, {
          headers: { Authorization: `Bearer ${token}` }
        })
        const data = await res.json()
        return data.total
      },
      { token, orgId }
    )

    // The seeded test data should have at least 50 children
    expect(expectedCount).toBeGreaterThanOrEqual(50)

    // Verify the paginator has multiple pages (indicating more than 10 children)
    // With 50+ children and 10 per page, we should have at least 5 page buttons
    const lastPageButton = page.getByRole('button', { name: 'Last Page' })
    await expect(lastPageButton).toBeEnabled({ timeout: 5000 })

    // The table should show 10 rows (default page size)
    const rows = page.locator('table tbody tr')
    const rowCount = await rows.count()
    expect(rowCount).toBe(10)

    // Verify we can navigate to the last page, confirming all data is accessible
    await lastPageButton.click()
    await page.waitForTimeout(500)

    // After clicking last page, the Next/Last buttons should be disabled
    await expect(page.getByRole('button', { name: 'Next Page' })).toBeDisabled()
  })
})
