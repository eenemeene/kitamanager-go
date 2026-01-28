import { expect, Page } from 'playwright/test'

/**
 * Shared test helper functions for E2E tests.
 * These provide reusable utilities for common operations.
 */

// Superadmin credentials (seeded in dev environment)
export const SUPERADMIN_EMAIL = 'admin@example.com'
export const SUPERADMIN_PASSWORD = 'supersecret'

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
 * Select an organization in the sidebar dropdown.
 * Uses timestamp-based filtering to find the org.
 */
export async function selectOrganization(page: Page, orgName: string, filterText?: string) {
  await page.reload()
  await page.waitForLoadState('networkidle')
  await page.waitForTimeout(1000)

  const orgDropdown = page.getByRole('combobox').first()
  await orgDropdown.click()

  const filterInput = page.getByPlaceholder('Search...')
  await filterInput.fill(filterText || orgName)
  await page.waitForTimeout(500)

  const orgOption = page.getByRole('option', { name: orgName })
  await expect(orgOption).toBeVisible({ timeout: 10000 })
  await orgOption.click()
  await page.waitForTimeout(500)
}

/**
 * Create an organization via the UI.
 * Assumes user is already logged in and on any page.
 * State must be explicitly provided.
 */
export async function createOrganization(page: Page, orgName: string, state: string) {
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

  await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })
  await expect(page.getByText(/organization created successfully/i)).toBeVisible({ timeout: 5000 })
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

  await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })
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

  await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 5000 })
  await expect(page.getByText('User created successfully')).toBeVisible({ timeout: 5000 })
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
 */
export async function getOrganizationByName(
  page: Page,
  token: string,
  orgName: string
): Promise<{ id: number; name: string }> {
  const response = await page.evaluate(
    async ({ token, orgName }) => {
      const res = await fetch('/api/v1/organizations?limit=100', {
        headers: { Authorization: `Bearer ${token}` }
      })
      const data = await res.json()
      return data.data.find((o: { name: string }) => o.name === orgName)
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
