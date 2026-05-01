---
paths:
  - "internal/handlers/**/*.go"
---

# API handler conventions

Loaded when working in `internal/handlers/`. The wider Go conventions in `go-conventions.md` also apply.

## swaggo annotations are mandatory

Every handler MUST be documented with swaggo annotations. Run `swag init -g cmd/api/main.go` after adding or changing handlers — this regenerates `docs/`.

```go
// Create godoc
// @Summary Create a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UserCreateRequest true "User data"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) { ... }
```

## DTO naming

**Request DTOs**: `{Resource}{Action}Request`
- Create: `UserCreateRequest`, `ChildCreateRequest`, `EmployeeContractCreateRequest`
- Update: `UserUpdateRequest`, `ChildUpdateRequest`, `FundingPeriodUpdateRequest`
- Other actions: `AssignFundingRequest`, `SetSuperAdminRequest`

**Response DTOs**: `{Resource}Response` — `UserResponse`, `ChildResponse`, etc.

**Nested resources** follow the same pattern: `ChildContractCreateRequest` (not `CreateChildContractRequest`), `FundingEntryUpdateRequest` (not `UpdateFundingEntryRequest`).

**Don't use:** `Create{Resource}Request`, `Update{Resource}Request`, or `{Resource}Create` (missing the `Request` suffix).

Add `example` tags on every DTO field for swagger docs:
```go
type UserCreateRequest struct {
    Name     string `json:"name" binding:"required" example:"John Doe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Password string `json:"password" binding:"required,min=6" example:"secret123"`
}
```

## REST conventions

### Resource-oriented endpoints — no RPC verbs

```
# GOOD
POST   /children/:id/attendance
PUT    /children/:id/attendance/:attendanceId

# BAD
POST   /children/:id/attendance/check-in
PUT    /children/:id/attendance/:id/check-out
POST   /children/:id/attendance/absent
```

### URL parameter naming

Nested resources: `:id` for the parent, named param (`:contractId`, `:attendanceId`, `:periodId`) for the child. Matches how Gin resolves route parameters.

```
/organizations/:orgId/employees/:id/contracts/:contractId
/organizations/:orgId/children/:id/attendance/:attendanceId
```

### 204 No Content has no body

```go
// CORRECT
c.Status(http.StatusNoContent)

// WRONG — sends a body with 204
c.JSON(http.StatusNoContent, nil)
```

### Required query params must be validated

Don't silently default a required param.
```go
// CORRECT
from, ok := parseRequiredDate(c, "from")

// WRONG
from, ok := parseOptionalDate(c, "from")  // when from is required
```

## Audit logging

Every mutating handler MUST emit an audit event, at minimum for delete:

```go
// Get resource info before deletion for audit log
resource, err := h.service.GetByID(ctx, id, orgID, parentID)
// ... perform delete ...
h.auditService.LogResourceDelete(actorID, "resource_type", id, resource.Name, c.ClientIP())
```
