package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type ChildHandler struct {
	service *service.ChildService
}

func NewChildHandler(service *service.ChildService) *ChildHandler {
	return &ChildHandler{service: service}
}

// List godoc
// @Summary List all children in an organization
// @Description Get a paginated list of all children in the specified organization
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.Child]
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children [get]
func (h *ChildHandler) List(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		respondError(c, apperror.BadRequest("invalid pagination parameters"))
		return
	}
	if err := params.Validate(); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}
	params.SetDefaults()

	children, total, err := h.service.ListByOrganization(c.Request.Context(), orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(children, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get child by ID
// @Description Get a single child by their ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Success 200 {object} models.Child
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id} [get]
func (h *ChildHandler) Get(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
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
// @Success 201 {object} models.Child
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children [post]
func (h *ChildHandler) Create(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.ChildCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	child, err := h.service.Create(c.Request.Context(), orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, child)
}

// Update godoc
// @Summary Update a child
// @Description Update an existing child by ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param request body models.ChildUpdateRequest true "Child data"
// @Success 200 {object} models.Child
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id} [put]
func (h *ChildHandler) Update(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.ChildUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	child, err := h.service.Update(c.Request.Context(), id, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

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
// @Param id path int true "Child ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id} [delete]
func (h *ChildHandler) Delete(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListContracts godoc
// @Summary List child contracts
// @Description Get all contracts for a child
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Success 200 {array} models.ChildContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts [get]
func (h *ChildHandler) ListContracts(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	contracts, err := h.service.ListContracts(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contracts)
}

// GetCurrentContract godoc
// @Summary Get current child contract
// @Description Get the currently active contract for a child
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Success 200 {object} models.ChildContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts/current [get]
func (h *ChildHandler) GetCurrentContract(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	contract, err := h.service.GetCurrentContract(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract)
}

// CreateContract godoc
// @Summary Create child contract
// @Description Create a new contract for a child
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param request body models.ChildContractCreateRequest true "Contract data"
// @Success 201 {object} models.ChildContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Contract overlaps with existing"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts [post]
func (h *ChildHandler) CreateContract(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.ChildContractCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	contract, err := h.service.CreateContract(c.Request.Context(), id, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, contract)
}

// DeleteContract godoc
// @Summary Delete child contract
// @Description Delete a contract by ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts/{contractId} [delete]
func (h *ChildHandler) DeleteContract(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	contractID, err := parseID(c, "contractId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.DeleteContract(c.Request.Context(), contractID, id, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
