import { test, expect } from 'playwright/test'
import {
  login,
  selectOrganizationById,
  createOrganization,
  SUPERADMIN_EMAIL,
  SUPERADMIN_PASSWORD
} from './utils/test-helpers'

/**
 * Employee Contract E2E tests:
 * Tests the contract creation and history viewing for employees.
 */
test.describe('Employee Contract Management', () => {
  // Generate unique names for this test run
  const timestamp = Date.now()
  const orgName = `Employee Contract Test Org ${timestamp}`
  const employeeFirstName = 'Test'
  const employeeLastName = `Employee ${timestamp}`

  // Increase timeout for this test
  test.setTimeout(120000)

  test('should create employee contract and view history', async ({ page }) => {
    // =====================================
    // Setup: Create organization and employee
    // =====================================

    // Login as superadmin
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)

    // Create a new organization - returns the org ID
    const orgId = await createOrganization(page, orgName, 'berlin')

    // Select the organization by ID (more reliable than dropdown)
    await selectOrganizationById(page, orgId)

    // =====================================
    // Step 1: Navigate to Employees and create an employee
    // =====================================

    await page.getByRole('link', { name: /employees/i }).click()
    await expect(page).toHaveURL(/.*employees/)

    // Click New Employee button
    await page.getByRole('button', { name: /new employee/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // Fill in employee details
    await page.getByLabel('First Name').fill(employeeFirstName)
    await page.getByLabel('Last Name').fill(employeeLastName)

    // Select gender - wait for dropdown panel to appear
    await page.locator('#gender').click()
    await page.waitForTimeout(300)
    const genderPanel = page.locator('.p-select-overlay, .p-dropdown-panel')
    await expect(genderPanel).toBeVisible({ timeout: 5000 })
    await page.getByRole('option', { name: 'Male', exact: true }).click()

    // Set birthdate - click the calendar icon and select a date
    const birthdateInput = page.locator('#birthdate')
    await birthdateInput.click()
    await page.waitForTimeout(300)

    // Select a date from the calendar (first day of current month)
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()

    // Save the employee
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for dialog to close (confirms success)
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // Verify employee appears in table
    const employeeFullName = `${employeeFirstName} ${employeeLastName}`
    await expect(page.getByRole('cell', { name: employeeFullName })).toBeVisible({ timeout: 5000 })

    // =====================================
    // Step 2: Add first contract to the employee
    // =====================================

    // Find the row with our employee and click the Add Contract button
    const employeeRow = page.getByRole('row').filter({ hasText: employeeFullName })
    await employeeRow.locator('button[title="Add Contract"]').click()

    // Wait for dialog
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).toContainText(/new contract/i)

    // Set start date
    const fromDateInput = page.locator('#from')
    await fromDateInput.click()
    await page.waitForTimeout(300)
    // Click today
    await page.locator('.p-datepicker-calendar td.p-datepicker-today span').click()

    // Fill in contract details
    await page.getByLabel('Position').fill('Erzieher')
    await page.getByLabel('Grade').fill('S8a')
    // InputNumber requires special handling - clear and type
    await page.locator('#step input').fill('3')
    await page.locator('#weekly_hours input').fill('40')

    // Save the contract
    await page.getByRole('button', { name: 'Save' }).click()

    // Wait for dialog to close (confirms success)
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // Verify contract data shows in the table - use exact cell matching
    await expect(employeeRow.getByRole('cell', { name: 'Erzieher' })).toBeVisible({ timeout: 5000 })
    await expect(employeeRow.getByRole('cell', { name: 'S8a' })).toBeVisible()
    await expect(employeeRow.getByRole('cell', { name: '3', exact: true })).toBeVisible()
    await expect(employeeRow.getByRole('cell', { name: '40', exact: true })).toBeVisible()

    // =====================================
    // Step 3: View contract history
    // =====================================

    // Click the history button
    await employeeRow.locator('button[title="Contract History"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('dialog')).toContainText(/contract history/i)

    // Should see the contract in the history dialog
    const historyDialog = page.getByRole('dialog')
    await expect(historyDialog.getByText('Erzieher')).toBeVisible()
    await expect(historyDialog.getByText('S8a')).toBeVisible()

    // Verify we have exactly 1 contract
    const contractRows = historyDialog.locator('tbody tr')
    await expect(contractRows).toHaveCount(1)

    // The contract should be Active
    const activeTag = historyDialog.locator('.p-tag').filter({ hasText: /^Active$/i })
    await expect(activeTag).toHaveCount(1)

    // Close history dialog (click the footer Close button)
    await historyDialog.locator('button:has-text("Close"):not(.p-dialog-close-button)').click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 5000 })
  })

  test('should show multiple contracts in history', async ({ page }) => {
    const timestamp2 = Date.now()
    const orgName2 = `Employee Multi-Contract Org ${timestamp2}`
    const employeeName2 = `Employee ${timestamp2}`

    // Login
    await login(page, SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD)

    // Create org and select it
    const orgId2 = await createOrganization(page, orgName2, 'berlin')
    await selectOrganizationById(page, orgId2)

    // Navigate to employees
    await page.getByRole('link', { name: /employees/i }).click()

    // Create employee
    await page.getByRole('button', { name: /new employee/i }).click()

    await page.getByLabel('First Name').fill('Test')
    await page.getByLabel('Last Name').fill(employeeName2)

    await page.locator('#gender').click()
    await page.waitForTimeout(300)
    await page.getByRole('option', { name: 'Female', exact: true }).click()

    await page.locator('#birthdate').click()
    await page.waitForTimeout(300)
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()

    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    const employeeRow = page.getByRole('row').filter({ hasText: `Test ${employeeName2}` })

    // =====================================
    // Add first contract (current, with end date to avoid overlap)
    // =====================================
    await employeeRow.locator('button[title="Add Contract"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // Set start date to today
    await page.locator('#from').click()
    await page.waitForTimeout(300)
    await page.locator('.p-datepicker-calendar td.p-datepicker-today span').click()

    // Set end date to last day of current month (to avoid overlap with next month's contract)
    await page.locator('#to').click()
    await page.waitForTimeout(300)
    // Select the last day of the current month
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .last()
      .click()

    await page.getByLabel('Position').fill('Kinderpfleger')
    await page.getByLabel('Grade').fill('S4')
    await page.locator('#step input').fill('1')
    await page.locator('#weekly_hours input').fill('30')

    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // =====================================
    // Add second contract (future - starts next month)
    // =====================================
    await employeeRow.locator('button[title="Add Contract"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    // Set start date to next month
    await page.locator('#from').click()
    await page.waitForTimeout(300)
    await page.locator('.p-datepicker-next-button').click()
    await page.waitForTimeout(300)
    await page
      .locator('.p-datepicker-calendar td:not(.p-datepicker-other-month) span')
      .first()
      .click()

    await page.getByLabel('Position').fill('Erzieher')
    await page.getByLabel('Grade').fill('S8a')
    await page.locator('#step input').fill('2')
    await page.locator('#weekly_hours input').fill('40')

    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

    // =====================================
    // Verify contract history shows both
    // =====================================
    await employeeRow.locator('button[title="Contract History"]').click()
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

    const historyDialog = page.getByRole('dialog')

    // Should have 2 contracts
    const contractRows = historyDialog.locator('tbody tr')
    await expect(contractRows).toHaveCount(2)

    // Verify different positions are shown
    await expect(historyDialog.getByText('Kinderpfleger')).toBeVisible()
    await expect(historyDialog.getByText('Erzieher')).toBeVisible()

    // Should have one Active and one Upcoming
    const activeTag = historyDialog.locator('.p-tag').filter({ hasText: /^Active$/i })
    const upcomingTag = historyDialog.locator('.p-tag').filter({ hasText: /^Upcoming$/i })

    await expect(activeTag).toHaveCount(1)
    await expect(upcomingTag).toHaveCount(1)

    // Close history dialog
    await historyDialog.locator('button:has-text("Close"):not(.p-dialog-close-button)').click()
    await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 5000 })
  })
})
