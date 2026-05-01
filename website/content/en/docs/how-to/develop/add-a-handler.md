---
title: Add a REST handler
weight: 2
---

You want to add a new REST endpoint following the project's conventions (RBAC, swaggo annotations, audit log, DTOs).

## Steps

### 1. Define the request and response DTOs

In `internal/models/<resource>_dto.go`:

```go
type WidgetCreateRequest struct {
    Name string `json:"name" binding:"required" example:"My Widget"`
}

type WidgetResponse struct {
    ID   uint   `json:"id" example:"1"`
    Name string `json:"name" example:"My Widget"`
}
```

Use the `{Resource}{Action}Request` naming. Add `example` tags for swagger.

### 2. Add the service method

In `internal/service/<resource>.go`:

```go
func (s *WidgetService) Create(ctx context.Context, orgID uint, req models.WidgetCreateRequest) (*models.Widget, error) {
    // business logic
}
```

### 3. Add the handler with swaggo annotations

In `internal/handlers/<resource>.go`:

```go
// Create godoc
// @Summary Create a widget
// @Description Create a widget within an organization
// @Tags widgets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.WidgetCreateRequest true "Widget data"
// @Success 201 {object} models.WidgetResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/widgets [post]
func (h *WidgetHandler) Create(c *gin.Context) {
    // bind, validate, call service, return
    // include audit log call before returning success
    h.auditService.LogResourceCreate(actorID, "widget", widget.ID, widget.Name, c.ClientIP())
}
```

### 4. Wire the route with RBAC

In `internal/routes/routes.go`:

```go
widgets := orgRouter.Group("/widgets",
    authzMiddleware.RequirePermission(rbac.ResourceWidgets, rbac.ActionWrite))
widgets.POST("", h.WidgetHandler.Create)
```

### 5. Regenerate the swagger doc and TypeScript types

```bash
make swagger-docs
make api-types
```

Commit the regenerated `docs/swagger.{json,yaml}` and `frontend/src/lib/api/openapi.d.ts`.

### 6. Add tests

- Unit tests for the service in `internal/service/<resource>_test.go`.
- Handler tests in `internal/handlers/<resource>_test.go` (set up via `testutil`).
- Integration test in `internal/integration/` if the path crosses RBAC + DB.

## Notes

- The full handler/RBAC/audit conventions are in `.claude/rules/api-handlers.md`. That file is the source of truth.
- For the route URL pattern (`/organizations/{orgId}/...` for org-scoped resources), see [Reference: RBAC](../../../reference/rbac/#url-pattern-for-organisation-scoped-resources).
- Don't `c.JSON(http.StatusNoContent, nil)` — use `c.Status(http.StatusNoContent)`. Sending a body with 204 violates RFC 7230 and several HTTP clients trip on it.
