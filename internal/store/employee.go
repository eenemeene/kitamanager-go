package store

import (
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
	var employees []models.Employee
	var total int64

	if err := s.db.Model(&models.Employee{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Where("organization_id = ?", orgID).Limit(limit).Offset(offset).Find(&employees).Error; err != nil {
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
	if err := s.db.Preload("Organization").Preload("Contracts").First(&employee, id).Error; err != nil {
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
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete all properties first
		if err := tx.Where("contract_id = ?", id).Delete(&models.EmployeeContractProperty{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.EmployeeContract{}, id).Error
	})
}

// FindContractByIDWithProperties returns a contract with its properties preloaded
func (s *EmployeeStore) FindContractByIDWithProperties(id uint) (*models.EmployeeContract, error) {
	var contract models.EmployeeContract
	if err := s.db.Preload("Properties").First(&contract, id).Error; err != nil {
		return nil, err
	}
	return &contract, nil
}

// FindPropertyByID returns a contract property by ID
func (s *EmployeeStore) FindPropertyByID(id uint) (*models.EmployeeContractProperty, error) {
	var property models.EmployeeContractProperty
	if err := s.db.First(&property, id).Error; err != nil {
		return nil, err
	}
	return &property, nil
}

// FindPropertiesByContractID returns all properties for a contract
func (s *EmployeeStore) FindPropertiesByContractID(contractID uint) ([]models.EmployeeContractProperty, error) {
	var properties []models.EmployeeContractProperty
	if err := s.db.Where("contract_id = ?", contractID).Find(&properties).Error; err != nil {
		return nil, err
	}
	return properties, nil
}

// PropertyExistsByName checks if a property with the given name exists for a contract
func (s *EmployeeStore) PropertyExistsByName(contractID uint, name string) (bool, error) {
	var count int64
	if err := s.db.Model(&models.EmployeeContractProperty{}).
		Where("contract_id = ? AND name = ?", contractID, name).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateProperty creates a new contract property
func (s *EmployeeStore) CreateProperty(property *models.EmployeeContractProperty) error {
	return s.db.Create(property).Error
}

// UpdateProperty updates an existing contract property
func (s *EmployeeStore) UpdateProperty(property *models.EmployeeContractProperty) error {
	return s.db.Save(property).Error
}

// DeleteProperty deletes a contract property
func (s *EmployeeStore) DeleteProperty(id uint) error {
	return s.db.Delete(&models.EmployeeContractProperty{}, id).Error
}
