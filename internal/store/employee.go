package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type EmployeeStore struct {
	db        *gorm.DB
	Contracts *PeriodStore[models.EmployeeContract]
}

func NewEmployeeStore(db *gorm.DB) *EmployeeStore {
	return &EmployeeStore{
		db:        db,
		Contracts: NewPeriodStore[models.EmployeeContract](db, "employee_id"),
	}
}

func (s *EmployeeStore) FindAll() ([]models.Employee, error) {
	var employees []models.Employee
	if err := s.db.Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

func (s *EmployeeStore) FindByOrganization(orgID uint) ([]models.Employee, error) {
	var employees []models.Employee
	if err := s.db.Where("organization_id = ?", orgID).Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

func (s *EmployeeStore) FindByID(id uint) (*models.Employee, error) {
	var employee models.Employee
	if err := s.db.Preload("Organization").Preload("Contracts").First(&employee, id).Error; err != nil {
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
