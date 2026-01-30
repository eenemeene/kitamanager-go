package store

import (
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type ChildStore struct {
	db            *gorm.DB
	contractStore *PeriodStore[models.ChildContract]
}

func NewChildStore(db *gorm.DB) *ChildStore {
	return &ChildStore{
		db:            db,
		contractStore: NewPeriodStore[models.ChildContract](db, "child_id"),
	}
}

func (s *ChildStore) FindAll(limit, offset int) ([]models.Child, int64, error) {
	var children []models.Child
	var total int64

	if err := s.db.Model(&models.Child{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Limit(limit).Offset(offset).Find(&children).Error; err != nil {
		return nil, 0, err
	}

	return children, total, nil
}

func (s *ChildStore) FindByOrganization(orgID uint, limit, offset int) ([]models.Child, int64, error) {
	return s.FindByOrganizationAndSection(orgID, nil, limit, offset)
}

func (s *ChildStore) FindByOrganizationAndSection(orgID uint, sectionID *uint, limit, offset int) ([]models.Child, int64, error) {
	var children []models.Child
	var total int64

	query := s.db.Model(&models.Child{}).Where("organization_id = ?", orgID)
	if sectionID != nil {
		query = query.Where("section_id = ?", *sectionID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = s.db.Preload("Contracts").Preload("Section").Where("organization_id = ?", orgID)
	if sectionID != nil {
		query = query.Where("section_id = ?", *sectionID)
	}

	if err := query.Limit(limit).Offset(offset).Find(&children).Error; err != nil {
		return nil, 0, err
	}

	return children, total, nil
}

// Contracts returns the contract store for children
func (s *ChildStore) Contracts() ContractStorer[models.ChildContract] {
	return s.contractStore
}

func (s *ChildStore) FindByID(id uint) (*models.Child, error) {
	var child models.Child
	if err := s.db.Preload("Organization").Preload("Section").Preload("Contracts").First(&child, id).Error; err != nil {
		return nil, err
	}
	return &child, nil
}

// FindByIDMinimal returns a child without preloading relationships.
// Useful for existence checks and org validation where relationships aren't needed.
func (s *ChildStore) FindByIDMinimal(id uint) (*models.Child, error) {
	var child models.Child
	if err := s.db.First(&child, id).Error; err != nil {
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
		// Delete all contracts
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

// FindByOrganizationWithContractOn returns children that have an active contract on the given date.
// A contract is active if: from_date <= date AND (to_date IS NULL OR to_date >= date)
func (s *ChildStore) FindByOrganizationWithContractOn(orgID uint, date time.Time) ([]models.Child, error) {
	var children []models.Child

	// Find children with contracts active on the given date
	if err := s.db.
		Preload("Contracts", "from_date <= ? AND (to_date IS NULL OR to_date >= ?)", date, date).
		Joins("JOIN child_contracts ON child_contracts.child_id = children.id").
		Where("children.organization_id = ?", orgID).
		Where("child_contracts.from_date <= ?", date).
		Where("child_contracts.to_date IS NULL OR child_contracts.to_date >= ?", date).
		Distinct().
		Find(&children).Error; err != nil {
		return nil, err
	}

	return children, nil
}

// CountByOrganizationWithContractOn counts children with active contracts on the given date.
// A contract is active if: from_date <= date AND (to_date IS NULL OR to_date >= date)
func (s *ChildStore) CountByOrganizationWithContractOn(orgID uint, date time.Time) (int64, error) {
	var count int64
	if err := s.db.Model(&models.Child{}).
		Joins("JOIN child_contracts ON child_contracts.child_id = children.id").
		Where("children.organization_id = ?", orgID).
		Where("child_contracts.from_date <= ?", date).
		Where("child_contracts.to_date IS NULL OR child_contracts.to_date >= ?", date).
		Distinct("children.id").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
