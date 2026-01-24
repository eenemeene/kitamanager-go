package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type ChildStore struct {
	db        *gorm.DB
	Contracts *PeriodStore[models.ChildContract]
}

func NewChildStore(db *gorm.DB) *ChildStore {
	return &ChildStore{
		db:        db,
		Contracts: NewPeriodStore[models.ChildContract](db, "child_id"),
	}
}

func (s *ChildStore) FindAll() ([]models.Child, error) {
	var children []models.Child
	if err := s.db.Find(&children).Error; err != nil {
		return nil, err
	}
	return children, nil
}

func (s *ChildStore) FindByOrganization(orgID uint) ([]models.Child, error) {
	var children []models.Child
	if err := s.db.Where("organization_id = ?", orgID).Find(&children).Error; err != nil {
		return nil, err
	}
	return children, nil
}

func (s *ChildStore) FindByID(id uint) (*models.Child, error) {
	var child models.Child
	if err := s.db.Preload("Organization").Preload("Contracts").First(&child, id).Error; err != nil {
		return nil, err
	}
	return &child, nil
}

func (s *ChildStore) Create(child *models.Child) error {
	return s.db.Create(child).Error
}

func (s *ChildStore) Update(child *models.Child) error {
	return s.db.Save(child).Error
}

func (s *ChildStore) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("child_id = ?", id).Delete(&models.ChildContract{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Child{}, id).Error
	})
}

func (s *ChildStore) CreateContract(contract *models.ChildContract) error {
	return s.db.Create(contract).Error
}

func (s *ChildStore) FindContractByID(id uint) (*models.ChildContract, error) {
	var contract models.ChildContract
	if err := s.db.First(&contract, id).Error; err != nil {
		return nil, err
	}
	return &contract, nil
}

func (s *ChildStore) UpdateContract(contract *models.ChildContract) error {
	return s.db.Save(contract).Error
}

func (s *ChildStore) DeleteContract(id uint) error {
	return s.db.Delete(&models.ChildContract{}, id).Error
}
