package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// --- Org-scoped nested helpers (routes: /organizations/:orgId/resource/:id/nested) ---

// handleOrgNestedList handles paginated listing of nested resources.
func handleOrgNestedList[Resp any](
	c *gin.Context,
	parentParam string,
	listFn func(context.Context, uint, uint, int, int) ([]Resp, int64, error),
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	items, total, err := listFn(c.Request.Context(), parentID, orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(items, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// handleOrgNestedCreate handles creating a nested resource with audit logging.
func handleOrgNestedCreate[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	createFn func(context.Context, uint, uint, *Req) (*Resp, error),
	getID func(*Resp) uint,
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := createFn(c.Request.Context(), parentID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, audit.auditService, audit.resourceType, getID(resp), fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.JSON(http.StatusCreated, resp)
}

// handleOrgNestedGet handles fetching a single nested resource.
func handleOrgNestedGet[Resp any](
	c *gin.Context,
	parentParam string,
	nestedParam string,
	getFn func(context.Context, uint, uint, uint) (*Resp, error),
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	resp, err := getFn(c.Request.Context(), nestedID, parentID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleOrgNestedUpdate handles updating a nested resource with audit logging.
func handleOrgNestedUpdate[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	nestedParam string,
	audit auditConfig,
	updateFn func(context.Context, uint, uint, uint, *Req) (*Resp, error),
	getID func(*Resp) uint,
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := updateFn(c.Request.Context(), nestedID, parentID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, audit.auditService, audit.resourceType, getID(resp), fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.JSON(http.StatusOK, resp)
}

// handleOrgNestedDelete handles deleting a nested resource with audit logging.
func handleOrgNestedDelete(
	c *gin.Context,
	parentParam string,
	nestedParam string,
	audit auditConfig,
	deleteFn func(context.Context, uint, uint, uint) error,
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := deleteFn(c.Request.Context(), nestedID, parentID, orgID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, audit.auditService, audit.resourceType, nestedID, fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.Status(http.StatusNoContent)
}

// --- Org-scoped deep nested helpers (routes: /organizations/:orgId/resource/:id/mid/:midParam/nested) ---

// handleOrgDeepNestedCreate handles creating a deep nested resource with audit logging.
func handleOrgDeepNestedCreate[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	midParam string,
	audit auditConfig,
	createFn func(context.Context, uint, uint, uint, *Req) (*Resp, error),
	getID func(*Resp) uint,
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	midID, err := parseID(c, midParam)
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := createFn(c.Request.Context(), midID, parentID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, audit.auditService, audit.resourceType, getID(resp), fmt.Sprintf("%s=%d", audit.parentLabel, midID))

	c.JSON(http.StatusCreated, resp)
}

// handleOrgDeepNestedGet handles fetching a single deep nested resource.
func handleOrgDeepNestedGet[Resp any](
	c *gin.Context,
	parentParam string,
	midParam string,
	nestedParam string,
	getFn func(context.Context, uint, uint, uint, uint) (*Resp, error),
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	midID, err := parseID(c, midParam)
	if err != nil {
		respondError(c, err)
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	resp, err := getFn(c.Request.Context(), nestedID, midID, parentID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleOrgDeepNestedUpdate handles updating a deep nested resource with audit logging.
func handleOrgDeepNestedUpdate[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	midParam string,
	nestedParam string,
	audit auditConfig,
	updateFn func(context.Context, uint, uint, uint, uint, *Req) (*Resp, error),
	getID func(*Resp) uint,
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	midID, err := parseID(c, midParam)
	if err != nil {
		respondError(c, err)
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := updateFn(c.Request.Context(), nestedID, midID, parentID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, audit.auditService, audit.resourceType, getID(resp), fmt.Sprintf("%s=%d", audit.parentLabel, midID))

	c.JSON(http.StatusOK, resp)
}

// handleOrgDeepNestedDelete handles deleting a deep nested resource with audit logging.
func handleOrgDeepNestedDelete(
	c *gin.Context,
	parentParam string,
	midParam string,
	nestedParam string,
	audit auditConfig,
	deleteFn func(context.Context, uint, uint, uint, uint) error,
) {
	orgID, parentID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	midID, err := parseID(c, midParam)
	if err != nil {
		respondError(c, err)
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := deleteFn(c.Request.Context(), nestedID, midID, parentID, orgID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, audit.auditService, audit.resourceType, nestedID, fmt.Sprintf("%s=%d", audit.parentLabel, midID))

	c.Status(http.StatusNoContent)
}

// --- Global nested helpers (routes: /resource/:id/nested) ---

// handleGlobalNestedCreate handles creating a nested resource under a global parent.
func handleGlobalNestedCreate[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	createFn func(context.Context, uint, *Req) (*Resp, error),
	getID func(*Resp) uint,
) {
	parentID, err := parseID(c, parentParam)
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := createFn(c.Request.Context(), parentID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, audit.auditService, audit.resourceType, getID(resp), fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.JSON(http.StatusCreated, resp)
}

// handleGlobalNestedUpdate handles updating a nested resource under a global parent.
func handleGlobalNestedUpdate[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	nestedParam string,
	audit auditConfig,
	updateFn func(context.Context, uint, uint, *Req) (*Resp, error),
	getID func(*Resp) uint,
) {
	parentID, err := parseID(c, parentParam)
	if err != nil {
		respondError(c, err)
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := updateFn(c.Request.Context(), parentID, nestedID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, audit.auditService, audit.resourceType, getID(resp), fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.JSON(http.StatusOK, resp)
}

// handleGlobalNestedDelete handles deleting a nested resource under a global parent.
func handleGlobalNestedDelete(
	c *gin.Context,
	parentParam string,
	nestedParam string,
	audit auditConfig,
	deleteFn func(context.Context, uint, uint) error,
) {
	parentID, err := parseID(c, parentParam)
	if err != nil {
		respondError(c, err)
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := deleteFn(c.Request.Context(), parentID, nestedID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, audit.auditService, audit.resourceType, nestedID, fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.Status(http.StatusNoContent)
}

// --- Global deep nested helpers (routes: /resource/:id/mid/:midParam/nested) ---

// handleGlobalDeepNestedCreate handles creating a deep nested resource under a global parent.
func handleGlobalDeepNestedCreate[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	midParam string,
	audit auditConfig,
	createFn func(context.Context, uint, uint, *Req) (*Resp, error),
	getID func(*Resp) uint,
) {
	parentID, err := parseID(c, parentParam)
	if err != nil {
		respondError(c, err)
		return
	}

	midID, err := parseID(c, midParam)
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := createFn(c.Request.Context(), parentID, midID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, audit.auditService, audit.resourceType, getID(resp), fmt.Sprintf("%s=%d", audit.parentLabel, midID))

	c.JSON(http.StatusCreated, resp)
}

// handleGlobalDeepNestedUpdate handles updating a deep nested resource under a global parent.
func handleGlobalDeepNestedUpdate[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	midParam string,
	nestedParam string,
	audit auditConfig,
	updateFn func(context.Context, uint, uint, uint, *Req) (*Resp, error),
	getID func(*Resp) uint,
) {
	parentID, err := parseID(c, parentParam)
	if err != nil {
		respondError(c, err)
		return
	}

	midID, err := parseID(c, midParam)
	if err != nil {
		respondError(c, err)
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := updateFn(c.Request.Context(), parentID, midID, nestedID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, audit.auditService, audit.resourceType, getID(resp), fmt.Sprintf("%s=%d", audit.parentLabel, midID))

	c.JSON(http.StatusOK, resp)
}

// handleGlobalDeepNestedDelete handles deleting a deep nested resource under a global parent.
func handleGlobalDeepNestedDelete(
	c *gin.Context,
	parentParam string,
	midParam string,
	nestedParam string,
	audit auditConfig,
	deleteFn func(context.Context, uint, uint, uint) error,
) {
	parentID, err := parseID(c, parentParam)
	if err != nil {
		respondError(c, err)
		return
	}

	midID, err := parseID(c, midParam)
	if err != nil {
		respondError(c, err)
		return
	}

	nestedID, err := parseID(c, nestedParam)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := deleteFn(c.Request.Context(), parentID, midID, nestedID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, audit.auditService, audit.resourceType, nestedID, fmt.Sprintf("%s=%d", audit.parentLabel, midID))

	c.Status(http.StatusNoContent)
}
