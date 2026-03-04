package service

import (
	"context"
	"errors"
	"fmt"

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
	return personList(ctx, s.store.FindAll, (*models.Employee).ToResponse, "employees", limit, offset)
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
	return personGetByID(ctx, s.store.FindByIDAndOrg, (*models.Employee).ToResponse, id, orgID, "employee")
}

// Create creates a new employee
func (s *EmployeeService) Create(ctx context.Context, orgID uint, req *models.EmployeeCreateRequest) (*models.EmployeeResponse, error) {
	return personCreate(ctx,
		&validation.PersonCreateFields{FirstName: req.FirstName, LastName: req.LastName, Gender: req.Gender, Birthdate: req.Birthdate},
		func(p models.Person) *models.Employee { return &models.Employee{Person: p} },
		s.store.Create, (*models.Employee).ToResponse, orgID, "employee")
}

// Update updates an existing employee, validating it belongs to the specified organization
func (s *EmployeeService) Update(ctx context.Context, id, orgID uint, req *models.EmployeeUpdateRequest) (*models.EmployeeResponse, error) {
	return personUpdate(ctx, s.transactor, s.store.FindByIDAndOrg, func(e *models.Employee) *models.Person { return &e.Person },
		s.store.Update, (*models.Employee).ToResponse, id, orgID,
		personUpdateFields{FirstName: req.FirstName, LastName: req.LastName, Gender: req.Gender, Birthdate: req.Birthdate},
		"employee")
}

// Delete deletes an employee and its contracts, validating it belongs to the specified organization.
// The ownership check and deletion run in a single transaction.
func (s *EmployeeService) Delete(ctx context.Context, id, orgID uint) error {
	return personDelete(ctx, s.transactor, s.store.FindByIDMinimalAndOrg, s.store.Delete, id, orgID, "employee")
}

// Import creates or updates employees with their contracts from an EmployeeImportExportData.
// Employees are matched by (first_name, last_name, birthdate, org_id).
// On match, existing contracts are replaced. Sections are auto-created if missing.
// Pay plans must already exist (looked up by name).
func (s *EmployeeService) Import(ctx context.Context, orgID uint, data *models.EmployeeImportExportData) ([]models.EmployeeResponse, error) {
	if len(data.Employees) == 0 {
		return nil, apperror.BadRequest("no employees in import data")
	}

	// Cache for resolved section and pay plan names → IDs within this import.
	sectionCache := map[string]uint{}
	payPlanCache := map[string]uint{}

	var results []models.EmployeeResponse

	if err := s.transactor.InTransaction(ctx, func(txCtx context.Context) error {
		for i, emp := range data.Employees {
			employeeID, err := personImportUpsert(txCtx,
				importPersonItem{Index: i + 1, FirstName: emp.FirstName, LastName: emp.LastName, Gender: emp.Gender, Birthdate: emp.Birthdate},
				"employee",
				s.store.FindByNameBirthdateAndOrg,
				func(e *models.Employee) *models.Person { return &e.Person },
				func(e *models.Employee) uint { return e.ID },
				s.store.Update, s.store.DeleteContractsByEmployee,
				func(p models.Person) *models.Employee { return &models.Employee{Person: p} },
				s.store.Create, orgID,
			)
			if err != nil {
				return err
			}

			// Create contracts.
			for j, c := range emp.Contracts {
				if c.From.IsZero() {
					return apperror.BadRequest(fmt.Sprintf("employee %d contract %d: from date is required", i+1, j+1))
				}

				// Resolve section by name.
				sectionID, err := resolveSection(txCtx, s.sectionStore, c.SectionName, orgID, sectionCache)
				if err != nil {
					return err
				}

				// Resolve pay plan by name.
				payPlanID, err := s.resolvePayPlan(txCtx, c.PayPlanName, orgID, payPlanCache)
				if err != nil {
					return apperror.BadRequest(fmt.Sprintf("employee %d contract %d: %s", i+1, j+1, err.Error()))
				}

				req := &models.EmployeeContractCreateRequest{
					From:          c.From,
					To:            c.To,
					SectionID:     sectionID,
					StaffCategory: c.StaffCategory,
					Grade:         c.Grade,
					Step:          c.Step,
					WeeklyHours:   c.WeeklyHours,
					PayPlanID:     payPlanID,
					Properties:    c.Properties,
				}
				if _, err := s.CreateContract(txCtx, employeeID, orgID, req); err != nil {
					return err
				}
			}

			// Re-fetch with preloads for the response.
			fetched, err := s.store.FindByIDAndOrg(txCtx, employeeID, orgID)
			if err != nil {
				return apperror.InternalWrap(err, "failed to fetch imported employee")
			}
			results = append(results, fetched.ToResponse())
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return results, nil
}

// resolvePayPlan looks up a pay plan by name (with caching). Returns an error if not found.
func (s *EmployeeService) resolvePayPlan(ctx context.Context, payPlanName *string, orgID uint, cache map[string]uint) (uint, error) {
	if payPlanName == nil || *payPlanName == "" {
		return 0, fmt.Errorf("pay_plan_name is required")
	}
	name := *payPlanName
	if id, ok := cache[name]; ok {
		return id, nil
	}
	pp, err := s.payPlanStore.FindByNameAndOrg(ctx, name, orgID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return 0, fmt.Errorf("pay plan %q not found in organization", name)
		}
		return 0, fmt.Errorf("failed to look up pay plan: %w", err)
	}
	cache[name] = pp.ID
	return pp.ID, nil
}

// FindAllByOrganization returns all employees for an organization (no pagination), with contracts preloaded.
func (s *EmployeeService) FindAllByOrganization(ctx context.Context, orgID uint) ([]models.EmployeeResponse, error) {
	return fetchAllPaginated(ctx,
		func(ctx context.Context, limit, offset int) ([]models.Employee, int64, error) {
			return s.store.FindByOrganizationAndSection(ctx, orgID, nil, nil, "", nil, limit, offset)
		},
		(*models.Employee).ToResponse, "employees")
}
