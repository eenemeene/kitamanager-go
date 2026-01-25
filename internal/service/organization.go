package service

import (
	"context"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// OrganizationService handles business logic for organization operations
type OrganizationService struct {
	store      store.OrganizationStorer
	groupStore store.GroupStorer
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(store store.OrganizationStorer, groupStore store.GroupStorer) *OrganizationService {
	return &OrganizationService{store: store, groupStore: groupStore}
}

// List returns a paginated list of organizations
func (s *OrganizationService) List(ctx context.Context, limit, offset int) ([]models.Organization, int64, error) {
	orgs, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch organizations")
	}
	return orgs, total, nil
}

// GetByID returns an organization by ID
func (s *OrganizationService) GetByID(ctx context.Context, id uint) (*models.Organization, error) {
	org, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("organization")
	}
	return org, nil
}

// OrganizationCreateRequest represents the request for creating an organization
type OrganizationCreateRequest struct {
	Name   string
	Active bool
}

// Create creates a new organization with a default group
func (s *OrganizationService) Create(ctx context.Context, req *OrganizationCreateRequest, createdBy string) (*models.Organization, error) {
	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)

	if validation.IsWhitespaceOnly(req.Name) {
		return nil, apperror.BadRequest("name cannot be empty or whitespace only")
	}

	org := &models.Organization{
		Name:      req.Name,
		Active:    req.Active,
		CreatedBy: createdBy,
	}

	if err := s.store.Create(org); err != nil {
		return nil, apperror.Internal("failed to create organization")
	}

	// Create default group for the organization
	defaultGroup := &models.Group{
		Name:           "Members",
		OrganizationID: org.ID,
		IsDefault:      true,
		Active:         true,
		CreatedBy:      createdBy,
	}

	if err := s.groupStore.Create(defaultGroup); err != nil {
		return nil, apperror.Internal("failed to create default group")
	}

	return org, nil
}

// OrganizationUpdateRequest represents the request for updating an organization
type OrganizationUpdateRequest struct {
	Name   string
	Active *bool
}

// Update updates an existing organization
func (s *OrganizationService) Update(ctx context.Context, id uint, req *OrganizationUpdateRequest) (*models.Organization, error) {
	org, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("organization")
	}

	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)

	if req.Name != "" {
		if validation.IsWhitespaceOnly(req.Name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		org.Name = req.Name
	}
	if req.Active != nil {
		org.Active = *req.Active
	}

	if err := s.store.Update(org); err != nil {
		return nil, apperror.Internal("failed to update organization")
	}

	return org, nil
}

// Delete deletes an organization
func (s *OrganizationService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete organization")
	}
	return nil
}
