package service

import (
	"context"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// UserOrganizationService handles business logic for user-organization-role operations
type UserOrganizationService struct {
	userOrgStore store.UserOrganizationStorer
	userStore    store.UserStorer
	transactor   store.Transactor
}

// NewUserOrganizationService creates a new UserOrganizationService
func NewUserOrganizationService(userOrgStore store.UserOrganizationStorer, userStore store.UserStorer, transactor store.Transactor) *UserOrganizationService {
	return &UserOrganizationService{
		userOrgStore: userOrgStore,
		userStore:    userStore,
		transactor:   transactor,
	}
}

// AddUserToOrganization adds a user to an organization with a specific role.
// requesterID is the user performing the operation (for authorization check).
func (s *UserOrganizationService) AddUserToOrganization(ctx context.Context, userID, orgID uint, role models.Role, createdBy string, requesterID uint) (*models.UserOrganizationResponse, error) {
	// Validate role
	if !role.IsValid() {
		return nil, apperror.BadRequest("invalid role: must be admin, manager, or member")
	}

	// Verify user exists
	_, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return nil, classifyStoreError(err, "user")
	}

	// Verify requester has admin access to the organization
	if err := s.verifyRequesterOrgAccess(ctx, requesterID, orgID); err != nil {
		return nil, err
	}

	// Check + create in a single transaction to prevent race conditions
	var uo *models.UserOrganization
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		exists, err := s.userOrgStore.Exists(txCtx, userID, orgID)
		if err != nil {
			return apperror.InternalWrap(err, "failed to check existing membership")
		}
		if exists {
			return apperror.BadRequest("user is already a member of this organization")
		}

		uo, err = s.userOrgStore.AddUserToOrg(txCtx, userID, orgID, role, createdBy)
		if err != nil {
			return apperror.InternalWrap(err, "failed to add user to organization")
		}
		return nil
	}); err != nil {
		return nil, err
	}

	resp := uo.ToResponse()
	return &resp, nil
}

// UpdateUserOrganizationRole updates a user's role in an organization.
// requesterID is the user performing the operation (for authorization check).
func (s *UserOrganizationService) UpdateUserOrganizationRole(ctx context.Context, userID, orgID uint, role models.Role, requesterID uint) (*models.UserOrganizationResponse, error) {
	// Validate role
	if !role.IsValid() {
		return nil, apperror.BadRequest("invalid role: must be admin, manager, or member")
	}

	// Verify membership exists
	uo, err := s.userOrgStore.FindByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return nil, classifyStoreError(err, "user-organization membership")
	}

	// Verify requester has admin access to the organization
	if err := s.verifyRequesterOrgAccess(ctx, requesterID, orgID); err != nil {
		return nil, err
	}

	// Update role
	if err := s.userOrgStore.UpdateRole(ctx, userID, orgID, role); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update role")
	}

	uo.Role = role
	resp := uo.ToResponse()
	return &resp, nil
}

// RemoveUserFromOrganization removes a user from an organization.
// requesterID is the user performing the operation (for authorization check).
func (s *UserOrganizationService) RemoveUserFromOrganization(ctx context.Context, userID, orgID uint, requesterID uint) error {
	// Verify requester has admin access to the organization
	if err := s.verifyRequesterOrgAccess(ctx, requesterID, orgID); err != nil {
		return err
	}

	// Verify user exists
	_, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return classifyStoreError(err, "user")
	}

	// Check + delete in a single transaction to prevent race conditions
	return s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		exists, err := s.userOrgStore.Exists(txCtx, userID, orgID)
		if err != nil {
			return apperror.InternalWrap(err, "failed to check membership")
		}
		if !exists {
			return apperror.NotFound("user-organization membership")
		}

		if err := s.userOrgStore.RemoveUserFromOrg(txCtx, userID, orgID); err != nil {
			return apperror.InternalWrap(err, "failed to remove user from organization")
		}
		return nil
	})
}

// GetUserMemberships returns the organization memberships for a user, scoped
// to what the requester is allowed to see:
//   - Superadmin: every membership.
//   - Self (requesterID == targetUserID): every membership.
//   - Anyone else: only memberships in organizations that the requester is
//     also a member of. If there is no overlap, returns NotFound to avoid
//     leaking the target's existence.
//
// Without this scoping any user with global users:read (e.g. a member in any
// org) could enumerate another user's complete org graph — including orgs
// they should not even know about. See H10.
func (s *UserOrganizationService) GetUserMemberships(ctx context.Context, targetUserID, requesterID uint) (*models.UserMembershipsResponse, error) {
	if _, err := s.userStore.FindByID(ctx, targetUserID); err != nil {
		return nil, classifyStoreError(err, "user")
	}

	memberships, err := s.userOrgStore.FindByUser(ctx, targetUserID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch memberships")
	}

	filtered, err := s.filterMembershipsForRequester(ctx, memberships, targetUserID, requesterID)
	if err != nil {
		return nil, err
	}

	result := make([]models.UserMembership, 0, len(filtered))
	for _, m := range filtered {
		result = append(result, models.UserMembership{
			UserID:         m.UserID,
			OrganizationID: m.OrganizationID,
			Role:           m.Role,
			Organization:   m.Organization,
		})
	}

	return &models.UserMembershipsResponse{Memberships: result}, nil
}

// filterMembershipsForRequester applies the visibility rules described on
// GetUserMemberships. It returns NotFound when the requester has no
// authorization to see any of the target's memberships.
func (s *UserOrganizationService) filterMembershipsForRequester(ctx context.Context, memberships []models.UserOrganization, targetUserID, requesterID uint) ([]models.UserOrganization, error) {
	if requesterID == targetUserID {
		return memberships, nil
	}

	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to check superadmin status")
	}
	if isSuperAdmin {
		return memberships, nil
	}

	requesterOrgIDs, err := s.requesterOrgSet(ctx, requesterID)
	if err != nil {
		return nil, err
	}

	filtered := make([]models.UserOrganization, 0, len(memberships))
	for _, m := range memberships {
		if requesterOrgIDs[m.OrganizationID] {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		// NotFound rather than Forbidden to avoid confirming the target
		// user's existence to callers outside their trust scope.
		return nil, apperror.NotFound("user")
	}
	return filtered, nil
}

// requesterOrgSet returns the set of organization IDs the requester belongs
// to. Used to scope visibility of other users' memberships.
func (s *UserOrganizationService) requesterOrgSet(ctx context.Context, requesterID uint) (map[uint]bool, error) {
	own, err := s.userOrgStore.FindByUser(ctx, requesterID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch requester memberships")
	}
	out := make(map[uint]bool, len(own))
	for _, m := range own {
		out[m.OrganizationID] = true
	}
	return out, nil
}

// ListMembershipOrgIDs returns every organization id `targetUserID` is a
// member of, with no visibility filtering. Intended for the audit log
// cross-post path (review finding M4): when a global PUT /users/:userId
// updates a user, the audit row must be visible to every org admin whose
// org contains the user, not just to the superadmin global feed.
//
// Distinct from GetUserMemberships, which scopes results to what the
// requester is allowed to see — the audit cross-post has no requester
// (the system is the actor), so the visibility filter would suppress
// exactly the rows the org admins need to see.
func (s *UserOrganizationService) ListMembershipOrgIDs(ctx context.Context, targetUserID uint) ([]uint, error) {
	memberships, err := s.userOrgStore.FindByUser(ctx, targetUserID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch memberships")
	}
	out := make([]uint, len(memberships))
	for i, m := range memberships {
		out[i] = m.OrganizationID
	}
	return out, nil
}

// SetSuperAdmin sets or unsets superadmin status for a user.
// The last superadmin cannot be demoted.
func (s *UserOrganizationService) SetSuperAdmin(ctx context.Context, userID uint, isSuperAdmin bool) error {
	// Verify user exists
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return classifyStoreError(err, "user")
	}

	// Prevent demoting the last superadmin
	if !isSuperAdmin && user.IsSuperAdmin {
		count, err := s.userOrgStore.CountSuperAdmins(ctx)
		if err != nil {
			return apperror.InternalWrap(err, "failed to count superadmins")
		}
		if count <= 1 {
			return apperror.BadRequest("cannot demote the last superadmin")
		}
	}

	if err := s.userOrgStore.SetSuperAdmin(ctx, userID, isSuperAdmin); err != nil {
		return apperror.InternalWrap(err, "failed to update superadmin status")
	}
	return nil
}

// verifyRequesterOrgAccess checks that the requester is a superadmin or has admin role
// in the given organization. Returns apperror.Forbidden if not authorized.
func (s *UserOrganizationService) verifyRequesterOrgAccess(ctx context.Context, requesterID, orgID uint) error {
	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check superadmin status")
	}
	if isSuperAdmin {
		return nil
	}

	role, err := s.userOrgStore.GetRoleInOrg(ctx, requesterID, orgID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check organization access")
	}
	if role != models.RoleAdmin {
		return apperror.Forbidden("insufficient permissions for this organization")
	}
	return nil
}
