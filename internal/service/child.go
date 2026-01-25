package service

import (
	"context"
	"errors"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// ChildService handles business logic for child operations
type ChildService struct {
	store      store.ChildStorer
	groupStore store.GroupStorer
}

// NewChildService creates a new child service
func NewChildService(store store.ChildStorer, groupStore store.GroupStorer) *ChildService {
	return &ChildService{store: store, groupStore: groupStore}
}

// List returns a paginated list of children
func (s *ChildService) List(ctx context.Context, limit, offset int) ([]models.Child, int64, error) {
	children, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch children")
	}
	return children, total, nil
}

// ListByOrganization returns a paginated list of children for an organization
func (s *ChildService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.Child, int64, error) {
	children, total, err := s.store.FindByOrganization(orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch children")
	}
	return children, total, nil
}

// GetByID returns a child by ID
func (s *ChildService) GetByID(ctx context.Context, id uint) (*models.Child, error) {
	child, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	return child, nil
}

// Create creates a new child
func (s *ChildService) Create(ctx context.Context, orgID uint, req *models.ChildCreate) (*models.Child, error) {
	// Trim and validate input
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if validation.IsWhitespaceOnly(req.FirstName) {
		return nil, apperror.BadRequest("first_name cannot be empty or whitespace only")
	}
	if validation.IsWhitespaceOnly(req.LastName) {
		return nil, apperror.BadRequest("last_name cannot be empty or whitespace only")
	}
	if err := validation.ValidateBirthdate(req.Birthdate); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	child := &models.Child{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Birthdate:      req.Birthdate,
		},
	}

	if err := s.store.Create(child); err != nil {
		return nil, apperror.Internal("failed to create child")
	}

	return child, nil
}

// Update updates an existing child
func (s *ChildService) Update(ctx context.Context, id uint, req *models.ChildUpdate) (*models.Child, error) {
	child, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}

	if req.FirstName != nil {
		trimmed := strings.TrimSpace(*req.FirstName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("first_name cannot be empty or whitespace only")
		}
		child.FirstName = trimmed
	}
	if req.LastName != nil {
		trimmed := strings.TrimSpace(*req.LastName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("last_name cannot be empty or whitespace only")
		}
		child.LastName = trimmed
	}
	if req.Birthdate != nil {
		if err := validation.ValidateBirthdate(*req.Birthdate); err != nil {
			return nil, apperror.BadRequest(err.Error())
		}
		child.Birthdate = *req.Birthdate
	}

	if err := s.store.Update(child); err != nil {
		return nil, apperror.Internal("failed to update child")
	}

	return child, nil
}

// Delete deletes a child
func (s *ChildService) Delete(ctx context.Context, id uint) error {
	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete child")
	}
	return nil
}

// ListContracts returns contract history for a child
func (s *ChildService) ListContracts(ctx context.Context, childID uint) ([]models.ChildContract, error) {
	// Verify child exists
	_, err := s.store.FindByID(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}

	contracts, err := s.store.Contracts().GetHistory(childID)
	if err != nil {
		return nil, apperror.Internal("failed to fetch contracts")
	}
	return contracts, nil
}

// GetCurrentContract returns the current active contract for a child
func (s *ChildService) GetCurrentContract(ctx context.Context, childID uint) (*models.ChildContract, error) {
	contract, err := s.store.Contracts().GetCurrentContract(childID)
	if err != nil {
		return nil, apperror.Internal("failed to fetch contract")
	}
	if contract == nil {
		return nil, apperror.NotFound("active contract")
	}
	return contract, nil
}

// CreateContract creates a new contract for a child
func (s *ChildService) CreateContract(ctx context.Context, childID uint, req *models.ChildContractCreate) (*models.ChildContract, error) {
	// Validate period
	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Sanitize SpecialNeeds for XSS
	req.SpecialNeeds = validation.SanitizeHTML(strings.TrimSpace(req.SpecialNeeds))

	// Verify child exists
	child, err := s.store.FindByID(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}

	// Validate GroupID belongs to same organization as child
	if req.GroupID != nil {
		group, err := s.groupStore.FindByID(*req.GroupID)
		if err != nil {
			return nil, apperror.NotFound("group")
		}
		if group.OrganizationID != child.OrganizationID {
			return nil, apperror.BadRequest("group must belong to the same organization as the child")
		}
	}

	// Validate no overlap
	if err := s.store.Contracts().ValidateNoOverlap(childID, req.From, req.To, nil); err != nil {
		if errors.Is(err, store.ErrContractOverlap) {
			return nil, apperror.Conflict(err.Error())
		}
		return nil, apperror.Internal("failed to validate contract")
	}

	contract := &models.ChildContract{
		ChildID: childID,
		Period: models.Period{
			From: req.From,
			To:   req.To,
		},
		CareHoursPerWeek: req.CareHoursPerWeek,
		GroupID:          req.GroupID,
		MealsIncluded:    req.MealsIncluded,
		SpecialNeeds:     req.SpecialNeeds,
	}

	if err := s.store.CreateContract(contract); err != nil {
		return nil, apperror.Internal("failed to create contract")
	}

	return contract, nil
}

// DeleteContract deletes a contract
func (s *ChildService) DeleteContract(ctx context.Context, contractID uint) error {
	if err := s.store.DeleteContract(contractID); err != nil {
		return apperror.Internal("failed to delete contract")
	}
	return nil
}
