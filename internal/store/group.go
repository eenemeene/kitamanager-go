package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type GroupStore struct {
	db *gorm.DB
}

func NewGroupStore(db *gorm.DB) *GroupStore {
	return &GroupStore{db: db}
}

func (s *GroupStore) FindAll() ([]models.Group, error) {
	var groups []models.Group
	if err := s.db.Preload("Organization").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *GroupStore) FindByID(id uint) (*models.Group, error) {
	var group models.Group
	if err := s.db.Preload("Users").Preload("Organization").First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *GroupStore) FindByOrganization(orgID uint) ([]models.Group, error) {
	var groups []models.Group
	if err := s.db.Where("organization_id = ?", orgID).Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *GroupStore) Create(group *models.Group) error {
	return s.db.Create(group).Error
}

func (s *GroupStore) Update(group *models.Group) error {
	return s.db.Save(group).Error
}

func (s *GroupStore) Delete(id uint) error {
	return s.db.Delete(&models.Group{}, id).Error
}
