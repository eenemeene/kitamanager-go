package service

import (
	"context"
	"errors"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// CostService handles business logic for costs and cost entries.
type CostService struct {
	store      store.CostStorer
	transactor store.Transactor
}

// NewCostService creates a new CostService.
func NewCostService(store store.CostStorer, transactor store.Transactor) *CostService {
	return &CostService{store: store, transactor: transactor}
}

// verifyCostOwnership verifies a cost exists and belongs to the organization.
func (s *CostService) verifyCostOwnership(ctx context.Context, costID, orgID uint) (*models.Cost, error) {
	cost, err := s.store.FindByID(ctx, costID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("cost")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch cost")
	}
	if cost.OrganizationID != orgID {
		return nil, apperror.NotFound("cost")
	}
	return cost, nil
}

// verifyEntryOwnership verifies a cost entry exists and belongs to the cost.
func (s *CostService) verifyEntryOwnership(ctx context.Context, entryID, costID uint) (*models.CostEntry, error) {
	entry, err := s.store.FindEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("cost entry")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch cost entry")
	}
	if entry.CostID != costID {
		return nil, apperror.NotFound("cost entry")
	}
	return entry, nil
}

// Cost CRUD

// Create creates a new cost.
func (s *CostService) Create(ctx context.Context, orgID uint, req *models.CostCreateRequest) (*models.CostResponse, error) {
	name, err := validateRequiredName(req.Name)
	if err != nil {
		return nil, err
	}

	cost := &models.Cost{
		OrganizationID: orgID,
		Name:           name,
	}

	if err := s.store.Create(ctx, cost); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create cost")
	}

	resp := cost.ToResponse()
	return &resp, nil
}

// GetByID retrieves a cost by ID with all entries.
func (s *CostService) GetByID(ctx context.Context, id, orgID uint) (*models.CostDetailResponse, error) {
	cost, err := s.store.FindByIDWithEntries(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("cost")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch cost")
	}

	if cost.OrganizationID != orgID {
		return nil, apperror.NotFound("cost")
	}

	resp := cost.ToDetailResponse()
	return &resp, nil
}

// List retrieves all costs for an organization.
func (s *CostService) List(ctx context.Context, orgID uint, limit, offset int) ([]models.CostResponse, int64, error) {
	costs, total, err := s.store.FindByOrganization(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch costs")
	}

	return toResponseList(costs, (*models.Cost).ToResponse), total, nil
}

// Update updates a cost.
func (s *CostService) Update(ctx context.Context, id, orgID uint, req *models.CostUpdateRequest) (*models.CostResponse, error) {
	cost, err := s.verifyCostOwnership(ctx, id, orgID)
	if err != nil {
		return nil, err
	}

	name, err := validateRequiredName(req.Name)
	if err != nil {
		return nil, err
	}

	cost.Name = name

	if err := s.store.Update(ctx, cost); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update cost")
	}

	resp := cost.ToResponse()
	return &resp, nil
}

// Delete deletes a cost and all its entries.
func (s *CostService) Delete(ctx context.Context, id, orgID uint) error {
	if _, err := s.verifyCostOwnership(ctx, id, orgID); err != nil {
		return err
	}

	return s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Delete(txCtx, id); err != nil {
			return apperror.InternalWrap(err, "failed to delete cost")
		}
		return nil
	})
}

// CostEntry CRUD

// CreateEntry creates a new cost entry with overlap validation.
func (s *CostService) CreateEntry(ctx context.Context, costID, orgID uint, req *models.CostEntryCreateRequest) (*models.CostEntryResponse, error) {
	if _, err := s.verifyCostOwnership(ctx, costID, orgID); err != nil {
		return nil, err
	}

	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	var resp models.CostEntryResponse
	err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Entries().ValidateNoOverlap(txCtx, costID, req.From, req.To, nil); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict("cost entry overlaps with existing entry")
			}
			return apperror.InternalWrap(err, "failed to validate overlap")
		}

		entry := &models.CostEntry{
			CostID:      costID,
			Period:      models.Period{From: req.From, To: req.To},
			AmountCents: req.AmountCents,
			Notes:       req.Notes,
		}

		if err := s.store.CreateEntry(txCtx, entry); err != nil {
			return apperror.InternalWrap(err, "failed to create cost entry")
		}

		resp = entry.ToResponse()
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetEntryByID retrieves a cost entry by ID.
func (s *CostService) GetEntryByID(ctx context.Context, entryID, costID, orgID uint) (*models.CostEntryResponse, error) {
	if _, err := s.verifyCostOwnership(ctx, costID, orgID); err != nil {
		return nil, err
	}

	entry, err := s.verifyEntryOwnership(ctx, entryID, costID)
	if err != nil {
		return nil, err
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// ListEntries retrieves paginated cost entries for a cost.
func (s *CostService) ListEntries(ctx context.Context, costID, orgID uint, limit, offset int) ([]models.CostEntryResponse, int64, error) {
	if _, err := s.verifyCostOwnership(ctx, costID, orgID); err != nil {
		return nil, 0, err
	}

	entries, total, err := s.store.FindEntriesByCostPaginated(ctx, costID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch cost entries")
	}

	return toResponseList(entries, (*models.CostEntry).ToResponse), total, nil
}

// UpdateEntry updates a cost entry with overlap validation.
func (s *CostService) UpdateEntry(ctx context.Context, entryID, costID, orgID uint, req *models.CostEntryUpdateRequest) (*models.CostEntryResponse, error) {
	if _, err := s.verifyCostOwnership(ctx, costID, orgID); err != nil {
		return nil, err
	}

	entry, err := s.verifyEntryOwnership(ctx, entryID, costID)
	if err != nil {
		return nil, err
	}

	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	var resp models.CostEntryResponse
	err = s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Entries().ValidateNoOverlap(txCtx, costID, req.From, req.To, &entryID); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict("cost entry overlaps with existing entry")
			}
			return apperror.InternalWrap(err, "failed to validate overlap")
		}

		entry.From = req.From
		entry.To = req.To
		entry.AmountCents = req.AmountCents
		entry.Notes = req.Notes

		if err := s.store.UpdateEntry(txCtx, entry); err != nil {
			return apperror.InternalWrap(err, "failed to update cost entry")
		}

		resp = entry.ToResponse()
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteEntry deletes a cost entry.
func (s *CostService) DeleteEntry(ctx context.Context, entryID, costID, orgID uint) error {
	if _, err := s.verifyCostOwnership(ctx, costID, orgID); err != nil {
		return err
	}

	if _, err := s.verifyEntryOwnership(ctx, entryID, costID); err != nil {
		return err
	}

	if err := s.store.DeleteEntry(ctx, entryID); err != nil {
		return apperror.InternalWrap(err, "failed to delete cost entry")
	}
	return nil
}
