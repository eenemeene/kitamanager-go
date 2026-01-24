import axios, { type AxiosInstance, type AxiosError } from 'axios'
import type {
  LoginRequest,
  LoginResponse,
  Organization,
  OrganizationCreate,
  OrganizationUpdate,
  User,
  UserCreate,
  UserUpdate,
  Group,
  GroupCreate,
  GroupUpdate,
  Employee,
  EmployeeCreate,
  EmployeeUpdate,
  EmployeeContract,
  EmployeeContractCreate,
  Child,
  ChildCreate,
  ChildUpdate,
  ChildContract,
  ChildContractCreate
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
    const response = await this.client.get<Organization[]>('/organizations')
    return response.data
  }

  async getOrganization(id: number): Promise<Organization> {
    const response = await this.client.get<Organization>(`/organizations/${id}`)
    return response.data
  }

  async createOrganization(data: OrganizationCreate): Promise<Organization> {
    const response = await this.client.post<Organization>('/organizations', data)
    return response.data
  }

  async updateOrganization(id: number, data: OrganizationUpdate): Promise<Organization> {
    const response = await this.client.put<Organization>(`/organizations/${id}`, data)
    return response.data
  }

  async deleteOrganization(id: number): Promise<void> {
    await this.client.delete(`/organizations/${id}`)
  }

  // Users
  async getUsers(): Promise<User[]> {
    const response = await this.client.get<User[]>('/users')
    return response.data
  }

  async getUser(id: number): Promise<User> {
    const response = await this.client.get<User>(`/users/${id}`)
    return response.data
  }

  async createUser(data: UserCreate): Promise<User> {
    const response = await this.client.post<User>('/users', data)
    return response.data
  }

  async updateUser(id: number, data: UserUpdate): Promise<User> {
    const response = await this.client.put<User>(`/users/${id}`, data)
    return response.data
  }

  async deleteUser(id: number): Promise<void> {
    await this.client.delete(`/users/${id}`)
  }

  // User-Group assignments
  async addUserToGroup(userId: number, groupId: number): Promise<void> {
    await this.client.post(`/users/${userId}/groups`, { group_id: groupId })
  }

  async removeUserFromGroup(userId: number, groupId: number): Promise<void> {
    await this.client.delete(`/users/${userId}/groups/${groupId}`)
  }

  // User-Organization assignments
  async addUserToOrganization(userId: number, organizationId: number): Promise<void> {
    await this.client.post(`/users/${userId}/organizations`, { organization_id: organizationId })
  }

  async removeUserFromOrganization(userId: number, organizationId: number): Promise<void> {
    await this.client.delete(`/users/${userId}/organizations/${organizationId}`)
  }

  // Groups
  async getGroups(): Promise<Group[]> {
    const response = await this.client.get<Group[]>('/groups')
    return response.data
  }

  async getGroup(id: number): Promise<Group> {
    const response = await this.client.get<Group>(`/groups/${id}`)
    return response.data
  }

  async createGroup(data: GroupCreate): Promise<Group> {
    const response = await this.client.post<Group>('/groups', data)
    return response.data
  }

  async updateGroup(id: number, data: GroupUpdate): Promise<Group> {
    const response = await this.client.put<Group>(`/groups/${id}`, data)
    return response.data
  }

  async deleteGroup(id: number): Promise<void> {
    await this.client.delete(`/groups/${id}`)
  }

  // Employees (organization-scoped)
  async getEmployees(orgId: number): Promise<Employee[]> {
    const response = await this.client.get<Employee[]>(`/organizations/${orgId}/employees`)
    return response.data
  }

  async getEmployee(orgId: number, id: number): Promise<Employee> {
    const response = await this.client.get<Employee>(`/organizations/${orgId}/employees/${id}`)
    return response.data
  }

  async createEmployee(
    orgId: number,
    data: Omit<EmployeeCreate, 'organization_id'>
  ): Promise<Employee> {
    const response = await this.client.post<Employee>(`/organizations/${orgId}/employees`, data)
    return response.data
  }

  async updateEmployee(orgId: number, id: number, data: EmployeeUpdate): Promise<Employee> {
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
    data: EmployeeContractCreate
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
    const response = await this.client.get<Child[]>(`/organizations/${orgId}/children`)
    return response.data
  }

  async getChild(orgId: number, id: number): Promise<Child> {
    const response = await this.client.get<Child>(`/organizations/${orgId}/children/${id}`)
    return response.data
  }

  async createChild(orgId: number, data: Omit<ChildCreate, 'organization_id'>): Promise<Child> {
    const response = await this.client.post<Child>(`/organizations/${orgId}/children`, data)
    return response.data
  }

  async updateChild(orgId: number, id: number, data: ChildUpdate): Promise<Child> {
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
    data: ChildContractCreate
  ): Promise<ChildContract> {
    const response = await this.client.post<ChildContract>(
      `/organizations/${orgId}/children/${childId}/contracts`,
      data
    )
    return response.data
  }

  async deleteChildContract(orgId: number, childId: number, contractId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/children/${childId}/contracts/${contractId}`)
  }
}

export const apiClient = new ApiClient()
