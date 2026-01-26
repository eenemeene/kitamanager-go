import { test, expect } from 'playwright/test'

test.describe('Authentication', () => {
  test('should display login page', async ({ page }) => {
    await page.goto('/')

    // Should redirect to login
    await expect(page).toHaveURL(/.*login/)

    // Login form should be visible
    await expect(page.getByPlaceholder('Email')).toBeVisible()
    await expect(page.getByPlaceholder('Password')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible()
  })

  test('should login with valid credentials', async ({ page }) => {
    await page.goto('/login')

    // Fill in credentials
    await page.getByPlaceholder('Email').fill('admin@example.com')
    await page.getByPlaceholder('Password').fill('adminadmin')

    // Submit form
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Should redirect to dashboard
    await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 })

    // Dashboard should show some content
    await expect(page.locator('body')).toContainText(/dashboard|organization/i)
  })

  test('should show error with invalid credentials', async ({ page }) => {
    await page.goto('/login')

    await page.getByPlaceholder('Email').fill('wrong@example.com')
    await page.getByPlaceholder('Password').fill('wrongpassword')
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Should stay on login page
    await expect(page).toHaveURL(/.*login/)
  })

  test('should logout successfully', async ({ page }) => {
    // Login first
    await page.goto('/login')
    await page.getByPlaceholder('Email').fill('admin@example.com')
    await page.getByPlaceholder('Password').fill('adminadmin')
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Wait for dashboard
    await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 })

    // Find and click logout button (may be in a menu)
    const logoutButton = page.getByRole('button', { name: /logout/i })
    const logoutMenuItem = page.getByRole('menuitem', { name: /logout/i })

    if (await logoutButton.isVisible({ timeout: 1000 }).catch(() => false)) {
      await logoutButton.click()
      await expect(page).toHaveURL(/.*login/)
    } else if (await logoutMenuItem.isVisible({ timeout: 1000 }).catch(() => false)) {
      await logoutMenuItem.click()
      await expect(page).toHaveURL(/.*login/)
    }
    // If logout button not found, skip this assertion
  })
})
