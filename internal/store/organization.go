package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type OrganizationStore struct {
	db *gorm.DB
}

func NewOrganizationStore(db *gorm.DB) *OrganizationStore {
	return &OrganizationStore{db: db}
}

func (s *OrganizationStore) FindAll(limit, offset int) ([]models.Organization, int64, error) {
	var organizations []models.Organization
	var total int64

	if err := s.db.Model(&models.Organization{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Limit(limit).Offset(offset).Find(&organizations).Error; err != nil {
		return nil, 0, err
	}

	return organizations, total, nil
}

func (s *OrganizationStore) FindByID(id uint) (*models.Organization, error) {
	var organization models.Organization
	if err := s.db.Preload("Groups").First(&organization, id).Error; err != nil {
		return nil, err
	}
	return &organization, nil
}

func (s *OrganizationStore) Create(organization *models.Organization) error {
	return s.db.Create(organization).Error
}

func (s *OrganizationStore) CreateWithDefaultGroup(org *models.Organization, defaultGroup *models.Group) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return err
		}
		defaultGroup.OrganizationID = org.ID
		if err := tx.Create(defaultGroup).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *OrganizationStore) Update(organization *models.Organization) error {
	return s.db.Save(organization).Error
}

func (s *OrganizationStore) Delete(id uint) error {
	return s.db.Delete(&models.Organization{}, id).Error
}
