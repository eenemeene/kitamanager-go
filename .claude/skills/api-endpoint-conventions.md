---
description: Conventions for creating or modifying API endpoints, handlers, routes, DTOs, and services
globs:
  - internal/handlers/**/*.go
  - internal/routes/**/*.go
  - internal/models/**/*.go
  - internal/service/**/*.go
  - cmd/api/main.go
---

# API Endpoint Conventions

Follow these conventions when creating or modifying API endpoints.

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

## Routes

Register in `internal/routes/routes.go` with appropriate RBAC middleware:
```go
resource.POST("", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionCreate), handler.Create)
resource.GET("", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionRead), handler.List)
resource.GET("/:subId", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionRead), handler.Get)
resource.PUT("/:subId", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionUpdate), handler.Update)
resource.DELETE("/:subId", authzMiddleware.RequirePermission(rbac.ResourceX, rbac.ActionDelete), handler.Delete)
```

## Files to Create/Modify

- `internal/models/` — request/response DTOs
- `internal/store/` — database operations + interface in `interfaces.go`
- `internal/service/` — business logic
- `internal/handlers/` — HTTP handlers with swagger annotations
- `internal/routes/routes.go` — route registration
- `internal/rbac/rbac.go` — add resource constant if new
- `cmd/api/main.go` — wire up store, service, handler
- `internal/service/*_test.go` — service tests

After implementation, run: `go build ./...` and `go test ./...` to verify.
