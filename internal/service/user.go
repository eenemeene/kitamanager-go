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
func (s *UserService) List(ctx context.Context, limit, offset int) ([]models.UserResponse, int64, error) {
	users, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch users")
	}

	responses := make([]models.UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns a user by ID
func (s *UserService) GetByID(ctx context.Context, id uint) (*models.UserResponse, error) {
	user, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("user")
	}
	resp := user.ToResponse()
	return &resp, nil
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, req *models.UserCreate, createdBy string) (*models.UserResponse, error) {
	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if validation.IsWhitespaceOnly(req.Name) {
		return nil, apperror.BadRequest("name cannot be empty or whitespace only")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.Internal("failed to hash password")
	}

	user := &models.User{
		Name:      req.Name,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Active:    req.Active,
		CreatedBy: createdBy,
	}

	if err := s.store.Create(user); err != nil {
		return nil, apperror.Internal("failed to create user")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// Update updates an existing user
func (s *UserService) Update(ctx context.Context, id uint, req *models.UserUpdate) (*models.UserResponse, error) {
	user, err := s.store.FindByID(id)
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
		user.Email = req.Email
	}
	if req.Active != nil {
		user.Active = *req.Active
	}

	if err := s.store.Update(user); err != nil {
		return nil, apperror.Internal("failed to update user")
	}

	resp := user.ToResponse()
	return &resp, nil
}

// Delete deletes a user
func (s *UserService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete user")
	}
	return nil
}

// AddToGroup adds a user to a group
func (s *UserService) AddToGroup(ctx context.Context, userID, groupID uint) error {
	_, err := s.store.FindByID(userID)
	if err != nil {
		return apperror.NotFound("user")
	}

	_, err = s.groupStore.FindByID(groupID)
	if err != nil {
		return apperror.NotFound("group")
	}

	if err := s.store.AddToGroup(userID, groupID); err != nil {
		return apperror.Internal("failed to add user to group")
	}
	return nil
}

// RemoveFromGroup removes a user from a group
func (s *UserService) RemoveFromGroup(ctx context.Context, userID, groupID uint) error {
	if err := s.store.RemoveFromGroup(userID, groupID); err != nil {
		return apperror.Internal("failed to remove user from group")
	}
	return nil
}

// AddToOrganization adds a user to an organization by adding them to the default group
func (s *UserService) AddToOrganization(ctx context.Context, userID, orgID uint) error {
	_, err := s.store.FindByID(userID)
	if err != nil {
		return apperror.NotFound("user")
	}

	// Find the default group for the organization
	defaultGroup, err := s.groupStore.FindDefaultGroup(orgID)
	if err != nil {
		return apperror.NotFound("organization or default group")
	}

	if err := s.store.AddToGroup(userID, defaultGroup.ID); err != nil {
		return apperror.Internal("failed to add user to organization")
	}
	return nil
}

// RemoveFromOrganization removes a user from an organization by removing them from all groups in that org
func (s *UserService) RemoveFromOrganization(ctx context.Context, userID, orgID uint) error {
	_, err := s.store.FindByID(userID)
	if err != nil {
		return apperror.NotFound("user")
	}

	if err := s.store.RemoveFromAllGroupsInOrg(userID, orgID); err != nil {
		return apperror.Internal("failed to remove user from organization")
	}
	return nil
}
