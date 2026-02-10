---
description: Conventions for creating or modifying API endpoints, handlers, routes, DTOs, and services
globs:
  - internal/handlers/**/*.go
  - internal/routes/**/*.go
  - internal/models/**/*.go
  - internal/service/**/*.go
  - internal/store/**/*.go
  - internal/rbac/**/*.go
  - cmd/api/main.go
---

# API Endpoint Conventions

Follow these conventions when creating or modifying API endpoints. New endpoints MUST be consistent with existing endpoints in the codebase — before implementing, review similar existing handlers (e.g., child, employee, organization) and match their patterns exactly.

## REST API Design

- **Resource-oriented URLs only** — no RPC-style action verbs (`/check-in`, `/activate`, `/approve`). Use HTTP methods: `POST` (create), `GET` (read), `PUT` (update), `DELETE` (delete).
- **URL param naming**: `:id` for the parent resource, named params for sub-resources (`:contractId`, `:attendanceId`, `:periodId`).
  ```
  /organizations/:orgId/employees/:id/contracts/:contractId
  ```

## HTTP Responses

- **204 No Content must NOT have a body**:
  ```go
  // CORRECT
  c.Status(http.StatusNoContent)

  // WRONG — sends body with 204, caught by forbidigo lint rule
  c.JSON(http.StatusNoContent, nil)
  ```
- **201 Created** for successful POST that creates a resource.
- **200 OK** for GET and PUT responses.

## DTO Naming

Follow `{Resource}{Action}Request` / `{Resource}Response`:
- `ChildCreateRequest`, `ChildUpdateRequest`, `ChildResponse`
- `ChildContractCreateRequest` (not `CreateChildContractRequest`)

Include `example` tags on all DTO fields for swagger docs.

## Required Query Params

Required query parameters must be validated with `parseRequiredDate` (or equivalent). Never silently default a required param.

## Handler Structure

1. Parse and validate URL params (`parseID`, `parseOrgAndResourceID`)
2. Bind and validate request body (`c.ShouldBindJSON`)
3. Call service method
4. Return response

## Swagger Annotations

Every handler MUST have swaggo annotations:
```go
// Create godoc
// @Summary Short description
// @Description Detailed description
// @Tags tag-name
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Resource ID"
// @Param request body models.ResourceCreateRequest true "Data"
// @Success 201 {object} models.ResourceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/resources [post]
```

## Audit Logging

All delete handlers MUST:
1. Fetch the resource before deletion (for audit context)
2. Perform the delete
3. Log via `h.auditService.LogResourceDelete(actorID, "resource_type", id, resourceName, c.ClientIP())`

The handler struct must include `auditService *service.AuditService` and receive it in the constructor.

## RBAC

### Resource constants

When adding a new resource type, add a constant in `internal/rbac/rbac.go`:
```go
const (
    ResourceMyResource = "my_resource"
)
```

Then add policies for all four roles in `SeedDefaultPolicies()`:
- **superadmin**: full CRUD (`create`, `read`, `update`, `delete`)
- **admin**: full CRUD within their org
- **manager**: decide per resource — typically full CRUD for operational resources, read-only for config
- **member**: typically read-only

### Route middleware

Choose the right middleware based on access pattern:

```go
// Org-scoped resource — most common. Checks role permission within the org from :orgId param.
authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionRead)

// Global resource (not org-scoped, e.g. users). Checks permission in any org the user belongs to.
authzMiddleware.RequireGlobalPermission(rbac.ResourceX, rbac.ActionRead)

// Superadmin-only (e.g. create/delete organizations, government fundings).
authzMiddleware.RequireSuperAdmin()
```

### Routes registration

Register in `internal/routes/routes.go`:
```go
resource.POST("", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionCreate), handler.Create)
resource.GET("", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionRead), handler.List)
resource.GET("/:subId", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionRead), handler.Get)
resource.PUT("/:subId", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionUpdate), handler.Update)
resource.DELETE("/:subId", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionDelete), handler.Delete)
```

## Testing

### Service tests

Service tests use a real SQLite database (not mocks). Follow this pattern:

```go
func setupMyResourceTest(t *testing.T) (*MyResourceService, *store.MyResourceStore, *models.Organization) {
    t.Helper()
    db := setupTestDB(t)
    db.AutoMigrate(&models.MyResource{})

    myStore := store.NewMyResourceStore(db)
    svc := NewMyResourceService(myStore)

    org := createTestOrganization(t, db, "Test Org")
    return svc, myStore, org
}
```

### What to test

For each service method, test:
- **Happy path** — valid input returns expected result
- **Not found** — non-existent ID returns `apperror.ErrNotFound`
- **Wrong org** — resource from different org returns `apperror.ErrNotFound`
- **Wrong parent** — resource belonging to a different parent returns `apperror.ErrNotFound` (e.g., attendance for wrong child)
- **Invalid input** — bad data returns `apperror.ErrBadRequest`
- **Duplicates** — if applicable, returns `apperror.ErrConflict`

### Edge cases to cover

- Empty strings and whitespace-only input (notes, names)
- Nil pointer fields in partial update requests
- Boundary values (zero IDs, max pagination limits)
- Cross-resource isolation (resource from org A not accessible via org B)
- Delete followed by get (verify resource is actually gone)
- Custom time values vs default time values (e.g., explicit check-in time vs auto-now)

### Test naming

Use `TestServiceName_MethodName_Scenario`:
```go
func TestMyResourceService_Create(t *testing.T) { ... }
func TestMyResourceService_Create_ChildNotFound(t *testing.T) { ... }
func TestMyResourceService_Delete_WrongOrg(t *testing.T) { ... }
```

### Error assertions

Use `errors.Is` with sentinel errors from `apperror`:
```go
if !errors.Is(err, apperror.ErrNotFound) {
    t.Errorf("expected ErrNotFound, got %v", err)
}
```

## Files to Create/Modify

- `internal/models/` — request/response DTOs
- `internal/store/` — database operations + interface in `interfaces.go`
- `internal/service/` — business logic
- `internal/handlers/` — HTTP handlers with swagger annotations
- `internal/routes/routes.go` — route registration
- `internal/rbac/rbac.go` — add resource constant + seed policies for all roles
- `cmd/api/main.go` — wire up store, service, handler
- `internal/service/*_test.go` — service tests (see Testing section above)

After implementation, run: `go build ./...` and `go test ./...` to verify.
