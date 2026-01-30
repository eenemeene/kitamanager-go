# API Architecture Improvements

This document tracks architectural issues and improvements identified in the codebase.

## High Priority

### 1. Inconsistent Contract Properties Architecture

**Problem:** Employee and Child contracts handle properties/attributes differently:

- **Employee contracts:** Relational model with `EmployeeContractProperty` table and full CRUD endpoints
  - `GET/POST /employees/{id}/contracts/{contractId}/properties`
  - `PUT/DELETE /employees/{id}/contracts/{contractId}/properties/{propId}`

- **Child contracts:** Denormalized JSON array (`Attributes []string`), no property endpoints

**Impact:** API inconsistency, different query capabilities, confusing for API consumers.

**Recommendation:** Unify the approach - either use relational properties for both, or JSON for both.

---

## Medium Priority

### 2. DTO Naming Convention Violations

Per CLAUDE.md, DTOs should follow `{Resource}{Action}Request` pattern. These violate:

| Current Name | Should Be |
|--------------|-----------|
| `AddUserToGroupRequest` | `UserGroupAssignRequest` |
| `UpdateUserGroupRoleRequest` | `UserGroupRoleUpdateRequest` |
| `SetSuperAdminRequest` | `UserSuperAdminUpdateRequest` |
| `AddToOrganizationRequest` | `UserOrganizationAssignRequest` |

**Location:** `internal/models/user.go`, `internal/models/user_group.go`

---

### 3. Service Layer Coupling Inconsistency

Services have inconsistent dependency patterns:

```go
// ChildService has 3 dependencies
type ChildService struct {
    store        store.ChildStorer
    orgStore     store.OrganizationStorer      // Why?
    fundingStore store.GovernmentFundingStorer // Why?
}

// EmployeeService has 1 dependency
type EmployeeService struct {
    store store.EmployeeStorer
}
```

**Question:** If funding calculation is needed for children, shouldn't employees have similar capability?

**Recommendation:** Document why dependencies differ, or standardize the pattern.

---

### 4. Incomplete Audit Logging

Current state:
- Child/Employee deletion: Logged
- Child/Employee creation: **NOT logged**
- Child/Employee updates: **NOT logged**
- Contract operations: **NOT logged**

**Recommendation:** Add audit logging to all mutating operations, or document which operations intentionally skip logging.

---

### 5. Missing Pagination on Property Endpoints

`ListContractProperties` returns all properties without pagination:

```go
// internal/handlers/employee.go
func (h *EmployeeHandler) ListContractProperties(c *gin.Context) {
    properties, err := h.service.ListContractProperties(...)
    c.JSON(http.StatusOK, properties)  // No pagination!
}
```

**Recommendation:** Add pagination to property list endpoints.

---

### 6. Inconsistent Error Response Formats

Handlers use structured `ErrorResponse`:
```go
models.ErrorResponse{Code: "error_code", Message: "..."}
```

Middleware uses plain maps:
```go
gin.H{"error": "..."}
```

**Recommendation:** Use `ErrorResponse` consistently in middleware.

---

## Low Priority

### 7. Query Parameter Naming Inconsistency

Different endpoints use different naming conventions:
- `section_id` (snake_case)
- `min_year`, `max_year` (snake_case but different pattern)
- `date` (no prefix)

**Recommendation:** Document query parameter naming convention.

---

### 8. Mixed Response Types

Some endpoints return raw models, others return Response wrappers:
- `ChildHandler.Get` returns `Child` model directly
- `UserHandler.Get` returns `UserResponse` wrapper

**Recommendation:** Document when to use Response wrappers vs raw models.

---

## Completed

- [x] RBAC system cleanup and documentation (see `docs/RBAC.md`)
