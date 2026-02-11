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
	userStore  store.UserStorer
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(store store.OrganizationStorer, groupStore store.GroupStorer, userStore store.UserStorer) *OrganizationService {
	return &OrganizationService{store: store, groupStore: groupStore, userStore: userStore}
}

// ListForUser returns a paginated list of organizations the user has access to.
// Superadmins see all organizations; other users see only organizations they belong to.
func (s *OrganizationService) ListForUser(ctx context.Context, userID uint, search string, limit, offset int) ([]models.OrganizationResponse, int64, error) {
	// Check if user is superadmin
	user, err := s.userStore.FindByID(userID)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch user")
	}

	if user.IsSuperAdmin {
		// Superadmins see all organizations
		return s.List(ctx, search, limit, offset)
	}

	// Regular users only see organizations they belong to via group membership
	orgs, err := s.userStore.GetUserOrganizations(userID)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch user organizations")
	}

	// Apply search filter in memory (users typically belong to 1-3 orgs)
	if search != "" {
		searchLower := strings.ToLower(search)
		filtered := orgs[:0]
		for _, org := range orgs {
			if strings.Contains(strings.ToLower(org.Name), searchLower) {
				filtered = append(filtered, org)
			}
		}
		orgs = filtered
	}

	total := int64(len(orgs))

	// Apply pagination manually
	start := offset
	if start > len(orgs) {
		start = len(orgs)
	}
	end := start + limit
	if end > len(orgs) {
		end = len(orgs)
	}
	pagedOrgs := orgs[start:end]

	responses := make([]models.OrganizationResponse, len(pagedOrgs))
	for i, org := range pagedOrgs {
		responses[i] = org.ToResponse()
	}
	return responses, total, nil
}

// List returns a paginated list of all organizations (for internal use)
func (s *OrganizationService) List(ctx context.Context, search string, limit, offset int) ([]models.OrganizationResponse, int64, error) {
	orgs, total, err := s.store.FindAll(search, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch organizations")
	}

	responses := make([]models.OrganizationResponse, len(orgs))
	for i, org := range orgs {
		responses[i] = org.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns an organization by ID
func (s *OrganizationService) GetByID(ctx context.Context, id uint) (*models.OrganizationResponse, error) {
	org, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("organization")
	}
	resp := org.ToResponse()
	return &resp, nil
}

// OrganizationCreateRequest represents the request for creating an organization
type OrganizationCreateRequest struct {
	Name   string
	Active bool
	State  string
}

// Create creates a new organization with a default group (transactional)
func (s *OrganizationService) Create(ctx context.Context, req *OrganizationCreateRequest, createdBy string) (*models.OrganizationResponse, error) {
	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)

	if validation.IsWhitespaceOnly(req.Name) {
		return nil, apperror.BadRequest("name cannot be empty or whitespace only")
	}

	if !models.IsValidState(req.State) {
		return nil, apperror.BadRequest("invalid state, must be one of: berlin")
	}

	org := &models.Organization{
		Name:      req.Name,
		Active:    req.Active,
		State:     req.State,
		CreatedBy: createdBy,
	}

	// Create default group for the organization
	defaultGroup := &models.Group{
		Name:      "Members",
		IsDefault: true,
		Active:    true,
		CreatedBy: createdBy,
	}

	// Create organization and default group in a single transaction
	if err := s.store.CreateWithDefaultGroup(org, defaultGroup); err != nil {
		return nil, apperror.Internal("failed to create organization")
	}

	resp := org.ToResponse()
	return &resp, nil
}

// OrganizationUpdateRequest represents the request for updating an organization
type OrganizationUpdateRequest struct {
	Name   string
	Active *bool
	State  *string
}

// Update updates an existing organization
func (s *OrganizationService) Update(ctx context.Context, id uint, req *OrganizationUpdateRequest) (*models.OrganizationResponse, error) {
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
	if req.State != nil {
		if !models.IsValidState(*req.State) {
			return nil, apperror.BadRequest("invalid state, must be one of: berlin")
		}
		org.State = *req.State
	}

	if err := s.store.Update(org); err != nil {
		return nil, apperror.Internal("failed to update organization")
	}

	resp := org.ToResponse()
	return &resp, nil
}

// Delete deletes an organization
func (s *OrganizationService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete organization")
	}
	return nil
}
