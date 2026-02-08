package service

import (
	"context"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// WaitlistService handles business logic for waitlist operations
type WaitlistService struct {
	store store.WaitlistStorer
}

// NewWaitlistService creates a new waitlist service
func NewWaitlistService(store store.WaitlistStorer) *WaitlistService {
	return &WaitlistService{store: store}
}

// List returns a paginated list of waitlist entries for an organization.
func (s *WaitlistService) List(ctx context.Context, orgID uint, limit, offset int) ([]models.WaitlistEntryResponse, int64, error) {
	entries, total, err := s.store.FindByOrganization(orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch waitlist entries")
	}

	responses := make([]models.WaitlistEntryResponse, len(entries))
	for i, e := range entries {
		responses[i] = e.ToResponse()
	}
	return responses, total, nil
}

// ListByStatus returns a paginated list of waitlist entries filtered by status.
func (s *WaitlistService) ListByStatus(ctx context.Context, orgID uint, status string, limit, offset int) ([]models.WaitlistEntryResponse, int64, error) {
	if !models.IsValidWaitlistStatus(status) {
		return nil, 0, apperror.BadRequest("invalid waitlist status")
	}

	entries, total, err := s.store.FindByOrganizationAndStatus(orgID, status, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch waitlist entries")
	}

	responses := make([]models.WaitlistEntryResponse, len(entries))
	for i, e := range entries {
		responses[i] = e.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns a waitlist entry by ID, validating it belongs to the organization.
func (s *WaitlistService) GetByID(ctx context.Context, id, orgID uint) (*models.WaitlistEntryResponse, error) {
	entry, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("waitlist entry")
	}
	if entry.OrganizationID != orgID {
		return nil, apperror.NotFound("waitlist entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// Create creates a new waitlist entry.
func (s *WaitlistService) Create(ctx context.Context, orgID uint, req *models.WaitlistEntryCreateRequest) (*models.WaitlistEntryResponse, error) {
	// Trim and validate input
	req.ChildFirstName = strings.TrimSpace(req.ChildFirstName)
	req.ChildLastName = strings.TrimSpace(req.ChildLastName)
	req.GuardianName = strings.TrimSpace(req.GuardianName)

	if validation.IsWhitespaceOnly(req.ChildFirstName) {
		return nil, apperror.BadRequest("child_first_name cannot be empty or whitespace only")
	}
	if validation.IsWhitespaceOnly(req.ChildLastName) {
		return nil, apperror.BadRequest("child_last_name cannot be empty or whitespace only")
	}
	if validation.IsWhitespaceOnly(req.GuardianName) {
		return nil, apperror.BadRequest("guardian_name cannot be empty or whitespace only")
	}

	entry := &models.WaitlistEntry{
		OrganizationID:   orgID,
		ChildFirstName:   req.ChildFirstName,
		ChildLastName:    req.ChildLastName,
		ChildBirthdate:   req.ChildBirthdate,
		GuardianName:     req.GuardianName,
		GuardianEmail:    strings.TrimSpace(req.GuardianEmail),
		GuardianPhone:    strings.TrimSpace(req.GuardianPhone),
		DesiredStartDate: req.DesiredStartDate,
		CareType:         strings.TrimSpace(req.CareType),
		Status:           models.WaitlistStatusWaiting,
		Priority:         req.Priority,
		Notes:            strings.TrimSpace(req.Notes),
	}

	if err := s.store.Create(entry); err != nil {
		return nil, apperror.Internal("failed to create waitlist entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// Update updates an existing waitlist entry.
func (s *WaitlistService) Update(ctx context.Context, id, orgID uint, req *models.WaitlistEntryUpdateRequest) (*models.WaitlistEntryResponse, error) {
	entry, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("waitlist entry")
	}
	if entry.OrganizationID != orgID {
		return nil, apperror.NotFound("waitlist entry")
	}

	if req.ChildFirstName != nil {
		trimmed := strings.TrimSpace(*req.ChildFirstName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("child_first_name cannot be empty or whitespace only")
		}
		entry.ChildFirstName = trimmed
	}
	if req.ChildLastName != nil {
		trimmed := strings.TrimSpace(*req.ChildLastName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("child_last_name cannot be empty or whitespace only")
		}
		entry.ChildLastName = trimmed
	}
	if req.ChildBirthdate != nil {
		entry.ChildBirthdate = *req.ChildBirthdate
	}
	if req.GuardianName != nil {
		trimmed := strings.TrimSpace(*req.GuardianName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("guardian_name cannot be empty or whitespace only")
		}
		entry.GuardianName = trimmed
	}
	if req.GuardianEmail != nil {
		entry.GuardianEmail = strings.TrimSpace(*req.GuardianEmail)
	}
	if req.GuardianPhone != nil {
		entry.GuardianPhone = strings.TrimSpace(*req.GuardianPhone)
	}
	if req.DesiredStartDate != nil {
		entry.DesiredStartDate = *req.DesiredStartDate
	}
	if req.CareType != nil {
		entry.CareType = strings.TrimSpace(*req.CareType)
	}
	if req.Status != nil {
		if !models.IsValidWaitlistStatus(*req.Status) {
			return nil, apperror.BadRequest("invalid waitlist status")
		}
		entry.Status = *req.Status
	}
	if req.Priority != nil {
		entry.Priority = *req.Priority
	}
	if req.Notes != nil {
		entry.Notes = strings.TrimSpace(*req.Notes)
	}

	if err := s.store.Update(entry); err != nil {
		return nil, apperror.Internal("failed to update waitlist entry")
	}

	resp := entry.ToResponse()
	return &resp, nil
}

// Delete deletes a waitlist entry.
func (s *WaitlistService) Delete(ctx context.Context, id, orgID uint) error {
	entry, err := s.store.FindByID(id)
	if err != nil {
		return apperror.NotFound("waitlist entry")
	}
	if entry.OrganizationID != orgID {
		return apperror.NotFound("waitlist entry")
	}

	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete waitlist entry")
	}
	return nil
}
