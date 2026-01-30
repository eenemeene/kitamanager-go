package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// SectionService handles business logic for section operations
type SectionService struct {
	store store.SectionStorer
}

// NewSectionService creates a new section service
func NewSectionService(store store.SectionStorer) *SectionService {
	return &SectionService{store: store}
}

// ListByOrganization returns a paginated list of sections for a specific organization
func (s *SectionService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.SectionResponse, int64, error) {
	sections, total, err := s.store.FindByOrganizationPaginated(orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch sections")
	}

	responses := make([]models.SectionResponse, len(sections))
	for i, section := range sections {
		responses[i] = section.ToResponse()
	}
	return responses, total, nil
}

// GetByIDAndOrg returns a section by ID if it belongs to the specified organization
func (s *SectionService) GetByIDAndOrg(ctx context.Context, id, orgID uint) (*models.SectionResponse, error) {
	section, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("section")
	}
	if section.OrganizationID != orgID {
		return nil, apperror.NotFound("section")
	}
	resp := section.ToResponse()
	return &resp, nil
}

// Create creates a new section
func (s *SectionService) Create(ctx context.Context, orgID uint, req *models.SectionCreateRequest, createdBy string) (*models.SectionResponse, error) {
	// Trim and validate input
	name := strings.TrimSpace(req.Name)

	if validation.IsWhitespaceOnly(name) {
		return nil, apperror.BadRequest("name cannot be empty or whitespace only")
	}

	// Check for duplicate name in organization
	existing, err := s.store.FindByNameAndOrg(name, orgID)
	if err == nil && existing != nil {
		return nil, apperror.BadRequest("section with this name already exists in the organization")
	}

	section := &models.Section{
		Name:           name,
		OrganizationID: orgID,
		CreatedBy:      createdBy,
	}

	if err := s.store.Create(section); err != nil {
		return nil, apperror.Internal("failed to create section")
	}

	resp := section.ToResponse()
	return &resp, nil
}

// UpdateByIDAndOrg updates a section if it belongs to the specified organization
func (s *SectionService) UpdateByIDAndOrg(ctx context.Context, id, orgID uint, req *models.SectionUpdateRequest) (*models.SectionResponse, error) {
	section, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("section")
	}
	// Security: Validate section belongs to the specified organization
	if section.OrganizationID != orgID {
		return nil, apperror.NotFound("section")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if validation.IsWhitespaceOnly(name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}

		// Check for duplicate name in organization (excluding current section)
		existing, err := s.store.FindByNameAndOrg(name, orgID)
		if err == nil && existing != nil && existing.ID != id {
			return nil, apperror.BadRequest("section with this name already exists in the organization")
		}

		section.Name = name
	}

	if err := s.store.Update(section); err != nil {
		return nil, apperror.Internal("failed to update section")
	}

	resp := section.ToResponse()
	return &resp, nil
}

// DeleteByIDAndOrg deletes a section if it belongs to the specified organization
func (s *SectionService) DeleteByIDAndOrg(ctx context.Context, id, orgID uint) error {
	section, err := s.store.FindByID(id)
	if err != nil {
		return apperror.NotFound("section")
	}
	// Security: Validate section belongs to the specified organization
	if section.OrganizationID != orgID {
		return apperror.NotFound("section")
	}

	// Prevent deletion of default section
	if section.IsDefault {
		return apperror.BadRequest("cannot delete the default section")
	}

	// Check if section has children
	hasChildren, err := s.store.HasChildren(id)
	if err != nil {
		return apperror.Internal("failed to check section children")
	}
	if hasChildren {
		return apperror.BadRequest("cannot delete section with assigned children")
	}

	// Check if section has employees
	hasEmployees, err := s.store.HasEmployees(id)
	if err != nil {
		return apperror.Internal("failed to check section employees")
	}
	if hasEmployees {
		return apperror.BadRequest("cannot delete section with assigned employees")
	}

	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete section")
	}
	return nil
}

// GetOrCreateDefaultSection gets or creates the default "Unassigned" section for an organization
func (s *SectionService) GetOrCreateDefaultSection(ctx context.Context, orgID uint, createdBy string) (*models.Section, error) {
	// Try to find existing default section
	section, err := s.store.FindDefaultSection(orgID)
	if err == nil {
		return section, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Internal("failed to check for default section")
	}

	// Create default section
	section = &models.Section{
		Name:           "Unassigned",
		OrganizationID: orgID,
		IsDefault:      true,
		CreatedBy:      createdBy,
	}

	if err := s.store.Create(section); err != nil {
		return nil, apperror.Internal("failed to create default section")
	}

	return section, nil
}
