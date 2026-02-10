import { expect, Page } from '@playwright/test';

/**
 * Test credentials (seeded by API)
 */
export const ADMIN_EMAIL = 'admin@example.com';
export const ADMIN_PASSWORD = 'supersecret';

/**
 * Login to the application via API and set up authentication state
 * This is more reliable than form-based login for E2E tests
 */
export async function login(page: Page, email: string = ADMIN_EMAIL, password: string = ADMIN_PASSWORD) {
  // Go to login page first to initialize browser context (won't redirect away)
  await page.goto('/login');
  await page.waitForLoadState('networkidle');

  // Login via API to get token
  const authData = await page.evaluate(
    async ({ email, password }) => {
      const response = await fetch('/api/v1/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      if (!response.ok) {
        throw new Error(`Login failed: ${response.status}`);
      }

      return response.json();
    },
    { email, password }
  );

  if (!authData?.token) {
    throw new Error('Failed to obtain auth token');
  }

  // Set cookie using Playwright's context API (more reliable than document.cookie)
  await page.context().addCookies([
    {
      name: 'token',
      value: authData.token,
      domain: 'localhost',
      path: '/',
      httpOnly: false,
      secure: false,
      sameSite: 'Strict',
    },
  ]);

  // Set localStorage for client-side auth state
  await page.evaluate((token) => {
    localStorage.setItem('token', token);
    localStorage.setItem('auth-storage', JSON.stringify({ state: { token }, version: 0 }));
  }, authData.token);

  // Navigate to dashboard to confirm authentication
  await page.goto('/');
  await expect(page).not.toHaveURL(/.*login/, { timeout: 10000 });
}

/**
 * Login via form (for testing the login form itself)
 */
export async function loginViaForm(page: Page, email: string = ADMIN_EMAIL, password: string = ADMIN_PASSWORD) {
  await page.goto('/login');

  // Wait for the form to be fully loaded and React to hydrate
  const emailInput = page.getByLabel(/email/i);
  const passwordInput = page.getByLabel(/password/i);
  const submitButton = page.getByRole('button', { name: /sign in|login|anmelden/i });

  await expect(emailInput).toBeVisible({ timeout: 10000 });
  await expect(passwordInput).toBeVisible();
  await expect(submitButton).toBeVisible();
  await page.waitForLoadState('networkidle');

  // Wait extra time for React hydration
  await page.waitForTimeout(500);

  // Clear and type into email field (more reliable than fill for React forms)
  await emailInput.click();
  await emailInput.press('Control+a');
  await emailInput.type(email, { delay: 50 });

  // Clear and type into password field
  await passwordInput.click();
  await passwordInput.press('Control+a');
  await passwordInput.type(password, { delay: 50 });

  // Verify values are set correctly
  await expect(emailInput).toHaveValue(email);
  await expect(passwordInput).toHaveValue(password);

  // Submit with click
  await submitButton.click();

  // Should redirect away from login
  await expect(page).not.toHaveURL(/.*login/, { timeout: 15000 });
}

/**
 * Logout from the application
 */
export async function logout(page: Page) {
  // Click user menu button
  const userMenuButton = page.getByRole('button', { name: /user menu|benutzermenü/i });
  if (await userMenuButton.isVisible({ timeout: 2000 }).catch(() => false)) {
    await userMenuButton.click();
    await page.getByRole('menuitem', { name: /logout|sign out|abmelden/i }).click();
    await expect(page).toHaveURL(/.*login/, { timeout: 10000 });
  }
}

/**
 * Get an API token by logging in via the API
 */
export async function getApiToken(
  page: Page,
  email: string = ADMIN_EMAIL,
  password: string = ADMIN_PASSWORD
): Promise<string> {
  return page.evaluate(
    async ({ email, password }) => {
      const response = await fetch('/api/v1/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      const data = await response.json();
      return data.token;
    },
    { email, password }
  );
}

/**
 * Navigate to an organization's page
 */
export async function navigateToOrganization(page: Page, orgId: number, section: string = 'users') {
  await page.goto(`/organizations/${orgId}/${section}`);
  await page.waitForLoadState('networkidle');
}

/**
 * Create an organization via the API
 */
export async function createOrganizationViaApi(
  page: Page,
  token: string,
  name: string,
  state: string = 'berlin'
): Promise<{ id: number; name: string }> {
  return page.evaluate(
    async ({ token, name, state }) => {
      // Get CSRF token from cookie
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      const response = await fetch('/api/v1/organizations', {
        method: 'POST',
        headers,
        body: JSON.stringify({ name, state, active: true }),
      });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(`Failed to create organization: ${response.status} - ${text}`);
      }
      return response.json();
    },
    { token, name, state }
  );
}

/**
 * Delete an organization via the API
 */
export async function deleteOrganizationViaApi(page: Page, token: string, orgId: number): Promise<void> {
  await page.evaluate(
    async ({ token, orgId }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      await fetch(`/api/v1/organizations/${orgId}`, {
        method: 'DELETE',
        headers,
      });
    },
    { token, orgId }
  );
}

/**
 * Get organizations via the API
 */
export async function getOrganizationsViaApi(
  page: Page,
  token: string
): Promise<Array<{ id: number; name: string }>> {
  return page.evaluate(async ({ token }) => {
    const response = await fetch('/api/v1/organizations?limit=100', {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await response.json();
    return data.data || [];
  }, { token });
}

/**
 * Clean up test organizations (those with timestamps in the name)
 */
export async function cleanupTestOrganizations(page: Page, token: string): Promise<void> {
  const orgs = await getOrganizationsViaApi(page, token);

  for (const org of orgs) {
    // Delete orgs with timestamps in name (test orgs)
    if (/\d{10,}/.test(org.name) && org.name !== 'Kita Sonnenschein') {
      await deleteOrganizationViaApi(page, token, org.id);
    }
  }
}

/**
 * Create an employee via the API
 */
export async function createEmployeeViaApi(
  page: Page,
  token: string,
  orgId: number,
  data: { first_name: string; last_name: string; gender: string; birthdate: string }
): Promise<{ id: number }> {
  return page.evaluate(
    async ({ token, orgId, data }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      const response = await fetch(`/api/v1/organizations/${orgId}/employees`, {
        method: 'POST',
        headers,
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        throw new Error(`Failed to create employee: ${response.status}`);
      }
      return response.json();
    },
    { token, orgId, data }
  );
}

/**
 * Create a child via the API
 */
export async function createChildViaApi(
  page: Page,
  token: string,
  orgId: number,
  data: { first_name: string; last_name: string; birthdate: string; gender: string }
): Promise<{ id: number }> {
  return page.evaluate(
    async ({ token, orgId, data }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      const response = await fetch(`/api/v1/organizations/${orgId}/children`, {
        method: 'POST',
        headers,
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        throw new Error(`Failed to create child: ${response.status}`);
      }
      return response.json();
    },
    { token, orgId, data }
  );
}

/**
 * Generate a unique test name with timestamp
 */
export function uniqueName(prefix: string): string {
  return `${prefix} ${Date.now()}`;
}

/**
 * Delete an employee via the API
 */
export async function deleteEmployeeViaApi(
  page: Page,
  token: string,
  orgId: number,
  employeeId: number
): Promise<void> {
  await page.evaluate(
    async ({ token, orgId, employeeId }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      await fetch(`/api/v1/organizations/${orgId}/employees/${employeeId}`, {
        method: 'DELETE',
        headers,
      });
    },
    { token, orgId, employeeId }
  );
}

/**
 * Delete a child via the API
 */
export async function deleteChildViaApi(
  page: Page,
  token: string,
  orgId: number,
  childId: number
): Promise<void> {
  await page.evaluate(
    async ({ token, orgId, childId }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      await fetch(`/api/v1/organizations/${orgId}/children/${childId}`, {
        method: 'DELETE',
        headers,
      });
    },
    { token, orgId, childId }
  );
}

/**
 * Create an employee contract via the API
 */
export async function createEmployeeContractViaApi(
  page: Page,
  token: string,
  orgId: number,
  employeeId: number,
  data: {
    from: string;
    to?: string | null;
    position: string;
    grade: string;
    step: number;
    weekly_hours: number;
  }
): Promise<{ id: number }> {
  return page.evaluate(
    async ({ token, orgId, employeeId, data }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      const response = await fetch(
        `/api/v1/organizations/${orgId}/employees/${employeeId}/contracts`,
        {
          method: 'POST',
          headers,
          body: JSON.stringify(data),
        }
      );
      if (!response.ok) {
        throw new Error(`Failed to create employee contract: ${response.status}`);
      }
      return response.json();
    },
    { token, orgId, employeeId, data }
  );
}

/**
 * Create a child contract via the API
 */
export async function createChildContractViaApi(
  page: Page,
  token: string,
  orgId: number,
  childId: number,
  data: {
    from: string;
    to?: string | null;
    properties?: Record<string, string>;
  }
): Promise<{ id: number }> {
  return page.evaluate(
    async ({ token, orgId, childId, data }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      const response = await fetch(
        `/api/v1/organizations/${orgId}/children/${childId}/contracts`,
        {
          method: 'POST',
          headers,
          body: JSON.stringify(data),
        }
      );
      if (!response.ok) {
        throw new Error(`Failed to create child contract: ${response.status}`);
      }
      return response.json();
    },
    { token, orgId, childId, data }
  );
}

/**
 * Get the first organization via API (assumes at least one exists from seeding)
 */
export async function getFirstOrganization(
  page: Page,
  token: string
): Promise<{ id: number; name: string }> {
  return page.evaluate(async ({ token }) => {
    const response = await fetch('/api/v1/organizations?limit=1', {
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await response.json();
    if (!data.data || data.data.length === 0) {
      throw new Error('No organizations found');
    }
    return data.data[0];
  }, { token });
}

/**
 * Format a date for API submission (RFC3339)
 */
export function formatDateForApi(dateStr: string): string {
  return `${dateStr}T00:00:00Z`;
}

/**
 * Get today's date as YYYY-MM-DD
 */
export function getTodayStr(): string {
  return new Date().toISOString().split('T')[0];
}

/**
 * Get a future date as YYYY-MM-DD
 */
export function getFutureDateStr(daysFromNow: number): string {
  const date = new Date();
  date.setDate(date.getDate() + daysFromNow);
  return date.toISOString().split('T')[0];
}

/**
 * Create a section via the API
 */
export async function createSectionViaApi(
  page: Page,
  token: string,
  orgId: number,
  name: string
): Promise<{ id: number; name: string }> {
  return page.evaluate(
    async ({ token, orgId, name }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      const response = await fetch(`/api/v1/organizations/${orgId}/sections`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ name }),
      });
      if (!response.ok) {
        const text = await response.text();
        throw new Error(`Failed to create section: ${response.status} - ${text}`);
      }
      return response.json();
    },
    { token, orgId, name }
  );
}

/**
 * Delete a section via the API
 */
export async function deleteSectionViaApi(
  page: Page,
  token: string,
  orgId: number,
  sectionId: number
): Promise<void> {
  await page.evaluate(
    async ({ token, orgId, sectionId }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      await fetch(`/api/v1/organizations/${orgId}/sections/${sectionId}`, {
        method: 'DELETE',
        headers,
      });
    },
    { token, orgId, sectionId }
  );
}

/**
 * Get sections via the API
 */
export async function getSectionsViaApi(
  page: Page,
  token: string,
  orgId: number
): Promise<Array<{ id: number; name: string }>> {
  return page.evaluate(
    async ({ token, orgId }) => {
      const response = await fetch(`/api/v1/organizations/${orgId}/sections?limit=100`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await response.json();
      return data.data || [];
    },
    { token, orgId }
  );
}

/**
 * Update a child's section via the API
 */
export async function updateChildSectionViaApi(
  page: Page,
  token: string,
  orgId: number,
  childId: number,
  sectionId: number | null
): Promise<void> {
  await page.evaluate(
    async ({ token, orgId, childId, sectionId }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      };
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      await fetch(`/api/v1/organizations/${orgId}/children/${childId}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({ section_id: sectionId }),
      });
    },
    { token, orgId, childId, sectionId }
  );
}
