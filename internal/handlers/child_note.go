package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type ChildNoteHandler struct {
	service *service.ChildNoteService
}

func NewChildNoteHandler(service *service.ChildNoteService) *ChildNoteHandler {
	return &ChildNoteHandler{service: service}
}

// List godoc
// @Summary List notes for a child
// @Description Get a paginated list of notes for a child, optionally filtered by category
// @Tags child-notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param category query string false "Filter by category (observation, development, medical, incident, general, parent_note)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.ChildNoteResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/notes [get]
func (h *ChildNoteHandler) List(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	category := c.Query("category")

	notes, total, err := h.service.List(c.Request.Context(), childID, orgID, category, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(notes, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get a child note by ID
// @Description Get a single note for a child by note ID
// @Tags child-notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param noteId path int true "Note ID"
// @Success 200 {object} models.ChildNoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/notes/{noteId} [get]
func (h *ChildNoteHandler) Get(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	noteID, err := parseID(c, "noteId")
	if err != nil {
		respondError(c, err)
		return
	}

	note, err := h.service.GetByID(c.Request.Context(), noteID, childID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, note)
}

// Create godoc
// @Summary Create a note for a child
// @Description Create a new observation/documentation note for a child
// @Tags child-notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param request body models.ChildNoteCreateRequest true "Note data"
// @Success 201 {object} models.ChildNoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child not found"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/notes [post]
func (h *ChildNoteHandler) Create(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.ChildNoteCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	authorID := getUserID(c)
	note, err := h.service.Create(c.Request.Context(), childID, orgID, authorID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, note)
}

// Update godoc
// @Summary Update a child note
// @Description Update an existing note for a child
// @Tags child-notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param noteId path int true "Note ID"
// @Param request body models.ChildNoteUpdateRequest true "Note data"
// @Success 200 {object} models.ChildNoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/notes/{noteId} [put]
func (h *ChildNoteHandler) Update(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	noteID, err := parseID(c, "noteId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.ChildNoteUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	note, err := h.service.Update(c.Request.Context(), noteID, childID, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, note)
}

// Delete godoc
// @Summary Delete a child note
// @Description Delete a note for a child
// @Tags child-notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param noteId path int true "Note ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/notes/{noteId} [delete]
func (h *ChildNoteHandler) Delete(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	noteID, err := parseID(c, "noteId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), noteID, childID, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
