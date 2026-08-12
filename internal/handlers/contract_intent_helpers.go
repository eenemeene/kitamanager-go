package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Audit plumbing for the intent-based contract endpoints.
//
// Correct and end reuse handleUpdateContract: they edit one row, so its
// pre-fetch-and-diff is exactly right (and its amend-inference branch never
// fires, because the contract id cannot change). Amend and boundary touch two
// rows each and get the helpers below.
//
// Note what the explicit intents buy the audit log: handleUpdateContract has to
// *infer* that an amend happened by comparing the response's contract id to the
// one in the URL, then read the closed contract back to diff it. Here the
// service says which row it closed and which it created, so the pair is recorded
// from what actually happened.

// handleAmendContract records an amendment as what it is: an update of the
// contract that was closed, plus a create of its successor. The two rows are
// cross-linked so a reader landing on either can find the other.
func handleAmendContract[Req any, Resp any, Item any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	amendFn func(context.Context, uint, uint, uint, *Req) (*Resp, error),
	getFn func(context.Context, uint, uint, uint) (*Item, error),
	diffFn func(before, after *Item) map[string]any,
	split func(*Resp) (closed, created *Item),
	getAuditInfo func(*Item) (uint, uint),
) {
	orgID, resourceID, contractID, ok := parseOrgResourceAndContractID(c, parentParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	// Best-effort: a failure here must not block the amendment. If the contract is
	// really missing, amendFn reports it properly a moment later.
	before, _ := getFn(c.Request.Context(), contractID, resourceID, orgID)

	resp, err := amendFn(c.Request.Context(), contractID, resourceID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	closed, created := split(resp)
	closedID, parentID := getAuditInfo(closed)
	createdID, _ := getAuditInfo(created)
	parentLabel := fmt.Sprintf("%s=%d", audit.parentLabel, parentID)

	changes := map[string]any{}
	if before != nil && diffFn != nil {
		changes = diffFn(before, closed)
	}
	changes["amended"] = map[string]any{
		"closed_contract_id": closedID,
		"new_contract_id":    createdID,
	}
	auditUpdateWithChanges(c, audit.auditService, audit.resourceType, closedID, parentLabel, changes)
	auditCreate(c, audit.auditService, audit.resourceType, createdID, parentLabel)

	c.JSON(http.StatusOK, resp)
}

// handleMoveBoundary records a seam move as one update per side, each carrying
// its own field diff. Both sides are read first: a boundary move is the one
// operation that legitimately changes dates on a contract the user did not
// address, so the log has to show what happened to the neighbour too.
func handleMoveBoundary[Req any, Resp any, Item any](
	c *gin.Context,
	parentParam string,
	audit auditConfig,
	moveFn func(context.Context, uint, uint, *Req) (*Resp, error),
	getFn func(context.Context, uint, uint, uint) (*Item, error),
	diffFn func(before, after *Item) map[string]any,
	sides func(*Resp) (earlier, later *Item),
	getAuditInfo func(*Item) (uint, uint),
	requestIDs func(*Req) (earlierID, laterID uint),
) {
	orgID, resourceID, ok := parseOrgAndResourceID(c, parentParam)
	if !ok {
		return
	}

	req, ok := bindJSON[Req](c)
	if !ok {
		return
	}

	earlierID, laterID := requestIDs(req)
	beforeEarlier, _ := getFn(c.Request.Context(), earlierID, resourceID, orgID)
	beforeLater, _ := getFn(c.Request.Context(), laterID, resourceID, orgID)

	resp, err := moveFn(c.Request.Context(), resourceID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	afterEarlier, afterLater := sides(resp)
	for _, side := range []struct {
		before, after *Item
		role          string
	}{
		{beforeEarlier, afterEarlier, "earlier"},
		{beforeLater, afterLater, "later"},
	} {
		id, parentID := getAuditInfo(side.after)
		changes := map[string]any{}
		if side.before != nil && diffFn != nil {
			changes = diffFn(side.before, side.after)
		}
		// Without this a reader sees two unrelated date edits seconds apart and
		// cannot tell that one drag caused both.
		changes["boundary_moved"] = map[string]any{
			"earlier_contract_id": earlierID,
			"later_contract_id":   laterID,
			"side":                side.role,
		}
		auditUpdateWithChanges(c, audit.auditService, audit.resourceType, id,
			fmt.Sprintf("%s=%d", audit.parentLabel, parentID), changes)
	}

	c.JSON(http.StatusOK, resp)
}
