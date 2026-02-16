package service

import (
	"context"
	"errors"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// BudgetItemService handles business logic for budget items and their entries.
type BudgetItemService struct {
	store      store.BudgetItemStorer
	transactor store.Transactor
}

// NewBudgetItemService creates a new BudgetItemService.
func NewBudgetItemService(store store.BudgetItemStorer, transactor store.Transactor) *BudgetItemService {
	return &BudgetItemService{store: store, transactor: transactor}
}

// verifyBudgetItemOwnership verifies a budget item exists and belongs to the organization.
func (s *BudgetItemService) verifyBudgetItemOwnership(ctx context.Context, itemID, orgID uint) (*models.BudgetItem, error) {
	item, err := s.store.FindByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("budget item")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch budget item")
	}
	if err := verifyOrgOwnership(item, orgID, "budget item"); err != nil {
		return nil, err
	}
	return item, nil
}

// verifyEntryOwnership verifies a budget item entry exists and belongs to the budget item.
func (s *BudgetItemService) verifyEntryOwnership(ctx context.Context, entryID, itemID uint) (*models.BudgetItemEntry, error) {
	entry, err := s.store.FindEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("budget item entry")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch budget item entry")
	}
	if err := verifyRecordOwnership(entry, itemID, "budget item entry"); err != nil {
		return nil, err
	}
	return entry, nil
}

// BudgetItem CRUD

// Create creates a new budget item.
func (s *BudgetItemService) Create(ctx context.Context, orgID uint, req *models.BudgetItemCreateRequest) (*models.BudgetItemResponse, error) {
	name, err := validateRequiredName(req.Name)
	if err != nil {
		return nil, err
	}

	if !models.ValidBudgetItemCategory(req.Category) {
		return nil, apperror.BadRequest("category must be 'income' or 'expense'")
	}

	item := &models.BudgetItem{
		OrganizationID: orgID,
		Name:           name,
		Category:       req.Category,
		PerChild:       req.PerChild,
	}

	if err := s.store.Create(ctx, item); err != nil {
		if store.IsDuplicateKeyError(err) {
			return nil, apperror.Conflict("budget item with this name already exists in the organization")
		}
		return nil, apperror.InternalWrap(err, "failed to create budget item")
	}

	resp := item.ToResponse()
	return &resp, nil
}

// GetByID retrieves a budget item by ID with all entries.
func (s *BudgetItemService) GetByID(ctx context.Context, id, orgID uint) (*models.BudgetItemDetailResponse, error) {
	item, err := s.store.FindByIDWithEntries(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperror.NotFound("budget item")
		}
		return nil, apperror.InternalWrap(err, "failed to fetch budget item")
	}

	if item.OrganizationID != orgID {
		return nil, apperror.NotFound("budget item")
	}

	resp := item.ToDetailResponse()
	return &resp, nil
}

// List retrieves all budget items for an organization.
func (s *BudgetItemService) List(ctx context.Context, orgID uint, limit, offset int) ([]models.BudgetItemResponse, int64, error) {
	items, total, err := s.store.FindByOrganization(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch budget items")
	}

	return toResponseList(items, (*models.BudgetItem).ToResponse), total, nil
}

// Update updates a budget item.
func (s *BudgetItemService) Update(ctx context.Context, id, orgID uint, req *models.BudgetItemUpdateRequest) (*models.BudgetItemResponse, error) {
	item, err := s.verifyBudgetItemOwnership(ctx, id, orgID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name, err := validateRequiredName(*req.Name)
		if err != nil {
			return nil, err
		}
		item.Name = name
	}

	if req.Category != nil {
		if !models.ValidBudgetItemCategory(*req.Category) {
			return nil, apperror.BadRequest("category must be 'income' or 'expense'")
		}
		item.Category = *req.Category
	}

	if req.PerChild != nil {
		item.PerChild = *req.PerChild
	}

	if err := s.store.Update(ctx, item); err != nil {
		if store.IsDuplicateKeyError(err) {
			return nil, apperror.Conflict("budget item with this name already exists in the organization")
		}
		return nil, apperror.InternalWrap(err, "failed to update budget item")
	}

	resp := item.ToResponse()
	return &resp, nil
}

// Delete deletes a budget item and all its entries.
func (s *BudgetItemService) Delete(ctx context.Context, id, orgID uint) error {
	if _, err := s.verifyBudgetItemOwnership(ctx, id, orgID); err != nil {
		return err
	}

	return s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Delete(txCtx, id); err != nil {
			return apperror.InternalWrap(err, "failed to delete budget item")
		}
		return nil
	})
}

// BudgetItemEntry CRUD

// CreateEntry creates a new budget item entry with overlap validation.
func (s *BudgetItemService) CreateEntry(ctx context.Context, itemID, orgID uint, req *models.BudgetItemEntryCreateRequest) (*models.BudgetItemEntryResponse, error) {
	if _, err := s.verifyBudgetItemOwnership(ctx, itemID, orgID); err != nil {
		return nil, err
	}

	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	var resp models.BudgetItemEntryResponse
	err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Entries().ValidateNoOverlap(txCtx, itemID, req.From, req.To, nil); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict("budget item entry overlaps with existing entry")
			}
			return apperror.InternalWrap(err, "failed to validate overlap")
		}

		entry := &models.BudgetItemEntry{
			BudgetItemID: itemID,
			Period:       models.Period{From: req.From, To: req.To},
			AmountCents:  req.AmountCents,
			Notes:        req.Notes,
		}

		if err := s.store.CreateEntry(txCtx, entry); err != nil {
			return apperror.InternalWrap(err, "failed to create budget item entry")
		}

		resp = entry.ToResponse()
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetEntryByID retrieves a budget item entry by ID.
func (s *BudgetItemService) GetEntryByID(ctx context.Context, entryID, itemID, orgID uint) (*models.BudgetItemEntryResponse, error) {
	if _, err := s.verifyBudgetItemOwnership(ctx, itemID, orgID); err != nil {
		return nil, err
	}

	entry, err := s.verifyEntryOwnership(ctx, entryID, itemID)
	if err != nil {
		return nil, err
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// ListEntries retrieves paginated budget item entries for a budget item.
func (s *BudgetItemService) ListEntries(ctx context.Context, itemID, orgID uint, limit, offset int) ([]models.BudgetItemEntryResponse, int64, error) {
	if _, err := s.verifyBudgetItemOwnership(ctx, itemID, orgID); err != nil {
		return nil, 0, err
	}

	entries, total, err := s.store.FindEntriesByBudgetItemPaginated(ctx, itemID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch budget item entries")
	}

	return toResponseList(entries, (*models.BudgetItemEntry).ToResponse), total, nil
}

// UpdateEntry updates a budget item entry with overlap validation.
func (s *BudgetItemService) UpdateEntry(ctx context.Context, entryID, itemID, orgID uint, req *models.BudgetItemEntryUpdateRequest) (*models.BudgetItemEntryResponse, error) {
	if _, err := s.verifyBudgetItemOwnership(ctx, itemID, orgID); err != nil {
		return nil, err
	}

	entry, err := s.verifyEntryOwnership(ctx, entryID, itemID)
	if err != nil {
		return nil, err
	}

	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	var resp models.BudgetItemEntryResponse
	err = s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Entries().ValidateNoOverlap(txCtx, itemID, req.From, req.To, &entryID); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict("budget item entry overlaps with existing entry")
			}
			return apperror.InternalWrap(err, "failed to validate overlap")
		}

		entry.From = req.From
		entry.To = req.To
		entry.AmountCents = req.AmountCents
		entry.Notes = req.Notes

		if err := s.store.UpdateEntry(txCtx, entry); err != nil {
			return apperror.InternalWrap(err, "failed to update budget item entry")
		}

		resp = entry.ToResponse()
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteEntry deletes a budget item entry.
func (s *BudgetItemService) DeleteEntry(ctx context.Context, entryID, itemID, orgID uint) error {
	if _, err := s.verifyBudgetItemOwnership(ctx, itemID, orgID); err != nil {
		return err
	}

	if _, err := s.verifyEntryOwnership(ctx, entryID, itemID); err != nil {
		return err
	}

	if err := s.store.DeleteEntry(ctx, entryID); err != nil {
		return apperror.InternalWrap(err, "failed to delete budget item entry")
	}
	return nil
}
