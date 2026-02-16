package service

import (
	"context"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// UserGroupService handles business logic for user-group-role operations
type UserGroupService struct {
	userGroupStore store.UserGroupStorer
	userStore      store.UserStorer
	groupStore     store.GroupStorer
}

// NewUserGroupService creates a new UserGroupService
func NewUserGroupService(userGroupStore store.UserGroupStorer, userStore store.UserStorer, groupStore store.GroupStorer) *UserGroupService {
	return &UserGroupService{
		userGroupStore: userGroupStore,
		userStore:      userStore,
		groupStore:     groupStore,
	}
}

// AddUserToGroup adds a user to a group with a specific role.
// requesterID is the user performing the operation (for authorization check).
func (s *UserGroupService) AddUserToGroup(ctx context.Context, userID, groupID uint, role models.Role, createdBy string, requesterID uint) (*models.UserGroupResponse, error) {
	// Validate role
	if !role.IsValid() {
		return nil, apperror.BadRequest("invalid role: must be admin, manager, or member")
	}

	// Verify user exists
	_, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return nil, apperror.NotFound("user")
	}

	// Verify group exists
	group, err := s.groupStore.FindByID(ctx, groupID)
	if err != nil {
		return nil, apperror.NotFound("group")
	}

	// Verify requester has admin access to the group's organization
	if err := s.verifyRequesterOrgAccess(ctx, requesterID, group.OrganizationID); err != nil {
		return nil, err
	}

	// Check if already a member
	exists, err := s.userGroupStore.Exists(ctx, userID, groupID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to check existing membership")
	}
	if exists {
		return nil, apperror.BadRequest("user is already a member of this group")
	}

	// Create membership
	ug, err := s.userGroupStore.AddUserToGroup(ctx, userID, groupID, role, createdBy)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to add user to group")
	}

	// Load group data for response
	ug.Group = group
	resp := ug.ToResponse()
	return &resp, nil
}

// UpdateUserGroupRole updates a user's role in a group.
// requesterID is the user performing the operation (for authorization check).
func (s *UserGroupService) UpdateUserGroupRole(ctx context.Context, userID, groupID uint, role models.Role, requesterID uint) (*models.UserGroupResponse, error) {
	// Validate role
	if !role.IsValid() {
		return nil, apperror.BadRequest("invalid role: must be admin, manager, or member")
	}

	// Verify membership exists
	ug, err := s.userGroupStore.FindByUserAndGroup(ctx, userID, groupID)
	if err != nil {
		return nil, apperror.NotFound("user-group membership")
	}

	// Load group to check org access
	group, err := s.groupStore.FindByID(ctx, groupID)
	if err != nil {
		return nil, apperror.NotFound("group")
	}

	// Verify requester has admin access to the group's organization
	if err := s.verifyRequesterOrgAccess(ctx, requesterID, group.OrganizationID); err != nil {
		return nil, err
	}

	// Update role
	if err := s.userGroupStore.UpdateRole(ctx, userID, groupID, role); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update role")
	}

	ug.Role = role
	ug.Group = group
	resp := ug.ToResponse()
	return &resp, nil
}

// RemoveUserFromGroup removes a user from a group.
// requesterID is the user performing the operation (for authorization check).
func (s *UserGroupService) RemoveUserFromGroup(ctx context.Context, userID, groupID uint, requesterID uint) error {
	// Verify group exists and get org
	group, err := s.groupStore.FindByID(ctx, groupID)
	if err != nil {
		return apperror.NotFound("group")
	}

	// Verify requester has admin access to the group's organization
	if err := s.verifyRequesterOrgAccess(ctx, requesterID, group.OrganizationID); err != nil {
		return err
	}

	// Check if membership exists
	exists, err := s.userGroupStore.Exists(ctx, userID, groupID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check membership")
	}
	if !exists {
		return apperror.NotFound("user-group membership")
	}

	if err := s.userGroupStore.RemoveUserFromGroup(ctx, userID, groupID); err != nil {
		return apperror.InternalWrap(err, "failed to remove user from group")
	}
	return nil
}

// GetUserMemberships returns all group memberships for a user with effective org roles
func (s *UserGroupService) GetUserMemberships(ctx context.Context, userID uint) (*models.UserMembershipsResponse, error) {
	// Verify user exists
	_, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return nil, apperror.NotFound("user")
	}

	memberships, err := s.userGroupStore.FindByUser(ctx, userID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch memberships")
	}

	// Calculate effective org roles
	orgRoles, err := s.userGroupStore.GetUserOrganizationsWithRoles(ctx, userID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to calculate effective roles")
	}

	result := make([]models.UserMembership, 0, len(memberships))
	for _, m := range memberships {
		var org *models.Organization
		var effectiveRole models.Role

		if m.Group != nil {
			org = m.Group.Organization
			effectiveRole = orgRoles[m.Group.OrganizationID]
		}

		result = append(result, models.UserMembership{
			UserID:           m.UserID,
			GroupID:          m.GroupID,
			Role:             m.Role,
			Group:            m.Group,
			EffectiveOrgRole: effectiveRole,
			Organization:     org,
		})
	}

	return &models.UserMembershipsResponse{Memberships: result}, nil
}

// SetSuperAdmin sets or unsets superadmin status for a user
func (s *UserGroupService) SetSuperAdmin(ctx context.Context, userID uint, isSuperAdmin bool) error {
	// Verify user exists
	_, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("user")
	}

	if err := s.userGroupStore.SetSuperAdmin(ctx, userID, isSuperAdmin); err != nil {
		return apperror.InternalWrap(err, "failed to update superadmin status")
	}
	return nil
}

// AddUserToOrganization adds a user to an organization's default group with member role.
// requesterID is the user performing the operation (for authorization check).
func (s *UserGroupService) AddUserToOrganization(ctx context.Context, userID, orgID uint, createdBy string, requesterID uint) (*models.UserGroupResponse, error) {
	// Verify requester has admin access to the target organization
	if err := s.verifyRequesterOrgAccess(ctx, requesterID, orgID); err != nil {
		return nil, err
	}

	// Find default group for organization
	defaultGroup, err := s.groupStore.FindDefaultGroup(ctx, orgID)
	if err != nil {
		return nil, apperror.NotFound("organization or default group")
	}

	return s.AddUserToGroup(ctx, userID, defaultGroup.ID, models.RoleMember, createdBy, requesterID)
}

// RemoveUserFromOrganization removes a user from all groups in an organization.
// requesterID is the user performing the operation (for authorization check).
func (s *UserGroupService) RemoveUserFromOrganization(ctx context.Context, userID, orgID uint, requesterID uint) error {
	// Verify requester has admin access to the target organization
	if err := s.verifyRequesterOrgAccess(ctx, requesterID, orgID); err != nil {
		return err
	}

	// Verify user exists
	_, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("user")
	}

	if err := s.userGroupStore.RemoveUserFromOrg(ctx, userID, orgID); err != nil {
		return apperror.InternalWrap(err, "failed to remove user from organization")
	}
	return nil
}

// verifyRequesterOrgAccess checks that the requester is a superadmin or has admin role
// in the given organization. Returns apperror.Forbidden if not authorized.
func (s *UserGroupService) verifyRequesterOrgAccess(ctx context.Context, requesterID, orgID uint) error {
	isSuperAdmin, err := s.userGroupStore.IsSuperAdmin(ctx, requesterID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check superadmin status")
	}
	if isSuperAdmin {
		return nil
	}

	role, err := s.userGroupStore.GetEffectiveRoleInOrg(ctx, requesterID, orgID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check organization access")
	}
	if role != models.RoleAdmin {
		return apperror.Forbidden("insufficient permissions for this organization")
	}
	return nil
}
