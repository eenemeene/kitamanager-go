package service

import (
	"context"
	"fmt"

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
	return personList(ctx, s.store.FindAll, (*models.Child).ToResponse, "children", limit, offset)
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
	return personGetByID(ctx, s.store.FindByIDAndOrg, (*models.Child).ToResponse, id, orgID, "child")
}

// Create creates a new child
func (s *ChildService) Create(ctx context.Context, orgID uint, req *models.ChildCreateRequest) (*models.ChildResponse, error) {
	return personCreate(ctx,
		&validation.PersonCreateFields{FirstName: req.FirstName, LastName: req.LastName, Gender: req.Gender, Birthdate: req.Birthdate},
		func(p models.Person) *models.Child { return &models.Child{Person: p} },
		s.store.Create, (*models.Child).ToResponse, orgID, "child")
}

// Update updates an existing child, validating it belongs to the specified organization
func (s *ChildService) Update(ctx context.Context, id, orgID uint, req *models.ChildUpdateRequest) (*models.ChildResponse, error) {
	return personUpdate(ctx, s.transactor, s.store.FindByIDAndOrg, func(ch *models.Child) *models.Person { return &ch.Person },
		s.store.Update, (*models.Child).ToResponse, id, orgID,
		personUpdateFields{FirstName: req.FirstName, LastName: req.LastName, Gender: req.Gender, Birthdate: req.Birthdate},
		"child")
}

// Delete deletes a child and its contracts, validating it belongs to the specified organization.
// The ownership check and deletion run in a single transaction.
func (s *ChildService) Delete(ctx context.Context, id, orgID uint) error {
	return personDelete(ctx, s.transactor, s.store.FindByIDMinimalAndOrg, s.store.Delete, id, orgID, "child")
}

// Import creates or updates children with their contracts from a ChildImportExportData.
// Children are matched by (first_name, last_name, birthdate, org_id).
// On match, existing contracts are replaced. Sections are auto-created if missing.
func (s *ChildService) Import(ctx context.Context, orgID uint, data *models.ChildImportExportData) ([]models.ChildResponse, error) {
	if len(data.Children) == 0 {
		return nil, apperror.BadRequest("no children in import data")
	}

	sectionCache := map[string]uint{}
	var results []models.ChildResponse

	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		for i, ch := range data.Children {
			childID, err := personImportUpsert(txCtx,
				importPersonItem{Index: i + 1, FirstName: ch.FirstName, LastName: ch.LastName, Gender: ch.Gender, Birthdate: ch.Birthdate},
				"child",
				s.store.FindByNameBirthdateAndOrg,
				func(c *models.Child) *models.Person { return &c.Person },
				func(c *models.Child) uint { return c.ID },
				s.store.Update, s.store.DeleteContractsByChild,
				func(p models.Person) *models.Child { return &models.Child{Person: p} },
				s.store.Create, orgID,
			)
			if err != nil {
				return err
			}

			for j, c := range ch.Contracts {
				if c.From.IsZero() {
					return apperror.BadRequest(fmt.Sprintf("child %d contract %d: from date is required", i+1, j+1))
				}

				sectionID, err := resolveSection(txCtx, s.sectionStore, c.SectionName, orgID, sectionCache)
				if err != nil {
					return err
				}

				req := &models.ChildContractCreateRequest{
					From:          c.From,
					To:            c.To,
					SectionID:     sectionID,
					VoucherNumber: c.VoucherNumber,
					Properties:    c.Properties,
				}
				if _, err := s.CreateContract(txCtx, childID, orgID, req); err != nil {
					return err
				}
			}

			fetched, err := s.store.FindByIDAndOrg(txCtx, childID, orgID)
			if err != nil {
				return apperror.InternalWrap(err, "failed to fetch imported child")
			}
			results = append(results, fetched.ToResponse())
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return results, nil
}

// FindAllByOrganization returns all children for an organization (no pagination), with contracts preloaded.
func (s *ChildService) FindAllByOrganization(ctx context.Context, orgID uint) ([]models.ChildResponse, error) {
	return fetchAllPaginated(ctx,
		func(ctx context.Context, limit, offset int) ([]models.Child, int64, error) {
			return s.store.FindByOrganizationAndSection(ctx, orgID, nil, nil, nil, "", limit, offset)
		},
		(*models.Child).ToResponse, "children")
}
