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
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	search := c.Query("search")

	sections, total, err := h.service.ListByOrganization(c.Request.Context(), orgID, search, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(sections, params.Page, params.Limit, total, c.Request.URL.Path))
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
	orgID, sectionID, ok := parseOrgAndResourceID(c, "sectionId")
	if !ok {
		return
	}

	section, err := h.service.GetByIDAndOrg(c.Request.Context(), sectionID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, section)
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
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[models.SectionCreateRequest](c)
	if !ok {
		return
	}

	section, err := h.service.Create(c.Request.Context(), orgID, req, getCreatedBy(c))
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "section", section.ID, section.Name)

	c.JSON(http.StatusCreated, section)
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
	orgID, sectionID, ok := parseOrgAndResourceID(c, "sectionId")
	if !ok {
		return
	}

	req, ok := bindJSON[models.SectionUpdateRequest](c)
	if !ok {
		return
	}

	section, err := h.service.UpdateByIDAndOrg(c.Request.Context(), sectionID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "section", section.ID, section.Name)

	c.JSON(http.StatusOK, section)
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
	orgID, sectionID, ok := parseOrgAndResourceID(c, "sectionId")
	if !ok {
		return
	}

	// Get section info before deletion for audit log
	section, err := h.service.GetByIDAndOrg(c.Request.Context(), sectionID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.DeleteByIDAndOrg(c.Request.Context(), sectionID, orgID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "section", sectionID, section.Name)

	c.Status(http.StatusNoContent)
}
