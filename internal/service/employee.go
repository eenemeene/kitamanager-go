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

// EmployeeService handles business logic for employee operations
type EmployeeService struct {
	store store.EmployeeStorer
}

// NewEmployeeService creates a new employee service
func NewEmployeeService(store store.EmployeeStorer) *EmployeeService {
	return &EmployeeService{store: store}
}

// List returns a paginated list of employees
func (s *EmployeeService) List(ctx context.Context, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	employees, total, err := s.store.FindAll(limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch employees")
	}

	responses := make([]models.EmployeeResponse, len(employees))
	for i, emp := range employees {
		responses[i] = emp.ToResponse()
	}
	return responses, total, nil
}

// ListByOrganization returns a paginated list of employees for an organization
func (s *EmployeeService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	return s.ListByOrganizationAndSection(ctx, orgID, nil, limit, offset)
}

// ListByOrganizationAndSection returns a paginated list of employees for an organization, optionally filtered by section
func (s *EmployeeService) ListByOrganizationAndSection(ctx context.Context, orgID uint, sectionID *uint, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	employees, total, err := s.store.FindByOrganizationAndSection(orgID, sectionID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch employees")
	}

	responses := make([]models.EmployeeResponse, len(employees))
	for i, emp := range employees {
		responses[i] = emp.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns an employee by ID, validating it belongs to the specified organization
func (s *EmployeeService) GetByID(ctx context.Context, id, orgID uint) (*models.EmployeeResponse, error) {
	employee, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	// Security: Validate employee belongs to the specified organization
	if employee.OrganizationID != orgID {
		return nil, apperror.NotFound("employee")
	}
	resp := employee.ToResponse()
	return &resp, nil
}

// Create creates a new employee
func (s *EmployeeService) Create(ctx context.Context, orgID uint, req *models.EmployeeCreateRequest) (*models.EmployeeResponse, error) {
	// Trim and validate input
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if validation.IsWhitespaceOnly(req.FirstName) {
		return nil, apperror.BadRequest("first_name cannot be empty or whitespace only")
	}
	if validation.IsWhitespaceOnly(req.LastName) {
		return nil, apperror.BadRequest("last_name cannot be empty or whitespace only")
	}
	if !models.IsValidGender(req.Gender) {
		return nil, apperror.BadRequest("gender must be one of: male, female, diverse")
	}
	if err := validation.ValidateBirthdate(req.Birthdate); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: orgID,
			SectionID:      req.SectionID,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Gender:         req.Gender,
			Birthdate:      req.Birthdate,
		},
	}

	if err := s.store.Create(employee); err != nil {
		return nil, apperror.Internal("failed to create employee")
	}

	resp := employee.ToResponse()
	return &resp, nil
}

// Update updates an existing employee, validating it belongs to the specified organization
func (s *EmployeeService) Update(ctx context.Context, id, orgID uint, req *models.EmployeeUpdateRequest) (*models.EmployeeResponse, error) {
	employee, err := s.store.FindByID(id)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	// Security: Validate employee belongs to the specified organization
	if employee.OrganizationID != orgID {
		return nil, apperror.NotFound("employee")
	}

	if req.FirstName != nil {
		trimmed := strings.TrimSpace(*req.FirstName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("first_name cannot be empty or whitespace only")
		}
		employee.FirstName = trimmed
	}
	if req.LastName != nil {
		trimmed := strings.TrimSpace(*req.LastName)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("last_name cannot be empty or whitespace only")
		}
		employee.LastName = trimmed
	}
	if req.Gender != nil {
		if !models.IsValidGender(*req.Gender) {
			return nil, apperror.BadRequest("gender must be one of: male, female, diverse")
		}
		employee.Gender = *req.Gender
	}
	if req.Birthdate != nil {
		if err := validation.ValidateBirthdate(*req.Birthdate); err != nil {
			return nil, apperror.BadRequest(err.Error())
		}
		employee.Birthdate = *req.Birthdate
	}
	if req.SectionID != nil {
		employee.SectionID = req.SectionID
	}

	if err := s.store.Update(employee); err != nil {
		return nil, apperror.Internal("failed to update employee")
	}

	resp := employee.ToResponse()
	return &resp, nil
}

// Delete deletes an employee, validating it belongs to the specified organization
func (s *EmployeeService) Delete(ctx context.Context, id, orgID uint) error {
	// Security: Validate employee belongs to the specified organization
	employee, err := s.store.FindByID(id)
	if err != nil {
		return apperror.NotFound("employee")
	}
	if employee.OrganizationID != orgID {
		return apperror.NotFound("employee")
	}

	if err := s.store.Delete(id); err != nil {
		return apperror.Internal("failed to delete employee")
	}
	return nil
}

// ListContracts returns paginated contract history for an employee, validating it belongs to the specified organization
func (s *EmployeeService) ListContracts(ctx context.Context, employeeID, orgID uint, limit, offset int) ([]models.EmployeeContractResponse, int64, error) {
	// Verify employee exists and belongs to org (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(employeeID)
	if err != nil {
		return nil, 0, apperror.NotFound("employee")
	}
	// Security: Validate employee belongs to the specified organization
	if employee.OrganizationID != orgID {
		return nil, 0, apperror.NotFound("employee")
	}

	contracts, total, err := s.store.Contracts().GetHistoryPaginated(employeeID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal("failed to fetch contracts")
	}

	responses := make([]models.EmployeeContractResponse, len(contracts))
	for i, c := range contracts {
		responses[i] = c.ToResponse()
	}
	return responses, total, nil
}

// GetCurrentContract returns the current active contract for an employee, validating it belongs to the specified organization
func (s *EmployeeService) GetCurrentContract(ctx context.Context, employeeID, orgID uint) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if employee.OrganizationID != orgID {
		return nil, apperror.NotFound("employee")
	}

	contract, err := s.store.Contracts().GetCurrentContract(employeeID)
	if err != nil {
		return nil, apperror.Internal("failed to fetch contract")
	}
	if contract == nil {
		return nil, apperror.NotFound("active contract")
	}
	resp := contract.ToResponse()
	return &resp, nil
}

// CreateContract creates a new contract for an employee, validating it belongs to the specified organization
func (s *EmployeeService) CreateContract(ctx context.Context, employeeID, orgID uint, req *models.EmployeeContractCreateRequest) (*models.EmployeeContractResponse, error) {
	// Trim and validate input
	req.Position = strings.TrimSpace(req.Position)

	if validation.IsWhitespaceOnly(req.Position) {
		return nil, apperror.BadRequest("position cannot be empty or whitespace only")
	}
	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if err := validation.ValidateWeeklyHours(req.WeeklyHours, "weekly_hours"); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	req.Grade = strings.TrimSpace(req.Grade)

	// Verify employee exists and belongs to org (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	// Security: Validate employee belongs to the specified organization
	if employee.OrganizationID != orgID {
		return nil, apperror.NotFound("employee")
	}

	// Validate no overlap
	if err := s.store.Contracts().ValidateNoOverlap(employeeID, req.From, req.To, nil); err != nil {
		if errors.Is(err, store.ErrContractOverlap) {
			return nil, apperror.Conflict(err.Error())
		}
		return nil, apperror.Internal("failed to validate contract")
	}

	contract := &models.EmployeeContract{
		EmployeeID: employeeID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: req.From,
				To:   req.To,
			},
			Properties: req.Properties,
		},
		Position:    req.Position,
		Grade:       req.Grade,
		Step:        req.Step,
		WeeklyHours: req.WeeklyHours,
	}

	if err := s.store.CreateContract(contract); err != nil {
		return nil, apperror.Internal("failed to create contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// DeleteContract deletes a contract, validating it belongs to an employee in the specified organization
func (s *EmployeeService) DeleteContract(ctx context.Context, contractID, employeeID, orgID uint) error {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(employeeID)
	if err != nil {
		return apperror.NotFound("employee")
	}
	if employee.OrganizationID != orgID {
		return apperror.NotFound("employee")
	}

	// Validate contract belongs to the employee
	contract, err := s.store.FindContractByID(contractID)
	if err != nil {
		return apperror.NotFound("contract")
	}
	if contract.EmployeeID != employeeID {
		return apperror.NotFound("contract")
	}

	if err := s.store.DeleteContract(contractID); err != nil {
		return apperror.Internal("failed to delete contract")
	}
	return nil
}

// GetContractByID returns a contract by ID, validating ownership
func (s *EmployeeService) GetContractByID(ctx context.Context, contractID, employeeID, orgID uint) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if employee.OrganizationID != orgID {
		return nil, apperror.NotFound("employee")
	}

	// Get contract
	contract, err := s.store.FindContractByID(contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if contract.EmployeeID != employeeID {
		return nil, apperror.NotFound("contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// UpdateContract updates an existing contract, validating ownership
func (s *EmployeeService) UpdateContract(ctx context.Context, contractID, employeeID, orgID uint, req *models.EmployeeContractUpdateRequest) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if employee.OrganizationID != orgID {
		return nil, apperror.NotFound("employee")
	}

	// Get contract
	contract, err := s.store.FindContractByID(contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if contract.EmployeeID != employeeID {
		return nil, apperror.NotFound("contract")
	}

	// Update fields if provided
	if req.Position != nil {
		trimmed := strings.TrimSpace(*req.Position)
		if validation.IsWhitespaceOnly(trimmed) {
			return nil, apperror.BadRequest("position cannot be empty or whitespace only")
		}
		contract.Position = trimmed
	}
	if req.WeeklyHours != nil {
		if err := validation.ValidateWeeklyHours(*req.WeeklyHours, "weekly_hours"); err != nil {
			return nil, apperror.BadRequest(err.Error())
		}
		contract.WeeklyHours = *req.WeeklyHours
	}
	if req.Grade != nil {
		contract.Grade = strings.TrimSpace(*req.Grade)
	}
	if req.Step != nil {
		contract.Step = *req.Step
	}
	// Properties can be replaced entirely
	if req.Properties != nil {
		contract.Properties = req.Properties
	}

	// Handle date changes
	newFrom := contract.From
	newTo := contract.To
	if req.From != nil {
		newFrom = *req.From
	}
	if req.To != nil {
		newTo = req.To
	}

	// Validate period
	if err := validation.ValidatePeriod(newFrom, newTo); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Check for overlap if dates changed
	if req.From != nil || req.To != nil {
		if err := s.store.Contracts().ValidateNoOverlap(employeeID, newFrom, newTo, &contractID); err != nil {
			if errors.Is(err, store.ErrContractOverlap) {
				return nil, apperror.Conflict(err.Error())
			}
			return nil, apperror.Internal("failed to validate contract")
		}
		contract.From = newFrom
		contract.To = newTo
	}

	if err := s.store.UpdateContract(contract); err != nil {
		return nil, apperror.Internal("failed to update contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

