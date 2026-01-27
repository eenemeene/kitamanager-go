// API Types - matching the Go backend models

// Roles for user-group membership
export type Role = 'admin' | 'manager' | 'member'

// Auth
export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
}

export interface ErrorResponse {
  error: string
}

export interface MessageResponse {
  message: string
}

// GovernmentFunding
export interface GovernmentFunding {
  id: number
  name: string
  created_at: string
  updated_at: string
  periods?: GovernmentFundingPeriod[]
}

export interface GovernmentFundingPeriod {
  id: number
  government_funding_id: number
  from: string
  to?: string | null
  comment?: string
  created_at: string
  entries?: GovernmentFundingEntry[]
}

export interface GovernmentFundingEntry {
  id: number
  period_id: number
  min_age: number
  max_age: number
  created_at: string
  properties?: GovernmentFundingProperty[]
}

export interface GovernmentFundingProperty {
  id: number
  entry_id: number
  name: string
  payment: number
  requirement: number
  comment?: string
  created_at: string
}

export interface GovernmentFundingCreateRequest {
  name: string
}

export interface GovernmentFundingUpdateRequest {
  name?: string
}

export interface GovernmentFundingPeriodCreateRequest {
  from: string
  to?: string | null
  comment?: string
}

export interface GovernmentFundingPeriodUpdateRequest {
  from?: string
  to?: string | null
  comment?: string
}

export interface GovernmentFundingEntryCreateRequest {
  min_age: number
  max_age: number
}

export interface GovernmentFundingEntryUpdateRequest {
  min_age?: number
  max_age?: number
}

export interface GovernmentFundingPropertyCreateRequest {
  name: string
  payment: number
  requirement: number
  comment?: string
}

export interface GovernmentFundingPropertyUpdateRequest {
  name?: string
  payment?: number
  requirement?: number
  comment?: string
}

export interface AssignGovernmentFundingRequest {
  government_funding_id: number
}

// Organization
export interface Organization {
  id: number
  name: string
  active: boolean
  government_funding_id?: number | null
  government_funding?: GovernmentFunding
  created_at: string
  created_by: string
  updated_at: string
  users?: User[]
  groups?: Group[]
}

export interface OrganizationCreateRequest {
  name: string
  active?: boolean
}

export interface OrganizationUpdateRequest {
  name?: string
  active?: boolean
}

// User
export interface User {
  id: number
  name: string
  email: string
  active: boolean
  is_superadmin: boolean
  last_login?: string | null
  created_at: string
  created_by: string
  updated_at: string
  organizations?: Organization[]
  groups?: Group[]
}

export interface UserCreateRequest {
  name: string
  email: string
  password: string
  active?: boolean
}

export interface UserUpdateRequest {
  name?: string
  email?: string
  active?: boolean
}

// Group (each group belongs to exactly one organization)
export interface Group {
  id: number
  name: string
  organization_id: number
  organization?: Organization
  active: boolean
  created_at: string
  created_by: string
  updated_at: string
  users?: User[]
}

export interface GroupCreateRequest {
  name: string
  active?: boolean
}

export interface GroupUpdateRequest {
  name?: string
  active?: boolean
}

// Person (base for Employee and Child)
export interface Person {
  id: number
  organization_id: number
  organization?: Organization
  first_name: string
  last_name: string
  birthdate: string
  created_at: string
  updated_at: string
}

// Employee
export interface Employee extends Person {
  contracts?: EmployeeContract[]
}

export interface EmployeeContract {
  id: number
  employee_id: number
  from: string
  to?: string | null
  position: string
  weekly_hours: number
  salary: number
  created_at: string
}

export interface EmployeeCreateRequest {
  organization_id: number
  first_name: string
  last_name: string
  birthdate: string
}

export interface EmployeeUpdateRequest {
  first_name?: string
  last_name?: string
  birthdate?: string
}

export interface EmployeeContractCreateRequest {
  from: string
  to?: string | null
  position: string
  weekly_hours: number
  salary: number
}

// Child
export interface Child extends Person {
  contracts?: ChildContract[]
}

export interface ChildContract {
  id: number
  child_id: number
  from: string
  to?: string | null
  care_hours_per_week: number
  group_id?: number | null
  meals_included: boolean
  special_needs: string
  attributes?: string[]
  created_at: string
}

export interface ChildCreateRequest {
  organization_id: number
  first_name: string
  last_name: string
  birthdate: string
}

export interface ChildUpdateRequest {
  first_name?: string
  last_name?: string
  birthdate?: string
}

export interface ChildContractCreateRequest {
  from: string
  to?: string | null
  care_hours_per_week: number
  group_id?: number | null
  meals_included?: boolean
  special_needs?: string
  attributes?: string[]
}

// Pagination response wrapper
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}

// Dashboard stats
export interface DashboardStats {
  total_organizations: number
  total_employees: number
  total_children: number
  total_users: number
}

// User-Group membership with role
export interface UserGroupResponse {
  user_id: number
  group_id: number
  role: Role
  created_at: string
  created_by: string
  group?: Group
}

// User membership for memberships endpoint
export interface UserMembership {
  user_id: number
  group_id: number
  role: Role
  group: Group
  effective_org_role: Role
}

// Response for user memberships
export interface UserMembershipsResponse {
  memberships: UserMembership[]
}

// Request to add user to group
export interface AddUserToGroupRequest {
  group_id: number
  role: Role
}

// Request to update user's role in a group
export interface UpdateUserGroupRoleRequest {
  role: Role
}

// Request to set superadmin status
export interface SetSuperAdminRequest {
  is_superadmin: boolean
}
