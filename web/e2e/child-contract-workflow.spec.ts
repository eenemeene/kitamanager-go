import { test, expect } from 'playwright/test'
import {
  login,
  selectOrganizationById,
  createOrganization,
  SUPERADMIN_EMAIL,
  SUPERADMIN_PASSWORD
} from './utils/test-helpers'

/**
 * Child Contract Workflow E2E test:
 * Tests the complete contract management workflow including:
 * 1. Creating a child
 * 2. Adding a contract
 * 3. Replacing contract with one that has attributes (ganztags)
 * 4. Adding a historical contract (before existing)
 * 5. Adding a future contract with multiple attributes (ganztags + ndh)
 */
test.describe('Child Contract Workflow', () => {
  const timestamp = Date.now()
  const orgName = `Contract Workflow Org ${timestamp}`
  const childFirstName = 'Workflow'
  const childLastName = `Child ${timestamp}`

  test.setTimeout(180000)

  test('should manage child contracts through full workflow', async ({ page }) => {
    // =====================================
    // Setup: Login and create organization
    // =====================================

    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)
    const orgId = await createOrganization(page, orgName, 'berlin')
    await selectOrganizationById(page, orgId)

    // =====================================
    // Step 1: Create a new child
    // =====================================

    await page.getByRole('link', { name: /children/i }).click()
    await expect(page).toHaveURL(/.*children/)

    await page.getByRole('button', { name: /new child/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // Fill in child details
    await page.getByLabel('First Name').fill(childFirstName)
    await page.getByLabel('Last Name').fill(childLastName)

    // Select gender - wait for dropdown panel to appear
    await page.locator('#gender').click()
    await page.waitForTimeout(300)
    const genderPanel = page.locator('.p-select-overlay, .p-dropdown-panel')
    await expect(genderPanel).toBeVisible({ timeout: 5000 })
    await page.getByRole('option', { name: 'Female' }).click()

    // Set birthdate
    const birthdateInput = page.locator('#birthdate')
    await birthdateInput.click()
    await page.waitForTimeout(300)
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()

    // Save the child
    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // Verify child appears in table
    const childFullName = `${childFirstName} ${childLastName}`
    await expect(page.getByRole('cell', { name: childFullName })).toBeVisible({ timeout: 5000 })

    const childRow = page.getByRole('row').filter({ hasText: childFullName })

    // =====================================
    // Step 2: Add first contract (no attributes)
    // =====================================

    await childRow.locator('button[title="Add Contract"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).toContainText(/new contract/i)

    // No warning should appear (first contract)
    await expect(page.getByText(/this child has an active contract/i)).not.toBeVisible()

    // Set start date to today
    await page.locator('#from').click()
    await page.waitForTimeout(300)
    await page.locator('.p-datepicker-calendar td.p-datepicker-today span').click()

    // Save contract without attributes
    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // Verify no attributes shown (just a dash)
    await expect(childRow.locator('td').nth(4)).toContainText('-')

    // =====================================
    // Step 3: "Update" contract by adding new one with ganztags
    // (This ends the previous contract and creates a new one)
    // =====================================

    await childRow.locator('button[title="Add Contract"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // Warning should appear now
    await expect(page.getByText(/this child has an active contract/i)).toBeVisible({ timeout: 5000 })

    // Checkbox should be visible and checked by default
    const endContractCheckbox = page.locator('#endContract')
    await expect(endContractCheckbox).toBeVisible()
    await expect(endContractCheckbox).toBeChecked()

    // Set start date to tomorrow
    const tomorrow = new Date()
    tomorrow.setDate(tomorrow.getDate() + 1)

    await page.locator('#from').click()
    await expect(page.locator('.p-datepicker-panel')).toBeVisible({ timeout: 5000 })

    // Click tomorrow's date
    const tomorrowDay = tomorrow.getDate().toString()
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .filter({ hasText: new RegExp(`^${tomorrowDay}$`) })
      .first()
      .click()
    await expect(page.locator('.p-datepicker-panel')).not.toBeVisible({ timeout: 5000 })

    // Add ganztags attribute
    const chipsInput = page.locator('#attributes input')
    await chipsInput.fill('ganztags')
    await chipsInput.press('Enter')
    await page.waitForTimeout(300)

    // Save
    await page.getByRole('button', { name: 'Save' }).click()

    // Check for error toast
    const errorToast = page.locator('.p-toast-message-error')
    if (await errorToast.isVisible().catch(() => false)) {
      const errorText = await errorToast.textContent()
      throw new Error(`API error: ${errorText}`)
    }

    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 15000 })

    // =====================================
    // Step 4: Add a historical contract (before existing contracts)
    // =====================================

    await childRow.locator('button[title="Add Contract"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // Set dates to last month (historical contract)
    const lastMonth = new Date()
    lastMonth.setMonth(lastMonth.getMonth() - 1)
    lastMonth.setDate(1)

    const lastMonthEnd = new Date(lastMonth)
    lastMonthEnd.setDate(15)

    // Navigate to last month in datepicker for start date
    await page.locator('#from').click()
    await expect(page.locator('.p-datepicker-panel')).toBeVisible({ timeout: 5000 })

    // Click previous month button using aria-label (PrimeVue 4)
    await page.locator('.p-datepicker-panel').getByRole('button', { name: /prev/i }).click()
    await page.waitForTimeout(300)

    // Click first day of last month
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()
    await expect(page.locator('.p-datepicker-panel')).not.toBeVisible({ timeout: 5000 })

    // Set end date
    await page.locator('#to').click()
    await expect(page.locator('.p-datepicker-panel')).toBeVisible({ timeout: 5000 })

    // Navigate to last month using aria-label
    await page.locator('.p-datepicker-panel').getByRole('button', { name: /prev/i }).click()
    await page.waitForTimeout(300)

    // Click day 15
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .filter({ hasText: /^15$/ })
      .first()
      .click()
    await expect(page.locator('.p-datepicker-panel')).not.toBeVisible({ timeout: 5000 })

    // Don't end current contract (uncheck if visible)
    const endContractCheckbox2 = page.locator('#endContract')
    if (await endContractCheckbox2.isVisible().catch(() => false)) {
      if (await endContractCheckbox2.isChecked()) {
        await endContractCheckbox2.uncheck()
      }
    }

    // Save historical contract (no attributes)
    await page.getByRole('button', { name: 'Save' }).click()

    if (await errorToast.isVisible().catch(() => false)) {
      const errorText = await errorToast.textContent()
      throw new Error(`API error: ${errorText}`)
    }

    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 15000 })

    // =====================================
    // Step 5: Add future contract with ganztags + ndh
    // =====================================

    await childRow.locator('button[title="Add Contract"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // Set start date to next month
    const nextMonth = new Date()
    nextMonth.setMonth(nextMonth.getMonth() + 1)
    nextMonth.setDate(1)

    await page.locator('#from').click()
    await expect(page.locator('.p-datepicker-panel')).toBeVisible({ timeout: 5000 })

    // Click next month button using aria-label (PrimeVue 4)
    await page.locator('.p-datepicker-panel').getByRole('button', { name: /next/i }).click()
    await page.waitForTimeout(300)

    // Click first day
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()
    await expect(page.locator('.p-datepicker-panel')).not.toBeVisible({ timeout: 5000 })

    // End the current contract so this future contract doesn't overlap
    // The current contract (from step 3) starts tomorrow and is open-ended
    // We need to end it so the new contract starting next month doesn't overlap
    const endContractCheckbox3 = page.locator('#endContract')
    if (await endContractCheckbox3.isVisible().catch(() => false)) {
      // Keep it checked (default) to end the current open-ended contract
      await expect(endContractCheckbox3).toBeChecked()
    }

    // Add ganztags attribute
    const chipsInput3 = page.locator('#attributes input')
    await chipsInput3.fill('ganztags')
    await chipsInput3.press('Enter')
    await page.waitForTimeout(200)

    // Add ndh attribute
    await chipsInput3.fill('ndh')
    await chipsInput3.press('Enter')
    await page.waitForTimeout(200)

    // Save
    await page.getByRole('button', { name: 'Save' }).click()

    if (await errorToast.isVisible().catch(() => false)) {
      const errorText = await errorToast.textContent()
      throw new Error(`API error: ${errorText}`)
    }

    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 15000 })

    // =====================================
    // Verify: Check contract history
    // =====================================

    await childRow.locator('button[title="Contract History"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).toContainText(/contract history/i)

    const historyDialog = page.getByRole('dialog')

    // Should have 4 contracts total:
    // 1. Original contract (ended today)
    // 2. Contract with ganztags (starts tomorrow - upcoming)
    // 3. Historical contract (last month - ended)
    // 4. Future contract with ganztags + ndh (next month - upcoming)

    // Verify we can see ganztags in at least 2 contracts
    const ganztagsTags = historyDialog.locator('.p-tag').filter({ hasText: 'ganztags' })
    await expect(ganztagsTags).toHaveCount(2)

    // Verify ndh appears in one contract
    await expect(historyDialog.getByText('ndh')).toBeVisible()

    // Verify different statuses exist
    const activeTag = historyDialog.locator('.p-tag').filter({ hasText: /^Active$/i })
    const upcomingTag = historyDialog.locator('.p-tag').filter({ hasText: /^Upcoming$/i })
    const endedTag = historyDialog.locator('.p-tag').filter({ hasText: /^Ended$/i })

    // Check we have the expected statuses
    // Note: exact counts depend on current date, but we should have some of each type
    const activeCount = await activeTag.count()
    const upcomingCount = await upcomingTag.count()
    const endedCount = await endedTag.count()

    console.log(`Contract statuses - Active: ${activeCount}, Upcoming: ${upcomingCount}, Ended: ${endedCount}`)

    // We expect at least 1 upcoming (tomorrow's or next month's)
    expect(upcomingCount).toBeGreaterThanOrEqual(1)
    // We expect at least 1 ended (historical)
    expect(endedCount).toBeGreaterThanOrEqual(1)

    // Total should be 4 contracts
    const totalContracts = await historyDialog.locator('tbody tr').count()
    expect(totalContracts).toBe(4)

    // Close history dialog
    await historyDialog.locator('button:has-text("Close"):not(.p-dialog-close-button)').click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 5000 })
  })
})
