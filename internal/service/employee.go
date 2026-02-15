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
	sectionStore store.SectionStorer
	transactor   store.Transactor
}

// NewEmployeeService creates a new employee service
func NewEmployeeService(store store.EmployeeStorer, payPlanStore store.PayPlanStorer, sectionStore store.SectionStorer, transactor store.Transactor) *EmployeeService {
	return &EmployeeService{store: store, payPlanStore: payPlanStore, sectionStore: sectionStore, transactor: transactor}
}

// List returns a paginated list of employees
func (s *EmployeeService) List(ctx context.Context, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	employees, total, err := s.store.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch employees")
	}

	return toResponseList(employees, (*models.Employee).ToResponse), total, nil
}

// ListByOrganization returns a paginated list of employees for an organization
func (s *EmployeeService) ListByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	return s.ListByOrganizationAndSection(ctx, orgID, models.EmployeeListFilter{}, limit, offset)
}

// ListByOrganizationAndSection returns a paginated list of employees for an organization,
// optionally filtered by section, active contract date, name search, and/or staff category.
func (s *EmployeeService) ListByOrganizationAndSection(ctx context.Context, orgID uint, filter models.EmployeeListFilter, limit, offset int) ([]models.EmployeeResponse, int64, error) {
	if err := filter.Validate(); err != nil {
		return nil, 0, apperror.BadRequest(err.Error())
	}

	employees, total, err := s.store.FindByOrganizationAndSection(ctx, orgID, filter.SectionID, filter.ActiveOn, filter.Search, filter.StaffCategory, limit, offset)
	if err != nil {
		return nil, 0, apperror.InternalWrap(err, "failed to fetch employees")
	}

	return toResponseList(employees, (*models.Employee).ToResponse), total, nil
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

	if err := s.store.Update(ctx, employee); err != nil {
		return nil, apperror.InternalWrap(err, "failed to update employee")
	}

	// Reload to get fresh associations
	employee, err = s.store.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.InternalWrap(err, "failed to reload employee after update")
	}

	resp := employee.ToResponse()
	return &resp, nil
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

	return toResponseList(contracts, (*models.EmployeeContract).ToResponse), total, nil
}

// GetCurrentRecord returns the current active contract for an employee, validating it belongs to the specified organization
func (s *EmployeeService) GetCurrentRecord(ctx context.Context, employeeID, orgID uint) (*models.EmployeeContractResponse, error) {
	// Security: Validate employee belongs to the specified organization (use minimal query - no preloads needed)
	employee, err := s.store.FindByIDMinimal(ctx, employeeID)
	if err != nil {
		return nil, apperror.NotFound("employee")
	}
	if err := verifyOrgOwnership(employee, orgID, "employee"); err != nil {
		return nil, err
	}

	contract, err := s.store.Contracts().GetCurrentRecord(ctx, employeeID)
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
	payPlan, err := s.payPlanStore.FindByID(ctx, req.PayPlanID)
	if err != nil {
		return nil, apperror.BadRequest("payplan_id not found")
	}
	if payPlan.OrganizationID != orgID {
		return nil, apperror.BadRequest("payplan does not belong to this organization")
	}

	// Validate section belongs to the same organization
	section, err := s.sectionStore.FindByID(ctx, req.SectionID)
	if err != nil {
		return nil, apperror.BadRequest("section not found")
	}
	if section.OrganizationID != orgID {
		return nil, apperror.BadRequest("section does not belong to this organization")
	}

	contract := &models.EmployeeContract{
		EmployeeID: employeeID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: req.From,
				To:   req.To,
			},
			SectionID:  req.SectionID,
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
			if errors.Is(err, store.ErrPeriodOverlap) {
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
	if err := verifyRecordOwnership(contract, employeeID, "contract"); err != nil {
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
	if err := verifyRecordOwnership(contract, employeeID, "contract"); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// UpdateContract updates an existing contract with amend semantics.
// - Contract started today or in the future → update in place
// - Contract started before today → end old contract (yesterday), create new contract (today) with changes
// - Contract already ended → reject with 400
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
	if err := verifyRecordOwnership(contract, employeeID, "contract"); err != nil {
		return nil, err
	}

	// Determine amend mode
	mode, err := determineAmendMode(contract.From, contract.To)
	if err != nil {
		return nil, err
	}

	// Validate pay plan if provided (applies to both modes)
	if req.PayPlanID != nil {
		payPlan, err := s.payPlanStore.FindByID(ctx, *req.PayPlanID)
		if err != nil {
			return nil, apperror.BadRequest("payplan_id not found")
		}
		if payPlan.OrganizationID != orgID {
			return nil, apperror.BadRequest("payplan does not belong to this organization")
		}
	}

	// Validate staff category if provided
	if req.StaffCategory != nil {
		if !models.IsValidStaffCategory(*req.StaffCategory) {
			return nil, apperror.BadRequest("staff_category must be one of: qualified, supplementary, non_pedagogical")
		}
	}

	// Validate weekly hours if provided
	if req.WeeklyHours != nil {
		if err := validation.ValidateWeeklyHours(*req.WeeklyHours, "weekly_hours"); err != nil {
			return nil, apperror.BadRequest(err.Error())
		}
	}

	// Validate section if provided (applies to both modes)
	if req.SectionID != nil {
		section, err := s.sectionStore.FindByID(ctx, *req.SectionID)
		if err != nil {
			return nil, apperror.BadRequest("section not found")
		}
		if section.OrganizationID != orgID {
			return nil, apperror.BadRequest("section does not belong to this organization")
		}
	}

	switch mode {
	case amendModeInPlace:
		return s.updateContractInPlace(ctx, contract, employeeID, req)
	case amendModeAmend:
		return s.amendContract(ctx, contract, employeeID, req)
	default:
		return nil, apperror.Internal("unexpected amend mode")
	}
}

// updateContractInPlace applies changes directly to the existing employee contract.
func (s *EmployeeService) updateContractInPlace(ctx context.Context, contract *models.EmployeeContract, employeeID uint, req *models.EmployeeContractUpdateRequest) (*models.EmployeeContractResponse, error) {
	if req.PayPlanID != nil {
		contract.PayPlanID = *req.PayPlanID
	}
	if req.StaffCategory != nil {
		contract.StaffCategory = *req.StaffCategory
	}
	if req.WeeklyHours != nil {
		contract.WeeklyHours = *req.WeeklyHours
	}
	if req.Grade != nil {
		contract.Grade = strings.TrimSpace(*req.Grade)
	}
	if req.Step != nil {
		contract.Step = *req.Step
	}
	if req.SectionID != nil {
		contract.SectionID = *req.SectionID
		contract.Section = nil
	}
	if req.Properties != nil {
		contract.Properties = req.Properties
	}
	if req.From != nil {
		contract.From = *req.From
	}
	if req.To != nil {
		contract.To = req.To
	}

	if err := validation.ValidatePeriod(contract.From, contract.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	contractID := contract.ID
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, employeeID, contract.From, contract.To, &contractID); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict(err.Error())
			}
			return apperror.InternalWrap(err, "failed to validate contract")
		}
		return s.store.UpdateContract(txCtx, contract)
	}); err != nil {
		return nil, err
	}

	resp := contract.ToResponse()
	return &resp, nil
}

// amendContract closes the old employee contract and creates a new one with changes applied.
func (s *EmployeeService) amendContract(ctx context.Context, contract *models.EmployeeContract, employeeID uint, req *models.EmployeeContractUpdateRequest) (*models.EmployeeContractResponse, error) {
	today := truncateToDate(time.Now())
	yesterday := today.AddDate(0, 0, -1)

	// Clone contract with current values, new contract starts today
	newContract := &models.EmployeeContract{
		EmployeeID: contract.EmployeeID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: today,
				To:   contract.To, // inherit original To
			},
			SectionID:  contract.SectionID,
			Properties: contract.Properties,
		},
		StaffCategory: contract.StaffCategory,
		Grade:         contract.Grade,
		Step:          contract.Step,
		WeeklyHours:   contract.WeeklyHours,
		PayPlanID:     contract.PayPlanID,
	}

	// Apply changes (From is ignored — always today; To is applied if provided)
	if req.SectionID != nil {
		newContract.SectionID = *req.SectionID
	}
	if req.Properties != nil {
		newContract.Properties = req.Properties
	}
	if req.To != nil {
		newContract.To = req.To
	}
	if req.PayPlanID != nil {
		newContract.PayPlanID = *req.PayPlanID
	}
	if req.StaffCategory != nil {
		newContract.StaffCategory = *req.StaffCategory
	}
	if req.WeeklyHours != nil {
		newContract.WeeklyHours = *req.WeeklyHours
	}
	if req.Grade != nil {
		newContract.Grade = strings.TrimSpace(*req.Grade)
	}
	if req.Step != nil {
		newContract.Step = *req.Step
	}

	if err := validation.ValidatePeriod(newContract.From, newContract.To); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	// Transaction: close old contract + validate overlap + create new
	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		// Close old contract (end = yesterday)
		contract.To = &yesterday
		if err := s.store.UpdateContract(txCtx, contract); err != nil {
			return apperror.InternalWrap(err, "failed to close old contract")
		}

		// Validate new contract doesn't overlap with any other contracts
		if err := s.store.Contracts().ValidateNoOverlap(txCtx, employeeID, newContract.From, newContract.To, nil); err != nil {
			if errors.Is(err, store.ErrPeriodOverlap) {
				return apperror.Conflict(err.Error())
			}
			return apperror.InternalWrap(err, "failed to validate contract")
		}

		// Create new contract
		return s.store.CreateContract(txCtx, newContract)
	}); err != nil {
		return nil, err
	}

	resp := newContract.ToResponse()
	return &resp, nil
}
