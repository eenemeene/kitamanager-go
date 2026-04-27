package service

import (
	"context"
	"strings"
	"time"

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
		if store.IsDuplicateKeyError(err) {
			return nil, apperror.Conflict("section with this name already exists in this organization")
		}
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
		// Mirror Create's TOCTOU fallback. Two requests racing to
		// rename two different sections to the same target name can
		// both pass the FindByNameAndOrg pre-check above; the unique
		// index then catches the second one. Without this, that
		// second request would surface as a 500 (InternalWrap). With
		// it, both Create and Update consistently return Conflict
		// for any duplicate-name path.
		if store.IsDuplicateKeyError(err) {
			return nil, apperror.Conflict("section with this name already exists in the organization")
		}
		return nil, apperror.InternalWrap(err, "failed to update section")
	}

	resp := section.ToResponse()
	return &resp, nil
}

// PromoteToDefault sets the given section as the org's default,
// clearing the flag from any existing default. Wraps the two-step
// store operation in a transaction so the partial-unique-index
// invariant from migration 000019 holds at every statement boundary.
//
// Cross-org calls (sectionID belongs to another org) return NotFound
// rather than Forbidden, mirroring the pattern of GetByIDAndOrg /
// UpdateByIDAndOrg — avoids leaking section existence across orgs.
//
// No-op when the section is already the default; the transaction
// still runs but both UPDATEs touch the same row (clear→set→clear→
// set is the same final state).
func (s *SectionService) PromoteToDefault(ctx context.Context, id, orgID uint) error {
	return s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		section, err := s.store.FindByID(txCtx, id)
		if err != nil {
			return classifyStoreError(err, "section")
		}
		if err := verifyOrgOwnership(section, orgID, "section"); err != nil {
			return err
		}
		if err := s.store.PromoteToDefault(txCtx, id, orgID); err != nil {
			return classifyStoreError(err, "section")
		}
		return nil
	})
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

		// Block deletion only when CURRENT contracts still reference
		// the section. Historical / ended contracts are fine — the
		// section row physically lives on (gorm soft-delete just sets
		// deleted_at), so their FK keeps resolving even after the
		// tombstone. Without the time filter, an org that reorganised
		// its sections years ago would have permanent "zombie"
		// sections it could never clean up.
		now := time.Now().UTC()
		hasChildren, err := s.store.HasActiveChildren(txCtx, id, now)
		if err != nil {
			return apperror.InternalWrap(err, "failed to check section children")
		}
		if hasChildren {
			return apperror.BadRequest("cannot delete section with currently-assigned children")
		}

		hasEmployees, err := s.store.HasActiveEmployees(txCtx, id, now)
		if err != nil {
			return apperror.InternalWrap(err, "failed to check section employees")
		}
		if hasEmployees {
			return apperror.BadRequest("cannot delete section with currently-assigned employees")
		}

		// Soft-delete via gorm's DeletedAt machinery. Section.Delete
		// stamps deleted_at; subsequent FindByID auto-scopes the row
		// out (so List / pickers stop showing it) but historical
		// contracts under it keep resolving via FK because the row
		// is still physically present.
		if err := s.store.Delete(txCtx, id); err != nil {
			return apperror.InternalWrap(err, "failed to delete section")
		}
		return nil
	})
}
