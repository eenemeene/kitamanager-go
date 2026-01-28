import { test, expect } from 'playwright/test'
import {
  login,
  selectOrganizationById,
  createOrganization,
  SUPERADMIN_EMAIL,
  SUPERADMIN_PASSWORD
} from './utils/test-helpers'

/**
 * Child Contract E2E test:
 * Tests the smart contract creation flow that auto-ends existing contracts.
 */
test.describe('Child Contract Management', () => {
  // Generate unique names for this test run
  const timestamp = Date.now()
  const orgName = `Contract Test Org ${timestamp}`
  const childFirstName = 'Test'
  const childLastName = `Child ${timestamp}`

  // Increase timeout for this test
  test.setTimeout(120000)

  test('should show warning when creating contract for child with active contract', async ({
    page
  }) => {
    // =====================================
    // Setup: Create organization and child
    // =====================================

    // Login as superadmin
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)

    // Create a new organization - returns the org ID
    const orgId = await createOrganization(page, orgName, 'berlin')

    // Select the organization by ID (more reliable than dropdown)
    await selectOrganizationById(page, orgId)

    // =====================================
    // Step 1: Navigate to Children and create a child
    // =====================================

    await page.getByRole('link', { name: /children/i }).click()
    await expect(page).toHaveURL(/.*children/)

    // Click New Child button
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
    await page.getByRole('option', { name: 'Female', exact: true }).click()

    // Set birthdate - click the calendar icon and select a date
    const birthdateInput = page.locator('#birthdate')
    await birthdateInput.click()
    await page.waitForTimeout(300)

    // Select a date from the calendar (first day of current month)
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()

    // Save the child
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for dialog to close (confirms success)
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // Verify child appears in table
    const childFullName = `${childFirstName} ${childLastName}`
    await expect(page.getByRole('cell', { name: childFullName })).toBeVisible({ timeout: 5000 })

    // =====================================
    // Step 2: Add first contract to the child
    // =====================================

    // Find the row with our child and click the Add Contract button
    const childRow = page.getByRole('row').filter({ hasText: childFullName })
    await childRow.locator('button[title="Add Contract"]').click()

    // Wait for dialog
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).toContainText(/new contract/i)

    // There should be NO warning message since this is the first contract
    await expect(page.getByText(/this child has an active contract/i)).not.toBeVisible()

    // Set start date (should default to today, but let's be explicit)
    const fromDateInput = page.locator('#from')
    await fromDateInput.click()
    await page.waitForTimeout(300)
    // Click today
    await page.locator('.p-datepicker-calendar td.p-datepicker-today span').click()

    // Add an attribute
    const chipsInput = page.locator('#attributes input')
    await chipsInput.fill('ganztags')
    await chipsInput.press('Enter')

    // Save the contract
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for dialog to close (confirms success)
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // Verify the attribute shows in the table
    await expect(childRow.getByText('ganztags')).toBeVisible({ timeout: 5000 })

    // =====================================
    // Step 3: Add second contract - should show warning
    // =====================================

    // Click Add Contract again
    await childRow.locator('button[title="Add Contract"]').click()

    // Wait for dialog
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // NOW there should be a warning message
    await expect(page.getByText(/this child has an active contract/i)).toBeVisible({
      timeout: 5000
    })
    // Shows current contract attributes in the warning message
    await expect(page.getByText(/active since.*ganztags/i)).toBeVisible()

    // The checkbox should be visible and checked by default
    const endContractCheckbox = page.locator('#endContract')
    await expect(endContractCheckbox).toBeVisible()
    await expect(endContractCheckbox).toBeChecked()

    // Set a future start date for the new contract (tomorrow)
    const tomorrow = new Date()
    tomorrow.setDate(tomorrow.getDate() + 1)

    await page.locator('#from').click()
    // Wait for calendar panel to appear
    await expect(page.locator('.p-datepicker-panel')).toBeVisible({ timeout: 5000 })

    // Navigate to next day if needed (click the day after today)
    const tomorrowDay = tomorrow.getDate().toString()
    const calendarDays = page.locator(
      '.p-datepicker-calendar td:not(.p-datepicker-other-month) span'
    )
    await calendarDays
      .filter({ hasText: new RegExp(`^${tomorrowDay}$`) })
      .first()
      .click()

    // Wait for calendar to close
    await expect(page.locator('.p-datepicker-panel')).not.toBeVisible({ timeout: 5000 })

    // Add different attribute for new contract
    const newChipsInput = page.locator('#attributes input')
    await newChipsInput.fill('halbtags')
    await newChipsInput.press('Enter')
    await page.waitForTimeout(300) // Wait for chip to register

    // Save - should end old contract and create new one
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for either dialog to close (success) or error toast
    // First wait a moment for the network request
    await page.waitForTimeout(500)

    // Check for error toast - if present, fail with the error message
    const errorToast = page.locator('.p-toast-message-error')
    if (await errorToast.isVisible().catch(() => false)) {
      const errorText = await errorToast.textContent()
      throw new Error(`API error occurred: ${errorText}`)
    }

    // Wait for dialog to close (indicates save completed successfully)
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 15000 })

    // =====================================
    // Step 4: Verify contract history
    // =====================================

    // Click the history button
    await childRow.locator('button[title="Contract History"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).toContainText(/contract history/i)

    // Should see both contracts in the history dialog
    const historyDialog = page.getByRole('dialog')
    await expect(historyDialog.getByText('ganztags')).toBeVisible()
    await expect(historyDialog.getByText('halbtags')).toBeVisible()

    // Verify we have exactly 2 contracts
    const contractRows = historyDialog.locator('tbody tr')
    await expect(contractRows).toHaveCount(2)

    // One contract should be Active (the newer one) and one should be Ended or Upcoming
    // The exact status depends on timing/timezone when the test runs
    const activeTag = historyDialog.locator('.p-tag').filter({ hasText: /^Active$/i })
    await expect(activeTag).toHaveCount(1)

    // The other contract should be either Ended (if new contract already started) or Upcoming
    const endedOrUpcomingTag = historyDialog.locator('.p-tag').filter({ hasText: /^(Ended|Upcoming)$/i })
    await expect(endedOrUpcomingTag).toHaveCount(1)

    // Close history dialog (click the footer Close button, not the header X)
    await historyDialog.locator('button:has-text("Close"):not(.p-dialog-close-button)').click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 5000 })
  })

  test('should allow creating contract without ending existing when unchecked', async ({
    page
  }) => {
    // This test verifies that unchecking the box creates overlapping contracts
    // (which should fail due to validation)

    const timestamp2 = Date.now()
    const orgName2 = `Contract Test Org 2 ${timestamp2}`
    const childName2 = `Child ${timestamp2}`

    // Login
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)

    // Create org and select it
    const orgId2 = await createOrganization(page, orgName2, 'berlin')
    await selectOrganizationById(page, orgId2)

    // Navigate to children
    await page.getByRole('link', { name: /children/i }).click()

    // Create child
    await page.getByRole('button', { name: /new child/i }).click()
    await page.getByLabel('First Name').fill('Test')
    await page.getByLabel('Last Name').fill(childName2)
    // Select gender - wait for dropdown panel to appear
    await page.locator('#gender').click()
    await page.waitForTimeout(300)
    const genderPanel2 = page.locator('.p-select-overlay, .p-dropdown-panel')
    await expect(genderPanel2).toBeVisible({ timeout: 5000 })
    await page.getByRole('option', { name: 'Male', exact: true }).click()
    await page.locator('#birthdate').click()
    await page.waitForTimeout(300)
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()
    await page.getByRole('button', { name: 'Save' }).click()
    // Wait for dialog to close (confirms child created)
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // Add first contract
    const childRow = page.getByRole('row').filter({ hasText: `Test ${childName2}` })
    await childRow.locator('button[title="Add Contract"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })
    await page.locator('#from').click()
    await page.waitForTimeout(300)
    await page.locator('.p-datepicker-calendar td.p-datepicker-today span').click()
    await page.getByRole('button', { name: 'Save' }).click()
    // Wait for dialog to close (confirms contract created)
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // Try to add overlapping contract with checkbox unchecked
    await childRow.locator('button[title="Add Contract"]').click()
    await expect(page.getByText(/this child has an active contract/i)).toBeVisible({
      timeout: 5000
    })

    // Uncheck the "end current contract" checkbox
    const endContractCheckbox = page.locator('#endContract')
    await endContractCheckbox.uncheck()

    // Set start date to today (overlapping with existing)
    await page.locator('#from').click()
    await page.waitForTimeout(300)
    await page.locator('.p-datepicker-calendar td.p-datepicker-today span').click()

    // Try to save - should fail with overlap error
    await page.getByRole('button', { name: 'Save' }).click()

    // Should show error about overlap
    await expect(page.getByText(/overlap/i)).toBeVisible({ timeout: 5000 })
  })
})
