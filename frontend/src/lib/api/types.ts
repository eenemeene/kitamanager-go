// API Types - matching the Go backend models

// Gender type
export type Gender = 'male' | 'female' | 'diverse';

// Roles for user-group membership
export type Role = 'admin' | 'manager' | 'member';

// Auth
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  refresh_token?: string;
  expires_in?: number;
  user?: {
    id: number;
    email: string;
  };
}

export interface ErrorResponse {
  code: string;
  message: string;
}

export interface MessageResponse {
  message: string;
}

// GovernmentFunding
export interface GovernmentFunding {
  id: number;
  name: string;
  state: string;
  created_at: string;
  updated_at: string;
  periods?: GovernmentFundingPeriod[];
  total_periods?: number;
}

export interface GovernmentFundingPeriod {
  id: number;
  government_funding_id: number;
  from: string;
  to?: string | null;
  comment?: string;
  created_at: string;
  properties?: GovernmentFundingProperty[];
}

export interface GovernmentFundingProperty {
  id: number;
  period_id: number;
  key: string;
  value: string;
  payment: number;
  requirement: number;
  min_age?: number | null;
  max_age?: number | null;
  comment?: string;
  created_at: string;
}

export interface GovernmentFundingCreateRequest {
  name: string;
  state: string;
}

export interface GovernmentFundingUpdateRequest {
  name?: string;
}

export interface GovernmentFundingPeriodCreateRequest {
  from: string;
  to?: string | null;
  comment?: string;
}

export interface GovernmentFundingPeriodUpdateRequest {
  from?: string;
  to?: string | null;
  comment?: string;
}

export interface GovernmentFundingPropertyCreateRequest {
  key: string;
  value: string;
  payment: number;
  requirement: number;
  min_age?: number | null;
  max_age?: number | null;
  comment?: string;
}

export interface GovernmentFundingPropertyUpdateRequest {
  key?: string;
  value?: string;
  payment?: number;
  requirement?: number;
  min_age?: number | null;
  max_age?: number | null;
  comment?: string;
}

// Organization
export interface Organization {
  id: number;
  name: string;
  active: boolean;
  state: string;
  created_at: string;
  created_by: string;
  updated_at: string;
  users?: User[];
  groups?: Group[];
}

export interface OrganizationCreateRequest {
  name: string;
  active?: boolean;
  state: string;
}

export interface OrganizationUpdateRequest {
  name?: string;
  active?: boolean;
  state?: string;
}

// User
export interface User {
  id: number;
  name: string;
  email: string;
  active: boolean;
  is_superadmin: boolean;
  last_login?: string | null;
  created_at: string;
  created_by: string;
  updated_at: string;
  organizations?: Organization[];
  groups?: Group[];
}

export interface UserCreateRequest {
  name: string;
  email: string;
  password: string;
  active?: boolean;
}

export interface UserUpdateRequest {
  name?: string;
  email?: string;
  active?: boolean;
}

// Group (each group belongs to exactly one organization)
export interface Group {
  id: number;
  name: string;
  organization_id: number;
  organization?: Organization;
  active: boolean;
  created_at: string;
  created_by: string;
  updated_at: string;
  users?: User[];
}

export interface GroupCreateRequest {
  name: string;
  active?: boolean;
}

export interface GroupUpdateRequest {
  name?: string;
  active?: boolean;
}

// Person (base for Employee and Child)
export interface Person {
  id: number;
  organization_id: number;
  organization?: Organization;
  first_name: string;
  last_name: string;
  gender: Gender;
  birthdate: string;
  created_at: string;
  updated_at: string;
}

// Employee
export interface Employee extends Person {
  section_id?: number | null;
  section?: Section | null;
  contracts?: EmployeeContract[];
}

export interface EmployeeContract {
  id: number;
  employee_id: number;
  from: string;
  to?: string | null;
  staff_category: string;
  grade: string;
  step: number;
  weekly_hours: number;
  payplan_id: number;
  created_at: string;
  updated_at: string;
}

export interface EmployeeCreateRequest {
  organization_id: number;
  first_name: string;
  last_name: string;
  gender: Gender;
  birthdate: string;
}

export interface EmployeeUpdateRequest {
  first_name?: string;
  last_name?: string;
  gender?: Gender;
  birthdate?: string;
  section_id?: number | null;
}

export interface EmployeeContractCreateRequest {
  from: string;
  to?: string | null;
  staff_category: string;
  grade: string;
  step: number;
  weekly_hours: number;
  payplan_id: number;
}

export interface EmployeeContractUpdateRequest {
  from?: string;
  to?: string | null;
  staff_category?: string;
  grade?: string;
  step?: number;
  weekly_hours?: number;
  payplan_id?: number;
}

// Section
export interface Section {
  id: number;
  organization_id: number;
  name: string;
  is_default: boolean;
  min_age_months?: number | null;
  max_age_months?: number | null;
  created_at: string;
  created_by: string;
  updated_at: string;
}

export interface SectionCreateRequest {
  name: string;
  min_age_months?: number | null;
  max_age_months?: number | null;
}

export interface SectionUpdateRequest {
  name?: string;
  min_age_months?: number | null;
  max_age_months?: number | null;
}

// Child
export interface Child extends Person {
  section_id?: number | null;
  section?: Section | null;
  contracts?: ChildContract[];
}

// ContractProperties is a map of property keys to values.
// Values can be strings (scalar) or string arrays.
// Example: {"care_type": "ganztag", "supplements": ["ndh", "mss"]}
export type ContractProperties = Record<string, string | string[]>;

export interface ChildContract {
  id: number;
  child_id: number;
  from: string;
  to?: string | null;
  properties?: ContractProperties;
  created_at: string;
  updated_at: string;
}

export interface ChildCreateRequest {
  organization_id: number;
  first_name: string;
  last_name: string;
  gender: Gender;
  birthdate: string;
  section_id?: number | null;
}

export interface ChildUpdateRequest {
  first_name?: string;
  last_name?: string;
  gender?: Gender;
  birthdate?: string;
  section_id?: number | null;
}

export interface ChildContractCreateRequest {
  from: string;
  to?: string | null;
  properties?: ContractProperties;
}

export interface ChildContractUpdateRequest {
  from?: string;
  to?: string | null;
  properties?: ContractProperties;
}

// Pagination response wrapper
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

// Pagination params for API calls
export interface PaginationParams {
  page?: number;
  limit?: number;
}

export const DEFAULT_PAGE_SIZE = 30;

// Dashboard stats
export interface DashboardStats {
  total_organizations: number;
  total_employees: number;
  total_children: number;
  total_users: number;
}

// User-Group membership with role
export interface UserGroupResponse {
  user_id: number;
  group_id: number;
  role: Role;
  created_at: string;
  created_by: string;
  group?: Group;
}

// User membership for memberships endpoint
export interface UserMembership {
  user_id: number;
  group_id: number;
  role: Role;
  group: Group;
  effective_org_role: Role;
}

// Response for user memberships
export interface UserMembershipsResponse {
  memberships: UserMembership[];
}

// Request to add user to group
export interface AddUserToGroupRequest {
  group_id: number;
  role: Role;
}

// Request to update user's role in a group
export interface UpdateUserGroupRoleRequest {
  role: Role;
}

// Request to set superadmin status
export interface SetSuperAdminRequest {
  is_superadmin: boolean;
}

// Child funding calculation
export interface ChildFundingMatchedProp {
  key: string;
  value: string;
}

export interface ChildFundingResponse {
  child_id: number;
  child_name: string;
  age: number;
  funding: number;
  requirement: number;
  matched_properties: ChildFundingMatchedProp[];
  unmatched_properties: ChildFundingMatchedProp[];
}

export interface ChildrenFundingResponse {
  date: string;
  weekly_hours_basis: number;
  children: ChildFundingResponse[];
}

// Children contract count by month
export interface ChildrenContractCountByMonthResponse {
  period: {
    start: string;
    end: string;
  };
  years: ContractCountByMonthYear[];
}

export interface ContractCountByMonthYear {
  year: number;
  counts: number[]; // 12 values, one per month (Jan=0, Dec=11)
}

// Age distribution
export interface AgeDistributionResponse {
  date: string;
  total_count: number;
  distribution: AgeDistributionBucket[];
}

export interface AgeDistributionBucket {
  age_label: string; // e.g., "0", "1", "2", "3", "4", "5", "6+"
  min_age: number;
  max_age?: number | null; // null for open-ended (6+)
  count: number;
  male_count: number;
  female_count: number;
  diverse_count: number;
}

// PayPlan (organization-scoped salary tables)
export interface PayPlan {
  id: number;
  organization_id: number;
  name: string;
  created_at: string;
  updated_at: string;
  periods?: PayPlanPeriod[];
  total_periods?: number;
}

export interface PayPlanPeriod {
  id: number;
  payplan_id: number;
  from: string;
  to?: string | null;
  weekly_hours: number;
  created_at: string;
  updated_at: string;
  entries?: PayPlanEntry[];
}

export interface PayPlanEntry {
  id: number;
  period_id: number;
  grade: string;
  step: number;
  monthly_amount: number; // cents
  step_min_years?: number | null;
  created_at: string;
  updated_at: string;
}

export interface PayPlanCreateRequest {
  name: string;
}

export interface PayPlanUpdateRequest {
  name?: string;
}

export interface PayPlanPeriodCreateRequest {
  from: string;
  to?: string | null;
  weekly_hours: number;
}

export interface PayPlanPeriodUpdateRequest {
  from?: string;
  to?: string | null;
  weekly_hours?: number;
}

export interface PayPlanEntryCreateRequest {
  grade: string;
  step: number;
  monthly_amount: number;
  step_min_years?: number | null;
}

export interface PayPlanEntryUpdateRequest {
  grade?: string;
  step?: number;
  monthly_amount?: number;
  step_min_years?: number | null;
}

// Step Promotions
export interface StepPromotion {
  employee_id: number;
  employee_name: string;
  current_step: number;
  eligible_step: number;
  years_of_service: number;
  service_start: string;
  grade: string;
  current_amount: number;
  new_amount: number;
  monthly_cost_delta: number;
  payplan_id: number;
  payplan_name: string;
}

export interface StepPromotionsResponse {
  date: string;
  total_monthly_cost_delta: number;
  promotions: StepPromotion[];
}
