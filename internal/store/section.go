package store

import (
	"context"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

type SectionStore struct {
	db *gorm.DB
}

func NewSectionStore(db *gorm.DB) *SectionStore {
	return &SectionStore{db: db}
}

func (s *SectionStore) FindByID(ctx context.Context, id uint) (*models.Section, error) {
	var section models.Section
	if err := DBFromContext(ctx, s.db).Preload("Organization").First(&section, id).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &section, nil
}

func (s *SectionStore) FindByOrganizationPaginated(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.Section, int64, error) {
	var sections []models.Section
	var total int64

	query := DBFromContext(ctx, s.db).Model(&models.Section{}).Where("organization_id = ?", orgID)
	if search != "" {
		query = query.Scopes(NameSearch("sections", "name", search))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := DBFromContext(ctx, s.db).Preload("Organization").Where("organization_id = ?", orgID)
	if search != "" {
		dataQuery = dataQuery.Scopes(NameSearch("sections", "name", search))
	}

	if err := dataQuery.Limit(limit).Offset(offset).Find(&sections).Error; err != nil {
		return nil, 0, err
	}

	return sections, total, nil
}

func (s *SectionStore) FindDefaultSection(ctx context.Context, orgID uint) (*models.Section, error) {
	var section models.Section
	if err := DBFromContext(ctx, s.db).Where("organization_id = ? AND is_default = ?", orgID, true).First(&section).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &section, nil
}

func (s *SectionStore) FindByNameAndOrg(ctx context.Context, name string, orgID uint) (*models.Section, error) {
	var section models.Section
	if err := DBFromContext(ctx, s.db).Where("organization_id = ? AND name = ?", orgID, name).First(&section).Error; err != nil {
		return nil, WrapNotFound(err)
	}
	return &section, nil
}

func (s *SectionStore) Create(ctx context.Context, section *models.Section) error {
	return DBFromContext(ctx, s.db).Create(section).Error
}

func (s *SectionStore) Update(ctx context.Context, section *models.Section) error {
	return DBFromContext(ctx, s.db).Save(section).Error
}

func (s *SectionStore) Delete(ctx context.Context, id uint) error {
	return DBFromContext(ctx, s.db).Delete(&models.Section{}, id).Error
}

func (s *SectionStore) HasChildren(ctx context.Context, id uint) (bool, error) {
	var count int64
	if err := DBFromContext(ctx, s.db).Model(&models.ChildContract{}).Where("section_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SectionStore) HasEmployees(ctx context.Context, id uint) (bool, error) {
	var count int64
	if err := DBFromContext(ctx, s.db).Model(&models.EmployeeContract{}).Where("section_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
