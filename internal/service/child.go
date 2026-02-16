package service

import (
	"context"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// ChildService handles business logic for child operations
type ChildService struct {
	store        store.ChildStorer
	orgStore     store.OrganizationStorer
	fundingStore store.GovernmentFundingStorer
	sectionStore store.SectionStorer
	transactor   store.Transactor
}

// NewChildService creates a new child service
func NewChildService(store store.ChildStorer, orgStore store.OrganizationStorer, fundingStore store.GovernmentFundingStorer, sectionStore store.SectionStorer, transactor store.Transactor) *ChildService {
	return &ChildService{
		store:        store,
		orgStore:     orgStore,
		fundingStore: fundingStore,
		sectionStore: sectionStore,
		transactor:   transactor,
	}
}

// List returns a paginated list of children
func (s *ChildService) List(ctx context.Context, limit, offset int) ([]models.ChildResponse, int64, error) {
	children, total, err := s.store.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch children")
	}

	return toResponseList(children, (*models.Child).ToResponse), total, nil
}

// ListByOrganization returns a paginated list of children for an organization
func (s *ChildService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.ChildResponse, int64, error) {
	return s.ListByOrganizationAndSection(ctx, orgID, models.ChildListFilter{}, limit, offset)
}

// ListByOrganizationAndSection returns a paginated list of children for an organization,
// optionally filtered by section, active contract date, contract-after date, and/or name search.
func (s *ChildService) ListByOrganizationAndSection(ctx context.Context, orgID uint, filter models.ChildListFilter, limit, offset int) ([]models.ChildResponse, int64, error) {
	if err := filter.Validate(); err != nil {
		return nil, 0, apperror.BadRequest(err.Error())
	}

	children, total, err := s.store.FindByOrganizationAndSection(ctx, orgID, filter.SectionID, filter.ActiveOn, filter.ContractAfter, filter.Search, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch children")
	}

	return toResponseList(children, (*models.Child).ToResponse), total, nil
}

// GetByID returns a child by ID, validating it belongs to the specified organization
func (s *ChildService) GetByID(ctx context.Context, id, orgID uint) (*models.ChildResponse, error) {
	child, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}
	resp := child.ToResponse()
	return &resp, nil
}

// Create creates a new child
func (s *ChildService) Create(ctx context.Context, orgID uint, req *models.ChildCreateRequest) (*models.ChildResponse, error) {
	// Trim and validate input
	person, err := validation.ValidatePersonCreate(&validation.PersonCreateFields{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Gender:    req.Gender,
		Birthdate: req.Birthdate,
	})
	if err != nil {
		return nil, err
	}

	child := &models.Child{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      person.FirstName,
			LastName:       person.LastName,
			Gender:         person.Gender,
			Birthdate:      person.Birthdate,
		},
	}

	if err := s.store.Create(ctx, child); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create child")
	}

	resp := child.ToResponse()
	return &resp, nil
}

// Update updates an existing child, validating it belongs to the specified organization
func (s *ChildService) Update(ctx context.Context, id, orgID uint, req *models.ChildUpdateRequest) (*models.ChildResponse, error) {
	child, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
		return nil, err
	}

	if req.FirstName != nil {
		trimmed, err := validation.ValidateAndTrimName(*req.FirstName, "first_name")
		if err != nil {
			return nil, err
		}
		child.FirstName = trimmed
	}
	if req.LastName != nil {
		trimmed, err := validation.ValidateAndTrimName(*req.LastName, "last_name")
		if err != nil {
			return nil, err
		}
		child.LastName = trimmed
	}
	if req.Gender != nil {
		if err := validation.ValidateGender(*req.Gender); err != nil {
			return nil, err
		}
		child.Gender = *req.Gender
	}
	if req.Birthdate != nil {
		bd, err := validation.ParseAndValidateBirthdate(*req.Birthdate)
		if err != nil {
			return nil, err
		}
		child.Birthdate = bd
	}

	if err := s.store.Update(ctx, child); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update child")
	}

	// Reload to get fresh associations
	child, err = s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to reload child after update")
	}

	resp := child.ToResponse()
	return &resp, nil
}

// Delete deletes a child and its contracts, validating it belongs to the specified organization.
// The ownership check and deletion run in a single transaction.
func (s *ChildService) Delete(ctx context.Context, id, orgID uint) error {
	return s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		// Security: Validate child belongs to the specified organization (use minimal query - no preloads needed)
		child, err := s.store.FindByIDMinimal(txCtx, id)
		if err != nil {
			return apperror.NotFound("child")
		}
		if err := verifyOrgOwnership(child, orgID, "child"); err != nil {
			return err
		}

		if err := s.store.Delete(txCtx, id); err != nil {
			return apperror.InternalWrap(err, "failed to delete child")
		}
		return nil
	})
}
