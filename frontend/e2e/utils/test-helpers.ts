import { expect, Page } from '@playwright/test';

/**
 * Test credentials (seeded by API, configurable via env vars)
 */
export const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL || 'admin@example.com';
export const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD || 'supersecret';

/**
 * Make an authenticated API request via page.evaluate.
 * Handles CSRF token extraction and auth headers in one place.
 */
async function apiRequest<T>(
  page: Page,
  token: string,
  method: string,
  url: string,
  body?: unknown
): Promise<T> {
  return page.evaluate(
    async ({ token, method, url, body }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
      };
      if (body !== undefined) {
        headers['Content-Type'] = 'application/json';
      }
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      const response = await fetch(url, {
        method,
        headers,
        body: body !== undefined ? JSON.stringify(body) : undefined,
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(`API ${method} ${url} failed: ${response.status} - ${text}`);
      }

      const contentType = response.headers.get('content-type');
      if (contentType?.includes('application/json')) {
        return response.json();
      }
      return null as T;
    },
    { token, method, url, body }
  );
}

/**
 * Login to the application via API and set up authentication state
 * This is more reliable than form-based login for E2E tests
 */
export async function login(
  page: Page,
  email: string = ADMIN_EMAIL,
  password: string = ADMIN_PASSWORD
) {
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
export async function loginViaForm(
  page: Page,
  email: string = ADMIN_EMAIL,
  password: string = ADMIN_PASSWORD
) {
  await page.goto('/login');

  // Wait for the form to be fully loaded and React to hydrate
  const emailInput = page.getByLabel(/email/i);
  const passwordInput = page.getByLabel(/password/i);
  const submitButton = page.getByRole('button', { name: /sign in|login/i });

  await expect(emailInput).toBeVisible({ timeout: 10000 });
  await expect(passwordInput).toBeVisible();
  await expect(submitButton).toBeVisible();
  await page.waitForLoadState('networkidle');

  // Wait for React hydration by ensuring submit button is enabled/interactive
  await expect(submitButton).toBeEnabled({ timeout: 5000 });

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
 * Create an organization via the API
 */
export async function createOrganizationViaApi(
  page: Page,
  token: string,
  name: string,
  state: string = 'berlin',
  defaultSectionName: string = 'Default'
): Promise<{ id: number; name: string }> {
  return apiRequest(page, token, 'POST', '/api/v1/organizations', {
    name,
    state,
    active: true,
    default_section_name: defaultSectionName,
  });
}

/**
 * Delete an organization via the API
 */
export async function deleteOrganizationViaApi(
  page: Page,
  token: string,
  orgId: number
): Promise<void> {
  await apiRequest(page, token, 'DELETE', `/api/v1/organizations/${orgId}`);
}

/**
 * Get organizations via the API
 */
export async function getOrganizationsViaApi(
  page: Page,
  token: string
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    token,
    'GET',
    '/api/v1/organizations?limit=100'
  );
  return data.data || [];
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
  return apiRequest(page, token, 'POST', `/api/v1/organizations/${orgId}/employees`, data);
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
  return apiRequest(page, token, 'POST', `/api/v1/organizations/${orgId}/children`, data);
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
  await apiRequest(page, token, 'DELETE', `/api/v1/organizations/${orgId}/employees/${employeeId}`);
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
  await apiRequest(page, token, 'DELETE', `/api/v1/organizations/${orgId}/children/${childId}`);
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
    section_id: number;
    staff_category: string;
    grade: string;
    step: number;
    weekly_hours: number;
    payplan_id: number;
  }
): Promise<{ id: number }> {
  return apiRequest(
    page,
    token,
    'POST',
    `/api/v1/organizations/${orgId}/employees/${employeeId}/contracts`,
    data
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
    section_id: number;
    properties?: Record<string, string>;
  }
): Promise<{ id: number }> {
  return apiRequest(
    page,
    token,
    'POST',
    `/api/v1/organizations/${orgId}/children/${childId}/contracts`,
    data
  );
}

/**
 * Get the first organization via API (assumes at least one exists from seeding)
 */
export async function getFirstOrganization(
  page: Page,
  token: string
): Promise<{ id: number; name: string }> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    token,
    'GET',
    '/api/v1/organizations?limit=1'
  );
  if (!data.data || data.data.length === 0) {
    throw new Error('No organizations found');
  }
  return data.data[0];
}

/**
 * Format a date for API submission (RFC3339)
 */
export function formatDateForApi(dateStr: string): string {
  return `${dateStr}T00:00:00Z`;
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
  return apiRequest(page, token, 'POST', `/api/v1/organizations/${orgId}/sections`, { name });
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
  await apiRequest(page, token, 'DELETE', `/api/v1/organizations/${orgId}/sections/${sectionId}`);
}

/**
 * Get sections via the API
 */
export async function getSectionsViaApi(
  page: Page,
  token: string,
  orgId: number
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    token,
    'GET',
    `/api/v1/organizations/${orgId}/sections?limit=100`
  );
  return data.data || [];
}

/**
 * Get pay plans via the API
 */
export async function getPayPlansViaApi(
  page: Page,
  token: string,
  orgId: number
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    token,
    'GET',
    `/api/v1/organizations/${orgId}/payplans?limit=100`
  );
  return data.data || [];
}
