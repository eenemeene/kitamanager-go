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
	store     store.OrganizationStorer
	userStore store.UserStorer
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(store store.OrganizationStorer, userStore store.UserStorer) *OrganizationService {
	return &OrganizationService{store: store, userStore: userStore}
}

// ListForUser returns a paginated list of organizations the user has access to.
// Superadmins see all organizations; other users see only organizations they belong to.
func (s *OrganizationService) ListForUser(ctx context.Context, userID uint, search string, limit, offset int) ([]models.OrganizationResponse, int64, error) {
	// Check if user is superadmin
	user, err := s.userStore.FindByID(ctx, userID)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch user")
	}

	if user.IsSuperAdmin {
		// Superadmins see all organizations
		return s.List(ctx, search, limit, offset)
	}

	// Regular users only see organizations they belong to via user_organizations
	orgs, err := s.userStore.GetUserOrganizations(ctx, userID)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch user organizations")
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

	return toResponseList(pagedOrgs, (*models.Organization).ToResponse), total, nil
}

// List returns a paginated list of all organizations (for internal use)
func (s *OrganizationService) List(ctx context.Context, search string, limit, offset int) ([]models.OrganizationResponse, int64, error) {
	orgs, total, err := s.store.FindAll(ctx, search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch organizations")
	}

	return toResponseList(orgs, (*models.Organization).ToResponse), total, nil
}

// GetByID returns an organization by ID
func (s *OrganizationService) GetByID(ctx context.Context, id uint) (*models.OrganizationResponse, error) {
	org, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "organization")
	}
	resp := org.ToResponse()
	return &resp, nil
}

// Create creates a new organization with a default section (transactional)
func (s *OrganizationService) Create(ctx context.Context, req *models.OrganizationCreateRequest, createdBy string) (*models.OrganizationResponse, error) {
	name, err := validateRequiredName(req.Name)
	if err != nil {
		return nil, err
	}

	if !models.IsValidState(req.State) {
		return nil, apperror.BadRequest("invalid state, must be one of: " + models.ValidStatesString())
	}

	sectionName, err := validateRequiredName(req.DefaultSectionName)
	if err != nil {
		return nil, apperror.BadRequest("default_section_name is required")
	}

	org := &models.Organization{
		Name:      name,
		Active:    req.Active,
		State:     req.State,
		CreatedBy: createdBy,
	}

	// Create default section for the organization
	defaultSection := &models.Section{
		Name:      sectionName,
		IsDefault: true,
		CreatedBy: createdBy,
	}

	// Create organization and default section in a single transaction
	if err := s.store.CreateWithDefaultSection(ctx, org, defaultSection); err != nil {
		if store.IsDuplicateKeyError(err) {
			return nil, apperror.Conflict("organization with this name already exists")
		}
		return nil, apperror.InternalWrap(err, "failed to create organization")
	}

	resp := org.ToResponse()
	return &resp, nil
}

// Update updates an existing organization
func (s *OrganizationService) Update(ctx context.Context, id uint, req *models.OrganizationUpdateRequest) (*models.OrganizationResponse, error) {
	org, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "organization")
	}

	// Trim and validate input
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		org.Name = trimmed
	}
	if req.Active != nil {
		org.Active = *req.Active
	}
	if req.State != nil {
		if !models.IsValidState(*req.State) {
			return nil, apperror.BadRequest("invalid state, must be one of: " + models.ValidStatesString())
		}
		org.State = *req.State
	}

	if err := s.store.Update(ctx, org); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update organization")
	}

	resp := org.ToResponse()
	return &resp, nil
}

// Delete deletes an organization
func (s *OrganizationService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return apperror.InternalWrap(err, "failed to delete organization")
	}
	return nil
}
