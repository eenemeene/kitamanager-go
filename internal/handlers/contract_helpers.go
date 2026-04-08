package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// handleListContracts handles paginated listing of contracts for a parent resource.
func handleListContracts[Resp any](
	c *gin.Context,
	parentParam string,
	listFn func(context.Context, uint, uint, int, int) ([]Resp, int64, error),
) {
	orgID, id, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	contracts, total, err := listFn(c.Request.Context(), id, orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(contracts, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// handleGetCurrentRecord handles fetching the currently active contract.
func handleGetCurrentRecord[Resp any](
	c *gin.Context,
	parentParam string,
	getFn func(context.Context, uint, uint) (*Resp, error),
) {
	orgID, id, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	contract, err := getFn(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract)
}

// handleGetContract handles fetching a single contract by ID.
func handleGetContract[Resp any](
	c *gin.Context,
	parentParam string,
	getFn func(context.Context, uint, uint, uint) (*Resp, error),
) {
	orgID, resourceID, contractID, ok := parseOrgResourceAndContractID(c, parentParam)
	if !ok {
		return
	}

	contract, err := getFn(c.Request.Context(), contractID, resourceID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract)
}

// handleCreateContract handles creating a new contract with audit logging.
func handleCreateContract[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	createFn func(context.Context, uint, uint, *Req) (*Resp, error),
	getAuditInfo func(*Resp) (uint, uint), // returns (contractID, parentID)
) {
	orgID, resourceID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := createFn(c.Request.Context(), resourceID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	id, parentID := getAuditInfo(resp)
	auditCreate(c, audit.auditService, audit.resourceType, id, fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.JSON(http.StatusCreated, resp)
}

// handleUpdateContract handles updating an existing contract with audit logging.
func handleUpdateContract[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	updateFn func(context.Context, uint, uint, uint, *Req) (*Resp, error),
	getAuditInfo func(*Resp) (uint, uint), // returns (contractID, parentID)
) {
	orgID, resourceID, contractID, ok := parseOrgResourceAndContractID(c, parentParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	resp, err := updateFn(c.Request.Context(), contractID, resourceID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	id, parentID := getAuditInfo(resp)
	auditUpdate(c, audit.auditService, audit.resourceType, id, fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.JSON(http.StatusOK, resp)
}

// handleBatchUpdateContracts handles atomically updating multiple contracts with audit logging.
func handleBatchUpdateContracts[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	batchUpdateFn func(context.Context, uint, uint, *Req) ([]Resp, error),
	getAuditInfo func(*Resp) (uint, uint), // returns (contractID, parentID)
) {
	orgID, resourceID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	results, err := batchUpdateFn(c.Request.Context(), resourceID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	for i := range results {
		id, parentID := getAuditInfo(&results[i])
		auditUpdate(c, audit.auditService, audit.resourceType, id, fmt.Sprintf("%s=%d", audit.parentLabel, parentID))
	}

	c.JSON(http.StatusOK, results)
}

// handleDeleteContract handles deleting a contract with pre-fetch for audit logging.
func handleDeleteContract[Resp any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	getFn func(context.Context, uint, uint, uint) (*Resp, error),
	deleteFn func(context.Context, uint, uint, uint) error,
	getAuditInfo func(*Resp) (uint, uint), // returns (contractID, parentID)
) {
	orgID, resourceID, contractID, ok := parseOrgResourceAndContractID(c, parentParam)
	if !ok {
		return
	}

	// Pre-fetch for audit log
	item, err := getFn(c.Request.Context(), contractID, resourceID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := deleteFn(c.Request.Context(), contractID, resourceID, orgID); err != nil {
		respondError(c, err)
		return
	}

	_, parentID := getAuditInfo(item)
	auditDelete(c, audit.auditService, audit.resourceType, contractID, fmt.Sprintf("%s=%d", audit.parentLabel, parentID))

	c.Status(http.StatusNoContent)
}
