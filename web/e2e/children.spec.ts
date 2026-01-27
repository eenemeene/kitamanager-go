import { test, expect } from 'playwright/test'
import {
  login,
  selectOrganization,
  SUPERADMIN_EMAIL,
  SUPERADMIN_PASSWORD
} from './utils/test-helpers'

test.describe('Children', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)
  })

  test('should display all seeded children (pagination test)', async ({ page }) => {
    // Select the test organization
    await selectOrganization(page, 'Kita Sonnenschein', 'Sonnenschein')

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

    // The seeded test data should have 50 children
    expect(expectedCount).toBeGreaterThanOrEqual(50)

    // Check that the DataTable shows the correct total in the paginator
    // PrimeVue DataTable shows "{first} - {last} of {totalRecords}"
    const paginatorInfo = page.locator('.p-paginator-current')
    if (await paginatorInfo.isVisible({ timeout: 2000 }).catch(() => false)) {
      const text = await paginatorInfo.textContent()
      // Extract the total from text like "1 - 10 of 50"
      const match = text?.match(/of\s+(\d+)/)
      if (match) {
        const displayedTotal = parseInt(match[1], 10)
        expect(displayedTotal).toBe(expectedCount)
      }
    }

    // Alternatively, count the rows visible with maximum pagination
    // Set rows per page to maximum if possible
    const rowsPerPageDropdown = page.locator('.p-paginator-rpp-options')
    if (await rowsPerPageDropdown.isVisible({ timeout: 2000 }).catch(() => false)) {
      await rowsPerPageDropdown.click()
      // Select the largest option (usually 50 or 100)
      const options = page.locator('.p-dropdown-item')
      const lastOption = options.last()
      if (await lastOption.isVisible({ timeout: 1000 }).catch(() => false)) {
        await lastOption.click()
        await page.waitForTimeout(500)
      }
    }

    // Count rows in the table body
    const rows = page.locator('table tbody tr')
    const rowCount = await rows.count()

    // We should see at least 50 children (or all children if less than page size)
    // The key assertion: we shouldn't be limited by a low default API limit
    expect(rowCount).toBeGreaterThanOrEqual(Math.min(expectedCount, 50))
  })
})
