package service

import (
	"context"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// validateAgeRange validates the age range fields.
// Both nil is ok, but if either is set: values must be non-negative and min < max.
func validateAgeRange(min, max *int) error {
	if min == nil && max == nil {
		return nil
	}
	if min != nil && *min < 0 {
		return apperror.BadRequest("min_age_months cannot be negative")
	}
	if max != nil && *max < 0 {
		return apperror.BadRequest("max_age_months cannot be negative")
	}
	if min != nil && max != nil && *min >= *max {
		return apperror.BadRequest("min_age_months must be less than max_age_months")
	}
	return nil
}

// SectionService handles business logic for section operations
type SectionService struct {
	store      store.SectionStorer
	transactor store.Transactor
}

// NewSectionService creates a new section service
func NewSectionService(store store.SectionStorer, transactor store.Transactor) *SectionService {
	return &SectionService{store: store, transactor: transactor}
}

// ListByOrganization returns a paginated list of sections for a specific organization
func (s *SectionService) ListByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.SectionResponse, int64, error) {
	sections, total, err := s.store.FindByOrganizationPaginated(ctx, orgID, search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch sections")
	}

	return toResponseList(sections, (*models.Section).ToResponse), total, nil
}

// GetByIDAndOrg returns a section by ID if it belongs to the specified organization
func (s *SectionService) GetByIDAndOrg(ctx context.Context, id, orgID uint) (*models.SectionResponse, error) {
	section, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "section")
	}
	if err := verifyOrgOwnership(section, orgID, "section"); err != nil {
		return nil, err
	}
	resp := section.ToResponse()
	return &resp, nil
}

// Create creates a new section
func (s *SectionService) Create(ctx context.Context, orgID uint, req *models.SectionCreateRequest, createdBy string) (*models.SectionResponse, error) {
	name, err := validateRequiredName(req.Name)
	if err != nil {
		return nil, err
	}

	// Validate age range
	if err := validateAgeRange(req.MinAgeMonths, req.MaxAgeMonths); err != nil {
		return nil, err
	}

	// Check for duplicate name in organization
	existing, err := s.store.FindByNameAndOrg(ctx, name, orgID)
	if err == nil && existing != nil {
		return nil, apperror.Conflict("section with this name already exists in the organization")
	}

	section := &models.Section{
		Name:           name,
		OrganizationID: orgID,
		CreatedBy:      createdBy,
		MinAgeMonths:   req.MinAgeMonths,
		MaxAgeMonths:   req.MaxAgeMonths,
	}

	if err := s.store.Create(ctx, section); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create section")
	}

	resp := section.ToResponse()
	return &resp, nil
}

// UpdateByIDAndOrg updates a section if it belongs to the specified organization
func (s *SectionService) UpdateByIDAndOrg(ctx context.Context, id, orgID uint, req *models.SectionUpdateRequest) (*models.SectionResponse, error) {
	section, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, classifyStoreError(err, "section")
	}
	if err := verifyOrgOwnership(section, orgID, "section"); err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if validation.IsWhitespaceOnly(name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}

		// Check for duplicate name in organization (excluding current section)
		existing, err := s.store.FindByNameAndOrg(ctx, name, orgID)
		if err == nil && existing != nil && existing.ID != id {
			return nil, apperror.Conflict("section with this name already exists in the organization")
		}

		section.Name = name
	}

	// Always update age fields — the frontend always sends them.
	// null means "clear the value", which is distinct from "not provided".
	section.MinAgeMonths = req.MinAgeMonths
	section.MaxAgeMonths = req.MaxAgeMonths

	// Validate the resulting age range
	if err := validateAgeRange(section.MinAgeMonths, section.MaxAgeMonths); err != nil {
		return nil, err
	}

	if err := s.store.Update(ctx, section); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update section")
	}

	resp := section.ToResponse()
	return &resp, nil
}

// DeleteByIDAndOrg deletes a section if it belongs to the specified organization.
// The check-then-delete sequence runs inside a transaction to prevent TOCTOU races.
func (s *SectionService) DeleteByIDAndOrg(ctx context.Context, id, orgID uint) error {
	return s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		section, err := s.store.FindByID(txCtx, id)
		if err != nil {
			return classifyStoreError(err, "section")
		}
		if err := verifyOrgOwnership(section, orgID, "section"); err != nil {
			return err
		}

		// Prevent deletion of default section
		if section.IsDefault {
			return apperror.BadRequest("cannot delete the default section")
		}

		// Check if section has children
		hasChildren, err := s.store.HasChildren(txCtx, id)
		if err != nil {
			return apperror.InternalWrap(err, "failed to check section children")
		}
		if hasChildren {
			return apperror.BadRequest("cannot delete section with assigned children")
		}

		// Check if section has employees
		hasEmployees, err := s.store.HasEmployees(txCtx, id)
		if err != nil {
			return apperror.InternalWrap(err, "failed to check section employees")
		}
		if hasEmployees {
			return apperror.BadRequest("cannot delete section with assigned employees")
		}

		if err := s.store.Delete(txCtx, id); err != nil {
			return apperror.InternalWrap(err, "failed to delete section")
		}
		return nil
	})
}
