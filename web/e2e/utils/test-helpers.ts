import { expect, Page } from 'playwright/test'

/**
 * Shared test helper functions for E2E tests.
 * These provide reusable utilities for common operations.
 */

// Superadmin credentials (seeded in dev environment)
export const SUPERADMIN_EMAIL = 'admin@example.com'
export const SUPERADMIN_PASSWORD = 'supersecret'

/**
 * Clean up test organizations (those with timestamps in the name).
 * Call this at the beginning of test suites to ensure a clean state.
 */
export async function cleanupTestOrganizations(page: Page, token: string) {
  await page.evaluate(
    async ({ token }) => {
      // Fetch all organizations (with pagination)
      const testOrgIds: number[] = []
      for (let pageNum = 1; pageNum <= 10; pageNum++) {
        const res = await fetch(`/api/v1/organizations?limit=100&page=${pageNum}`, {
          headers: { Authorization: `Bearer ${token}` }
        })
        if (!res.ok) break
        const data = await res.json()
        const orgs = data.data || data
        if (!Array.isArray(orgs) || orgs.length === 0) break

        // Find test organizations (those with timestamps like "Test Org 1234567890")
        for (const org of orgs) {
          if (/\d{10,}/.test(org.name) && org.name !== 'Kita Sonnenschein') {
            testOrgIds.push(org.id)
          }
        }
      }

      // Delete test organizations (limit to 50 to avoid timeout)
      for (const id of testOrgIds.slice(0, 50)) {
        await fetch(`/api/v1/organizations/${id}`, {
          method: 'DELETE',
          headers: { Authorization: `Bearer ${token}` }
        })
      }
    },
    { token }
  )
}

/**
 * Login to the application
 */
export async function login(page: Page, email: string, password: string) {
  await page.goto('/login')
  await page.getByPlaceholder('Email').fill(email)
  await page.getByPlaceholder('Password').fill(password)
  await page.getByRole('button', { name: 'Sign In' }).click()
  await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 })
}

/**
 * Logout from the application
 */
export async function logout(page: Page, currentUserEmail: string) {
  await page.getByRole('button', { name: currentUserEmail }).click()
  await page.getByRole('menuitem', { name: /logout|sign out|abmelden/i }).click()
  await expect(page).toHaveURL(/.*login/, { timeout: 10000 })
}

/**
 * Select an organization by navigating directly to an org-scoped page.
 * This bypasses the dropdown which may not show all organizations.
 * The organization will be auto-selected when navigating to its scoped page.
 */
export async function selectOrganizationById(page: Page, orgId: number) {
  // Navigate to the organization's children page (any org-scoped page works)
  await page.goto(`/organizations/${orgId}/children`)
  await page.waitForLoadState('networkidle')

  // Store the selected org ID in localStorage so the UI picks it up
  await page.evaluate((id) => {
    localStorage.setItem('selectedOrganizationId', id.toString())
  }, orgId)

  // Reload to ensure the sidebar reflects the selection
  await page.reload()
  await page.waitForLoadState('networkidle')
}

/**
 * Select an organization in the sidebar dropdown.
 * Uses filtering to find the org with retry logic for reliability.
 * Falls back to direct navigation if dropdown selection fails.
 */
export async function selectOrganization(page: Page, orgName: string, _filterText?: string) {
  // Ensure page is fully loaded
  await page.waitForLoadState('domcontentloaded')
  await page.waitForLoadState('networkidle')

  // Wait for the dropdown to be ready and have options loaded
  const orgDropdown = page.getByRole('combobox').first()
  await expect(orgDropdown).toBeVisible({ timeout: 10000 })

  // Wait for API data to populate the dropdown (increased for reliability)
  await page.waitForTimeout(1000)

  // Retry logic for dropdown selection (handles timing issues)
  let retries = 3
  while (retries > 0) {
    try {
      // Click to open dropdown
      await orgDropdown.click()

      // Wait for dropdown panel to appear
      const panel = page.locator('.p-select-overlay, .p-dropdown-panel')
      await expect(panel).toBeVisible({ timeout: 5000 })

      // Filter if filter input is available
      const filterInput = page.getByPlaceholder('Search...')
      if (await filterInput.isVisible({ timeout: 1000 }).catch(() => false)) {
        // Clear any existing filter first
        await filterInput.clear()
        // Always use the full org name for filtering (more reliable than partial timestamp)
        await filterInput.fill(orgName)
        // Wait for filter to apply and API to respond
        await page.waitForTimeout(1000)
      }

      // Find and click the option - use exact match to avoid partial matches
      const orgOption = page.getByRole('option', { name: orgName, exact: true })
      await expect(orgOption).toBeVisible({ timeout: 5000 })
      await orgOption.click()

      // Wait for selection to complete (dropdown closes)
      await expect(panel).not.toBeVisible({ timeout: 5000 })
      return // Success
    } catch (error) {
      retries--
      if (retries === 0) {
        // Fallback: Try to find org via API and navigate directly
        const token = await page.evaluate(() => localStorage.getItem('token'))
        if (token) {
          const org = await getOrganizationByName(page, token, orgName)
          if (org) {
            await selectOrganizationById(page, org.id)
            return
          }
        }
        throw error
      }
      // Close dropdown if still open and retry with increasing delay
      await page.keyboard.press('Escape')
      await page.waitForTimeout(1000)
    }
  }
}

/**
 * Create an organization via the UI.
 * Assumes user is already logged in and on any page.
 * State must be explicitly provided.
 * Returns the created organization's ID.
 */
export async function createOrganization(
  page: Page,
  orgName: string,
  state: string
): Promise<number> {
  await page.getByRole('link', { name: /organization/i }).first().click()
  await expect(page).toHaveURL(/.*organization/)

  await page.getByRole('button', { name: /new organization/i }).click()
  await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

  // Fill organization name (input has id="name")
  await page.locator('#name').fill(orgName)

  // Select state from dropdown (the Select component has id="state")
  const stateDropdown = page.locator('#state')
  await stateDropdown.click()
  await page.waitForTimeout(300)
  await page.getByRole('option', { name: new RegExp(state, 'i') }).click()

  await page.getByRole('button', { name: 'Save' }).click()

  // Wait for dialog to close (confirms success)
  await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })

  // Wait for success toast to appear (confirms API success)
  await expect(page.locator('.p-toast-message-success')).toBeVisible({ timeout: 5000 })

  // Get the auth token from localStorage
  const token = await page.evaluate(() => localStorage.getItem('token'))
  if (!token) throw new Error('No auth token found')

  // Find the organization via API to get its ID (with pagination support)
  const org = await page.evaluate(
    async ({ token, orgName }) => {
      // Search through pages until we find the org (max 10 pages = 1000 orgs)
      for (let page = 1; page <= 10; page++) {
        const res = await fetch(`/api/v1/organizations?limit=100&page=${page}`, {
          headers: { Authorization: `Bearer ${token}` }
        })
        if (!res.ok) {
          throw new Error(`Failed to fetch organizations: ${res.status}`)
        }
        const data = await res.json()
        const orgs = data.data || data
        if (!Array.isArray(orgs) || orgs.length === 0) break

        const found = orgs.find((o: { name: string }) => o.name === orgName)
        if (found) return found
      }
      return null
    },
    { token, orgName }
  )

  if (!org) {
    throw new Error(`Organization "${orgName}" not found after creation`)
  }

  return org.id
}

/**
 * Create a group via the UI.
 * Assumes user is logged in and has selected an organization.
 */
export async function createGroup(page: Page, groupName: string) {
  await page.getByRole('link', { name: /group/i }).first().click()
  await expect(page).toHaveURL(/.*groups/)

  await page.getByRole('button', { name: /new group/i }).click()
  await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

  // The name field has id="name"
  await page.locator('#name').fill(groupName)
  await page.getByRole('button', { name: 'Save' }).click()

  await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })
  await expect(page.getByRole('cell', { name: groupName })).toBeVisible()
}

/**
 * Create a user via the UI.
 * Assumes user is logged in and has selected an organization.
 */
export async function createUser(page: Page, name: string, email: string, password: string) {
  await page.getByRole('link', { name: /user/i }).first().click()
  await expect(page).toHaveURL(/.*users/)

  await page.getByRole('button', { name: /new user/i }).click()
  await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 })

  // Fields have id attributes: name, email, password
  // Password field is a PrimeVue component, so we need to target the inner input
  await page.locator('#name').fill(name)
  await page.locator('#email').fill(email)
  await page.locator('#password input').fill(password)
  await page.getByRole('button', { name: 'Save' }).click()

  // Wait for dialog to close (confirms success)
  await expect(page.locator('.p-dialog')).not.toBeVisible({ timeout: 10000 })
}

// ============================================================================
// API Helper Functions
// ============================================================================

/**
 * Get an API token by logging in via the API.
 */
export async function getApiToken(page: Page, email: string, password: string): Promise<string> {
  return page.evaluate(
    async ({ email, password }) => {
      const response = await fetch('/api/v1/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      })
      const data = await response.json()
      return data.token
    },
    { email, password }
  )
}

/**
 * Get a user by email via the API.
 * Note: Requires superadmin token to see all users.
 */
export async function getUserByEmail(
  page: Page,
  token: string,
  email: string
): Promise<{ id: number; email: string; name: string }> {
  const response = await page.evaluate(
    async ({ token, email }) => {
      const res = await fetch('/api/v1/users?limit=100', {
        headers: { Authorization: `Bearer ${token}` }
      })
      if (!res.ok) {
        throw new Error(`Failed to fetch users: ${res.status} ${res.statusText}`)
      }
      const data = await res.json()
      const user = data.data.find((u: { email: string }) => u.email === email)
      if (!user) {
        throw new Error(`User with email ${email} not found in ${data.data.length} users`)
      }
      return user
    },
    { token, email }
  )
  return response
}

/**
 * Get organization by name via the API.
 * Uses pagination to search through all organizations.
 */
export async function getOrganizationByName(
  page: Page,
  token: string,
  orgName: string
): Promise<{ id: number; name: string } | null> {
  const response = await page.evaluate(
    async ({ token, orgName }) => {
      // Search through pages until we find the org (max 10 pages = 1000 orgs)
      for (let pageNum = 1; pageNum <= 10; pageNum++) {
        const res = await fetch(`/api/v1/organizations?limit=100&page=${pageNum}`, {
          headers: { Authorization: `Bearer ${token}` }
        })
        if (!res.ok) return null
        const data = await res.json()
        const orgs = data.data || data
        if (!Array.isArray(orgs) || orgs.length === 0) break

        const found = orgs.find((o: { name: string }) => o.name === orgName)
        if (found) return found
      }
      return null
    },
    { token, orgName }
  )
  return response
}

/**
 * Get group by name within an organization via the API.
 */
export async function getGroupByName(
  page: Page,
  token: string,
  orgId: number,
  groupName: string
): Promise<{ id: number; name: string }> {
  const response = await page.evaluate(
    async ({ token, orgId, groupName }) => {
      const res = await fetch(`/api/v1/organizations/${orgId}/groups`, {
        headers: { Authorization: `Bearer ${token}` }
      })
      const data = await res.json()
      const groups = Array.isArray(data) ? data : data.data || []
      return groups.find((g: { name: string }) => g.name === groupName)
    },
    { token, orgId, groupName }
  )
  return response
}

/**
 * Add a user to a group with a specific role via the API.
 */
export async function addUserToGroup(
  page: Page,
  token: string,
  userId: number,
  groupId: number,
  role: 'admin' | 'manager' | 'member'
) {
  await page.evaluate(
    async ({ token, userId, groupId, role }) => {
      const response = await fetch(`/api/v1/users/${userId}/groups`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ group_id: groupId, role })
      })
      if (!response.ok) {
        throw new Error(`Failed to add user to group: ${response.status}`)
      }
    },
    { token, userId, groupId, role }
  )
}

/**
 * Create a user and add them to a group with a role.
 * This is a convenience function that combines UI and API operations.
 * Returns the created user's details.
 */
export async function createUserWithRole(
  page: Page,
  superadminToken: string,
  groupId: number,
  name: string,
  email: string,
  password: string,
  role: 'admin' | 'manager' | 'member'
): Promise<{ id: number; email: string; name: string }> {
  // Create user via UI
  await createUser(page, name, email, password)

  // Get user ID via API (using superadmin token to see all users)
  const user = await getUserByEmail(page, superadminToken, email)

  // Add user to group with role
  await addUserToGroup(page, superadminToken, user.id, groupId, role)

  return user
}

/**
 * Get all government fundings via the API.
 * Requires superadmin token.
 */
export async function getGovernmentFundings(
  page: Page,
  token: string
): Promise<Array<{ id: number; name: string }>> {
  return page.evaluate(async ({ token }) => {
    const res = await fetch('/api/v1/government-fundings', {
      headers: { Authorization: `Bearer ${token}` }
    })
    if (!res.ok) {
      throw new Error(`Failed to fetch government fundings: ${res.status}`)
    }
    const data = await res.json()
    return data.data || data
  }, { token })
}

// Note: assignGovernmentFundingToOrganization was removed.
// Government funding is now automatically determined by the organization's state.
