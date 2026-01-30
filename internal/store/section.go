package store

import (
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type SectionStore struct {
	db *gorm.DB
}

func NewSectionStore(db *gorm.DB) *SectionStore {
	return &SectionStore{db: db}
}

func (s *SectionStore) FindByID(id uint) (*models.Section, error) {
	var section models.Section
	if err := s.db.Preload("Organization").First(&section, id).Error; err != nil {
		return nil, err
	}
	return &section, nil
}

func (s *SectionStore) FindByOrganization(orgID uint) ([]models.Section, error) {
	var sections []models.Section
	if err := s.db.Where("organization_id = ?", orgID).Find(&sections).Error; err != nil {
		return nil, err
	}
	return sections, nil
}

func (s *SectionStore) FindByOrganizationPaginated(orgID uint, limit, offset int) ([]models.Section, int64, error) {
	var sections []models.Section
	var total int64

	if err := s.db.Model(&models.Section{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Preload("Organization").Where("organization_id = ?", orgID).Limit(limit).Offset(offset).Find(&sections).Error; err != nil {
		return nil, 0, err
	}

	return sections, total, nil
}

func (s *SectionStore) FindDefaultSection(orgID uint) (*models.Section, error) {
	var section models.Section
	if err := s.db.Where("organization_id = ? AND is_default = ?", orgID, true).First(&section).Error; err != nil {
		return nil, err
	}
	return &section, nil
}

func (s *SectionStore) FindByNameAndOrg(name string, orgID uint) (*models.Section, error) {
	var section models.Section
	if err := s.db.Where("organization_id = ? AND name = ?", orgID, name).First(&section).Error; err != nil {
		return nil, err
	}
	return &section, nil
}

func (s *SectionStore) Create(section *models.Section) error {
	return s.db.Create(section).Error
}

func (s *SectionStore) Update(section *models.Section) error {
	return s.db.Save(section).Error
}

func (s *SectionStore) Delete(id uint) error {
	return s.db.Delete(&models.Section{}, id).Error
}

func (s *SectionStore) HasChildren(id uint) (bool, error) {
	var count int64
	if err := s.db.Model(&models.Child{}).Where("section_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SectionStore) HasEmployees(id uint) (bool, error) {
	var count int64
	if err := s.db.Model(&models.Employee{}).Where("section_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
