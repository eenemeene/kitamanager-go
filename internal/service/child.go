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
	store store.ChildStorer
}

// NewChildService creates a new child service
func NewChildService(store store.ChildStorer) *ChildService {
	return &ChildService{store: store}
}

// List returns a paginated list of children
func (s *ChildService) List(ctx context.Context, limit, offset int) ([]models.ChildResponse, int64, error) {
	children, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch children")
	}

	responses := make([]models.ChildResponse, len(children))
	for i, c := range children {
		responses[i] = c.ToResponse()
	}
	return responses, total, nil
}

// ListByOrganization returns a paginated list of children for an organization
func (s *ChildService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.ChildResponse, int64, error) {
	children, total, err := s.store.FindByOrganization(orgID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch children")
	}

	responses := make([]models.ChildResponse, len(children))
	for i, c := range children {
		responses[i] = c.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns a child by ID, validating it belongs to the specified organization
func (s *ChildService) GetByID(ctx context.Context, id, orgID uint) (*models.ChildResponse, error) {
	child, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	// Security: Validate child belongs to the specified organization
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}
	resp := child.ToResponse()
	return &resp, nil
}

// Create creates a new child
func (s *ChildService) Create(ctx context.Context, orgID uint, req *models.ChildCreateRequest) (*models.ChildResponse, error) {
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

	resp := child.ToResponse()
	return &resp, nil
}

// Update updates an existing child, validating it belongs to the specified organization
func (s *ChildService) Update(ctx context.Context, id, orgID uint, req *models.ChildUpdateRequest) (*models.ChildResponse, error) {
	child, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	// Security: Validate child belongs to the specified organization
	if child.OrganizationID != orgID {
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

	resp := child.ToResponse()
	return &resp, nil
}

// Delete deletes a child, validating it belongs to the specified organization
func (s *ChildService) Delete(ctx context.Context, id, orgID uint) error {
	// Security: Validate child belongs to the specified organization
	child, err := s.store.FindByID(id)
	if err != nil {
		return apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return apperror.NotFound("child")
	}

	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete child")
	}
	return nil
}

// ListContracts returns contract history for a child, validating it belongs to the specified organization
func (s *ChildService) ListContracts(ctx context.Context, childID, orgID uint) ([]models.ChildContractResponse, error) {
	// Verify child exists and belongs to org
	child, err := s.store.FindByID(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	// Security: Validate child belongs to the specified organization
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	contracts, err := s.store.Contracts().GetHistory(childID)
	if err != nil {
		return nil, apperror.Internal("failed to fetch contracts")
	}

	responses := make([]models.ChildContractResponse, len(contracts))
	for i, c := range contracts {
		responses[i] = c.ToResponse()
	}
	return responses, nil
}

// GetCurrentContract returns the current active contract for a child, validating it belongs to the specified organization
func (s *ChildService) GetCurrentContract(ctx context.Context, childID, orgID uint) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization
	child, err := s.store.FindByID(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	contract, err := s.store.Contracts().GetCurrentContract(childID)
	if err != nil {
		return nil, apperror.Internal("failed to fetch contract")
	}
	if contract == nil {
		return nil, apperror.NotFound("active contract")
	}
	resp := contract.ToResponse()
	return &resp, nil
}

// CreateContract creates a new contract for a child, validating it belongs to the specified organization
func (s *ChildService) CreateContract(ctx context.Context, childID, orgID uint, req *models.ChildContractCreateRequest) (*models.ChildContractResponse, error) {
	// Validate period
	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Verify child exists and belongs to org
	child, err := s.store.FindByID(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	// Security: Validate child belongs to the specified organization
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
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
		Attributes: req.Attributes,
	}

	if err := s.store.CreateContract(contract); err != nil {
		return nil, apperror.Internal("failed to create contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// UpdateContract updates an existing contract, validating it belongs to a child in the specified organization
func (s *ChildService) UpdateContract(ctx context.Context, contractID, childID, orgID uint, req *models.ChildContractUpdateRequest) (*models.ChildContractResponse, error) {
	// Security: Validate child belongs to the specified organization
	child, err := s.store.FindByID(childID)
	if err != nil {
		return nil, apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return nil, apperror.NotFound("child")
	}

	// Validate contract belongs to the child
	contract, err := s.store.FindContractByID(contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if contract.ChildID != childID {
		return nil, apperror.NotFound("contract")
	}

	// Update fields if provided
	if req.From != nil {
		contract.From = *req.From
	}
	if req.To != nil {
		contract.To = req.To
	}
	if req.Attributes != nil {
		contract.Attributes = req.Attributes
	}

	// Validate period
	if err := validation.ValidatePeriod(contract.From, contract.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Validate no overlap (excluding this contract)
	if err := s.store.Contracts().ValidateNoOverlap(childID, contract.From, contract.To, &contractID); err != nil {
		if errors.Is(err, store.ErrContractOverlap) {
			return nil, apperror.Conflict(err.Error())
		}
		return nil, apperror.Internal("failed to validate contract")
	}

	if err := s.store.UpdateContract(contract); err != nil {
		return nil, apperror.Internal("failed to update contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// DeleteContract deletes a contract, validating it belongs to a child in the specified organization
func (s *ChildService) DeleteContract(ctx context.Context, contractID, childID, orgID uint) error {
	// Security: Validate child belongs to the specified organization
	child, err := s.store.FindByID(childID)
	if err != nil {
		return apperror.NotFound("child")
	}
	if child.OrganizationID != orgID {
		return apperror.NotFound("child")
	}

	// Validate contract belongs to the child
	contract, err := s.store.FindContractByID(contractID)
	if err != nil {
		return apperror.NotFound("contract")
	}
	if contract.ChildID != childID {
		return apperror.NotFound("contract")
	}

	if err := s.store.DeleteContract(contractID); err != nil {
		return apperror.Internal("failed to delete contract")
	}
	return nil
}
