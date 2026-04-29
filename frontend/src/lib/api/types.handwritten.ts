// Frontend-only API types — kept hand-written because the OpenAPI spec
// can't (or shouldn't) express them as-is:
//
//   - Literal-string unions narrow Go's stringly-typed enums (Gender,
//     Role, FactorType, etc.) for autocomplete and exhaustive checks.
//   - LoginResponse is a discriminated union the spec doesn't model
//     as oneOf (swaggo 2.0 limitation).
//   - AuditAction is intentionally open (`(string & {})`) so the
//     backend can emit new actions without breaking the frontend
//     build before the type list is updated.
//   - PaginatedResponse<T> is a generic; the spec instead emits one
//     discrete schema per element type (PaginatedResponse-FooResponse).
//   - Constants (DEFAULT_PAGE_SIZE etc.) are values, not types.
//   - DashboardStats, AuditLogListParams, PaginationParams are
//     frontend composite shapes with no API counterpart.

// === Literal-string unions ===

export type Gender = 'male' | 'female' | 'diverse';
export type Role = 'admin' | 'manager' | 'member' | 'staff';
export type FactorType = 'totp' | 'backup_codes' | 'webauthn';
export type ChildAttendanceStatus = 'present' | 'absent' | 'sick' | 'vacation';
export type MismatchType = 'missing' | 'additional' | 'different';

// Map type for ContractProperties. Backend serializes this as
// `map[string]any` which the spec records as `additionalProperties: {}`,
// but at runtime the values are always string or string[].
export type ContractProperties = Record<string, string | string[]>;

// AuditAction mirrors the list in internal/models/audit.go but stays
// open so the backend can emit new variants before the frontend type
// is updated. Known values still autocomplete.
export type AuditAction =
  | 'login'
  | 'login_failed'
  | 'logout'
  | 'superadmin_grant'
  | 'superadmin_revoke'
  | 'user_create'
  | 'user_update'
  | 'user_delete'
  | 'user_add_to_org'
  | 'user_remove_from_org'
  | 'role_change'
  | 'employee_delete'
  | 'child_delete'
  | 'org_create'
  | 'org_delete'
  | 'password_reset'
  | 'password_change'
  | 'password_change_failed'
  | (string & {});

// === Constants ===

export const DEFAULT_PAGE_SIZE = 30;

/** Fetch limit for lookup/dropdown data (sections, pay plans, etc.) where all items are needed. */
export const LOOKUP_FETCH_LIMIT = 100;

/** Valid state (Bundesland) values. Must match the backend's models.ValidStates. */
export const VALID_STATES = ['berlin'] as const;
export type ValidState = (typeof VALID_STATES)[number];

// === Frontend composite shapes ===

// Pagination response wrapper. Generic on T so consumers can request
// PaginatedResponse<User>, PaginatedResponse<Child> etc. without
// going through the per-element-type schemas the spec emits.
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

// Composed query params for the audit log list endpoint.
export interface AuditLogListParams {
  page?: number;
  limit?: number;
  action?: AuditAction;
  user_id?: number;
  from?: string;
  to?: string;
  organization_id?: number;
}

// Dashboard summary view assembled client-side from multiple endpoints.
export interface DashboardStats {
  totalEmployees: number;
  totalChildren: number;
  totalSections: number;
  activeContracts: number;
}

// Forecast scenario inputs are slim composites — minimum data needed to
// describe a hypothetical added child / employee / contract. They are NOT
// the same as the full ChildResponse / EmployeeResponse shapes (which
// require id, created_at etc.). The backend's POST /forecast accepts these
// thin shapes; the spec types them via a different DTO than ChildResponse.

export interface ForecastChildContract {
  child_id?: number;
  from: string;
  to?: string | null;
  section_id: number;
  properties?: ContractProperties;
}

export interface ForecastChild {
  first_name: string;
  last_name: string;
  gender: string;
  birthdate: string;
  contracts: ForecastChildContract[];
}

export interface ForecastEmployeeContract {
  employee_id?: number;
  from: string;
  to?: string | null;
  section_id: number;
  staff_category: string;
  grade?: string;
  step?: number;
  weekly_hours: number;
  payplan_id: number;
}

export interface ForecastEmployee {
  first_name: string;
  last_name: string;
  gender: string;
  birthdate: string;
  email?: string;
  contracts: ForecastEmployeeContract[];
}
