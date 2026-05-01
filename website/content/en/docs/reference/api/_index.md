---
title: API
weight: 1
aliases:
  - /docs/api/
---

KitaManager exposes a REST API. The interactive OpenAPI/Swagger UI is bundled with the server at `/swagger/index.html` — that is the **canonical** reference and reflects the running build exactly. The pages below are a hand-written summary that organises the endpoints by resource for browsability.

The OpenAPI spec at `/swagger/index.html` provides per-endpoint detail; this page groups the endpoints by resource for browsability.

## Authentication

Authentication is **cookie-based**: a successful login sets an HttpOnly `access_token` session cookie plus a JS-readable `csrf_token` cookie. Mutating requests (POST/PUT/PATCH/DELETE) must echo the CSRF token via the `X-CSRF-Token` header. There is no separate refresh endpoint — sessions remain valid until you log out, the cookie expires, or you revoke them from `/me/sessions`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/login` | Step 1 of login: validate password. Returns either the session cookies (no MFA) or a short-lived MFA challenge token (MFA enrolled). |
| POST | `/api/v1/auth/mfa/challenge` | Issue a WebAuthn challenge for the in-progress login. Used during the MFA step when the user has security keys enrolled. |
| POST | `/api/v1/auth/mfa/verify` | Step 2 of login: verify the TOTP code, backup code, or WebAuthn assertion against the challenge token. On success, sets the session cookies. |
| POST | `/api/v1/logout` | Invalidate the current session and clear cookies. |
| GET | `/api/v1/me` | Get the current user's profile. |
| PUT | `/api/v1/me/password` | Change the current user's password. Revokes all other sessions. |
| GET | `/api/v1/me/sessions` | List active sessions for the current user (device, IP, last active). |
| DELETE | `/api/v1/me/sessions/{id}` | Revoke a specific active session. |

### Login example (no MFA)

```bash
curl -i -c cookies.txt -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "supersecret"}'
```

The response sets two cookies:

```
Set-Cookie: access_token=...; HttpOnly; Path=/
Set-Cookie: csrf_token=...; Path=/
```

### Login example (MFA enrolled)

If the user has MFA enabled, the login response uses `status: "mfa_required"` and carries a short-lived `pending_token` plus a list of available factors. No cookies are set yet.

```json
{
  "status": "mfa_required",
  "pending_token": "...",
  "expires_at": "2026-04-23T10:05:00Z",
  "factors": [{"id": 42, "type": "totp"}]
}
```

Complete the login by submitting a code (TOTP or backup code) or a WebAuthn assertion to `/auth/mfa/verify`:

```bash
curl -i -c cookies.txt -X POST http://localhost:8080/api/v1/auth/mfa/verify \
  -H "Content-Type: application/json" \
  -d '{"pending_token": "...", "factor_id": 42, "code": "123456"}'
```

For WebAuthn factors, first call `/auth/mfa/challenge` with `{pending_token, factor_id}` to get the request options, then submit the browser's `PublicKeyCredential` JSON in the `webauthn_response` field of the verify call.

### Using the cookie

For subsequent calls, send back the cookie file. Mutating calls must add the CSRF token from the `csrf_token` cookie:

```bash
curl http://localhost:8080/api/v1/organizations -b cookies.txt
curl -X POST http://localhost:8080/api/v1/organizations \
  -b cookies.txt \
  -H "X-CSRF-Token: $(grep csrf_token cookies.txt | awk '{print $7}')" \
  -H "Content-Type: application/json" \
  -d '{"name": "New Org", "state": "berlin"}'
```

## Multi-Factor Authentication (Factors)

Users self-manage their factors via `/users/{userId}/factors`. The path parameter `:userId` accepts the literal `me` as a self-alias — addressing anyone but yourself returns 403.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users/{userId}/factors` | List enrolled factors. |
| POST | `/api/v1/users/{userId}/factors` | Enrol a new factor. The body specifies factor type (`totp` or `webauthn`) and current password. Returns enrolment payload (TOTP secret + otpauth URI, or WebAuthn registration challenge). |
| GET | `/api/v1/users/{userId}/factors/{id}` | Get a factor. |
| PATCH | `/api/v1/users/{userId}/factors/{id}` | Update the human label on a factor. |
| DELETE | `/api/v1/users/{userId}/factors/{id}` | Delete a factor. Requires password and current TOTP code. Deleting the last primary factor also sweeps the backup-codes factor. |
| POST | `/api/v1/users/{userId}/factors/{id}/activate` | Activate a pending enrolment with a verification code. First successful activation also returns single-use backup codes. |
| POST | `/api/v1/users/{userId}/factors/{id}/regenerate` | Regenerate the backup-codes factor; old codes are invalidated. |

## Organizations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/organizations` | List organizations |
| POST | `/api/v1/organizations` | Create organization (superadmin) |
| GET | `/api/v1/organizations/{orgId}` | Get organization |
| PUT | `/api/v1/organizations/{orgId}` | Update organization |
| DELETE | `/api/v1/organizations/{orgId}` | Delete organization (superadmin) |

## Sections

All section endpoints are scoped to an organization: `/api/v1/organizations/{orgId}/sections`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../sections` | List sections |
| POST | `.../sections` | Create section |
| GET | `.../sections/{sectionId}` | Get section |
| PUT | `.../sections/{sectionId}` | Update section |
| DELETE | `.../sections/{sectionId}` | Delete section |
| POST | `.../sections/{sectionId}/promote-default` | Mark this section as the default for new contracts (demotes the previous default) |

## Employees

All employee endpoints are scoped to an organization: `/api/v1/organizations/{orgId}/employees`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../employees` | List employees |
| POST | `.../employees` | Create employee |
| GET | `.../employees/{id}` | Get employee |
| PUT | `.../employees/{id}` | Update employee |
| DELETE | `.../employees/{id}` | Delete employee |
| GET | `.../employees/export/excel` | Export employees to Excel |
| GET | `.../employees/export/yaml` | Export employees to YAML |
| POST | `.../employees/import` | Import employees from YAML |
| GET | `.../employees/step-promotions` | Get step promotion eligibility |

### Employee contracts

Nested under an employee: `.../employees/{id}/contracts`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../contracts` | List contracts |
| POST | `.../contracts` | Create contract |
| GET | `.../contracts/current` | Get current active contract |
| GET | `.../contracts/{contractId}` | Get contract |
| PUT | `.../contracts/{contractId}` | Update contract |
| PUT | `.../contracts/batch` | Batch-update employee contracts (used by the contract-amend flow) |
| DELETE | `.../contracts/{contractId}` | Delete contract |

## Children

All child endpoints are scoped to an organization: `/api/v1/organizations/{orgId}/children`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../children` | List children |
| POST | `.../children` | Create child |
| GET | `.../children/{id}` | Get child |
| PUT | `.../children/{id}` | Update child |
| DELETE | `.../children/{id}` | Delete child |
| GET | `.../children/export/excel` | Export children to Excel |
| GET | `.../children/export/yaml` | Export children to YAML |
| POST | `.../children/import` | Import children from YAML |
| GET | `.../children/attendance` | Org-wide attendance by date |
| GET | `.../children/attendance/summary` | Daily attendance summary |
| GET | `.../children/without-vouchers` | List children with a contract but no Gutscheinnummer |
| GET | `.../children/billing-summary` | Org-wide billing summary across uploaded bills |
| POST | `.../children/{childId}/vouchers` | Assign or update the Kita-Gutschein number on a child |
| GET | `.../children/{childId}/billing-history` | Per-child history of every billed amount across all uploaded bills |

### Child contracts

Nested under a child: `.../children/{id}/contracts`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../contracts` | List contracts |
| POST | `.../contracts` | Create contract |
| GET | `.../contracts/current` | Get current active contract |
| GET | `.../contracts/{contractId}` | Get contract |
| PUT | `.../contracts/{contractId}` | Update contract |
| PUT | `.../contracts/batch` | Batch-update child contracts (used by the contract-amend flow) |
| DELETE | `.../contracts/{contractId}` | Delete contract |

### Child attendance

Nested under a child: `.../children/{id}/attendance`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `.../attendance` | Create attendance record |
| GET | `.../attendance` | List child's attendance records |
| GET | `.../attendance/{attendanceId}` | Get attendance record |
| PUT | `.../attendance/{attendanceId}` | Update attendance record |
| DELETE | `.../attendance/{attendanceId}` | Delete attendance record |

## Government funding rates

Global resource managed by superadmins.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/government-funding-rates` | List funding configurations |
| POST | `/api/v1/government-funding-rates` | Create funding configuration |
| GET | `/api/v1/government-funding-rates/{id}` | Get funding configuration |
| PUT | `/api/v1/government-funding-rates/{id}` | Update funding configuration |
| DELETE | `/api/v1/government-funding-rates/{id}` | Delete funding configuration |
| POST | `/api/v1/government-funding-rates/import` | Import funding rates from YAML |

### Funding periods

Nested under a funding rate: `.../government-funding-rates/{id}/periods`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../periods` | List periods |
| POST | `.../periods` | Create period |
| GET | `.../periods/{periodId}` | Get period |
| PUT | `.../periods/{periodId}` | Update period |
| DELETE | `.../periods/{periodId}` | Delete period |

### Funding properties

Nested under a period: `.../periods/{periodId}/properties`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../properties` | List properties |
| POST | `.../properties` | Create property |
| GET | `.../properties/{propertyId}` | Get property |
| PUT | `.../properties/{propertyId}` | Update property |
| DELETE | `.../properties/{propertyId}` | Delete property |

## Government funding bills (ISBJ)

Scoped to an organization: `/api/v1/organizations/{orgId}/government-funding-bills`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../government-funding-bills` | List bills |
| POST | `.../government-funding-bills` | Upload ISBJ bill |
| GET | `.../government-funding-bills/compare` | Org-wide multi-bill comparison (across all uploaded bills for the org) |
| GET | `.../government-funding-bills/{billId}` | Get bill |
| GET | `.../government-funding-bills/{billId}/compare` | Compare calculated vs. billed amounts for one bill |
| DELETE | `.../government-funding-bills/{billId}` | Delete bill |

## Pay plans

Scoped to an organization: `/api/v1/organizations/{orgId}/pay-plans`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../pay-plans` | List pay plans |
| POST | `.../pay-plans` | Create pay plan |
| GET | `.../pay-plans/{id}` | Get pay plan |
| PUT | `.../pay-plans/{id}` | Update pay plan |
| DELETE | `.../pay-plans/{id}` | Delete pay plan |
| GET | `.../pay-plans/{id}/export` | Export pay plan to YAML |
| POST | `.../pay-plans/import` | Import pay plan from YAML |

### Pay plan periods

Nested under a pay plan: `.../pay-plans/{id}/periods`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../periods` | List periods |
| POST | `.../periods` | Create period |
| GET | `.../periods/{periodId}` | Get period |
| PUT | `.../periods/{periodId}` | Update period |
| DELETE | `.../periods/{periodId}` | Delete period |

### Pay plan entries

Nested under a period: `.../periods/{periodId}/entries`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../entries` | List entries |
| POST | `.../entries` | Create entry |
| GET | `.../entries/{entryId}` | Get entry |
| PUT | `.../entries/{entryId}` | Update entry |
| DELETE | `.../entries/{entryId}` | Delete entry |

## Budget items

Scoped to an organization: `/api/v1/organizations/{orgId}/budget-items`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../budget-items` | List budget items |
| POST | `.../budget-items` | Create budget item |
| GET | `.../budget-items/{id}` | Get budget item |
| PUT | `.../budget-items/{id}` | Update budget item |
| DELETE | `.../budget-items/{id}` | Delete budget item |

### Budget item entries

Nested under a budget item: `.../budget-items/{id}/entries`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../entries` | List entries |
| POST | `.../entries` | Create entry |
| GET | `.../entries/{entryId}` | Get entry |
| PUT | `.../entries/{entryId}` | Update entry |
| DELETE | `.../entries/{entryId}` | Delete entry |

## Statistics

Scoped to an organization: `/api/v1/organizations/{orgId}/statistics`. All statistics endpoints require `from` and `to` query parameters specifying a date range (format: `YYYY-MM-DD`).

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `.../statistics/staffing-hours` | Staffing hours summary |
| GET | `.../statistics/staffing-hours/employees` | Per-employee staffing detail |
| GET | `.../statistics/financials` | Financial overview |
| GET | `.../statistics/occupancy` | Occupancy statistics |
| GET | `.../statistics/age-distribution` | Age distribution |
| GET | `.../statistics/contract-properties` | Contract property distribution |
| GET | `.../statistics/funding` | Funding statistics |

### Forecast and estimates

These endpoints power the **Forecast** page. They take the same date range plus a list of scenario modifications (added/removed children, added/removed employees) and return the projected statistics with the modifications applied.

{{< callout type="warning" >}}
The `estimates/child-funding` and `estimates/employee-cost` endpoints are currently parked pending a domain-model refactor. The `forecast` endpoint works against current data; the per-child / per-employee estimates may return errors or stale calculations.
{{< /callout >}}

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `.../statistics/forecast` | Project the full Kita year with the supplied scenario layered on top of current data. |
| POST | `.../statistics/estimates/child-funding` | Estimate monthly funding for a hypothetical child without affecting stored data. |
| POST | `.../statistics/estimates/employee-cost` | Estimate monthly cost (gross + employer contributions) for a hypothetical employee. |

## Audit logs

Two scopes:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/audit-logs` | **Superadmin-only** global audit log including login/auth events. Filters: `from`, `to`, `action`, `user_id`. |
| GET | `/api/v1/audit-logs/{id}` | **Superadmin-only** get a specific audit log entry. |
| GET | `/api/v1/organizations/{orgId}/audit-logs` | **Org admin** view, scoped to one organization. Login/password events are excluded. |

## Users

Global user management.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users` | List users |
| POST | `/api/v1/users` | Create user |
| GET | `/api/v1/users/{id}` | Get user |
| PUT | `/api/v1/users/{id}` | Update user |
| DELETE | `/api/v1/users/{id}` | Delete user |
| GET | `/api/v1/users/{id}/memberships` | Get user's organization memberships |
| POST | `/api/v1/users/{id}/organizations` | Add user to organization |
| PUT | `/api/v1/users/{id}/organizations/{orgId}` | Update user's role in organization |
| DELETE | `/api/v1/users/{id}/organizations/{orgId}` | Remove user from organization |
| PUT | `/api/v1/users/{id}/password` | Reset user's password (admin) |
| PUT | `/api/v1/users/{id}/superadmin` | Set superadmin status |

### Organization users

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/organizations/{orgId}/users` | List users in an organization |

## Pagination

List endpoints support pagination via query parameters:

```bash
curl "http://localhost:8080/api/v1/organizations?page=1&limit=10" -b cookies.txt
```

Response shape:

```json
{
  "data": [],
  "total": 100,
  "page": 1,
  "limit": 10
}
```

## Error responses

| Status | Meaning |
|--------|---------|
| 400 | Bad Request — invalid input or missing required parameters |
| 401 | Unauthorized — missing or invalid authentication |
| 403 | Forbidden — insufficient permissions |
| 404 | Not Found — resource does not exist |
| 500 | Internal Server Error — unexpected error |

Error body:

```json
{
  "error": "Description of the error"
}
```
