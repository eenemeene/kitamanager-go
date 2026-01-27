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

// GovernmentFundingService handles business logic for government funding operations
type GovernmentFundingService struct {
	store    store.GovernmentFundingStorer
	orgStore store.OrganizationStorer
}

// NewGovernmentFundingService creates a new government funding service
func NewGovernmentFundingService(store store.GovernmentFundingStorer, orgStore store.OrganizationStorer) *GovernmentFundingService {
	return &GovernmentFundingService{store: store, orgStore: orgStore}
}

// List returns a paginated list of government fundings
func (s *GovernmentFundingService) List(ctx context.Context, limit, offset int) ([]models.GovernmentFundingResponse, int64, error) {
	fundings, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch government fundings")
	}

	responses := make([]models.GovernmentFundingResponse, len(fundings))
	for i, f := range fundings {
		responses[i] = f.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns a government funding by ID without nested details
func (s *GovernmentFundingService) GetByID(ctx context.Context, id uint) (*models.GovernmentFundingResponse, error) {
	funding, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("government funding")
	}
	resp := funding.ToResponse()
	return &resp, nil
}

// GetByIDWithDetails returns a government funding by ID with all nested periods, entries, and properties
// Note: Returns raw model for complex nested structure
func (s *GovernmentFundingService) GetByIDWithDetails(ctx context.Context, id uint) (*models.GovernmentFunding, error) {
	funding, err := s.store.FindByIDWithDetails(id)
	if err != nil {
		return nil, apperror.NotFound("government funding")
	}
	return funding, nil
}

// GovernmentFundingCreateRequest represents the request for creating a government funding
type GovernmentFundingCreateRequest struct {
	Name string
}

// Create creates a new government funding
func (s *GovernmentFundingService) Create(ctx context.Context, req *GovernmentFundingCreateRequest) (*models.GovernmentFundingResponse, error) {
	req.Name = strings.TrimSpace(req.Name)

	if validation.IsWhitespaceOnly(req.Name) {
		return nil, apperror.BadRequest("name cannot be empty or whitespace only")
	}

	funding := &models.GovernmentFunding{
		Name: req.Name,
	}

	if err := s.store.Create(funding); err != nil {
		return nil, apperror.Internal("failed to create government funding")
	}

	resp := funding.ToResponse()
	return &resp, nil
}

// GovernmentFundingUpdateRequest represents the request for updating a government funding
type GovernmentFundingUpdateRequest struct {
	Name *string
}

// Update updates an existing government funding
func (s *GovernmentFundingService) Update(ctx context.Context, id uint, req *GovernmentFundingUpdateRequest) (*models.GovernmentFundingResponse, error) {
	funding, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("government funding")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if validation.IsWhitespaceOnly(name) {
			return nil, apperror.BadRequest("name cannot be empty or whitespace only")
		}
		funding.Name = name
	}

	if err := s.store.Update(funding); err != nil {
		return nil, apperror.Internal("failed to update government funding")
	}

	resp := funding.ToResponse()
	return &resp, nil
}

// Delete deletes a government funding
func (s *GovernmentFundingService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete government funding")
	}
	return nil
}

// Period operations

// GovernmentFundingPeriodServiceCreateRequest represents the request for creating a period
type GovernmentFundingPeriodServiceCreateRequest struct {
	GovernmentFundingID uint
	From                models.GovernmentFundingPeriodCreateRequest
}

// governmentFundingPeriodsOverlap checks if two date ranges overlap.
// A period with nil To date extends indefinitely into the future.
func governmentFundingPeriodsOverlap(from1 time.Time, to1 *time.Time, from2 time.Time, to2 *time.Time) bool {
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
func (s *GovernmentFundingService) validatePeriodNoOverlap(governmentFundingID uint, from time.Time, to *time.Time, excludeID *uint) error {
	existingPeriods, err := s.store.FindPeriodsByGovernmentFundingID(governmentFundingID)
	if err != nil {
		return apperror.Internal("failed to check for period overlaps")
	}

	for _, existing := range existingPeriods {
		// Skip the period being updated
		if excludeID != nil && existing.ID == *excludeID {
			continue
		}

		if governmentFundingPeriodsOverlap(from, to, existing.From, existing.To) {
			return apperror.BadRequest("period overlaps with existing period")
		}
	}

	return nil
}

// CreatePeriod creates a new period
func (s *GovernmentFundingService) CreatePeriod(ctx context.Context, governmentFundingID uint, req *models.GovernmentFundingPeriodCreateRequest) (*models.GovernmentFundingPeriodResponse, error) {
	// Verify government funding exists
	if _, err := s.store.FindByID(governmentFundingID); err != nil {
		return nil, apperror.NotFound("government funding")
	}

	// Validate no overlap with existing periods
	if err := s.validatePeriodNoOverlap(governmentFundingID, req.From, req.To, nil); err != nil {
		return nil, err
	}

	period := &models.GovernmentFundingPeriod{
		GovernmentFundingID: governmentFundingID,
		From:                req.From,
		To:                  req.To,
		Comment:             strings.TrimSpace(req.Comment),
	}

	if err := s.store.CreatePeriod(period); err != nil {
		return nil, apperror.Internal("failed to create period")
	}

	resp := period.ToResponse()
	return &resp, nil
}

// GetPeriodByID returns a period by ID
func (s *GovernmentFundingService) GetPeriodByID(ctx context.Context, id uint) (*models.GovernmentFundingPeriodResponse, error) {
	period, err := s.store.FindPeriodByID(id)
	if err != nil {
		return nil, apperror.NotFound("period")
	}
	resp := period.ToResponse()
	return &resp, nil
}

// UpdatePeriod updates an existing period
func (s *GovernmentFundingService) UpdatePeriod(ctx context.Context, periodID uint, req *models.GovernmentFundingPeriodUpdateRequest) (*models.GovernmentFundingPeriodResponse, error) {
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
	if err := s.validatePeriodNoOverlap(period.GovernmentFundingID, newFrom, newTo, &periodID); err != nil {
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
func (s *GovernmentFundingService) DeletePeriod(ctx context.Context, periodID uint) error {
	if err := s.store.DeletePeriod(periodID); err != nil {
		return apperror.Internal("failed to delete period")
	}
	return nil
}

// Entry operations

// CreateEntry creates a new entry
func (s *GovernmentFundingService) CreateEntry(ctx context.Context, periodID uint, req *models.GovernmentFundingEntryCreateRequest) (*models.GovernmentFundingEntryResponse, error) {
	// Verify period exists
	if _, err := s.store.FindPeriodByID(periodID); err != nil {
		return nil, apperror.NotFound("period")
	}

	if req.MinAge >= req.MaxAge {
		return nil, apperror.BadRequest("max_age must be greater than min_age")
	}

	entry := &models.GovernmentFundingEntry{
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
func (s *GovernmentFundingService) GetEntryByID(ctx context.Context, id uint) (*models.GovernmentFundingEntryResponse, error) {
	entry, err := s.store.FindEntryByID(id)
	if err != nil {
		return nil, apperror.NotFound("entry")
	}
	resp := entry.ToResponse()
	return &resp, nil
}

// UpdateEntry updates an existing entry
func (s *GovernmentFundingService) UpdateEntry(ctx context.Context, entryID uint, req *models.GovernmentFundingEntryUpdateRequest) (*models.GovernmentFundingEntryResponse, error) {
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
func (s *GovernmentFundingService) DeleteEntry(ctx context.Context, entryID uint) error {
	if err := s.store.DeleteEntry(entryID); err != nil {
		return apperror.Internal("failed to delete entry")
	}
	return nil
}

// Property operations

// CreateProperty creates a new property
func (s *GovernmentFundingService) CreateProperty(ctx context.Context, entryID uint, req *models.GovernmentFundingPropertyCreateRequest) (*models.GovernmentFundingPropertyResponse, error) {
	// Verify entry exists
	if _, err := s.store.FindEntryByID(entryID); err != nil {
		return nil, apperror.NotFound("entry")
	}

	property := &models.GovernmentFundingProperty{
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
func (s *GovernmentFundingService) GetPropertyByID(ctx context.Context, id uint) (*models.GovernmentFundingPropertyResponse, error) {
	property, err := s.store.FindPropertyByID(id)
	if err != nil {
		return nil, apperror.NotFound("property")
	}
	resp := property.ToResponse()
	return &resp, nil
}

// UpdateProperty updates an existing property
func (s *GovernmentFundingService) UpdateProperty(ctx context.Context, propertyID uint, req *models.GovernmentFundingPropertyUpdateRequest) (*models.GovernmentFundingPropertyResponse, error) {
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
func (s *GovernmentFundingService) DeleteProperty(ctx context.Context, propertyID uint) error {
	if err := s.store.DeleteProperty(propertyID); err != nil {
		return apperror.Internal("failed to delete property")
	}
	return nil
}

// Organization government funding assignment

// AssignGovernmentFundingToOrg assigns a government funding to an organization
func (s *GovernmentFundingService) AssignGovernmentFundingToOrg(ctx context.Context, orgID, governmentFundingID uint) error {
	// Verify organization exists
	if _, err := s.orgStore.FindByID(orgID); err != nil {
		return apperror.NotFound("organization")
	}

	// Verify government funding exists
	if _, err := s.store.FindByID(governmentFundingID); err != nil {
		return apperror.NotFound("government funding")
	}

	if err := s.store.AssignGovernmentFundingToOrg(orgID, governmentFundingID); err != nil {
		return apperror.Internal("failed to assign government funding to organization")
	}
	return nil
}

// RemoveGovernmentFundingFromOrg removes the government funding assignment from an organization
func (s *GovernmentFundingService) RemoveGovernmentFundingFromOrg(ctx context.Context, orgID uint) error {
	// Verify organization exists
	if _, err := s.orgStore.FindByID(orgID); err != nil {
		return apperror.NotFound("organization")
	}

	if err := s.store.RemoveGovernmentFundingFromOrg(orgID); err != nil {
		return apperror.Internal("failed to remove government funding from organization")
	}
	return nil
}
