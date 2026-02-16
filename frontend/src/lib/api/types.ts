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
}

export interface GovernmentFundingPeriod {
  id: number;
  government_funding_id: number;
  from: string;
  to?: string | null;
  full_time_weekly_hours: number;
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
  full_time_weekly_hours: number;
  comment?: string;
}

export interface GovernmentFundingPeriodUpdateRequest {
  from?: string;
  to?: string | null;
  full_time_weekly_hours?: number;
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
  default_section_name: string;
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
  contracts?: EmployeeContract[];
}

export interface EmployeeContract {
  id: number;
  employee_id: number;
  from: string;
  to?: string | null;
  section_id: number;
  section_name?: string | null;
  staff_category: string;
  grade: string;
  step: number;
  weekly_hours: number;
  payplan_id: number;
  properties?: ContractProperties;
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
}

export interface EmployeeContractCreateRequest {
  from: string;
  to?: string | null;
  section_id: number;
  staff_category: string;
  grade: string;
  step: number;
  weekly_hours: number;
  payplan_id: number;
  properties?: ContractProperties;
}

export interface EmployeeContractUpdateRequest {
  from?: string;
  to?: string | null;
  section_id?: number | null;
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
  section_id: number;
  section_name?: string | null;
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
}

export interface ChildUpdateRequest {
  first_name?: string;
  last_name?: string;
  gender?: Gender;
  birthdate?: string;
}

export interface ChildContractCreateRequest {
  from: string;
  to?: string | null;
  section_id: number;
  properties?: ContractProperties;
}

export interface ChildContractUpdateRequest {
  from?: string;
  to?: string | null;
  section_id?: number | null;
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

// Pagination params for API calls (index signature allows arbitrary filter params)
export interface PaginationParams {
  page?: number;
  limit?: number;
  [key: string]: string | number | boolean | undefined;
}

export const DEFAULT_PAGE_SIZE = 30;

/** Fetch limit for lookup/dropdown data (sections, pay plans, etc.) where all items are needed. */
export const LOOKUP_FETCH_LIMIT = 100;

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
  employer_contribution_rate: number; // hundredths of percent: 2200 = 22.00%
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
  employer_contribution_rate: number;
}

export interface PayPlanPeriodUpdateRequest {
  from?: string;
  to?: string | null;
  weekly_hours?: number;
  employer_contribution_rate?: number;
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

// BudgetItem (organization-scoped income/expense categories)
export interface BudgetItem {
  id: number;
  organization_id: number;
  name: string;
  category: 'income' | 'expense';
  per_child: boolean;
  entries?: BudgetItemEntry[];
  created_at: string;
  updated_at: string;
}

export interface BudgetItemEntry {
  id: number;
  budget_item_id: number;
  from: string;
  to?: string | null;
  amount_cents: number;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface BudgetItemCreateRequest {
  name: string;
  category: string;
  per_child: boolean;
}

export interface BudgetItemUpdateRequest {
  name: string;
  category: string;
  per_child: boolean;
}

export interface BudgetItemEntryCreateRequest {
  from: string;
  to?: string | null;
  amount_cents: number;
  notes?: string;
}

export interface BudgetItemEntryUpdateRequest {
  from: string;
  to?: string | null;
  amount_cents: number;
  notes?: string;
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

// Financial statistics
export interface FinancialDataPoint {
  date: string;
  funding_income: number;
  gross_salary: number;
  employer_costs: number;
  budget_income: number;
  budget_expenses: number;
  total_income: number;
  total_expenses: number;
  balance: number;
  child_count: number;
  staff_count: number;
}

export interface FinancialResponse {
  data_points: FinancialDataPoint[];
}

// Staffing hours statistics
export interface StaffingHoursDataPoint {
  date: string;
  required_hours: number;
  available_hours: number;
  child_count: number;
  staff_count: number;
}

export interface StaffingHoursResponse {
  data_points: StaffingHoursDataPoint[];
}

// Occupancy matrix statistics
export interface OccupancyAgeGroup {
  label: string;
  min_age: number;
  max_age: number;
}

export interface OccupancySupplementType {
  key: string;
  value: string;
  label: string;
}

export interface OccupancyDataPoint {
  date: string;
  total: number;
  by_age_and_care_type: Record<string, Record<string, number>>;
  by_supplement: Record<string, number>;
}

export interface OccupancyResponse {
  age_groups: OccupancyAgeGroup[];
  care_types: string[];
  supplement_types: OccupancySupplementType[];
  data_points: OccupancyDataPoint[];
}

// Contract properties distribution
export interface ContractPropertyCount {
  key: string;
  value: string;
  count: number;
}

export interface ContractPropertiesDistributionResponse {
  date: string;
  total_children: number;
  properties: ContractPropertyCount[];
}
