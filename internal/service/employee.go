package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
	"github.com/eenemeene/kitamanager-go/internal/validation"
)

// EmployeeService handles business logic for employee operations
type EmployeeService struct {
	store        store.EmployeeStorer
	payPlanStore store.PayPlanStorer
	transactor   store.Transactor
}

// NewEmployeeService creates a new employee service
func NewEmployeeService(store store.EmployeeStorer, payPlanStore store.PayPlanStorer, transactor store.Transactor) *EmployeeService {
	return &EmployeeService{store: store, payPlanStore: payPlanStore, transactor: transactor}
}

// List returns a paginated list of employees
func (s *EmployeeService) List(ctx context.Context, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	employees, total, err := s.store.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch employees")
	}

	responses := make([]models.EmployeeResponse, len(employees))
	for i, emp := range employees {
		responses[i] = emp.ToResponse()
	}
	return responses, total, nil
}

// ListByOrganization returns a paginated list of employees for an organization
func (s *EmployeeService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	return s.ListByOrganizationAndSection(ctx, orgID, nil, nil, "", nil, limit, offset)
}

// ListByOrganizationAndSection returns a paginated list of employees for an organization, optionally filtered by section, active contract date, name search, and/or staff category
func (s *EmployeeService) ListByOrganizationAndSection(ctx context.Context, orgID uint, sectionID *uint, activeOn *time.Time, search string, staffCategory *string, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	employees, total, err := s.store.FindByOrganizationAndSection(ctx, orgID, sectionID, activeOn, search, staffCategory, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch employees")
	}

	responses := make([]models.EmployeeResponse, len(employees))
	for i, emp := range employees {
		responses[i] = emp.ToResponse()
	}
	return responses, total, nil
}

// GetByID returns an employee by ID, validating it belongs to the specified organization
func (s *EmployeeService) GetByID(ctx context.Context, id, orgID uint) (*models.EmployeeResponse, error) {
	employee, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}
	resp := employee.ToResponse()
	return &resp, nil
}

// Create creates a new employee
func (s *EmployeeService) Create(ctx context.Context, orgID uint, req *models.EmployeeCreateRequest) (*models.EmployeeResponse, error) {
	// Trim and validate input
	person, err := validation.ValidatePersonCreate(&validation.PersonCreateFields{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Gender:    req.Gender,
		Birthdate: req.Birthdate,
	})
	if err != nil {
		return nil, err
	}

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: orgID,
			SectionID:      req.SectionID,
			FirstName:      person.FirstName,
			LastName:       person.LastName,
			Gender:         person.Gender,
			Birthdate:      person.Birthdate,
		},
	}

	if err := s.store.Create(ctx, employee); err != nil {
		return nil, apperror.InternalWrap(err, "failed to create employee")
	}

	resp := employee.ToResponse()
	return &resp, nil
}

// Update updates an existing employee, validating it belongs to the specified organization
func (s *EmployeeService) Update(ctx context.Context, id, orgID uint, req *models.EmployeeUpdateRequest) (*models.EmployeeResponse, error) {
	employee, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	if req.FirstName != nil {
		trimmed, err := validation.ValidateAndTrimName(*req.FirstName, "first_name")
		if err != nil {
			return nil, err
		}
		employee.FirstName = trimmed
	}
	if req.LastName != nil {
		trimmed, err := validation.ValidateAndTrimName(*req.LastName, "last_name")
		if err != nil {
			return nil, err
		}
		employee.LastName = trimmed
	}
	if req.Gender != nil {
		if err := validation.ValidateGender(*req.Gender); err != nil {
			return nil, err
		}
		employee.Gender = *req.Gender
	}
	if req.Birthdate != nil {
		bd, err := validation.ParseAndValidateBirthdate(*req.Birthdate)
		if err != nil {
			return nil, err
		}
		employee.Birthdate = bd
	}

	sectionChanged := req.SectionID != nil && !sameSectionID(req.SectionID, employee.SectionID)

	if req.SectionID != nil {
		employee.SectionID = req.SectionID
		// Clear preloaded association so GORM Save doesn't override the foreign key
		employee.Section = nil
	}

	if sectionChanged {
		// Wrap employee update + contract transition in a single transaction
		if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
			if err := s.store.Update(txCtx, employee); err != nil {
				return apperror.InternalWrap(err, "failed to update employee")
			}
			return s.handleEmployeeSectionTransfer(txCtx, id, req.SectionID)
		}); err != nil {
			return nil, err
		}
	} else {
		if err := s.store.Update(ctx, employee); err != nil {
			return nil, apperror.InternalWrap(err, "failed to update employee")
		}
	}

	// Reload to get fresh associations (e.g., new Section after section_id change)
	employee, err = s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to reload employee after update")
	}

	resp := employee.ToResponse()
	return &resp, nil
}

// handleEmployeeSectionTransfer creates a contract transition when an employee moves between sections.
func (s *EmployeeService) handleEmployeeSectionTransfer(ctx context.Context, employeeID uint, newSectionID *uint) error {
	contract, err := s.store.Contracts().GetCurrentContract(ctx, employeeID)
	if err != nil {
		return apperror.InternalWrap(err, "failed to fetch current contract")
	}
	if contract == nil {
		return nil // no active contract, nothing to transition
	}

	// If the contract already has the same section, nothing to do
	if sameSectionID(contract.SectionID, newSectionID) {
		return nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	switch decideSectionTransfer(contract.From) {
	case transferUpdate:
		// Same-day: just update the section on the existing contract
		contract.SectionID = newSectionID
		contract.Section = nil
		return s.store.UpdateContract(ctx, contract)
	case transferReplace:
		// Close old contract (end = yesterday) and create a new one starting today
		yesterday := today.AddDate(0, 0, -1)
		contract.To = &yesterday
		contract.Section = nil
		if err := s.store.UpdateContract(ctx, contract); err != nil {
			return apperror.InternalWrap(err, "failed to close existing contract")
		}

		newContract := &models.EmployeeContract{
			EmployeeID: employeeID,
			BaseContract: models.BaseContract{
				Period: models.Period{
					From: today,
					To:   nil,
				},
				SectionID:  newSectionID,
				Properties: contract.Properties,
			},
			StaffCategory: contract.StaffCategory,
			Grade:         contract.Grade,
			Step:          contract.Step,
			WeeklyHours:   contract.WeeklyHours,
			PayPlanID:     contract.PayPlanID,
		}
		return s.store.CreateContract(ctx, newContract)
	}
	return nil
}

// Delete deletes an employee and its contracts, validating it belongs to the specified organization.
// The ownership check and deletion run in a single transaction.
func (s *EmployeeService) Delete(ctx context.Context, id, orgID uint) error {
	return s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		// Security: Validate employee belongs to the specified organization
		employee, err := s.store.FindByID(txCtx, id)
		if err != nil {
			return apperror.NotFound("employee")
		}
		if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
			return err
		}

		if err := s.store.Delete(txCtx, id); err != nil {
			return apperror.InternalWrap(err, "failed to delete employee")
		}
		return nil
	})
}

// ListContracts returns paginated contract history for an employee, validating it belongs to the specified organization
func (s *EmployeeService) ListContracts(ctx context.Context, employeeID, orgID uint, limit, offset int) ([]models.EmployeeContractResponse, int64, error) {
	// Verify employee exists and belongs to org (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, 0, apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, 0, err
	}

	contracts, total, err := s.store.FindContractsByEmployeePaginated(ctx, employeeID, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch contracts")
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
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	contract, err := s.store.Contracts().GetCurrentContract(ctx, employeeID)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to fetch contract")
	}
	if contract == nil {
		return nil, apperror.NotFound("active contract")
	}
	resp := contract.ToResponse()
	return &resp, nil
}

// CreateContract creates a new contract for an employee, validating it belongs to the specified organization
func (s *EmployeeService) CreateContract(ctx context.Context, employeeID, orgID uint, req *models.EmployeeContractCreateRequest) (*models.EmployeeContractResponse, error) {
	// Validate staff category
	if !models.IsValidStaffCategory(req.StaffCategory) {
		return nil, apperror.BadRequest("staff_category must be one of: qualified, supplementary, non_pedagogical")
	}
	if err := validation.ValidatePeriod(req.From, req.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if err := validation.ValidateWeeklyHours(req.WeeklyHours, "weekly_hours"); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	req.Grade = strings.TrimSpace(req.Grade)

	// Verify employee exists and belongs to org (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	// Validate pay plan exists and belongs to same organization
	payPlan, err := s.payPlanStore.GetByID(ctx, req.PayPlanID)
	if err != nil {
		return nil, apperror.BadRequest("payplan_id not found")
	}
	if payPlan.OrganizationID != orgID {
		return nil, apperror.BadRequest("payplan does not belong to this organization")
	}

	// Auto-set section from employee when not specified in request
	sectionID := req.SectionID
	if sectionID == nil {
		sectionID = employee.SectionID
	}

	contract := &models.EmployeeContract{
		EmployeeID: employeeID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: req.From,
				To:   req.To,
			},
			SectionID:  sectionID,
			Properties: req.Properties,
		},
		StaffCategory: req.StaffCategory,
		Grade:         req.Grade,
		Step:          req.Step,
		WeeklyHours:   req.WeeklyHours,
		PayPlanID:     req.PayPlanID,
	}

	// Validate + create in a single transaction to prevent race conditions
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, employeeID, req.From, req.To, nil); err != nil {
			if errors.Is(err, store.ErrContractOverlap) {
				return apperror.Conflict(err.Error())
			}
			return apperror.InternalWrap(err, "failed to validate contract")
		}
		return s.store.CreateContract(txCtx, contract)
	}); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// DeleteContract deletes a contract, validating it belongs to an employee in the specified organization
func (s *EmployeeService) DeleteContract(ctx context.Context, contractID, employeeID, orgID uint) error {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return err
	}

	// Validate contract belongs to the employee
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return apperror.NotFound("contract")
	}
	if err := verifyContractOwnership(contract, employeeID); err != nil {
		return err
	}

	if err := s.store.DeleteContract(ctx, contractID); err != nil {
		return apperror.InternalWrap(err, "failed to delete contract")
	}
	return nil
}

// GetContractByID returns a contract by ID, validating ownership
func (s *EmployeeService) GetContractByID(ctx context.Context, contractID, employeeID, orgID uint) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	// Get contract
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if err := verifyContractOwnership(contract, employeeID); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// UpdateContract updates an existing contract, validating ownership
func (s *EmployeeService) UpdateContract(ctx context.Context, contractID, employeeID, orgID uint, req *models.EmployeeContractUpdateRequest) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	// Get contract
	contract, err := s.store.FindContractByID(ctx, contractID)
	if err != nil {
		return nil, apperror.NotFound("contract")
	}
	if err := verifyContractOwnership(contract, employeeID); err != nil {
		return nil, err
	}

	// Validate and update pay plan if provided
	if req.PayPlanID != nil {
		payPlan, err := s.payPlanStore.GetByID(ctx, *req.PayPlanID)
		if err != nil {
			return nil, apperror.BadRequest("payplan_id not found")
		}
		if payPlan.OrganizationID != orgID {
			return nil, apperror.BadRequest("payplan does not belong to this organization")
		}
		contract.PayPlanID = *req.PayPlanID
	}

	// Update fields if provided
	if req.StaffCategory != nil {
		if !models.IsValidStaffCategory(*req.StaffCategory) {
			return nil, apperror.BadRequest("staff_category must be one of: qualified, supplementary, non_pedagogical")
		}
		contract.StaffCategory = *req.StaffCategory
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

	contract.From = newFrom
	contract.To = newTo

	// Validate + update in a single transaction to prevent race conditions
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if req.From != nil || req.To != nil {
			if err := s.store.Contracts().ValidateNoOverlap(txCtx, employeeID, newFrom, newTo, &contractID); err != nil {
				if errors.Is(err, store.ErrContractOverlap) {
					return apperror.Conflict(err.Error())
				}
				return apperror.InternalWrap(err, "failed to validate contract")
			}
		}
		return s.store.UpdateContract(txCtx, contract)
	}); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}
