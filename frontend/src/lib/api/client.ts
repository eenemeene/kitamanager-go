import axios, { type AxiosInstance, type AxiosError } from 'axios';
import { currentLocale, getProblem, problemMessage } from './problem';
import type {
  AgeDistributionResponse,
  AuditLogListParams,
  AuditLogResponse,
  BackupCodesPayload,
  BudgetItem,
  BudgetItemCreateRequest,
  BudgetItemDetail,
  BudgetItemEntry,
  BudgetItemEntryCreateRequest,
  BudgetItemEntryUpdateRequest,
  BudgetItemUpdateRequest,
  Child,
  ChildAttendanceCreateRequest,
  ChildAttendanceDailySummaryResponse,
  ChildAttendanceResponse,
  ChildAttendanceUpdateRequest,
  ChildBillingHistoryResponse,
  ChildContract,
  ChildContractAmendRequest,
  ChildContractAmendResponse,
  ChildContractBoundaryResponse,
  ChildContractCorrectRequest,
  ChildContractCreateRequest,
  ChildCreateRequest,
  ChildUpdateRequest,
  ChildVoucher,
  ChildWithoutVoucherResponse,
  ChildrenBillingSummaryResponse,
  ChildrenFundingResponse,
  ContractBoundaryMoveRequest,
  ContractEndRequest,
  ContractPropertiesDistributionResponse,
  Employee,
  EmployeeContract,
  EmployeeContractAmendRequest,
  EmployeeContractAmendResponse,
  EmployeeContractBoundaryResponse,
  EmployeeContractCorrectRequest,
  EmployeeContractCreateRequest,
  EmployeeCreateRequest,
  EmployeeStaffingHoursResponse,
  EmployeeUpdateRequest,
  FactorActivateRequest,
  FactorActivateResponse,
  FactorDeleteRequest,
  FactorEnrolRequest,
  FactorLabelUpdateRequest,
  FactorListResponse,
  FactorRegenerateRequest,
  FactorResponse,
  FinancialResponse,
  ForecastRequest,
  ForecastResponse,
  FundingComparisonResponse,
  FundingComparisonWrappedResponse,
  GovernmentFunding,
  GovernmentFundingBillPeriodListItem,
  GovernmentFundingBillPeriodResponse,
  GovernmentFundingBillResponse,
  GovernmentFundingCreateRequest,
  GovernmentFundingDetail,
  GovernmentFundingPeriod,
  GovernmentFundingPeriodCreateRequest,
  GovernmentFundingPeriodUpdateRequest,
  GovernmentFundingProperty,
  GovernmentFundingPropertyCreateRequest,
  GovernmentFundingPropertyUpdateRequest,
  GovernmentFundingUpdateRequest,
  LoginRequest,
  LoginResponse,
  LoginSuccessResponse,
  MfaChallengeRequest,
  MfaChallengeResponse,
  MfaVerifyRequest,
  OccupancyResponse,
  Organization,
  OrganizationCreateRequest,
  OrganizationUpdateRequest,
  PaginatedResponse,
  PaginationParams,
  PayPlan,
  PayPlanCreateRequest,
  PayPlanDetail,
  PayPlanEntry,
  PayPlanEntryCreateRequest,
  PayPlanEntryUpdateRequest,
  PayPlanPeriod,
  PayPlanPeriodCreateRequest,
  PayPlanPeriodUpdateRequest,
  PayPlanUpdateRequest,
  Role,
  Section,
  SectionCreateRequest,
  SectionUpdateRequest,
  StaffingHoursResponse,
  StepPromotionsResponse,
  UnmatchedBillChild,
  User,
  UserCreateRequest,
  UserMembershipsResponse,
  UserOrganizationResponse,
  UserSessionsResponse,
  UserUpdateRequest,
} from './types';
import { DEFAULT_PAGE_SIZE } from './types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL
  ? `${process.env.NEXT_PUBLIC_API_URL}/api/v1`
  : '/api/v1';

import { getCookie } from '@/lib/utils';
import { todayBerlinString } from '@/lib/utils/contracts';

// Helper to get CSRF token from cookie
function getCSRFToken(): string | null {
  return getCookie('csrf_token');
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
      // Enable sending cookies with requests (for HttpOnly session cookie).
      withCredentials: true,
    });

    // Request interceptor to add CSRF token for state-changing requests.
    this.client.interceptors.request.use(
      (config) => {
        const method = config.method?.toLowerCase();
        if (method && !['get', 'head', 'options'].includes(method)) {
          const csrfToken = getCSRFToken();
          if (csrfToken) {
            config.headers['X-CSRF-Token'] = csrfToken;
          }
        }
        // The reader's language, so the server can answer in it. next-intl keeps
        // the choice in a cookie, and the browser's own Accept-Language is the
        // wrong source: a German user on an English-locale laptop has already
        // told us which they want, in the app.
        config.headers['Accept-Language'] = currentLocale();
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor: on 401 to a non-auth endpoint, invoke the
    // onUnauthorized callback so the auth store can clear local state.
    // There is no client-side token refresh anymore — sessions live for their
    // full expiry and the server is the source of truth.
    this.client.interceptors.response.use(
      (response) => response,
      async (error: AxiosError) => {
        const originalRequest = error.config as typeof error.config | undefined;

        // 429s used to be enriched here with an English sentence built from
        // Retry-After, because the body had no usable message. The server now
        // sends one, localized, so synthesising a second — in one language —
        // would only override it. Retry-After is still on the response for any
        // caller that wants to show a countdown.
        if (error.response?.status === 429) {
          return Promise.reject(error);
        }

        const url = originalRequest?.url || '';
        const method = (originalRequest?.method || 'get').toLowerCase();
        // Endpoints where a 401 is part of the expected flow, not a
        // sign the user's existing session has gone stale. /login
        // returns 401 on bad creds; /logout 401 is meaningless (no
        // session anyway); /auth/mfa/verify returns 401 on wrong
        // MFA codes — surfacing that as "session expired" would
        // wipe the auth store the user hasn't even built yet.
        const isAuthEndpoint =
          url.includes('/login') || url.includes('/logout') || url.includes('/auth/mfa/');
        // State-changing factor endpoints perform step-up password
        // (or WebAuthn assertion) verification; a 401 means "wrong
        // step-up credential," not "session gone." Dispatching the
        // generic unauthorised handler would redirect the user to
        // /login while they're in the middle of a step-up dialog —
        // see webauthn.spec.ts for the regression scenario. GET on
        // the same paths still falls through so a genuinely expired
        // session bounces the user to the login screen.
        const isFactorMutation = /\/users\/[^/]+\/factors(\/|$)/.test(url) && method !== 'get';

        if (error.response?.status === 401 && !isAuthEndpoint && !isFactorMutation) {
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

  private topLevelCrud<T, TCreate, TUpdate>(resource: string) {
    return {
      // Forwards filters, like orgScopedCrud below. It used to destructure only
      // page and limit and hand-build the query string, which silently dropped
      // everything else: the Fördersätze page passed `search`, the query key
      // included it so typing triggered a refetch, and the server -- which does
      // support the parameter -- answered with the unfiltered list. The search
      // box looked like it worked and returned everything.
      list: (params: PaginationParams = {}) => {
        const { page = 1, limit = DEFAULT_PAGE_SIZE, ...filters } = params;
        const qp = new URLSearchParams({ page: String(page), limit: String(limit) });
        for (const [key, value] of Object.entries(filters)) {
          if (value !== undefined && value !== null) qp.set(key, String(value));
        }
        return this.client.get<PaginatedResponse<T>>(`/${resource}?${qp}`).then((r) => r.data);
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
      list: (orgId: number, params: PaginationParams = {}) => {
        const { page = 1, limit = DEFAULT_PAGE_SIZE, ...filters } = params;
        const qp = new URLSearchParams({ page: String(page), limit: String(limit) });
        for (const [key, value] of Object.entries(filters)) {
          if (value !== undefined && value !== null) qp.set(key, String(value));
        }
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

  /** Fetch all pages of a paginated endpoint and return the combined data array. */
  private async fetchAllPages<T>(baseUrl: string, pageSize = 100): Promise<T[]> {
    const separator = baseUrl.includes('?') ? '&' : '?';
    const all: T[] = [];
    let page = 1;
    let totalPages = 1;
    do {
      const response = await this.client.get<PaginatedResponse<T>>(
        `${baseUrl}${separator}limit=${pageSize}&page=${page}`
      );
      all.push(...response.data.data);
      totalPages = response.data.total_pages;
      page++;
    } while (page <= totalPages);
    return all;
  }

  /** Fetch an org-scoped statistic with optional section + date range filters. */
  private async getStatistic<T>(
    orgId: number,
    endpoint: string,
    opts?: { sectionId?: number; from?: string; to?: string }
  ): Promise<T> {
    const params: Record<string, string> = {};
    if (opts?.sectionId) params.section_id = opts.sectionId.toString();
    if (opts?.from) params.from = opts.from;
    if (opts?.to) params.to = opts.to;
    const response = await this.client.get<T>(`/organizations/${orgId}/statistics/${endpoint}`, {
      params,
    });
    return response.data;
  }

  /** Build an export download URL with optional query string filters. */
  private buildExportUrl(
    orgId: number,
    resource: string,
    format: 'excel' | 'yaml',
    filters?: Record<string, string | undefined>
  ): string {
    const base = `${API_BASE_URL}/organizations/${orgId}/${resource}/export/${format}`;
    if (!filters) return base;
    const qp = new URLSearchParams();
    for (const [key, value] of Object.entries(filters)) {
      if (value !== undefined && value !== '') qp.set(key, value);
    }
    const qs = qp.toString();
    return qs ? `${base}?${qs}` : base;
  }

  // Auth
  async login(request: LoginRequest): Promise<LoginResponse> {
    const response = await this.client.post<LoginResponse>('/login', request);
    return response.data;
  }

  // MFA step-two. Pairs with a LoginMfaRequiredResponse returned from
  // /login. A 200 here sets the session cookie just like a direct
  // /login does; a 401 means wrong code (input should stay on screen
  // with the error), 429 means the per-row or per-user limit hit and
  // the caller should revert back to the password step.
  async verifyMfa(request: MfaVerifyRequest): Promise<LoginSuccessResponse> {
    const response = await this.client.post<LoginSuccessResponse>('/auth/mfa/verify', request);
    return response.data;
  }

  // Fetches the WebAuthn challenge for step 2a of the MFA login
  // flow. TOTP/backup factors don't need this. The returned
  // request_options blob is passed to navigator.credentials.get().
  async beginMfaChallenge(request: MfaChallengeRequest): Promise<MfaChallengeResponse> {
    const response = await this.client.post<MfaChallengeResponse>('/auth/mfa/challenge', request);
    return response.data;
  }

  async logout(): Promise<void> {
    await this.client.post('/logout');
  }

  // Factor (MFA) management — all scoped to the caller via `/users/me/`.
  async listMyFactors(): Promise<FactorListResponse> {
    const response = await this.client.get<FactorListResponse>('/users/me/factors');
    return response.data;
  }

  async enrolTotp(password: string, label?: string): Promise<FactorResponse> {
    const body: FactorEnrolRequest = { type: 'totp', password, label };
    const response = await this.client.post<FactorResponse>('/users/me/factors', body);
    return response.data;
  }

  // Starts a WebAuthn registration ceremony. Returns a factor
  // descriptor whose `enrollment` field carries the raw
  // PublicKeyCredentialCreationOptions JSON the caller feeds to
  // `navigator.credentials.create()`. The pending factor row lives
  // on the server with a 5-minute challenge expiry.
  async enrolWebAuthn(password: string, label?: string): Promise<FactorResponse> {
    const body: FactorEnrolRequest = { type: 'webauthn', password, label };
    const response = await this.client.post<FactorResponse>('/users/me/factors', body);
    return response.data;
  }

  // Activates a factor. For TOTP, pass `code`. For WebAuthn, pass the
  // PublicKeyCredential JSON returned by
  // `navigator.credentials.create()` as `webauthnResponse`.
  async activateFactor(
    factorId: number,
    args: { code?: string; webauthnResponse?: unknown }
  ): Promise<FactorActivateResponse> {
    const body: FactorActivateRequest = {
      code: args.code,
      webauthn_response: args.webauthnResponse as Record<string, never> | undefined,
    };
    const response = await this.client.post<FactorActivateResponse>(
      `/users/me/factors/${factorId}/activate`,
      body
    );
    return response.data;
  }

  async regenerateBackupCodes(
    factorId: number,
    password: string,
    code: string
  ): Promise<BackupCodesPayload> {
    const body: FactorRegenerateRequest = { password, code };
    const response = await this.client.post<BackupCodesPayload>(
      `/users/me/factors/${factorId}/regenerate`,
      body
    );
    return response.data;
  }

  async deleteFactor(factorId: number, password: string, code?: string): Promise<void> {
    const body: FactorDeleteRequest = { password, code };
    await this.client.delete(`/users/me/factors/${factorId}`, { data: body });
  }

  async updateFactorLabel(factorId: number, label: string | null): Promise<FactorResponse> {
    const body: FactorLabelUpdateRequest = { label: label ?? undefined };
    const response = await this.client.patch<FactorResponse>(`/users/me/factors/${factorId}`, body);
    return response.data;
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.client.get<User>('/me');
    return response.data;
  }

  async changePassword(
    currentPassword: string,
    newPassword: string
  ): Promise<LoginSuccessResponse> {
    // Backend rotates the session + csrf cookies on success, so the caller
    // stays live with fresh credentials. Other sessions the user has on other
    // devices are revoked server-side.
    const response = await this.client.put<LoginSuccessResponse>('/me/password', {
      current_password: currentPassword,
      new_password: newPassword,
    });
    return response.data;
  }

  async getSessions(): Promise<UserSessionsResponse> {
    const response = await this.client.get<UserSessionsResponse>('/me/sessions');
    return response.data;
  }

  async revokeSession(id: string): Promise<void> {
    await this.client.delete(`/me/sessions/${encodeURIComponent(id)}`);
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

  // User-Organization assignments with roles
  async addUserToOrganization(
    userId: number,
    organizationId: number,
    role?: Role
  ): Promise<UserOrganizationResponse> {
    const response = await this.client.post<UserOrganizationResponse>(
      `/users/${userId}/organizations`,
      { organization_id: organizationId, role }
    );
    return response.data;
  }

  async removeUserFromOrganization(userId: number, organizationId: number): Promise<void> {
    await this.client.delete(`/users/${userId}/organizations/${organizationId}`);
  }

  async updateUserOrganizationRole(
    userId: number,
    organizationId: number,
    role: Role
  ): Promise<UserOrganizationResponse> {
    const response = await this.client.put<UserOrganizationResponse>(
      `/users/${userId}/organizations/${organizationId}`,
      { role }
    );
    return response.data;
  }

  async getUserMemberships(userId: number): Promise<UserMembershipsResponse> {
    const response = await this.client.get<UserMembershipsResponse>(`/users/${userId}/memberships`);
    return response.data;
  }

  // Self-memberships endpoint. Required by useCurrentRole (via auth-store) on
  // every page load — the admin-facing getUserMemberships is gated on
  // users:read which staff and member don't have, so they would silently end
  // up with an empty orgRoleMap and a blank sidebar.
  async getMyMemberships(): Promise<UserMembershipsResponse> {
    const response = await this.client.get<UserMembershipsResponse>(`/me/memberships`);
    return response.data;
  }

  async setSuperAdmin(userId: number, isSuperAdmin: boolean): Promise<User> {
    const response = await this.client.put<User>(`/users/${userId}/superadmin`, {
      is_superadmin: isSuperAdmin,
    });
    return response.data;
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
  //
  // Every page, not just the first: the endpoint is paginated and defaults to 20,
  // while the contracts page renders the whole history with no pager. Reading
  // only `data` therefore dropped contract 21 onwards -- and because the backend
  // orders by `from_date DESC`, what vanished was the *oldest* history, from the
  // table and the timeline alike. Each amendment adds a row, so a long-tenured
  // employee reaches 20.
  async getEmployeeContracts(orgId: number, employeeId: number): Promise<EmployeeContract[]> {
    return this.fetchAllPages<EmployeeContract>(
      `/organizations/${orgId}/employees/${employeeId}/contracts`
    );
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

  /**
   * Corrects a employee contract in place: the recorded facts were wrong.
   *
   * A true partial update — a field you omit is left alone, and `to`/`properties`
   * are cleared only by an explicit null. `weekly_hours: 0` is expressible here,
   * which the old request could not do. Use amendEmployeeContract when the terms
   * changed as of a date, so pay history stays correct for past months.
   */
  async correctEmployeeContract(
    orgId: number,
    employeeId: number,
    contractId: number,
    version: number,
    data: EmployeeContractCorrectRequest
  ): Promise<EmployeeContract> {
    const response = await this.client.patch<EmployeeContract>(
      `/organizations/${orgId}/employees/${employeeId}/contracts/${contractId}`,
      data,
      { headers: { 'If-Match': `"${version}"` } }
    );
    return response.data;
  }

  /**
   * Amends a employee contract from a date: closes it the day before
   * `effective_from` and creates a successor carrying the changes. Returns both.
   *
   * This is the operation for anything affecting pay: a raise recorded as a
   * correction would silently restate what the employee earned last month.
   */
  async amendEmployeeContract(
    orgId: number,
    employeeId: number,
    contractId: number,
    version: number,
    data: EmployeeContractAmendRequest
  ): Promise<EmployeeContractAmendResponse> {
    const response = await this.client.post<EmployeeContractAmendResponse>(
      `/organizations/${orgId}/employees/${employeeId}/contracts/${contractId}/amend`,
      data,
      { headers: { 'If-Match': `"${version}"` } }
    );
    return response.data;
  }

  /** Sets or clears a employee contract's end date; `to: null` reopens it. */
  async endEmployeeContract(
    orgId: number,
    employeeId: number,
    contractId: number,
    version: number,
    data: ContractEndRequest
  ): Promise<EmployeeContract> {
    const response = await this.client.post<EmployeeContract>(
      `/organizations/${orgId}/employees/${employeeId}/contracts/${contractId}/end`,
      data,
      { headers: { 'If-Match': `"${version}"` } }
    );
    return response.data;
  }

  /**
   * Moves the seam between two adjacent employee contracts. One date; the server
   * derives both sides, which is why this cannot clear the neighbour's end date
   * or wipe its properties the way the batch payload it replaces could.
   */
  async moveEmployeeContractBoundary(
    orgId: number,
    employeeId: number,
    data: ContractBoundaryMoveRequest
  ): Promise<EmployeeContractBoundaryResponse> {
    const response = await this.client.post<EmployeeContractBoundaryResponse>(
      `/organizations/${orgId}/employees/${employeeId}/contracts/boundary`,
      data
    );
    return response.data;
  }

  /**
   * Deletes an employee contract. `version` is sent as an If-Match precondition —
   * see deleteChildContract for why deletes carry one.
   */
  async deleteEmployeeContract(
    orgId: number,
    employeeId: number,
    contractId: number,
    version: number
  ): Promise<void> {
    await this.client.delete(
      `/organizations/${orgId}/employees/${employeeId}/contracts/${contractId}`,
      { headers: { 'If-Match': `"${version}"` } }
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
  //
  // Paginated like the employee equivalent above, and truncated the same way
  // before this used fetchAllPages.
  async getChildContracts(orgId: number, childId: number): Promise<ChildContract[]> {
    return this.fetchAllPages<ChildContract>(
      `/organizations/${orgId}/children/${childId}/contracts`
    );
  }

  async getChildBillingHistory(
    orgId: number,
    childId: number
  ): Promise<ChildBillingHistoryResponse> {
    const response = await this.client.get<ChildBillingHistoryResponse>(
      `/organizations/${orgId}/children/${childId}/billing-history`
    );
    return response.data;
  }

  async getChildrenBillingSummary(orgId: number): Promise<ChildrenBillingSummaryResponse> {
    const response = await this.client.get<ChildrenBillingSummaryResponse>(
      `/organizations/${orgId}/children/billing-summary`
    );
    return response.data;
  }

  async compareBills(
    orgId: number,
    params?: { bill_id?: number; from?: string; to?: string }
  ): Promise<FundingComparisonWrappedResponse> {
    const response = await this.client.get<FundingComparisonWrappedResponse>(
      `/organizations/${orgId}/government-funding-bills/compare`,
      { params }
    );
    return response.data;
  }

  async getChildrenWithoutVouchers(orgId: number): Promise<ChildWithoutVoucherResponse[]> {
    const response = await this.client.get<ChildWithoutVoucherResponse[]>(
      `/organizations/${orgId}/children/without-vouchers`
    );
    return response.data;
  }

  async assignChildVoucher(orgId: number, childId: number, voucherNumber: string): Promise<void> {
    await this.client.post(`/organizations/${orgId}/children/${childId}/vouchers`, {
      voucher_number: voucherNumber,
    });
  }

  async getChildVouchers(orgId: number, childId: number): Promise<ChildVoucher[]> {
    const response = await this.client.get<ChildVoucher[]>(
      `/organizations/${orgId}/children/${childId}/vouchers`
    );
    return response.data;
  }

  async removeChildVoucher(orgId: number, childId: number, voucherId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/children/${childId}/vouchers/${voucherId}`);
  }

  async getUnmatchedBillChildren(orgId: number): Promise<UnmatchedBillChild[]> {
    const response = await this.client.get<UnmatchedBillChild[]>(
      `/organizations/${orgId}/government-funding-bills/unmatched-children`
    );
    return response.data;
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

  /**
   * Corrects a child contract in place: the recorded facts were wrong.
   *
   * A true partial update — a field you omit is left alone, and `to`/`properties`
   * are cleared only by an explicit null. Use amendChildContract instead when the
   * facts changed as of a date, so the old ones stay on record for the months
   * they applied to.
   */
  async correctChildContract(
    orgId: number,
    childId: number,
    contractId: number,
    version: number,
    data: ChildContractCorrectRequest
  ): Promise<ChildContract> {
    const response = await this.client.patch<ChildContract>(
      `/organizations/${orgId}/children/${childId}/contracts/${contractId}`,
      data,
      { headers: { 'If-Match': `"${version}"` } }
    );
    return response.data;
  }

  /**
   * Amends a child contract from a date: closes it the day before
   * `effective_from` and creates a successor carrying the changes. Returns both.
   *
   * `effective_from` is honoured, including in the past, so a Bescheid that
   * arrives late is one call.
   */
  async amendChildContract(
    orgId: number,
    childId: number,
    contractId: number,
    version: number,
    data: ChildContractAmendRequest
  ): Promise<ChildContractAmendResponse> {
    const response = await this.client.post<ChildContractAmendResponse>(
      `/organizations/${orgId}/children/${childId}/contracts/${contractId}/amend`,
      data,
      { headers: { 'If-Match': `"${version}"` } }
    );
    return response.data;
  }

  /** Sets or clears a child contract's end date; `to: null` reopens it. */
  async endChildContract(
    orgId: number,
    childId: number,
    contractId: number,
    version: number,
    data: ContractEndRequest
  ): Promise<ChildContract> {
    const response = await this.client.post<ChildContract>(
      `/organizations/${orgId}/children/${childId}/contracts/${contractId}/end`,
      data,
      { headers: { 'If-Match': `"${version}"` } }
    );
    return response.data;
  }

  /**
   * Moves the seam between two adjacent child contracts. One date; the server
   * derives both sides, which is why this cannot clear the neighbour's end date
   * or wipe its properties the way the batch payload it replaces could.
   */
  async moveChildContractBoundary(
    orgId: number,
    childId: number,
    data: ContractBoundaryMoveRequest
  ): Promise<ChildContractBoundaryResponse> {
    const response = await this.client.post<ChildContractBoundaryResponse>(
      `/organizations/${orgId}/children/${childId}/contracts/boundary`,
      data
    );
    return response.data;
  }

  /**
   * Deletes a child contract.
   *
   * `version` is the contract's optimistic-concurrency token, sent as an If-Match
   * precondition: deleting a contract destroys its care type, supplements and
   * period, so if someone changed any of that since this client read it the
   * server refuses with 412 rather than silently discarding their edit.
   */
  async deleteChildContract(
    orgId: number,
    childId: number,
    contractId: number,
    version: number
  ): Promise<void> {
    await this.client.delete(
      `/organizations/${orgId}/children/${childId}/contracts/${contractId}`,
      { headers: { 'If-Match': `"${version}"` } }
    );
  }

  async getChildrenFunding(orgId: number, date?: string): Promise<ChildrenFundingResponse> {
    const params = date ? { date } : {};
    const response = await this.client.get<ChildrenFundingResponse>(
      `/organizations/${orgId}/statistics/funding`,
      { params }
    );
    return response.data;
  }

  async getAgeDistribution(orgId: number, date?: string): Promise<AgeDistributionResponse> {
    const params = date ? { date } : {};
    const response = await this.client.get<AgeDistributionResponse>(
      `/organizations/${orgId}/statistics/age-distribution`,
      { params }
    );
    return response.data;
  }

  async getContractPropertiesDistribution(
    orgId: number,
    date?: string
  ): Promise<ContractPropertiesDistributionResponse> {
    const params = date ? { date } : {};
    const response = await this.client.get<ContractPropertiesDistributionResponse>(
      `/organizations/${orgId}/statistics/contract-properties`,
      { params }
    );
    return response.data;
  }

  // GovernmentFundings
  private _governmentFundings = this.topLevelCrud<
    GovernmentFunding,
    GovernmentFundingCreateRequest,
    GovernmentFundingUpdateRequest
  >('government-funding-rates');
  getGovernmentFundings = this._governmentFundings.list;
  createGovernmentFunding = this._governmentFundings.create;
  updateGovernmentFunding = this._governmentFundings.update;
  deleteGovernmentFunding = this._governmentFundings.delete;

  // GET /government-funding-rates/:id returns the detail shape
  // (GovernmentFundingDetailResponse), which embeds periods. The list
  // endpoint returns the simpler GovernmentFunding shape.
  async getGovernmentFunding(id: number, periodsLimit?: number): Promise<GovernmentFundingDetail> {
    const params = periodsLimit !== undefined ? { periods_limit: periodsLimit } : {};
    const response = await this.client.get<GovernmentFundingDetail>(
      `/government-funding-rates/${id}`,
      { params }
    );
    return response.data;
  }

  // GovernmentFunding Periods
  async createGovernmentFundingPeriod(
    governmentFundingId: number,
    data: GovernmentFundingPeriodCreateRequest
  ): Promise<GovernmentFundingPeriod> {
    const response = await this.client.post<GovernmentFundingPeriod>(
      `/government-funding-rates/${governmentFundingId}/periods`,
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
      `/government-funding-rates/${governmentFundingId}/periods/${periodId}`,
      data
    );
    return response.data;
  }

  async deleteGovernmentFundingPeriod(
    governmentFundingId: number,
    periodId: number
  ): Promise<void> {
    await this.client.delete(
      `/government-funding-rates/${governmentFundingId}/periods/${periodId}`
    );
  }

  // GovernmentFunding Properties
  async createGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    data: GovernmentFundingPropertyCreateRequest
  ): Promise<GovernmentFundingProperty> {
    const response = await this.client.post<GovernmentFundingProperty>(
      `/government-funding-rates/${governmentFundingId}/periods/${periodId}/properties`,
      data
    );
    return response.data;
  }

  async updateGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    propertyId: number,
    data: GovernmentFundingPropertyUpdateRequest
  ): Promise<GovernmentFundingProperty> {
    const response = await this.client.put<GovernmentFundingProperty>(
      `/government-funding-rates/${governmentFundingId}/periods/${periodId}/properties/${propertyId}`,
      data
    );
    return response.data;
  }

  async deleteGovernmentFundingProperty(
    governmentFundingId: number,
    periodId: number,
    propertyId: number
  ): Promise<void> {
    await this.client.delete(
      `/government-funding-rates/${governmentFundingId}/periods/${periodId}/properties/${propertyId}`
    );
  }

  // PayPlans (organization-scoped). The list / create / update endpoints
  // return the simpler PayPlan shape; GET /:id returns PayPlanDetail with
  // nested periods, so we override get with the detail type.
  private _payPlans = this.orgScopedCrud<PayPlan, PayPlanCreateRequest, PayPlanUpdateRequest>(
    'pay-plans'
  );
  getPayPlans = this._payPlans.list;
  getPayPlan = (orgId: number, id: number): Promise<PayPlanDetail> =>
    this.client.get<PayPlanDetail>(`/organizations/${orgId}/pay-plans/${id}`).then((r) => r.data);
  createPayPlan = this._payPlans.create;
  updatePayPlan = this._payPlans.update;
  deletePayPlan = this._payPlans.delete;

  getPayPlanExportUrl(orgId: number, payplanId: number): string {
    return `${API_BASE_URL}/organizations/${orgId}/pay-plans/${payplanId}/export`;
  }

  // Returns the detail shape, not the list shape: the import answers with the
  // plan it just built, periods included. Typing it as PayPlan promised a
  // `periods_count` the response does not carry and hid the `periods` it does.
  async importPayPlan(orgId: number, file: File): Promise<PayPlanDetail> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await this.client.post<PayPlanDetail>(
      `/organizations/${orgId}/pay-plans/import`,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    );
    return response.data;
  }

  // PayPlan Periods
  async createPayPlanPeriod(
    orgId: number,
    payplanId: number,
    data: PayPlanPeriodCreateRequest
  ): Promise<PayPlanPeriod> {
    const response = await this.client.post<PayPlanPeriod>(
      `/organizations/${orgId}/pay-plans/${payplanId}/periods`,
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
      `/organizations/${orgId}/pay-plans/${payplanId}/periods/${periodId}`
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
      `/organizations/${orgId}/pay-plans/${payplanId}/periods/${periodId}`,
      data
    );
    return response.data;
  }

  async deletePayPlanPeriod(orgId: number, payplanId: number, periodId: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/pay-plans/${payplanId}/periods/${periodId}`);
  }

  // PayPlan Entries
  async createPayPlanEntry(
    orgId: number,
    payplanId: number,
    periodId: number,
    data: PayPlanEntryCreateRequest
  ): Promise<PayPlanEntry> {
    const response = await this.client.post<PayPlanEntry>(
      `/organizations/${orgId}/pay-plans/${payplanId}/periods/${periodId}/entries`,
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
      `/organizations/${orgId}/pay-plans/${payplanId}/periods/${periodId}/entries/${entryId}`
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
      `/organizations/${orgId}/pay-plans/${payplanId}/periods/${periodId}/entries/${entryId}`,
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
      `/organizations/${orgId}/pay-plans/${payplanId}/periods/${periodId}/entries/${entryId}`
    );
  }

  // Budget Items (organization-scoped). Same list/detail split as PayPlan:
  // GET /:id returns BudgetItemDetail with nested entries.
  private _budgetItems = this.orgScopedCrud<
    BudgetItem,
    BudgetItemCreateRequest,
    BudgetItemUpdateRequest
  >('budget-items');
  getBudgetItems = this._budgetItems.list;
  getBudgetItem = (orgId: number, id: number): Promise<BudgetItemDetail> =>
    this.client
      .get<BudgetItemDetail>(`/organizations/${orgId}/budget-items/${id}`)
      .then((r) => r.data);
  createBudgetItem = this._budgetItems.create;
  updateBudgetItem = this._budgetItems.update;
  deleteBudgetItem = this._budgetItems.delete;

  // Budget Item Entries
  async createBudgetItemEntry(
    orgId: number,
    budgetItemId: number,
    data: BudgetItemEntryCreateRequest
  ): Promise<BudgetItemEntry> {
    const response = await this.client.post<BudgetItemEntry>(
      `/organizations/${orgId}/budget-items/${budgetItemId}/entries`,
      data
    );
    return response.data;
  }

  async updateBudgetItemEntry(
    orgId: number,
    budgetItemId: number,
    entryId: number,
    data: BudgetItemEntryUpdateRequest
  ): Promise<BudgetItemEntry> {
    const response = await this.client.put<BudgetItemEntry>(
      `/organizations/${orgId}/budget-items/${budgetItemId}/entries/${entryId}`,
      data
    );
    return response.data;
  }

  async deleteBudgetItemEntry(orgId: number, budgetItemId: number, entryId: number): Promise<void> {
    await this.client.delete(
      `/organizations/${orgId}/budget-items/${budgetItemId}/entries/${entryId}`
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
    const today = todayBerlinString();
    return this.fetchAllPages<Employee>(`/organizations/${orgId}/employees?active_on=${today}`);
  }

  async getStepPromotions(orgId: number, date?: string): Promise<StepPromotionsResponse> {
    const params = date ? { date } : {};
    const response = await this.client.get<StepPromotionsResponse>(
      `/organizations/${orgId}/employees/step-promotions`,
      { params }
    );
    return response.data;
  }

  async getStaffingHours(
    orgId: number,
    opts?: { sectionId?: number; from?: string; to?: string }
  ): Promise<StaffingHoursResponse> {
    return this.getStatistic<StaffingHoursResponse>(orgId, 'staffing-hours', opts);
  }

  async getFinancials(
    orgId: number,
    opts?: { from?: string; to?: string }
  ): Promise<FinancialResponse> {
    return this.getStatistic<FinancialResponse>(orgId, 'financials', opts);
  }

  async getOccupancy(
    orgId: number,
    opts?: { sectionId?: number; from?: string; to?: string }
  ): Promise<OccupancyResponse> {
    return this.getStatistic<OccupancyResponse>(orgId, 'occupancy', opts);
  }

  async getEmployeeStaffingHours(
    orgId: number,
    opts?: { sectionId?: number; from?: string; to?: string }
  ): Promise<EmployeeStaffingHoursResponse> {
    return this.getStatistic<EmployeeStaffingHoursResponse>(
      orgId,
      'staffing-hours/employees',
      opts
    );
  }

  async postForecast(
    orgId: number,
    request: ForecastRequest,
    signal?: AbortSignal
  ): Promise<ForecastResponse> {
    const response = await this.client.post<ForecastResponse>(
      `/organizations/${orgId}/statistics/forecast`,
      request,
      { signal }
    );
    return response.data;
  }

  getEmployeesExportUrl(orgId: number, filters?: Record<string, string | undefined>): string {
    return this.buildExportUrl(orgId, 'employees', 'excel', filters);
  }

  getEmployeesExportYamlUrl(orgId: number): string {
    return this.buildExportUrl(orgId, 'employees', 'yaml');
  }

  async importEmployees(orgId: number, file: File): Promise<Employee[]> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await this.client.post<Employee[]>(
      `/organizations/${orgId}/employees/import`,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    );
    return response.data;
  }

  getChildrenExportUrl(orgId: number, filters?: Record<string, string | undefined>): string {
    return this.buildExportUrl(orgId, 'children', 'excel', filters);
  }

  getChildrenExportYamlUrl(orgId: number): string {
    return this.buildExportUrl(orgId, 'children', 'yaml');
  }

  async importChildren(orgId: number, file: File): Promise<Child[]> {
    const formData = new FormData();
    formData.append('file', file);
    const response = await this.client.post<Child[]>(
      `/organizations/${orgId}/children/import`,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    );
    return response.data;
  }

  // Child Attendance
  async getChildAttendanceByDateAll(
    orgId: number,
    date: string
  ): Promise<ChildAttendanceResponse[]> {
    return this.fetchAllPages<ChildAttendanceResponse>(
      `/organizations/${orgId}/children/attendance?date=${date}`
    );
  }

  async getChildAttendanceSummary(
    orgId: number,
    date?: string
  ): Promise<ChildAttendanceDailySummaryResponse> {
    const params = date ? { date } : {};
    const response = await this.client.get<ChildAttendanceDailySummaryResponse>(
      `/organizations/${orgId}/children/attendance/summary`,
      { params }
    );
    return response.data;
  }

  async createChildAttendance(
    orgId: number,
    childId: number,
    data: ChildAttendanceCreateRequest
  ): Promise<ChildAttendanceResponse> {
    const response = await this.client.post<ChildAttendanceResponse>(
      `/organizations/${orgId}/children/${childId}/attendance`,
      data
    );
    return response.data;
  }

  async updateChildAttendance(
    orgId: number,
    childId: number,
    attendanceId: number,
    data: ChildAttendanceUpdateRequest
  ): Promise<ChildAttendanceResponse> {
    const response = await this.client.put<ChildAttendanceResponse>(
      `/organizations/${orgId}/children/${childId}/attendance/${attendanceId}`,
      data
    );
    return response.data;
  }

  async deleteChildAttendance(orgId: number, childId: number, attendanceId: number): Promise<void> {
    await this.client.delete(
      `/organizations/${orgId}/children/${childId}/attendance/${attendanceId}`
    );
  }

  // Children - fetch all with active contracts for a specific date
  //
  // `active_on` is the parameter the endpoint declares. This used to send
  // `contract_on`, which the backend never parsed: gin drops unknown query
  // parameters silently, so the handler fell back to its default of today and the
  // caller believed it had filtered. The section board's date picker therefore did
  // nothing — it always showed today's roster.
  async getChildrenAllForDate(orgId: number, date: string, sectionId?: number): Promise<Child[]> {
    let url = `/organizations/${orgId}/children?active_on=${date}`;
    if (sectionId) url += `&section_id=${sectionId}`;
    return this.fetchAllPages<Child>(url);
  }

  // Government Funding Bill
  async uploadGovernmentFundingBill(
    orgId: number,
    formData: FormData
  ): Promise<GovernmentFundingBillResponse> {
    const response = await this.client.post<GovernmentFundingBillResponse>(
      `/organizations/${orgId}/government-funding-bills`,
      formData,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    );
    return response.data;
  }

  async getGovernmentFundingBillPeriods(
    orgId: number,
    params: PaginationParams & { search?: string } = {}
  ): Promise<PaginatedResponse<GovernmentFundingBillPeriodListItem>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE, search } = params;
    const qs = new URLSearchParams({ page: String(page), limit: String(limit) });
    if (search) qs.set('search', search);
    const response = await this.client.get<PaginatedResponse<GovernmentFundingBillPeriodListItem>>(
      `/organizations/${orgId}/government-funding-bills?${qs.toString()}`
    );
    return response.data;
  }

  async getGovernmentFundingBillPeriod(
    orgId: number,
    id: number
  ): Promise<GovernmentFundingBillPeriodResponse> {
    const response = await this.client.get<GovernmentFundingBillPeriodResponse>(
      `/organizations/${orgId}/government-funding-bills/${id}`
    );
    return response.data;
  }

  async deleteGovernmentFundingBillPeriod(orgId: number, id: number): Promise<void> {
    await this.client.delete(`/organizations/${orgId}/government-funding-bills/${id}`);
  }

  async compareGovernmentFundingBill(
    orgId: number,
    id: number
  ): Promise<FundingComparisonResponse> {
    const response = await this.client.get<FundingComparisonResponse>(
      `/organizations/${orgId}/government-funding-bills/${id}/compare`
    );
    return response.data;
  }

  async getHealth(): Promise<{ status: string; version: string }> {
    const response = await this.client.get<{ status: string; version: string }>('/health');
    return response.data;
  }

  async getAuditLogs(
    orgId: number,
    params: AuditLogListParams = {}
  ): Promise<PaginatedResponse<AuditLogResponse>> {
    const { page = 1, limit = DEFAULT_PAGE_SIZE, action, user_id, from, to } = params;
    const qs = new URLSearchParams({ page: String(page), limit: String(limit) });
    if (action) qs.set('action', action);
    if (user_id !== undefined) qs.set('user_id', String(user_id));
    if (from) qs.set('from', from);
    if (to) qs.set('to', to);
    const response = await this.client.get<PaginatedResponse<AuditLogResponse>>(
      `/organizations/${orgId}/audit-logs?${qs.toString()}`
    );
    return response.data;
  }

  // Children - fetch upcoming (contracts starting after today)
  async getUpcomingChildren(orgId: number): Promise<Child[]> {
    const today = todayBerlinString();
    return this.fetchAllPages<Child>(`/organizations/${orgId}/children?contract_after=${today}`);
  }

  // Children - fetch all with active contracts (for kanban board view)
  async getChildrenAll(orgId: number): Promise<Child[]> {
    const today = todayBerlinString();
    return this.fetchAllPages<Child>(`/organizations/${orgId}/children?active_on=${today}`);
  }

  // Employees - fetch all with active contracts on a specific date (for
  // the kanban board's "as of date" picker — see getChildrenAllForDate
  // for the children-side counterpart).
  async getEmployeesAllForDate(orgId: number, date: string): Promise<Employee[]> {
    return this.fetchAllPages<Employee>(`/organizations/${orgId}/employees?active_on=${date}`);
  }
}

export const apiClient = new ApiClient();

/**
 * The message to show a user for a failed request.
 *
 * The API answers every error with an RFC 9457 problem document, so this reads
 * `code` first and translates it — which is what makes a German user see German.
 * See `problem.ts` for why the resolution order differs per locale, and for
 * `getInvalidParams` / `getRequestId` when a caller wants to mark form fields or
 * quote a request id.
 *
 * The `fallback` argument still wins for anything unparseable — a network error,
 * a timeout, a proxy's own HTML error page — which is most of what it ever
 * covered.
 */
export function getErrorMessage(error: unknown, fallback: string): string {
  const problem = getProblem(error);
  if (problem) {
    return problemMessage(problem) || fallback;
  }
  return fallback;
}
