package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// employeeContractAuditInfo reports (contractID, employeeID) for audit rows.
func employeeContractAuditInfo(r *models.EmployeeContractResponse) (uint, uint) {
	return r.ID, r.EmployeeID
}

// CorrectContract godoc
// @Summary Correct an employee contract
// @Description Fix what a contract period records, in place — a typo'd start date, the wrong
// @Description section, a pay grade entered incorrectly. Use amend instead when the terms actually
// @Description changed as of a date, so the old terms stay on record for the months they applied to.
// @Description
// @Description A true partial update: a field you omit is left exactly as it was, and `to` or
// @Description `properties` are cleared only by an explicit null. `weekly_hours` accepts 0 — a
// @Description contract with no hours (parental leave) is legitimate and the old request could not
// @Description express it, because `required` rejects zero.
// @Description
// @Description Pay-plan coverage is re-checked only when pay plan, grade, step or `from` move, so
// @Description correcting a section cannot fail because the pay plan was edited years later.
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.EmployeeContractCorrectRequest true "Fields to correct"
// @Param If-Match header string true "The contract's current version, quoted, e.g. \"3\" — read it from the contract's `version` field or its ETag. Required: it is what makes a concurrent edit fail loudly instead of silently winning."
// @Success 200 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g. from after to, unknown pay plan, grade/step not covered)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "Dates overlap another contract (contract_overlap), or the contract was changed by someone else"
// @Failure 500 {object} models.ErrorResponse
// @Failure 412 {object} models.ErrorResponse "The contract was changed by someone else since you read it (precondition_failed) — reload and reapply"
// @Failure 428 {object} models.ErrorResponse "If-Match header missing (precondition_required)"
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts/{contractId} [patch]
func (h *EmployeeHandler) CorrectContract(c *gin.Context) {
	handleUpdateContract(c, "employeeId", h.contractAudit(), h.service.CorrectContract,
		employeeContractAuditInfo, h.service.GetContractByID, employeeContractChanges)
}

// AmendContract godoc
// @Summary Amend an employee contract from a date
// @Description Record that the terms changed as of a date — a raise, a change in weekly hours, a
// @Description move to another section. The addressed contract is closed the day before
// @Description `effective_from` and a successor carrying the changes starts on it; both are returned.
// @Description
// @Description This is the operation to use for anything that affects pay, because it keeps the old
// @Description terms on record for the months they applied to. `effective_from` is honoured,
// @Description including in the past, and it anchors the pay-plan coverage check — the old path
// @Description checked at today, which accepted a backdated amendment to a grade the plan only
// @Description gained later, and rejected one it had at the time.
// @Description
// @Description Fields you omit inherit from the contract being amended. Send `to` as null to make
// @Description the successor open-ended.
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.EmployeeContractAmendRequest true "Effective date and the terms that changed"
// @Param If-Match header string true "The contract's current version, quoted, e.g. \"3\" — read it from the contract's `version` field or its ETag. Required: it is what makes a concurrent edit fail loudly instead of silently winning."
// @Success 200 {object} models.EmployeeContractAmendResponse
// @Failure 400 {object} models.ErrorResponse "effective_from not after the start, contract already ended before it, or grade/step not covered at that date"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "The successor would overlap another contract (contract_overlap)"
// @Failure 500 {object} models.ErrorResponse
// @Failure 412 {object} models.ErrorResponse "The contract was changed by someone else since you read it (precondition_failed) — reload and reapply"
// @Failure 428 {object} models.ErrorResponse "If-Match header missing (precondition_required)"
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts/{contractId}/amend [post]
func (h *EmployeeHandler) AmendContract(c *gin.Context) {
	handleAmendContract(c, "employeeId", h.contractAudit(), h.service.AmendContract,
		h.service.GetContractByID, employeeContractChanges,
		func(r *models.EmployeeContractAmendResponse) (*models.EmployeeContractResponse, *models.EmployeeContractResponse) {
			return &r.Closed, &r.Created
		},
		employeeContractAuditInfo)
}

// EndContract godoc
// @Summary Set or clear an employee contract's end date
// @Description Record that a contract stops on a date — an employee leaving, or a fixed term — or
// @Description undo that by sending `to` as null, which makes it ongoing again.
// @Description
// @Description `to` is required in the body: the old surface could only reopen a contract by
// @Description omitting the field, which was indistinguishable from "leave it alone".
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.ContractEndRequest true "The end date, or null to reopen"
// @Param If-Match header string true "The contract's current version, quoted, e.g. \"3\" — read it from the contract's `version` field or its ETag. Required: it is what makes a concurrent edit fail loudly instead of silently winning."
// @Success 200 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse "`to` missing, or before `from`"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "Reopening would overlap a later contract (contract_overlap)"
// @Failure 500 {object} models.ErrorResponse
// @Failure 412 {object} models.ErrorResponse "The contract was changed by someone else since you read it (precondition_failed) — reload and reapply"
// @Failure 428 {object} models.ErrorResponse "If-Match header missing (precondition_required)"
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts/{contractId}/end [post]
func (h *EmployeeHandler) EndContract(c *gin.Context) {
	handleUpdateContract(c, "employeeId", h.contractAudit(), h.service.EndContract,
		employeeContractAuditInfo, h.service.GetContractByID, employeeContractChanges)
}

// MoveContractBoundary godoc
// @Summary Move the boundary between two adjacent employee contracts
// @Description Move the seam between two contracts that meet: the later one starts on `at` and the
// @Description earlier one is closed the day before. One date, both sides derived on the server.
// @Description
// @Description The two contracts must actually be adjacent — the earlier one's `to` plus one day
// @Description equals the later one's `from`. With a gap there are two independent boundaries rather
// @Description than one seam, so set each end date instead. The seam must leave both sides at least
// @Description one day long, and the pay plan must cover the later contract's grade and step at its
// @Description new start date.
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Param request body models.ContractBoundaryMoveRequest true "The two contracts and the new seam date"
// @Success 200 {object} models.EmployeeContractBoundaryResponse
// @Failure 400 {object} models.ErrorResponse "Same id twice, not adjacent, wrong order, seam would empty one side, or grade/step not covered at the new start"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Employee or contract not found"
// @Failure 409 {object} models.ErrorResponse "The resulting timeline would overlap another contract (contract_overlap)"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts/boundary [post]
func (h *EmployeeHandler) MoveContractBoundary(c *gin.Context) {
	handleMoveBoundary(c, "employeeId", h.contractAudit(), h.service.MoveContractBoundary,
		h.service.GetContractByID, employeeContractChanges,
		func(r *models.EmployeeContractBoundaryResponse) (*models.EmployeeContractResponse, *models.EmployeeContractResponse) {
			return &r.Earlier, &r.Later
		},
		employeeContractAuditInfo,
		func(req *models.ContractBoundaryMoveRequest) (uint, uint) {
			return req.EarlierID, req.LaterID
		})
}
