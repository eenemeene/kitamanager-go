package service

import (
	"context"
	"log/slog"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// UserService handles business logic for user operations
type UserService struct {
	store        store.UserStorer
	userOrgStore store.UserOrganizationStorer
	tokenStore   store.TokenStorer
}

// NewUserService creates a new user service
func NewUserService(store store.UserStorer, userOrgStore store.UserOrganizationStorer, tokenStore ...store.TokenStorer) *UserService {
	svc := &UserService{store: store, userOrgStore: userOrgStore}
	if len(tokenStore) > 0 {
		svc.tokenStore = tokenStore[0]
	}
	return svc
}

// List returns a paginated list of users visible to the requester.
// Superadmins see all users; other users see only users who share at least one organization.
func (s *UserService) List(ctx context.Context, requesterID uint, search string, limit, offset int) ([]models.UserResponse, int64, error) {
	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to check superadmin status")
	}

	if isSuperAdmin {
		users, total, err := s.store.FindAll(ctx, search, limit, offset)
		if err != nil {
			return nil, 0, apperror.InternalWrap(err, "failed to fetch users")
		}
		return toResponseList(users, (*models.User).ToResponse), total, nil
	}

	orgRoles, err := s.userOrgStore.GetUserOrganizationsWithRoles(ctx, requesterID)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch requester organizations")
	}

	orgIDs := make([]uint, 0, len(orgRoles))
	for orgID := range orgRoles {
		orgIDs = append(orgIDs, orgID)
	}

	if len(orgIDs) == 0 {
		return []models.UserResponse{}, 0, nil
	}

	users, total, err := s.store.FindByOrganizations(ctx, orgIDs, search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch users")
	}

	return toResponseList(users, (*models.User).ToResponse), total, nil
}

// ListByOrganization returns a paginated list of users in a specific organization
func (s *UserService) ListByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.UserResponse, int64, error) {
	users, total, err := s.store.FindByOrganization(ctx, orgID, search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch users")
	}

	return toResponseList(users, (*models.User).ToResponse), total, nil
}

// GetByID returns a user by ID. Users can always view themselves.
// For other users, requester must be a superadmin or share an organization.
func (s *UserService) GetByID(ctx context.Context, id uint, requesterID uint) (*models.UserResponse, error) {
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "user")
	}

	if err := s.verifyRequesterCanAccessUser(ctx, requesterID, id); err != nil {
		return nil, apperror.NotFound("user")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, req *models.UserCreateRequest, createdBy string) (*models.UserResponse, error) {
	name, err := validateRequiredName(req.Name)
	if err != nil {
		return nil, err
	}
	req.Email = strings.TrimSpace(req.Email)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to hash password")
	}

	user := &models.User{
		Name:      name,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Active:    req.Active,
		CreatedBy: createdBy,
	}

	if err := s.store.Create(ctx, user); err != nil {
		if store.IsDuplicateKeyError(err) {
			return nil, apperror.EmailConflict()
		}
		return nil, apperror.InternalWrap(err, "failed to create user")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// Update updates an existing user
func (s *UserService) Update(ctx context.Context, id uint, req *models.UserUpdateRequest, requesterID uint) (*models.UserResponse, error) {
	if err := s.verifyRequesterCanAccessUser(ctx, requesterID, id); err != nil {
		return nil, apperror.NotFound("user")
	}

	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "user")
	}

	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if req.Name != "" {
		if validation.IsWhitespaceOnly(req.Name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		user.Name = req.Name
	}
	if req.Email != "" {
		// Check if email is already used by another user
		exists, err := s.store.EmailExistsForOtherUser(ctx, req.Email, id)
		if err != nil {
			return nil, apperror.InternalWrap(err, "failed to validate email")
		}
		if exists {
			return nil, apperror.EmailConflict()
		}
		user.Email = req.Email
	}
	deactivating := req.Active != nil && !*req.Active && user.Active
	if req.Active != nil {
		user.Active = *req.Active
	}

	if err := s.store.Update(ctx, user); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update user")
	}

	// Revoke all tokens when a user is deactivated
	if deactivating && s.tokenStore != nil {
		if err := s.tokenStore.RevokeAllForUser(ctx, id); err != nil {
			slog.Error("failed to revoke tokens after user deactivation", "user_id", id, "error", err)
		}
	}

	resp := user.ToResponse()
	return &resp, nil
}

// ResetPassword sets a new password for a user (admin-initiated).
// Non-superadmin requesters cannot reset a superadmin's password.
func (s *UserService) ResetPassword(ctx context.Context, userID uint, newPassword string, requesterID uint) error {
	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return classifyStoreError(err, "user")
	}

	// Prevent non-superadmin from resetting a superadmin's password
	if user.IsSuperAdmin && requesterID != userID {
		requesterIsSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
		if err != nil {
			return apperror.InternalWrap(err, "failed to check superadmin status")
		}
		if !requesterIsSuperAdmin {
			return apperror.Forbidden("only superadmins can reset a superadmin's password")
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.InternalWrap(err, "failed to hash password")
	}

	user.Password = string(hashedPassword)
	if err := s.store.Update(ctx, user); err != nil {
		return apperror.InternalWrap(err, "failed to update password")
	}
	return nil
}

// Delete deletes a user.
// Users cannot delete themselves. The last superadmin cannot be deleted.
func (s *UserService) Delete(ctx context.Context, id uint, requesterID uint) error {
	if id == requesterID {
		return apperror.BadRequest("cannot delete your own account")
	}

	if err := s.verifyRequesterCanAccessUser(ctx, requesterID, id); err != nil {
		return apperror.NotFound("user")
	}

	// Prevent deleting the last superadmin
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return classifyStoreError(err, "user")
	}
	if user.IsSuperAdmin {
		count, err := s.userOrgStore.CountSuperAdmins(ctx)
		if err != nil {
			return apperror.InternalWrap(err, "failed to count superadmins")
		}
		if count <= 1 {
			return apperror.BadRequest("cannot delete the last superadmin")
		}
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return apperror.InternalWrap(err, "failed to delete user")
	}
	return nil
}

// verifyRequesterCanAccessUser checks that the requester can access the target user.
// Superadmins can access all users. A user can always access themselves.
// Others can only access users who share at least one organization.
func (s *UserService) verifyRequesterCanAccessUser(ctx context.Context, requesterID, targetUserID uint) error {
	if requesterID == targetUserID {
		return nil
	}

	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, requesterID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check superadmin status")
	}
	if isSuperAdmin {
		return nil
	}

	shares, err := s.store.SharesOrganization(ctx, requesterID, targetUserID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to check shared organization")
	}
	if !shares {
		return apperror.NotFound("user")
	}
	return nil
}
