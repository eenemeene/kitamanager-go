package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type EmployeeStore struct {
	db          *gorm.DB
	periodStore *PeriodStore[models.EmployeeContract]
}

func NewEmployeeStore(db *gorm.DB) *EmployeeStore {
	return &EmployeeStore{
		db:          db,
		periodStore: NewPeriodStore[models.EmployeeContract](db, "employee_id"),
	}
}

func (s *EmployeeStore) FindAll(ctx context.Context, limit, offset int) ([]models.Employee, int64, error) {
	var employees []models.Employee
	var total int64

	if err := DBFromContext(ctx, s.db).Model(&models.Employee{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := DBFromContext(ctx, s.db).Limit(limit).Offset(offset).Find(&employees).Error; err != nil {
		return nil, 0, err
	}

	return employees, total, nil
}

func (s *EmployeeStore) FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.Employee, int64, error) {
	return s.FindByOrganizationAndSection(ctx, orgID, nil, nil, "", nil, limit, offset)
}

// applyListFilters adds WHERE/JOIN clauses for the employee list filters.
// Returns the modified query and whether DISTINCT is needed (due to JOINs).
func (s *EmployeeStore) applyListFilters(query *gorm.DB, orgID uint, sectionID *uint, activeOn *time.Time, search string, staffCategory *string) (*gorm.DB, bool) {
	needsDistinct := false
	query = query.Where("employees.organization_id = ?", orgID)
	if search != "" {
		query = query.Scopes(PersonNameSearch("employees", search))
	}

	// Section filtering is on contracts, so section_id requires a contract JOIN
	if staffCategory != nil {
		query = query.
			Joins("JOIN employee_contracts ec_cat ON ec_cat.employee_id = employees.id").
			Where("ec_cat.staff_category = ?", *staffCategory)
		if sectionID != nil {
			query = query.Where("ec_cat.section_id = ?", *sectionID)
		}
		if activeOn != nil {
			query = query.Scopes(PeriodActiveOn("ec_cat.from_date", "ec_cat.to_date", *activeOn))
		}
		needsDistinct = true
	} else if sectionID != nil || activeOn != nil {
		query = query.
			Joins("JOIN employee_contracts ON employee_contracts.employee_id = employees.id")
		if sectionID != nil {
			query = query.Where("employee_contracts.section_id = ?", *sectionID)
		}
		if activeOn != nil {
			query = query.Scopes(PeriodActiveOn("employee_contracts.from_date", "employee_contracts.to_date", *activeOn))
		}
		needsDistinct = true
	}
	return query, needsDistinct
}

func (s *EmployeeStore) FindByOrganizationAndSection(ctx context.Context, orgID uint, sectionID *uint, activeOn *time.Time, search string, staffCategory *string, limit, offset int) ([]models.Employee, int64, error) {
	var employees []models.Employee
	var total int64

	countQuery, needsDistinct := s.applyListFilters(
		DBFromContext(ctx, s.db).Model(&models.Employee{}),
		orgID, sectionID, activeOn, search, staffCategory,
	)
	if needsDistinct {
		countQuery = countQuery.Distinct("employees.id")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery, needsDistinct := s.applyListFilters(
		DBFromContext(ctx, s.db).Preload("Contracts").Preload("Contracts.Section").Preload("Contracts.PayPlan"),
		orgID, sectionID, activeOn, search, staffCategory,
	)
	if needsDistinct {
		dataQuery = dataQuery.Distinct()
	}
	if err := dataQuery.Order("employees.first_name ASC, employees.last_name ASC").Limit(limit).Offset(offset).Find(&employees).Error; err != nil {
		return nil, 0, err
	}

	return employees, total, nil
}

// Contracts returns the contract store for employees
func (s *EmployeeStore) Contracts() PeriodStorer[models.EmployeeContract] {
	return s.periodStore
}

func (s *EmployeeStore) FindByID(ctx context.Context, id uint) (*models.Employee, error) {
	var employee models.Employee
	if err := DBFromContext(ctx, s.db).Preload("Organization").Preload("Contracts.Section").Preload("Contracts.PayPlan").Preload("Contracts").First(&employee, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &employee, nil
}

// FindByIDAndOrg returns an employee by ID with full preloads, scoped to the given organization.
func (s *EmployeeStore) FindByIDAndOrg(ctx context.Context, id, orgID uint) (*models.Employee, error) {
	var employee models.Employee
	if err := DBFromContext(ctx, s.db).Preload("Organization").Preload("Contracts.Section").Preload("Contracts.PayPlan").Preload("Contracts").
		Where("id = ? AND organization_id = ?", id, orgID).First(&employee).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &employee, nil
}

// FindByIDMinimal returns an employee without preloading relationships.
// Useful for existence checks and org validation where relationships aren't needed.
func (s *EmployeeStore) FindByIDMinimal(ctx context.Context, id uint) (*models.Employee, error) {
	var employee models.Employee
	if err := DBFromContext(ctx, s.db).First(&employee, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &employee, nil
}

// FindByIDMinimalAndOrg returns an employee without preloading, scoped to the given organization.
func (s *EmployeeStore) FindByIDMinimalAndOrg(ctx context.Context, id, orgID uint) (*models.Employee, error) {
	var employee models.Employee
	if err := DBFromContext(ctx, s.db).Where("id = ? AND organization_id = ?", id, orgID).First(&employee).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &employee, nil
}

func (s *EmployeeStore) Create(ctx context.Context, employee *models.Employee) error {
	return DBFromContext(ctx, s.db).Create(employee).Error
}

func (s *EmployeeStore) Update(ctx context.Context, employee *models.Employee) error {
	return DBFromContext(ctx, s.db).Save(employee).Error
}

func (s *EmployeeStore) Delete(ctx context.Context, id uint) error {
	db := DBFromContext(ctx, s.db)
	if err := db.Where("employee_id = ?", id).Delete(&models.EmployeeContract{}).Error; err != nil {
		return err
	}
	return db.Delete(&models.Employee{}, id).Error
}

func (s *EmployeeStore) CreateContract(ctx context.Context, contract *models.EmployeeContract) error {
	return DBFromContext(ctx, s.db).Create(contract).Error
}

func (s *EmployeeStore) FindContractByID(ctx context.Context, id uint) (*models.EmployeeContract, error) {
	var contract models.EmployeeContract
	if err := DBFromContext(ctx, s.db).Preload("Section").First(&contract, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &contract, nil
}

// FindContractsByEmployeePaginated returns paginated contracts for an employee with Section preloaded.
func (s *EmployeeStore) FindContractsByEmployeePaginated(ctx context.Context, employeeID uint, limit, offset int) ([]models.EmployeeContract, int64, error) {
	var contracts []models.EmployeeContract
	var total int64

	db := DBFromContext(ctx, s.db)
	if err := db.Model(&models.EmployeeContract{}).Where("employee_id = ?", employeeID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("Section").Where("employee_id = ?", employeeID).
		Order("from_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&contracts).Error
	return contracts, total, err
}

func (s *EmployeeStore) UpdateContract(ctx context.Context, contract *models.EmployeeContract) error {
	return DBFromContext(ctx, s.db).Save(contract).Error
}

func (s *EmployeeStore) DeleteContract(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.EmployeeContract{}, id).Error
}

// FindContractsByOrganizationInDateRange returns employee contracts for an organization
// where the contract overlaps with the given date range, filtered by staff categories.
func (s *EmployeeStore) FindContractsByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time, staffCategories []string, sectionID *uint) ([]models.EmployeeContract, error) {
	var contracts []models.EmployeeContract

	query := DBFromContext(ctx, s.db).
		Joins("JOIN employees ON employees.id = employee_contracts.employee_id").
		Where("employees.organization_id = ?", orgID).
		Where("employee_contracts.from_date <= ?", rangeEnd).
		Where("employee_contracts.to_date IS NULL OR employee_contracts.to_date >= ?", rangeStart)

	if len(staffCategories) > 0 {
		query = query.Where("employee_contracts.staff_category IN ?", staffCategories)
	}

	if sectionID != nil {
		query = query.Where("employee_contracts.section_id = ?", *sectionID)
	}

	if err := query.Find(&contracts).Error; err != nil {
		return nil, err
	}
	return contracts, nil
}

// FindByOrganizationInDateRange returns employees that have contracts overlapping the given date range.
// Employees are returned with their contracts preloaded (only those overlapping the range).
// Optional sectionID filters on the contract's section.
func (s *EmployeeStore) FindByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time, sectionID *uint) ([]models.Employee, error) {
	var employees []models.Employee

	query := DBFromContext(ctx, s.db).
		Preload("Contracts", func(db *gorm.DB) *gorm.DB {
			q := db.Where("from_date <= ? AND (to_date IS NULL OR to_date >= ?)", rangeEnd, rangeStart)
			if sectionID != nil {
				q = q.Where("section_id = ?", *sectionID)
			}
			return q
		}).
		Joins("JOIN employee_contracts ON employee_contracts.employee_id = employees.id").
		Where("employees.organization_id = ?", orgID).
		Where("employee_contracts.from_date <= ?", rangeEnd).
		Where("employee_contracts.to_date IS NULL OR employee_contracts.to_date >= ?", rangeStart)

	if sectionID != nil {
		query = query.Where("employee_contracts.section_id = ?", *sectionID)
	}

	if err := query.Distinct().Find(&employees).Error; err != nil {
		return nil, err
	}

	return employees, nil
}

// FindByNameBirthdateAndOrg looks up an employee by first name, last name, birthdate and organization.
func (s *EmployeeStore) FindByNameBirthdateAndOrg(ctx context.Context, firstName, lastName string, birthdate time.Time, orgID uint) (*models.Employee, error) {
	var employee models.Employee
	if err := DBFromContext(ctx, s.db).
		Where("first_name = ? AND last_name = ? AND birthdate = ? AND organization_id = ?", firstName, lastName, birthdate, orgID).
		First(&employee).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &employee, nil
}

// DeleteContractsByEmployee deletes all contracts for the given employee.
func (s *EmployeeStore) DeleteContractsByEmployee(ctx context.Context, employeeID uint) error {
	return DBFromContext(ctx, s.db).Where("employee_id = ?", employeeID).Delete(&models.EmployeeContract{}).Error
}

// FindByOrganizationWithContracts fetches employees in an org who have an
// active contract on `date`, with ALL their contracts preloaded (for seniority calculation).
func (s *EmployeeStore) FindByOrganizationWithContracts(ctx context.Context, orgID uint, date time.Time) ([]models.Employee, error) {
	var employees []models.Employee
	err := DBFromContext(ctx, s.db).
		Preload("Contracts", func(db *gorm.DB) *gorm.DB {
			return db.Order("employee_contracts.from_date ASC")
		}).
		Joins("JOIN employee_contracts ON employee_contracts.employee_id = employees.id").
		Where("employees.organization_id = ?", orgID).
		Scopes(PeriodActiveOn("employee_contracts.from_date", "employee_contracts.to_date", date)).
		Distinct().
		Find(&employees).Error
	if err != nil {
		return nil, err
	}
	return employees, nil
}
