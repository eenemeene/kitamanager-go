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

func (s *OrganizationStore) FindAll() ([]models.Organization, error) {
	var organizations []models.Organization
	if err := s.db.Find(&organizations).Error; err != nil {
		return nil, err
	}
	return organizations, nil
}

func (s *OrganizationStore) FindByID(id uint) (*models.Organization, error) {
	var organization models.Organization
	if err := s.db.Preload("Users").Preload("Groups").First(&organization, id).Error; err != nil {
		return nil, err
	}
	return &organization, nil
}

func (s *OrganizationStore) Create(organization *models.Organization) error {
	return s.db.Create(organization).Error
}

func (s *OrganizationStore) Update(organization *models.Organization) error {
	return s.db.Save(organization).Error
}

func (s *OrganizationStore) Delete(id uint) error {
	return s.db.Delete(&models.Organization{}, id).Error
}
