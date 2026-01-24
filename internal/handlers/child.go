package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

type ChildHandler struct {
	store *store.ChildStore
}

func NewChildHandler(store *store.ChildStore) *ChildHandler {
	return &ChildHandler{store: store}
}

// List godoc
// @Summary List all children
// @Description Get a list of all children
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Child
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/children [get]
func (h *ChildHandler) List(c *gin.Context) {
	children, err := h.store.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch children"})
		return
	}
	c.JSON(http.StatusOK, children)
}

// Get godoc
// @Summary Get child by ID
// @Description Get a single child by their ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Child ID"
// @Success 200 {object} models.Child
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/children/{id} [get]
func (h *ChildHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	child, err := h.store.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "child not found"})
		return
	}

	c.JSON(http.StatusOK, child)
}

// Create godoc
// @Summary Create a new child
// @Description Create a new child
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChildCreate true "Child data"
// @Success 201 {object} models.Child
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/children [post]
func (h *ChildHandler) Create(c *gin.Context) {
	var req models.ChildCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	child := &models.Child{
		Person: models.Person{
			OrganizationID: req.OrganizationID,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Birthdate:      req.Birthdate,
		},
	}

	if err := h.store.Create(child); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create child"})
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
// @Param id path int true "Child ID"
// @Param request body models.ChildUpdate true "Child data"
// @Success 200 {object} models.Child
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/children/{id} [put]
func (h *ChildHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	child, err := h.store.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "child not found"})
		return
	}

	var req models.ChildUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FirstName != nil {
		child.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		child.LastName = *req.LastName
	}
	if req.Birthdate != nil {
		child.Birthdate = *req.Birthdate
	}

	if err := h.store.Update(child); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update child"})
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
// @Param id path int true "Child ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/children/{id} [delete]
func (h *ChildHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.store.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete child"})
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
// @Param id path int true "Child ID"
// @Success 200 {array} models.ChildContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/children/{id}/contracts [get]
func (h *ChildHandler) ListContracts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	contracts, err := h.store.Contracts.GetHistory(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch contracts"})
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
// @Param id path int true "Child ID"
// @Success 200 {object} models.ChildContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/children/{id}/contracts/current [get]
func (h *ChildHandler) GetCurrentContract(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	contract, err := h.store.Contracts.GetCurrentContract(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch contract"})
		return
	}
	if contract == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active contract"})
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
// @Param id path int true "Child ID"
// @Param request body models.ChildContractCreate true "Contract data"
// @Success 201 {object} models.ChildContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Contract overlaps with existing"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/children/{id}/contracts [post]
func (h *ChildHandler) CreateContract(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.ChildContractCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate no overlap
	if err := h.store.Contracts.ValidateNoOverlap(uint(id), req.From, req.To, nil); err != nil {
		if errors.Is(err, store.ErrContractOverlap) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate contract"})
		return
	}

	contract := &models.ChildContract{
		ChildID: uint(id),
		Period: models.Period{
			From: req.From,
			To:   req.To,
		},
		CareHoursPerWeek: req.CareHoursPerWeek,
		GroupID:          req.GroupID,
		MealsIncluded:    req.MealsIncluded,
		SpecialNeeds:     req.SpecialNeeds,
	}

	if err := h.store.CreateContract(contract); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create contract"})
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
// @Param id path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/children/{id}/contracts/{contractId} [delete]
func (h *ChildHandler) DeleteContract(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contractId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract id"})
		return
	}

	if err := h.store.DeleteContract(uint(contractID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete contract"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
