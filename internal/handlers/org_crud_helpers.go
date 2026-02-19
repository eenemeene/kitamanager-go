package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// handleOrgList handles paginated listing of org-scoped resources.
func handleOrgList[R any](
	c *gin.Context,
	listFn func(ctx context.Context, orgID uint, search string, limit, offset int) ([]R, int64, error),
) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	search, ok := parseSearch(c)
	if !ok {
		return
	}

	items, total, err := listFn(c.Request.Context(), orgID, search, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(items, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// handleOrgGet handles fetching a single org-scoped resource by ID.
func handleOrgGet[R any](
	c *gin.Context,
	idParam string,
	getFn func(ctx context.Context, id, orgID uint) (*R, error),
) {
	orgID, resourceID, ok := parseOrgAndResourceID(c, idParam)
	if !ok {
		return
	}

	item, err := getFn(c.Request.Context(), resourceID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

// handleOrgCreate handles creating a new org-scoped resource with audit logging.
func handleOrgCreate[Req any, R any](
	c *gin.Context,
	audit auditConfig,
	createFn func(ctx context.Context, orgID uint, req *Req, createdBy string) (*R, error),
	getAuditInfo func(*R) (id uint, name string),
) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := createFn(c.Request.Context(), orgID, req, getCreatedBy(c))
	if err != nil {
		respondError(c, err)
		return
	}

	id, name := getAuditInfo(resp)
	auditCreate(c, audit.auditService, audit.resourceType, id, name)

	c.JSON(http.StatusCreated, resp)
}

// handleOrgUpdate handles updating an org-scoped resource with audit logging.
func handleOrgUpdate[Req any, R any](
	c *gin.Context,
	idParam string,
	audit auditConfig,
	updateFn func(ctx context.Context, id, orgID uint, req *Req) (*R, error),
	getAuditInfo func(*R) (id uint, name string),
) {
	orgID, resourceID, ok := parseOrgAndResourceID(c, idParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := updateFn(c.Request.Context(), resourceID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	id, name := getAuditInfo(resp)
	auditUpdate(c, audit.auditService, audit.resourceType, id, name)

	c.JSON(http.StatusOK, resp)
}

// handleOrgDelete handles deleting an org-scoped resource with pre-fetch for audit.
func handleOrgDelete[R any](
	c *gin.Context,
	idParam string,
	audit auditConfig,
	getFn func(ctx context.Context, id, orgID uint) (*R, error),
	deleteFn func(ctx context.Context, id, orgID uint) error,
	getAuditInfo func(*R) (id uint, name string),
) {
	orgID, resourceID, ok := parseOrgAndResourceID(c, idParam)
	if !ok {
		return
	}

	// Pre-fetch for audit log
	item, err := getFn(c.Request.Context(), resourceID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := deleteFn(c.Request.Context(), resourceID, orgID); err != nil {
		respondError(c, err)
		return
	}

	_, name := getAuditInfo(item)
	auditDelete(c, audit.auditService, audit.resourceType, resourceID, name)

	c.Status(http.StatusNoContent)
}
