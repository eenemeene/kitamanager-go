package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

type UserHandler struct {
	service        *service.UserService
	userOrgService *service.UserOrganizationService
	auditService   *service.AuditService
	tokenStore     store.TokenStorer
}

func NewUserHandler(service *service.UserService, userOrgService *service.UserOrganizationService, auditService *service.AuditService, tokenStore store.TokenStorer) *UserHandler {
	return &UserHandler{
		service:        service,
		userOrgService: userOrgService,
		auditService:   auditService,
		tokenStore:     tokenStore,
	}
}

// List godoc
// @Summary List all users
// @Description Get a paginated list of all users
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name or email (case-insensitive)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.UserResponse]
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	params, ok := parsePagination(c)
	if !ok {
		return
	}

	search, ok := parseSearch(c)
	if !ok {
		return
	}

	users, total, err := h.service.List(c.Request.Context(), getUserID(c), search, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(users, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// ListByOrganization godoc
// @Summary List users in an organization
// @Description Get a paginated list of users who are members of the specified organization
// @Tags users,organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param search query string false "Search by name or email (case-insensitive)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.UserResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/users [get]
func (h *UserHandler) ListByOrganization(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	search, ok := parseSearch(c)
	if !ok {
		return
	}

	users, total, err := h.service.ListByOrganization(c.Request.Context(), orgID, search, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(users, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// Get godoc
// @Summary Get user by ID
// @Description Get a single user by their ID
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id, getUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// Create godoc
// @Summary Create a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UserCreateRequest true "User data"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	req, ok := bindJSON[models.UserCreateRequest](c)
	if !ok {
		return
	}

	user, err := h.service.Create(c.Request.Context(), req, getCreatedBy(c))
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "user", user.ID, user.Email)

	c.JSON(http.StatusCreated, user)
}

// Update godoc
// @Summary Update a user
// @Description Update an existing user by ID
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserUpdateRequest true "User data"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserUpdateRequest](c)
	if !ok {
		return
	}

	user, err := h.service.Update(c.Request.Context(), id, req, getUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "user", user.ID, user.Email)

	c.JSON(http.StatusOK, user)
}

// Delete godoc
// @Summary Delete a user
// @Description Delete a user by ID
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	// Get user info before deletion for audit log
	requesterID := getUserID(c)
	user, err := h.service.GetByID(c.Request.Context(), id, requesterID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, requesterID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "user", id, user.Email)

	c.Status(http.StatusNoContent)
}

// AddToOrganization godoc
// @Summary Add user to organization
// @Description Add a user to an organization with a specific role
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserAddOrganizationRequest true "Organization ID and role"
// @Success 201 {object} models.UserOrganizationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/organizations [post]
func (h *UserHandler) AddToOrganization(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserAddOrganizationRequest](c)
	if !ok {
		return
	}

	role := req.Role
	if role == "" {
		role = models.RoleMember
	}

	resp, err := h.userOrgService.AddUserToOrganization(c.Request.Context(), userID, req.OrganizationID, role, getCreatedBy(c), getUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}

	h.auditService.LogUserAddToOrg(getUserID(c), userID, req.OrganizationID, string(role), c.ClientIP())

	c.JSON(http.StatusCreated, resp)
}

// UpdateOrganizationRole godoc
// @Summary Update user's role in organization
// @Description Update a user's role within a specific organization
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param orgId path int true "Organization ID"
// @Param request body models.UserOrganizationRoleUpdateRequest true "New role"
// @Success 200 {object} models.UserOrganizationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/organizations/{orgId} [put]
func (h *UserHandler) UpdateOrganizationRole(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserOrganizationRoleUpdateRequest](c)
	if !ok {
		return
	}

	// Get current role before update for audit log
	memberships, _ := h.userOrgService.GetUserMemberships(c.Request.Context(), userID)
	oldRole := ""
	if memberships != nil {
		for _, m := range memberships.Memberships {
			if m.OrganizationID == orgID {
				oldRole = string(m.Role)
				break
			}
		}
	}

	resp, err := h.userOrgService.UpdateUserOrganizationRole(c.Request.Context(), userID, orgID, req.Role, getUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}

	h.auditService.LogRoleChange(getUserID(c), userID, orgID, oldRole, string(req.Role), c.ClientIP())

	c.JSON(http.StatusOK, resp)
}

// RemoveFromOrganization godoc
// @Summary Remove user from organization
// @Description Remove a user from an organization
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param orgId path int true "Organization ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/organizations/{orgId} [delete]
func (h *UserHandler) RemoveFromOrganization(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.userOrgService.RemoveUserFromOrganization(c.Request.Context(), userID, orgID, getUserID(c)); err != nil {
		respondError(c, err)
		return
	}

	h.auditService.LogUserRemoveFromOrg(getUserID(c), userID, orgID, c.ClientIP())

	c.Status(http.StatusNoContent)
}

// GetMemberships godoc
// @Summary Get user's organization memberships
// @Description Get all organization memberships for a user with their roles
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Success 200 {object} models.UserMembershipsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/memberships [get]
func (h *UserHandler) GetMemberships(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	resp, err := h.userOrgService.GetUserMemberships(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SetSuperAdmin godoc
// @Summary Set user's superadmin status
// @Description Set or unset a user's superadmin status. Requires superadmin access.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserSetSuperAdminRequest true "Superadmin status"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/superadmin [put]
func (h *UserHandler) SetSuperAdmin(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserSetSuperAdminRequest](c)
	if !ok {
		return
	}

	// Get user info before change for audit log
	actorID := getUserID(c)
	targetUser, err := h.service.GetByID(c.Request.Context(), userID, actorID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.userOrgService.SetSuperAdmin(c.Request.Context(), userID, req.IsSuperAdmin); err != nil {
		respondError(c, err)
		return
	}

	// Audit log superadmin change
	h.auditService.LogSuperAdminChange(actorID, userID, targetUser.Email, req.IsSuperAdmin, c.ClientIP())

	// Return updated user
	user, err := h.service.GetByID(c.Request.Context(), userID, actorID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// ResetPassword godoc
// @Summary Reset a user's password (admin)
// @Description Admin-initiated password reset. Sets a new password for the specified user and revokes all their tokens.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserPasswordResetRequest true "New password"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/password [put]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	targetUserID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserPasswordResetRequest](c)
	if !ok {
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), targetUserID, req.NewPassword); err != nil {
		respondError(c, err)
		return
	}

	// Revoke all tokens for the target user
	if h.tokenStore != nil {
		if err := h.tokenStore.RevokeAllForUser(c.Request.Context(), targetUserID); err != nil {
			slog.Error("failed to revoke tokens after password reset", "user_id", targetUserID, "error", err)
		}
	}

	// Audit log
	actorID := getUserID(c)
	targetUser, _ := h.service.GetByID(c.Request.Context(), targetUserID, actorID)
	email := ""
	if targetUser != nil {
		email = targetUser.Email
	}
	h.auditService.LogResourceUpdate(actorID, "user_password_reset", targetUserID, email, c.ClientIP())

	c.Status(http.StatusNoContent)
}
