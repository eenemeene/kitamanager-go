package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

type UserHandler struct {
	service        *service.UserService
	userOrgService *service.UserOrganizationService
	auditService   *service.AuditService
	sessionStore   store.SessionStorer
}

func NewUserHandler(service *service.UserService, userOrgService *service.UserOrganizationService, auditService *service.AuditService, sessionStore store.SessionStorer) *UserHandler {
	return &UserHandler{
		service:        service,
		userOrgService: userOrgService,
		auditService:   auditService,
		sessionStore:   sessionStore,
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

	// Cross-post the user_update audit event to every org the user
	// belongs to (review finding M4). The /users/:userId route has
	// no :orgId in the URL, so the default auditUpdate path would
	// emit a single OrganizationID=NULL row — invisible to org
	// admins in their org-scoped audit feed. Look up the memberships
	// now and emit one row per org, plus the identity-level row.
	// Membership lookup failure must not 500 the update; degrade to
	// the legacy single-row path so we still produce an audit trail.
	orgIDs, lookupErr := h.userOrgService.ListMembershipOrgIDs(c.Request.Context(), user.ID)
	if lookupErr != nil {
		slog.Warn("failed to fetch user memberships for audit cross-post", "user_id", user.ID, "error", lookupErr)
		auditUpdate(c, h.auditService, "user", user.ID, user.Email)
	} else {
		h.auditService.LogResourceUpdateAcrossOrgs(c.Request.Context(), getUserID(c), getUserEmail(c),
			"user", user.ID, user.Email, c.ClientIP(), orgIDs)
	}

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

// Purge godoc
// @Summary Hard-delete a user (DSGVO Art. 17 erasure)
// @Description Physically removes a user and CASCADEs through the FK
// @Description graph (sessions, factors, user_organizations, audit
// @Description trail rows that don't reference the user). Bypasses the
// @Description tombstone soft-delete entirely. Irreversible.
// @Description
// @Description Restricted to superadmins. Cannot purge self. Cannot
// @Description purge the last superadmin. Accepts both live and
// @Description already-tombstoned rows so an Art. 17 request can fire
// @Description against either state.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/purge [delete]
func (h *UserHandler) Purge(c *gin.Context) {
	id, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	requesterID := getUserID(c)

	// Capture identity for audit BEFORE the cascade fires. GetByID
	// uses the GORM-scoped lookup; for a row that's already tombstoned
	// we still want to record what we're about to vaporize, so look
	// up unscoped.
	target, err := h.service.GetByIDIncludingTombstoned(c.Request.Context(), id, requesterID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.HardDelete(c.Request.Context(), id, requesterID); err != nil {
		respondError(c, err)
		return
	}

	if h.auditService != nil {
		h.auditService.LogResourcePurged(c.Request.Context(), requesterID, getUserEmail(c),
			"user", id, target.Email, c.ClientIP(), nil)
	}

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

	h.auditService.LogUserAddToOrg(c.Request.Context(), getUserID(c), getUserEmail(c), userID, req.OrganizationID, string(role), c.ClientIP())

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

	// Get current role before update for audit log. The requester already
	// has admin access to `orgID` (enforced by RequirePermission above), so
	// the scoped lookup will return at least that membership.
	requesterID := getUserID(c)
	memberships, err := h.userOrgService.GetUserMemberships(c.Request.Context(), userID, requesterID)
	if err != nil {
		slog.Warn("failed to fetch memberships for audit log", "user_id", userID, "error", err)
	}
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

	h.auditService.LogRoleChange(c.Request.Context(), getUserID(c), getUserEmail(c), userID, orgID, oldRole, string(req.Role), c.ClientIP())

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

	h.auditService.LogUserRemoveFromOrg(c.Request.Context(), getUserID(c), getUserEmail(c), userID, orgID, c.ClientIP())

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

	resp, err := h.userOrgService.GetUserMemberships(c.Request.Context(), userID, getUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetMyMemberships godoc
// @Summary Get the current user's organization memberships
// @Description Returns the calling user's own memberships and roles. Self-only —
// @Description does not require users:read. Used by the frontend on every page
// @Description load to populate orgRoleMap, which the sidebar's role-based nav
// @Description filtering depends on. The admin-facing /users/{userId}/memberships
// @Description still exists separately for user-management screens.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserMembershipsResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/me/memberships [get]
func (h *UserHandler) GetMyMemberships(c *gin.Context) {
	actorID := getUserID(c)
	resp, err := h.userOrgService.GetUserMemberships(c.Request.Context(), actorID, actorID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// SetSuperAdmin godoc
// @Summary Set user's superadmin status
// @Description Set or unset a user's superadmin status. Requires superadmin access
// @Description AND step-up authentication (the actor's current password in
// @Description `actor_password`). Symmetric with the admin password-reset endpoint:
// @Description without step-up a stolen superadmin session could mint a confederate
// @Description superadmin (or demote the actor) at full API rate.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserSetSuperAdminRequest true "Superadmin status + actor_password"
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

	// Get target user info up-front so the audit row (success or failure)
	// can carry the target email even if the step-up fails.
	actorID := getUserID(c)
	targetUser, err := h.service.GetByID(c.Request.Context(), userID, actorID)
	if err != nil {
		respondError(c, err)
		return
	}

	// Step-up: require the actor's current password before mutating
	// superadmin status. Closes the H1 finding from the architecture
	// review — without this a stolen superadmin session could promote a
	// confederate (or demote the actor) without proving control of the
	// actor's credentials.
	if err := h.service.VerifyActorPassword(c.Request.Context(), actorID, req.ActorPassword); err != nil {
		// Only the bcrypt-mismatch path emits a forensic row; a missing
		// actor_password is a client bug already surfaced as 400 by the
		// binding tag.
		if errors.Is(err, apperror.ErrUnauthorized) {
			h.auditService.LogSuperAdminChangeFailed(c.Request.Context(), actorID, getUserEmail(c),
				userID, targetUser.Email, req.IsSuperAdmin, c.ClientIP(), "invalid actor_password")
		}
		respondError(c, err)
		return
	}

	if err := h.userOrgService.SetSuperAdmin(c.Request.Context(), userID, req.IsSuperAdmin); err != nil {
		respondError(c, err)
		return
	}

	// Audit log superadmin change
	h.auditService.LogSuperAdminChange(c.Request.Context(), actorID, getUserEmail(c), userID, targetUser.Email, req.IsSuperAdmin, c.ClientIP())

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

	actorID := getUserID(c)

	if err := h.service.ResetPassword(c.Request.Context(), targetUserID, req.NewPassword, req.ActorPassword, actorID, c.ClientIP()); err != nil {
		respondError(c, err)
		return
	}

	// Revoke all sessions for the target user — if this fails, the password
	// was changed but old sessions remain valid, which violates the security
	// invariant. Return 500 so the caller knows the operation is incomplete.
	if h.sessionStore != nil {
		if err := h.sessionStore.DeleteAllForUser(c.Request.Context(), targetUserID); err != nil {
			slog.Error("failed to revoke sessions after password reset", "user_id", targetUserID, "error", err)
			respondError(c, apperror.InternalWrap(err, "password changed but failed to revoke existing sessions"))
			return
		}
	}

	// Audit log with dedicated password reset tracking
	targetUser, err := h.service.GetByID(c.Request.Context(), targetUserID, actorID)
	if err != nil {
		slog.Warn("failed to fetch user for audit log", "user_id", targetUserID, "error", err)
	}
	email := ""
	if targetUser != nil {
		email = targetUser.Email
	}
	h.auditService.LogPasswordReset(c.Request.Context(), actorID, getUserEmail(c), targetUserID, email, c.ClientIP())

	if actorID != targetUserID {
		slog.Warn("Admin reset another user's password",
			"actor_id", actorID,
			"target_user_id", targetUserID,
			"target_email", email,
			"ip", c.ClientIP(),
		)
	}

	c.Status(http.StatusNoContent)
}
