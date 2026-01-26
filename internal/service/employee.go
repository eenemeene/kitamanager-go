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
	employees, total, err := s.store.FindByOrganization(orgID, limit, offset)
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
	if err := validation.ValidateBirthdate(req.Birthdate); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: orgID,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
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
	if req.Birthdate != nil {
		if err := validation.ValidateBirthdate(*req.Birthdate); err != nil {
			return nil, apperror.BadRequest(err.Error())
		}
		employee.Birthdate = *req.Birthdate
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

// ListContracts returns contract history for an employee, validating it belongs to the specified organization
func (s *EmployeeService) ListContracts(ctx context.Context, employeeID, orgID uint) ([]models.EmployeeContractResponse, error) {
	// Verify employee exists and belongs to org
	employee, err := s.store.FindByID(employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	// Security: Validate employee belongs to the specified organization
	if employee.OrganizationID != orgID {
		return nil, apperror.NotFound("employee")
	}

	contracts, err := s.store.Contracts().GetHistory(employeeID)
	if err != nil {
		return nil, apperror.Internal("failed to fetch contracts")
	}

	responses := make([]models.EmployeeContractResponse, len(contracts))
	for i, c := range contracts {
		responses[i] = c.ToResponse()
	}
	return responses, nil
}

// GetCurrentContract returns the current active contract for an employee, validating it belongs to the specified organization
func (s *EmployeeService) GetCurrentContract(ctx context.Context, employeeID, orgID uint) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization
	employee, err := s.store.FindByID(employeeID)
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
	if err := validation.ValidateSalary(req.Salary); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Verify employee exists and belongs to org
	employee, err := s.store.FindByID(employeeID)
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
		Period: models.Period{
			From: req.From,
			To:   req.To,
		},
		Position:    req.Position,
		WeeklyHours: req.WeeklyHours,
		Salary:      req.Salary,
	}

	if err := s.store.CreateContract(contract); err != nil {
		return nil, apperror.Internal("failed to create contract")
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// DeleteContract deletes a contract, validating it belongs to an employee in the specified organization
func (s *EmployeeService) DeleteContract(ctx context.Context, contractID, employeeID, orgID uint) error {
	// Security: Validate employee belongs to the specified organization
	employee, err := s.store.FindByID(employeeID)
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
