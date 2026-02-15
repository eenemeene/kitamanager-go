package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// contractAuditConfig holds audit configuration for contract operations.
type contractAuditConfig struct {
	auditService *service.AuditService
	resourceType string // e.g. "child_contract", "employee_contract"
	parentName   string // e.g. "child", "employee"
}

// handleListContracts handles paginated listing of contracts for a parent resource.
func handleListContracts[Resp any](
	c *gin.Context,
	listFn func(context.Context, uint, uint, int, int) ([]Resp, int64, error),
) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
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

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(contracts, params.Page, params.Limit, total, c.Request.URL.Path))
}

// handleGetCurrentRecord handles fetching the currently active contract.
func handleGetCurrentRecord[Resp any](
	c *gin.Context,
	getFn func(context.Context, uint, uint) (*Resp, error),
) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
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
	getFn func(context.Context, uint, uint, uint) (*Resp, error),
) {
	orgID, resourceID, contractID, ok := parseOrgResourceAndContractID(c)
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
	audit contractAuditConfig,
	createFn func(context.Context, uint, uint, *Req) (*Resp, error),
	getAuditInfo func(*Resp) (uint, uint), // returns (contractID, parentID)
) {
	orgID, resourceID, ok := parseOrgAndResourceID(c, "id")
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
	auditCreate(c, audit.auditService, audit.resourceType, id, fmt.Sprintf("%s=%d", audit.parentName, parentID))

	c.JSON(http.StatusCreated, resp)
}

// handleUpdateContract handles updating an existing contract with audit logging.
func handleUpdateContract[Req any, Resp any](
	c *gin.Context,
	audit contractAuditConfig,
	updateFn func(context.Context, uint, uint, uint, *Req) (*Resp, error),
	getAuditInfo func(*Resp) (uint, uint), // returns (contractID, parentID)
) {
	orgID, resourceID, contractID, ok := parseOrgResourceAndContractID(c)
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
	auditUpdate(c, audit.auditService, audit.resourceType, id, fmt.Sprintf("%s=%d", audit.parentName, parentID))

	c.JSON(http.StatusOK, resp)
}

// handleDeleteContract handles deleting a contract with audit logging.
func handleDeleteContract(
	c *gin.Context,
	audit contractAuditConfig,
	deleteFn func(context.Context, uint, uint, uint) error,
) {
	orgID, resourceID, contractID, ok := parseOrgResourceAndContractID(c)
	if !ok {
		return
	}

	if err := deleteFn(c.Request.Context(), contractID, resourceID, orgID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, audit.auditService, audit.resourceType, contractID, fmt.Sprintf("%s=%d", audit.parentName, resourceID))

	c.Status(http.StatusNoContent)
}
