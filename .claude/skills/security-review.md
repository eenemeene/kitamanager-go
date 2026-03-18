---
description: Security review for code changes or files in this repo — covers auth, RBAC, audit logging, GDPR/DSGVO, and known vulnerability classes
globs:
  - internal/**/*.go
  - frontend/src/**/*.ts
  - frontend/src/**/*.tsx
  - internal/middleware/**/*.go
  - internal/handlers/**/*.go
  - internal/models/**/*.go
---

# Security Review

Perform a thorough security review of the code indicated by the user (or, if no specific code is given, review recent changes via `git diff HEAD~1`). This app handles **children's personal data, employee financial records, and government funding data** — all subject to German DSGVO (GDPR) and requiring the highest security standards.

## How to Run the Review

1. Read the relevant code files carefully.
2. Check each category below systematically.
3. Output a structured report (see Report Format at the bottom).
4. For each finding, include the file path and line number, severity, description, and a concrete fix.

---

## Checklist

### 1. Authentication & Session Security

- [ ] **All non-public routes** are behind `AuthMiddleware`. Check `internal/routes/routes.go` — no handler group is missing the middleware.
- [ ] **JWT not stored in localStorage** — frontend must use httpOnly cookies or memory only. Flag any use of `localStorage.setItem` / `sessionStorage.setItem` with tokens in `frontend/src/stores/auth.ts` or `frontend/src/lib/api/client.ts`.
- [ ] **Token expiry** — access tokens should be short-lived (15 min target). Flag if `ExpiresAt` is set to > 1 hour in `internal/handlers/auth.go`.
- [ ] **Refresh token flow** — check if refresh tokens are properly rotated and invalidated on logout.
- [ ] **Password validation** — minimum 8 chars, max 72 chars (bcrypt limit). Check `internal/handlers/auth.go` and relevant DTOs.
- [ ] **No credentials in error messages** — authentication failures must return generic messages (e.g., "invalid credentials"), never reveal whether email or password was wrong.

### 2. Authorization & RBAC

- [ ] **Every mutating route** has `authzMiddleware.RequirePermission(...)` or `authzMiddleware.RequireSuperAdmin()`. Routes without authorization middleware are a critical vulnerability.
- [ ] **Org-scoped routes verify orgId** — for any handler accessing org-scoped data, confirm the service layer filters by `orgID` (from the URL param, not just from the JWT). Cross-org data leakage is a critical issue.
- [ ] **New resources have Casbin policies** — check `internal/rbac/rbac.go` `SeedDefaultPolicies()`. Every new resource type must have policies for all four roles: superadmin, admin, manager, member.
- [ ] **Superadmin boolean not bypassable** — `IsSuperAdmin` check in DB must not be spoofable. Confirm it reads from the database, not from JWT claims alone.
- [ ] **No IDOR (Insecure Direct Object Reference)** — when fetching a resource by ID, the service must also filter by `orgID`. A user from org A must not be able to access data from org B by guessing IDs.

### 3. Audit Logging

- [ ] **Delete operations** — every `DELETE` handler must: (1) fetch the resource before deleting, (2) call `h.auditService.LogResourceDelete(actorID, "resource_type", id, name, c.ClientIP())`. Missing audit logs on deletes are a compliance violation.
- [ ] **Failed login attempts** — `internal/handlers/auth.go` must log failed logins with IP address and timestamp.
- [ ] **Superadmin operations** — role changes, org creation/deletion, and user management must be audit logged.
- [ ] **No PII in log messages** — logs must not contain names, birthdates, addresses, or other personal data. IDs are fine; full objects are not.
- [ ] **Audit log immutability** — the audit log table should not be deletable by any non-superadmin. Check that no handler exists to delete audit log entries.

### 4. Data Protection (DSGVO/GDPR)

This app stores data about **minors (children)** which receives the highest level of protection under German law.

- [ ] **Minimal data collection** — new fields on child records must have a documented legal basis. Flag any new PII fields (name, birthdate, address, nationality, special needs) being added without comment explaining why they're needed.
- [ ] **Soft deletes for child and employee records** — physical deletes of personal data may violate DSGVO retention requirements. Check if `deleted_at` (GORM soft delete) is used. Hard deletes of personal records should be flagged.
- [ ] **No PII in URLs** — names, email addresses, birthdates must not appear in URL parameters (only IDs).
- [ ] **No PII in API error responses** — error messages returned to clients must not include personal data from other users/children.
- [ ] **Data export/access controls** — any endpoint that exports bulk data (Excel, CSV) must be restricted to admin/superadmin roles.

### 5. Input Validation & Injection

- [ ] **Request body binding uses `ShouldBindJSON` with struct validation tags** — raw `c.Query()` or `c.Param()` values used in business logic without validation are risky.
- [ ] **No raw SQL** — all database queries must use GORM's query builder or parameterized queries. Flag any `db.Raw(...)` or `db.Exec(...)` calls that concatenate user input.
- [ ] **Integer parsing** — URL params like `:id` and `:orgId` must be parsed with `strconv.Atoi` or the project's `parseID()` helper, not used as strings. Verify no direct string interpolation into queries.
- [ ] **Pagination limits** — `limit` query params must be capped (e.g., max 500). Unlimited pagination can lead to DoS or data exfiltration.

### 6. HTTP Security

- [ ] **204 responses have no body** — `c.JSON(http.StatusNoContent, ...)` is forbidden; use `c.Status(http.StatusNoContent)`.
- [ ] **Security headers middleware** is applied to all routes — check `internal/middleware/security.go` is registered in `cmd/api/main.go`.
- [ ] **CORS configuration** — in production, `AllowOrigins` must not be `["*"]`. Check `internal/config/` for environment-based CORS config.
- [ ] **Rate limiting on auth endpoints** — `/api/v1/login` and `/api/v1/refresh` must have rate limiting middleware applied. Check `internal/routes/routes.go`.
- [ ] **Request size limits** — body size limit middleware should be applied. Large uploads to non-upload endpoints are a DoS vector.
- [ ] **Trusted proxy configuration** — `c.ClientIP()` used in audit logs must come from a trusted header only. Check Gin's trusted proxy settings in `cmd/api/main.go`.

### 7. Sensitive Data Handling

- [ ] **Monetary values stored as cents (integers)** — any new financial field must use `int` type, never `float64`. Floating point is prohibited per project conventions.
- [ ] **Passwords never logged or returned** — `User` structs returned from handlers must not include password hashes. Check `json:"-"` tags on password fields in models.
- [ ] **No secrets in code** — no hardcoded JWT secrets, DB passwords, or API keys. All secrets must come from environment variables via `internal/config/`.
- [ ] **Context timeouts on DB operations** — long-running service methods should use `context.WithTimeout()` to prevent unbounded DB query times.

### 8. Frontend Security

- [ ] **No `dangerouslySetInnerHTML`** with user-supplied content — XSS vector.
- [ ] **API error messages not rendered as HTML** — error text from the API must be escaped before display.
- [ ] **Auth state in memory, not localStorage** — JWT access tokens should not persist in localStorage across sessions.
- [ ] **CSRF tokens sent on mutations** — POST/PUT/DELETE requests should include the CSRF token header.
- [ ] **No sensitive data in browser console logs** — `console.log` must not output tokens, user PII, or financial data.

---

## Report Format

Output your findings in this structure:

```
## Security Review Report

### Critical Findings (fix before merge)
- [CRIT-1] file.go:42 — <issue description>
  Fix: <concrete fix>

### High Findings (fix soon)
- [HIGH-1] file.go:88 — <issue description>
  Fix: <concrete fix>

### Medium Findings (track in SECURITY-TODO.md)
- [MED-1] file.go:12 — <issue description>
  Fix: <concrete fix>

### Informational
- [INFO-1] <observation with no immediate action needed>

### Passed Checks
- Auth middleware: ✓
- Org-scoped IDOR prevention: ✓
- ...
```

If no issues are found in a severity level, say "None found."

After the report, state whether the code is **safe to merge** (no critical/high issues), **merge with caution** (medium issues only), or **do not merge** (critical or high issues present).
