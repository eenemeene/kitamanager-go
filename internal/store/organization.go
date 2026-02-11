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

func (s *OrganizationStore) FindAll(search string, limit, offset int) ([]models.Organization, int64, error) {
	var organizations []models.Organization
	var total int64

	countQuery := s.db.Model(&models.Organization{})
	if search != "" {
		countQuery = countQuery.Scopes(NameSearch("organizations", "name", search))
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := s.db.Model(&models.Organization{})
	if search != "" {
		dataQuery = dataQuery.Scopes(NameSearch("organizations", "name", search))
	}
	if err := dataQuery.Limit(limit).Offset(offset).Find(&organizations).Error; err != nil {
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
