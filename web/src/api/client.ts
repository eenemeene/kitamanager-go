import axios, { type AxiosInstance, type AxiosError } from 'axios'
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
  Child,
  ChildCreateRequest,
  ChildUpdateRequest,
  ChildContract,
  ChildContractCreateRequest,
  ChildContractUpdateRequest,
  Role,
  UserGroupResponse,
  UserMembershipsResponse,
  GovernmentFunding,
  GovernmentFundingCreateRequest,
  GovernmentFundingUpdateRequest,
  GovernmentFundingPeriod,
  GovernmentFundingPeriodCreateRequest,
  GovernmentFundingPeriodUpdateRequest,
  GovernmentFundingEntry,
  GovernmentFundingEntryCreateRequest,
  GovernmentFundingEntryUpdateRequest,
  GovernmentFundingProperty,
  GovernmentFundingPropertyCreateRequest,
  GovernmentFundingPropertyUpdateRequest
} from './types'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

class ApiClient {
  private client: AxiosInstance
  private onUnauthorized?: () => void

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json'
      }
    })

    // Request interceptor to add auth token
    this.client.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('token')
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => Promise.reject(error)
    )

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        if (error.response?.status === 401) {
          localStorage.removeItem('token')
          if (this.onUnauthorized) {
            this.onUnauthorized()
          }
        }
        return Promise.reject(error)
      }
    )
  }

  setOnUnauthorized(callback: () => void) {
    this.onUnauthorized = callback
  }

  // Auth
  async login(request: LoginRequest): Promise<LoginResponse> {
    const response = await this.client.post<LoginResponse>('/login', request)
    return response.data
  }

  // Organizations
  async getOrganizations(): Promise<Organization[]> {
    // Request maximum limit to get all orgs for sidebar dropdown
    const response = await this.client.get<{ data: Organization[] }>('/organizations?limit=100')
    return response.data.data
  }

  async getOrganization(id: number): Promise<Organization> {
    const response = await this.client.get<Organization>(`/organizations/${id}`)
    return response.data
  }

  async createOrganization(data: OrganizationCreateRequest): Promise<Organization> {
    const response = await this.client.post<Organization>('/organizations', data)
    return response.data
  }

  async updateOrganization(id: number, data: OrganizationUpdateRequest): Promise<Organization> {
    const response = await this.client.put<Organization>(`/organizations/${id}`, data)
    return response.data
  }

  async deleteOrganization(id: number): Promise<void> {
    await this.client.delete(`/organizations/${id}`)
  }

  // Users
  async getUsers(): Promise<User[]> {
    const response = await this.client.get<{ data: User[] }>('/users')
    return response.data.data
  }

  async getUser(id: number): Promise<User> {
    const response = await this.client.get<User>(`/users/${id}`)
    return response.data
  }

  async createUser(data: UserCreateRequest): Promise<User> {
    const response = await this.client.post<User>('/users', data)
    return response.data
  }

  async updateUser(id: number, data: UserUpdateRequest): Promise<User> {
    const response = await this.client.put<User>(`/users/${id}`, data)
    return response.data
  }

  async deleteUser(id: number): Promise<void> {
    await this.client.delete(`/users/${id}`)
  }

  // User-Group assignments with roles
  async addUserToGroup(userId: number, groupId: number, role: Role): Promise<UserGroupResponse> {
    const response = await this.client.post<UserGroupResponse>(`/users/${userId}/groups`, {
      group_id: groupId,
      role
    })
    return response.data
  }

  async removeUserFromGroup(userId: number, groupId: number): Promise<void> {
    await this.client.delete(`/users/${userId}/groups/${groupId}`)
  }

  async updateUserGroupRole(
    userId: number,
    groupId: number,
    role: Role
  ): Promise<UserGroupResponse> {
    const response = await this.client.put<UserGroupResponse>(
      `/users/${userId}/groups/${groupId}`,
      {
        role
      }
    )
    return response.data
  }

  async getUserMemberships(userId: number): Promise<UserMembershipsResponse> {
    const response = await this.client.get<UserMembershipsResponse>(`/users/${userId}/memberships`)
    return response.data
  }

  async setSuperAdmin(userId: number, isSuperAdmin: boolean): Promise<User> {
    const response = await this.client.put<User>(`/users/${userId}/superadmin`, {
      is_superadmin: isSuperAdmin
    })
    return response.data
  }

  // User-Organization assignments
  async addUserToOrganization(userId: number, organizationId: number): Promise<void> {
    await this.client.post(`/users/${userId}/organizations`, { organization_id: organizationId })
  }

  async removeUserFromOrganization(userId: number, organizationId: number): Promise<void> {
    await this.client.delete(`/users/${userId}/organizations/${organizationId}`)
  }

  // Groups (organization-scoped)
  async getGroups(orgId: number): Promise<Group[]> {
    const response = await this.client.get<{ data: Group[] }>(`/organizations/${orgId}/groups`)
    return response.data.data
  }

  async getGroup(orgId: number, groupId: number): Promise<Group> {
    const response = await this.client.get<Group>(`/organizations/${orgId}/groups/${groupId}`)
    return response.data
  }

  async createGroup(orgId: number, data: GroupCreateRequest): Promise<Group> {
    const response = await this.client.post<Group>(`/organizations/${orgId}/groups`, data)
    return response.data
  }

  async updateGroup(orgId: number, groupId: number, data: GroupUpdateRequest): Promise<Group> {
    const response = await this.client.put<Group>(`/organizations/${orgId}/groups/${groupId}`, data)
    return response.data
  }

  async deleteGroup(orgId: number, groupId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/groups/${groupId}`)
  }

  // Organization users
  async getOrganizationUsers(orgId: number): Promise<User[]> {
    const response = await this.client.get<{ data: User[] }>(`/organizations/${orgId}/users`)
    return response.data.data
  }

  // Employees (organization-scoped)
  async getEmployees(orgId: number): Promise<Employee[]> {
    const response = await this.client.get<{ data: Employee[] }>(
      `/organizations/${orgId}/employees`
    )
    return response.data.data
  }

  async getEmployee(orgId: number, id: number): Promise<Employee> {
    const response = await this.client.get<Employee>(`/organizations/${orgId}/employees/${id}`)
    return response.data
  }

  async createEmployee(
    orgId: number,
    data: Omit<EmployeeCreateRequest, 'organization_id'>
  ): Promise<Employee> {
    const response = await this.client.post<Employee>(`/organizations/${orgId}/employees`, data)
    return response.data
  }

  async updateEmployee(orgId: number, id: number, data: EmployeeUpdateRequest): Promise<Employee> {
    const response = await this.client.put<Employee>(
      `/organizations/${orgId}/employees/${id}`,
      data
    )
    return response.data
  }

  async deleteEmployee(orgId: number, id: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/employees/${id}`)
  }

  // Employee Contracts
  async getEmployeeContracts(orgId: number, employeeId: number): Promise<EmployeeContract[]> {
    const response = await this.client.get<EmployeeContract[]>(
      `/organizations/${orgId}/employees/${employeeId}/contracts`
    )
    return response.data
  }

  async createEmployeeContract(
    orgId: number,
    employeeId: number,
    data: EmployeeContractCreateRequest
  ): Promise<EmployeeContract> {
    const response = await this.client.post<EmployeeContract>(
      `/organizations/${orgId}/employees/${employeeId}/contracts`,
      data
    )
    return response.data
  }

  async deleteEmployeeContract(
    orgId: number,
    employeeId: number,
    contractId: number
  ): Promise<void> {
    await this.client.delete(
      `/organizations/${orgId}/employees/${employeeId}/contracts/${contractId}`
    )
  }

  // Children (organization-scoped)
  async getChildren(orgId: number): Promise<Child[]> {
    const response = await this.client.get<{ data: Child[] }>(`/organizations/${orgId}/children`)
    return response.data.data
  }

  async getChild(orgId: number, id: number): Promise<Child> {
    const response = await this.client.get<Child>(`/organizations/${orgId}/children/${id}`)
    return response.data
  }

  async createChild(
    orgId: number,
    data: Omit<ChildCreateRequest, 'organization_id'>
  ): Promise<Child> {
    const response = await this.client.post<Child>(`/organizations/${orgId}/children`, data)
    return response.data
  }

  async updateChild(orgId: number, id: number, data: ChildUpdateRequest): Promise<Child> {
    const response = await this.client.put<Child>(`/organizations/${orgId}/children/${id}`, data)
    return response.data
  }

  async deleteChild(orgId: number, id: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/children/${id}`)
  }

  // Child Contracts
  async getChildContracts(orgId: number, childId: number): Promise<ChildContract[]> {
    const response = await this.client.get<ChildContract[]>(
      `/organizations/${orgId}/children/${childId}/contracts`
    )
    return response.data
  }

  async createChildContract(
    orgId: number,
    childId: number,
    data: ChildContractCreateRequest
  ): Promise<ChildContract> {
    const response = await this.client.post<ChildContract>(
      `/organizations/${orgId}/children/${childId}/contracts`,
      data
    )
    return response.data
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
    )
    return response.data
  }

  async deleteChildContract(orgId: number, childId: number, contractId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/children/${childId}/contracts/${contractId}`)
  }

  // GovernmentFundings
  async getGovernmentFundings(): Promise<GovernmentFunding[]> {
    const response = await this.client.get<{ data: GovernmentFunding[] }>('/government-fundings')
    return response.data.data
  }

  async getGovernmentFunding(id: number): Promise<GovernmentFunding> {
    const response = await this.client.get<GovernmentFunding>(`/government-fundings/${id}`)
    return response.data
  }

  async createGovernmentFunding(data: GovernmentFundingCreateRequest): Promise<GovernmentFunding> {
    const response = await this.client.post<GovernmentFunding>('/government-fundings', data)
    return response.data
  }

  async updateGovernmentFunding(
    id: number,
    data: GovernmentFundingUpdateRequest
  ): Promise<GovernmentFunding> {
    const response = await this.client.put<GovernmentFunding>(`/government-fundings/${id}`, data)
    return response.data
  }

  async deleteGovernmentFunding(id: number): Promise<void> {
    await this.client.delete(`/government-fundings/${id}`)
  }

  // GovernmentFunding Periods
  async createGovernmentFundingPeriod(
    governmentFundingId: number,
    data: GovernmentFundingPeriodCreateRequest
  ): Promise<GovernmentFundingPeriod> {
    const response = await this.client.post<GovernmentFundingPeriod>(
      `/government-fundings/${governmentFundingId}/periods`,
      data
    )
    return response.data
  }

  async updateGovernmentFundingPeriod(
    governmentFundingId: number,
    periodId: number,
    data: GovernmentFundingPeriodUpdateRequest
  ): Promise<GovernmentFundingPeriod> {
    const response = await this.client.put<GovernmentFundingPeriod>(
      `/government-fundings/${governmentFundingId}/periods/${periodId}`,
      data
    )
    return response.data
  }

  async deleteGovernmentFundingPeriod(
    governmentFundingId: number,
    periodId: number
  ): Promise<void> {
    await this.client.delete(`/government-fundings/${governmentFundingId}/periods/${periodId}`)
  }

  // GovernmentFunding Entries
  async createGovernmentFundingEntry(
    governmentFundingId: number,
    periodId: number,
    data: GovernmentFundingEntryCreateRequest
  ): Promise<GovernmentFundingEntry> {
    const response = await this.client.post<GovernmentFundingEntry>(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/entries`,
      data
    )
    return response.data
  }

  async updateGovernmentFundingEntry(
    governmentFundingId: number,
    periodId: number,
    entryId: number,
    data: GovernmentFundingEntryUpdateRequest
  ): Promise<GovernmentFundingEntry> {
    const response = await this.client.put<GovernmentFundingEntry>(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/entries/${entryId}`,
      data
    )
    return response.data
  }

  async deleteGovernmentFundingEntry(
    governmentFundingId: number,
    periodId: number,
    entryId: number
  ): Promise<void> {
    await this.client.delete(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/entries/${entryId}`
    )
  }

  // GovernmentFunding Properties
  async createGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    entryId: number,
    data: GovernmentFundingPropertyCreateRequest
  ): Promise<GovernmentFundingProperty> {
    const response = await this.client.post<GovernmentFundingProperty>(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/entries/${entryId}/properties`,
      data
    )
    return response.data
  }

  async updateGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    entryId: number,
    propId: number,
    data: GovernmentFundingPropertyUpdateRequest
  ): Promise<GovernmentFundingProperty> {
    const response = await this.client.put<GovernmentFundingProperty>(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/entries/${entryId}/properties/${propId}`,
      data
    )
    return response.data
  }

  async deleteGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    entryId: number,
    propId: number
  ): Promise<void> {
    await this.client.delete(
      `/government-fundings/${governmentFundingId}/periods/${periodId}/entries/${entryId}/properties/${propId}`
    )
  }

  // Organization GovernmentFunding Assignment
  async assignGovernmentFundingToOrganization(
    orgId: number,
    governmentFundingId: number
  ): Promise<void> {
    await this.client.put(`/organizations/${orgId}/government-funding`, {
      government_funding_id: governmentFundingId
    })
  }

  async removeGovernmentFundingFromOrganization(orgId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/government-funding`)
  }
}

export const apiClient = new ApiClient()

// Helper to extract error message from API errors
export function getErrorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object' && 'response' in error) {
    const axiosError = error as { response?: { data?: { message?: string } } }
    if (axiosError.response?.data?.message) {
      return axiosError.response.data.message
    }
  }
  return fallback
}
