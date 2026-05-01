---
title: REST-Handler hinzufügen
weight: 2
---

Sie wollen einen neuen REST-Endpunkt hinzufügen, der die Projekt-Konventionen einhält (RBAC, swaggo-Annotationen, Audit-Log, DTOs).

## Schritte

### 1. Request- und Response-DTOs definieren

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

Nutzen Sie die Benennung `{Resource}{Action}Request`. Fügen Sie `example`-Tags für swagger hinzu.

### 2. Service-Methode hinzufügen

In `internal/service/<resource>.go`:

```go
func (s *WidgetService) Create(ctx context.Context, orgID uint, req models.WidgetCreateRequest) (*models.Widget, error) {
    // Geschäftslogik
}
```

### 3. Handler mit swaggo-Annotationen hinzufügen

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
    // Bind, validieren, Service aufrufen, zurückgeben
    // Audit-Log-Aufruf vor dem Erfolgs-Return einbauen
    h.auditService.LogResourceCreate(actorID, "widget", widget.ID, widget.Name, c.ClientIP())
}
```

### 4. Route mit RBAC verdrahten

In `internal/routes/routes.go`:

```go
widgets := orgRouter.Group("/widgets",
    authzMiddleware.RequirePermission(rbac.ResourceWidgets, rbac.ActionWrite))
widgets.POST("", h.WidgetHandler.Create)
```

### 5. Swagger-Doc und TypeScript-Typen neu generieren

```bash
make docs        # Sammelziel: regeneriert swagger + Schema-Doku
make api-types   # regeneriert die TypeScript-API-Typen aus dem neuen swagger
```

Die regenerierten `docs/swagger.{json,yaml}`, `docs/schema/` und `frontend/src/lib/api/openapi.d.ts` mit committen.

### 6. Tests hinzufügen

- Unit-Tests für den Service in `internal/service/<resource>_test.go`.
- Handler-Tests in `internal/handlers/<resource>_test.go` (über `testutil` aufgesetzt).
- Integrationstest in `internal/integration/`, wenn der Pfad RBAC + DB kreuzt.

## Hinweise

- Die vollständigen Handler-/RBAC-/Audit-Konventionen liegen in `.claude/rules/api-handlers.md`. Diese Datei ist die maßgebliche Quelle.
- Für das Routen-URL-Schema (`/organizations/{orgId}/...` für org-bezogene Ressourcen) siehe [Referenz: RBAC](../../../reference/rbac/) (Abschnitt URL-Schema).
- Nicht `c.JSON(http.StatusNoContent, nil)` schreiben — `c.Status(http.StatusNoContent)` nutzen. Einen Body mit 204 zu senden verstößt gegen RFC 7230 und mehrere HTTP-Clients stolpern darüber.
