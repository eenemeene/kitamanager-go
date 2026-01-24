// API Types - matching the Go backend models

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

// Organization
export interface Organization {
  id: number
  name: string
  active: boolean
  created_at: string
  created_by: string
  updated_at: string
  users?: User[]
  groups?: Group[]
}

export interface OrganizationCreate {
  name: string
  active?: boolean
}

export interface OrganizationUpdate {
  name?: string
  active?: boolean
}

// User
export interface User {
  id: number
  name: string
  email: string
  active: boolean
  last_login?: string | null
  created_at: string
  created_by: string
  updated_at: string
  organizations?: Organization[]
  groups?: Group[]
}

export interface UserCreate {
  name: string
  email: string
  password: string
  active?: boolean
}

export interface UserUpdate {
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

export interface GroupCreate {
  name: string
  organization_id: number
  active?: boolean
}

export interface GroupUpdate {
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

export interface EmployeeCreate {
  organization_id: number
  first_name: string
  last_name: string
  birthdate: string
}

export interface EmployeeUpdate {
  first_name?: string
  last_name?: string
  birthdate?: string
}

export interface EmployeeContractCreate {
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
  created_at: string
}

export interface ChildCreate {
  organization_id: number
  first_name: string
  last_name: string
  birthdate: string
}

export interface ChildUpdate {
  first_name?: string
  last_name?: string
  birthdate?: string
}

export interface ChildContractCreate {
  from: string
  to?: string | null
  care_hours_per_week: number
  group_id?: number | null
  meals_included?: boolean
  special_needs?: string
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
