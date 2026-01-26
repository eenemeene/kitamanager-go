package service

import (
	"context"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// GroupService handles business logic for group operations
type GroupService struct {
	store store.GroupStorer
}

// NewGroupService creates a new group service
func NewGroupService(store store.GroupStorer) *GroupService {
	return &GroupService{store: store}
}

// List returns a paginated list of groups
func (s *GroupService) List(ctx context.Context, limit, offset int) ([]models.GroupResponse, int64, error) {
	groups, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch groups")
	}

	responses := make([]models.GroupResponse, len(groups))
	for i, group := range groups {
		responses[i] = group.ToResponse()
	}
	return responses, total, nil
}

// ListByOrganization returns a paginated list of groups for a specific organization
func (s *GroupService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.GroupResponse, int64, error) {
	groups, total, err := s.store.FindByOrganizationPaginated(orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch groups")
	}

	responses := make([]models.GroupResponse, len(groups))
	for i, group := range groups {
		responses[i] = group.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns a group by ID
func (s *GroupService) GetByID(ctx context.Context, id uint) (*models.GroupResponse, error) {
	group, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("group")
	}
	resp := group.ToResponse()
	return &resp, nil
}

// GetByIDAndOrg returns a group by ID if it belongs to the specified organization
func (s *GroupService) GetByIDAndOrg(ctx context.Context, id, orgID uint) (*models.GroupResponse, error) {
	group, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("group")
	}
	if group.OrganizationID != orgID {
		return nil, apperror.NotFound("group")
	}
	resp := group.ToResponse()
	return &resp, nil
}

// GroupCreateRequest represents the request for creating a group
type GroupCreateRequest struct {
	Name           string
	OrganizationID uint
	Active         bool
}

// Create creates a new group
func (s *GroupService) Create(ctx context.Context, req *GroupCreateRequest, createdBy string) (*models.GroupResponse, error) {
	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)

	if validation.IsWhitespaceOnly(req.Name) {
		return nil, apperror.BadRequest("name cannot be empty or whitespace only")
	}

	group := &models.Group{
		Name:           req.Name,
		OrganizationID: req.OrganizationID,
		Active:         req.Active,
		CreatedBy:      createdBy,
	}

	if err := s.store.Create(group); err != nil {
		return nil, apperror.Internal("failed to create group")
	}

	resp := group.ToResponse()
	return &resp, nil
}

// GroupUpdateRequest represents the request for updating a group
type GroupUpdateRequest struct {
	Name   string
	Active *bool
}

// Update updates an existing group
func (s *GroupService) Update(ctx context.Context, id uint, req *GroupUpdateRequest) (*models.GroupResponse, error) {
	group, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("group")
	}

	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)

	if req.Name != "" {
		if validation.IsWhitespaceOnly(req.Name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		group.Name = req.Name
	}
	if req.Active != nil {
		group.Active = *req.Active
	}

	if err := s.store.Update(group); err != nil {
		return nil, apperror.Internal("failed to update group")
	}

	resp := group.ToResponse()
	return &resp, nil
}

// UpdateByIDAndOrg updates a group if it belongs to the specified organization
func (s *GroupService) UpdateByIDAndOrg(ctx context.Context, id, orgID uint, req *GroupUpdateRequest) (*models.GroupResponse, error) {
	group, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("group")
	}
	// Security: Validate group belongs to the specified organization
	if group.OrganizationID != orgID {
		return nil, apperror.NotFound("group")
	}

	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)

	if req.Name != "" {
		if validation.IsWhitespaceOnly(req.Name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		group.Name = req.Name
	}
	if req.Active != nil {
		group.Active = *req.Active
	}

	if err := s.store.Update(group); err != nil {
		return nil, apperror.Internal("failed to update group")
	}

	resp := group.ToResponse()
	return &resp, nil
}

// Delete deletes a group
func (s *GroupService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete group")
	}
	return nil
}

// DeleteByIDAndOrg deletes a group if it belongs to the specified organization
func (s *GroupService) DeleteByIDAndOrg(ctx context.Context, id, orgID uint) error {
	group, err := s.store.FindByID(id)
	if err != nil {
		return apperror.NotFound("group")
	}
	// Security: Validate group belongs to the specified organization
	if group.OrganizationID != orgID {
		return apperror.NotFound("group")
	}

	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete group")
	}
	return nil
}
