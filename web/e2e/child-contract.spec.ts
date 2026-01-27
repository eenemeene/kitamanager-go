import { test, expect } from 'playwright/test'
import {
  login,
  selectOrganization,
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

    // Create a new organization
    await createOrganization(page, orgName)

    // Select the organization
    await selectOrganization(page, orgName, timestamp.toString())

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

    // Wait for success message and dialog to close
    await expect(page.getByText(/child created successfully/i)).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })

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

    // Wait for success
    await expect(page.getByText(/contract created successfully/i)).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })

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
    await page.waitForTimeout(300)

    // Navigate to next day if needed (click the day after today)
    const tomorrowDay = tomorrow.getDate().toString()
    const calendarDays = page.locator(
      '.p-datepicker-calendar td:not(.p-datepicker-other-month) span'
    )
    await calendarDays
      .filter({ hasText: new RegExp(`^${tomorrowDay}$`) })
      .first()
      .click()

    // Add different attribute for new contract
    const newChipsInput = page.locator('#attributes input')
    await newChipsInput.fill('halbtags')
    await newChipsInput.press('Enter')

    // Save - should end old contract and create new one
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for success message mentioning both operations
    await expect(
      page.getByText(/previous contract ended and new contract created successfully/i)
    ).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })

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

    // First contract should now show as inactive (has end date)
    // The newer contract should be active
    const activeTag = historyDialog.locator('.p-tag').filter({ hasText: /^Active$/i })
    const inactiveTag = historyDialog.locator('.p-tag').filter({ hasText: /^Inactive$/i })

    // Should have one active and one inactive
    await expect(activeTag).toHaveCount(1)
    await expect(inactiveTag).toHaveCount(1)

    // Close history dialog (click the footer Close button, not the header X)
    await historyDialog.locator('button:has-text("Close"):not(.p-dialog-close-button)').click()
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })
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

    // Create org
    await createOrganization(page, orgName2)
    await selectOrganization(page, orgName2, timestamp2.toString())

    // Navigate to children
    await page.getByRole('link', { name: /children/i }).click()

    // Create child
    await page.getByRole('button', { name: /new child/i }).click()
    await page.getByLabel('First Name').fill('Test')
    await page.getByLabel('Last Name').fill(childName2)
    await page.locator('#birthdate').click()
    await page.waitForTimeout(300)
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()
    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.getByText(/child created successfully/i)).toBeVisible({ timeout: 5000 })

    // Add first contract
    const childRow = page.getByRole('row').filter({ hasText: `Test ${childName2}` })
    await childRow.locator('button[title="Add Contract"]').click()
    await page.locator('#from').click()
    await page.waitForTimeout(300)
    await page.locator('.p-datepicker-calendar td.p-datepicker-today span').click()
    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.getByText(/contract created successfully/i)).toBeVisible({ timeout: 5000 })

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
