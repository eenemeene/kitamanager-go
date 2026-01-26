import { test, expect } from 'playwright/test'

test.describe('Navigation', () => {
  test.beforeEach(async ({ page }) => {
    // Login before each test
    await page.goto('/login')
    await page.getByPlaceholder('Email').fill('admin@example.com')
    await page.getByPlaceholder('Password').fill('adminadmin')
    await page.getByRole('button', { name: 'Sign In' }).click()
    await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 })
  })

  test('should navigate to organizations', async ({ page }) => {
    await page.getByRole('link', { name: /organization/i }).first().click()
    await expect(page).toHaveURL(/.*organization/)
  })

  test('should navigate to employees', async ({ page }) => {
    await page.getByRole('link', { name: /employee/i }).first().click()
    await expect(page).toHaveURL(/.*employee/)
  })

  test('should navigate to children', async ({ page }) => {
    await page.getByRole('link', { name: /child/i }).first().click()
    await expect(page).toHaveURL(/.*child/)
  })

  test('should navigate to groups', async ({ page }) => {
    await page.getByRole('link', { name: /group/i }).first().click()
    await expect(page).toHaveURL(/.*group/)
  })
})
