package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// childContractAuditInfo reports (contractID, childID) for audit rows.
func childContractAuditInfo(r *models.ChildContractResponse) (uint, uint) {
	return r.ID, r.ChildID
}

// CorrectContract godoc
// @Summary Correct a child contract
// @Description Fix what a contract period records, in place. Use this when the stored facts were
// @Description wrong — a typo'd start date, the wrong section, a care type entered incorrectly.
// @Description Use amend instead when the facts themselves changed as of a date.
// @Description
// @Description This is a true partial update: a field you omit is left exactly as it was. To clear
// @Description `to` or `properties`, send them as null. That is the difference from the old PUT,
// @Description where omitting `to` cleared it.
// @Description
// @Description Corrections are allowed on past contracts, including ones that already ended,
// @Description because correcting history is the point. Every changed field is written to the audit
// @Description log with its old and new value, since a change to care type or a supplement changes
// @Description what the Kita is paid.
// @Description
// @Description Auto-applied funding properties are re-merged only when the request touches
// @Description `properties` or `from`, so correcting a section cannot quietly add funding keys.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.ChildContractCorrectRequest true "Fields to correct"
// @Param If-Match header string true "The contract's current version, quoted, e.g. \"3\" — read it from the contract's `version` field or its ETag. Required: it is what makes a concurrent edit fail loudly instead of silently winning."
// @Success 200 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g. from after to, null from, dates before birthdate)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "Dates overlap another contract (contract_overlap), or the contract was changed by someone else"
// @Failure 500 {object} models.ErrorResponse
// @Failure 412 {object} models.ErrorResponse "The contract was changed by someone else since you read it (precondition_failed) — reload and reapply"
// @Failure 428 {object} models.ErrorResponse "If-Match header missing (precondition_required)"
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/{contractId} [patch]
func (h *ChildHandler) CorrectContract(c *gin.Context) {
	handleUpdateContract(c, "childId", h.contractAudit(), h.service.CorrectContract,
		childContractAuditInfo, h.service.GetContractByID, childContractChanges)
}

// AmendContract godoc
// @Summary Amend a child contract from a date
// @Description Record that the facts changed as of a date: the addressed contract is closed the day
// @Description before `effective_from`, and a successor carrying the changes starts on it. Both
// @Description contracts are returned, so the caller never has to guess which id it now holds.
// @Description
// @Description `effective_from` is honoured, including in the past — a Bescheid that arrives late is
// @Description one call. It also anchors the auto-applied funding properties, which the old PUT
// @Description resolved at today and therefore got wrong for any backdated change that crossed a
// @Description funding period.
// @Description
// @Description Fields you omit inherit from the contract being amended, which is usually what you
// @Description want: a new Bescheid typically changes the care type and nothing else. Send `to` as
// @Description null to make the successor open-ended.
// @Description
// @Description Rejected with 400 if `effective_from` is not after the contract's start (that is a
// @Description correction, so use PATCH), or if the contract already ended before it (amending would
// @Description silently extend a finished contract over months that were already billed).
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.ChildContractAmendRequest true "Effective date and the fields that changed"
// @Param If-Match header string true "The contract's current version, quoted, e.g. \"3\" — read it from the contract's `version` field or its ETag. Required: it is what makes a concurrent edit fail loudly instead of silently winning."
// @Success 200 {object} models.ChildContractAmendResponse
// @Failure 400 {object} models.ErrorResponse "effective_from not after the start, or the contract already ended before it"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "The successor would overlap another contract (contract_overlap)"
// @Failure 500 {object} models.ErrorResponse
// @Failure 412 {object} models.ErrorResponse "The contract was changed by someone else since you read it (precondition_failed) — reload and reapply"
// @Failure 428 {object} models.ErrorResponse "If-Match header missing (precondition_required)"
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/{contractId}/amend [post]
func (h *ChildHandler) AmendContract(c *gin.Context) {
	handleAmendContract(c, "childId", h.contractAudit(), h.service.AmendContract,
		h.service.GetContractByID, childContractChanges,
		func(r *models.ChildContractAmendResponse) (*models.ChildContractResponse, *models.ChildContractResponse) {
			return &r.Closed, &r.Created
		},
		childContractAuditInfo)
}

// EndContract godoc
// @Summary Set or clear a child contract's end date
// @Description Record that a contract stops on a date — a child leaving — or undo that by sending
// @Description `to` as null, which makes it ongoing again.
// @Description
// @Description `to` is required in the body. The old surface could only reopen a contract by
// @Description *omitting* the field, which was indistinguishable from "leave it alone"; here the
// @Description null is explicit.
// @Description
// @Description Reopening a contract that has a successor is a 409, not a silent overwrite.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.ContractEndRequest true "The end date, or null to reopen"
// @Param If-Match header string true "The contract's current version, quoted, e.g. \"3\" — read it from the contract's `version` field or its ETag. Required: it is what makes a concurrent edit fail loudly instead of silently winning."
// @Success 200 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse "`to` missing, before `from`, or before the child's birthdate"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "Reopening would overlap a later contract (contract_overlap)"
// @Failure 500 {object} models.ErrorResponse
// @Failure 412 {object} models.ErrorResponse "The contract was changed by someone else since you read it (precondition_failed) — reload and reapply"
// @Failure 428 {object} models.ErrorResponse "If-Match header missing (precondition_required)"
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/{contractId}/end [post]
func (h *ChildHandler) EndContract(c *gin.Context) {
	handleUpdateContract(c, "childId", h.contractAudit(), h.service.EndContract,
		childContractAuditInfo, h.service.GetContractByID, childContractChanges)
}

// MoveContractBoundary godoc
// @Summary Move the boundary between two adjacent child contracts
// @Description Move the seam between two contracts that meet: the later one starts on `at` and the
// @Description earlier one is closed the day before. This backs the timeline drag in the UI.
// @Description
// @Description One date, both sides derived on the server. The client used to compute four dates and
// @Description send them as a batch, which went wrong twice: it cleared the neighbour's `to`
// @Description (dragging any but the newest boundary failed with a 409) and it wiped the
// @Description neighbour's properties, silently recomputing its funding at the base rate.
// @Description
// @Description The two contracts must actually be adjacent — the earlier one's `to` plus one day
// @Description equals the later one's `from`. With a gap between them there are two independent
// @Description boundaries rather than one seam, so set each end date instead. The seam must also
// @Description leave both sides at least one day long.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param request body models.ContractBoundaryMoveRequest true "The two contracts and the new seam date"
// @Success 200 {object} models.ChildContractBoundaryResponse
// @Failure 400 {object} models.ErrorResponse "Same id twice, not adjacent, wrong order, or the seam would empty one side"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child or contract not found"
// @Failure 409 {object} models.ErrorResponse "The resulting timeline would overlap another contract (contract_overlap)"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/boundary [post]
func (h *ChildHandler) MoveContractBoundary(c *gin.Context) {
	handleMoveBoundary(c, "childId", h.contractAudit(), h.service.MoveContractBoundary,
		h.service.GetContractByID, childContractChanges,
		func(r *models.ChildContractBoundaryResponse) (*models.ChildContractResponse, *models.ChildContractResponse) {
			return &r.Earlier, &r.Later
		},
		childContractAuditInfo,
		func(req *models.ContractBoundaryMoveRequest) (uint, uint) {
			return req.EarlierID, req.LaterID
		})
}
