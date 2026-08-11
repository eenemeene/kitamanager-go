package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type ChildHandler struct {
	service      *service.ChildService
	auditService *service.AuditService
}

func NewChildHandler(service *service.ChildService, auditService *service.AuditService) *ChildHandler {
	return &ChildHandler{
		service:      service,
		auditService: auditService,
	}
}

func (h *ChildHandler) contractAudit() auditConfig {
	return auditConfig{
		auditService: h.auditService,
		resourceType: "child_contract",
		parentLabel:  "child",
	}
}

// List godoc
// @Summary List all children in an organization
// @Description Get a paginated list of all children in the specified organization
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param section_id query int false "Filter by section ID"
// @Param active_on query string false "Filter by active contract date (YYYY-MM-DD, defaults to today). Mutually exclusive with contract_after."
// @Param contract_after query string false "Filter children with contracts starting after this date (YYYY-MM-DD). Mutually exclusive with active_on."
// @Param search query string false "Search by first or last name (case-insensitive)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.ChildResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children [get]
func (h *ChildHandler) List(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	// Parse optional section_id filter
	sectionID, ok := parseOptionalUint(c, "section_id")
	if !ok {
		return
	}

	// Parse optional date filters
	contractAfter, ok := parseOptionalDatePtr(c, "contract_after")
	if !ok {
		return
	}

	activeOn, ok := parseOptionalDatePtr(c, "active_on")
	if !ok {
		return
	}

	if activeOn == nil && contractAfter == nil {
		// Default active_on to today when neither filter is specified.
		// models.Today() honors the application timezone (Europe/Berlin by
		// default) so a Berlin user editing the list at 23:30 sees the
		// roster they expect, not yesterday's.
		now := models.Today()
		activeOn = &now
	}

	search, ok := parseSearch(c)
	if !ok {
		return
	}

	filter := models.ChildListFilter{
		SectionID:     sectionID,
		ActiveOn:      activeOn,
		ContractAfter: contractAfter,
		Search:        search,
	}

	children, total, err := h.service.ListByOrganizationAndSection(c.Request.Context(), orgID, filter, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(children, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// Get godoc
// @Summary Get child by ID
// @Description Get a single child by their ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Success 200 {object} models.ChildResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId} [get]
func (h *ChildHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "childId")
	if !ok {
		return
	}

	child, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, child)
}

// Create godoc
// @Summary Create a new child
// @Description Create a new child in the specified organization
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.ChildCreateRequest true "Child data"
// @Success 201 {object} models.ChildResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children [post]
func (h *ChildHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[models.ChildCreateRequest](c)
	if !ok {
		return
	}

	child, err := h.service.Create(c.Request.Context(), orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "child", child.ID, child.FullName())

	c.JSON(http.StatusCreated, child)
}

// Update godoc
// @Summary Update a child
// @Description Update an existing child by ID.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param request body models.ChildUpdateRequest true "Child data"
// @Success 200 {object} models.ChildResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId} [put]
func (h *ChildHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "childId")
	if !ok {
		return
	}

	req, ok := bindJSON[models.ChildUpdateRequest](c)
	if !ok {
		return
	}

	// Capture the pre-update state for the audit diff (review finding
	// H2). On a NotFound the Update path below would surface the same
	// error, so propagating it here keeps the response canonical.
	before, beforeErr := h.service.GetByID(c.Request.Context(), id, orgID)
	if beforeErr != nil {
		respondError(c, beforeErr)
		return
	}

	child, err := h.service.Update(c.Request.Context(), id, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	changes := map[string]any{}
	recordChange(changes, "first_name", before.FirstName, child.FirstName)
	recordChange(changes, "last_name", before.LastName, child.LastName)
	recordChange(changes, "gender", before.Gender, child.Gender)
	recordTimeChange(changes, "birthdate", before.Birthdate, child.Birthdate)
	auditUpdateWithChanges(c, h.auditService, "child", child.ID, child.FullName(), changes)

	c.JSON(http.StatusOK, child)
}

// Delete godoc
// @Summary Delete a child
// @Description Delete a child by ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId} [delete]
func (h *ChildHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "childId")
	if !ok {
		return
	}

	// Get child info before deletion for audit log
	child, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, orgID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "child", id, child.FullName())

	c.Status(http.StatusNoContent)
}

// ListContracts godoc
// @Summary List child contracts
// @Description Get paginated contracts for a child
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.ChildContractResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts [get]
func (h *ChildHandler) ListContracts(c *gin.Context) {
	handleListContracts(c, "childId", h.service.ListContracts)
}

// GetCurrentRecord godoc
// @Summary Get current child contract
// @Description Get the currently active contract for a child
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Success 200 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/current [get]
func (h *ChildHandler) GetCurrentRecord(c *gin.Context) {
	handleGetCurrentRecord(c, "childId", h.service.GetCurrentRecord)
}

// GetContract godoc
// @Summary Get child contract by ID
// @Description Get a single contract by ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Success 200 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/{contractId} [get]
func (h *ChildHandler) GetContract(c *gin.Context) {
	handleGetContract(c, "childId", h.service.GetContractByID)
}

// CreateContract godoc
// @Summary Create child contract
// @Description Create a new contract for a child.
// @Description
// @Description **Contract Date Rules:**
// @Description - Both `from` and `to` dates are inclusive (the contract is active on both dates)
// @Description - Same-day contracts are allowed (`from` == `to`)
// @Description - Contracts must not overlap with existing contracts
// @Description - "Touching" contracts (where contract A ends on the same day contract B starts) are considered overlapping
// @Description - To transition between contracts, the new contract must start the day AFTER the previous one ends
// @Description
// @Description **Example:** If contract A ends on 2025-01-31, contract B must start on 2025-02-01 or later.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param request body models.ChildContractCreateRequest true "Contract data"
// @Success 201 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., from date after to date)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child not found"
// @Failure 409 {object} models.ErrorResponse "Contract overlaps with existing contract"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts [post]
func (h *ChildHandler) CreateContract(c *gin.Context) {
	handleCreateContract(c, "childId", h.contractAudit(), h.service.CreateContract,
		func(r *models.ChildContractResponse) (uint, uint) { return r.ID, r.ChildID })
}

// UpdateContract godoc
// @Summary Update child contract
// @Description Update an existing contract by ID. Date rules as for creation: both dates
// @Description inclusive, same-day contracts allowed, no overlaps.
// @Description
// @Description IMPORTANT — what happens depends on when the contract started, because changing a
// @Description past contract in place would silently recompute funding for months already billed:
// @Description
// @Description   - started BEFORE today: the contract is AMENDED, not edited. The existing row is
// @Description     closed with to = yesterday and a NEW contract is created starting TODAY carrying
// @Description     the changes. Any `from` in the request is IGNORED; `to`, if given, applies to the
// @Description     new contract, so a `to` in the past is rejected (400) as from-after-to.
// @Description   - starts TODAY or later: updated in place.
// @Description   - already ENDED (to before today): rejected with 400.
// @Description
// @Description So this endpoint records a change going FORWARD. To correct a historical timeline —
// @Description move a boundary into the past — use the batch endpoint instead.
// @Description
// @Description Nullable fields (`to`, `properties`) are replaced wholesale: omitting one CLEARS it.
// @Description Send the full properties map on every update unless you mean to remove them.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.ChildContractUpdateRequest true "Contract data"
// @Success 200 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., from date after to date)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "Updated dates would overlap with another contract"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/{contractId} [put]
func (h *ChildHandler) UpdateContract(c *gin.Context) {
	handleUpdateContract(c, "childId", h.contractAudit(), h.service.UpdateContract,
		func(r *models.ChildContractResponse) (uint, uint) { return r.ID, r.ChildID },
		h.service.GetContractByID, childContractChanges)
}

// childContractChanges builds the audit diff for a child contract. Properties
// carry the funding-relevant fields (care_type plus supplements), which is why
// they get their own map-aware comparison — see recordPropertiesChange.
func childContractChanges(before, after *models.ChildContractResponse) map[string]any {
	changes := map[string]any{}
	recordTimeChange(changes, "from", before.From, after.From)
	recordNullableTimeChange(changes, "to", before.To, after.To)
	recordChange(changes, "section_id", before.SectionID, after.SectionID)
	recordPropertiesChange(changes, "properties", before.Properties, after.Properties)
	return changes
}

// BatchUpdateContracts godoc
// @Summary Batch update child contracts
// @Description Atomically update multiple contracts for a child. This is the endpoint for
// @Description CORRECTING A HISTORICAL TIMELINE: unlike the single-contract PUT there is NO amend
// @Description mode, so dates and fields are written in place even on contracts that started — or
// @Description ended — in the past. It backs the timeline boundary drag in the UI.
// @Description
// @Description Entries are PARTIAL: a field you omit keeps its current value, with one exception —
// @Description `to` is CLEARED when omitted, because that is how a contract is set back to ongoing.
// @Description Always send `to` explicitly when you want to keep it, otherwise a contract that has a
// @Description successor becomes ongoing and collides with it (409). `properties` is carried forward
// @Description when omitted; send an empty object to clear it deliberately.
// @Description
// @Description Adjacent ranges may be swapped within one request (extend A's `to`, shift B's `from`):
// @Description the no-overlap constraint is deferred to commit, so only the final state must be
// @Description valid. If any entry fails, the whole batch is rolled back.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param request body models.ChildContractBatchUpdateRequest true "Batch update data"
// @Success 200 {array} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., duplicate IDs, invalid dates)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child or contract not found"
// @Failure 409 {object} models.ErrorResponse "Updated dates would overlap"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/batch [put]
func (h *ChildHandler) BatchUpdateContracts(c *gin.Context) {
	handleBatchUpdateContracts(c, "childId", h.contractAudit(), h.service.BatchUpdateContracts,
		func(r *models.ChildContractResponse) (uint, uint) { return r.ID, r.ChildID },
		h.service.GetContractByID, childContractChanges,
		func(req *models.ChildContractBatchUpdateRequest) []uint {
			ids := make([]uint, 0, len(req.Updates))
			for _, u := range req.Updates {
				ids = append(ids, u.ID)
			}
			return ids
		})
}

// DeleteContract godoc
// @Summary Delete child contract
// @Description Delete a contract by ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/contracts/{contractId} [delete]
func (h *ChildHandler) DeleteContract(c *gin.Context) {
	handleDeleteContract(c, "childId", h.contractAudit(), h.service.GetContractByID, h.service.DeleteContract,
		func(r *models.ChildContractResponse) (uint, uint) { return r.ID, r.ChildID },
		childContractSnapshot)
}

// childContractSnapshot captures a child contract's fields for the audit row of
// a deletion. Unlike an update, nothing survives to diff against afterwards, so
// this is the only remaining record of the care type and supplements the
// contract carried — the values that determined its funding.
func childContractSnapshot(r *models.ChildContractResponse) map[string]any {
	return map[string]any{
		"from":       timeValue(r.From),
		"to":         nullableTimeValue(r.To),
		"section_id": r.SectionID,
		"properties": r.Properties,
	}
}

// ExportYAML godoc
// @Summary Export children as YAML
// @Description Download children with contracts as a YAML file. Supports the same filters as the list endpoint.
// @Description Without filters, exports all children (suitable for backup/restore).
// @Tags children
// @Produce application/x-yaml
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param section_id query int false "Filter by section ID"
// @Param active_on query string false "Filter by active contract date (YYYY-MM-DD). Unlike the list endpoint, does NOT default to today — omit to export all."
// @Param contract_after query string false "Filter children with contracts starting after this date (YYYY-MM-DD). Mutually exclusive with active_on."
// @Param search query string false "Search by first or last name (case-insensitive)"
// @Success 200 {file} file "YAML file"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/export/yaml [get]
func (h *ChildHandler) ExportYAML(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	sectionID, ok := parseOptionalUint(c, "section_id")
	if !ok {
		return
	}

	contractAfter, ok := parseOptionalDatePtr(c, "contract_after")
	if !ok {
		return
	}

	activeOn, ok := parseOptionalDatePtr(c, "active_on")
	if !ok {
		return
	}

	search, ok := parseSearch(c)
	if !ok {
		return
	}

	filter := models.ChildListFilter{
		SectionID:     sectionID,
		ActiveOn:      activeOn,
		ContractAfter: contractAfter,
		Search:        search,
	}

	all, ok := fetchAllChildren(c, h.service, orgID, filter)
	if !ok {
		return
	}

	data := models.ChildImportExportData{Children: all}
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		respondError(c, apperror.Internal("failed to marshal YAML"))
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.Header("Content-Disposition", `attachment; filename="children.yaml"`)
	c.Writer.WriteHeader(http.StatusOK)
	if _, err := c.Writer.Write(yamlBytes); err != nil {
		slog.Error("failed to write YAML export response", "error", err)
	}
}

// Import godoc
// @Summary Import children from YAML
// @Description Upload a YAML file to create or update children with contracts (upsert by name+birthdate)
// @Tags children
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param file formData file true "Children YAML file"
// @Success 201 {array} models.ChildResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/import [post]
func (h *ChildHandler) Import(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	fileBytes, header, ok := readUploadFileWithHeader(c)
	if !ok {
		return
	}

	data, err := decodeYAMLStrict[models.ChildImportExportData](fileBytes)
	if err != nil {
		respondError(c, apperror.BadRequest("invalid YAML: "+err.Error()))
		return
	}

	results, err := h.service.Import(c.Request.Context(), orgID, data)
	if err != nil {
		respondError(c, err)
		return
	}

	ids := make([]uint, len(results))
	for i := range results {
		ids[i] = results[i].ID
	}
	h.auditService.LogResourceImport(c.Request.Context(), getUserID(c), getUserEmail(c),
		"child", orgID, len(results), ids, sanitizeFilename(header.Filename), c.ClientIP())

	c.JSON(http.StatusCreated, results)
}
