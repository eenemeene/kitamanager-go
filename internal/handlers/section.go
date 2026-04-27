package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type SectionHandler struct {
	service      *service.SectionService
	auditService *service.AuditService
}

func NewSectionHandler(service *service.SectionService, auditService *service.AuditService) *SectionHandler {
	return &SectionHandler{service: service, auditService: auditService}
}

func (h *SectionHandler) audit() auditConfig {
	return auditConfig{auditService: h.auditService, resourceType: "section"}
}

func sectionAuditInfo(s *models.SectionResponse) (uint, string) { return s.ID, s.Name }

// List godoc
// @Summary List sections in an organization
// @Description Get a paginated list of sections within a specific organization
// @Tags sections
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param search query string false "Search by name (case-insensitive)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.SectionResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/sections [get]
func (h *SectionHandler) List(c *gin.Context) {
	handleOrgList(c, h.service.ListByOrganization)
}

// Get godoc
// @Summary Get section by ID
// @Description Get a single section by its ID within an organization
// @Tags sections
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param sectionId path int true "Section ID"
// @Success 200 {object} models.SectionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/sections/{sectionId} [get]
func (h *SectionHandler) Get(c *gin.Context) {
	handleOrgGet(c, "sectionId", h.service.GetByIDAndOrg)
}

// Create godoc
// @Summary Create a new section
// @Description Create a new section within an organization
// @Tags sections
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.SectionCreateRequest true "Section data"
// @Success 201 {object} models.SectionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/sections [post]
func (h *SectionHandler) Create(c *gin.Context) {
	handleOrgCreate(c, h.audit(), h.service.Create, sectionAuditInfo)
}

// Update godoc
// @Summary Update a section
// @Description Update an existing section by ID within an organization
// @Tags sections
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param sectionId path int true "Section ID"
// @Param request body models.SectionUpdateRequest true "Section data"
// @Success 200 {object} models.SectionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/sections/{sectionId} [put]
func (h *SectionHandler) Update(c *gin.Context) {
	handleOrgUpdate(c, "sectionId", h.audit(), h.service.UpdateByIDAndOrg, sectionAuditInfo)
}

// Delete godoc
// @Summary Delete a section
// @Description Delete a section by ID within an organization. Cannot delete sections with assigned children or employees.
// @Tags sections
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param sectionId path int true "Section ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse "Cannot delete section with assigned children/employees or default section"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/sections/{sectionId} [delete]
func (h *SectionHandler) Delete(c *gin.Context) {
	handleOrgDelete(c, "sectionId", h.audit(), h.service.GetByIDAndOrg, h.service.DeleteByIDAndOrg, sectionAuditInfo)
}

// PromoteToDefault godoc
// @Summary Set a section as the organization's default
// @Description Atomically clears the previous default flag (if any) and
// @Description sets is_default = true on the given section. The
// @Description (organization_id) WHERE is_default index from migration
// @Description 000019 enforces "exactly one default per org" at the DB
// @Description layer; this endpoint is the only sanctioned way to flip
// @Description the flag through the API.
// @Tags sections
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param sectionId path int true "Section ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/sections/{sectionId}/promote-default [post]
func (h *SectionHandler) PromoteToDefault(c *gin.Context) {
	orgID, sectionID, ok := parseOrgAndResourceID(c, "sectionId")
	if !ok {
		return
	}

	// Fetch the name BEFORE the update so the audit row carries
	// human-readable context. Lookup uses the same NotFound semantics
	// the service then re-runs (cross-org leaks remain prevented).
	section, err := h.service.GetByIDAndOrg(c.Request.Context(), sectionID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.PromoteToDefault(c.Request.Context(), sectionID, orgID); err != nil {
		respondError(c, err)
		return
	}

	// Resource type discriminator carries the "this was a promotion"
	// semantic so audit consumers can grep for "section_default_set"
	// without a new AuditAction enum value.
	auditUpdate(c, h.auditService, "section_default_set", sectionID, section.Name)

	c.Status(http.StatusNoContent)
}
