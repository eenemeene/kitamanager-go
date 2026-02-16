package service

import (
	"context"

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
