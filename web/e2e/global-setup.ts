import { chromium } from 'playwright/test'
import { SUPERADMIN_EMAIL, SUPERADMIN_PASSWORD, cleanupTestOrganizations } from './utils/test-helpers'

/**
 * Global setup for E2E tests.
 * Runs once before all tests to ensure a clean database state.
 */
async function globalSetup() {
  const browser = await chromium.launch()
  const page = await browser.newPage()

  try {
    // Navigate to login page
    await page.goto('http://localhost:8080/login')

    // Login as superadmin
    await page.getByPlaceholder('Email').fill(SUPERADMIN_EMAIL)
    await page.getByPlaceholder('Password').fill(SUPERADMIN_PASSWORD)
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Wait for login to complete
    await page.waitForURL(/.*(?<!login)$/, { timeout: 10000 })

    // Get auth token
    const token = await page.evaluate(() => localStorage.getItem('token'))
    if (!token) {
      console.log('Warning: Could not get auth token for cleanup')
      return
    }

    // Clean up old test organizations
    console.log('Cleaning up old test organizations...')
    await cleanupTestOrganizations(page, token)
    console.log('Cleanup complete')
  } catch (error) {
    console.log('Warning: Global setup cleanup failed:', error)
    // Don't fail tests if cleanup fails
  } finally {
    await browser.close()
  }
}

export default globalSetup
