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

// PayplanService handles business logic for payplan operations
type PayplanService struct {
	store    store.PayplanStorer
	orgStore store.OrganizationStorer
}

// NewPayplanService creates a new payplan service
func NewPayplanService(store store.PayplanStorer, orgStore store.OrganizationStorer) *PayplanService {
	return &PayplanService{store: store, orgStore: orgStore}
}

// List returns a paginated list of payplans
func (s *PayplanService) List(ctx context.Context, limit, offset int) ([]models.PayplanResponse, int64, error) {
	payplans, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch payplans")
	}

	responses := make([]models.PayplanResponse, len(payplans))
	for i, p := range payplans {
		responses[i] = p.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns a payplan by ID without nested details
func (s *PayplanService) GetByID(ctx context.Context, id uint) (*models.PayplanResponse, error) {
	payplan, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("payplan")
	}
	resp := payplan.ToResponse()
	return &resp, nil
}

// GetByIDWithDetails returns a payplan by ID with all nested periods, entries, and properties
// Note: Returns raw model for complex nested structure
func (s *PayplanService) GetByIDWithDetails(ctx context.Context, id uint) (*models.Payplan, error) {
	payplan, err := s.store.FindByIDWithDetails(id)
	if err != nil {
		return nil, apperror.NotFound("payplan")
	}
	return payplan, nil
}

// PayplanCreateRequest represents the request for creating a payplan
type PayplanCreateRequest struct {
	Name string
}

// Create creates a new payplan
func (s *PayplanService) Create(ctx context.Context, req *PayplanCreateRequest) (*models.PayplanResponse, error) {
	req.Name = strings.TrimSpace(req.Name)

	if validation.IsWhitespaceOnly(req.Name) {
		return nil, apperror.BadRequest("name cannot be empty or whitespace only")
	}

	payplan := &models.Payplan{
		Name: req.Name,
	}

	if err := s.store.Create(payplan); err != nil {
		return nil, apperror.Internal("failed to create payplan")
	}

	resp := payplan.ToResponse()
	return &resp, nil
}

// PayplanUpdateRequest represents the request for updating a payplan
type PayplanUpdateRequest struct {
	Name *string
}

// Update updates an existing payplan
func (s *PayplanService) Update(ctx context.Context, id uint, req *PayplanUpdateRequest) (*models.PayplanResponse, error) {
	payplan, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("payplan")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if validation.IsWhitespaceOnly(name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		payplan.Name = name
	}

	if err := s.store.Update(payplan); err != nil {
		return nil, apperror.Internal("failed to update payplan")
	}

	resp := payplan.ToResponse()
	return &resp, nil
}

// Delete deletes a payplan
func (s *PayplanService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete payplan")
	}
	return nil
}

// Period operations

// PeriodCreateRequest represents the request for creating a period
type PeriodCreateRequest struct {
	PayplanID uint
	From      models.PayplanPeriodCreateRequest
}

// periodsOverlap checks if two date ranges overlap.
// A period with nil To date extends indefinitely into the future.
func periodsOverlap(from1 time.Time, to1 *time.Time, from2 time.Time, to2 *time.Time) bool {
	// Period 1 ends before period 2 starts (no overlap)
	if to1 != nil && !to1.After(from2) && !to1.Equal(from2) {
		// to1 < from2, but we need to check if to1 == from2 is allowed
		// For date-based periods, to1 being equal to from2 means no overlap
		// (e.g., period 1 ends on 2024-01-31, period 2 starts on 2024-02-01)
		if to1.Before(from2) {
			return false
		}
	}

	// Period 2 ends before period 1 starts (no overlap)
	if to2 != nil && to2.Before(from1) {
		return false
	}

	// If we reach here, the periods overlap
	return true
}

// validatePeriodNoOverlap checks that the new/updated period doesn't overlap with existing periods.
// excludeID is used when updating to exclude the period being updated from the check.
func (s *PayplanService) validatePeriodNoOverlap(payplanID uint, from time.Time, to *time.Time, excludeID *uint) error {
	existingPeriods, err := s.store.FindPeriodsByPayplanID(payplanID)
	if err != nil {
		return apperror.Internal("failed to check for period overlaps")
	}

	for _, existing := range existingPeriods {
		// Skip the period being updated
		if excludeID != nil && existing.ID == *excludeID {
			continue
		}

		if periodsOverlap(from, to, existing.From, existing.To) {
			return apperror.BadRequest("period overlaps with existing period")
		}
	}

	return nil
}

// CreatePeriod creates a new period
func (s *PayplanService) CreatePeriod(ctx context.Context, payplanID uint, req *models.PayplanPeriodCreateRequest) (*models.PayplanPeriodResponse, error) {
	// Verify payplan exists
	if _, err := s.store.FindByID(payplanID); err != nil {
		return nil, apperror.NotFound("payplan")
	}

	// Validate no overlap with existing periods
	if err := s.validatePeriodNoOverlap(payplanID, req.From, req.To, nil); err != nil {
		return nil, err
	}

	period := &models.PayplanPeriod{
		PayplanID: payplanID,
		From:      req.From,
		To:        req.To,
		Comment:   strings.TrimSpace(req.Comment),
	}

	if err := s.store.CreatePeriod(period); err != nil {
		return nil, apperror.Internal("failed to create period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// GetPeriodByID returns a period by ID
func (s *PayplanService) GetPeriodByID(ctx context.Context, id uint) (*models.PayplanPeriodResponse, error) {
	period, err := s.store.FindPeriodByID(id)
	if err != nil {
		return nil, apperror.NotFound("period")
	}
	resp := period.ToResponse()
	return &resp, nil
}

// UpdatePeriod updates an existing period
func (s *PayplanService) UpdatePeriod(ctx context.Context, periodID uint, req *models.PayplanPeriodUpdateRequest) (*models.PayplanPeriodResponse, error) {
	period, err := s.store.FindPeriodByID(periodID)
	if err != nil {
		return nil, apperror.NotFound("period")
	}

	// Apply updates to determine new date range
	newFrom := period.From
	newTo := period.To
	if req.From != nil {
		newFrom = *req.From
	}
	if req.To != nil {
		newTo = req.To
	}

	// Validate no overlap with other periods (excluding this one)
	if err := s.validatePeriodNoOverlap(period.PayplanID, newFrom, newTo, &periodID); err != nil {
		return nil, err
	}

	// Apply updates
	period.From = newFrom
	period.To = newTo
	if req.Comment != nil {
		period.Comment = strings.TrimSpace(*req.Comment)
	}

	if err := s.store.UpdatePeriod(period); err != nil {
		return nil, apperror.Internal("failed to update period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// DeletePeriod deletes a period
func (s *PayplanService) DeletePeriod(ctx context.Context, periodID uint) error {
	if err := s.store.DeletePeriod(periodID); err != nil {
		return apperror.Internal("failed to delete period")
	}
	return nil
}

// Entry operations

// CreateEntry creates a new entry
func (s *PayplanService) CreateEntry(ctx context.Context, periodID uint, req *models.PayplanEntryCreateRequest) (*models.PayplanEntryResponse, error) {
	// Verify period exists
	if _, err := s.store.FindPeriodByID(periodID); err != nil {
		return nil, apperror.NotFound("period")
	}

	if req.MinAge >= req.MaxAge {
		return nil, apperror.BadRequest("max_age must be greater than min_age")
	}

	entry := &models.PayplanEntry{
		PeriodID: periodID,
		MinAge:   req.MinAge,
		MaxAge:   req.MaxAge,
	}

	if err := s.store.CreateEntry(entry); err != nil {
		return nil, apperror.Internal("failed to create entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// GetEntryByID returns an entry by ID
func (s *PayplanService) GetEntryByID(ctx context.Context, id uint) (*models.PayplanEntryResponse, error) {
	entry, err := s.store.FindEntryByID(id)
	if err != nil {
		return nil, apperror.NotFound("entry")
	}
	resp := entry.ToResponse()
	return &resp, nil
}

// UpdateEntry updates an existing entry
func (s *PayplanService) UpdateEntry(ctx context.Context, entryID uint, req *models.PayplanEntryUpdateRequest) (*models.PayplanEntryResponse, error) {
	entry, err := s.store.FindEntryByID(entryID)
	if err != nil {
		return nil, apperror.NotFound("entry")
	}

	if req.MinAge != nil {
		entry.MinAge = *req.MinAge
	}
	if req.MaxAge != nil {
		entry.MaxAge = *req.MaxAge
	}

	if entry.MinAge >= entry.MaxAge {
		return nil, apperror.BadRequest("max_age must be greater than min_age")
	}

	if err := s.store.UpdateEntry(entry); err != nil {
		return nil, apperror.Internal("failed to update entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// DeleteEntry deletes an entry
func (s *PayplanService) DeleteEntry(ctx context.Context, entryID uint) error {
	if err := s.store.DeleteEntry(entryID); err != nil {
		return apperror.Internal("failed to delete entry")
	}
	return nil
}

// Property operations

// CreateProperty creates a new property
func (s *PayplanService) CreateProperty(ctx context.Context, entryID uint, req *models.PayplanPropertyCreateRequest) (*models.PayplanPropertyResponse, error) {
	// Verify entry exists
	if _, err := s.store.FindEntryByID(entryID); err != nil {
		return nil, apperror.NotFound("entry")
	}

	property := &models.PayplanProperty{
		EntryID:     entryID,
		Name:        strings.TrimSpace(req.Name),
		Payment:     req.Payment,
		Requirement: req.Requirement,
		Comment:     strings.TrimSpace(req.Comment),
	}

	if validation.IsWhitespaceOnly(property.Name) {
		return nil, apperror.BadRequest("name cannot be empty or whitespace only")
	}

	if err := s.store.CreateProperty(property); err != nil {
		return nil, apperror.Internal("failed to create property")
	}

	resp := property.ToResponse()
	return &resp, nil
}

// GetPropertyByID returns a property by ID
func (s *PayplanService) GetPropertyByID(ctx context.Context, id uint) (*models.PayplanPropertyResponse, error) {
	property, err := s.store.FindPropertyByID(id)
	if err != nil {
		return nil, apperror.NotFound("property")
	}
	resp := property.ToResponse()
	return &resp, nil
}

// UpdateProperty updates an existing property
func (s *PayplanService) UpdateProperty(ctx context.Context, propertyID uint, req *models.PayplanPropertyUpdateRequest) (*models.PayplanPropertyResponse, error) {
	property, err := s.store.FindPropertyByID(propertyID)
	if err != nil {
		return nil, apperror.NotFound("property")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if validation.IsWhitespaceOnly(name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		property.Name = name
	}
	if req.Payment != nil {
		property.Payment = *req.Payment
	}
	if req.Requirement != nil {
		property.Requirement = *req.Requirement
	}
	if req.Comment != nil {
		property.Comment = strings.TrimSpace(*req.Comment)
	}

	if err := s.store.UpdateProperty(property); err != nil {
		return nil, apperror.Internal("failed to update property")
	}

	resp := property.ToResponse()
	return &resp, nil
}

// DeleteProperty deletes a property
func (s *PayplanService) DeleteProperty(ctx context.Context, propertyID uint) error {
	if err := s.store.DeleteProperty(propertyID); err != nil {
		return apperror.Internal("failed to delete property")
	}
	return nil
}

// Organization payplan assignment

// AssignPayplanToOrg assigns a payplan to an organization
func (s *PayplanService) AssignPayplanToOrg(ctx context.Context, orgID, payplanID uint) error {
	// Verify organization exists
	if _, err := s.orgStore.FindByID(orgID); err != nil {
		return apperror.NotFound("organization")
	}

	// Verify payplan exists
	if _, err := s.store.FindByID(payplanID); err != nil {
		return apperror.NotFound("payplan")
	}

	if err := s.store.AssignPayplanToOrg(orgID, payplanID); err != nil {
		return apperror.Internal("failed to assign payplan to organization")
	}
	return nil
}

// RemovePayplanFromOrg removes the payplan assignment from an organization
func (s *PayplanService) RemovePayplanFromOrg(ctx context.Context, orgID uint) error {
	// Verify organization exists
	if _, err := s.orgStore.FindByID(orgID); err != nil {
		return apperror.NotFound("organization")
	}

	if err := s.store.RemovePayplanFromOrg(orgID); err != nil {
		return apperror.Internal("failed to remove payplan from organization")
	}
	return nil
}
