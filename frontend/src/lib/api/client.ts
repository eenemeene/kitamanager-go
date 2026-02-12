import axios, { type AxiosInstance, type AxiosError } from 'axios';
import type {
  LoginRequest,
  LoginResponse,
  Organization,
  OrganizationCreateRequest,
  OrganizationUpdateRequest,
  User,
  UserCreateRequest,
  UserUpdateRequest,
  Group,
  GroupCreateRequest,
  GroupUpdateRequest,
  Employee,
  EmployeeCreateRequest,
  EmployeeUpdateRequest,
  EmployeeContract,
  EmployeeContractCreateRequest,
  EmployeeContractUpdateRequest,
  Child,
  ChildCreateRequest,
  ChildUpdateRequest,
  ChildContract,
  ChildContractCreateRequest,
  ChildContractUpdateRequest,
  ChildrenFundingResponse,
  ChildrenContractCountByMonthResponse,
  AgeDistributionResponse,
  Role,
  UserGroupResponse,
  UserMembershipsResponse,
  GovernmentFunding,
  GovernmentFundingCreateRequest,
  GovernmentFundingUpdateRequest,
  GovernmentFundingPeriod,
  GovernmentFundingPeriodCreateRequest,
  GovernmentFundingPeriodUpdateRequest,
  GovernmentFundingProperty,
  GovernmentFundingPropertyCreateRequest,
  GovernmentFundingPropertyUpdateRequest,
  PayPlan,
  PayPlanCreateRequest,
  PayPlanUpdateRequest,
  PayPlanPeriod,
  PayPlanPeriodCreateRequest,
  PayPlanPeriodUpdateRequest,
  PayPlanEntry,
  PayPlanEntryCreateRequest,
  PayPlanEntryUpdateRequest,
  StepPromotionsResponse,
  Section,
  SectionCreateRequest,
  SectionUpdateRequest,
  PaginatedResponse,
  PaginationParams,
} from './types';
import { DEFAULT_PAGE_SIZE } from './types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL
  ? `${process.env.NEXT_PUBLIC_API_URL}/api/v1`
  : '/api/v1';

import { getCookie } from '@/lib/utils';

// Helper to get CSRF token from cookie
function getCSRFToken(): string | null {
  return getCookie('csrf_token');
}

class ApiClient {
  private client: AxiosInstance;
  private onUnauthorized?: () => void;
  private refreshToken: string | null = null;
  private isRefreshing = false;
  private refreshQueue: Array<{
    resolve: (value?: unknown) => void;
    reject: (reason?: unknown) => void;
  }> = [];

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
      // Enable sending cookies with requests (for HttpOnly auth cookies)
      withCredentials: true,
    });

    // Request interceptor to add CSRF token for state-changing requests
    this.client.interceptors.request.use(
      (config) => {
        // Add CSRF token header for non-GET requests (POST, PUT, DELETE, PATCH)
        const method = config.method?.toLowerCase();
        if (method && !['get', 'head', 'options'].includes(method)) {
          const csrfToken = getCSRFToken();
          if (csrfToken) {
            config.headers['X-CSRF-Token'] = csrfToken;
          }
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor with token refresh on 401
    this.client.interceptors.response.use(
      (response) => response,
      async (error: AxiosError) => {
        const originalRequest = error.config as typeof error.config & { _retry?: boolean };

        // Skip refresh for login/refresh/logout endpoints or already-retried requests
        const url = originalRequest?.url || '';
        const isAuthEndpoint =
          url.includes('/login') || url.includes('/refresh') || url.includes('/logout');

        if (error.response?.status === 401 && !isAuthEndpoint && !originalRequest?._retry) {
          if (this.refreshToken) {
            if (this.isRefreshing) {
              // Queue this request while refresh is in progress
              return new Promise((resolve, reject) => {
                this.refreshQueue.push({ resolve, reject });
              }).then(() => {
                return this.client(originalRequest!);
              });
            }

            this.isRefreshing = true;
            originalRequest!._retry = true;

            try {
              const response = await this.client.post<LoginResponse>('/refresh', {
                refresh_token: this.refreshToken,
              });
              this.refreshToken = response.data.refresh_token || null;

              // Resolve all queued requests
              this.refreshQueue.forEach((pending) => pending.resolve());
              this.refreshQueue = [];

              // Retry the original request
              return this.client(originalRequest!);
            } catch {
              // Refresh failed - clear queue and log out
              this.refreshQueue.forEach((pending) => pending.reject(error));
              this.refreshQueue = [];
              this.refreshToken = null;

              if (this.onUnauthorized) {
                this.onUnauthorized();
              }
              return Promise.reject(error);
            } finally {
              this.isRefreshing = false;
            }
          }

          // No refresh token available - log out
          if (this.onUnauthorized) {
            this.onUnauthorized();
          }
        }
        return Promise.reject(error);
      }
    );
  }

  setOnUnauthorized(callback: () => void) {
    this.onUnauthorized = callback;
  }

  setRefreshToken(token: string | null) {
    this.refreshToken = token;
  }

  private topLevelCrud<T, TCreate, TUpdate>(resource: string) {
    return {
      list: (params: PaginationParams = {}) => {
        const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
        return this.client
          .get<PaginatedResponse<T>>(`/${resource}?page=${page}&limit=${limit}`)
          .then((r) => r.data);
      },
      get: (id: number) => this.client.get<T>(`/${resource}/${id}`).then((r) => r.data),
      create: (data: TCreate) => this.client.post<T>(`/${resource}`, data).then((r) => r.data),
      update: (id: number, data: TUpdate) =>
        this.client.put<T>(`/${resource}/${id}`, data).then((r) => r.data),
      delete: (id: number) => this.client.delete(`/${resource}/${id}`).then(() => {}),
    };
  }

  private orgScopedCrud<T, TCreate, TUpdate>(resource: string) {
    return {
      list: (
        orgId: number,
        params: PaginationParams & { search?: string; staff_category?: string } = {}
      ) => {
        const { page = 1, limit = DEFAULT_PAGE_SIZE, search, staff_category } = params;
        const qp = new URLSearchParams({ page: String(page), limit: String(limit) });
        if (search) qp.set('search', search);
        if (staff_category) qp.set('staff_category', staff_category);
        return this.client
          .get<PaginatedResponse<T>>(`/organizations/${orgId}/${resource}?${qp}`)
          .then((r) => r.data);
      },
      get: (orgId: number, id: number) =>
        this.client.get<T>(`/organizations/${orgId}/${resource}/${id}`).then((r) => r.data),
      create: (orgId: number, data: TCreate) =>
        this.client.post<T>(`/organizations/${orgId}/${resource}`, data).then((r) => r.data),
      update: (orgId: number, id: number, data: TUpdate) =>
        this.client.put<T>(`/organizations/${orgId}/${resource}/${id}`, data).then((r) => r.data),
      delete: (orgId: number, id: number) =>
        this.client.delete(`/organizations/${orgId}/${resource}/${id}`).then(() => {}),
    };
  }

  // Auth
  async login(request: LoginRequest): Promise<LoginResponse> {
    const response = await this.client.post<LoginResponse>('/login', request);
    // Store refresh token for automatic token refresh on 401
    this.refreshToken = response.data.refresh_token || null;
    return response.data;
  }

  async logout(): Promise<void> {
    this.refreshToken = null;
    await this.client.post('/logout');
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.client.get<User>('/me');
    return response.data;
  }

  // Organizations
  private _organizations = this.topLevelCrud<
    Organization,
    OrganizationCreateRequest,
    OrganizationUpdateRequest
  >('organizations');
  getOrganizations = this._organizations.list;
  getOrganization = this._organizations.get;
  createOrganization = this._organizations.create;
  updateOrganization = this._organizations.update;
  deleteOrganization = this._organizations.delete;

  async getOrganizationsAll(): Promise<Organization[]> {
    // Backend max limit is 100
    const response = await this.client.get<PaginatedResponse<Organization>>(
      '/organizations?limit=100'
    );
    return response.data.data;
  }

  // Users
  private _users = this.topLevelCrud<User, UserCreateRequest, UserUpdateRequest>('users');
  getUsers = this._users.list;
  getUser = this._users.get;
  createUser = this._users.create;
  updateUser = this._users.update;
  deleteUser = this._users.delete;

  // User-Group assignments with roles
  async addUserToGroup(userId: number, groupId: number, role: Role): Promise<UserGroupResponse> {
    const response = await this.client.post<UserGroupResponse>(`/users/${userId}/groups`, {
      group_id: groupId,
      role,
    });
    return response.data;
  }

  async removeUserFromGroup(userId: number, groupId: number): Promise<void> {
    await this.client.delete(`/users/${userId}/groups/${groupId}`);
  }

  async updateUserGroupRole(
    userId: number,
    groupId: number,
    role: Role
  ): Promise<UserGroupResponse> {
    const response = await this.client.put<UserGroupResponse>(
      `/users/${userId}/groups/${groupId}`,
      { role }
    );
    return response.data;
  }

  async getUserMemberships(userId: number): Promise<UserMembershipsResponse> {
    const response = await this.client.get<UserMembershipsResponse>(`/users/${userId}/memberships`);
    return response.data;
  }

  async setSuperAdmin(userId: number, isSuperAdmin: boolean): Promise<User> {
    const response = await this.client.put<User>(`/users/${userId}/superadmin`, {
      is_superadmin: isSuperAdmin,
    });
    return response.data;
  }

  // User-Organization assignments
  async addUserToOrganization(userId: number, organizationId: number): Promise<void> {
    await this.client.post(`/users/${userId}/organizations`, { organization_id: organizationId });
  }

  async removeUserFromOrganization(userId: number, organizationId: number): Promise<void> {
    await this.client.delete(`/users/${userId}/organizations/${organizationId}`);
  }

  // Groups (organization-scoped)
  private _groups = this.orgScopedCrud<Group, GroupCreateRequest, GroupUpdateRequest>('groups');
  getGroups = this._groups.list;
  getGroup = this._groups.get;
  createGroup = this._groups.create;
  updateGroup = this._groups.update;
  deleteGroup = this._groups.delete;

  // Organization users
  async getOrganizationUsers(
    orgId: number,
    params: PaginationParams = {}
  ): Promise<PaginatedResponse<User>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<User>>(
      `/organizations/${orgId}/users?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  // Employees (organization-scoped)
  private _employees = this.orgScopedCrud<
    Employee,
    Omit<EmployeeCreateRequest, 'organization_id'>,
    EmployeeUpdateRequest
  >('employees');
  getEmployees = this._employees.list;
  getEmployee = this._employees.get;
  createEmployee = this._employees.create;
  updateEmployee = this._employees.update;
  deleteEmployee = this._employees.delete;

  // Employee Contracts
  async getEmployeeContracts(orgId: number, employeeId: number): Promise<EmployeeContract[]> {
    const response = await this.client.get<{ data: EmployeeContract[] }>(
      `/organizations/${orgId}/employees/${employeeId}/contracts`
    );
    return response.data.data;
  }

  async createEmployeeContract(
    orgId: number,
    employeeId: number,
    data: EmployeeContractCreateRequest
  ): Promise<EmployeeContract> {
    const response = await this.client.post<EmployeeContract>(
      `/organizations/${orgId}/employees/${employeeId}/contracts`,
      data
    );
    return response.data;
  }

  async updateEmployeeContract(
    orgId: number,
    employeeId: number,
    contractId: number,
    data: EmployeeContractUpdateRequest
  ): Promise<EmployeeContract> {
    const response = await this.client.put<EmployeeContract>(
      `/organizations/${orgId}/employees/${employeeId}/contracts/${contractId}`,
      data
    );
    return response.data;
  }

  async deleteEmployeeContract(
    orgId: number,
    employeeId: number,
    contractId: number
  ): Promise<void> {
    await this.client.delete(
      `/organizations/${orgId}/employees/${employeeId}/contracts/${contractId}`
    );
  }

  // Children (organization-scoped)
  private _children = this.orgScopedCrud<
    Child,
    Omit<ChildCreateRequest, 'organization_id'>,
    ChildUpdateRequest
  >('children');
  getChildren = this._children.list;
  getChild = this._children.get;
  createChild = this._children.create;
  updateChild = this._children.update;
  deleteChild = this._children.delete;

  // Child Contracts
  async getChildContracts(orgId: number, childId: number): Promise<ChildContract[]> {
    const response = await this.client.get<{ data: ChildContract[] }>(
      `/organizations/${orgId}/children/${childId}/contracts`
    );
    return response.data.data;
  }

  async createChildContract(
    orgId: number,
    childId: number,
    data: ChildContractCreateRequest
  ): Promise<ChildContract> {
    const response = await this.client.post<ChildContract>(
      `/organizations/${orgId}/children/${childId}/contracts`,
      data
    );
    return response.data;
  }

  async updateChildContract(
    orgId: number,
    childId: number,
    contractId: number,
    data: ChildContractUpdateRequest
  ): Promise<ChildContract> {
    const response = await this.client.put<ChildContract>(
      `/organizations/${orgId}/children/${childId}/contracts/${contractId}`,
      data
    );
    return response.data;
  }

  async deleteChildContract(orgId: number, childId: number, contractId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/children/${childId}/contracts/${contractId}`);
  }

  async getChildrenFunding(orgId: number, date?: string): Promise<ChildrenFundingResponse> {
    const params = date ? { date } : {};
    const response = await this.client.get<ChildrenFundingResponse>(
      `/organizations/${orgId}/children/funding`,
      { params }
    );
    return response.data;
  }

  async getChildrenContractCountByMonth(
    orgId: number,
    minYear?: number,
    maxYear?: number
  ): Promise<ChildrenContractCountByMonthResponse> {
    const params: { min_year?: number; max_year?: number } = {};
    if (minYear !== undefined) params.min_year = minYear;
    if (maxYear !== undefined) params.max_year = maxYear;
    const response = await this.client.get<ChildrenContractCountByMonthResponse>(
      `/organizations/${orgId}/children/statistics/contract-count-by-month`,
      { params }
    );
    return response.data;
  }

  async getAgeDistribution(orgId: number, date?: string): Promise<AgeDistributionResponse> {
    const params = date ? { date } : {};
    const response = await this.client.get<AgeDistributionResponse>(
      `/organizations/${orgId}/children/statistics/age-distribution`,
      { params }
    );
    return response.data;
  }

  // GovernmentFundings
  private _governmentFundings = this.topLevelCrud<
    GovernmentFunding,
    GovernmentFundingCreateRequest,
    GovernmentFundingUpdateRequest
  >('government-fundings');
  getGovernmentFundings = this._governmentFundings.list;
  createGovernmentFunding = this._governmentFundings.create;
  updateGovernmentFunding = this._governmentFundings.update;
  deleteGovernmentFunding = this._governmentFundings.delete;

  // Custom getGovernmentFunding with periodsLimit param
  async getGovernmentFunding(id: number, periodsLimit?: number): Promise<GovernmentFunding> {
    const params = periodsLimit !== undefined ? { periods_limit: periodsLimit } : {};
    const response = await this.client.get<GovernmentFunding>(`/government-fundings/${id}`, {
      params,
    });
    return response.data;
  }

  // GovernmentFunding Periods
  async createGovernmentFundingPeriod(
    governmentFundingId: number,
    data: GovernmentFundingPeriodCreateRequest
  ): Promise<GovernmentFundingPeriod> {
    const response = await this.client.post<GovernmentFundingPeriod>(
      `/government-fundings/${governmentFundingId}/periods`,
      data
    );
    return response.data;
  }

  async updateGovernmentFundingPeriod(
    governmentFundingId: number,
    periodId: number,
    data: GovernmentFundingPeriodUpdateRequest
  ): Promise<GovernmentFundingPeriod> {
    const response = await this.client.put<GovernmentFundingPeriod>(
      `/government-fundings/${governmentFundingId}/periods/${periodId}`,
      data
    );
    return response.data;
  }

  async deleteGovernmentFundingPeriod(
    governmentFundingId: number,
    periodId: number
  ): Promise<void> {
    await this.client.delete(`/government-fundings/${governmentFundingId}/periods/${periodId}`);
  }

  // GovernmentFunding Properties
  async createGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    data: GovernmentFundingPropertyCreateRequest
  ): Promise<GovernmentFundingProperty> {
    const response = await this.client.post<GovernmentFundingProperty>(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/properties`,
      data
    );
    return response.data;
  }

  async updateGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    propId: number,
    data: GovernmentFundingPropertyUpdateRequest
  ): Promise<GovernmentFundingProperty> {
    const response = await this.client.put<GovernmentFundingProperty>(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/properties/${propId}`,
      data
    );
    return response.data;
  }

  async deleteGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    propId: number
  ): Promise<void> {
    await this.client.delete(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/properties/${propId}`
    );
  }

  // PayPlans (organization-scoped)
  private _payPlans = this.orgScopedCrud<PayPlan, PayPlanCreateRequest, PayPlanUpdateRequest>(
    'payplans'
  );
  getPayPlans = this._payPlans.list;
  getPayPlan = this._payPlans.get;
  createPayPlan = this._payPlans.create;
  updatePayPlan = this._payPlans.update;
  deletePayPlan = this._payPlans.delete;

  // PayPlan Periods
  async createPayPlanPeriod(
    orgId: number,
    payplanId: number,
    data: PayPlanPeriodCreateRequest
  ): Promise<PayPlanPeriod> {
    const response = await this.client.post<PayPlanPeriod>(
      `/organizations/${orgId}/payplans/${payplanId}/periods`,
      data
    );
    return response.data;
  }

  async getPayPlanPeriod(
    orgId: number,
    payplanId: number,
    periodId: number
  ): Promise<PayPlanPeriod> {
    const response = await this.client.get<PayPlanPeriod>(
      `/organizations/${orgId}/payplans/${payplanId}/periods/${periodId}`
    );
    return response.data;
  }

  async updatePayPlanPeriod(
    orgId: number,
    payplanId: number,
    periodId: number,
    data: PayPlanPeriodUpdateRequest
  ): Promise<PayPlanPeriod> {
    const response = await this.client.put<PayPlanPeriod>(
      `/organizations/${orgId}/payplans/${payplanId}/periods/${periodId}`,
      data
    );
    return response.data;
  }

  async deletePayPlanPeriod(orgId: number, payplanId: number, periodId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/payplans/${payplanId}/periods/${periodId}`);
  }

  // PayPlan Entries
  async createPayPlanEntry(
    orgId: number,
    payplanId: number,
    periodId: number,
    data: PayPlanEntryCreateRequest
  ): Promise<PayPlanEntry> {
    const response = await this.client.post<PayPlanEntry>(
      `/organizations/${orgId}/payplans/${payplanId}/periods/${periodId}/entries`,
      data
    );
    return response.data;
  }

  async getPayPlanEntry(
    orgId: number,
    payplanId: number,
    periodId: number,
    entryId: number
  ): Promise<PayPlanEntry> {
    const response = await this.client.get<PayPlanEntry>(
      `/organizations/${orgId}/payplans/${payplanId}/periods/${periodId}/entries/${entryId}`
    );
    return response.data;
  }

  async updatePayPlanEntry(
    orgId: number,
    payplanId: number,
    periodId: number,
    entryId: number,
    data: PayPlanEntryUpdateRequest
  ): Promise<PayPlanEntry> {
    const response = await this.client.put<PayPlanEntry>(
      `/organizations/${orgId}/payplans/${payplanId}/periods/${periodId}/entries/${entryId}`,
      data
    );
    return response.data;
  }

  async deletePayPlanEntry(
    orgId: number,
    payplanId: number,
    periodId: number,
    entryId: number
  ): Promise<void> {
    await this.client.delete(
      `/organizations/${orgId}/payplans/${payplanId}/periods/${periodId}/entries/${entryId}`
    );
  }

  // Sections (organization-scoped)
  private _sections = this.orgScopedCrud<Section, SectionCreateRequest, SectionUpdateRequest>(
    'sections'
  );
  getSections = this._sections.list;
  getSection = this._sections.get;
  createSection = this._sections.create;
  updateSection = this._sections.update;
  deleteSection = this._sections.delete;

  // Employees - fetch all with active contracts (for kanban board view)
  async getEmployeesAll(orgId: number): Promise<Employee[]> {
    const today = new Date().toISOString().slice(0, 10);
    const response = await this.client.get<PaginatedResponse<Employee>>(
      `/organizations/${orgId}/employees?limit=100&active_on=${today}`
    );
    return response.data.data;
  }

  async getStepPromotions(orgId: number, date?: string): Promise<StepPromotionsResponse> {
    const params = date ? { date } : {};
    const response = await this.client.get<StepPromotionsResponse>(
      `/organizations/${orgId}/employees/step-promotions`,
      { params }
    );
    return response.data;
  }

  // Children - fetch upcoming (contracts starting after today)
  async getUpcomingChildren(orgId: number): Promise<Child[]> {
    const today = new Date().toISOString().slice(0, 10);
    const response = await this.client.get<PaginatedResponse<Child>>(
      `/organizations/${orgId}/children?limit=100&contract_after=${today}`
    );
    return response.data.data;
  }

  // Children - fetch all with active contracts (for kanban board view)
  async getChildrenAll(orgId: number): Promise<Child[]> {
    const today = new Date().toISOString().slice(0, 10);
    const response = await this.client.get<PaginatedResponse<Child>>(
      `/organizations/${orgId}/children?limit=100&contract_on=${today}`
    );
    return response.data.data;
  }
}

export const apiClient = new ApiClient();

// Helper to extract error message from API errors
export function getErrorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object' && 'response' in error) {
    const axiosError = error as { response?: { data?: { message?: string } } };
    if (axiosError.response?.data?.message) {
      return axiosError.response.data.message;
    }
  }
  return fallback;
}
