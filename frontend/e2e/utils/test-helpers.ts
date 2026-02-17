import { expect, Page } from '@playwright/test';

/**
 * Test credentials (seeded by API, configurable via env vars)
 */
export const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL || 'admin@example.com';
export const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD || 'supersecret';

/**
 * Make an authenticated API request via page.evaluate.
 * Auth is handled via HttpOnly cookies (set during login).
 * CSRF token is extracted from the non-HttpOnly csrf_token cookie.
 */
async function apiRequest<T>(
  page: Page,
  method: string,
  url: string,
  body?: unknown
): Promise<T> {
  return page.evaluate(
    async ({ method, url, body }) => {
      const csrfMatch = document.cookie.match(/csrf_token=([^;]+)/);
      const csrfToken = csrfMatch ? csrfMatch[1] : null;

      const headers: Record<string, string> = {};
      if (body !== undefined) {
        headers['Content-Type'] = 'application/json';
      }
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }

      const response = await fetch(url, {
        method,
        headers,
        credentials: 'same-origin',
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
    { method, url, body }
  );
}

/**
 * Login to the application via API and set up authentication state.
 * The API sets HttpOnly cookies (access_token, refresh_token, csrf_token)
 * which are automatically sent with subsequent requests.
 */
export async function login(
  page: Page,
  email: string = ADMIN_EMAIL,
  password: string = ADMIN_PASSWORD
) {
  // Go to login page first to initialize browser context
  await page.goto('/login');
  await page.waitForLoadState('networkidle');

  // Login via API - cookies are set automatically by the browser
  await page.evaluate(
    async ({ email, password }) => {
      const response = await fetch('/api/v1/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify({ email, password }),
      });

      if (!response.ok) {
        throw new Error(`Login failed: ${response.status}`);
      }
    },
    { email, password }
  );

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
 * Create an organization via the API
 */
export async function createOrganizationViaApi(
  page: Page,
  name: string,
  state: string = 'berlin',
  defaultSectionName: string = 'Default'
): Promise<{ id: number; name: string }> {
  return apiRequest(page, 'POST', '/api/v1/organizations', {
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
  orgId: number
): Promise<void> {
  await apiRequest(page, 'DELETE', `/api/v1/organizations/${orgId}`);
}

/**
 * Get organizations via the API
 */
export async function getOrganizationsViaApi(
  page: Page
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    'GET',
    '/api/v1/organizations?limit=100'
  );
  if (!Array.isArray(data.data)) {
    throw new Error('getOrganizationsViaApi: response missing data array');
  }
  return data.data;
}

/**
 * Create an employee via the API
 */
export async function createEmployeeViaApi(
  page: Page,
  orgId: number,
  data: { first_name: string; last_name: string; gender: string; birthdate: string }
): Promise<{ id: number }> {
  return apiRequest(page, 'POST', `/api/v1/organizations/${orgId}/employees`, data);
}

/**
 * Create a child via the API
 */
export async function createChildViaApi(
  page: Page,
  orgId: number,
  data: { first_name: string; last_name: string; birthdate: string; gender: string }
): Promise<{ id: number }> {
  return apiRequest(page, 'POST', `/api/v1/organizations/${orgId}/children`, data);
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
  orgId: number,
  employeeId: number
): Promise<void> {
  await apiRequest(page, 'DELETE', `/api/v1/organizations/${orgId}/employees/${employeeId}`);
}

/**
 * Delete a child via the API
 */
export async function deleteChildViaApi(
  page: Page,
  orgId: number,
  childId: number
): Promise<void> {
  await apiRequest(page, 'DELETE', `/api/v1/organizations/${orgId}/children/${childId}`);
}

/**
 * Create an employee contract via the API
 */
export async function createEmployeeContractViaApi(
  page: Page,
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
    'POST',
    `/api/v1/organizations/${orgId}/children/${childId}/contracts`,
    data
  );
}

/**
 * Get the first organization via API (assumes at least one exists from seeding)
 */
export async function getFirstOrganization(
  page: Page
): Promise<{ id: number; name: string }> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
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
  orgId: number,
  name: string
): Promise<{ id: number; name: string }> {
  return apiRequest(page, 'POST', `/api/v1/organizations/${orgId}/sections`, { name });
}

/**
 * Delete a section via the API
 */
export async function deleteSectionViaApi(
  page: Page,
  orgId: number,
  sectionId: number
): Promise<void> {
  await apiRequest(page, 'DELETE', `/api/v1/organizations/${orgId}/sections/${sectionId}`);
}

/**
 * Get sections via the API
 */
export async function getSectionsViaApi(
  page: Page,
  orgId: number
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    'GET',
    `/api/v1/organizations/${orgId}/sections?limit=100`
  );
  if (!Array.isArray(data.data)) {
    throw new Error(`getSectionsViaApi: response missing data array for org ${orgId}`);
  }
  return data.data;
}

/**
 * Get pay plans via the API
 */
export async function getPayPlansViaApi(
  page: Page,
  orgId: number
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    'GET',
    `/api/v1/organizations/${orgId}/payplans?limit=100`
  );
  if (!Array.isArray(data.data)) {
    throw new Error(`getPayPlansViaApi: response missing data array for org ${orgId}`);
  }
  return data.data;
}

/**
 * Get employees via the API
 */
export async function getEmployeesViaApi(
  page: Page,
  orgId: number
): Promise<Array<{ id: number; first_name: string; last_name: string }>> {
  const data = await apiRequest<{
    data: Array<{ id: number; first_name: string; last_name: string }>;
  }>(page, 'GET', `/api/v1/organizations/${orgId}/employees?limit=100`);
  if (!Array.isArray(data.data)) {
    throw new Error(`getEmployeesViaApi: response missing data array for org ${orgId}`);
  }
  return data.data;
}

/**
 * Get children via the API
 */
export async function getChildrenViaApi(
  page: Page,
  orgId: number
): Promise<Array<{ id: number; first_name: string; last_name: string }>> {
  const data = await apiRequest<{
    data: Array<{ id: number; first_name: string; last_name: string }>;
  }>(page, 'GET', `/api/v1/organizations/${orgId}/children?limit=100`);
  if (!Array.isArray(data.data)) {
    throw new Error(`getChildrenViaApi: response missing data array for org ${orgId}`);
  }
  return data.data;
}

/**
 * Get users via the API
 */
export async function getUsersViaApi(
  page: Page
): Promise<Array<{ id: number; name: string; email: string }>> {
  const data = await apiRequest<{
    data: Array<{ id: number; name: string; email: string }>;
  }>(page, 'GET', '/api/v1/users?limit=100');
  if (!Array.isArray(data.data)) {
    throw new Error('getUsersViaApi: response missing data array');
  }
  return data.data;
}

/**
 * Create a user via the API
 */
export async function createUserViaApi(
  page: Page,
  data: { name: string; email: string; password: string; active?: boolean }
): Promise<{ id: number; name: string; email: string }> {
  return apiRequest(page, 'POST', '/api/v1/users', {
    active: true,
    ...data,
  });
}

/**
 * Delete a user via the API
 */
export async function deleteUserViaApi(
  page: Page,
  userId: number
): Promise<void> {
  await apiRequest(page, 'DELETE', `/api/v1/users/${userId}`);
}

/**
 * Get groups via the API
 */
export async function getGroupsViaApi(
  page: Page,
  orgId: number
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    'GET',
    `/api/v1/organizations/${orgId}/groups?limit=100`
  );
  if (!Array.isArray(data.data)) {
    throw new Error(`getGroupsViaApi: response missing data array for org ${orgId}`);
  }
  return data.data;
}

/**
 * Create a group via the API
 */
export async function createGroupViaApi(
  page: Page,
  orgId: number,
  data: { name: string; active?: boolean }
): Promise<{ id: number; name: string }> {
  return apiRequest(page, 'POST', `/api/v1/organizations/${orgId}/groups`, {
    active: true,
    ...data,
  });
}

/**
 * Delete a group via the API
 */
export async function deleteGroupViaApi(
  page: Page,
  orgId: number,
  groupId: number
): Promise<void> {
  await apiRequest(
    page,
    'DELETE',
    `/api/v1/organizations/${orgId}/groups/${groupId}`
  );
}

/**
 * Create a government funding via the API
 */
export async function createGovernmentFundingViaApi(
  page: Page,
  data: { name: string; state: string }
): Promise<{ id: number; name: string }> {
  return apiRequest(page, 'POST', '/api/v1/government-fundings', data);
}

/**
 * Delete a government funding via the API
 */
export async function deleteGovernmentFundingViaApi(
  page: Page,
  fundingId: number
): Promise<void> {
  await apiRequest(page, 'DELETE', `/api/v1/government-fundings/${fundingId}`);
}

/**
 * Get government fundings via the API
 */
export async function getGovernmentFundingsViaApi(
  page: Page
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    'GET',
    '/api/v1/government-fundings?limit=100'
  );
  if (!Array.isArray(data.data)) {
    throw new Error('getGovernmentFundingsViaApi: response missing data array');
  }
  return data.data;
}

/**
 * Get government funding details (with periods) via the API
 */
export async function getGovernmentFundingViaApi(
  page: Page,
  fundingId: number
): Promise<{ id: number; name: string; periods?: Array<{ id: number; properties?: Array<{ key: string; value: string }> }> }> {
  return apiRequest(page, 'GET', `/api/v1/government-fundings/${fundingId}?periods_limit=0`);
}

/**
 * Create a government funding period via the API
 */
export async function createFundingPeriodViaApi(
  page: Page,
  fundingId: number,
  data: { from: string; to?: string; full_time_weekly_hours?: number }
): Promise<{ id: number }> {
  // API expects RFC3339 dates (time.Time)
  const body = {
    full_time_weekly_hours: 39,
    ...data,
    from: data.from.includes('T') ? data.from : `${data.from}T00:00:00Z`,
    ...(data.to ? { to: data.to.includes('T') ? data.to : `${data.to}T00:00:00Z` } : {}),
  };
  return apiRequest(
    page,
    'POST',
    `/api/v1/government-fundings/${fundingId}/periods`,
    body
  );
}

/**
 * Create a government funding property via the API
 */
export async function createFundingPropertyViaApi(
  page: Page,
  fundingId: number,
  periodId: number,
  data: { key: string; value: string; payment: number; requirement: number; min_age?: number; max_age?: number }
): Promise<{ id: number }> {
  return apiRequest(
    page,
    'POST',
    `/api/v1/government-fundings/${fundingId}/periods/${periodId}/properties`,
    data
  );
}

/**
 * Ensure the Berlin government funding has at least one period with properties.
 * Used by contract tests that depend on property suggestions being available.
 */
export async function ensureFundingHasProperties(page: Page): Promise<void> {
  const fundings = await getGovernmentFundingsViaApi(page);
  if (fundings.length === 0) return;

  const funding = fundings[0];
  const details = await getGovernmentFundingViaApi(page, funding.id);

  // Check if funding already has periods with properties
  if (details.periods && details.periods.length > 0) {
    const hasProperties = details.periods.some(
      (p) => p.properties && p.properties.length > 0
    );
    if (hasProperties) return;
  }

  // Create a period covering a wide date range
  const period = await createFundingPeriodViaApi(page, funding.id, {
    from: '2020-01-01',
    full_time_weekly_hours: 39,
  });

  // Add care_type properties
  const properties = [
    { key: 'care_type', value: 'ganztag', payment: 100000, requirement: 0.301, min_age: 0, max_age: 8 },
    { key: 'care_type', value: 'halbtag', payment: 70000, requirement: 0.15, min_age: 0, max_age: 8 },
    { key: 'care_type', value: 'teilzeit', payment: 85000, requirement: 0.217, min_age: 0, max_age: 8 },
    { key: 'ndh', value: 'ndh', payment: 10000, requirement: 0.017, min_age: 0, max_age: 8 },
  ];

  for (const prop of properties) {
    await createFundingPropertyViaApi(page, funding.id, period.id, prop);
  }
}

/**
 * Create a budget item via the API
 */
export async function createBudgetItemViaApi(
  page: Page,
  orgId: number,
  data: { name: string; category: string; per_child?: boolean }
): Promise<{ id: number; name: string }> {
  return apiRequest(
    page,
    'POST',
    `/api/v1/organizations/${orgId}/budget-items`,
    data
  );
}

/**
 * Delete a budget item via the API
 */
export async function deleteBudgetItemViaApi(
  page: Page,
  orgId: number,
  budgetItemId: number
): Promise<void> {
  await apiRequest(
    page,
    'DELETE',
    `/api/v1/organizations/${orgId}/budget-items/${budgetItemId}`
  );
}

/**
 * Get budget items via the API
 */
export async function getBudgetItemsViaApi(
  page: Page,
  orgId: number
): Promise<Array<{ id: number; name: string }>> {
  const data = await apiRequest<{ data: Array<{ id: number; name: string }> }>(
    page,
    'GET',
    `/api/v1/organizations/${orgId}/budget-items?limit=100`
  );
  if (!Array.isArray(data.data)) {
    throw new Error(`getBudgetItemsViaApi: response missing data array for org ${orgId}`);
  }
  return data.data;
}

/**
 * Create a pay plan via the API
 */
export async function createPayPlanViaApi(
  page: Page,
  orgId: number,
  name: string
): Promise<{ id: number; name: string }> {
  return apiRequest(
    page,
    'POST',
    `/api/v1/organizations/${orgId}/payplans`,
    { name }
  );
}

/**
 * Delete a pay plan via the API
 */
export async function deletePayPlanViaApi(
  page: Page,
  orgId: number,
  payPlanId: number
): Promise<void> {
  await apiRequest(
    page,
    'DELETE',
    `/api/v1/organizations/${orgId}/payplans/${payPlanId}`
  );
}

/**
 * Create a child with an active contract so it appears in the list.
 * The children list filters by active_on=today, so children without
 * an active contract won't show up.
 */
export async function createChildWithContractViaApi(
  page: Page,
  orgId: number,
  data: { first_name: string; last_name: string; gender: string; birthdate: string }
): Promise<{ id: number }> {
  const child = await createChildViaApi(page, orgId, data);
  const sections = await getSectionsViaApi(page, orgId);
  await createChildContractViaApi(page, orgId, child.id, {
    from: '2024-01-01T00:00:00Z',
    section_id: sections[0].id,
  });
  return child;
}

/**
 * Create an employee with an active contract so it appears in the list.
 * The employee list filters by active_on=today, so employees without
 * an active contract won't show up.
 */
export async function createEmployeeWithContractViaApi(
  page: Page,
  orgId: number,
  data: { first_name: string; last_name: string; gender: string; birthdate: string }
): Promise<{ id: number }> {
  const emp = await createEmployeeViaApi(page, orgId, data);
  const sections = await getSectionsViaApi(page, orgId);
  const payPlans = await getPayPlansViaApi(page, orgId);
  await createEmployeeContractViaApi(page, orgId, emp.id, {
    from: '2024-01-01T00:00:00Z',
    section_id: sections[0].id,
    staff_category: 'qualified',
    grade: 'S8a',
    step: 1,
    weekly_hours: 39,
    payplan_id: payPlans[0].id,
  });
  return emp;
}
