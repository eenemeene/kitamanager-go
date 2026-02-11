package store

import (
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type EmployeeStore struct {
	db            *gorm.DB
	contractStore *PeriodStore[models.EmployeeContract]
}

func NewEmployeeStore(db *gorm.DB) *EmployeeStore {
	return &EmployeeStore{
		db:            db,
		contractStore: NewPeriodStore[models.EmployeeContract](db, "employee_id"),
	}
}

func (s *EmployeeStore) FindAll(limit, offset int) ([]models.Employee, int64, error) {
	var employees []models.Employee
	var total int64

	if err := s.db.Model(&models.Employee{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Limit(limit).Offset(offset).Find(&employees).Error; err != nil {
		return nil, 0, err
	}

	return employees, total, nil
}

func (s *EmployeeStore) FindByOrganization(orgID uint, limit, offset int) ([]models.Employee, int64, error) {
	return s.FindByOrganizationAndSection(orgID, nil, nil, "", limit, offset)
}

func (s *EmployeeStore) FindByOrganizationAndSection(orgID uint, sectionID *uint, activeOn *time.Time, search string, limit, offset int) ([]models.Employee, int64, error) {
	var employees []models.Employee
	var total int64

	// Count query
	countQuery := s.db.Model(&models.Employee{}).Where("employees.organization_id = ?", orgID)
	if sectionID != nil {
		countQuery = countQuery.Where("employees.section_id = ?", *sectionID)
	}
	if search != "" {
		countQuery = countQuery.Scopes(PersonNameSearch("employees", search))
	}
	if activeOn != nil {
		countQuery = countQuery.
			Joins("JOIN employee_contracts ON employee_contracts.employee_id = employees.id").
			Scopes(PeriodActiveOn("employee_contracts.from_date", "employee_contracts.to_date", *activeOn)).
			Distinct("employees.id")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := s.db.Preload("Contracts").Preload("Section").Where("employees.organization_id = ?", orgID)
	if sectionID != nil {
		dataQuery = dataQuery.Where("employees.section_id = ?", *sectionID)
	}
	if search != "" {
		dataQuery = dataQuery.Scopes(PersonNameSearch("employees", search))
	}
	if activeOn != nil {
		dataQuery = dataQuery.
			Joins("JOIN employee_contracts ON employee_contracts.employee_id = employees.id").
			Scopes(PeriodActiveOn("employee_contracts.from_date", "employee_contracts.to_date", *activeOn)).
			Distinct()
	}
	if err := dataQuery.Limit(limit).Offset(offset).Find(&employees).Error; err != nil {
		return nil, 0, err
	}

	return employees, total, nil
}

// Contracts returns the contract store for employees
func (s *EmployeeStore) Contracts() ContractStorer[models.EmployeeContract] {
	return s.contractStore
}

func (s *EmployeeStore) FindByID(id uint) (*models.Employee, error) {
	var employee models.Employee
	if err := s.db.Preload("Organization").Preload("Section").Preload("Contracts").First(&employee, id).Error; err != nil {
		return nil, err
	}
	return &employee, nil
}

// FindByIDMinimal returns an employee without preloading relationships.
// Useful for existence checks and org validation where relationships aren't needed.
func (s *EmployeeStore) FindByIDMinimal(id uint) (*models.Employee, error) {
	var employee models.Employee
	if err := s.db.First(&employee, id).Error; err != nil {
		return nil, err
	}
	return &employee, nil
}

func (s *EmployeeStore) Create(employee *models.Employee) error {
	return s.db.Create(employee).Error
}

func (s *EmployeeStore) Update(employee *models.Employee) error {
	return s.db.Save(employee).Error
}

func (s *EmployeeStore) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("employee_id = ?", id).Delete(&models.EmployeeContract{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Employee{}, id).Error
	})
}

func (s *EmployeeStore) CreateContract(contract *models.EmployeeContract) error {
	return s.db.Create(contract).Error
}

func (s *EmployeeStore) FindContractByID(id uint) (*models.EmployeeContract, error) {
	var contract models.EmployeeContract
	if err := s.db.First(&contract, id).Error; err != nil {
		return nil, err
	}
	return &contract, nil
}

func (s *EmployeeStore) UpdateContract(contract *models.EmployeeContract) error {
	return s.db.Save(contract).Error
}

func (s *EmployeeStore) DeleteContract(id uint) error {
	return s.db.Delete(&models.EmployeeContract{}, id).Error
}
