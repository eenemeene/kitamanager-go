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

// ListByOrganization returns a paginated list of groups for a specific organization
func (s *GroupService) ListByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.GroupResponse, int64, error) {
	groups, total, err := s.store.FindByOrganizationPaginated(ctx, orgID, search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch groups")
	}

	return toResponseList(groups, (*models.Group).ToResponse), total, nil
}

// GetByIDAndOrg returns a group by ID if it belongs to the specified organization
func (s *GroupService) GetByIDAndOrg(ctx context.Context, id, orgID uint) (*models.GroupResponse, error) {
	group, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("group")
	}
	if err := verifyOrgOwnership(group, orgID, "group"); err != nil {
		return nil, err
	}
	resp := group.ToResponse()
	return &resp, nil
}

// Create creates a new group
func (s *GroupService) Create(ctx context.Context, orgID uint, req *models.GroupCreateRequest, createdBy string) (*models.GroupResponse, error) {
	name, err := validateRequiredName(req.Name)
	if err != nil {
		return nil, err
	}

	group := &models.Group{
		Name:           name,
		OrganizationID: orgID,
		Active:         req.Active,
		CreatedBy:      createdBy,
	}

	if err := s.store.Create(ctx, group); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create group")
	}

	resp := group.ToResponse()
	return &resp, nil
}

// UpdateByIDAndOrg updates a group if it belongs to the specified organization
func (s *GroupService) UpdateByIDAndOrg(ctx context.Context, id, orgID uint, req *models.GroupUpdateRequest) (*models.GroupResponse, error) {
	group, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("group")
	}
	if err := verifyOrgOwnership(group, orgID, "group"); err != nil {
		return nil, err
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

	if err := s.store.Update(ctx, group); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update group")
	}

	resp := group.ToResponse()
	return &resp, nil
}

// DeleteByIDAndOrg deletes a group if it belongs to the specified organization
func (s *GroupService) DeleteByIDAndOrg(ctx context.Context, id, orgID uint) error {
	group, err := s.store.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("group")
	}
	if err := verifyOrgOwnership(group, orgID, "group"); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return apperror.InternalWrap(err, "failed to delete group")
	}
	return nil
}
