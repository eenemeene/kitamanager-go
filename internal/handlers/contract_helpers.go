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
//
// The contract is fetched before the update so the audit row can carry a
// per-field diff — otherwise the log only says "contract N was updated" and a
// silent change to care_type or a funding supplement leaves no trace. Same
// pre-fetch pattern as handleDeleteContract below.
func handleUpdateContract[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	updateFn func(context.Context, uint, uint, uint, *Req) (*Resp, error),
	getAuditInfo func(*Resp) (uint, uint), // returns (contractID, parentID)
	getFn func(context.Context, uint, uint, uint) (*Resp, error),
	diffFn func(before, after *Resp) map[string]any,
) {
	orgID, resourceID, contractID, ok := parseOrgResourceAndContractID(c, parentParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	// Best-effort: a failure here must not block the update. If the contract is
	// really missing, updateFn reports it properly a moment later.
	before, _ := getFn(c.Request.Context(), contractID, resourceID, orgID)

	resp, err := updateFn(c.Request.Context(), contractID, resourceID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	id, parentID := getAuditInfo(resp)
	changes := contractAuditChanges(before, resp, id, getAuditInfo, diffFn)
	auditUpdateWithChanges(c, audit.auditService, audit.resourceType, id,
		fmt.Sprintf("%s=%d", audit.parentLabel, parentID), changes)

	c.JSON(http.StatusOK, resp)
}

// contractAuditChanges builds the audit `changes` map for one contract update.
//
// Amend mode makes this less obvious than it looks: updating a contract that
// started before today closes that row and returns a *different* contract, so a
// raw before/after diff would silently compare two identities. When that
// happens the diff is still the useful part — it is what the operator changed —
// but the row also records which contract was closed and which was created, so
// the reader is not misled into thinking one row was edited in place.
func contractAuditChanges[Resp any](
	before, after *Resp,
	afterID uint,
	getAuditInfo func(*Resp) (uint, uint),
	diffFn func(before, after *Resp) map[string]any,
) map[string]any {
	if before == nil || diffFn == nil {
		return nil
	}
	changes := diffFn(before, after)
	if beforeID, _ := getAuditInfo(before); beforeID != afterID {
		changes["amended"] = map[string]any{
			"closed_contract_id": beforeID,
			"new_contract_id":    afterID,
		}
	}
	return changes
}

// handleBatchUpdateContracts handles atomically updating multiple contracts with audit logging.
//
// Each contract is fetched before the update so its audit row can carry a
// per-field diff. entryIDsFn is needed because this helper is generic over the
// request type and cannot otherwise know which contracts the batch touches; it
// costs one extra read per entry, and a timeline boundary drag sends two.
func handleBatchUpdateContracts[Req any, Resp any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	batchUpdateFn func(context.Context, uint, uint, *Req) ([]Resp, error),
	getAuditInfo func(*Resp) (uint, uint), // returns (contractID, parentID)
	getFn func(context.Context, uint, uint, uint) (*Resp, error),
	diffFn func(before, after *Resp) map[string]any,
	entryIDsFn func(*Req) []uint,
) {
	orgID, resourceID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	// Best-effort pre-fetch, keyed by contract id. Errors are ignored for the
	// same reason as in handleUpdateContract: the update itself reports them.
	before := make(map[uint]*Resp)
	if getFn != nil && entryIDsFn != nil {
		for _, id := range entryIDsFn(req) {
			if prev, err := getFn(c.Request.Context(), id, resourceID, orgID); err == nil {
				before[id] = prev
			}
		}
	}

	results, err := batchUpdateFn(c.Request.Context(), resourceID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	for i := range results {
		id, parentID := getAuditInfo(&results[i])
		// Batch updates are always in place, so the id is stable and the
		// before-state is found by it.
		changes := contractAuditChanges(before[id], &results[i], id, getAuditInfo, diffFn)
		auditUpdateWithChanges(c, audit.auditService, audit.resourceType, id,
			fmt.Sprintf("%s=%d", audit.parentLabel, parentID), changes)
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
