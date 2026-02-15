package service

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// UserService handles business logic for user operations
type UserService struct {
	store      store.UserStorer
	groupStore store.GroupStorer
}

// NewUserService creates a new user service
func NewUserService(store store.UserStorer, groupStore store.GroupStorer) *UserService {
	return &UserService{store: store, groupStore: groupStore}
}

// List returns a paginated list of users
func (s *UserService) List(ctx context.Context, search string, limit, offset int) ([]models.UserResponse, int64, error) {
	users, total, err := s.store.FindAll(ctx, search, limit, offset)
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

// GetByID returns a user by ID
func (s *UserService) GetByID(ctx context.Context, id uint) (*models.UserResponse, error) {
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
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
		return nil, apperror.InternalWrap(err, "failed to create user")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// Update updates an existing user
func (s *UserService) Update(ctx context.Context, id uint, req *models.UserUpdateRequest) (*models.UserResponse, error) {
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("user")
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
	if req.Active != nil {
		user.Active = *req.Active
	}

	if err := s.store.Update(ctx, user); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update user")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// ResetPassword sets a new password for a user (admin-initiated).
func (s *UserService) ResetPassword(ctx context.Context, userID uint, newPassword string) error {
	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("user")
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

// Delete deletes a user
func (s *UserService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return apperror.InternalWrap(err, "failed to delete user")
	}
	return nil
}

// AddToGroup adds a user to a group
func (s *UserService) AddToGroup(ctx context.Context, userID, groupID uint) error {
	_, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("user")
	}

	_, err = s.groupStore.FindByID(ctx, groupID)
	if err != nil {
		return apperror.NotFound("group")
	}

	if err := s.store.AddToGroup(ctx, userID, groupID); err != nil {
		return apperror.InternalWrap(err, "failed to add user to group")
	}
	return nil
}

// RemoveFromGroup removes a user from a group
func (s *UserService) RemoveFromGroup(ctx context.Context, userID, groupID uint) error {
	if err := s.store.RemoveFromGroup(ctx, userID, groupID); err != nil {
		return apperror.InternalWrap(err, "failed to remove user from group")
	}
	return nil
}

// AddToOrganization adds a user to an organization by adding them to the default group
func (s *UserService) AddToOrganization(ctx context.Context, userID, orgID uint) error {
	_, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("user")
	}

	// Find the default group for the organization
	defaultGroup, err := s.groupStore.FindDefaultGroup(ctx, orgID)
	if err != nil {
		return apperror.NotFound("organization or default group")
	}

	if err := s.store.AddToGroup(ctx, userID, defaultGroup.ID); err != nil {
		return apperror.InternalWrap(err, "failed to add user to organization")
	}
	return nil
}

// RemoveFromOrganization removes a user from an organization by removing them from all groups in that org
func (s *UserService) RemoveFromOrganization(ctx context.Context, userID, orgID uint) error {
	_, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("user")
	}

	if err := s.store.RemoveFromAllGroupsInOrg(ctx, userID, orgID); err != nil {
		return apperror.InternalWrap(err, "failed to remove user from organization")
	}
	return nil
}
