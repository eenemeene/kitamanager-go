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

// Helper to get CSRF token from cookie
function getCSRFToken(): string | null {
  if (typeof document === 'undefined') return null;
  const match = document.cookie.match(/csrf_token=([^;]+)/);
  return match ? match[1] : null;
}

class ApiClient {
  private client: AxiosInstance;
  private onUnauthorized?: () => void;

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

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        if (error.response?.status === 401) {
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

  // Auth
  async login(request: LoginRequest): Promise<LoginResponse> {
    const response = await this.client.post<LoginResponse>('/login', request);
    return response.data;
  }

  async logout(): Promise<void> {
    await this.client.post('/logout');
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.client.get<User>('/me');
    return response.data;
  }

  // Organizations
  async getOrganizations(params: PaginationParams = {}): Promise<PaginatedResponse<Organization>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<Organization>>(
      `/organizations?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  async getOrganizationsAll(): Promise<Organization[]> {
    // Backend max limit is 100
    const response = await this.client.get<PaginatedResponse<Organization>>(
      '/organizations?limit=100'
    );
    return response.data.data;
  }

  async getOrganization(id: number): Promise<Organization> {
    const response = await this.client.get<Organization>(`/organizations/${id}`);
    return response.data;
  }

  async createOrganization(data: OrganizationCreateRequest): Promise<Organization> {
    const response = await this.client.post<Organization>('/organizations', data);
    return response.data;
  }

  async updateOrganization(id: number, data: OrganizationUpdateRequest): Promise<Organization> {
    const response = await this.client.put<Organization>(`/organizations/${id}`, data);
    return response.data;
  }

  async deleteOrganization(id: number): Promise<void> {
    await this.client.delete(`/organizations/${id}`);
  }

  // Users
  async getUsers(params: PaginationParams = {}): Promise<PaginatedResponse<User>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<User>>(
      `/users?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  async getUser(id: number): Promise<User> {
    const response = await this.client.get<User>(`/users/${id}`);
    return response.data;
  }

  async createUser(data: UserCreateRequest): Promise<User> {
    const response = await this.client.post<User>('/users', data);
    return response.data;
  }

  async updateUser(id: number, data: UserUpdateRequest): Promise<User> {
    const response = await this.client.put<User>(`/users/${id}`, data);
    return response.data;
  }

  async deleteUser(id: number): Promise<void> {
    await this.client.delete(`/users/${id}`);
  }

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
  async getGroups(orgId: number, params: PaginationParams = {}): Promise<PaginatedResponse<Group>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<Group>>(
      `/organizations/${orgId}/groups?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  async getGroup(orgId: number, groupId: number): Promise<Group> {
    const response = await this.client.get<Group>(`/organizations/${orgId}/groups/${groupId}`);
    return response.data;
  }

  async createGroup(orgId: number, data: GroupCreateRequest): Promise<Group> {
    const response = await this.client.post<Group>(`/organizations/${orgId}/groups`, data);
    return response.data;
  }

  async updateGroup(orgId: number, groupId: number, data: GroupUpdateRequest): Promise<Group> {
    const response = await this.client.put<Group>(
      `/organizations/${orgId}/groups/${groupId}`,
      data
    );
    return response.data;
  }

  async deleteGroup(orgId: number, groupId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/groups/${groupId}`);
  }

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
  async getEmployees(
    orgId: number,
    params: PaginationParams = {}
  ): Promise<PaginatedResponse<Employee>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<Employee>>(
      `/organizations/${orgId}/employees?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  async getEmployee(orgId: number, id: number): Promise<Employee> {
    const response = await this.client.get<Employee>(`/organizations/${orgId}/employees/${id}`);
    return response.data;
  }

  async createEmployee(
    orgId: number,
    data: Omit<EmployeeCreateRequest, 'organization_id'>
  ): Promise<Employee> {
    const response = await this.client.post<Employee>(`/organizations/${orgId}/employees`, data);
    return response.data;
  }

  async updateEmployee(orgId: number, id: number, data: EmployeeUpdateRequest): Promise<Employee> {
    const response = await this.client.put<Employee>(
      `/organizations/${orgId}/employees/${id}`,
      data
    );
    return response.data;
  }

  async deleteEmployee(orgId: number, id: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/employees/${id}`);
  }

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
  async getChildren(
    orgId: number,
    params: PaginationParams = {}
  ): Promise<PaginatedResponse<Child>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<Child>>(
      `/organizations/${orgId}/children?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  async getChild(orgId: number, id: number): Promise<Child> {
    const response = await this.client.get<Child>(`/organizations/${orgId}/children/${id}`);
    return response.data;
  }

  async createChild(
    orgId: number,
    data: Omit<ChildCreateRequest, 'organization_id'>
  ): Promise<Child> {
    const response = await this.client.post<Child>(`/organizations/${orgId}/children`, data);
    return response.data;
  }

  async updateChild(orgId: number, id: number, data: ChildUpdateRequest): Promise<Child> {
    const response = await this.client.put<Child>(`/organizations/${orgId}/children/${id}`, data);
    return response.data;
  }

  async deleteChild(orgId: number, id: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/children/${id}`);
  }

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
  async getGovernmentFundings(
    params: PaginationParams = {}
  ): Promise<PaginatedResponse<GovernmentFunding>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<GovernmentFunding>>(
      `/government-fundings?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  async getGovernmentFunding(id: number, periodsLimit?: number): Promise<GovernmentFunding> {
    const params = periodsLimit !== undefined ? { periods_limit: periodsLimit } : {};
    const response = await this.client.get<GovernmentFunding>(`/government-fundings/${id}`, {
      params,
    });
    return response.data;
  }

  async createGovernmentFunding(data: GovernmentFundingCreateRequest): Promise<GovernmentFunding> {
    const response = await this.client.post<GovernmentFunding>('/government-fundings', data);
    return response.data;
  }

  async updateGovernmentFunding(
    id: number,
    data: GovernmentFundingUpdateRequest
  ): Promise<GovernmentFunding> {
    const response = await this.client.put<GovernmentFunding>(`/government-fundings/${id}`, data);
    return response.data;
  }

  async deleteGovernmentFunding(id: number): Promise<void> {
    await this.client.delete(`/government-fundings/${id}`);
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
  async getPayPlans(
    orgId: number,
    params: PaginationParams = {}
  ): Promise<PaginatedResponse<PayPlan>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<PayPlan>>(
      `/organizations/${orgId}/payplans?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  async getPayPlan(orgId: number, id: number): Promise<PayPlan> {
    const response = await this.client.get<PayPlan>(`/organizations/${orgId}/payplans/${id}`);
    return response.data;
  }

  async createPayPlan(orgId: number, data: PayPlanCreateRequest): Promise<PayPlan> {
    const response = await this.client.post<PayPlan>(`/organizations/${orgId}/payplans`, data);
    return response.data;
  }

  async updatePayPlan(orgId: number, id: number, data: PayPlanUpdateRequest): Promise<PayPlan> {
    const response = await this.client.put<PayPlan>(`/organizations/${orgId}/payplans/${id}`, data);
    return response.data;
  }

  async deletePayPlan(orgId: number, id: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/payplans/${id}`);
  }

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
  async getSections(
    orgId: number,
    params: PaginationParams = {}
  ): Promise<PaginatedResponse<Section>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE } = params;
    const response = await this.client.get<PaginatedResponse<Section>>(
      `/organizations/${orgId}/sections?page=${page}&limit=${limit}`
    );
    return response.data;
  }

  async getSection(orgId: number, sectionId: number): Promise<Section> {
    const response = await this.client.get<Section>(
      `/organizations/${orgId}/sections/${sectionId}`
    );
    return response.data;
  }

  async createSection(orgId: number, data: SectionCreateRequest): Promise<Section> {
    const response = await this.client.post<Section>(`/organizations/${orgId}/sections`, data);
    return response.data;
  }

  async updateSection(
    orgId: number,
    sectionId: number,
    data: SectionUpdateRequest
  ): Promise<Section> {
    const response = await this.client.put<Section>(
      `/organizations/${orgId}/sections/${sectionId}`,
      data
    );
    return response.data;
  }

  async deleteSection(orgId: number, sectionId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/sections/${sectionId}`);
  }

  // Children - fetch all (for kanban board view)
  async getChildrenAll(orgId: number): Promise<Child[]> {
    const response = await this.client.get<PaginatedResponse<Child>>(
      `/organizations/${orgId}/children?limit=500`
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
